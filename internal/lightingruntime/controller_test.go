package lightingruntime

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type lightingFixture struct {
	calls      []string
	selection  Selection
	backend    Backend
	bound      Backend
	verifyErr  error
	backendErr error
	bindErr    error
	stopErr    error
	quiesceErr error
	ensure     func(context.Context) error
	quiesce    func(context.Context) error
}

func (f *lightingFixture) VerifyLightingSetup(_ context.Context, selection Selection) error {
	f.calls = append(f.calls, "verify")
	f.selection = selection
	return f.verifyErr
}

func (f *lightingFixture) EnsureOpenRGB(ctx context.Context, selection Selection) (Backend, error) {
	f.calls = append(f.calls, "backend")
	f.selection = selection
	if f.ensure != nil {
		if err := f.ensure(ctx); err != nil {
			return Backend{}, err
		}
	}
	return f.backend, f.backendErr
}

func (f *lightingFixture) DisableOpenRGB(context.Context) error {
	f.calls = append(f.calls, "stop")
	return f.stopErr
}

func (f *lightingFixture) EnsureLightingReconciler(_ context.Context, backend Backend) error {
	f.calls = append(f.calls, "scenario:"+backend.Incarnation)
	if backend != f.backend {
		return errors.New("unexpected backend endpoint or identity")
	}
	if f.bindErr != nil {
		return f.bindErr
	}
	f.bound = backend
	return nil
}

func (f *lightingFixture) QuiesceLightingReconciler(ctx context.Context) error {
	f.calls = append(f.calls, "quiesce")
	if f.quiesce != nil {
		if err := f.quiesce(ctx); err != nil {
			return err
		}
	}
	if f.quiesceErr != nil {
		return f.quiesceErr
	}
	f.bound = Backend{}
	return nil
}

func newLightingFixture(t *testing.T) (string, *Controller, *lightingFixture) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home) // The production owned writer also resolves its lock under this home.
	t.Setenv("SUDO_USER", "")
	fixture := &lightingFixture{backend: Backend{
		Endpoint: netip.MustParseAddrPort("[::1]:6742"), Incarnation: "boot-a:process-a",
	}}
	controller, err := NewController(home, Owners{Resource: fixture, Scenario: fixture})
	if err != nil {
		t.Fatal(err)
	}
	return home, controller, fixture
}

func selectedLighting() Selection {
	return Selection{Version: "1.1", SHA256: strings.Repeat("a", 64), DevicePolicyID: "fixture-device-policy"}
}

func expectLightingCalls(t *testing.T, fixture *lightingFixture, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(fixture.calls, want) {
		t.Fatalf("owner calls = %v, want %v", fixture.calls, want)
	}
}

func TestHeadlessLightingDefaultIsInactiveAndWritesNothing(t *testing.T) {
	_, controller, fixture := newLightingFixture(t)
	if err := controller.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	state, err := controller.Current()
	if err != nil || state.Phase != Disabled || state.Selection != nil {
		t.Fatalf("fresh intent = %+v, %v", state, err)
	}
	if _, err := os.Lstat(filepath.Dir(controller.path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reconcile wrote on a clean host: %v", err)
	}
	expectLightingCalls(t, fixture)
}

func TestHeadlessLightingRequiresSetupAndRestoresFromNativeOwnerRestart(t *testing.T) {
	home, controller, fixture := newLightingFixture(t)
	selection := selectedLighting()
	fixture.verifyErr = errors.New("qualified device access is unavailable")
	if err := controller.Enable(t.Context(), selection); !errors.Is(err, fixture.verifyErr) {
		t.Fatalf("unqualified setup = %v", err)
	}
	if err := controller.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	expectLightingCalls(t, fixture, "verify")
	fixture.verifyErr = nil
	if err := controller.Enable(t.Context(), selection); err != nil {
		t.Fatal(err)
	}
	state, err := controller.Current()
	if err != nil || state.Phase != Enabled || state.Selection == nil || *state.Selection != selection {
		t.Fatalf("enabled intent = %+v, %v", state, err)
	}
	expectLightingCalls(t, fixture, "verify", "verify")
	boot := &lightingFixture{backend: fixture.backend, backendErr: errors.New("verified staged artifact is missing")}
	resumed, err := NewController(home, Owners{Resource: boot, Scenario: boot})
	if err != nil {
		t.Fatal(err)
	}
	if err := resumed.Reconcile(t.Context()); !errors.Is(err, boot.backendErr) {
		t.Fatalf("missing artifact = %v", err)
	}
	expectLightingCalls(t, boot, "backend", "quiesce")
	boot.backendErr = nil
	if err := resumed.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	if boot.selection != selection {
		t.Fatalf("startup did not retain exact selection: %+v", boot.selection)
	}
	boot.backend.Endpoint = netip.MustParseAddrPort("127.0.0.1:6742")
	boot.backend.Incarnation = "boot-b:process-b"
	if err := resumed.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	expectLightingCalls(t, boot, "backend", "quiesce", "backend", "scenario:boot-a:process-a", "backend", "scenario:boot-b:process-b")

	boot.calls = nil
	boot.verifyErr = errors.New("replacement artifact is unverified")
	replacement := selection
	replacement.Version = "1.2"
	if err := resumed.Enable(t.Context(), replacement); !errors.Is(err, boot.verifyErr) {
		t.Fatalf("replacement setup failure = %v", err)
	}
	state, err = resumed.Current()
	if err != nil || state.Phase != Enabled || state.Selection == nil || *state.Selection != selection || boot.bound != boot.backend {
		t.Fatalf("rejected replacement changed active intent or binding: %+v, %+v, %v", state, boot.bound, err)
	}
	expectLightingCalls(t, boot, "verify")
}

func TestHeadlessLightingFailsClosedWithoutBothOwnersOrLoopbackBinding(t *testing.T) {
	home, controller, fixture := newLightingFixture(t)
	unwired, err := NewController(home, Owners{Resource: fixture})
	if err != nil {
		t.Fatal(err)
	}
	if err := unwired.Enable(t.Context(), selectedLighting()); err == nil {
		t.Fatal("enabled without scenario owner")
	}
	expectLightingCalls(t, fixture)
	if err := controller.Enable(t.Context(), selectedLighting()); err != nil {
		t.Fatal(err)
	}
	if err := unwired.Reconcile(t.Context()); err == nil {
		t.Fatal("reconciled without scenario owner")
	}
	for _, endpoint := range []string{"0.0.0.0:6742", "192.0.2.1:6742", "[::]:6742", "[::1]:0", "[::1%fixture]:6742"} {
		fixture.calls = nil
		fixture.backend.Endpoint = netip.MustParseAddrPort(endpoint)
		if err := controller.Reconcile(t.Context()); err == nil {
			t.Fatalf("invalid endpoint %s was used", endpoint)
		}
		expectLightingCalls(t, fixture, "backend", "quiesce")
	}
	fixture.calls = nil
	fixture.backend.Endpoint = netip.MustParseAddrPort("127.0.0.1:6742")
	fixture.backend.Incarnation = ""
	if err := controller.Reconcile(t.Context()); err == nil {
		t.Fatal("missing backend incarnation accepted")
	}
	expectLightingCalls(t, fixture, "backend", "quiesce")
}

func TestHeadlessLightingQuiescesStaleBindingAndRebindsAfterRecovery(t *testing.T) {
	for _, failure := range []string{"owner failure", "invalid binding", "scenario binding failure"} {
		t.Run(failure, func(t *testing.T) {
			_, controller, fixture := newLightingFixture(t)
			selection := selectedLighting()
			if err := controller.Enable(t.Context(), selection); err != nil {
				t.Fatal(err)
			}
			if err := controller.Reconcile(t.Context()); err != nil || fixture.bound != fixture.backend {
				t.Fatalf("initial writer binding = %+v, %v", fixture.bound, err)
			}
			previous := fixture.bound
			fixture.backend = Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742"), Incarnation: "boot-b:process-b"}
			cause := errors.New("cannot establish the exact backend binding")
			wantCalls := []string{"backend", "quiesce"}
			switch failure {
			case "owner failure":
				fixture.backendErr = cause
			case "invalid binding":
				fixture.backend.Endpoint = netip.MustParseAddrPort("192.0.2.1:6742")
			case "scenario binding failure":
				fixture.bindErr = cause
				wantCalls = []string{"backend", "scenario:boot-b:process-b", "quiesce"}
			}
			fixture.quiesceErr = errors.New("cannot confirm the writer is quiesced")
			fixture.calls = nil
			err := controller.Reconcile(t.Context())
			if !errors.Is(err, fixture.quiesceErr) || (failure != "invalid binding" && !errors.Is(err, cause)) ||
				(failure == "invalid binding" && !strings.Contains(err.Error(), "no verified loopback process binding")) {
				t.Fatalf("reconcile must report both failures, got %v", err)
			}
			if fixture.bound != previous {
				t.Fatalf("fixture lost binding despite a failed quiesce: %+v", fixture.bound)
			}
			expectLightingCalls(t, fixture, wantCalls...)

			fixture.quiesceErr = nil
			fixture.calls = nil
			if err := controller.Reconcile(t.Context()); err == nil || fixture.bound != (Backend{}) {
				t.Fatalf("unverified backend left writer bound: %+v, %v", fixture.bound, err)
			}
			expectLightingCalls(t, fixture, wantCalls...)
			state, err := controller.Current()
			if err != nil || state.Phase != Enabled || state.Selection == nil || *state.Selection != selection {
				t.Fatalf("backend failure changed durable enablement: %+v, %v", state, err)
			}

			fixture.backend.Endpoint = netip.MustParseAddrPort("127.0.0.1:6742")
			if failure == "owner failure" {
				fixture.backend = previous
			}
			fixture.backendErr, fixture.bindErr, fixture.calls = nil, nil, nil
			if err := controller.Reconcile(t.Context()); err != nil || fixture.bound != fixture.backend {
				t.Fatalf("recovered writer binding = %+v, %v", fixture.bound, err)
			}
			expectLightingCalls(t, fixture, "backend", "scenario:"+fixture.backend.Incarnation)
		})
	}
}

func TestHeadlessLightingQuiescesAfterCanceledBackendCheck(t *testing.T) {
	_, controller, fixture := newLightingFixture(t)
	if err := controller.Enable(t.Context(), selectedLighting()); err != nil {
		t.Fatal(err)
	}
	if err := controller.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	fixture.ensure = func(ctx context.Context) error {
		cancel()
		return ctx.Err()
	}
	fixture.quiesce = func(ctx context.Context) error {
		if _, bounded := ctx.Deadline(); !bounded {
			t.Fatal("quiesce has no deadline")
		}
		return ctx.Err()
	}
	fixture.calls = nil
	if err := controller.Reconcile(ctx); !errors.Is(err, context.Canceled) || fixture.bound != (Backend{}) {
		t.Fatalf("canceled check left writer bound: %+v, %v", fixture.bound, err)
	}
	expectLightingCalls(t, fixture, "backend", "quiesce")
}

func TestHeadlessLightingDisableResumesFromDurableFence(t *testing.T) {
	home, controller, fixture := newLightingFixture(t)
	if err := controller.Enable(t.Context(), selectedLighting()); err != nil {
		t.Fatal(err)
	}
	fixture.quiesce = func(context.Context) error {
		state, err := controller.Current()
		if err != nil || state.Phase != Disabling {
			t.Fatalf("disable intent was not durable before quiescing: %+v, %v", state, err)
		}
		return errors.New("quiesce interrupted")
	}
	if err := controller.Disable(t.Context()); err == nil {
		t.Fatal("disable accepted failed quiesce")
	}
	expectLightingCalls(t, fixture, "verify", "quiesce")
	if err := controller.Enable(t.Context(), selectedLighting()); err == nil {
		t.Fatal("enable bypassed pending disable")
	}
	boot := &lightingFixture{stopErr: errors.New("resource owner cannot confirm stop")}
	resumed, err := NewController(home, Owners{Resource: boot, Scenario: boot})
	if err != nil {
		t.Fatal(err)
	}
	if err := resumed.Reconcile(t.Context()); !errors.Is(err, boot.stopErr) {
		t.Fatalf("failed stop = %v", err)
	}
	state, err := resumed.Current()
	if err != nil || state.Phase != Disabling {
		t.Fatalf("failed stop recorded as completed: %+v, %v", state, err)
	}
	boot.stopErr = nil
	if err := resumed.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	state, err = resumed.Current()
	if err != nil || state.Phase != Disabled || state.Selection != nil {
		t.Fatalf("disabled intent = %+v, %v", state, err)
	}
	if err := resumed.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	expectLightingCalls(t, boot, "quiesce", "stop", "quiesce", "stop")
}

func TestHeadlessLightingSerializesDisableWithStartup(t *testing.T) {
	home, controller, fixture := newLightingFixture(t)
	if err := controller.Enable(t.Context(), selectedLighting()); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	finish := make(chan struct{})
	fixture.ensure = func(ctx context.Context) error {
		close(entered)
		select {
		case <-finish:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	started := make(chan error, 1)
	go func() { started <- controller.Reconcile(ctx) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("startup did not enter resource owner")
	}
	other, err := NewController(home, Owners{Resource: fixture, Scenario: fixture})
	if err != nil {
		t.Fatal(err)
	}
	queued, cancelQueued := context.WithCancel(ctx)
	cancelQueued()
	if err := other.Disable(queued); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled disable = %v", err)
	}
	disabled := make(chan error, 1)
	go func() { disabled <- other.Disable(ctx) }()
	close(finish)
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	if err := <-disabled; err != nil {
		t.Fatal(err)
	}
	if err := other.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	expectLightingCalls(t, fixture, "verify", "backend", "scenario:boot-a:process-a", "quiesce", "stop")
}

func TestHeadlessLightingCorruptIntentPreventsOwnerEffects(t *testing.T) {
	_, controller, fixture := newLightingFixture(t)
	if err := os.MkdirAll(filepath.Dir(controller.path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(controller.path, []byte(`{"schema_version":1,"phase":"enabled"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := controller.Reconcile(t.Context()); err == nil {
		t.Fatal("malformed intent was accepted")
	}
	expectLightingCalls(t, fixture)
}
