package agentaccess

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

const permissionParityCallerID = "run:00000000-0000-4000-8000-000000000001"

// This matrix deliberately builds each backend through the same Broker and
// Session.Manager owners. The fake driver records browser-side dispatches, so
// every denial can assert that the action did not cross that boundary.
func TestManagedAndInstalledPermissionParityMatrix(t *testing.T) {
	for _, backend := range []string{"managed", string(sessionpermission.BackendInstalled)} {
		for _, tc := range []struct {
			name           string
			mutate         func(*permissionParityHarness)
			instruction    contracts.CompiledInstruction
			targetURL      string
			wantRejected   bool
			wantDispatches int32
			wantActions    int32
			wantStepType   string
		}{
			{name: "valid in-scope click", instruction: parityClick("button#continue"), wantDispatches: 1, wantActions: 1, wantStepType: "click"},
			{name: "valid in-scope navigation", instruction: parityNavigate("https://safe.test/next"), wantDispatches: 1, wantActions: 1, wantStepType: "navigate"},
			{name: "valid in-scope navigation with matching authorization target", instruction: parityNavigate("https://safe.test/next"), targetURL: "https://safe.test/next", wantDispatches: 1, wantActions: 1, wantStepType: "navigate"},
			{name: "denied-origin navigation", instruction: parityNavigate("https://blocked.test/private"), wantRejected: true},
			{name: "typed navigation and authorization target disagree", instruction: parityNavigate("https://safe.test/next"), targetURL: "https://blocked.test/private", wantRejected: true},
			{name: "redirected current origin", mutate: func(h *permissionParityHarness) { h.currentURL.Store("https://blocked.test/redirect") }, instruction: parityClick("button#continue"), wantRejected: true},
			{name: "caller changed after admission", mutate: func(h *permissionParityHarness) { h.caller.value.Store("run:attacker") }, instruction: parityClick("button#continue"), wantRejected: true},
			{name: "profile revoked after admission", mutate: func(h *permissionParityHarness) { h.resolver.profile.Control = false }, instruction: parityClick("button#continue"), wantRejected: true},
			{name: "stale opaque ref rejected by driver owner", instruction: parityClick("@bas-ref:e123456"), wantRejected: true, wantDispatches: 1},
		} {
			t.Run(backend+"/"+tc.name, func(t *testing.T) {
				h := newPermissionParityHarness(t, backend)
				defer h.close()
				if tc.mutate != nil {
					tc.mutate(h)
				}
				result, err := h.broker.Act(h.ctx, h.info.SessionID, ActRequest{Instruction: tc.instruction, TargetURL: tc.targetURL})
				if tc.wantRejected {
					if err == nil {
						t.Errorf("action was accepted, want rejection")
					}
				} else if err != nil {
					t.Errorf("in-scope action failed: %v", err)
				} else if result.Outcome.StepType != tc.wantStepType {
					t.Errorf("driver action type=%q, want %q", result.Outcome.StepType, tc.wantStepType)
				}
				if got := h.dispatches.Load(); got != tc.wantDispatches || h.actions.Load() != tc.wantActions {
					t.Errorf("action dispatches/effects=%d/%d, want %d/%d", got, h.actions.Load(), tc.wantDispatches, tc.wantActions)
				}
			})
		}

		t.Run(backend+"/wrong-session", func(t *testing.T) {
			h := newPermissionParityHarness(t, backend)
			defer h.close()
			_, err := h.broker.Act(h.ctx, "some-other-session", ActRequest{Instruction: parityClick("button#continue")})
			if err == nil || h.dispatches.Load() != 0 {
				t.Fatalf("wrong session result err=%v driver-dispatches=%d", err, h.dispatches.Load())
			}
		})
		t.Run(backend+"/backend-profile-mismatch", func(t *testing.T) {
			h := newPermissionParityHarness(t, backend)
			defer h.server.Close()
			if backend == string(sessionpermission.BackendInstalled) {
				h.resolver.profile.ProfileMode = "fresh"
			} else {
				h.resolver.profile.ProfileMode = "isolated"
			}
			startsBefore := h.starts.Load()
			_, err := h.broker.Open(h.ctx, OpenRequest{Backend: backend, ReuseMode: "fresh", StartURL: "https://safe.test/start"})
			if err == nil || h.starts.Load() != startsBefore {
				t.Fatalf("backend/profile mismatch err=%v starts-before=%d starts-after=%d", err, startsBefore, h.starts.Load())
			}
		})
	}
}

// This is one composed browser task, rather than isolated operation checks:
// each backend opens once, navigates and acts in scope, then proves URL denial,
// grant revocation, and owner cleanup all use the same session lease.
func TestManagedAndInstalledTaskFamilyRevalidatesScopeRevocationAndCleanup(t *testing.T) {
	for _, backend := range []string{"managed", string(sessionpermission.BackendInstalled)} {
		t.Run(backend, func(t *testing.T) {
			h := newPermissionParityHarness(t, backend)
			defer h.close()

			if h.starts.Load() != 1 {
				t.Fatalf("session opens=%d, want exactly one", h.starts.Load())
			}
			if _, err := h.broker.Act(h.ctx, h.info.SessionID, ActRequest{Instruction: parityNavigate("https://safe.test/orders")}); err != nil {
				t.Fatalf("in-scope navigation: %v", err)
			}
			if _, err := h.broker.Act(h.ctx, h.info.SessionID, ActRequest{Instruction: parityClick("button#continue")}); err != nil {
				t.Fatalf("in-scope task action after navigation: %v", err)
			}
			if h.dispatches.Load() != 2 || h.actions.Load() != 2 {
				t.Fatalf("legitimate task driver dispatch/effects=%d/%d, want 2/2", h.dispatches.Load(), h.actions.Load())
			}

			if _, err := h.broker.Act(h.ctx, h.info.SessionID, ActRequest{Instruction: parityNavigate("https://blocked.test/private")}); err == nil {
				t.Fatal("out-of-scope task transition was accepted")
			}
			if h.dispatches.Load() != 2 {
				t.Fatalf("out-of-scope transition reached driver: dispatches=%d", h.dispatches.Load())
			}

			h.resolver.profile.Control = false
			if _, err := h.broker.Act(h.ctx, h.info.SessionID, ActRequest{Instruction: parityClick("button#submit")}); err == nil {
				t.Fatal("revoked profile accepted the next task action")
			}
			if h.dispatches.Load() != 2 || h.actions.Load() != 2 {
				t.Fatalf("post-revocation action reached driver: dispatch/effects=%d/%d", h.dispatches.Load(), h.actions.Load())
			}

			if err := h.broker.Close(h.ctx, h.info.SessionID); err != nil {
				t.Fatalf("current owner could not clean up revoked task: %v", err)
			}
			if h.closes.Load() != 1 {
				t.Fatalf("owner cleanup reached driver close %d times, want 1", h.closes.Load())
			}
		})
	}
}

func TestManagedAndInstalledRevocationDuringInFlightActionStopsNextDispatch(t *testing.T) {
	for _, backend := range []string{"managed", string(sessionpermission.BackendInstalled)} {
		t.Run(backend, func(t *testing.T) {
			h := newPermissionParityHarness(t, backend)
			defer h.close()
			h.runStarted = make(chan struct{}, 1)
			h.releaseRun = make(chan struct{})
			actionDone := make(chan error, 1)
			go func() {
				_, err := h.broker.Act(h.ctx, h.info.SessionID, ActRequest{Instruction: parityClick("button#continue")})
				actionDone <- err
			}()
			<-h.runStarted
			// Once dispatched to the existing driver, the browser action has
			// crossed the documented no-rollback boundary. Revocation must block
			// the next action, but cannot claim to cancel this accepted action.
			h.resolver.profile.Control = false
			close(h.releaseRun)
			if err := <-actionDone; err != nil {
				t.Fatalf("already-dispatched action unexpectedly failed: %v", err)
			}
			if h.actions.Load() != 1 {
				t.Fatalf("in-flight driver action count=%d, want 1", h.actions.Load())
			}
			if _, err := h.broker.Act(h.ctx, h.info.SessionID, ActRequest{Instruction: parityClick("button#continue")}); err == nil {
				t.Fatal("revoked profile allowed a later action")
			}
			if h.dispatches.Load() != 1 {
				t.Fatalf("post-revocation action reached driver; dispatches=%d", h.dispatches.Load())
			}
		})
	}
}

func TestManagedAndInstalledProfileRevocationBlocksRecordingStopBeforeDriverEffect(t *testing.T) {
	for _, backend := range []string{"managed", string(sessionpermission.BackendInstalled)} {
		t.Run(backend, func(t *testing.T) {
			h := newPermissionParityHarness(t, backend)
			defer h.close()
			entry, err := h.broker.get(h.info.SessionID)
			if err != nil {
				t.Fatal(err)
			}
			// The exact session's current grant is narrowed after open. Its old
			// profile cannot authorize a recording receipt or stop mutation.
			h.resolver.profile.HistoryRead = false
			if _, err := entry.session.StopRecording(h.ctx); err == nil {
				t.Fatal("revoked history permission allowed recording stop")
			}
			if got := h.recordStops.Load(); got != 0 {
				t.Fatalf("revoked recording stop reached driver %d times", got)
			}
			if err := entry.session.Close(h.ctx); err != nil {
				t.Fatalf("owner close must remain available to terminate revoked session: %v", err)
			}
			if got := h.closes.Load(); got != 1 {
				t.Fatalf("revoked session cleanup reached driver close %d times, want 1", got)
			}
		})
	}
}

func TestManagedAndInstalledExtensionLifecyclePermissionParity(t *testing.T) {
	for _, tc := range []struct {
		name           string
		backend        string
		installAllowed bool
		loadAllowed    bool
		wantOpenDenied bool
		wantInstall    bool
		wantReload     bool
	}{
		{name: "managed explicit install and load", backend: "managed", installAllowed: true, loadAllowed: true, wantInstall: true, wantReload: true},
		{name: "managed load without install", backend: "managed", loadAllowed: true},
		{name: "managed install without load", backend: "managed", installAllowed: true, wantInstall: true},
		{name: "installed rejects managed lifecycle opt-ins before session start", backend: string(sessionpermission.BackendInstalled), installAllowed: true, loadAllowed: true, wantOpenDenied: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			controller := &parityExtensionController{}
			verifier := &parityExtensionArtifactVerifier{artifact: parityExtensionArtifact()}
			h := newPermissionParityHarnessWithExtensionContract(t, tc.backend, tc.installAllowed, tc.loadAllowed, controller, verifier, tc.wantOpenDenied)
			defer h.close()
			if tc.wantOpenDenied {
				if h.openErr == nil || h.starts.Load() != 0 || verifier.calls.Load() != 0 || controller.installs.Load() != 0 || controller.reloads.Load() != 0 {
					t.Fatalf("unsupported installed lifecycle open err=%v starts=%d verifier=%d controller=%d/%d", h.openErr, h.starts.Load(), verifier.calls.Load(), controller.installs.Load(), controller.reloads.Load())
				}
				return
			}
			if h.openErr != nil {
				t.Fatalf("managed profile failed to open: %v", h.openErr)
			}
			ctx := withBridgeContextCaller(h.ctx, permissionParityCallerID)
			_, installErr := h.broker.InstallExtension(ctx, h.info.SessionID, "", verifier.artifact.ArtifactRef, verifier.artifact.Digest)
			if tc.wantInstall {
				if installErr != nil || controller.installs.Load() != 1 || verifier.calls.Load() != 1 {
					t.Fatalf("explicit install err=%v verifier=%d controller installs=%d", installErr, verifier.calls.Load(), controller.installs.Load())
				}
			} else if installErr == nil || controller.installs.Load() != 0 || verifier.calls.Load() != 0 {
				t.Fatalf("install without opt-in err=%v verifier=%d controller installs=%d", installErr, verifier.calls.Load(), controller.installs.Load())
			}
			for _, operation := range []struct {
				name string
				call func() error
			}{
				{name: "load", call: func() error {
					_, err := h.broker.LoadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
					return err
				}},
				{name: "reload", call: func() error {
					_, err := h.broker.ReloadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
					return err
				}},
				{name: "unload", call: func() error {
					_, err := h.broker.UnloadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
					return err
				}},
				{name: "validate", call: func() error {
					_, err := h.broker.ValidateExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
					return err
				}},
			} {
				err := operation.call()
				if tc.wantReload && err != nil {
					t.Errorf("explicit %s failed: %v", operation.name, err)
				}
				if !tc.wantReload && err == nil {
					t.Errorf("%s without a separate load grant and broker-verified install was accepted", operation.name)
				}
			}
			if tc.wantReload {
				if controller.loads.Load() != 1 || controller.reloads.Load() != 1 || controller.unloads.Load() != 1 || controller.validations.Load() != 1 {
					t.Fatalf("explicit load grant effects: load=%d reload=%d unload=%d validate=%d", controller.loads.Load(), controller.reloads.Load(), controller.unloads.Load(), controller.validations.Load())
				}
			} else if controller.loads.Load() != 0 || controller.reloads.Load() != 0 || controller.unloads.Load() != 0 || controller.validations.Load() != 0 {
				t.Fatalf("denied lifecycle reached controller: load=%d reload=%d unload=%d validate=%d", controller.loads.Load(), controller.reloads.Load(), controller.unloads.Load(), controller.validations.Load())
			}
		})
	}
}

func TestManagedAndInstalledExtensionInventoryAndActionMatrix(t *testing.T) {
	t.Run("managed inventory requires explicit access", func(t *testing.T) {
		controller := &parityExtensionController{}
		verifier := &parityExtensionArtifactVerifier{artifact: parityExtensionArtifact()}
		denied := newPermissionParityHarnessWithExtensionAccess(t, "managed", false, false, false, controller, verifier, false)
		defer denied.close()
		if _, err := denied.broker.SelectedTabExtensions(withBridgeContextCaller(denied.ctx, permissionParityCallerID), denied.info.SessionID); err == nil {
			t.Fatal("inventory without extension access was allowed")
		}
		if controller.inventory.Load() != 0 || controller.installs.Load() != 0 || verifier.calls.Load() != 0 {
			t.Fatalf("denied inventory reached owners: inventory=%d installs=%d verifier=%d", controller.inventory.Load(), controller.installs.Load(), verifier.calls.Load())
		}
	})

	t.Run("managed explicit inventory permission reaches exact owner", func(t *testing.T) {
		controller := &parityExtensionController{}
		h := newPermissionParityHarnessWithExtensionAccess(t, "managed", false, false, true, controller, nil, false)
		defer h.close()
		inventory, err := h.broker.SelectedTabExtensions(withBridgeContextCaller(h.ctx, permissionParityCallerID), h.info.SessionID)
		if err != nil {
			t.Fatalf("explicit managed inventory: %v", err)
		}
		if inventory.SessionID != h.info.SessionID || inventory.Binding.Backend != "managed" || controller.inventory.Load() != 1 {
			t.Fatalf("inventory was not owner-bound: response=%#v owner calls=%d", inventory, controller.inventory.Load())
		}
	})

	t.Run("managed inventory rejects foreign caller before owner read", func(t *testing.T) {
		controller := &parityExtensionController{}
		h := newPermissionParityHarnessWithExtensionAccess(t, "managed", false, false, true, controller, nil, false)
		defer h.close()
		h.caller.value.Store("run:attacker")
		if _, err := h.broker.SelectedTabExtensions(h.ctx, h.info.SessionID); err == nil {
			t.Fatal("foreign caller read managed extension inventory")
		}
		if controller.inventory.Load() != 0 {
			t.Fatalf("foreign caller reached inventory owner %d times", controller.inventory.Load())
		}
	})

	t.Run("managed lifecycle rejects foreign caller before verifier and controller", func(t *testing.T) {
		controller := &parityExtensionController{}
		verifier := &parityExtensionArtifactVerifier{artifact: parityExtensionArtifact()}
		h := newPermissionParityHarnessWithExtensionAccess(t, "managed", true, true, true, controller, verifier, false)
		defer h.close()
		h.caller.value.Store("run:attacker")
		ctx := h.ctx
		if _, err := h.broker.InstallExtension(ctx, h.info.SessionID, "", verifier.artifact.ArtifactRef, verifier.artifact.Digest); err == nil {
			t.Fatal("foreign caller installed an extension")
		}
		for _, operation := range []struct {
			name string
			call func() error
		}{
			{name: "load", call: func() error {
				_, err := h.broker.LoadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
				return err
			}},
			{name: "reload", call: func() error {
				_, err := h.broker.ReloadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
				return err
			}},
			{name: "unload", call: func() error {
				_, err := h.broker.UnloadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
				return err
			}},
			{name: "validate", call: func() error {
				_, err := h.broker.ValidateExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
				return err
			}},
		} {
			if err := operation.call(); err == nil {
				t.Errorf("foreign caller %s was accepted", operation.name)
			}
		}
		if verifier.calls.Load() != 0 || controller.installs.Load() != 0 || controller.loads.Load() != 0 || controller.reloads.Load() != 0 || controller.unloads.Load() != 0 || controller.validations.Load() != 0 {
			t.Fatalf("foreign extension operations reached owners: verifier=%d install=%d load=%d reload=%d unload=%d validate=%d", verifier.calls.Load(), controller.installs.Load(), controller.loads.Load(), controller.reloads.Load(), controller.unloads.Load(), controller.validations.Load())
		}
	})

	for _, tc := range []struct {
		name           string
		installAllowed bool
		loadAllowed    bool
		inventory      bool
	}{
		{name: "inventory-only", inventory: true},
		{name: "install-only", installAllowed: true},
		{name: "load-only", loadAllowed: true},
		{name: "combined", installAllowed: true, loadAllowed: true, inventory: true},
	} {
		t.Run("installed rejects unsupported "+tc.name+" grant before start", func(t *testing.T) {
			controller := &parityExtensionController{}
			verifier := &parityExtensionArtifactVerifier{artifact: parityExtensionArtifact()}
			h := newPermissionParityHarnessWithExtensionAccess(t, string(sessionpermission.BackendInstalled), tc.installAllowed, tc.loadAllowed, tc.inventory, controller, verifier, true)
			defer h.close()
			if h.openErr == nil || h.starts.Load() != 0 || controller.inventory.Load() != 0 || controller.installs.Load() != 0 || verifier.calls.Load() != 0 {
				t.Fatalf("unsupported installed grant err=%v starts=%d inventory=%d installs=%d verifier=%d", h.openErr, h.starts.Load(), controller.inventory.Load(), controller.installs.Load(), verifier.calls.Load())
			}
		})
	}

	t.Run("installed keeps supported typed action and rejects every extension verb", func(t *testing.T) {
		controller := &parityExtensionController{}
		verifier := &parityExtensionArtifactVerifier{artifact: parityExtensionArtifact()}
		h := newPermissionParityHarnessWithExtensionAccess(t, string(sessionpermission.BackendInstalled), false, false, false, controller, verifier, false)
		defer h.close()
		h.broker.extensions, h.broker.extensionVerifier = controller, verifier
		ctx := withBridgeContextCaller(h.ctx, permissionParityCallerID)
		if _, err := h.broker.Act(ctx, h.info.SessionID, ActRequest{Instruction: parityClick("button#continue")}); err != nil || h.actions.Load() != 1 {
			t.Fatalf("supported installed-mode typed action err=%v effects=%d", err, h.actions.Load())
		}
		if _, err := h.broker.SelectedTabExtensions(ctx, h.info.SessionID); err == nil {
			t.Fatal("installed extension inventory was accepted")
		}
		if _, err := h.broker.InstallExtension(ctx, h.info.SessionID, "", verifier.artifact.ArtifactRef, verifier.artifact.Digest); err == nil {
			t.Fatal("installed extension install was accepted")
		}
		for _, operation := range []struct {
			name string
			call func() error
		}{
			{name: "load", call: func() error {
				_, err := h.broker.LoadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
				return err
			}},
			{name: "reload", call: func() error {
				_, err := h.broker.ReloadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
				return err
			}},
			{name: "unload", call: func() error {
				_, err := h.broker.UnloadExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
				return err
			}},
			{name: "validate", call: func() error {
				_, err := h.broker.ValidateExtension(ctx, h.info.SessionID, verifier.artifact.ExtensionID)
				return err
			}},
		} {
			if err := operation.call(); err == nil {
				t.Errorf("installed extension %s was accepted", operation.name)
			}
		}
		if verifier.calls.Load() != 0 || controller.installs.Load() != 0 || controller.loads.Load() != 0 || controller.reloads.Load() != 0 || controller.unloads.Load() != 0 || controller.validations.Load() != 0 || controller.inventory.Load() != 0 {
			t.Fatalf("unsupported installed extension calls reached owners: verifier=%d install=%d load=%d reload=%d unload=%d validate=%d inventory=%d", verifier.calls.Load(), controller.installs.Load(), controller.loads.Load(), controller.reloads.Load(), controller.unloads.Load(), controller.validations.Load(), controller.inventory.Load())
		}
	})
}

type permissionParityHarness struct {
	ctx         context.Context
	backend     string
	info        SessionInfo
	broker      *Broker
	server      *httptest.Server
	openErr     error
	resolver    *permissionParityResolver
	caller      *permissionParityCaller
	currentURL  atomic.Value
	starts      atomic.Int32
	recordStops atomic.Int32
	closes      atomic.Int32
	actions     atomic.Int32
	dispatches  atomic.Int32
	runStarted  chan struct{}
	releaseRun  chan struct{}
}

type permissionParityResolver struct {
	profile sessionpermission.Profile
	policy  sessionpermission.Authorizer
}

func (r *permissionParityResolver) Resolve(context.Context, string) (sessionpermission.Profile, sessionpermission.Authorizer, error) {
	return r.profile, r.policy, nil
}

type permissionParityCaller struct{ value atomic.Value }

func (c *permissionParityCaller) BrowserCallerID(context.Context) (string, error) {
	return c.value.Load().(string), nil
}

func newPermissionParityHarness(t *testing.T, backend string) *permissionParityHarness {
	return newPermissionParityHarnessWithExtensionContract(t, backend, false, false, nil, nil, false)
}

func newPermissionParityHarnessWithExtensionContract(t *testing.T, backend string, install, load bool, controller ExtensionController, verifier ExtensionArtifactVerifier, allowOpenFailure bool) *permissionParityHarness {
	return newPermissionParityHarnessWithExtensionAccess(t, backend, install, load, install || load, controller, verifier, allowOpenFailure)
}

func newPermissionParityHarnessWithExtensionAccess(t *testing.T, backend string, install, load, extensionAccess bool, controller ExtensionController, verifier ExtensionArtifactVerifier, allowOpenFailure bool) *permissionParityHarness {
	t.Helper()
	mode := "fresh"
	backendValue := sessionpermission.BackendManaged
	if backend == string(sessionpermission.BackendInstalled) {
		mode, backendValue = "isolated", sessionpermission.BackendInstalled
	}
	h := &permissionParityHarness{backend: backend, ctx: context.Background()}
	h.currentURL.Store("https://safe.test/start")
	profile := sessionpermission.Profile{ID: "profile-parity", ProfileID: "profile-parity", ProfileMode: mode, Effect: sessionpermission.EffectAct, View: true, Control: true, HistoryRead: true, Approval: sessionpermission.ApprovalNone, URLAllow: []string{"https://safe.test/*"}, ExtensionAccess: extensionAccess, ExtensionInstall: install, ExtensionLoad: load}
	if extensionAccess {
		profile.ExtensionPermissions = map[string][]string{parityExtensionArtifact().ExtensionID: {"storage"}}
	}
	policy, err := sessionpermission.New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	h.resolver = &permissionParityResolver{profile: profile, policy: policy}
	h.caller = &permissionParityCaller{}
	h.caller.value.Store(permissionParityCallerID)
	h.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/session/start":
			var request driver.CreateSessionRequest
			_ = json.NewDecoder(r.Body).Decode(&request)
			if request.Options.BrowserBackend != backend {
				t.Errorf("driver backend = %q, want %q", request.Options.BrowserBackend, backend)
			}
			h.starts.Add(1)
			_ = json.NewEncoder(w).Encode(driver.CreateSessionResponse{SessionID: backend + "-parity-session", LeaseID: "lease-parity", ActivePageID: "page-parity"})
		case strings.HasSuffix(r.URL.Path, "/record/navigate"):
			var request struct {
				URL string `json:"url"`
			}
			_ = json.NewDecoder(r.Body).Decode(&request)
			h.currentURL.Store(request.URL)
			_ = json.NewEncoder(w).Encode(driver.NavigateResponse{DriverPageID: "page-parity", URL: request.URL})
		case strings.HasSuffix(r.URL.Path, "/record/navigation-state"):
			_ = json.NewEncoder(w).Encode(driver.NavigationStateResponse{SessionID: backend + "-parity-session", URL: h.currentURL.Load().(string)})
		case strings.HasSuffix(r.URL.Path, "/record/stop"):
			h.recordStops.Add(1)
			_ = json.NewEncoder(w).Encode(driver.StopRecordingResponse{SessionID: backend + "-parity-session", RecordingID: "recording-parity", ActionCount: 0, StoppedAt: time.Now().UTC().Format(time.RFC3339Nano)})
		case strings.HasSuffix(r.URL.Path, "/run"):
			h.dispatches.Add(1)
			if h.runStarted != nil {
				h.runStarted <- struct{}{}
				<-h.releaseRun
			}
			var request struct {
				Instruction struct {
					Action struct {
						Click struct {
							Selector string `json:"selector"`
						} `json:"click"`
						Navigate struct {
							URL string `json:"url"`
						} `json:"navigate"`
					} `json:"action"`
				} `json:"instruction"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode typed driver instruction: %v", err)
				http.Error(w, "invalid run request", http.StatusBadRequest)
				return
			}
			stepType := ""
			if target := request.Instruction.Action.Navigate.URL; target != "" {
				stepType = "navigate"
				h.currentURL.Store(target)
			}
			if selector := request.Instruction.Action.Click.Selector; strings.HasPrefix(selector, "@bas-ref:") {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":{"code":"ACCESSIBILITY_REFERENCE_STALE","message":"unknown, expired, or belongs to another page"}}`))
				return
			}
			if request.Instruction.Action.Click.Selector != "" {
				stepType = "click"
			}
			h.actions.Add(1)
			_ = json.NewEncoder(w).Encode(contracts.StepOutcome{Success: true, StepType: stepType})
		case strings.HasSuffix(r.URL.Path, "/close"):
			h.closes.Add(1)
			_ = json.NewEncoder(w).Encode(driver.CloseSessionResponse{Success: true})
		default:
			http.NotFound(w, r)
		}
	}))
	client, err := driver.NewClientWithURL(h.server.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	manager := session.NewManagerWithClient(client)
	open := &bridgeOpenRecorder{}
	broker, err := NewBroker(manager, h.resolver, open, h.caller)
	if err != nil {
		t.Fatal(err)
	}
	h.broker = broker
	h.broker.extensions, h.broker.extensionVerifier = controller, verifier
	trusted := TrustedBrowserOpenContext{CallerID: permissionParityCallerID, GrantReceipt: "browser:parity", IdentityToken: "signed-token", PermissionRef: profile.ID}
	broker.SetTrustedOpenContextResolver(fixedOpenContext(trusted))
	broker.SetEvidenceRecorder(identityEvidenceRecorder{})
	capabilities := []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl, sessionpermission.CapabilityHistoryRead}
	if extensionAccess {
		capabilities = append(capabilities, sessionpermission.CapabilityExtensionInventory, sessionpermission.CapabilityExtensionHostPermissions)
	}
	if install {
		capabilities = append(capabilities, sessionpermission.CapabilityExtensionInstall)
	}
	if load {
		capabilities = append(capabilities, sessionpermission.CapabilityExtensionLoad)
	}
	broker.SetCapabilityReporter(staticCapabilityReporter{{Backend: backendValue, Qualified: true, EvidenceID: "fake-owner-parity", Capabilities: capabilities}})
	info, err := broker.Open(h.ctx, OpenRequest{PermissionRef: profile.ID, Backend: backend, ReuseMode: "fresh", StartURL: "https://safe.test/start"})
	h.openErr = err
	if err != nil {
		if !allowOpenFailure {
			h.server.Close()
			t.Fatalf("open %s fake owner: %v", backend, err)
		}
		return h
	}
	h.info = info
	return h
}

type parityExtensionArtifactVerifier struct {
	calls    atomic.Int32
	artifact ExtensionArtifact
}

func (v *parityExtensionArtifactVerifier) VerifyBrowserExtensionArtifact(_ context.Context, ref, digest string) (ExtensionArtifact, error) {
	v.calls.Add(1)
	if ref != v.artifact.ArtifactRef || digest != v.artifact.Digest {
		return ExtensionArtifact{}, errors.New("unexpected parity artifact reference")
	}
	return v.artifact, nil
}

type parityExtensionController struct {
	installs    atomic.Int32
	loads       atomic.Int32
	reloads     atomic.Int32
	unloads     atomic.Int32
	validations atomic.Int32
	inventory   atomic.Int32
}

func (c *parityExtensionController) Install(_ context.Context, request ExtensionInstallRequest) (ExtensionReceipt, error) {
	c.installs.Add(1)
	return bindExtensionReceipt(ExtensionReceipt{}, request.Binding, request.Artifact, "install"), nil
}
func (c *parityExtensionController) Load(_ context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	c.loads.Add(1)
	return bindExtensionReceipt(ExtensionReceipt{}, request.Binding, request.Artifact, "load"), nil
}
func (c *parityExtensionController) Reload(_ context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	c.reloads.Add(1)
	return bindExtensionReceipt(ExtensionReceipt{}, request.Binding, request.Artifact, "reload"), nil
}
func (c *parityExtensionController) Unload(_ context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	c.unloads.Add(1)
	return bindExtensionReceipt(ExtensionReceipt{}, request.Binding, request.Artifact, "unload"), nil
}
func (c *parityExtensionController) Validate(_ context.Context, request ExtensionLifecycleRequest) (ExtensionValidation, error) {
	c.validations.Add(1)
	return ExtensionValidation{Receipt: bindExtensionReceipt(ExtensionReceipt{}, request.Binding, request.Artifact, "validate"), Valid: true}, nil
}
func (c *parityExtensionController) Inventory(_ context.Context, binding ExtensionSessionBinding) (ExtensionInventoryReceipt, error) {
	c.inventory.Add(1)
	return ExtensionInventoryReceipt{Binding: binding, ObservedAt: time.Now().UTC()}, nil
}

func parityExtensionArtifact() ExtensionArtifact {
	return ExtensionArtifact{ArtifactRef: "parity-build", ExtensionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Digest: strings.Repeat("a", 64), ManifestSHA256: strings.Repeat("b", 64), Version: "1.0.0", Permissions: []string{"storage"}}
}

func (h *permissionParityHarness) close() {
	if h.broker != nil && h.info.SessionID != "" {
		_ = h.broker.Close(h.ctx, h.info.SessionID)
	}
	if h.server != nil {
		h.server.Close()
	}
}

func TestBrokerCloseCompletesAfterCallerContextCancellation(t *testing.T) {
	h := newPermissionParityHarness(t, string(sessionpermission.BackendManaged))
	defer h.server.Close()

	ctx, cancel := context.WithCancel(h.ctx)
	cancel()
	if err := h.broker.Close(ctx, h.info.SessionID); err != nil {
		t.Fatalf("authorized close failed after the caller disconnected: %v", err)
	}
	if got := h.closes.Load(); got != 1 {
		t.Fatalf("driver close calls=%d, want one bounded cleanup", got)
	}
	h.broker.mu.RLock()
	entry := h.broker.entries[h.info.SessionID]
	h.broker.mu.RUnlock()
	if entry == nil || !entry.closed {
		t.Fatalf("Broker entry was not finalized after caller cancellation: %#v", entry)
	}
}

type closedSessionFinalizerFixture struct {
	ExtensionController
	manager *session.Manager
	calls   []string
}

func (f *closedSessionFinalizerFixture) FinalizeClosedSession(_ context.Context, sessionID string) error {
	if _, stillOwned := f.manager.Get(sessionID); stillOwned {
		return errors.New("finalizer ran before the canonical session owner closed")
	}
	f.calls = append(f.calls, sessionID)
	return nil
}

func TestBrokerCloseFinalizesExtensionStagingAfterSessionOwnerClose(t *testing.T) {
	h := newPermissionParityHarness(t, string(sessionpermission.BackendManaged))
	defer h.server.Close()
	finalizer := &closedSessionFinalizerFixture{manager: h.broker.manager}
	h.broker.extensions = finalizer
	if err := h.broker.Close(h.ctx, h.info.SessionID); err != nil {
		t.Fatalf("close managed session with extension cleanup: %v", err)
	}
	if len(finalizer.calls) != 1 || finalizer.calls[0] != h.info.SessionID {
		t.Fatalf("closed-session finalizer calls=%v, want exact session %q once", finalizer.calls, h.info.SessionID)
	}
}

func parityClick(selector string) contracts.CompiledInstruction {
	return contracts.CompiledInstruction{Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: selector}}}}
}

func parityNavigate(target string) contracts.CompiledInstruction {
	return contracts.CompiledInstruction{Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE, Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: target}}}}
}
