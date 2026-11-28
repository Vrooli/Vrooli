package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/rootrebaselinedecision"
	"notification-hub/internal/hub"
)

type rootStore struct {
	spec hub.AskSpec
	ask  hub.AskRecord
}

func (s *rootStore) ResolveRecipient(context.Context) string { return "owner" }
func (s *rootStore) AskWithSpec(_ context.Context, spec hub.AskSpec) (hub.AskRecord, hub.Notification, error) {
	s.spec = spec
	return hub.AskRecord{ID: "ask-1"}, hub.Notification{}, nil
}
func (s *rootStore) GetAskByRequester(context.Context, string, string, string) (hub.AskRecord, error) {
	return s.ask, nil
}
func (s *rootStore) RootRebaselineConsumedAt(context.Context, string) (string, error) { return "", nil }

type rootVerifier struct{ p apiidentity.Principal }

func (v rootVerifier) Verify(context.Context, string) (apiidentity.Principal, error) { return v.p, nil }
func rootPrincipal() apiidentity.Principal {
	return apiidentity.Principal{Verified: true, Kind: apiidentity.ActorService, Subject: RootRebaselineSubject, Audience: RootRebaselineAudience, Scopes: []string{RootRebaselineScope}}
}
func rootTestDigest(in rootRebaselineRequest) string {
	digest, _ := rootrebaselinedecision.Digest(rootrebaselinedecision.Terms{Version: in.Version, Operation: in.Operation, RequesterRef: in.RequesterRef, Deadline: in.Deadline, CandidateSHA: in.CandidateSHA, ServingSHA: in.ServingSHA, WatchdogSHA: in.WatchdogSHA, BundleSHA: in.BundleSHA, CandidateContract: in.CandidateContract, ServingContract: in.ServingContract, Changes: in.Changes})
	return digest
}
func rootCall(store *rootStore, p apiidentity.Principal, publish bool, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	RootRebaselineHandler(store, rootVerifier{p}, publish).ServeHTTP(w, r)
	return w
}
func TestRootRebaselinePublishOwnsCopyAndRequiresIdentity(t *testing.T) {
	digest := strings.Repeat("a", 64)
	req := rootRebaselineRequest{Version: 2, Operation: "contract_change", RequesterRef: "request-1", Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), CandidateSHA: digest, ServingSHA: digest, WatchdogSHA: digest, BundleSHA: digest, CandidateContract: strings.Repeat("b", 64), ServingContract: digest, Changes: []string{"State format changed"}}
	req.Digest = rootTestDigest(req)
	b, _ := json.Marshal(req)
	store := &rootStore{}
	for name, mutate := range map[string]func(*apiidentity.Principal){"subject": func(p *apiidentity.Principal) { p.Subject = "other" }, "audience": func(p *apiidentity.Principal) { p.Audience = "other" }, "scope": func(p *apiidentity.Principal) { p.Scopes = nil }, "wildcard": func(p *apiidentity.Principal) { p.Scopes = []string{"notification-hub:*"} }, "human": func(p *apiidentity.Principal) { p.Kind = apiidentity.ActorHuman }, "unverified": func(p *apiidentity.Principal) { p.Verified = false }} {
		t.Run(name, func(t *testing.T) {
			p := rootPrincipal()
			mutate(&p)
			if w := rootCall(store, p, true, string(b)); w.Code != http.StatusUnauthorized {
				t.Fatal(w.Code)
			}
		})
	}
	w := rootCall(store, rootPrincipal(), true, string(b))
	if w.Code != http.StatusAccepted {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if store.spec.Source != RootRebaselineSubject || store.spec.ProviderDecisionContext.CanonicalTermsSHA256 != req.Digest || !strings.Contains(store.spec.TechnicalDetails, "Watchdog: "+digest) || len(store.spec.Options) != 2 || store.spec.Options[0].Key != "approve" || store.spec.Options[1].Key != "reject" || store.spec.DefaultAnswer != "" {
		t.Fatalf("bad spec: %+v", store.spec)
	}
	if w := rootCall(store, rootPrincipal(), true, strings.TrimSuffix(string(b), "}")+`,"question":"forged"}`); w.Code != http.StatusBadRequest {
		t.Fatal(w.Code)
	}
	req.WatchdogSHA = strings.Repeat("b", 64)
	b, _ = json.Marshal(req)
	if w := rootCall(store, rootPrincipal(), true, string(b)); w.Code != http.StatusBadRequest {
		t.Fatalf("changed watchdog accepted: %d", w.Code)
	}
}
func TestRootRebaselineLookupBindsExactRequest(t *testing.T) {
	digest := strings.Repeat("a", 64)
	s := &rootStore{ask: hub.AskRecord{ID: "ask-1", Source: RootRebaselineSubject, SourceEventType: RootRebaselineEventType, RequesterRef: "req", Options: []hub.AskOption{{Key: "approve"}, {Key: "reject"}}, ProviderDecisionContext: &hub.ProviderDecisionContext{SchemaVersion: 1, CanonicalTermsSHA256: digest, Scope: hub.ProviderDecisionScopeRootRebaseline}}}
	body := `{"requester_ref":"req","canonical_terms_sha256":"` + digest + `"}`
	if w := rootCall(s, rootPrincipal(), false, body); w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	s.ask.ProviderDecisionContext.CanonicalTermsSHA256 = strings.Repeat("b", 64)
	if w := rootCall(s, rootPrincipal(), false, body); w.Code != http.StatusNotFound {
		t.Fatal(w.Code)
	}
}

func TestRootRebaselineContinuationPresentsUnchangedContract(t *testing.T) {
	sha := strings.Repeat("a", 64)
	req := rootRebaselineRequest{Version: 2, Operation: "continuation", RequesterRef: "continuation-1", Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), CandidateSHA: sha, ServingSHA: sha, WatchdogSHA: sha, BundleSHA: sha, CandidateContract: sha, ServingContract: sha, Changes: []string{"Continue interrupted transition with a rebuilt root."}}
	req.Digest = rootTestDigest(req)
	b, _ := json.Marshal(req)
	store := &rootStore{}
	if got := rootCall(store, rootPrincipal(), true, string(b)); got.Code != http.StatusAccepted {
		t.Fatalf("continuation rejected: %d %s", got.Code, got.Body.String())
	}
	if !strings.Contains(store.spec.BigPicture, "contract is unchanged") {
		t.Fatalf("misleading decision: %s", store.spec.BigPicture)
	}
	req.CandidateContract = strings.Repeat("b", 64)
	req.Digest = rootTestDigest(req)
	b, _ = json.Marshal(req)
	if got := rootCall(store, rootPrincipal(), true, string(b)); got.Code != http.StatusBadRequest {
		t.Fatalf("different contract accepted as continuation: %d", got.Code)
	}
}
