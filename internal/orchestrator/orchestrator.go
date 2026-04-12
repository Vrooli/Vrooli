package orchestrator

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sort"
	"time"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
)

type Service struct {
	Root   string
	Home   string
	Stdout io.Writer
	Stderr io.Writer
	Logger *slog.Logger
}

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
	}
}

func (s *Service) List() ([]ScenarioView, error) {
	items, err := s.Inventory()
	if err != nil {
		return nil, err
	}

	views := make([]ScenarioView, 0, len(items))
	for _, item := range items {
		views = append(views, s.viewForDetail(item))
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })
	return views, nil
}

func (s *Service) Running() ([]ScenarioView, error) {
	items, err := s.List()
	if err != nil {
		return nil, err
	}

	running := make([]ScenarioView, 0, len(items))
	for _, item := range items {
		if item.Processes > 0 {
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
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr, s.logger())
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
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr, s.logger())
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
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr, s.logger())
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

func (s *Service) viewForScenario(item scenario.Scenario) (ScenarioView, error) {
	detail, err := s.Detail(item.Slug)
	if err != nil {
		return ScenarioView{}, err
	}
	return s.viewForDetail(detail), nil
}
