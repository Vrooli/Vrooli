package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// legacyConsumer builds a scenario whose ui/node_modules was linked against the
// pre-v10 pnpm store layout (store/v3) that this pnpm selects as store/v3/v10, so
// the recorded store cannot be reused. The fake pnpm logs installs.
type legacyConsumer struct {
	root, modules, installLog string
	env                       map[string]string
	step                      scenario.PhaseStep
	item                      scenario.Scenario
}

func newLegacyConsumer(t *testing.T) legacyConsumer {
	t.Helper()
	bin, root := t.TempDir(), t.TempDir()
	log := filepath.Join(root, "pnpm.log")
	script := "#!/bin/sh\n" +
		"base=${npm_config_store_dir:-/user/config/store/v3}\n" +
		"if [ \"$1\" = store ]; then echo \"$base/v10\"; exit 0; fi\n" +
		"echo \"$@\" >> " + log + "\n"
	if err := os.WriteFile(filepath.Join(bin, "pnpm"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	modules := filepath.Join(root, "ui", "node_modules")
	if err := os.MkdirAll(modules, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "{\n  \"storeDir\": \"/home/u/.local/share/pnpm/store/v3\"\n}\n"
	if err := os.WriteFile(filepath.Join(modules, ".modules.yaml"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	return legacyConsumer{
		root: root, modules: modules, installLog: log,
		env:  map[string]string{"PATH": bin + string(os.PathListSeparator) + os.Getenv("PATH")},
		step: scenario.PhaseStep{Name: "install-ui", Exec: []string{filepath.Join(bin, "pnpm"), "install", "--ignore-workspace"}, CWD: "ui"},
		item: scenario.Scenario{Slug: "demo", Path: root},
	}
}

func registryRunner(t *testing.T, open func(context.Context, string) (scenarioRuntimeStore, error)) (*Runner, string) {
	home := t.TempDir()
	if open == nil {
		open = func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
			return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		}
	}
	return &Runner{Home: home, deps: lifecycleDeps{runtimeRegistry: open}}, home
}

func (c legacyConsumer) run(r *Runner, phase string) (string, error) {
	return c.runIn(context.Background(), r, phase)
}

func (c legacyConsumer) runIn(ctx context.Context, r *Runner, phase string) (string, error) {
	var out strings.Builder
	err := r.runForegroundStep(ctx, c.item, phase, c.step, c.env, &out)
	return out.String(), err
}

// occupy registers an instance of the scenario in the registry and returns its id.
func occupy(t *testing.T, home string, status string) string {
	t.Helper()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	pid := 4242
	instance, err := store.CreateLease(context.Background(), scenarioruntime.Instance{
		Scenario: "demo", BuildIdentity: "b", Status: status, OwnerPID: &pid, HostBootID: "boot",
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		t.Fatal(err)
	}
	return instance.InstanceID
}

func (c legacyConsumer) assertRecreated(t *testing.T, out string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(c.modules, ".modules.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy node_modules still present: %v", err)
	}
	if logged, _ := os.ReadFile(c.installLog); !strings.Contains(string(logged), "install --ignore-workspace") {
		t.Fatalf("install did not run after the repair: %q\n%s", logged, out)
	}
	asides, _ := filepath.Glob(c.modules + staleModulesInfix + "*")
	if len(asides) != 0 {
		t.Fatalf("the set-aside copy must be deleted after a successful install: %v", asides)
	}
}

func (c legacyConsumer) assertKept(t *testing.T, err error, out, wantReason string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), wantReason) {
		t.Fatalf("err = %v, want it to say %q\n%s", err, wantReason, out)
	}
	if !strings.Contains(err.Error(), "cannot be reused") {
		t.Fatalf("the original store mismatch must stay in the error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(c.modules, ".modules.yaml")); statErr != nil {
		t.Fatalf("node_modules must be untouched when recreation is refused: %v", statErr)
	}
	if _, statErr := os.Stat(c.installLog); statErr == nil {
		t.Fatal("pnpm install must not run after a refused recreation")
	}
}

func TestSetupRecreatesNodeModulesOfAStoppedConsumerLinkedAgainstALegacyStore(t *testing.T) {
	c := newLegacyConsumer(t)
	r, _ := registryRunner(t, nil)

	out, err := c.run(r, phasesSetup)
	if err != nil {
		t.Fatalf("runForegroundStep: %v\n%s", err, out)
	}
	c.assertRecreated(t, out)
	if !strings.Contains(out, "scenario demo is stopped") {
		t.Fatalf("the step log must say why node_modules was set aside: %q", out)
	}
}

// A start registers its own starting instance before it builds. That instance is
// the operation itself, so it must not block the repair that start needs.
func TestStartRecreatesNodeModulesWhileItsOwnInstanceIsStarting(t *testing.T) {
	c := newLegacyConsumer(t)
	r, home := registryRunner(t, nil)
	own := occupy(t, home, scenarioruntime.StatusStarting)

	out, err := c.runIn(withOwnedInstances(context.Background(), own), r, phasesSetup)
	if err != nil {
		t.Fatalf("runForegroundStep: %v\n%s", err, out)
	}
	c.assertRecreated(t, out)
}

// A restart builds the replacement while the old instance serves. Its own running
// instance is owned by the restart, and the repair says so in the log.
func TestRestartRecreatesNodeModulesOfTheInstanceItIsReplacing(t *testing.T) {
	c := newLegacyConsumer(t)
	r, home := registryRunner(t, nil)
	own := occupy(t, home, scenarioruntime.StatusRunning)

	out, err := c.runIn(withOwnedInstances(context.Background(), own), r, phasesSetup)
	if err != nil {
		t.Fatalf("runForegroundStep: %v\n%s", err, out)
	}
	c.assertRecreated(t, out)
	if !strings.Contains(out, "serving and being replaced by this restart") {
		t.Fatalf("the log must say a serving instance was involved: %q", out)
	}
}

// A failed install must not cost the consumer the node_modules it was serving from.
func TestFailedInstallRestoresTheLegacyNodeModules(t *testing.T) {
	c := newLegacyConsumer(t)
	r, home := registryRunner(t, nil)
	own := occupy(t, home, scenarioruntime.StatusRunning)
	failing := "#!/bin/sh\nbase=${npm_config_store_dir:-/user/config/store/v3}\nif [ \"$1\" = store ]; then echo \"$base/v10\"; exit 0; fi\nexit 7\n"
	if err := os.WriteFile(c.step.Exec[0], []byte(failing), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := c.runIn(withOwnedInstances(context.Background(), own), r, phasesSetup)

	if err == nil {
		t.Fatalf("a failing install must fail the step\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(c.modules, ".modules.yaml")); statErr != nil {
		t.Fatalf("the legacy node_modules must be restored after a failed install: %v\n%s", statErr, out)
	}
	if asides, _ := filepath.Glob(c.modules + staleModulesInfix + "*"); len(asides) != 0 {
		t.Fatalf("restoring must not leave the set-aside copy behind: %v", asides)
	}
}

// A running instance this operation does not own (another start, a shadow variant,
// a plain setup beside a live site) may be serving from node_modules.
func TestSetupKeepsNodeModulesOfAnInstanceTheOperationDoesNotOwn(t *testing.T) {
	c := newLegacyConsumer(t)
	r, home := registryRunner(t, nil)
	occupy(t, home, scenarioruntime.StatusRunning)

	out, err := c.run(r, phasesSetup)

	c.assertKept(t, err, out, "demo@live that this operation does not own")
}

func TestSetupKeepsNodeModulesWhenAnotherInstanceJoinsTheOwnedOne(t *testing.T) {
	c := newLegacyConsumer(t)
	r, home := registryRunner(t, nil)
	own := occupy(t, home, scenarioruntime.StatusStarting)
	occupy(t, home, scenarioruntime.StatusRunning)

	out, err := c.runIn(withOwnedInstances(context.Background(), own), r, phasesSetup)

	c.assertKept(t, err, out, "does not own")
}

func TestSetupKeepsNodeModulesWhenTheRegistryCannotConfirmTheConsumerIsStopped(t *testing.T) {
	c := newLegacyConsumer(t)
	r, _ := registryRunner(t, func(context.Context, string) (scenarioRuntimeStore, error) {
		return nil, errors.New("registry unavailable")
	})

	out, err := c.run(r, phasesSetup)

	c.assertKept(t, err, out, "could not confirm scenario demo has no other occupant")
}

func TestOnlySetupRecreatesNodeModulesOfAConsumerLinkedAgainstALegacyStore(t *testing.T) {
	c := newLegacyConsumer(t)
	r, _ := registryRunner(t, nil)

	out, err := c.run(r, "develop")

	c.assertKept(t, err, out, "only recreated during setup")
}

func TestRecreationRefusesASymlinkedNodeModules(t *testing.T) {
	c := newLegacyConsumer(t)
	r, _ := registryRunner(t, nil)
	real := filepath.Join(c.root, "elsewhere")
	if err := os.Rename(c.modules, real); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, c.modules); err != nil {
		t.Fatal(err)
	}

	out, err := c.run(r, phasesSetup)

	if err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("err = %v, want a refusal for a symlinked node_modules\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(real, ".modules.yaml")); statErr != nil {
		t.Fatalf("the symlink target must survive: %v", statErr)
	}
}
