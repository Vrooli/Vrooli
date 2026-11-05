package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/testutil"
)

// fakeCapacityProvider implements CapacityProvider for handler tests.
type fakeCapacityProvider struct {
	overview     services.CapacityOverview
	overviewErr  error
	claims       []capacityapp.ClaimView
	findings     []engine.Finding
	reconcileErr error
	policy       []capacityapp.PolicyEntry
	setErr       error
	gotSetKey    string
	gotSetValue  string
}

func (f *fakeCapacityProvider) Overview(context.Context) (services.CapacityOverview, error) {
	return f.overview, f.overviewErr
}

func (f *fakeCapacityProvider) ListClaims(context.Context, string, bool) ([]capacityapp.ClaimView, error) {
	return f.claims, nil
}

func (f *fakeCapacityProvider) Reconcile(context.Context) ([]engine.Finding, error) {
	return f.findings, f.reconcileErr
}

func (f *fakeCapacityProvider) Policy(context.Context) ([]capacityapp.PolicyEntry, error) {
	return f.policy, nil
}

func (f *fakeCapacityProvider) SetPolicy(_ context.Context, key, value string) ([]capacityapp.PolicyEntry, error) {
	f.gotSetKey, f.gotSetValue = key, value
	if f.setErr != nil {
		return nil, f.setErr
	}
	return f.policy, nil
}

func TestCapacity_Overview_OK(t *testing.T) {
	h := NewCapacityHandler(&fakeCapacityProvider{
		overview: services.CapacityOverview{
			GPUs:             []services.GpuContention{{Index: 0, Name: "RTX", TotalBytes: 16, FreeBytes: 3, ClaimedBytes: 8}},
			Claims:           []capacityapp.ClaimView{{ClaimID: "a", OwnerID: "whisper"}},
			SensingAvailable: true,
		},
	}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/capacity/overview", nil)
	w := httptest.NewRecorder()
	h.Overview(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	body := testutil.DecodeJSONBody[map[string]any](t, w.Body.Bytes())
	if body["success"] != true {
		t.Errorf("expected success=true, got %v", body["success"])
	}
	if body["sensing_available"] != true {
		t.Errorf("expected sensing_available=true, got %v", body["sensing_available"])
	}
}

func TestCapacity_ListClaims_RejectsBadActiveOnly(t *testing.T) {
	h := NewCapacityHandler(&fakeCapacityProvider{}, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/capacity/claims?active_only=maybe", nil)
	w := httptest.NewRecorder()
	h.ListClaims(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusBadRequest)
}

func TestCapacity_Reconcile_SensingFailureIsUnavailable(t *testing.T) {
	h := NewCapacityHandler(&fakeCapacityProvider{reconcileErr: errors.New("nvidia-smi gone")}, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/capacity/reconcile", nil)
	w := httptest.NewRecorder()
	h.Reconcile(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusServiceUnavailable)
}

func TestCapacity_SetPolicy_ForwardsAndValidates(t *testing.T) {
	fake := &fakeCapacityProvider{policy: []capacityapp.PolicyEntry{{Key: "enforce", Value: "on"}}}
	h := NewCapacityHandler(fake, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capacity/policy", strings.NewReader(`{"key":"enforce","value":"on"}`))
	w := httptest.NewRecorder()
	h.SetPolicy(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	if fake.gotSetKey != "enforce" || fake.gotSetValue != "on" {
		t.Errorf("key/value not forwarded: %q=%q", fake.gotSetKey, fake.gotSetValue)
	}
}

func TestCapacity_SetPolicy_RequiresKey(t *testing.T) {
	h := NewCapacityHandler(&fakeCapacityProvider{}, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capacity/policy", strings.NewReader(`{"value":"on"}`))
	w := httptest.NewRecorder()
	h.SetPolicy(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusBadRequest)
}

func TestCapacity_SetPolicy_InvalidValueIs400(t *testing.T) {
	h := NewCapacityHandler(&fakeCapacityProvider{setErr: engine.ErrInvalidClaim}, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capacity/policy", strings.NewReader(`{"key":"enforce","value":"bogus"}`))
	w := httptest.NewRecorder()
	h.SetPolicy(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusBadRequest)
}
