package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	checksproto "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

type observationStore struct {
	mockStore
	saved []checks.Result
}

func (s *observationStore) SaveResult(ctx context.Context, r checks.Result) error {
	s.saved = append(s.saved, r)
	return s.mockStore.SaveResult(ctx, r)
}

type countedObservation struct {
	mockCheck
	calls int
}

func (c *countedObservation) Run(ctx context.Context) checks.Result {
	c.calls++
	return c.mockCheck.Run(ctx)
}

func TestRunCheckRefreshesOnlyRequestedObservationWithoutHealingOrFleetFreshness(t *testing.T) {
	store := &observationStore{}
	h, healable := setupTestHandlersWithAutoHeal(store)
	other := &countedObservation{mockCheck: mockCheck{id: "other", status: checks.StatusOK}}
	h.registry.Register(other)
	old := time.Now().Add(-2 * time.Hour)
	h.tickEnded = old
	h.registry.SetResult(checks.Result{CheckID: "critical-check", Status: checks.StatusOK, Timestamp: old})
	resp, err := (&typedChecks{h: h}).RunCheck(context.Background(), connect.NewRequest(&checksproto.RunCheckRequest{CheckId: "critical-check"}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Result.GetStatus() != checksproto.CheckStatus_CHECK_STATUS_CRITICAL || !resp.Msg.Result.GetObservedAt().AsTime().After(old) {
		t.Fatalf("observation not refreshed: %v", resp.Msg)
	}
	if other.calls != 0 || healable.executeCalled || store.actionLogCalls != 0 {
		t.Fatalf("unrequested work: other=%d healed=%v actions=%d", other.calls, healable.executeCalled, store.actionLogCalls)
	}
	if len(store.saved) != 1 || store.saved[0].CheckID != "critical-check" || len(store.outageObserved) != 1 {
		t.Fatalf("observation not persisted: %+v", store.saved)
	}
	if !h.tickEnded.Equal(old) || h.tickRunning {
		t.Fatal("single observation changed fleet freshness or retained admission")
	}
	got, ok := h.registry.GetResult("critical-check")
	if !ok || got.Status != checks.StatusCritical {
		t.Fatal("current observation did not replace retained result")
	}
}

func TestRunCheckRejectsInvalidBusyCanceledAndUnpersistedRequests(t *testing.T) {
	for _, tc := range []struct {
		name, id                   string
		busy, cancel, expire, fail bool
		code                       connect.Code
	}{
		{name: "empty", code: connect.CodeInvalidArgument},
		{name: "unknown", id: "missing", code: connect.CodeNotFound},
		{name: "busy", id: "test-check", busy: true, code: connect.CodeAlreadyExists},
		{name: "canceled", id: "test-check", cancel: true, code: connect.CodeCanceled},
		{name: "deadline", id: "test-check", expire: true, code: connect.CodeDeadlineExceeded},
		{name: "persistence", id: "test-check", fail: true, code: connect.CodeInternal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &observationStore{}
			if tc.fail {
				s.saveErr = errors.New("store failed")
			}
			h := setupTestHandlers(s)
			h.tickRunning = tc.busy
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.expire {
				var expiredCancel context.CancelFunc
				ctx, expiredCancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer expiredCancel()
			}
			if tc.cancel {
				cancel()
			}
			_, err := (&typedChecks{h: h}).RunCheck(ctx, connect.NewRequest(&checksproto.RunCheckRequest{CheckId: tc.id}))
			if connect.CodeOf(err) != tc.code {
				t.Fatalf("code=%v, err=%v; want %v", connect.CodeOf(err), err, tc.code)
			}
			if !tc.fail && len(s.saved) != 0 {
				t.Fatal("rejected request persisted an observation")
			}
			if h.tickRunning != tc.busy {
				t.Fatal("observation admission was not released")
			}
		})
	}
}
