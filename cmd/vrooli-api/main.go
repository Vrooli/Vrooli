package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	apiserver "github.com/vrooli/api-core/server"
	repocontract "github.com/vrooli/repo-contract-go"
	vrooliapi "github.com/vrooli/vrooli/internal/api"
	"github.com/vrooli/vrooli/internal/bootstrap"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/shell"
)

type HealthCheckConfig = vrooliapi.HealthCheckConfig

var (
	vrooliRoot          = getVrooliRoot()
	lookPathFn          = defaultLookPath
	commandFn           = defaultCommand
	startAllScenariosFn func() (control.StartReport, error)
	stopAllScenariosFn  func() (control.StopReport, error)
	stopScenarioFn      func(name string) error
)

func defaultLookPath(name string) (string, error) {
	return shell.LookPath(name)
}

func defaultCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return shell.Output(shell.Spec{
		Context: ctx,
		Name:    name,
		Args:    args,
	})
}

func apiHomeDir() string {
	home, err := config.HomeDir()
	if err != nil {
		return ""
	}
	return home
}

func getVrooliRoot() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return canonicalRepoRootFromOverride(root)
	}
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		return root
	}
	ex, _ := os.Executable()
	return filepath.Dir(filepath.Dir(ex))
}

func canonicalRepoRootFromOverride(root string) string {
	if resolved, err := repocontract.FindRepoRootFromPath(root); err == nil {
		return resolved
	}
	return filepath.Clean(root)
}

func buildApp(logger *slog.Logger) *vrooliapi.App {
	services := bootstrap.New(vrooliRoot, apiHomeDir(), io.Discard, io.Discard, logger)
	app := vrooliapi.NewWithServices(services)
	app.LookPathFn = lookPathFn
	app.CommandFn = commandFn
	if startAllScenariosFn != nil {
		app.StartAllScenariosFn = startAllScenariosFn
	}
	if stopAllScenariosFn != nil {
		app.StopAllScenariosFn = stopAllScenariosFn
	}
	if stopScenarioFn != nil {
		app.StopScenarioFn = stopScenarioFn
	}
	return app
}

func healthCheck(w http.ResponseWriter, r *http.Request) { buildApp(nil).HealthCheck(w, r) }
func listScenariosNative(w http.ResponseWriter, r *http.Request) {
	buildApp(nil).ListScenariosNative(w, r)
}

func getScenarioStatusNative(w http.ResponseWriter, r *http.Request) {
	buildApp(nil).GetScenarioStatusNative(w, r)
}
func listApps(w http.ResponseWriter, r *http.Request)       { buildApp(nil).ListApps(w, r) }
func getRunningApps(w http.ResponseWriter, r *http.Request) { buildApp(nil).GetRunningApps(w, r) }
func getDetailedAppStatus(w http.ResponseWriter, r *http.Request) {
	buildApp(nil).GetDetailedAppStatus(w, r)
}
func startAllApps(w http.ResponseWriter, r *http.Request) { buildApp(nil).StartAllApps(w, r) }
func stopAllApps(w http.ResponseWriter, r *http.Request)  { buildApp(nil).StopAllApps(w, r) }
func protectApp(w http.ResponseWriter, r *http.Request)   { buildApp(nil).ProtectApp(w, r) }
func processMetricsHandler(w http.ResponseWriter, r *http.Request) {
	buildApp(nil).ProcessMetricsHandler(w, r)
}
func listResources(w http.ResponseWriter, r *http.Request)   { buildApp(nil).ListResources(w, r) }
func handleLifecycle(w http.ResponseWriter, r *http.Request) { buildApp(nil).HandleLifecycle(w, r) }
func getAppLogs(w http.ResponseWriter, r *http.Request)      { buildApp(nil).GetAppLogs(w, r) }
func startApp(w http.ResponseWriter, r *http.Request)        { buildApp(nil).StartApp(w, r) }
func stopApp(w http.ResponseWriter, r *http.Request)         { buildApp(nil).StopApp(w, r) }
func restartApp(w http.ResponseWriter, r *http.Request)      { buildApp(nil).RestartApp(w, r) }
func startAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	buildApp(nil).StartAllScenariosEndpoint(w, r)
}

func stopAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	buildApp(nil).StopAllScenariosEndpoint(w, r)
}

func stopScenarioEndpoint(w http.ResponseWriter, r *http.Request) {
	buildApp(nil).StopScenarioEndpoint(w, r)
}

func performHealthCheck(check HealthCheckConfig, scenarioName string, ports map[string]int) error {
	return buildApp(nil).PerformHealthCheck(check, scenarioName, ports)
}

func installAPILogger() (*slog.Logger, func()) {
	logger, _, restore := logx.InstallAndReport(logx.Options{
		Component:      "vrooli-api",
		SetDefault:     true,
		RedirectStdlib: true,
	})
	return logger, restore
}

func main() {
	logger, restoreLogger := installAPILogger()
	defer restoreLogger()

	if err := enforceStrictFingerprint(); err != nil {
		logger.Error("Stale fingerprint check failed", logx.ErrorArgs(err)...)
		os.Exit(1)
	}

	port := os.Getenv("VROOLI_API_PORT")
	if port == "" {
		port = "8092"
	}

	logger.Info(
		"Build metadata loaded",
		logx.AttrFingerprint, buildinfo.Fingerprint,
		logx.AttrCommit, buildinfo.GitCommit,
		logx.AttrBuildTime, buildinfo.BuildTime,
		logx.AttrPort, port,
	)

	app := buildApp(logger)
	if err := apiserver.Run(apiserver.Config{
		Handler: app.Router(),
		Port:    port,
		Logger: func(format string, args ...interface{}) {
			logger.Info(fmt.Sprintf(format, args...))
		},
	}); err != nil {
		logger.Error("API server failed", logx.ErrorArgs(err)...)
		os.Exit(1)
	}
}

func enforceStrictFingerprint() error {
	if os.Getenv("VROOLI_STRICT_FINGERPRINT") != "1" {
		return nil
	}

	current, err := buildinfo.CurrentFingerprint()
	if err != nil {
		return err
	}
	if current == buildinfo.Fingerprint {
		return nil
	}
	return fmt.Errorf("binary fingerprint %s does not match current sources %s", buildinfo.Fingerprint, current)
}
