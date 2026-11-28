package conversations

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	apiidentity "github.com/vrooli/api-core/identity"
	conversation "notification-hub/internal/conversations"
	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
	"notification-hub/internal/testutil/hubtest"
)

type enrollmentProvider struct{ subject string }

func (p enrollmentProvider) Source() apiidentity.AuthSource {
	return apiidentity.SourceCloudflareAccess
}
func (p enrollmentProvider) VerifyRequest(context.Context, *http.Request) (apiidentity.Principal, error) {
	return apiidentity.Principal{Kind: apiidentity.ActorHuman, Subject: p.subject, Verified: true, Source: p.Source()}, nil
}

func TestEnrollmentRequiresBothProofsAndCurrentPolicy(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(conversation.Schema)); err != nil {
		t.Fatal(err)
	}
	svc := hubtest.New(t, database.NewFromPrimary(db), nil, nil)
	svc.SetDefaultRecipient("owner")
	svc.SetAppEligibility(func(context.Context, string) (string, error) { return "https://hub.example", nil })
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"cloudflare_access","subject":"cf","canonical_subject":"owner","scopes":["root:rebaseline"]}]`)
	login := &fakeRootLogin{}
	h := &handler{service: svc, stepUp: login}
	serve := func(subject, origin, body string) *httptest.ResponseRecorder {
		auth := identity.OwnerAuthenticator{Providers: []authn.Provider{enrollmentProvider{subject}}, OwnerSubject: func(context.Context) string { return "owner" }}
		r := httptest.NewRequest("POST", "https://hub.example/api/v1/identity/root-enrollment", strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		auth.Middleware(http.HandlerFunc(h.rootEnrollment)).ServeHTTP(w, r)
		return w
	}
	body := `{"email":"a@example.test","password":"private-password"}`
	for _, tc := range []struct{ subject, origin string }{{"other", "https://hub.example"}, {"cf", "https://evil.example"}, {"", "https://hub.example"}} {
		if w := serve(tc.subject, tc.origin, body); w.Code == 200 {
			t.Fatalf("untrusted enrollment accepted: %+v", tc)
		}
	}
	login.fail = true
	if w := serve("cf", "https://hub.example", body); w.Code != 401 || strings.Contains(w.Body.String(), "private-password") {
		t.Fatalf("bad proof: %d %s", w.Code, w.Body.String())
	}
	login.fail = false
	login.mfa = true
	w := serve("cf", "https://hub.example", body)
	if w.Code != 200 {
		t.Fatalf("MFA challenge: %d %s", w.Code, w.Body.String())
	}
	var challenge struct {
		MFAChallenge string `json:"mfaChallenge"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &challenge); err != nil || challenge.MFAChallenge == "" {
		t.Fatal("missing challenge")
	}
	if err := svc.RevokeRootEnrollments(ctx, "owner", "cf"); err != nil {
		t.Fatal(err)
	}
	login.mfa = false
	completion, _ := json.Marshal(map[string]string{"email": "a@example.test", "password": "private-password", "mfaChallenge": challenge.MFAChallenge, "totpCode": "123456"})
	if w := serve("cf", "https://hub.example", string(completion)); w.Code == 200 {
		t.Fatal("revoked MFA challenge enrolled account")
	}
	bindingsBefore, _ := identity.OwnerBindings()
	if enrolled(t, h, ctx, bindingsBefore, "account", "owner") {
		t.Fatal("MFA challenge enrolled account")
	}
	login.mfa = false
	if w := serve("cf", "https://hub.example", body); w.Code != 200 {
		t.Fatalf("enrollment: %d %s", w.Code, w.Body.String())
	}
	bindings, err := identity.OwnerBindings()
	if err != nil {
		t.Fatal(err)
	}
	if !enrolled(t, h, ctx, bindings, "account", "owner") || enrolled(t, h, ctx, bindings, "other", "owner") {
		t.Fatal("association scope incorrect")
	}
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[]`)
	revokedPolicy, _ := identity.OwnerBindings()
	if enrolled(t, h, ctx, revokedPolicy, "account", "owner") {
		t.Fatal("revoked policy retained authority")
	}
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"cloudflare_access","subject":"cf","canonical_subject":"owner","scopes":["root:rebaseline"]}]`)
	bindings, _ = identity.OwnerBindings()
	generation, err := svc.RootEnrollmentGeneration(ctx, "owner", "cf")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RevokeRootEnrollments(ctx, "owner", "cf"); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnrollRootAccount(ctx, "owner", "cf", "account", generation); err == nil {
		t.Fatal("in-flight enrollment survived revocation")
	}
	if enrolled(t, h, ctx, bindings, "account", "owner") {
		t.Fatal("revoked enrollment retained authority")
	}
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[]`)
	bindings, _ = identity.OwnerBindings()
	if enrolled(t, h, ctx, bindings, "account", "owner") {
		t.Fatal("revoked owner retained authority")
	}
}

func enrolled(t *testing.T, h *handler, ctx context.Context, bindings []authn.OwnerBinding, account, recipient string) bool {
	t.Helper()
	ok, err := h.checkRootPasswordAllowed(ctx, bindings, account, recipient)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestChallengeConcealsOwnerSnapshotAndRejectsTampering(t *testing.T) {
	token, err := wrapChallengeWithGrants("mfa-secret", "private-owner", "recipient", 7, []hub.RootEnrollmentGrant{{Owner: "private-cloudflare-subject", Generation: 4}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private-cloudflare-subject") || strings.Contains(string(raw), "mfa-secret") {
		t.Fatal("challenge disclosed private data")
	}
	raw[len(raw)-1] ^= 1
	if _, err := unwrapChallengeWithGrants(base64.RawURLEncoding.EncodeToString(raw), "private-owner", "recipient"); err == nil {
		t.Fatal("tampered challenge accepted")
	}
}
