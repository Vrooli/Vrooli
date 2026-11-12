package resourcehandlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/resources"
)

type resourceDisableLightingFixture struct {
	t           *testing.T
	c           *resources.Controller
	intent      *lightingruntime.Controller
	calls       []string
	stopErr     error
	stop        func(context.Context) error
	verifyCalls int
}

func (f *resourceDisableLightingFixture) VerifyLightingSetup(context.Context, lightingruntime.Selection) error {
	f.verifyCalls++
	return errors.New("disable must not qualify a new launch")
}
func (f *resourceDisableLightingFixture) EnsureOpenRGB(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error) {
	f.t.Fatal("disable started OpenRGB")
	return lightingruntime.Backend{}, nil
}
func (f *resourceDisableLightingFixture) EnsureLightingReconciler(context.Context, lightingruntime.Backend) error {
	f.t.Fatal("disable started Light Desk")
	return nil
}
func (f *resourceDisableLightingFixture) QuiesceLightingReconciler(context.Context) error {
	state, err := f.intent.Current()
	if err != nil || state.Phase != lightingruntime.Disabling {
		f.t.Fatalf("writer quiesce preceded durable intent: %+v %v", state, err)
	}
	f.calls = append(f.calls, "quiesce")
	return nil
}
func (f *resourceDisableLightingFixture) DisableOpenRGB(ctx context.Context) error {
	// The concrete resource owner has separate whole-owner retirement tests.
	// Preserve its persist-disabled-before-native-stop contract here.
	if err := f.c.SetEnabled("openrgb", false); err != nil {
		return err
	}
	f.calls = append(f.calls, "resource-disabled", "native-stop")
	if f.stop != nil {
		return f.stop(ctx)
	}
	return f.stopErr
}

func resourceDisableLightingHandlerFixture(t *testing.T) (rootcli.Handler[testContext], testContext, *resourceDisableLightingFixture) {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", t.TempDir())
	for _, key := range []string{"SUDO_USER", "SUDO_UID", "SUDO_GID"} {
		t.Setenv(key, "")
	}
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	for _, name := range []string{operatorstate.SchemaPath, ".vrooli/schemas/common.schema.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c := resources.NewController(root, home)
	if err := c.SetEnabled("openrgb", true); err != nil {
		t.Fatal(err)
	}
	f := &resourceDisableLightingFixture{t: t, c: c}
	f.intent, _ = lightingruntime.NewController(home, lightingruntime.Owners{Resource: f, Scenario: f})
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{
		Stdout:             func(c testContext) io.Writer { return c.stdout },
		Stderr:             func(c testContext) io.Writer { return c.stderr },
		ResourceController: func(testContext) (*resources.Controller, error) { return c, nil },
	}
	handlers := buildResourceCommandHandlers(deps)
	handlers["disable"] = resourceToggleHandlerWithLighting(deps, false, runtimeapp.Service{Lighting: &lightingruntime.Owners{Resource: f, Scenario: f}})
	return rootHandlerWithCommands(deps, handlers), ctx, f
}

func TestResourceDisableOpenRGBNormalEntryRetiresAndRetainsFailedStopIntent(t *testing.T) {
	run, ctx, f := resourceDisableLightingHandlerFixture(t)
	for _, args := range [][]string{{"disable"}, {"disable", "openrgb", "--unexpected"}} {
		if err := run(ctx, args); err == nil || len(f.calls) != 0 {
			t.Fatalf("invalid CLI arguments reached cleanup: %v %v", args, err)
		}
	}
	f.stopErr = errors.New("native tree remains unconfirmed")
	parsed, err := rootcli.ParseArgs([]string{"resource", "disable", "openrgb", "--json"})
	if err != nil || parsed.Command != "resource" {
		t.Fatalf("normal root parsing failed: %+v %v", parsed, err)
	}
	globals, args, warnings := rootcli.ConsumeInlineGlobalFlags(parsed.Globals, parsed.Args)
	if !globals.JSON || len(warnings) != 0 {
		t.Fatal("normal resource flags were not handled by the canonical root parser")
	}
	if err := run(ctx, args); !errors.Is(err, f.stopErr) {
		t.Fatalf("normal entry lost failed stop: %v", err)
	}
	state, err := operatorstate.New(operatorstate.Config{RepoRoot: f.c.Root}).Load(t.Context())
	if err != nil || state.Resources["openrgb"].Enabled == nil || *state.Resources["openrgb"].Enabled {
		t.Fatalf("stop failure restored resource enablement: %+v %v", state, err)
	}
	intent, err := f.intent.Current()
	if err != nil || intent.Phase != lightingruntime.Disabling {
		t.Fatalf("stop failure was reported as cleaned up: %+v %v", intent, err)
	}
	f.stopErr = nil
	for range 2 {
		if err := run(ctx, []string{"disable", "openrgb"}); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Join(f.calls, ","); got != strings.TrimSuffix(strings.Repeat("quiesce,resource-disabled,native-stop,", 3), ",") {
		t.Fatalf("normal disable used a different lifecycle: %s", got)
	}
	if f.verifyCalls != 0 {
		t.Fatal("disable qualified or launched a backend")
	}
}

func TestResourceDisableOpenRGBHoldsCanonicalIntentFenceThroughNativeStop(t *testing.T) {
	run, ctx, f := resourceDisableLightingHandlerFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	f.stop = func(context.Context) error { close(entered); <-release; return nil }
	done := make(chan error, 1)
	go func() { done <- run(ctx, []string{"disable", "openrgb"}) }()
	<-entered
	blocked, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	err := f.intent.Enable(blocked, lightingruntime.Selection{Version: "1.1", SHA256: strings.Repeat("a", 64), DevicePolicyID: "fixture"})
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) || f.verifyCalls != 0 {
		t.Fatalf("enable crossed resource disable: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
