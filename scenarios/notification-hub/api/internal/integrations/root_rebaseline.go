package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/rootrebaselinedecision"
	"notification-hub/internal/hub"
)

const (
	RootRebaselinePublishRoute = "/api/v1/internal/root-rebaseline/publish"
	RootRebaselineLookupRoute  = "/api/v1/internal/root-rebaseline/lookup"
	RootRebaselineConsumeRoute = "/api/v1/internal/root-rebaseline/consume"
	RootRebaselineEventType    = "root.rebaseline.decision_requested.v1"
	RootRebaselineSubject      = "root-rebaseline"
	RootRebaselineAudience     = "scenario-authenticator:notification-hub-root-rebaseline"
	RootRebaselineScope        = "notification-hub:root-rebaseline"
)

type rootRebaselineStore interface {
	providerDecisionAskWriter
	GetAskByRequester(context.Context, string, string, string) (hub.AskRecord, error)
	RootRebaselineConsumedAt(context.Context, string) (string, error)
}

type rootRebaselineConsumer interface {
	ConsumeRootRebaseline(context.Context, string, string) (hub.AskRecord, string, error)
}

// RootRebaselineConsumeHandler spends exactly one verified human approval.
func RootRebaselineConsumeHandler(store rootRebaselineConsumer, verifier authn.TokenVerifier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if store == nil || verifier == nil {
			http.Error(w, "decision service unavailable", http.StatusServiceUnavailable)
			return
		}
		principal, err := providerDecisionPrincipal(r, verifier)
		if err != nil || !principal.Verified || principal.Kind != apiidentity.ActorService || principal.Subject != RootRebaselineSubject || principal.Audience != RootRebaselineAudience || !slices.Contains(principal.Scopes, RootRebaselineScope) {
			http.Error(w, "service identity required", http.StatusUnauthorized)
			return
		}
		var in rootRebaselineRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil {
			http.Error(w, "invalid decision request", http.StatusBadRequest)
			return
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) || in.RequesterRef == "" || strings.TrimSpace(in.RequesterRef) != in.RequesterRef || len(in.RequesterRef) > 128 || !validProviderDecisionDigest(in.Digest) || in.Version != 0 || in.Operation != "" || in.Deadline != "" || in.CandidateSHA != "" || in.ServingSHA != "" || in.WatchdogSHA != "" || in.BundleSHA != "" || in.CandidateContract != "" || in.ServingContract != "" || len(in.Changes) != 0 {
			http.Error(w, "invalid decision request", http.StatusBadRequest)
			return
		}
		ask, consumed, err := store.ConsumeRootRebaseline(r.Context(), in.RequesterRef, in.Digest)
		if err != nil {
			if errors.Is(err, hub.ErrRootDecisionUnavailable) {
				http.Error(w, "decision unavailable", http.StatusConflict)
			} else {
				http.Error(w, "decision service unavailable", http.StatusServiceUnavailable)
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"ask_id": ask.ID, "requester_ref": ask.RequesterRef, "canonical_terms_sha256": in.Digest, "state": ask.State, "answer": ask.Answer, "answered_by_identity_kind": ask.AnsweredByIdentityKind, "answered_by_identity_source": ask.AnsweredByIdentitySource, "created_at": ask.CreatedAt, "answered_at": ask.AnsweredAt, "deadline": ask.Deadline, "consumed_at": consumed})
	})
}

type rootRebaselineRequest struct {
	Version           int      `json:"version,omitempty"`
	Operation         string   `json:"operation,omitempty"`
	RequesterRef      string   `json:"requester_ref"`
	Digest            string   `json:"canonical_terms_sha256"`
	Deadline          string   `json:"deadline,omitempty"`
	CandidateSHA      string   `json:"candidate_sha256,omitempty"`
	ServingSHA        string   `json:"serving_sha256,omitempty"`
	WatchdogSHA       string   `json:"watchdog_sha256,omitempty"`
	BundleSHA         string   `json:"bundle_sha256,omitempty"`
	CandidateContract string   `json:"candidate_contract_sha256,omitempty"`
	ServingContract   string   `json:"serving_contract_sha256,omitempty"`
	Changes           []string `json:"changes,omitempty"`
}

// RootRebaselineHandler owns the presentation and reads back the exact immutable
// decision. A service credential can request a decision; it cannot answer one.
func RootRebaselineHandler(store rootRebaselineStore, verifier authn.TokenVerifier, publish bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if store == nil || verifier == nil {
			http.Error(w, "decision service unavailable", http.StatusServiceUnavailable)
			return
		}
		principal, err := providerDecisionPrincipal(r, verifier)
		if err != nil || !principal.Verified || principal.Kind != apiidentity.ActorService || principal.Subject != RootRebaselineSubject || principal.Audience != RootRebaselineAudience || !slices.Contains(principal.Scopes, RootRebaselineScope) {
			http.Error(w, "service identity required", http.StatusUnauthorized)
			return
		}
		var in rootRebaselineRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil {
			http.Error(w, "invalid decision request", http.StatusBadRequest)
			return
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) || strings.TrimSpace(in.RequesterRef) != in.RequesterRef || in.RequesterRef == "" || len(in.RequesterRef) > 128 || !validProviderDecisionDigest(in.Digest) {
			http.Error(w, "invalid decision request", http.StatusBadRequest)
			return
		}
		if publish {
			deadline, err := time.Parse(time.RFC3339, in.Deadline)
			now := time.Now()
			if err != nil || in.Version != 2 || (in.Operation != "contract_change" && in.Operation != "continuation") || (in.Operation == "contract_change") == (in.CandidateContract == in.ServingContract) || !deadline.After(now) || deadline.After(now.Add(24*time.Hour)) || !validProviderDecisionDigest(in.CandidateSHA) || !validProviderDecisionDigest(in.ServingSHA) || !validProviderDecisionDigest(in.WatchdogSHA) || !validProviderDecisionDigest(in.BundleSHA) || !validProviderDecisionDigest(in.CandidateContract) || !validProviderDecisionDigest(in.ServingContract) || rootrebaselinedecision.ValidateChanges(in.Changes) != nil {
				http.Error(w, "invalid decision terms", http.StatusBadRequest)
				return
			}
			digest, err := rootrebaselinedecision.Digest(rootrebaselinedecision.Terms{Version: in.Version, Operation: in.Operation, RequesterRef: in.RequesterRef, Deadline: in.Deadline, CandidateSHA: in.CandidateSHA, ServingSHA: in.ServingSHA, WatchdogSHA: in.WatchdogSHA, BundleSHA: in.BundleSHA, CandidateContract: in.CandidateContract, ServingContract: in.ServingContract, Changes: in.Changes})
			if err != nil || digest != in.Digest {
				http.Error(w, "decision terms digest mismatch", http.StatusBadRequest)
				return
			}
			recipient := strings.TrimSpace(store.ResolveRecipient(r.Context()))
			if recipient == "" {
				http.Error(w, "recipient unavailable", http.StatusServiceUnavailable)
				return
			}
			contextText := "The candidate changes the persistent-state contract."
			if in.Operation == "continuation" {
				contextText = "An interrupted rebaseline needs a rebuilt candidate; its persistent-state contract is unchanged."
			}
			spec := hub.AskSpec{Recipient: recipient, Title: "Root control plane rebaseline", Body: "Review a root control plane transition.", Question: "Allow this root control plane build to become the new baseline?", Options: []hub.AskOption{{Key: "approve", Label: "Approve — install this build as the new baseline"}, {Key: "reject", Label: "Reject — keep the current baseline"}}, Recommended: "reject", RecommendationReason: "Approve only after reviewing the candidate and persistent-state changes.", BigPicture: contextText + " Approving permits the transition without automatic rollback; rejecting or not answering leaves the current baseline in place. After answering, the agent rechecks the qualified images and resumes the transition. A changed build or expired decision needs a new answer. Changes:\n- " + strings.Join(in.Changes, "\n- "), TechnicalDetails: fmt.Sprintf("Candidate: %s\nServing: %s\nWatchdog: %s\nPrepared bundle: %s\nCandidate contract: %s\nServing contract: %s\nTerms: %s", in.CandidateSHA, in.ServingSHA, in.WatchdogSHA, in.BundleSHA, in.CandidateContract, in.ServingContract, in.Digest), Urgency: "high", SensitivityLabel: "private", Deadline: deadline, DeadlineSpecified: true, Source: RootRebaselineSubject, SourceEventType: RootRebaselineEventType, IdempotencyKey: in.RequesterRef, ProviderDecisionContext: &hub.ProviderDecisionContext{SchemaVersion: 1, CanonicalTermsSHA256: in.Digest, Scope: hub.ProviderDecisionScopeRootRebaseline}}
			ask, _, err := store.AskWithSpec(r.Context(), spec)
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, hub.ErrInvalidArgument) {
					status = http.StatusBadRequest
				}
				if errors.Is(err, hub.ErrAskIdentityConflict) {
					status = http.StatusConflict
				}
				http.Error(w, "decision could not be stored", status)
				return
			}
			writeJSON(w, http.StatusAccepted, map[string]string{"ask_id": ask.ID, "requester_ref": in.RequesterRef})
			return
		}
		if in.Version != 0 || in.Operation != "" || in.Deadline != "" || in.CandidateSHA != "" || in.ServingSHA != "" || in.WatchdogSHA != "" || in.BundleSHA != "" || in.CandidateContract != "" || in.ServingContract != "" || len(in.Changes) != 0 {
			http.Error(w, "invalid lookup", http.StatusBadRequest)
			return
		}
		ask, err := store.GetAskByRequester(r.Context(), RootRebaselineSubject, RootRebaselineEventType, in.RequesterRef)
		if err != nil || ask.Source != RootRebaselineSubject || ask.SourceEventType != RootRebaselineEventType || ask.RequesterRef != in.RequesterRef || ask.ProviderDecisionContext == nil || ask.ProviderDecisionContext.SchemaVersion != 1 || ask.ProviderDecisionContext.Scope != hub.ProviderDecisionScopeRootRebaseline || ask.ProviderDecisionContext.CanonicalTermsSHA256 != in.Digest || len(ask.Options) != 2 || ask.Options[0].Key != "approve" || ask.Options[1].Key != "reject" {
			http.Error(w, "decision not found", http.StatusNotFound)
			return
		}
		consumed, err := store.RootRebaselineConsumedAt(r.Context(), ask.ID)
		if err != nil {
			http.Error(w, "decision service unavailable", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"ask_id": ask.ID, "requester_ref": ask.RequesterRef, "canonical_terms_sha256": in.Digest, "state": ask.State, "answer": ask.Answer, "answered_by_identity_kind": ask.AnsweredByIdentityKind, "answered_by_identity_source": ask.AnsweredByIdentitySource, "created_at": ask.CreatedAt, "answered_at": ask.AnsweredAt, "deadline": ask.Deadline, "consumed_at": consumed})
	})
}
