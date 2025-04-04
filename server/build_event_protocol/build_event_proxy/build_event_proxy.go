package build_event_proxy

import (
	"context"
	"sync"

	"github.com/buildbuddy-io/buildbuddy/server/environment"
	"github.com/buildbuddy-io/buildbuddy/server/real_environment"
	"github.com/buildbuddy-io/buildbuddy/server/util/flag"
	"github.com/buildbuddy-io/buildbuddy/server/util/grpc_client"
	"github.com/buildbuddy-io/buildbuddy/server/util/log"
	"github.com/buildbuddy-io/buildbuddy/server/util/status"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	pepb "github.com/buildbuddy-io/buildbuddy/proto/publish_build_event"
)

var (
	hosts      = flag.Slice("build_event_proxy.hosts", []string{}, "The list of hosts to pass build events onto.")
	bufferSize = flag.Int("build_event_proxy.buffer_size", 100, "The number of build events to buffer locally when proxying build events.")
)

type BuildEventProxyClient struct {
	client      pepb.PublishBuildEventClient
	rootCtx     context.Context
	target      string
	clientMux   sync.Mutex // PROTECTS(client)
	synchronous bool
}

func (c *BuildEventProxyClient) reconnectIfNecessary() {
	if c.client != nil {
		return
	}
	c.clientMux.Lock()
	defer c.clientMux.Unlock()
	conn, err := grpc_client.DialSimple(c.target)
	if err != nil {
		log.Warningf("Unable to connect to proxy host '%s': %s", c.target, err)
		c.client = nil
		return
	}
	c.client = pepb.NewPublishBuildEventClient(conn)
}

func Register(env *real_environment.RealEnv) error {
	buildEventProxyClients := make([]pepb.PublishBuildEventClient, len(*hosts))
	for i, target := range *hosts {
		// NB: This can block for up to a second on connecting. This would be a
		// great place to have our health checker and mark these as optional.
		buildEventProxyClients[i] = NewBuildEventProxyClient(env, target, false)
		log.Printf("Proxy: forwarding build events to: %s", target)
	}
	env.SetBuildEventProxyClients(buildEventProxyClients)
	return nil
}

func NewBuildEventProxyClient(env environment.Env, target string, synchronous bool) *BuildEventProxyClient {
	c := &BuildEventProxyClient{
		target:      target,
		rootCtx:     env.GetServerContext(),
		synchronous: synchronous,
	}
	c.reconnectIfNecessary()
	return c
}

func (c *BuildEventProxyClient) PublishLifecycleEvent(ctx context.Context, req *pepb.PublishLifecycleEventRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	c.reconnectIfNecessary()

	if c.synchronous {
		return c.client.PublishLifecycleEvent(ctx, req)
	}

	go func() {
		// TODO: retry these, since the client won't retry.
		if _, err := c.client.PublishLifecycleEvent(ctx, req); err != nil {
			log.Warningf("Error publishing lifecycle event: %s", err.Error())
		}
	}()
	return &emptypb.Empty{}, nil
}

type asyncStreamProxy struct {
	pepb.PublishBuildEvent_PublishBuildToolEventStreamClient
	ctx    context.Context
	events chan *pepb.PublishBuildToolEventStreamRequest
}

func (c *BuildEventProxyClient) newAsyncStreamProxy(ctx context.Context, opts ...grpc.CallOption) (*asyncStreamProxy, error) {
	asp := &asyncStreamProxy{
		ctx:    ctx,
		events: make(chan *pepb.PublishBuildToolEventStreamRequest, *bufferSize),
	}
	stream, err := c.client.PublishBuildToolEventStream(ctx, opts...)
	if err != nil {
		log.Warningf("Error opening BES stream to proxy: %s", err.Error())
		return nil, status.UnavailableErrorf("Error opening BES stream to proxy: %s", err.Error())
	}
	asp.PublishBuildEvent_PublishBuildToolEventStreamClient = stream
	// Start a goroutine that will open the stream and pass along events.
	go func() {
		// `range` *copies* the values it returns into the loopvar, and
		// copies of protos are not permitted, so rather than range over the
		// channel we read from the channel inside of an outer loop.
		for {
			req, ok := <-asp.events
			if !ok {
				break
			}
			err := stream.Send(req)
			if err != nil {
				log.Warningf("Error sending req on stream: %s", err.Error())
				break
			}
		}
		stream.CloseSend()
	}()
	return asp, nil
}

func (asp *asyncStreamProxy) Send(req *pepb.PublishBuildToolEventStreamRequest) error {
	select {
	case asp.events <- req:
		// does not fallthrough.
	default:
		log.Warningf("BuildEventProxy dropped message.")
	}
	return nil
}

func (asp *asyncStreamProxy) Recv() (*pepb.PublishBuildToolEventStreamResponse, error) {
	return asp.PublishBuildEvent_PublishBuildToolEventStreamClient.Recv()
}

func (asp *asyncStreamProxy) CloseSend() error {
	close(asp.events)
	return nil
}

func (c *BuildEventProxyClient) PublishBuildToolEventStream(_ context.Context, opts ...grpc.CallOption) (pepb.PublishBuildEvent_PublishBuildToolEventStreamClient, error) {
	c.reconnectIfNecessary()
	return c.newAsyncStreamProxy(c.rootCtx, opts...)
}
