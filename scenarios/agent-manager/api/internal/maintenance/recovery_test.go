package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/health"
)

func TestRecoveryServesInitializingWhileHistoryIsBlocked(t *testing.T) {
	recovery := NewRecovery()
	entered, release := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(release); recovery.Stop() })
	returned := make(chan struct{})
	go func() {
		recovery.Start(t.Context(), []RecoveryStep{{Name: "workflow_accounting", Run: func(ctx context.Context) error {
			close(entered)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}}})
		close(returned)
	}()
	<-entered
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("historical recovery blocked startup")
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := recovery.Health(next)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"initializing"`) || !strings.Contains(response.Body.String(), `"readiness":false`) || !strings.Contains(response.Body.String(), "workflow_accounting") {
		t.Fatalf("initializing health=%d %s", response.Code, response.Body.String())
	}
}

func TestRecoveryReadyAndDegradedRemainDistinct(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
		status  string
		ready   bool
		code    int
	}{
		{"ready", nil, "ready", true, http.StatusNoContent},
		{"failed", errors.New("accounting unavailable"), "degraded", false, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recovery := NewRecovery()
			recovery.Start(t.Context(), []RecoveryStep{{Name: "accounting", Run: func(context.Context) error { return tc.failure }}})
			<-recovery.Done()
			defer recovery.Stop()
			state := recovery.Snapshot()
			if state.Status != tc.status || state.Readiness != tc.ready {
				t.Fatalf("standing=%+v", state)
			}
			response := httptest.NewRecorder()
			recovery.Health(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
			if response.Code != tc.code {
				t.Fatalf("health=%d %s", response.Code, response.Body.String())
			}
			if tc.failure != nil && !strings.Contains(response.Body.String(), tc.failure.Error()) {
				t.Fatalf("failure evidence erased: %s", response.Body.String())
			}
		})
	}
}

func TestRecoveryConnectHealthDoesNotReportReadyDuringInitialization(t *testing.T) {
	recovery := NewRecovery()
	called := false
	handler := recovery.ConnectHealth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(http.StatusNoContent) }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/agent_manager.v1.AgentManagerService/Health", nil))
	if called || response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":"unavailable"`) || !strings.Contains(response.Body.String(), "initializing") {
		t.Fatalf("RPC health=%d %s called=%t", response.Code, response.Body.String(), called)
	}
	recovery.Start(t.Context(), nil)
	<-recovery.Done()
	defer recovery.Stop()
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/agent_manager.v1.AgentManagerService/Health", nil))
	if !called || response.Code != http.StatusNoContent {
		t.Fatalf("ready RPC health=%d called=%t", response.Code, called)
	}
}

func TestRecoveryShutdownJoinsBeforeStorageCanClose(t *testing.T) {
	recovery := NewRecovery()
	entered, exited := make(chan struct{}), make(chan struct{})
	recovery.Start(t.Context(), []RecoveryStep{{Name: "history", Run: func(ctx context.Context) error { close(entered); <-ctx.Done(); close(exited); return ctx.Err() }}})
	<-entered
	recovery.Stop()
	select {
	case <-exited:
	default:
		t.Fatal("cleanup returned while recovery was using storage")
	}
	if recovery.Snapshot().Readiness {
		t.Fatal("cancelled recovery became ready")
	}
}

func TestRecoveryLifecycleSchemaKeepsBuildIdentityAndFalseReadiness(t *testing.T) {
	t.Setenv("VROOLI_BUILD_IDENTITY", "sha256:maintenance-fixture")
	recovery := NewRecovery()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("initializing recovery reached dependency checks")
	})
	for _, tc := range []struct {
		handler http.Handler
		code    int
	}{
		{recovery.Health(next), http.StatusOK},
		{recovery.Readiness(next), http.StatusServiceUnavailable},
	} {
		response := httptest.NewRecorder()
		tc.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
		var body health.Response
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != tc.code || body.Status != health.StatusDegraded || body.Readiness || body.Service == "" || body.Timestamp == "" || body.BuildIdentity != "sha256:maintenance-fixture" || body.Functional == nil || body.Functional.Healthy {
			t.Fatalf("lifecycle cannot distinguish liveness/readiness: %d %+v", response.Code, body)
		}
	}
}

func TestRecoveryExpiredStepDoesNotHideFailureOrSkipLaterOwner(t *testing.T) {
	recovery := NewRecovery()
	later := false
	recovery.Start(t.Context(), []RecoveryStep{
		{Name: "history", Timeout: time.Nanosecond, Run: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }},
		{Name: "owner_workers", Run: func(ctx context.Context) error { later = true; return ctx.Err() }},
	})
	<-recovery.Done()
	defer recovery.Stop()
	state := recovery.Snapshot()
	if !later || state.Readiness || state.Status != "degraded" || len(state.Failures) != 1 {
		t.Fatalf("expired recovery=%+v later=%t", state, later)
	}
}

func TestRecoveryHistoricalFailureDoesNotPermanentlyCloseReadiness(t *testing.T) {
	for _, critical := range []bool{false, true} {
		r := NewRecovery()
		r.Start(t.Context(), []RecoveryStep{
			{Name: "historical_projection", NonCritical: true, Run: func(context.Context) error { return errors.New("old accounting remains unknown") }},
			{Name: "ownership", Run: func(context.Context) error {
				if critical {
					return errors.New("ownership unknown")
				}
				return nil
			}},
		})
		<-r.Done()
		state := r.Snapshot()
		r.Stop()
		if state.Readiness == critical || state.Status != "degraded" || len(state.Failures) == 0 || !strings.Contains(state.Failures[0], "old accounting remains unknown") {
			t.Fatalf("critical=%t state=%+v", critical, state)
		}
	}
}

func TestRecoveryDeclarationRetryClearsOnlyRepairedStep(t *testing.T) {
	recovery := NewRecovery()
	recovery.Start(t.Context(), []RecoveryStep{
		{Name: "self_declarations", Run: func(context.Context) error { return errors.New("self declaration unavailable") }},
		{Name: "ownership", Run: func(context.Context) error { return errors.New("owner state unknown") }},
		{Name: "scenario_declarations", NonCritical: true, Run: func(context.Context) error { return errors.New("historical declaration failed") }},
	})
	<-recovery.Done()
	defer recovery.Stop()

	if state := recovery.Snapshot(); state.Readiness || state.Status != "degraded" || len(state.Failures) != 3 {
		t.Fatalf("initial state=%+v", state)
	}
	if !recovery.ReconcileDeclarationStep("self_declarations", nil) {
		t.Fatal("known failed declaration step was not reconciled")
	}
	state := recovery.Snapshot()
	if state.Readiness || state.Status != "degraded" || len(state.Failures) != 2 || strings.Contains(strings.Join(state.Failures, " "), "self_declarations") {
		t.Fatalf("repairing self declaration hid another failure: %+v", state)
	}
	if !recovery.ReconcileDeclarationStep("scenario_declarations", nil) {
		t.Fatal("known noncritical declaration step was not reconciled")
	}
	state = recovery.Snapshot()
	if state.Readiness || state.Status != "degraded" || len(state.Failures) != 1 || !strings.Contains(state.Failures[0], "ownership: owner state unknown") {
		t.Fatalf("repairing declarations cleared unrelated critical failure: %+v", state)
	}
	if recovery.ReconcileDeclarationStep("ownership", nil) {
		t.Fatal("declaration retry changed a step it does not own")
	}
}

func TestRecoverySuccessfulDeclarationRetryRestoresReadiness(t *testing.T) {
	recovery := NewRecovery()
	recovery.Start(t.Context(), []RecoveryStep{{Name: "self_declarations", Run: func(context.Context) error {
		return errors.New("initial declaration error")
	}}})
	<-recovery.Done()
	defer recovery.Stop()
	if state := recovery.Snapshot(); state.Readiness || state.Status != "degraded" {
		t.Fatalf("initial state=%+v", state)
	}
	if !recovery.ReconcileDeclarationStep("self_declarations", nil) {
		t.Fatal("known declaration step was not reconciled")
	}
	if state := recovery.Snapshot(); !state.Readiness || state.Status != "ready" || len(state.Failures) != 0 {
		t.Fatalf("successful retry did not restore readiness: %+v", state)
	}
}

func TestRecoveryDeclarationRetryCannotReadyDuringStartupOrShutdown(t *testing.T) {
	recovery := NewRecovery()
	entered, release := make(chan struct{}), make(chan struct{})
	recovery.Start(t.Context(), []RecoveryStep{
		{Name: "self_declarations", Run: func(context.Context) error { return errors.New("initial failure") }},
		{Name: "owner_recovery", Run: func(ctx context.Context) error {
			close(entered)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}},
	})
	<-entered
	if recovery.ReconcileDeclarationStep("self_declarations", nil) {
		t.Fatal("retry updated recovery before startup completed")
	}
	if state := recovery.Snapshot(); state.Readiness || state.Status != "initializing" {
		t.Fatalf("startup became ready early: %+v", state)
	}
	close(release)
	<-recovery.Done()
	recovery.Stop()
	if recovery.ReconcileDeclarationStep("self_declarations", nil) {
		t.Fatal("retry updated recovery after shutdown")
	}
	if state := recovery.Snapshot(); state.Readiness || state.Status != "stopping" {
		t.Fatalf("shutdown became ready: %+v", state)
	}
}

func TestRecoveryStoppedBeforeStartCannotBecomeReady(t *testing.T) {
	recovery := NewRecovery()
	recovery.Stop()
	ran := false
	recovery.Start(t.Context(), []RecoveryStep{{Name: "self_declarations", Run: func(context.Context) error {
		ran = true
		return nil
	}}})
	<-recovery.Done()
	if ran || recovery.ReconcileDeclarationStep("self_declarations", nil) {
		t.Fatal("stopped recovery ran or accepted a deferred update")
	}
	if state := recovery.Snapshot(); state.Readiness || state.Status != "stopping" {
		t.Fatalf("stopped recovery became ready: %+v", state)
	}
}

func TestRecoveryCancelledContextCannotBeReconciledToReady(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	recovery := NewRecovery()
	recovery.Start(ctx, []RecoveryStep{{Name: "self_declarations", Run: func(context.Context) error {
		return errors.New("initial failure")
	}}})
	<-recovery.Done()
	cancel()
	if recovery.ReconcileDeclarationStep("self_declarations", nil) {
		t.Fatal("cancelled recovery accepted a retry")
	}
	if state := recovery.Snapshot(); state.Readiness || state.Status != "stopping" {
		t.Fatalf("cancelled recovery became ready: %+v", state)
	}
}

type blockingDoneContext struct {
	context.Context
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (c *blockingDoneContext) Done() <-chan struct{} {
	c.once.Do(func() {
		close(c.entered)
		<-c.release
	})
	return c.Context.Done()
}

func TestRecoveryStartStopRaceInstallsCancelBeforeStopCanReturn(t *testing.T) {
	parent := &blockingDoneContext{Context: context.Background(), entered: make(chan struct{}), release: make(chan struct{})}
	recovery := NewRecovery()
	var stopReturned atomic.Bool
	var startedAfterStop atomic.Bool
	startReturned := make(chan struct{})
	go func() {
		recovery.Start(parent, []RecoveryStep{{Name: "owner", Run: func(ctx context.Context) error {
			if stopReturned.Load() {
				startedAfterStop.Store(true)
			}
			return ctx.Err()
		}}})
		close(startReturned)
	}()
	<-parent.entered
	stopReturnedCh := make(chan struct{})
	go func() {
		recovery.Stop()
		stopReturned.Store(true)
		close(stopReturnedCh)
	}()
	select {
	case <-stopReturnedCh:
		// With atomic installation this is impossible: Start still owns the
		// recovery lock while parent cancellation is being connected.
	case <-time.After(20 * time.Millisecond):
	}
	close(parent.release)
	<-startReturned
	<-stopReturnedCh
	<-recovery.Done()
	if startedAfterStop.Load() {
		t.Fatal("startup recovery ran after Stop returned without an installed cancel")
	}
}

func TestRecoveryRetryErrorsDoNotCreateOrRewriteStartupFailures(t *testing.T) {
	for _, tc := range []struct {
		name        string
		initialErr  error
		wantStatus  string
		wantReady   bool
		wantFailure string
	}{
		{name: "clean startup stays clean", wantStatus: "ready", wantReady: true},
		{name: "existing failure remains original", initialErr: errors.New("startup declaration failure"), wantStatus: "degraded", wantFailure: "self_declarations: startup declaration failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recovery := NewRecovery()
			recovery.Start(t.Context(), []RecoveryStep{{Name: "self_declarations", Run: func(context.Context) error { return tc.initialErr }}})
			<-recovery.Done()
			defer recovery.Stop()
			if !recovery.ReconcileDeclarationStep("self_declarations", errors.New("retry still failing")) {
				t.Fatal("known retry result was not recognized")
			}
			state := recovery.Snapshot()
			if state.Status != tc.wantStatus || state.Readiness != tc.wantReady {
				t.Fatalf("retry error changed startup readiness: %+v", state)
			}
			if tc.wantFailure == "" {
				if len(state.Failures) != 0 {
					t.Fatalf("retry error introduced startup failure: %+v", state.Failures)
				}
			} else if len(state.Failures) != 1 || state.Failures[0] != tc.wantFailure {
				t.Fatalf("retry error rewrote startup failure: %+v", state.Failures)
			}
		})
	}
}
