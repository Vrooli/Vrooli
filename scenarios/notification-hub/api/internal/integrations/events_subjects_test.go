package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/operatordecision"
)

const (
	subjectSource      = "data-backup-manager"
	subjectRequestType = "data_backup_manager.backup_failure.decision_requested.v1"
	subjectWithdrawn   = "data_backup_manager.backup_failure.decision_withdrawn.v1"
)

type subjectHarness struct {
	t       *testing.T
	service *hub.Service
	db      *database.RoutedDB
	handler http.Handler
	secret  string
}

func newSubjectHarness(t *testing.T) *subjectHarness {
	t.Helper()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "subject-test-secret"
	return &subjectHarness{t: t, service: service, db: db, handler: EventWebhook(service, secret), secret: secret}
}

func (h *subjectHarness) post(eventID, source, eventType string, payload map[string]any) (int, map[string]any) {
	h.t.Helper()
	body, err := json.Marshal(map[string]any{"event_type": eventType, "source_scenario": source, "payload": payload})
	require.NoError(h.t, err)
	recorder := httptest.NewRecorder()
	h.handler.ServeHTTP(recorder, signedEventRequest(h.t, h.secret, eventID, body))
	var response map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	return recorder.Code, response
}

func blockedBackup(extra map[string]any) map[string]any {
	payload := map[string]any{
		"subject":     "Backups",
		"question":    "Backups to the Elements drive are blocked. Retry or skip?",
		"options":     []map[string]string{{"key": "retry", "label": "Retry"}, {"key": "skip", "label": "Skip"}},
		"recommended": "retry",
		"subject_key": "backup-failure:elements",
	}
	for key, value := range extra {
		payload[key] = value
	}
	return payload
}

func (h *subjectHarness) openAsks() int {
	h.t.Helper()
	var count int
	require.NoError(h.t, h.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks WHERE state IN ('pending','escalated')`).Scan(&count))
	return count
}

func TestDecisionAskSpecParsesSubjectFieldsAndRejectsBadOnes(t *testing.T) {
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	spec := func(extra map[string]any) (hub.AskSpec, error) {
		encoded, err := json.Marshal(blockedBackup(extra))
		require.NoError(t, err)
		return DecisionAskSpec("operator", subjectSource, subjectRequestType, "evt-1", encoded, now)
	}
	got, err := spec(map[string]any{"reassert_within_seconds": 900, "asserted_at": "2026-11-13T08:59:30.5Z"})
	require.NoError(t, err)
	require.Equal(t, "backup-failure:elements", got.SubjectKey)
	require.Equal(t, 15*time.Minute, got.ReassertWithin)
	require.True(t, got.AssertedAt.Equal(time.Date(2026, 11, 13, 8, 59, 30, 500_000_000, time.UTC)))

	plain, err := DecisionAskSpec("operator", subjectSource, subjectRequestType, "evt-2", mustJSON(t, map[string]any{
		"question": "q?", "options": []map[string]string{{"key": "a"}, {"key": "b"}}, "recommended": "a",
	}), now)
	require.NoError(t, err)
	require.Empty(t, plain.SubjectKey)
	require.Zero(t, plain.ReassertWithin)

	_, err = spec(map[string]any{"asserted_at": "yesterday"})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	_, err = spec(map[string]any{"reassert_within_seconds": -5})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	_, err = spec(map[string]any{"reassert_within_seconds": int64(1) << 40})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return encoded
}

// The producer contract (operatordecision) and the hub's intake must agree on
// the wire: whatever Facts and Withdrawal emit is what the hub accepts.
func TestWebhookAcceptsWhatTheOperatordecisionProducerEmits(t *testing.T) {
	h := newSubjectHarness(t)
	facts := operatordecision.Facts{
		Subject: "Backups", Question: "Backups to the Elements drive are blocked. Retry or skip?",
		Options:     []operatordecision.Option{{Key: "retry", Label: "Retry"}, {Key: "skip", Label: "Skip"}},
		Recommended: "retry", SubjectKey: "backup-failure:elements", ReassertWithin: 15 * time.Minute,
		AssertedAt: time.Now().UTC(),
	}
	require.NoError(t, facts.Validate())
	event := operatordecision.BuildEvent(subjectSource, "backup_failure", time.Now().UTC(), facts)
	code, response := h.post("evt-1", event.Source, event.EventType, event.Payload)
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, hub.AskCreated, response["outcome"])
	askID := response["ask_id"].(string)

	closing := operatordecision.BuildWithdrawalEvent(subjectSource, "backup_failure", time.Now().UTC().Add(time.Minute), operatordecision.Withdrawal{
		SubjectKey: "backup-failure:elements", Reason: "Backups to the Elements drive succeeded again at 14:02.",
	})
	code, response = h.post("evt-2", closing.Source, closing.EventType, closing.Payload)
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, true, response["withdrawn"])
	ask, err := h.service.GetAsk(context.Background(), askID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, ask.State)
	require.Equal(t, "Backups to the Elements drive succeeded again at 14:02.", ask.Reason)
}

func TestWebhookDecisionRequestsFollowTheSubjectRules(t *testing.T) {
	h := newSubjectHarness(t)

	code, first := h.post("evt-1", subjectSource, subjectRequestType, blockedBackup(nil))
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, hub.AskCreated, first["outcome"])
	askID := first["ask_id"].(string)

	// The same facts re-sent: one decision, nothing new.
	code, again := h.post("evt-2", subjectSource, subjectRequestType, blockedBackup(map[string]any{"asserted_at": time.Now().UTC().Format(time.RFC3339Nano)}))
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, hub.AskReasserted, again["outcome"])
	require.Equal(t, askID, again["ask_id"])

	// New words for the same menu: updated in place.
	code, updated := h.post("evt-3", subjectSource, subjectRequestType, blockedBackup(map[string]any{
		"question": "Backups to the Elements drive have been blocked for a day. Retry or skip?",
	}))
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, hub.AskUpdated, updated["outcome"])
	require.Equal(t, askID, updated["ask_id"])
	ask, err := h.service.GetAsk(context.Background(), askID)
	require.NoError(t, err)
	require.Contains(t, ask.Question, "blocked for a day")
	require.Equal(t, 1, h.openAsks())

	// A different menu replaces it.
	code, replaced := h.post("evt-4", subjectSource, subjectRequestType, blockedBackup(map[string]any{
		"options": []map[string]string{{"key": "retry", "label": "Retry"}, {"key": "skip", "label": "Skip"}, {"key": "pause", "label": "Pause"}},
	}))
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, hub.AskReplaced, replaced["outcome"])
	require.NotEqual(t, askID, replaced["ask_id"])
	require.Equal(t, 1, h.openAsks())

	// An invalid subject key is a client error.
	code, _ = h.post("evt-5", subjectSource, subjectRequestType, blockedBackup(map[string]any{"subject_key": "bad key"}))
	require.Equal(t, http.StatusBadRequest, code)
}

func TestWebhookReportsAStaleRequestAfterAWithdrawal(t *testing.T) {
	h := newSubjectHarness(t)
	asserted := time.Now().UTC().Add(-time.Hour)
	code, _ := h.post("evt-1", subjectSource, subjectRequestType, blockedBackup(map[string]any{"asserted_at": asserted.Format(time.RFC3339Nano)}))
	require.Equal(t, http.StatusAccepted, code)
	code, closed := h.post("evt-2", subjectSource, subjectWithdrawn, map[string]any{
		"subject_key": "backup-failure:elements", "reason": "recovered", "asserted_at": asserted.Add(10 * time.Minute).Format(time.RFC3339Nano),
	})
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, true, closed["withdrawn"])

	// The request that was delayed in the queue, asserted before the withdrawal.
	code, stale := h.post("evt-late", subjectSource, subjectRequestType, blockedBackup(map[string]any{
		"asserted_at": asserted.Add(5 * time.Minute).Format(time.RFC3339Nano), "question": "A delayed question.",
	}))
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, true, stale["stale"])
	require.NotContains(t, stale, "ask_id")
	require.Zero(t, h.openAsks())
}

func TestWebhookGenericWithdrawalAcceptsAnySourceByOwnSubjectOrRef(t *testing.T) {
	h := newSubjectHarness(t)
	code, first := h.post("evt-1", subjectSource, subjectRequestType, blockedBackup(nil))
	require.Equal(t, http.StatusAccepted, code)
	askID := first["ask_id"].(string)

	// Another source naming the same subject key closes nothing.
	code, other := h.post("evt-x", "storage-manager", "storage_manager.backup_failure.decision_withdrawn.v1", map[string]any{
		"subject_key": "backup-failure:elements", "reason": "not mine",
	})
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, false, other["withdrawn"])
	// The wrong request type for the right source closes nothing either.
	code, wrongType := h.post("evt-y", subjectSource, "data_backup_manager.restore.decision_withdrawn.v1", map[string]any{
		"subject_key": "backup-failure:elements", "reason": "wrong kind",
	})
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, false, wrongType["withdrawn"])
	require.Equal(t, 1, h.openAsks())

	// An explicit source_event_type overrides the derived one.
	code, explicit := h.post("evt-z", subjectSource, "data_backup_manager.cleared.decision_withdrawn.v1", map[string]any{
		"requester_ref": "evt-1", "source_event_type": subjectRequestType, "reason": "cleared",
	})
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, true, explicit["withdrawn"])
	ask, err := h.service.GetAsk(context.Background(), askID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, ask.State)
	require.Equal(t, "cleared", ask.Reason)

	for name, payload := range map[string]map[string]any{
		"no identity":     {"reason": "x"},
		"no reason":       {"subject_key": "backup-failure:elements"},
		"reason too long": {"subject_key": "backup-failure:elements", "reason": string(make([]byte, 301))},
		"bad asserted_at": {"subject_key": "backup-failure:elements", "reason": "x", "asserted_at": "soon"},
		"bad subject key": {"subject_key": "no spaces allowed", "reason": "x"},
		"not a request":   {"subject_key": "a", "reason": "x", "source_event_type": "a.b.notification.v1"},
	} {
		code, _ := h.post("evt-bad-"+name, subjectSource, subjectWithdrawn, payload)
		require.Equal(t, http.StatusBadRequest, code, name)
	}
}

// Agent Manager's typed withdrawals keep working through the generic
// subscription: their fixed reasons are not free text.
func TestWebhookAgentManagerTypedWithdrawalsKeepTheirFixedReasons(t *testing.T) {
	h := newSubjectHarness(t)
	code, created := h.post("run-evt-1", "agent-manager", RunApprovalDecisionEventType, map[string]any{
		"question": "Apply this run?", "options": []map[string]string{{"key": "approve"}, {"key": "reject"}}, "recommended": "approve",
	})
	require.Equal(t, http.StatusAccepted, code)
	// A free-text reason is not a typed reason.
	code, _ = h.post("w-1", "agent-manager", DecisionWithdrawalEventType, map[string]any{
		"requester_ref": "run-evt-1", "source_event_type": RunApprovalDecisionEventType, "reason": "because I said so",
	})
	require.Equal(t, http.StatusBadRequest, code)
	code, closed := h.post("w-2", "agent-manager", DecisionWithdrawalEventType, map[string]any{
		"requester_ref": "run-evt-1", "source_event_type": RunApprovalDecisionEventType, "reason": "verified_empty_diff",
	})
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, true, closed["withdrawn"])
	ask, err := h.service.GetAsk(context.Background(), created["ask_id"].(string))
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, ask.State)
	require.Equal(t, "requester confirmed no decision is needed", ask.Reason)

	// Agent Manager's other decision kinds use the generic contract.
	code, usage := h.post("usage-1", "agent-manager", "agent_manager.usage_mode.decision_requested.v1", blockedBackup(map[string]any{"subject_key": "usage-mode"}))
	require.Equal(t, http.StatusAccepted, code)
	code, closedUsage := h.post("usage-2", "agent-manager", "agent_manager.usage_mode.decision_withdrawn.v1", map[string]any{
		"subject_key": "usage-mode", "reason": "Usage mode was switched back to normal.",
	})
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, true, closedUsage["withdrawn"])
	ask, err = h.service.GetAsk(context.Background(), usage["ask_id"].(string))
	require.NoError(t, err)
	require.Equal(t, "Usage mode was switched back to normal.", ask.Reason)
}

func TestWebhookWithdrawalArrivingBeforeItsRequestSuppressesTheDelayedRequest(t *testing.T) {
	h := newSubjectHarness(t)
	code, _ := h.post("w-1", subjectSource, subjectWithdrawn, map[string]any{"requester_ref": "evt-late", "reason": "recovered"})
	require.Equal(t, http.StatusAccepted, code)
	code, response := h.post("evt-late", subjectSource, subjectRequestType, blockedBackup(nil))
	require.Equal(t, http.StatusAccepted, code)
	require.Equal(t, true, response["withdrawn"])
	require.Zero(t, h.openAsks())
}
