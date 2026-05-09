package orchestrator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sort"
	"time"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type Service struct {
	Root   string
	Home   string
	Stdout io.Writer
	Stderr io.Writer
	// Logger is the orchestrator-scoped logger (subsystem=orchestrator).
	// Use for emissions that should be tagged with the orchestrator
	// subsystem. To construct a child service's own scoped logger, use
	// base instead so the child's subsystem does not inherit this layer's
	// attribute and produce duplicate subsystem= tokens.
	Logger *slog.Logger
	base   *slog.Logger

	newRunner       lifecycleRunnerFactory
	runtimeRegistry runtimeRegistryFactory
	hostSession     func(context.Context, string) (hostsession.Snapshot, error)
}

type lifecycleRunner interface {
	Start(name string, opts lifecycle.StartOptions) (lifecycle.Result, error)
	Restart(name string, opts lifecycle.StartOptions) (lifecycle.Result, error)
	Stop(name string, opts lifecycle.StopOptions) error
}

type lifecycleRunnerFactory func(root, home string, stdout, stderr io.Writer, logger ...*slog.Logger) (lifecycleRunner, error)

type runtimeRegistryQueryStore interface {
	scenarioruntime.QueryRepository
	scenarioruntime.ProcessRefRepository
	Close() error
}

type runtimeRegistryFactory func(ctx context.Context, home string) (runtimeRegistryQueryStore, error)

type ScenarioView struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name,omitempty"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Status      string         `json:"status"`
	Processes   int            `json:"processes"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	Runtime     string         `json:"runtime"`
	Ports       map[string]int `json:"ports"`
	Health      any            `json:"health_status,omitempty"`
}

type InventoryReport struct {
	Items    []Detail            `json:"items"`
	Failures []discovery.Failure `json:"failures,omitempty"`
}

type ListReport struct {
	Items    []ScenarioView      `json:"items"`
	Failures []discovery.Failure `json:"failures,omitempty"`
}

func New(root, home string, stdout, stderr io.Writer, logger ...*slog.Logger) *Service {
	baseLogger := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		baseLogger = logger[0]
	}
	return &Service{
		Root:   filepath.Clean(root),
		Home:   filepath.Clean(home),
		Stdout: stdout,
		Stderr: stderr,
		Logger: logx.WithSubsystem(baseLogger, "orchestrator"),
		base:   baseLogger,
		newRunner: func(root, home string, stdout, stderr io.Writer, logger ...*slog.Logger) (lifecycleRunner, error) {
			return lifecycle.NewRunner(root, home, stdout, stderr, logger...)
		},
		runtimeRegistry: func(ctx context.Context, home string) (runtimeRegistryQueryStore, error) {
			return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		},
		hostSession: hostsession.DefaultProvider{}.Current,
	}
}

func (s *Service) List() ([]ScenarioView, error) {
	report, err := s.ListReport()
	if err != nil {
		return nil, err
	}
	return report.Items, nil
}

func (s *Service) ListReport() (ListReport, error) {
	report, err := s.InventoryReport()
	if err != nil {
		return ListReport{}, err
	}

	views := make([]ScenarioView, 0, len(report.Items))
	for _, item := range report.Items {
		views = append(views, s.viewForDetail(item))
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })
	return ListReport{Items: views, Failures: append([]discovery.Failure(nil), report.Failures...)}, nil
}

func (s *Service) Running() ([]ScenarioView, error) {
	items, err := s.List()
	if err != nil {
		return nil, err
	}

	running := make([]ScenarioView, 0, len(items))
	for _, item := range items {
		if item.Status == "running" {
			running = append(running, item)
		}
	}
	return running, nil
}

func (s *Service) Status(name string) (ScenarioView, bool, error) {
	detail, exists, err := s.Lookup(name)
	if err != nil {
		return ScenarioView{}, false, err
	}
	if !exists {
		return ScenarioView{
			Name:    name,
			Status:  "stopped",
			Runtime: "N/A",
			Ports:   map[string]int{},
		}, false, nil
	}
	return s.viewForDetail(detail), true, nil
}

func (s *Service) Start(name string, opts lifecycle.StartOptions) (ScenarioView, error) {
	s.logger().Info("Scenario start dispatched", logx.AttrScenario, name)
	result, err := s.StartDetailed(name, opts)
	if err != nil {
		logx.Error(s.logger(), "Scenario start failed", err, logx.AttrScenario, name)
		return ScenarioView{}, err
	}
	s.logger().Info("Scenario start resolved", logx.AttrScenario, name, logx.AttrStatus, result.Details.Health)
	return result.View, err
}

func (s *Service) Stop(name string, opts lifecycle.StopOptions) error {
	s.logger().Info("Scenario stop dispatched", logx.AttrScenario, name)
	runner, err := s.runner()
	if err != nil {
		logx.Error(s.logger(), "Failed to construct lifecycle runner for stop", err, logx.AttrScenario, name)
		return err
	}
	if err := runner.Stop(name, opts); err != nil {
		logx.Error(s.logger(), "Scenario stop failed", err, logx.AttrScenario, name)
		return err
	}
	s.logger().Info("Scenario stop resolved", logx.AttrScenario, name)
	return nil
}

func (s *Service) Restart(name string, opts lifecycle.StartOptions) (ScenarioView, error) {
	s.logger().Info("Scenario restart dispatched", logx.AttrScenario, name)
	result, err := s.RestartDetailed(name, opts)
	if err != nil {
		logx.Error(s.logger(), "Scenario restart failed", err, logx.AttrScenario, name)
		return ScenarioView{}, err
	}
	s.logger().Info("Scenario restart resolved", logx.AttrScenario, name, logx.AttrStatus, result.Details.Health)
	return result.View, err
}

func (s *Service) StartAll() (control.StartReport, error) {
	items, err := scenario.Discover(s.Root, scenario.SandboxEnvFromEnv())
	if err != nil {
		return control.StartReport{}, err
	}
	runner, err := s.runner()
	if err != nil {
		return control.StartReport{}, err
	}

	started := make([]control.ResultItem, 0, len(items))
	failed := make([]control.ResultItem, 0)
	for _, item := range items {
		if _, err := runner.Start(item.Slug, lifecycle.StartOptions{}); err != nil {
			args := append([]any{logx.AttrScenario, item.Slug}, logx.ErrorArgs(err)...)
			s.logger().Warn("Scenario start-all item failed", args...)
			failed = append(failed, control.Failed(item.Slug, err))
			continue
		}
		started = append(started, control.Started(item.Slug, "Started successfully"))
	}
	s.logger().Info("Scenario start-all completed", "started", len(started), "failed", len(failed))
	return control.StartReport{
		Started: started,
		Failed:  failed,
		Message: fmt.Sprintf("Started %d scenarios, %d failed", len(started), len(failed)),
	}, nil
}

func (s *Service) StopAll() (control.StopReport, error) {
	running, err := s.Running()
	if err != nil {
		return control.StopReport{}, err
	}
	runner, err := s.runner()
	if err != nil {
		return control.StopReport{}, err
	}

	stopped := make([]control.ResultItem, 0, len(running))
	failed := make([]control.ResultItem, 0)
	for _, item := range running {
		if err := runner.Stop(item.Name, lifecycle.StopOptions{}); err != nil {
			args := append([]any{logx.AttrScenario, item.Name}, logx.ErrorArgs(err)...)
			s.logger().Warn("Scenario stop-all item failed", args...)
			failed = append(failed, control.Failed(item.Name, err))
			continue
		}
		stopped = append(stopped, control.Stopped(item.Name, "Stopped successfully"))
	}
	s.logger().Info("Scenario stop-all completed", "stopped", len(stopped), "failed", len(failed))
	return control.StopReport{
		Stopped: stopped,
		Failed:  failed,
		Message: fmt.Sprintf("Stopped %d scenarios, %d failed", len(stopped), len(failed)),
	}, nil
}

func (s *Service) logger() *slog.Logger {
	if s == nil || s.Logger == nil {
		return logx.WithSubsystem(slog.Default(), "orchestrator")
	}
	return s.Logger
}

func (s *Service) runner() (lifecycleRunner, error) {
	runnerLogger := s.runnerBaseLogger()
	if s != nil && s.newRunner != nil {
		return s.newRunner(s.Root, s.Home, s.Stdout, s.Stderr, runnerLogger)
	}
	return lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr, runnerLogger)
}

// runnerBaseLogger returns the unscoped logger the lifecycle runner should
// wrap with its own subsystem. Passing the already-scoped s.Logger would
// cause lifecycle to inherit subsystem=orchestrator and emit duplicate
// subsystem= tokens on every line.
func (s *Service) runnerBaseLogger() *slog.Logger {
	if s != nil && s.base != nil {
		return s.base
	}
	return slog.Default()
}
