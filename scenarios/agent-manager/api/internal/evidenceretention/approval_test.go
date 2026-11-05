package evidenceretention

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatordecision"
)

type capturePublisher struct{ events []eventbus.DomainEvent }

func (p *capturePublisher) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	p.events = append(p.events, e)
	return nil
}

const approvalEffort = "demo-2026-10-20"

// approvalHarness is one evidence home with the agent approval path wired
// through the real operatordecision Raiser, a captured publisher, and a
// synchronous apply so tests observe the prune deterministically.
type approvalHarness struct {
	root      string
	svc       *Service
	approvals *Approvals
	handler   *Handler
	published *capturePublisher
	removed   []string
}

func newApprovalHarness(t *testing.T) *approvalHarness {
	t.Helper()
	root, records := fixture(t, approvalEffort)
	h := &approvalHarness{root: root, svc: newService(t, root, records, nil), published: &capturePublisher{}}
	h.svc.opts.remove = func(p string) error {
		h.removed = append(h.removed, p)
		return os.RemoveAll(p)
	}
	store := NewFileAskStore(filepath.Join(filepath.Dir(root), "receipts", "approvals"))
	h.approvals = NewApprovals(h.svc, store, NewApprovalRaiser(store, h.published))
	h.approvals.run = func(f func()) { f() }
	owner := func(*http.Request) (string, int, error) { return "", http.StatusUnauthorized, os.ErrPermission }
	h.handler = NewHandler(h.svc, owner, h.approvals)
	return h
}

var agentRunID = uuid.NewString()

func (h *approvalHarness) agentPrune(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/evidence/prune", strings.NewReader(body))
	req = req.WithContext(identity.WithPrincipal(req.Context(), identity.Principal{Kind: identity.ActorAgent, Subject: agentRunID, Verified: true}))
	req.Header.Set(WebConsoleSessionIDHeader, "wc-session-1")
	rec := httptest.NewRecorder()
	h.handler.Prune(rec, req)
	return rec
}

const agentApply = `{"efforts":["` + approvalEffort + `"],"classes":["build_binary","release_archive"],"dry_run":false,"reason":"effort finished"}`

func (h *approvalHarness) raise(t *testing.T) string {
	t.Helper()
	rec := h.agentPrune(t, agentApply)
	var out struct {
		Status       string `json:"status"`
		DecisionID   string `json:"decision_id"`
		PreviewBytes int64  `json:"preview_bytes"`
		PreviewFiles int    `json:"preview_files"`
	}
	if rec.Code != http.StatusAccepted || json.Unmarshal(rec.Body.Bytes(), &out) != nil || out.Status != "approval_pending" || out.DecisionID == "" {
		t.Fatalf("agent apply = %d %s", rec.Code, rec.Body.String())
	}
	if out.PreviewFiles != 2 || out.PreviewBytes <= 0 {
		t.Fatalf("preview = %+v", out)
	}
	return out.DecisionID
}

func resolution(id, state, answer, by string) operatordecision.DecisionResolution {
	return operatordecision.DecisionResolution{
		State: state, Answer: answer, AnsweredBy: by, SourceEventType: ApprovalEventType,
		Correlation: map[string]string{correlationRequestID: id},
	}
}

func (h *approvalHarness) exists(rel string) bool {
	_, err := os.Lstat(filepath.Join(h.root, approvalEffort, rel))
	return err == nil
}

func TestAgentApplyRaisesOperatorDecisionInsteadOfRemoving(t *testing.T) {
	h := newApprovalHarness(t)
	id := h.raise(t)
	if len(h.removed) != 0 || !h.exists("E3/demo/files/data/renderjobs/a/worker") {
		t.Fatal("an agent request removed evidence before approval")
	}
	if h.published.len() != 1 {
		t.Fatalf("published %d decisions, want 1", len(h.published.events))
	}
	event := h.published.events[0]
	if event.EventType != ApprovalEventType || event.Payload["recommended"] != "deny" || event.Payload["reversible"] != false || event.Payload["default_answer"] != nil {
		t.Fatalf("decision event = %+v", event)
	}
	details, _ := event.Payload["technical_details"].(string)
	for _, want := range []string{approvalEffort, "build_binary, release_archive", "2 files", "effort finished", agentRunID} {
		if !strings.Contains(details, want) {
			t.Errorf("technical details lack %q:\n%s", want, details)
		}
	}
	corr, _ := event.Payload["correlation"].(map[string]any)
	if corr[operatordecision.CorrelationRaisedByRunID] != agentRunID || corr[operatordecision.CorrelationWebConsoleSessionID] != "wc-session-1" || corr[operatordecision.CorrelationAgentAwaitKey] != id {
		t.Fatalf("correlation = %+v", event.Payload["correlation"])
	}
	// A retry (any class order, any reason) dedupes onto the one ask.
	again := h.agentPrune(t, `{"efforts":["`+approvalEffort+`"],"classes":["release_archive","build_binary"],"reason":"retry"}`)
	if again.Code != http.StatusAccepted || !strings.Contains(again.Body.String(), id) || len(h.published.events) != 1 {
		t.Fatalf("retry = %d %s, published %d", again.Code, again.Body.String(), len(h.published.events))
	}
	// A dry run stays an ordinary preview for agents.
	if rec := h.agentPrune(t, `{"efforts":["`+approvalEffort+`"],"dry_run":true}`); rec.Code != http.StatusOK {
		t.Fatalf("agent dry run = %d", rec.Code)
	}
}

func (p *capturePublisher) len() int { return len(p.events) }

func TestApprovedDecisionAppliesPreviewOnceWithApprover(t *testing.T) {
	h := newApprovalHarness(t)
	id := h.raise(t)
	res := resolution(id, operatordecision.StateAnswered, "approve", "operator@example.test")
	if !h.approvals.Handles(res.SourceEventType) || h.approvals.Handles("agent_manager.authorization.decision_requested.v1") {
		t.Fatal("approvals must own exactly evidence-prune resolutions")
	}
	if handled, err := h.approvals.Apply(context.Background(), res); !handled || err != nil {
		t.Fatalf("apply = %v %v", handled, err)
	}
	if len(h.removed) != 2 || h.exists("E3/demo/files/data/renderjobs/a/worker") || h.exists("E3/source/blender-5.2.2-linux-x64.tar.xz") {
		t.Fatalf("removed = %v", h.removed)
	}
	if !h.exists("E1/stage/node_modules") || !h.exists("E1/report.json") {
		t.Fatal("an approval removed outside its classes")
	}
	status, err := h.approvals.Status(context.Background(), id)
	if err != nil || status == nil || status.State != operatordecision.PendingStateApproved || status.ResolvedBy != "operator@example.test" || status.ReceiptPath == "" {
		t.Fatalf("status = %+v %v", status, err)
	}
	data, err := os.ReadFile(status.ReceiptPath)
	var receipt Receipt
	if err != nil || json.Unmarshal(data, &receipt) != nil {
		t.Fatalf("receipt: %v", err)
	}
	if receipt.Actor != "operator@example.test" || receipt.Approval == nil || receipt.Approval.DecisionID != id || receipt.Approval.RequestedBy != agentRunID || receipt.RemovedFiles != 2 || receipt.DryRun {
		t.Fatalf("receipt = %+v", receipt)
	}
	// A replayed resolution is a benign no-op: nothing is pruned twice.
	if handled, err := h.approvals.Apply(context.Background(), res); !handled || err != nil || len(h.removed) != 2 {
		t.Fatalf("replay = %v %v removed=%d", handled, err, len(h.removed))
	}
}

func TestDeniedOrUnattributedDecisionRemovesNothing(t *testing.T) {
	for name, res := range map[string]func(string) operatordecision.DecisionResolution{
		"deny": func(id string) operatordecision.DecisionResolution {
			return resolution(id, operatordecision.StateAnswered, "deny", "operator")
		},
		"no approver": func(id string) operatordecision.DecisionResolution {
			return resolution(id, operatordecision.StateAnswered, "approve", "")
		},
		"defaulted": func(id string) operatordecision.DecisionResolution {
			return resolution(id, operatordecision.StateDefaulted, "approve", "operator")
		},
		"expired": func(id string) operatordecision.DecisionResolution {
			return resolution(id, operatordecision.StateExpired, "", "")
		},
		"other answer": func(id string) operatordecision.DecisionResolution {
			return resolution(id, operatordecision.StateAnswered, "Approve all", "operator")
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newApprovalHarness(t)
			id := h.raise(t)
			if handled, err := h.approvals.Apply(context.Background(), res(id)); !handled || err != nil {
				t.Fatalf("apply = %v %v", handled, err)
			}
			// A later approval of the decided ask cannot revive it.
			_, _ = h.approvals.Apply(context.Background(), resolution(id, operatordecision.StateAnswered, "approve", "operator"))
			if len(h.removed) != 0 {
				t.Fatalf("removed = %v", h.removed)
			}
			if status, _ := h.approvals.Status(context.Background(), id); status == nil || status.State != operatordecision.PendingStateRejected {
				t.Fatalf("status = %+v", status)
			}
		})
	}
}

func TestApprovedScopeCannotWiden(t *testing.T) {
	h := newApprovalHarness(t)
	id := h.raise(t)
	// After the operator saw the preview, a new binary of the approved class
	// appears and another class is present in the same effort.
	write(t, filepath.Join(h.root, approvalEffort, "E9/late/worker"), elf(2<<20))
	write(t, filepath.Join(h.root, "other-2026-10-20", "E1/worker"), elf(2<<20))
	res := resolution(id, operatordecision.StateAnswered, "approve", "operator")
	// Fields smuggled onto the resolution never reach the prune.
	res.Correlation["efforts"] = `["other-2026-10-20"]`
	res.Correlation["classes"] = `[]`
	res.Correlation["paths"] = `["other-2026-10-20/E1/worker"]`
	if handled, err := h.approvals.Apply(context.Background(), res); !handled || err != nil {
		t.Fatalf("apply = %v %v", handled, err)
	}
	if len(h.removed) != 2 {
		t.Fatalf("removed = %v", h.removed)
	}
	if !h.exists("E9/late/worker") || !h.exists("E1/stage/node_modules") {
		t.Fatal("an approval widened past its previewed paths or classes")
	}
	if _, err := os.Stat(filepath.Join(h.root, "other-2026-10-20", "E1/worker")); err != nil {
		t.Fatal("an approval widened to another effort")
	}
	// An unknown or foreign request id is acknowledged without change.
	if handled, err := h.approvals.Apply(context.Background(), resolution("evidence-prune-unknown", operatordecision.StateAnswered, "approve", "operator")); !handled || err != nil {
		t.Fatalf("unknown = %v %v", handled, err)
	}
}

func TestAgentRequestRequiresNamedEffortAndReason(t *testing.T) {
	h := newApprovalHarness(t)
	for _, body := range []string{
		`{"dry_run":false,"reason":"r"}`,
		`{"efforts":["` + approvalEffort + `"],"dry_run":false}`,
		`{"efforts":["../etc"],"dry_run":false,"reason":"r"}`,
	} {
		if rec := h.agentPrune(t, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s = %d %s", body, rec.Code, rec.Body.String())
		}
	}
	// Nothing removable raises nothing.
	rec := h.agentPrune(t, `{"efforts":["missing-effort"],"dry_run":false,"reason":"r"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "nothing_to_prune") || h.published.len() != 0 {
		t.Fatalf("empty preview = %d %s", rec.Code, rec.Body.String())
	}
}
