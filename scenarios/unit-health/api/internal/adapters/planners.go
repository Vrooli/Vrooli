package adapters

import (
	"fmt"
	"runtime"
	"strings"
)

type Facts struct {
	Language           string
	Framework          string
	PackageManager     string
	CoverageScript     bool
	Platform           string
	TestPath           string
	Executable         string
	CoverageExecutable string
}

type Command struct {
	Executable string
	Args       []string
	Display    string
	Artifacts  []Artifact
}

type Artifact struct {
	Label string
	Kind  string
	Path  string
}

type Plan struct {
	Adapter  Identity
	Test     Command
	Coverage *Command
	TestKind string
}

type Planner interface {
	Adapter
	Plan(Facts) (Plan, error)
}

type plannerRegistry struct{ planners []Planner }

func NewPlannerRegistry() *plannerRegistry { return &plannerRegistry{} }

func (r *plannerRegistry) Register(planner Planner) error {
	if planner == nil || strings.TrimSpace(planner.Identity().ID) == "" || strings.TrimSpace(planner.Identity().Version) == "" {
		return fmt.Errorf("adapter planner: complete identity is required")
	}
	for _, existing := range r.planners {
		if existing.Identity() == planner.Identity() {
			return fmt.Errorf("adapter planner: duplicate %s@%s", planner.Identity().ID, planner.Identity().Version)
		}
	}
	r.planners = append(r.planners, planner)
	return nil
}

func (r *plannerRegistry) Resolve(facts Facts) (Plan, error) {
	platform := facts.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	facts.Platform = platform
	var matches []Planner
	for _, planner := range r.planners {
		if planner.Matches(Match{Language: facts.Language, Framework: facts.Framework, Platform: platform}) {
			matches = append(matches, planner)
		}
	}
	if len(matches) == 0 {
		return Plan{}, fmt.Errorf("adapter planner: unsupported language=%q framework=%q platform=%q", facts.Language, facts.Framework, platform)
	}
	if len(matches) > 1 {
		return Plan{}, fmt.Errorf("adapter planner: ambiguous language=%q framework=%q platform=%q", facts.Language, facts.Framework, platform)
	}
	return matches[0].Plan(facts)
}

// ValidateIdentity checks a declared adapter identity against the registered
// planner without constructing or executing a command. Policy validation uses
// this to reject stale, unknown, or platform-incompatible adapter references.
func (r *plannerRegistry) ValidateIdentity(identity Identity, match Match) error {
	if r == nil {
		return fmt.Errorf("adapter planner: registry is nil")
	}
	for _, planner := range r.planners {
		if planner.Identity() != identity {
			continue
		}
		if !planner.Matches(match) {
			return fmt.Errorf("adapter planner: adapter %s@%s does not support requested match", identity.ID, identity.Version)
		}
		return nil
	}
	return fmt.Errorf("adapter planner: unsupported adapter %s@%s", identity.ID, identity.Version)
}

func DefaultPlannerRegistry() *plannerRegistry {
	r := NewPlannerRegistry()
	for _, planner := range []Planner{goPlanner{}, nodeVitestPlanner{}, nodeJestPlanner{}, pytestPlanner{}, cargoPlanner{}, batsPlanner{}, pesterPlanner{}} {
		_ = r.Register(planner)
	}
	return r
}

type goPlanner struct{}

func (goPlanner) Identity() Identity   { return Identity{ID: "go", Version: "1.0.0"} }
func (goPlanner) Matches(m Match) bool { return strings.EqualFold(m.Language, "go") }
func (goPlanner) Plan(f Facts) (Plan, error) {
	executable := commandExecutable("go", f.Executable)
	return Plan{Adapter: (goPlanner{}).Identity(), TestKind: "unit", Test: Command{Executable: executable, Args: []string{"test", "./..."}, Display: "go test ./..."}, Coverage: &Command{Executable: executable, Args: []string{"test", "-covermode=atomic", "-coverprofile=coverage.out", "./..."}, Display: "go test -covermode=atomic -coverprofile=coverage.out ./...", Artifacts: []Artifact{{Label: "go coverage", Kind: "go-cover-profile", Path: "coverage.out"}}}}, nil
}

type nodeVitestPlanner struct{}

func (nodeVitestPlanner) Identity() Identity { return Identity{ID: "react-vitest", Version: "1.0.0"} }
func (nodeVitestPlanner) Matches(m Match) bool {
	return isNodeLanguage(m.Language) && strings.EqualFold(m.Framework, "vitest")
}

func (nodeVitestPlanner) Plan(f Facts) (Plan, error) {
	pm := packageManager(f.PackageManager)
	executable := commandExecutable(pm, f.Executable)
	coverage := (*Command)(nil)
	if f.CoverageScript {
		coverage = &Command{
			Executable: executable,
			Args:       []string{"run", "test:coverage"},
			Display:    pm + " test:coverage",
			Artifacts: []Artifact{
				{Label: "coverage summary", Kind: "istanbul-summary", Path: "coverage/coverage-summary.json"},
				{Label: "coverage lcov", Kind: "lcov", Path: "coverage/lcov.info"},
			},
		}
	}
	return Plan{Adapter: (nodeVitestPlanner{}).Identity(), TestKind: "unit", Test: Command{Executable: executable, Args: []string{"run", "test"}, Display: pm + " test"}, Coverage: coverage}, nil
}

type nodeJestPlanner struct{}

func (nodeJestPlanner) Identity() Identity { return Identity{ID: "node-jest", Version: "1.0.0"} }
func (nodeJestPlanner) Matches(m Match) bool {
	return isNodeLanguage(m.Language) && strings.EqualFold(m.Framework, "jest")
}

func (nodeJestPlanner) Plan(f Facts) (Plan, error) {
	pm := packageManager(f.PackageManager)
	executable := commandExecutable(pm, f.Executable)
	var coverage *Command
	if f.CoverageScript {
		coverage = &Command{
			Executable: executable,
			Args:       []string{"run", "test:coverage"},
			Display:    pm + " test:coverage",
			Artifacts: []Artifact{
				{Label: "coverage summary", Kind: "istanbul-summary", Path: "coverage/coverage-summary.json"},
				{Label: "coverage lcov", Kind: "lcov", Path: "coverage/lcov.info"},
			},
		}
	}
	return Plan{Adapter: (nodeJestPlanner{}).Identity(), TestKind: "unit", Test: Command{Executable: executable, Args: []string{"run", "test"}, Display: pm + " test"}, Coverage: coverage}, nil
}

type pytestPlanner struct{}

func (pytestPlanner) Identity() Identity { return Identity{ID: "python-pytest", Version: "1.0.0"} }
func (pytestPlanner) Matches(m Match) bool {
	return strings.EqualFold(m.Language, "python") && strings.EqualFold(m.Framework, "pytest")
}

func (pytestPlanner) Plan(f Facts) (Plan, error) {
	executable := "python3"
	platform := f.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	if platform == "windows" {
		executable = "py"
	}
	executable = commandExecutable(executable, f.Executable)
	var coverage *Command
	if f.CoverageScript {
		coverage = &Command{
			Executable: executable,
			Args:       []string{"-m", "pytest", "--cov=.", "--cov-report=xml:coverage.xml", "-q"},
			Display:    executable + " -m pytest --cov=. --cov-report=xml:coverage.xml -q",
			Artifacts:  []Artifact{{Label: "coverage Cobertura", Kind: "cobertura", Path: "coverage.xml"}},
		}
	}
	return Plan{Adapter: (pytestPlanner{}).Identity(), TestKind: "unit", Test: Command{Executable: executable, Args: []string{"-m", "pytest", "-q"}, Display: executable + " -m pytest -q"}, Coverage: coverage}, nil
}

type cargoPlanner struct{}

func (cargoPlanner) Identity() Identity { return Identity{ID: "rust-cargo", Version: "1.0.0"} }
func (cargoPlanner) Matches(m Match) bool {
	return strings.EqualFold(m.Language, "rust") && strings.EqualFold(m.Framework, "cargo")
}

func (cargoPlanner) Plan(f Facts) (Plan, error) {
	executable := commandExecutable("cargo", f.Executable)
	var coverage *Command
	if f.CoverageScript {
		coverage = &Command{
			Executable: commandExecutable("cargo-llvm-cov", f.CoverageExecutable),
			Args:       []string{"--workspace", "--lcov", "--output-path", "coverage/lcov.info"},
			Display:    "cargo-llvm-cov --workspace --lcov --output-path coverage/lcov.info",
			Artifacts:  []Artifact{{Label: "coverage lcov", Kind: "lcov", Path: "coverage/lcov.info"}},
		}
	}
	return Plan{Adapter: (cargoPlanner{}).Identity(), TestKind: "unit", Test: Command{Executable: executable, Args: []string{"test", "--all-targets"}, Display: "cargo test --all-targets"}, Coverage: coverage}, nil
}

type batsPlanner struct{}

func (batsPlanner) Identity() Identity { return Identity{ID: "bash-bats", Version: "1.0.0"} }
func (batsPlanner) Matches(m Match) bool {
	return strings.EqualFold(m.Language, "bash") && strings.EqualFold(m.Framework, "bats") && m.Platform != "windows"
}

func (batsPlanner) Plan(f Facts) (Plan, error) {
	executable := f.Executable
	if executable == "" {
		executable = "bats"
	}
	return Plan{Adapter: (batsPlanner{}).Identity(), TestKind: "unit", Test: Command{Executable: executable, Args: []string{"--recursive", "."}, Display: executable + " --recursive ."}}, nil
}

type pesterPlanner struct{}

func (pesterPlanner) Identity() Identity { return Identity{ID: "powershell-pester", Version: "1.0.0"} }
func (pesterPlanner) Matches(m Match) bool {
	return strings.EqualFold(m.Language, "powershell") && strings.EqualFold(m.Framework, "pester") && m.Platform == "windows"
}

func (pesterPlanner) Plan(f Facts) (Plan, error) {
	if strings.TrimSpace(f.TestPath) == "" {
		return Plan{}, fmt.Errorf("pester adapter: test path is required")
	}
	return Plan{Adapter: (pesterPlanner{}).Identity(), TestKind: "unit", Test: Command{Executable: commandExecutable("pwsh", f.Executable), Args: []string{"-NoProfile", "-NonInteractive", "-File", f.TestPath, "-CI"}, Display: "pwsh -NoProfile -NonInteractive -File <test-path> -CI"}}, nil
}

func commandExecutable(fallback, resolved string) string {
	if strings.TrimSpace(resolved) != "" {
		return resolved
	}
	return fallback
}

func packageManager(preferred string) string {
	switch strings.ToLower(preferred) {
	case "npm", "yarn", "pnpm":
		return strings.ToLower(preferred)
	default:
		return "pnpm"
	}
}

func isNodeLanguage(language string) bool {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript", "javascript", "node":
		return true
	default:
		return false
	}
}
