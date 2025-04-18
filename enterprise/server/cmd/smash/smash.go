package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/bojand/ghz/printer"
	"github.com/bojand/ghz/runner"
	"github.com/buildbuddy-io/buildbuddy/server/remote_cache/cachetools"
	"github.com/buildbuddy-io/buildbuddy/server/remote_cache/digest"
	"github.com/buildbuddy-io/buildbuddy/server/util/grpc_client"
	"github.com/buildbuddy-io/buildbuddy/server/util/log"
	"github.com/jhump/protoreflect/dynamic"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"

	repb "github.com/buildbuddy-io/buildbuddy/proto/remote_execution"
	bspb "google.golang.org/genproto/googleapis/bytestream"
)

var (
	cacheTarget  = flag.String("cache_target", "localhost:1985", "Cache target to connect to.")
	method       = flag.String("method", "google.bytestream.ByteStream/Write", "One of google.bytestream.ByteStream/{Read,Write},build.bazel.remote.execution.v2.ContentAddressableStorage/FindMissingBlobs.")
	rps          = flag.Uint("rps", 1000, "How many requests per second to attempt.")
	testDuration = flag.Duration("test_duration", 10*time.Second, "The duration of the loadtest.")
	concurrency  = flag.Uint("concurrency", 10, "Number of concurrent workers to use")
	instanceName = flag.String("instance_name", "loadtest", "An optional Remote Instance name.")
	apiKey       = flag.String("api_key", "", "An optional API key to use when reading / writing data.")

	randomSeed         = flag.Int64("random_seed", 0, "Random seed.")
	realisticBlobSizes = flag.Bool("realistic_blob_sizes", true, "If true, use realistic blob sizes, ignoring blob_size flag.")
	ssl                = flag.Bool("ssl", false, "If true, use ssl.")
	blobSize           = flag.Int64("blob_size", 100000, "Num bytes (max) of blob to send/read.")
	htmlOutputFile     = flag.String("html_output_file", "", "If set, results will be written to this file in HTML format")
)

const (
	byteStreamRead   = "google.bytestream.ByteStream/Read"
	byteStreamWrite  = "google.bytestream.ByteStream/Write"
	findMissingBlobs = "build.bazel.remote.execution.v2.ContentAddressableStorage/FindMissingBlobs"

	writeBufSizeBytes = 1000000 // 1MB
)

var (
	digestGenerator   *digest.Generator
	mu                sync.Mutex
	preWrittenDigests []*repb.Digest
)

var (
	// Data computed by sampling stored cache blob sizes.
	histBuckets     = []int{1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000}
	histCounts      = []int{23, 33611, 33498, 20473, 10036, 3265, 504, 62}
	histCountsTotal int
)

func init() {
	for _, c := range histCounts {
		histCountsTotal += c
	}
}

func randRange(low, high int) int64 {
	i := int64(rand.Intn(high-low+1) + low)
	return i
}

func randomBlobSize() int64 {
	if !*realisticBlobSizes {
		return *blobSize
	}
	n := rand.Intn(histCountsTotal)
	var sumTotal, low, high int
	for i, c := range histCounts {
		sumTotal += c
		high = histBuckets[i+1]
		if n < sumTotal {
			return randRange(low, high)
		}
		low = histBuckets[i+1]
	}
	return randRange(histBuckets[len(histBuckets)-2], histBuckets[len(histBuckets)-1])
}

func writeBlobsForReading(ctx context.Context, numBlobs int) []*repb.Digest {
	log.Print("Pre-writing blobs for read test.")
	prefix := "grpc://"
	if *ssl {
		prefix = "grpcs://"
	}
	conn, err := grpc_client.DialSimple(prefix + *cacheTarget)
	if err != nil {
		log.Fatalf("Unable to connect to cache '%s': %s", *cacheTarget, err)
	}
	bsClient := bspb.NewByteStreamClient(conn)
	if *apiKey != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-buildbuddy-api-key", *apiKey)
	}
	eg, ctx := errgroup.WithContext(ctx)

	mu := sync.Mutex{}
	digests := make([]*repb.Digest, 0)

	blobsPerThread := numBlobs / int(*concurrency)
	for c := 0; c < int(*concurrency); c++ {
		eg.Go(func() error {
			for i := 0; i < blobsPerThread; i++ {
				d, buf := newRandomDigestBuf(randomBlobSize())
				_, err := cachetools.UploadBlob(ctx, bsClient, *instanceName, repb.DigestFunction_SHA256, bytes.NewReader(buf))
				if err != nil {
					return err
				}
				mu.Lock()
				digests = append(digests, d)
				mu.Unlock()
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		log.Fatalf("Error pre-writing blobs: %s", err)
	}
	return digests
}

func newRandomDigestBuf(sizeBytes int64) (*repb.Digest, []byte) {
	d, buf, err := digestGenerator.RandomDigestBuf(sizeBytes)
	if err != nil {
		log.Fatalf("Error generating digset: %s", err)
	}
	return d, buf
}

func writeDataFunc(cd *runner.CallData) ([]*dynamic.Message, error) {
	d, buf := newRandomDigestBuf(randomBlobSize())
	resourceName := digest.NewCASResourceName(d, *instanceName, repb.DigestFunction_SHA256).NewUploadString()
	r := bytes.NewReader(buf)

	var messages []*dynamic.Message
	writeBuf := make([]byte, writeBufSizeBytes)
	bytesUploaded := int64(0)
	for {
		n, err := r.Read(writeBuf)
		if err != nil && err != io.EOF {
			log.Fatalf("failed to read in writeBuf: %s", err)
		}
		readDone := err == io.EOF
		data := make([]byte, n)
		copy(data, writeBuf[:n])
		wr := &bspb.WriteRequest{
			ResourceName: resourceName,
			WriteOffset:  bytesUploaded,
			Data:         data,
			FinishWrite:  readDone,
		}
		bytesUploaded += int64(len(wr.Data))
		dynamicMsg, err := dynamic.AsDynamicMessage(wr)
		if err != nil {
			return nil, err
		}
		messages = append(messages, dynamicMsg)
		if readDone {
			break
		}
	}
	return messages, nil
}

func readDataFunc(cd *runner.CallData) ([]*dynamic.Message, error) {
	randomDigest := preWrittenDigests[rand.Intn(len(preWrittenDigests))]

	downloadString := digest.NewCASResourceName(randomDigest, *instanceName, repb.DigestFunction_SHA256).DownloadString()

	rr := &bspb.ReadRequest{
		ResourceName: downloadString,
		ReadOffset:   0,
		ReadLimit:    0,
	}
	dynamicMsg, err := dynamic.AsDynamicMessage(rr)
	if err != nil {
		return nil, err
	}
	return []*dynamic.Message{dynamicMsg}, nil
}

func findMissingBlobsDataFunc(cd *runner.CallData) ([]*dynamic.Message, error) {
	req := &repb.FindMissingBlobsRequest{
		InstanceName: *instanceName,
		BlobDigests:  make([]*repb.Digest, 100),
	}
	for i := 0; i < 100; i++ {
		req.BlobDigests[i] = preWrittenDigests[rand.Intn(len(preWrittenDigests))]
	}
	dynamicMsg, err := dynamic.AsDynamicMessage(req)
	if err != nil {
		return nil, err
	}
	return []*dynamic.Message{dynamicMsg}, nil
}

func dataFunc(cd *runner.CallData) ([]*dynamic.Message, error) {
	switch *method {
	case findMissingBlobs:
		return findMissingBlobsDataFunc(cd)
	case byteStreamRead:
		return readDataFunc(cd)
	case byteStreamWrite:
		return writeDataFunc(cd)
	}
	return nil, fmt.Errorf("Unknown rpc method: %q", *method)
}

func main() {
	flag.Parse()

	seed := *randomSeed
	if seed == 0 {
		seed = time.Now().Unix()
	}
	digestGenerator = digest.RandomGenerator(seed)
	ctx := context.Background()

	if *method == byteStreamRead {
		preWrittenDigests = writeBlobsForReading(ctx, int(*concurrency))
	} else if *method == findMissingBlobs {
		preWrittenDigests = writeBlobsForReading(ctx, 1000)
	}

	blobSizeDesc := fmt.Sprintf("size %d bytes", *blobSize)
	if *realisticBlobSizes {
		blobSizeDesc = "simulating real blob sizes."
	}

	md := make(map[string]string)
	if *apiKey != "" {
		md["x-buildbuddy-api-key"] = *apiKey
	}
	log.Printf("Running a %s test @ %d r/sec, concurrency: %d, %s", *testDuration, *rps, *concurrency, blobSizeDesc)
	report, err := runner.Run(
		*method,
		*cacheTarget,
		runner.WithConcurrency(*concurrency),
		runner.WithRPS(*rps),
		runner.WithRunDuration(*testDuration),
		runner.WithInsecure(!*ssl),
		runner.WithMetadata(md),
		runner.WithDataProvider(dataFunc),
	)

	if err != nil {
		log.Fatal(err.Error())
	}

	printer := printer.ReportPrinter{
		Out:    os.Stdout,
		Report: report,
	}
	printer.Print("summary")

	if *htmlOutputFile != "" {
		f, err := os.Create(*htmlOutputFile)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer f.Close()
		printer.Out = f
		if err := printer.Print("html"); err != nil {
			log.Fatal(err.Error())
		}
		log.Printf("Wrote results to f: %+v", f.Name())
	}
}
