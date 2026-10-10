package lifecycle

import (
	"errors"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

// hostTreeFixture writes alpha -> beta where beta declares a required,
// develop-only host tool the host cannot provide (the 2026-10-07
// scenario-to-desktop webkitgtk declaration that blocked every start whose
// tree reached it).
func hostTreeFixture(t *testing.T) (root, home string) {
	t.Helper()
	root, home = t.TempDir(), t.TempDir()
	beta := lifecycleFixtureManifest("beta")
	beta.HostTools = []hostreqspec.Declaration{{Name: "webkitgtk", Required: true, Reason: "native desktop toolkit", When: []string{"setup", "develop"}}}
	writeLifecycleFixtureManifest(t, root, beta)
	alpha := lifecycleFixtureManifest("alpha")
	alpha.Dependencies.Scenarios = map[string]scenario.Dependency{"beta": {Required: true}}
	writeLifecycleFixtureManifest(t, root, alpha)
	return root, home
}

// unsupportedOn fails enforcement whenever a preflight includes the named
// scenario directory, the way the host runtime refuses an unsupported tool.
func unsupportedOn(root, name string, seen *[][]string) func(vrooliruntime.Options) (vrooliruntime.Report, error) {
	blocked := filepath.Join(root, "scenarios", name)
	return func(opts vrooliruntime.Options) (vrooliruntime.Report, error) {
		*seen = append(*seen, append([]string(nil), opts.ScenarioPaths...))
		if slices.Contains(opts.ScenarioPaths, blocked) {
			return vrooliruntime.Report{}, errors.New("webkitgtk: no native runtime handler registered")
		}
		return vrooliruntime.Report{}, nil
	}
}

// A dependency that is already serving is neither built nor started by the
// consumer's start, so its develop-time host requirements must not block it.
func TestStartDoesNotPreflightServingDependencyHostRequirements(t *testing.T) {
	if runtime.GOOS != "linux" {
		testkitgo.SkipPlatform(t, "lifecycle process management currently targets linux")
	}
	root, home := hostTreeFixture(t)
	var seen [][]string
	permissive := newLifecycleRunnerForTest(t, root, home, nil)
	if _, err := permissive.Start("beta", StartOptions{}); err != nil {
		t.Fatalf("start beta: %v", err)
	}
	cleanupRunner(t, permissive, "beta", StopOptions{})

	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.enforceHostRequirements = unsupportedOn(root, "beta", &seen)
	})
	cleanupRunner(t, runner, "alpha", StopOptions{})
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("start alpha with a serving dependency: %v (preflights %v)", err, seen)
	}
	if _, err := runner.Restart("alpha", StartOptions{}); err != nil {
		t.Fatalf("restart alpha with a serving dependency: %v (preflights %v)", err, seen)
	}
}

// A dependency the start must bring up is preflighted, and its unsupported
// requirement fails the start before anything runs.
func TestStartPreflightsDependencyThatMustStart(t *testing.T) {
	if runtime.GOOS != "linux" {
		testkitgo.SkipPlatform(t, "lifecycle process management currently targets linux")
	}
	root, home := hostTreeFixture(t)
	var seen [][]string
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.enforceHostRequirements = unsupportedOn(root, "beta", &seen)
	})
	cleanupRunner(t, runner, "alpha", StopOptions{})
	cleanupRunner(t, runner, "beta", StopOptions{})
	_, err := runner.Start("alpha", StartOptions{})
	if err == nil || !strings.Contains(err.Error(), "no native runtime handler") {
		t.Fatalf("start with a stopped dependency that needs an unsupported tool: %v", err)
	}
	if len(seen) == 0 || !slices.Contains(seen[0], filepath.Join(root, "scenarios", "alpha")) {
		t.Fatalf("the root scenario must stay in the develop preflight: %v", seen)
	}
}

// The root scenario keeps its develop-time requirements even when it is
// already running: a restart rebuilds it.
func TestRestartPreflightsRootBeforeStopping(t *testing.T) {
	if runtime.GOOS != "linux" {
		testkitgo.SkipPlatform(t, "lifecycle process management currently targets linux")
	}
	root, home := t.TempDir(), t.TempDir()
	alpha := lifecycleFixtureManifest("alpha")
	alpha.HostTools = []hostreqspec.Declaration{{Name: "webkitgtk", Required: true, Reason: "native desktop toolkit", When: []string{"setup", "develop"}}}
	writeLifecycleFixtureManifest(t, root, alpha)
	permissive := newLifecycleRunnerForTest(t, root, home, nil)
	if _, err := permissive.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("start alpha: %v", err)
	}
	cleanupRunner(t, permissive, "alpha", StopOptions{})
	before := liveScenarioPID(t, home, "alpha")

	var seen [][]string
	strict := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.enforceHostRequirements = unsupportedOn(root, "alpha", &seen)
	})
	if _, err := strict.Restart("alpha", StartOptions{}); err == nil || !strings.Contains(err.Error(), "was not stopped") {
		t.Fatalf("restart with an unsupported root requirement: %v", err)
	}
	if after := liveScenarioPID(t, home, "alpha"); after != before {
		t.Fatalf("failed preflight stopped the running instance: pid %d -> %d", before, after)
	}
}
