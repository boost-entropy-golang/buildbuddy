package main

import (
	"flag"

	"github.com/buildbuddy-io/buildbuddy/server/config"
	"github.com/buildbuddy-io/buildbuddy/server/janitor"
	"github.com/buildbuddy-io/buildbuddy/server/libmain"
	"github.com/buildbuddy-io/buildbuddy/server/telemetry"
	"github.com/buildbuddy-io/buildbuddy/server/util/healthcheck"
	"github.com/buildbuddy-io/buildbuddy/server/util/log"
	"github.com/buildbuddy-io/buildbuddy/server/version"

	app_bundle "github.com/buildbuddy-io/buildbuddy/app"
)

var (
	serverType = flag.String("server_type", "buildbuddy-server", "The server type to match on health checks")
)

// NB: Most of the logic you'd typically find in a main.go file is in
// libmain.go. We put it there to reduce the duplicated code between the open
// source main() entry point and the enterprise main() entry point, both of
// which import from libmain.go.

func main() {
	version.Print("BuildBuddy")

	flag.Parse()
	if err := config.Load(); err != nil {
		log.Fatalf("Error loading config from file: %s", err)
	}

	config.ReloadOnSIGHUP()

	healthChecker := healthcheck.NewHealthChecker(*serverType)
	appFS, err := app_bundle.GetAppFS()
	if err != nil {
		log.Fatalf("Error getting app FS from bundle: %s", err)
	}
	env := libmain.GetConfiguredEnvironmentOrDie(healthChecker, appFS)

	telemetryClient := telemetry.NewTelemetryClient(env)
	telemetryClient.Start()
	defer telemetryClient.Stop()

	cleanupService := janitor.NewInvocationJanitor(env)
	cleanupService.Start()
	defer cleanupService.Stop()

	libmain.StartMonitoringHandler(env)
	libmain.StartAndRunServices(env) // Does not return
}
