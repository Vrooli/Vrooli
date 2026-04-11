package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	apiserver "github.com/vrooli/api-core/server"
	vrooliapi "github.com/vrooli/vrooli/internal/api"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/logx"
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
	return exec.LookPath(name)
}

func defaultCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

func apiHomeDir() string {
	home, err := config.HomeDir()
	if err != nil {
		return ""
	}
	return home
}

func getVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return string(bytes.TrimSpace(out))
	}
	ex, _ := os.Executable()
	return filepath.Dir(filepath.Dir(ex))
}

func buildApp() *vrooliapi.App {
	app := vrooliapi.New(vrooliRoot, apiHomeDir())
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

func healthCheck(w http.ResponseWriter, r *http.Request) { buildApp().HealthCheck(w, r) }
func listScenariosNative(w http.ResponseWriter, r *http.Request) {
	buildApp().ListScenariosNative(w, r)
}
func getScenarioStatusNative(w http.ResponseWriter, r *http.Request) {
	buildApp().GetScenarioStatusNative(w, r)
}
func listApps(w http.ResponseWriter, r *http.Request)       { buildApp().ListApps(w, r) }
func getRunningApps(w http.ResponseWriter, r *http.Request) { buildApp().GetRunningApps(w, r) }
func getDetailedAppStatus(w http.ResponseWriter, r *http.Request) {
	buildApp().GetDetailedAppStatus(w, r)
}
func startAllApps(w http.ResponseWriter, r *http.Request) { buildApp().StartAllApps(w, r) }
func stopAllApps(w http.ResponseWriter, r *http.Request)  { buildApp().StopAllApps(w, r) }
func protectApp(w http.ResponseWriter, r *http.Request)   { buildApp().ProtectApp(w, r) }
func processMetricsHandler(w http.ResponseWriter, r *http.Request) {
	buildApp().ProcessMetricsHandler(w, r)
}
func listResources(w http.ResponseWriter, r *http.Request)   { buildApp().ListResources(w, r) }
func handleLifecycle(w http.ResponseWriter, r *http.Request) { buildApp().HandleLifecycle(w, r) }
func getAppLogs(w http.ResponseWriter, r *http.Request)      { buildApp().GetAppLogs(w, r) }
func startApp(w http.ResponseWriter, r *http.Request)        { buildApp().StartApp(w, r) }
func stopApp(w http.ResponseWriter, r *http.Request)         { buildApp().StopApp(w, r) }
func restartApp(w http.ResponseWriter, r *http.Request)      { buildApp().RestartApp(w, r) }
func startAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	buildApp().StartAllScenariosEndpoint(w, r)
}
func stopAllScenariosEndpoint(w http.ResponseWriter, r *http.Request) {
	buildApp().StopAllScenariosEndpoint(w, r)
}
func stopScenarioEndpoint(w http.ResponseWriter, r *http.Request) {
	buildApp().StopScenarioEndpoint(w, r)
}
func performHealthCheck(check HealthCheckConfig, scenarioName string, ports map[string]int) error {
	return buildApp().PerformHealthCheck(check, scenarioName, ports)
}

func main() {
	logger := logx.New(logx.Options{Name: "vrooli-api"})
	slog.SetDefault(logger)
	logx.RedirectStandardLibrary(logger)

	if err := enforceStrictFingerprint(); err != nil {
		log.Fatalf("stale fingerprint check failed: %v", err)
	}

	port := os.Getenv("VROOLI_API_PORT")
	if port == "" {
		port = "8092"
	}

	log.Printf(
		"build metadata loaded fingerprint=%s commit=%s build_time=%s",
		buildinfo.Fingerprint,
		buildinfo.GitCommit,
		buildinfo.BuildTime,
	)

	app := buildApp()
	if err := apiserver.Run(apiserver.Config{
		Handler: app.Router(),
		Port:    port,
		Logger: func(format string, args ...interface{}) {
			log.Printf(format, args...)
		},
	}); err != nil {
		log.Fatalf("API server failed: %v", err)
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
