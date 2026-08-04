package packageapp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/internal/shell"
)

type ScenarioRuntime interface {
	Lookup(name string) (orchestrator.Detail, bool, error)
	StartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error)
}

type ScenarioPhaseRunner interface {
	Stop(name string, opts lifecycle.StopOptions) error
	RunPhaseDetailed(name, phase string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error)
}

type Service struct {
	Root            string
	Stdout          io.Writer
	Stderr          io.Writer
	ScenarioService func() (ScenarioRuntime, error)
	ScenarioRunner  func() (ScenarioPhaseRunner, error)
}

type RunResponse struct {
	PackageName string `json:"package_name"`
	Action      string `json:"action"`
}

type RefreshRequest struct {
	PackageName string
	Target      string
	NoRestart   bool
}

type RefreshItem struct {
	Consumer string                       `json:"consumer"`
	Class    packagegov.ConsumerClass     `json:"consumer_class"`
	Classes  []packagegov.ConsumerClass   `json:"consumer_classes,omitempty"`
	Action   packagegov.RefreshActionKind `json:"action"`
	Status   string                       `json:"status"`
}

type RefreshResponse struct {
	PackageName string        `json:"package_name"`
	Items       []RefreshItem `json:"items"`
}

func (s Service) List() ([]packagegov.Package, []packagegov.ValidationIssue, error) {
	return packagegov.LoadAll(s.Root)
}

func (s Service) Info(name string) (packagegov.Package, error) {
	items, _, err := packagegov.LoadAll(s.Root)
	if err != nil {
		return packagegov.Package{}, err
	}
	item, ok := packagegov.FindByName(items, name)
	if !ok {
		return packagegov.Package{}, fmt.Errorf("package %q not found", name)
	}
	return item, nil
}

func (s Service) Dependents(name string) (packagegov.Package, packagegov.DiscoveryReport, error) {
	item, err := s.Info(name)
	if err != nil {
		return packagegov.Package{}, packagegov.DiscoveryReport{}, err
	}
	report, err := packagegov.DiscoverDependents(s.Root, item)
	if err != nil {
		return packagegov.Package{}, packagegov.DiscoveryReport{}, err
	}
	return item, report, nil
}

func (s Service) Validate(name string) (packagegov.ValidationReport, error) {
	return packagegov.Validate(s.Root, name)
}

func (s Service) Audit(name string) (packagegov.AuditReport, error) {
	return packagegov.Audit(s.Root, name)
}

func (s Service) Build(name string) (RunResponse, error) {
	return s.runLifecycle(name, "build")
}

func (s Service) Generate(name string) (RunResponse, error) {
	return s.runLifecycle(name, "generate")
}

func (s Service) Test(name string) (RunResponse, error) {
	return s.runLifecycle(name, "test")
}

func (s Service) Refresh(req RefreshRequest) (RefreshResponse, error) {
	item, err := s.Info(req.PackageName)
	if err != nil {
		return RefreshResponse{}, err
	}

	if item.Manifest.Package.Refresh.Strategy == packagegov.RefreshGenerateThenSetup {
		if err := packagegov.RunCommands(item.RootPath, item.Manifest.Package.Lifecycle.Generate, s.Stdout, s.Stderr); err != nil {
			return RefreshResponse{}, err
		}
	}
	if err := packagegov.RunCommands(item.RootPath, item.Manifest.Package.Lifecycle.Build, s.Stdout, s.Stderr); err != nil {
		return RefreshResponse{}, err
	}

	discovery, err := packagegov.DiscoverDependents(s.Root, item)
	if err != nil {
		return RefreshResponse{}, err
	}
	actions := packagegov.PlanRefresh(item, discovery.Dependents, req.Target)
	resp := RefreshResponse{PackageName: item.Name}

	var (
		service ScenarioRuntime
		runner  ScenarioPhaseRunner
	)
	getScenarioService := func() (ScenarioRuntime, error) {
		if service != nil {
			return service, nil
		}
		if s.ScenarioService == nil {
			return nil, fmt.Errorf("scenario service is not configured")
		}
		resolved, err := s.ScenarioService()
		if err != nil {
			return nil, err
		}
		service = resolved
		return service, nil
	}
	getScenarioRunner := func() (ScenarioPhaseRunner, error) {
		if runner != nil {
			return runner, nil
		}
		if s.ScenarioRunner == nil {
			return nil, fmt.Errorf("scenario runner is not configured")
		}
		resolved, err := s.ScenarioRunner()
		if err != nil {
			return nil, err
		}
		runner = resolved
		return runner, nil
	}

	for _, action := range actions {
		status := "no_action"

		switch action.Action {
		case packagegov.RefreshActionScenarioSetup:
			service, err := getScenarioService()
			if err != nil {
				return RefreshResponse{}, err
			}
			runner, err := getScenarioRunner()
			if err != nil {
				return RefreshResponse{}, err
			}
			detail, _, err := service.Lookup(action.ConsumerName)
			if err != nil {
				return RefreshResponse{}, err
			}
			wasRunning := detail.Runtime.ProcessCount > 0
			if wasRunning {
				if err := runner.Stop(action.ConsumerName, lifecycle.StopOptions{}); err != nil {
					return RefreshResponse{}, err
				}
			}
			if _, err := runner.RunPhaseDetailed(action.ConsumerName, "setup", lifecycle.PhaseOptions{}); err != nil {
				return RefreshResponse{}, err
			}
			status = "setup_only"
			if wasRunning && !req.NoRestart && item.Manifest.Package.Refresh.RestartRunningConsumers {
				if _, err := service.StartDetailed(action.ConsumerName, lifecycle.StartOptions{}); err != nil {
					return RefreshResponse{}, err
				}
				status = "restarted"
			} else if wasRunning {
				status = "stopped_after_setup"
			}
		case packagegov.RefreshActionRestartScenario:
			service, err := getScenarioService()
			if err != nil {
				return RefreshResponse{}, err
			}
			runner, err := getScenarioRunner()
			if err != nil {
				return RefreshResponse{}, err
			}
			detail, _, err := service.Lookup(action.ConsumerName)
			if err != nil {
				return RefreshResponse{}, err
			}
			wasRunning := detail.Runtime.ProcessCount > 0
			if !wasRunning {
				status = "not_running"
				break
			}
			if req.NoRestart || !item.Manifest.Package.Refresh.RestartRunningConsumers {
				status = "running_not_restarted"
				break
			}
			if err := runner.Stop(action.ConsumerName, lifecycle.StopOptions{}); err != nil {
				return RefreshResponse{}, err
			}
			if _, err := service.StartDetailed(action.ConsumerName, lifecycle.StartOptions{}); err != nil {
				return RefreshResponse{}, err
			}
			status = "restarted"
		case packagegov.RefreshActionRebuildGoConsumer:
			rebuilt, err := rebuildGoConsumerTargets(action.Dependents, s.Stdout, s.Stderr)
			if err != nil {
				return RefreshResponse{}, err
			}
			if rebuilt {
				status = "rebuilt"
			} else {
				status = "no_buildable_target"
			}
		case packagegov.RefreshActionNoRuntimeRefresh:
			status = "no_runtime_refresh"
		case packagegov.RefreshActionNoAction:
			status = "no_action"
		}

		resp.Items = append(resp.Items, RefreshItem{
			Consumer: action.ConsumerName,
			Class:    action.ConsumerClass,
			Classes:  append([]packagegov.ConsumerClass(nil), action.ConsumerClasses...),
			Action:   action.Action,
			Status:   status,
		})
	}

	return resp, nil
}

func (s Service) runLifecycle(name, action string) (RunResponse, error) {
	if name != "" {
		item, err := s.Info(name)
		if err != nil {
			return RunResponse{}, err
		}
		if err := runPackageLifecycle(item, action, s.Stdout, s.Stderr); err != nil {
			return RunResponse{}, err
		}
		return RunResponse{PackageName: item.Name, Action: action}, nil
	}
	if action != "test" {
		return RunResponse{}, fmt.Errorf("package name is required for %s", action)
	}
	items, _, err := packagegov.LoadAll(s.Root)
	if err != nil {
		return RunResponse{}, err
	}
	for _, item := range items {
		if len(item.Manifest.Package.Lifecycle.Test) == 0 {
			continue
		}
		if err := runPackageLifecycle(item, action, s.Stdout, s.Stderr); err != nil {
			return RunResponse{}, err
		}
	}
	return RunResponse{PackageName: "all", Action: action}, nil
}

func runPackageLifecycle(item packagegov.Package, action string, stdout, stderr io.Writer) error {
	var commands []packagegov.CommandSpec
	switch action {
	case "build":
		commands = item.Manifest.Package.Lifecycle.Build
	case "generate":
		commands = item.Manifest.Package.Lifecycle.Generate
	case "test":
		commands = item.Manifest.Package.Lifecycle.Test
	default:
		return fmt.Errorf("unsupported lifecycle action: %s", action)
	}
	return packagegov.RunCommands(item.RootPath, commands, stdout, stderr)
}

func rebuildGoConsumerTargets(dependents []packagegov.Dependent, stdout, stderr io.Writer) (bool, error) {
	seen := make(map[string]struct{}, len(dependents))
	rebuilt := false
	for _, dep := range dependents {
		buildPath := filepath.Clean(dep.ConsumerPath)
		if strings.EqualFold(filepath.Base(dep.DependencyFile), "go.mod") {
			buildPath = filepath.Dir(dep.DependencyFile)
		}
		if buildPath == "." || buildPath == "" {
			continue
		}
		if _, ok := seen[buildPath]; ok {
			continue
		}
		seen[buildPath] = struct{}{}
		if _, err := os.Stat(filepath.Join(buildPath, "go.mod")); err != nil {
			continue
		}
		spec := shell.Spec{
			Name:   "go",
			Args:   []string{"build", "./..."},
			Dir:    buildPath,
			Env:    append(os.Environ(), "GOWORK=off"),
			Stdout: stdout,
			Stderr: stderr,
		}
		if err := shell.Run(spec); err != nil {
			return rebuilt, err
		}
		rebuilt = true
	}
	return rebuilt, nil
}
