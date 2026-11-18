package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/hookreceipt"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

const (
	HookReceiptPath        = "/api/v1/runtime/hook-receipts"
	hookReceiptTokenHeader = "X-Vrooli-Hook-Receipt-Token"
)

type hookReceiptRunReader interface {
	GetStoredRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
}

// HookReceiptHandler accepts only owner-authenticated, launch-scoped hook
// receipts. The bearer Agent Manager identity determines the run; request
// fields never select or create a run association.
type HookReceiptHandler struct {
	Store hookreceipt.Consumer
	Runs  hookReceiptRunReader
	Now   func() time.Time
}

type hookReceiptRequest struct {
	ReceiptID    string    `json:"receipt_id"`
	InvocationID string    `json:"invocation_id"`
	SessionID    string    `json:"session_id"`
	Runner       string    `json:"runner"`
	EvaluatorID  string    `json:"evaluator_id"`
	DurationMS   int64     `json:"duration_ms"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func (h HookReceiptHandler) Receive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.Store == nil || h.Runs == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "hook receipt owner unavailable"})
		return
	}
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || principal.Kind != identity.ActorAgent || !principal.IsVerified() {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "verified Agent Manager run identity required"})
		return
	}
	verifiedProvenance := provenance.FromContext(r.Context())
	if !verifiedProvenance.IsVerifiedAgent() || verifiedProvenance.RunID != strings.TrimSpace(principal.Subject) || provenance.ForwardedIdentityToken(r.Context()) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "verified Agent Manager run provenance required"})
		return
	}
	runID, err := uuid.Parse(strings.TrimSpace(principal.Subject))
	if err != nil || runID == uuid.Nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "run identity is not bound to a valid Agent Manager run"})
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<10))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hook receipt body"})
		return
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var req hookReceiptRequest
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed hook receipt"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hook receipt must contain one JSON object"})
		return
	}
	receiptID, err := uuid.Parse(strings.TrimSpace(req.ReceiptID))
	if err != nil || receiptID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid receipt identity"})
		return
	}
	invocationID, err := uuid.Parse(strings.TrimSpace(req.InvocationID))
	if err != nil || invocationID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid invocation identity"})
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Runner = strings.TrimSpace(req.Runner)
	req.EvaluatorID = strings.TrimSpace(req.EvaluatorID)
	if req.SessionID == "" || len(req.SessionID) > 256 || !validHookReceiptRunner(req.Runner) || !validHookReceiptEvaluator(req.EvaluatorID) || req.DurationMS < 0 || req.DurationMS > 10*60*1000 || req.OccurredAt.IsZero() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hook receipt fields are invalid"})
		return
	}
	now := time.Now().UTC()
	if h.Now != nil {
		now = h.Now().UTC()
	}
	if req.OccurredAt.After(now.Add(30 * time.Second)) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hook receipt timestamp is in the future"})
		return
	}
	run, err := h.Runs.GetStoredRun(r.Context(), runID)
	if err != nil || run == nil || run.ID != runID || run.Status.IsTerminal() || run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "receipt run is not an active authenticated launch"})
		return
	}
	if strings.TrimSpace(run.SessionID) == "" || run.SessionID != req.SessionID {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "hook session does not match the Agent Manager run"})
		return
	}
	credential := strings.TrimSpace(r.Header.Get(hookReceiptTokenHeader))
	if credential == "" || len(credential) > 256 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "launch receipt credential required"})
		return
	}
	err = h.Store.Consume(r.Context(), credential, runID, invocationID, hookreceipt.Receipt{
		ReceiptID: receiptID.String(), SessionID: req.SessionID, Runner: req.Runner,
		EvaluatorID: req.EvaluatorID, DurationMS: req.DurationMS, OccurredAt: req.OccurredAt.UTC(),
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "launch receipt could not be authenticated or consumed"})
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func validHookReceiptRunner(value string) bool {
	for _, runner := range domain.ValidRunnerTypes() {
		if string(runner) == value {
			return true
		}
	}
	return false
}

func validHookReceiptEvaluator(value string) bool {
	provider, rule, ok := strings.Cut(value, ":")
	if !ok || provider == "" || rule == "" || strings.Contains(rule, ":") || len(value) > 256 {
		return false
	}
	for _, char := range value {
		if char == ':' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("._/-", char) {
			continue
		}
		return false
	}
	return true
}
