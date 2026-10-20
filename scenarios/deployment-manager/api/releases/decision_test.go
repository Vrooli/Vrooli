package releases

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatordecision"
)

type recordingRaiser struct {
	raised bool
	err    error
	ask    operatordecision.PendingAsk
	facts  operatordecision.Facts
	calls  int
}

func (r *recordingRaiser) Raise(_ context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts) (bool, error) {
	r.calls++
	r.ask = ask
	r.facts = facts
	return r.raised, r.err
}

func agentContext(parent context.Context) context.Context {
	return identity.WithPrincipal(parent, identity.Principal{Kind: identity.ActorAgent, Subject: "run-agent-7", Verified: true})
}

func startAgentRequest(ctx context.Context, body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", bytes.NewBufferString(body))
	return request.WithContext(ctx)
}

const notYetApprovedBody = `{"git_commit_hash":"abc","artifact_digest":"sha256:candidate","candidate_id":"candidate-1","destination_revision_id":"destination-1","authorization_epoch":1,"readiness_review_key":"rr-1","release_version":"1.0.0","channel":"stable","platforms":["linux-x64"]}`

func pendingReviewApproval(status string) *ReadinessApproval {
	var approvedAt *time.Time
	if status == "approved" {
		approvedAt = timePointer(time.Now())
	}
	return &ReadinessApproval{
		Key: "rr-1", Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc",
		ArtifactDigest: "sha256:candidate", CandidateID: "candidate-1", DestinationRevisionID: "destination-1",
		AuthorizationEpoch: 1, Targets: []string{"linux-x64"}, Channel: "stable", PolicyVersion: 3,
		Status: status, ApprovedAt: approvedAt,
	}
}

// An AGENT asking to ship a release whose readiness review exists but is not yet
// approved gets 202 pending (not 412) and the decision is raised.
func TestStartAgentWithUnapprovedReviewRaisesDecisionAndReturnsPending(t *testing.T) {
	repo, cfg, orch := newMockRepo(), newMockLPBSConfig(), &mockOrch{}
	raiser := &recordingRaiser{raised: true}
	handler := NewHandler(repo, cfg, nil, orch, nil).
		WithExecutionAuthorization(func(context.Context) error { return nil }).
		WithIdentityRepository(&identityRepoFake{}).
		WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) {
			return pendingReviewApproval("pending"), nil
		}).
		WithReleaseDecisionRaiser(raiser).
		WithReleaseGateLookup(func(context.Context, string, string) (bool, string, []map[string]string, error) {
			return true, "all evidence satisfied", []map[string]string{{"target": "linux-x64", "evidence_disposition": "passed", "approval_status": "approved"}}, nil
		})
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, startAgentRequest(agentContext(context.Background()), notYetApprovedBody))

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202 pending, got %d body=%s", response.Code, response.Body.String())
	}
	if raiser.calls != 1 {
		t.Fatalf("expected one raise, got %d", raiser.calls)
	}
	if orch.called != 0 {
		t.Fatalf("release must not execute while pending, orch called %d", orch.called)
	}
	if len(repo.store) != 0 {
		t.Fatalf("a pending ask must not create a release row, got %d", len(repo.store))
	}
	if !strings.Contains(response.Body.String(), `"status":"pending"`) {
		t.Fatalf("expected pending status in body, got %s", response.Body.String())
	}
}

// The raised Facts carry the requesting agent distinct from the approver, are not
// reversible, have no default answer, and the correlation is artifact+target bound.
func TestReleaseDecisionFactsPreserveIndependenceAndAreArtifactBound(t *testing.T) {
	repo, cfg, orch := newMockRepo(), newMockLPBSConfig(), &mockOrch{}
	raiser := &recordingRaiser{raised: true}
	handler := NewHandler(repo, cfg, nil, orch, nil).
		WithExecutionAuthorization(func(context.Context) error { return nil }).
		WithIdentityRepository(&identityRepoFake{}).
		WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) {
			return pendingReviewApproval("pending"), nil
		}).
		WithReleaseDecisionRaiser(raiser)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, startAgentRequest(agentContext(context.Background()), notYetApprovedBody))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", response.Code, response.Body.String())
	}

	facts := raiser.facts
	if facts.Reversible {
		t.Fatal("a release decision must not be reversible")
	}
	if facts.DefaultAnswer != "" {
		t.Fatalf("a non-reversible release decision must have no default answer, got %q", facts.DefaultAnswer)
	}
	if got := facts.Correlation["requesting_agent"]; got != "run-agent-7" {
		t.Fatalf("expected requesting_agent in correlation, got %q", got)
	}
	// The approver is NOT present in the facts: it only arrives on the resolution's
	// answered_by, preserving independence.
	for k, v := range facts.Correlation {
		if strings.Contains(strings.ToLower(k), "approv") && v != "" {
			t.Fatalf("facts must not pre-name an approver, found %s=%s", k, v)
		}
	}
	// Artifact + target bound: the ask id changes when the artifact or target set changes.
	base := releaseDecisionCorrelationID("p1", "abc", "sha256:candidate", []string{"linux-x64"})
	if raiser.ask.ID != base {
		t.Fatalf("ask id %q is not the artifact-bound correlation %q", raiser.ask.ID, base)
	}
	if releaseDecisionCorrelationID("p1", "abc", "sha256:DIFFERENT", []string{"linux-x64"}) == base {
		t.Fatal("a different artifact digest must produce a different correlation id")
	}
	if releaseDecisionCorrelationID("p1", "abc", "sha256:candidate", []string{"darwin-arm64"}) == base {
		t.Fatal("a different target set must produce a different correlation id")
	}
	if raiser.ask.Subject != "rr-1" {
		t.Fatalf("flood cap subject must be the review key, got %q", raiser.ask.Subject)
	}
}

// ErrFlood from the raiser surfaces as 429 without creating a release.
func TestStartAgentReleaseDecisionFloodReturns429(t *testing.T) {
	repo, cfg, orch := newMockRepo(), newMockLPBSConfig(), &mockOrch{}
	raiser := &recordingRaiser{err: operatordecision.ErrFlood}
	handler := NewHandler(repo, cfg, nil, orch, nil).
		WithExecutionAuthorization(func(context.Context) error { return nil }).
		WithIdentityRepository(&identityRepoFake{}).
		WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) {
			return pendingReviewApproval("pending"), nil
		}).
		WithReleaseDecisionRaiser(raiser)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, startAgentRequest(agentContext(context.Background()), notYetApprovedBody))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on flood, got %d body=%s", response.Code, response.Body.String())
	}
	if len(repo.store) != 0 || orch.called != 0 {
		t.Fatalf("flood must not ship, releases=%d orch=%d", len(repo.store), orch.called)
	}
}

// Without a raiser configured, the agent keeps the prior 412 hard-refuse (no regression).
func TestStartAgentWithoutRaiserKeepsPreconditionFailed(t *testing.T) {
	repo, cfg, orch := newMockRepo(), newMockLPBSConfig(), &mockOrch{}
	handler := NewHandler(repo, cfg, nil, orch, nil).
		WithExecutionAuthorization(func(context.Context) error { return nil }).
		WithIdentityRepository(&identityRepoFake{}).
		WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) {
			return pendingReviewApproval("pending"), nil
		})
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, startAgentRequest(agentContext(context.Background()), notYetApprovedBody))
	if response.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 without raiser, got %d body=%s", response.Code, response.Body.String())
	}
}

// An AGENT with an already-approved review still executes exactly as before
// (the ask-don't-block path is skipped).
func TestStartAgentWithApprovedReviewStillExecutes(t *testing.T) {
	repo, cfg, orch := newMockRepo(), newMockLPBSConfig(), &mockOrch{}
	raiser := &recordingRaiser{raised: true}
	handler := NewHandler(repo, cfg, nil, orch, nil).
		WithExecutionAuthorization(func(context.Context) error { return nil }).
		WithIdentityRepository(&identityRepoFake{}).
		WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) {
			return pendingReviewApproval("approved"), nil
		}).
		WithReleaseDecisionRaiser(raiser)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, startAgentRequest(agentContext(context.Background()), notYetApprovedBody))
	// An approved review skips the ask-don't-block path entirely: no decision is
	// raised and the request proceeds to the normal (binding/execution) flow — it
	// is NOT answered with 202 pending. Full execution of an approved agent release
	// is covered by TestAuthorizedAgentExecutesExactApprovedReleaseWithoutBecomingApprover.
	if response.Code == http.StatusAccepted {
		t.Fatalf("approved review must not return pending (202), got %d body=%s", response.Code, response.Body.String())
	}
	if raiser.calls != 0 {
		t.Fatalf("approved review must not raise a decision, got %d", raiser.calls)
	}
}

// A HUMAN caller is unchanged: the ask-don't-block path is agent-only, so a human
// with an unapproved review keeps the 412 precondition failure.
func TestStartHumanWithUnapprovedReviewUnchanged(t *testing.T) {
	repo, cfg, orch := newMockRepo(), newMockLPBSConfig(), &mockOrch{}
	raiser := &recordingRaiser{raised: true}
	handler := NewHandler(repo, cfg, nil, orch, nil).
		WithAuthorization(func(context.Context) error { return nil }).
		WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) {
			return pendingReviewApproval("pending"), nil
		}).
		WithReleaseDecisionRaiser(raiser)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, startAgentRequest(ctx, notYetApprovedBody))
	if response.Code != http.StatusPreconditionFailed {
		t.Fatalf("human path must be unchanged (412), got %d body=%s", response.Code, response.Body.String())
	}
	if raiser.calls != 0 {
		t.Fatalf("human must not raise an agent decision, got %d", raiser.calls)
	}
}
