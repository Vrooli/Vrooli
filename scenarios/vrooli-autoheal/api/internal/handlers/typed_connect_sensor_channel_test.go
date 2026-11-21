package handlers

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reconcile"

	checksproto "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
)

type stubExpectedProvider struct {
	expected []string
	err      error
}

func (p stubExpectedProvider) Expected(context.Context) ([]string, error) {
	return p.expected, p.err
}

type stubInstalledProvider struct {
	installed []string
	err       error
}

func (p stubInstalledProvider) Installed(context.Context) ([]string, error) {
	return p.installed, p.err
}

func reconcileTestHandlers(t *testing.T, checkIDs []string) *Handlers {
	t.Helper()
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	for _, id := range checkIDs {
		registry.Register(&mockCheck{id: id, status: checks.StatusOK, message: "ok"})
	}
	handlers := NewWithInterface(registry, &mockStore{}, caps)
	handlers.hostCollector = fakeHostCollector{}
	return handlers
}

// A check whose scenario is installed but outside the core-set closure is a
// supervision-scope signal, not a ghost. Ghost readings are dropped from every
// aggregate, so reporting one here removes live plant from uptime accounting.
func TestGetReconcileDoesNotGhostInstalledScenarios(t *testing.T) {
	handlers := reconcileTestHandlers(t, []string{"scenario-in-core", "scenario-out-of-core", "scenario-removed"})
	handlers.SetReconcileProvider(stubExpectedProvider{expected: []string{"in-core"}})
	handlers.SetInstalledProvider(stubInstalledProvider{installed: []string{"in-core", "out-of-core"}})

	service := &typedChecks{h: handlers}
	response, err := service.GetReconcile(context.Background(), connect.NewRequest(&checksproto.GetReconcileRequest{}))
	if err != nil {
		t.Fatalf("GetReconcile: %v", err)
	}
	got := response.Msg.GetReconcile()
	if !got.GetAvailable() || !got.GetGhostDetectionAvailable() {
		t.Fatalf("expected an available reconcile with ghost detection, got %+v", got)
	}
	if len(got.GetGhostCheckIds()) != 1 || got.GetGhostCheckIds()[0] != "scenario-removed" {
		t.Fatalf("ghost check ids = %#v, want only scenario-removed", got.GetGhostCheckIds())
	}
	if len(got.GetOutOfScopeCheckIds()) != 1 || got.GetOutOfScopeCheckIds()[0] != "scenario-out-of-core" {
		t.Fatalf("out-of-scope check ids = %#v", got.GetOutOfScopeCheckIds())
	}
}

// Absence from an unreadable installed set is not evidence that a target is
// gone, so an unavailable provider must classify nothing rather than ghosting
// every registered check.
func TestGetReconcileWithoutInstalledSetReportsUnavailableGhostDetection(t *testing.T) {
	handlers := reconcileTestHandlers(t, []string{"scenario-alpha"})
	handlers.SetReconcileProvider(stubExpectedProvider{expected: []string{"alpha"}})
	handlers.SetInstalledProvider(stubInstalledProvider{err: errors.New("scenarios directory is unreadable")})

	service := &typedChecks{h: handlers}
	response, err := service.GetReconcile(context.Background(), connect.NewRequest(&checksproto.GetReconcileRequest{}))
	if err != nil {
		t.Fatalf("GetReconcile: %v", err)
	}
	got := response.Msg.GetReconcile()
	if got.GetGhostDetectionAvailable() {
		t.Fatal("ghost detection must be unavailable when the installed set cannot be read")
	}
	if len(got.GetGhostCheckIds()) != 0 || len(got.GetOutOfScopeCheckIds()) != 0 {
		t.Fatalf("expected no classification, got ghosts=%#v out-of-scope=%#v", got.GetGhostCheckIds(), got.GetOutOfScopeCheckIds())
	}
	if got.GetGhostUnavailableReason() == "" {
		t.Fatal("expected a stated reason for unavailable ghost detection")
	}
}

// ListSaturation answers for every registered check from one window read, so a
// caller with a bounded deadline no longer has to issue one request per check
// and cannot mistake a deadline for a saturated sensor.
func TestListSaturationCoversEveryRegisteredCheckInOneRead(t *testing.T) {
	handlers := reconcileTestHandlers(t, []string{"system-gpu", "system-mce-recent", "host-kernel-error-signals"})
	handlers.store = &mockStore{transitions: &persistence.TransitionsResponse{
		Transitions: []persistence.Transition{
			{CheckID: "system-gpu", Timestamp: "2026-08-20T10:00:00Z", FromStatus: "ok", ToStatus: "warning"},
			{CheckID: "system-gpu", Timestamp: "2026-08-20T11:00:00Z", FromStatus: "warning", ToStatus: "ok"},
		},
		WindowHours: 24,
		Total:       2,
	}}

	service := &typedChecks{h: handlers}
	response, err := service.ListSaturation(context.Background(), connect.NewRequest(&checksproto.ListSaturationRequest{WindowHours: 24}))
	if err != nil {
		t.Fatalf("ListSaturation: %v", err)
	}
	saturations := response.Msg.GetSaturations()
	if len(saturations) != 3 {
		t.Fatalf("expected one row per registered check, got %d", len(saturations))
	}
	byID := map[string]*checksproto.Saturation{}
	for _, item := range saturations {
		byID[item.GetCheckId()] = item
	}
	if !byID["system-gpu"].GetTransitioned() || byID["system-gpu"].GetTransitionCount() != 2 {
		t.Fatalf("system-gpu = %+v, want two transitions", byID["system-gpu"])
	}
	// A check with no transitions is reported explicitly rather than omitted,
	// so the caller can tell "did not transition" from "was not read".
	if byID["system-mce-recent"] == nil || byID["system-mce-recent"].GetTransitioned() {
		t.Fatalf("system-mce-recent = %+v, want an explicit non-transitioned row", byID["system-mce-recent"])
	}
	// A check steady at OK has not transitioned, but it is healthy, not
	// saturated. Marking it saturated would exclude it from every aggregate.
	for _, item := range saturations {
		if item.GetSaturated() {
			t.Fatalf("%s reported saturated while steady at OK", item.GetCheckId())
		}
	}
	if response.Msg.GetTruncated() {
		t.Fatal("a two-transition window must not report truncation")
	}
}

// The filesystem provider is the existence oracle. If it ever returned an
// empty set successfully, every scenario check would be ghosted at once.
func TestFilesystemInstalledProviderRejectsAnEmptySet(t *testing.T) {
	provider := &reconcile.FilesystemInstalledProvider{Root: t.TempDir()}
	if _, err := provider.Installed(context.Background()); err == nil {
		t.Fatal("expected an error rather than an empty installed set")
	}
}

// Saturation means pinned in a non-normal state, not merely quiet. Deriving it
// from the transition count alone marked every healthy check saturated, and
// saturated readings are excluded from aggregates.
func TestListSaturationOnlyFlagsChecksPinnedInANonNormalState(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	registry.Register(&mockCheck{id: "steady-ok", status: checks.StatusOK, message: "fine"})
	registry.Register(&mockCheck{id: "stuck-warning", status: checks.StatusWarning, message: "degraded"})
	registry.Register(&mockCheck{id: "stuck-critical", status: checks.StatusCritical, message: "down"})
	handlers := NewWithInterface(registry, &mockStore{}, caps)
	handlers.hostCollector = fakeHostCollector{}
	// Populate current status by running the registry once.
	registry.RunAll(context.Background(), false)

	service := &typedChecks{h: handlers}
	response, err := service.ListSaturation(context.Background(), connect.NewRequest(&checksproto.ListSaturationRequest{WindowHours: 24}))
	if err != nil {
		t.Fatalf("ListSaturation: %v", err)
	}
	saturated := map[string]bool{}
	for _, item := range response.Msg.GetSaturations() {
		saturated[item.GetCheckId()] = item.GetSaturated()
	}
	if saturated["steady-ok"] {
		t.Fatal("a check steady at OK must not be reported saturated")
	}
	if !saturated["stuck-warning"] || !saturated["stuck-critical"] {
		t.Fatalf("checks pinned in a non-normal state must be saturated: %#v", saturated)
	}
}
