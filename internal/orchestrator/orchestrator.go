package orchestrator

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

type Service struct {
	Root   string
	Home   string
	Stdout io.Writer
	Stderr io.Writer
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

func New(root, home string, stdout, stderr io.Writer) *Service {
	return &Service{
		Root:   filepath.Clean(root),
		Home:   filepath.Clean(home),
		Stdout: stdout,
		Stderr: stderr,
	}
}

func (s *Service) List() ([]ScenarioView, error) {
	items, err := scenario.Discover(s.Root, scenario.SandboxEnvFromEnv())
	if err != nil {
		return nil, err
	}

	views := make([]ScenarioView, 0, len(items))
	for _, item := range items {
		view, err := s.viewForScenario(item)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
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
	item, err := scenario.Load(s.Root, name, scenario.SandboxEnvFromEnv())
	if err != nil {
		if err == scenario.ErrNotFound {
			return ScenarioView{
				Name:    name,
				Status:  "stopped",
				Runtime: "N/A",
				Ports:   map[string]int{},
			}, false, nil
		}
		return ScenarioView{}, false, err
	}

	view, err := s.viewForScenario(item)
	if err != nil {
		return ScenarioView{}, false, err
	}
	return view, true, nil
}

func (s *Service) Start(name string, opts lifecycle.StartOptions) (ScenarioView, error) {
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr)
	if err != nil {
		return ScenarioView{}, err
	}
	if _, err := runner.Start(name, opts); err != nil {
		return ScenarioView{}, err
	}
	view, _, err := s.Status(name)
	return view, err
}

func (s *Service) Stop(name string, opts lifecycle.StopOptions) error {
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr)
	if err != nil {
		return err
	}
	return runner.Stop(name, opts)
}

func (s *Service) Restart(name string, opts lifecycle.StartOptions) (ScenarioView, error) {
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr)
	if err != nil {
		return ScenarioView{}, err
	}
	if _, err := runner.Restart(name, opts); err != nil {
		return ScenarioView{}, err
	}
	view, _, err := s.Status(name)
	return view, err
}

func (s *Service) StartAll() (control.StartReport, error) {
	items, err := scenario.Discover(s.Root, scenario.SandboxEnvFromEnv())
	if err != nil {
		return control.StartReport{}, err
	}
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr)
	if err != nil {
		return control.StartReport{}, err
	}

	started := make([]control.ResultItem, 0, len(items))
	failed := make([]control.ResultItem, 0)
	for _, item := range items {
		if _, err := runner.Start(item.Slug, lifecycle.StartOptions{}); err != nil {
			failed = append(failed, control.Failed(item.Slug, err))
			continue
		}
		started = append(started, control.Started(item.Slug, "Started successfully"))
	}
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
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr)
	if err != nil {
		return control.StopReport{}, err
	}

	stopped := make([]control.ResultItem, 0, len(running))
	failed := make([]control.ResultItem, 0)
	for _, item := range running {
		if err := runner.Stop(item.Name, lifecycle.StopOptions{}); err != nil {
			failed = append(failed, control.Failed(item.Name, err))
			continue
		}
		stopped = append(stopped, control.Stopped(item.Name, "Stopped successfully"))
	}
	return control.StopReport{
		Stopped: stopped,
		Failed:  failed,
		Message: fmt.Sprintf("Stopped %d scenarios, %d failed", len(stopped), len(failed)),
	}, nil
}

func (s *Service) viewForScenario(item scenario.Scenario) (ScenarioView, error) {
	records, err := process.ReadScenarioRecords(s.Home, item.Slug)
	if err != nil {
		return ScenarioView{}, err
	}
	runtimeState := process.SummarizeScenario(item.Slug, records)
	ports := runtimePorts(item.Manifest, runtimeState.Records)

	status := "stopped"
	health := any(nil)
	if runtimeState.ProcessCount > 0 {
		status = "running"
		health = scenario.EvaluateHealth(item.Manifest.HealthConfig(), ports)
	}

	return ScenarioView{
		Name:        item.Slug,
		DisplayName: item.Manifest.Service.DisplayName,
		Description: item.Manifest.Service.Description,
		Tags:        append([]string(nil), item.Manifest.Service.Tags...),
		Status:      status,
		Processes:   runtimeState.ProcessCount,
		StartedAt:   runtimeState.StartedAt,
		Runtime:     runtimeState.Runtime,
		Ports:       ports,
		Health:      health,
	}, nil
}

func runtimePorts(manifest scenario.ServiceManifest, records []process.Record) map[string]int {
	portsByEnv := make(map[string]int)
	for _, record := range records {
		if record.Port <= 0 {
			continue
		}
		key := inferPortEnvVar(manifest, record.Step)
		if key == "" {
			continue
		}
		if _, exists := portsByEnv[key]; !exists {
			portsByEnv[key] = record.Port
		}
	}

	envPorts := process.ReadEnvironmentPorts(records, manifest.PortEnvVars())
	for key, port := range envPorts {
		if _, exists := portsByEnv[key]; !exists {
			portsByEnv[key] = port
		}
	}
	return portsByEnv
}

func inferPortEnvVar(manifest scenario.ServiceManifest, step string) string {
	step = strings.ToLower(strings.TrimSpace(step))
	switch {
	case strings.Contains(step, "ui"), strings.Contains(step, "frontend"), strings.Contains(step, "vite"):
		if key := manifest.PortEnvVar("ui"); key != "" {
			return key
		}
	case strings.Contains(step, "ws"), strings.Contains(step, "socket"):
		for _, candidate := range []string{"websocket", "ws"} {
			if key := manifest.PortEnvVar(candidate); key != "" {
				return key
			}
		}
	default:
		if key := manifest.PortEnvVar("api"); key != "" {
			return key
		}
	}

	for _, portSummary := range manifest.SortedPorts() {
		if strings.Contains(step, strings.ToLower(portSummary.Name)) {
			return portSummary.EnvVar
		}
	}
	return ""
}
