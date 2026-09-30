package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
