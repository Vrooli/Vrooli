package api

import (
	"context"
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
)

type RunningScenario struct {
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Processes int            `json:"processes"`
	StartedAt *time.Time     `json:"started_at"`
	Runtime   string         `json:"runtime"`
	Ports     map[string]int `json:"ports"`
}

type HealthCheckConfig struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Critical bool   `json:"critical"`
	Timeout  int    `json:"timeout"`
	Interval int    `json:"interval"`
}

type ScenarioHealthConfig struct {
	Description        string              `json:"description"`
	Endpoints          map[string]string   `json:"endpoints"`
	Checks             []HealthCheckConfig `json:"checks"`
	Timeout            int                 `json:"timeout"`
	Interval           int                 `json:"interval"`
	StartupGracePeriod int                 `json:"startup_grace_period"`
}

type App struct {
	Root                string
	Home                string
	AppsDir             string
	Scenarios           *orchestrator.Service
	Resources           *resources.Controller
	Project             *project.Controller
	Logger              *slog.Logger
	LookPathFn          func(string) (string, error)
	CommandFn           func(context.Context, string, ...string) ([]byte, error)
	StartAllScenariosFn func() (control.StartReport, error)
	StopAllScenariosFn  func() (control.StopReport, error)
	StopScenarioFn      func(string) error
	ProcessSnapshotFn   func() (maintenance.ProcessSnapshot, error)
}

type messageData struct {
	Message string `json:"message"`
}

type appLogsData struct {
	Logs     string `json:"logs"`
	Scenario string `json:"scenario"`
}

type apiProcessData struct {
	ProcessID  string     `json:"process_id,omitempty"`
	StepName   string     `json:"step_name,omitempty"`
	PID        int        `json:"pid"`
	PGID       int        `json:"pgid,omitempty"`
	Status     string     `json:"status,omitempty"`
	Phase      string     `json:"phase,omitempty"`
	Command    string     `json:"command,omitempty"`
	WorkingDir string     `json:"working_dir,omitempty"`
	LogFile    string     `json:"log_file,omitempty"`
	Port       int        `json:"port,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
}

type lifecycleActionData struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}

type stoppedAppData struct {
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Processes int            `json:"processes"`
	Runtime   string         `json:"runtime"`
	Ports     map[string]int `json:"ports"`
}

type appInfo struct {
	Name          string                 `json:"name"`
	Path          string                 `json:"path"`
	Protected     bool                   `json:"protected"`
	HasGit        bool                   `json:"has_git"`
	Customized    bool                   `json:"customized"`
	Modified      time.Time              `json:"modified"`
	RuntimeStatus string                 `json:"runtime_status,omitempty"`
	Ports         map[string]interface{} `json:"ports,omitempty"`
	PID           *int                   `json:"pid,omitempty"`
}

func New(root, home string, logger ...*slog.Logger) *App {
	baseLogger := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		baseLogger = logger[0]
	}
	apiLogger := logx.WithSubsystem(baseLogger, "api")
	app := &App{
		Root:       filepath.Clean(root),
		Home:       filepath.Clean(home),
		AppsDir:    filepath.Join(filepath.Clean(root), "scenarios"),
		Scenarios:  orchestrator.New(root, home, ioDiscard{}, ioDiscard{}, apiLogger),
		Resources:  resources.NewController(root, home),
		Project:    project.New(root, home, ioDiscard{}, ioDiscard{}),
		Logger:     apiLogger,
		LookPathFn: exec.LookPath,
		CommandFn: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
	}
	app.StartAllScenariosFn = func() (control.StartReport, error) {
		return app.Scenarios.StartAll()
	}
	app.StopAllScenariosFn = func() (control.StopReport, error) {
		return app.Scenarios.StopAll()
	}
	app.StopScenarioFn = func(name string) error {
		return app.Scenarios.Stop(name, lifecycle.StopOptions{})
	}
	return app
}

func (a *App) logger() *slog.Logger {
	if a == nil || a.Logger == nil {
		return logx.WithSubsystem(slog.Default(), "api")
	}
	return a.Logger
}

func (a *App) logInfo(msg string, args ...any) {
	a.logger().Info(msg, args...)
}

func (a *App) logWarn(msg string, args ...any) {
	a.logger().Warn(msg, args...)
}

func (a *App) logError(msg string, err error, args ...any) {
	logx.Error(a.logger(), msg, err, args...)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func (a *App) maintenanceController() *maintenance.Controller {
	return maintenance.NewController(a.Root, a.Home)
}

func (a *App) processSnapshot() (maintenance.ProcessSnapshot, error) {
	if a != nil && a.ProcessSnapshotFn != nil {
		return a.ProcessSnapshotFn()
	}
	return a.maintenanceController().Snapshot()
}

func (a *App) Router() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/health", a.HealthCheck).Methods("GET")
	r.HandleFunc("/metrics/processes", a.ProcessMetricsHandler).Methods("GET")
	r.HandleFunc("/apps", a.ListApps).Methods("GET")
	r.HandleFunc("/apps/running", a.GetRunningApps).Methods("GET")
	r.HandleFunc("/apps/start-all", a.StartAllApps).Methods("POST")
	r.HandleFunc("/apps/stop-all", a.StopAllApps).Methods("POST")
	r.HandleFunc("/apps/{name}/protect", a.ProtectApp).Methods("POST")
	r.HandleFunc("/apps/{name}/start", a.StartApp).Methods("POST")
	r.HandleFunc("/apps/{name}/stop", a.StopApp).Methods("POST")
	r.HandleFunc("/apps/{name}/restart", a.RestartApp).Methods("POST")
	r.HandleFunc("/apps/{name}/logs", a.GetAppLogs).Methods("GET")
	r.HandleFunc("/apps/{name}/status", a.GetDetailedAppStatus).Methods("GET")
	r.HandleFunc("/scenarios", a.ListScenariosNative).Methods("GET")
	r.HandleFunc("/scenarios/{name}/status", a.GetScenarioStatusNative).Methods("GET")
	r.HandleFunc("/scenarios/{name}/start", a.StartApp).Methods("POST")
	r.HandleFunc("/scenarios/{name}/stop", a.StopScenarioEndpoint).Methods("POST")
	r.HandleFunc("/scenarios/start-all", a.StartAllScenariosEndpoint).Methods("POST")
	r.HandleFunc("/scenarios/stop-all", a.StopAllScenariosEndpoint).Methods("POST")
	r.HandleFunc("/resources", a.ListResources).Methods("GET")
	r.HandleFunc("/lifecycle/{action}", a.HandleLifecycle).Methods("POST")
	return r
}
