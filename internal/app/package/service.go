package packageapp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/envkit-go"
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
	// TestGenieRunner owns package-target test execution when the control-plane
	// CLI is wired to the server-owned Test Genie runtime.
	TestGenieRunner func(target string, stdout, stderr io.Writer) error
}

type RunResponse struct {
	PackageName string `json:"package_name"`
	Action      string `json:"action"`
}

type RefreshRequest struct {
	PackageName string
	Target      string
	NoRestart   bool
	Interactive bool
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

func (s Service) Build(name string) (RunResponse, error) {
	return s.runLifecycle(name, "build")
}

func (s Service) Generate(name string) (RunResponse, error) {
	return s.runLifecycle(name, "generate")
}

func (s Service) Test(name string) (RunResponse, error) {
	if s.TestGenieRunner != nil {
		if strings.TrimSpace(name) == "" {
			return RunResponse{}, fmt.Errorf("package name is required for test")
		}
		if err := s.TestGenieRunner("package:"+strings.TrimSpace(name), s.Stdout, s.Stderr); err != nil {
			return RunResponse{}, err
		}
		return RunResponse{PackageName: name, Action: "test-genie"}, nil
	}
	return s.runLifecycle(name, "test")
}

func (s Service) Refresh(req RefreshRequest) (RefreshResponse, error) {
	noRestart := req.NoRestart || !req.Interactive
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
	runtime := refreshRuntime{owner: s, item: item, noRestart: noRestart}
	for _, action := range actions {
		status, err := runtime.execute(action)
		if err != nil {
			return RefreshResponse{}, err
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

type refreshRuntime struct {
	owner     Service
	item      packagegov.Package
	noRestart bool
	service   ScenarioRuntime
	runner    ScenarioPhaseRunner
}

func (r *refreshRuntime) scenarioService() (ScenarioRuntime, error) {
	if r.service != nil {
		return r.service, nil
	}
	if r.owner.ScenarioService == nil {
		return nil, fmt.Errorf("scenario service is not configured")
	}
	service, err := r.owner.ScenarioService()
	if err == nil {
		r.service = service
	}
	return service, err
}

func (r *refreshRuntime) scenarioRunner() (ScenarioPhaseRunner, error) {
	if r.runner != nil {
		return r.runner, nil
	}
	if r.owner.ScenarioRunner == nil {
		return nil, fmt.Errorf("scenario runner is not configured")
	}
	runner, err := r.owner.ScenarioRunner()
	if err == nil {
		r.runner = runner
	}
	return runner, err
}

func (r *refreshRuntime) execute(action packagegov.RefreshAction) (string, error) {
	switch action.Action {
	case packagegov.RefreshActionScenarioSetup:
		return r.setupScenario(action.ConsumerName)
	case packagegov.RefreshActionRestartScenario:
		return r.restartScenario(action.ConsumerName)
	case packagegov.RefreshActionRebuildGoConsumer:
		rebuilt, err := rebuildGoConsumerTargets(action.Dependents, r.owner.Stdout, r.owner.Stderr)
		if err != nil {
			return "", err
		}
		if rebuilt {
			return "rebuilt", nil
		}
		return "no_buildable_target", nil
	case packagegov.RefreshActionNoRuntimeRefresh:
		return "no_runtime_refresh", nil
	default:
		return "no_action", nil
	}
}

func (r *refreshRuntime) setupScenario(name string) (string, error) {
	service, err := r.scenarioService()
	if err != nil {
		return "", err
	}
	runner, err := r.scenarioRunner()
	if err != nil {
		return "", err
	}
	detail, _, err := service.Lookup(name)
	if err != nil {
		return "", err
	}
	wasRunning := detail.Runtime.ProcessCount > 0
	if wasRunning {
		if err := runner.Stop(name, lifecycle.StopOptions{}); err != nil {
			return "", err
		}
	}
	if _, err := runner.RunPhaseDetailed(name, "setup", lifecycle.PhaseOptions{}); err != nil {
		return "", err
	}
	if wasRunning && !r.noRestart && r.item.Manifest.Package.Refresh.RestartRunningConsumers {
		if _, err := service.StartDetailed(name, lifecycle.StartOptions{}); err != nil {
			return "", err
		}
		return "restarted", nil
	}
	if wasRunning {
		return "stopped_after_setup", nil
	}
	return "setup_only", nil
}

func (r *refreshRuntime) restartScenario(name string) (string, error) {
	service, err := r.scenarioService()
	if err != nil {
		return "", err
	}
	runner, err := r.scenarioRunner()
	if err != nil {
		return "", err
	}
	detail, _, err := service.Lookup(name)
	if err != nil {
		return "", err
	}
	if detail.Runtime.ProcessCount == 0 {
		return "not_running", nil
	}
	if r.noRestart || !r.item.Manifest.Package.Refresh.RestartRunningConsumers {
		return "running_not_restarted", nil
	}
	if err := runner.Stop(name, lifecycle.StopOptions{}); err != nil {
		return "", err
	}
	if _, err := service.StartDetailed(name, lifecycle.StartOptions{}); err != nil {
		return "", err
	}
	return "restarted", nil
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
			Env:    envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"GOWORK=off"}),
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
