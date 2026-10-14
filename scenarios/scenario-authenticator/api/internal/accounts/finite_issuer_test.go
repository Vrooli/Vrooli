package accounts

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/effortauthority"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"
)

func TestFiniteIssuerSameUIDGuardPrecedesActualHumanApprovalEffects(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	account, e := repo.Create(ctx, CreateInput{RealmID: "default", Email: "isolation-fixture@example.test", PasswordHash: "unused-disposable-hash"})
	if e != nil {
		t.Fatal(e)
	}
	scopes := []string{"agent-manager:write", "agent-manager:orchestrate"}
	grants := authorization.NewService(repo.(authorization.ScopeStore), nil)
	for _, scope := range scopes {
		if _, e = grants.Grant(ctx, account.ID, scope, authorization.Meta{}); e != nil {
			t.Fatal(e)
		}
	}
	keys, e := authcrypto.LoadOrGenerate(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour})
	service := NewService(ServiceConfig{Repo: repo, Signer: signer, Sessions: sessions.NewManager(redisstate.NewMemory(), nil), Authorization: grants, Clock: schedule.System()})
	token, e := signer.Sign(authcrypto.TokenInput{UserID: account.ID, Audience: "scenario-authenticator:default", Scopes: scopes})
	if e != nil {
		t.Fatal(e)
	}
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC()
	root := t.TempDir()
	digest := effortauthority.Digest("disposable exact owner scope")
	p := effortauthority.Policy{ID: "guarded-fixture", Owner: account.ID, Client: "pm-fixture", ClientKey: pub, Epoch: 1, Repository: root, Effort: "fixture", Revision: "1", ContentDigest: digest, Team: "fixture", TeamDigest: digest, BindingDigest: digest, Members: []string{"lead"}, Profiles: map[string]string{"existing": digest}, Scopes: scopes, Effects: []string{"run.create"}, NotBefore: now.Add(-time.Minute), Deadline: now.Add(time.Hour), MaxStarts: 1, MaxConcurrent: 1, MaxTurns: 1, MaxToolCalls: 1, MaxRunSeconds: 60, TotalTurns: 1, TotalToolCalls: 1, TotalRunSeconds: 60, SurviveLogout: true}
	path := filepath.Join(root, "ledger.db")
	db, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	ledger := &effortauthority.SQLStore{DB: db}
	if e = ledger.Ensure(ctx); e != nil {
		t.Fatal(e)
	}
	binary, e := os.ReadFile("/proc/self/exe")
	if e != nil {
		t.Fatal(e)
	}
	hash := sha256.Sum256(binary)
	process := effortauthority.ProtectedProcess{PID: os.Getpid(), ExecutableSHA256: hex.EncodeToString(hash[:])}
	setup := FiniteIssuerSetup{Store: ledger, Scope: joinedAcceptedScope{p}, Custody: &effortauthority.CustodySigner{Clients: map[string]crypto.Signer{p.Client: key}}, Peers: map[string]map[string]effortauthority.BrokerGrant{"fixture-pin": {p.ID: {PolicyID: p.ID, Operations: map[string]bool{"sign": true}}}}, Proposals: map[string]effortauthority.Policy{p.ID: p}, Origin: "https://issuer.example"}
	if _, e = service.FiniteIssuer(setup); e == nil {
		t.Fatal("absent isolation admitted source setup")
	}
	setup.Isolation = &effortauthority.NativeUIDIsolation{NativeUID: uint32(os.Getuid()), NativeProbePID: os.Getpid(), Services: map[string]effortauthority.ProtectedProcess{"issuer": process, "prompt-manager": process, "agent-manager": process}, ProtectedPaths: []string{root, path}}
	handlers, e := service.FiniteIssuer(setup)
	if e != nil {
		t.Fatal(e)
	}
	form := url.Values{"id": {p.ID}, "digest": {effortauthority.Digest(p)}, "action": {"approve"}, "surviveLogout": {"yes"}}
	request := httptest.NewRequest("POST", "https://issuer.example"+effortauthority.ConsentEndpoint, strings.NewReader(form.Encode()))
	request.Header.Set("Origin", setup.Origin)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: "vrooli_access_token", Value: token})
	result := httptest.NewRecorder()
	handlers.Consent.ServeHTTP(result, request)
	if result.Code != 403 || !strings.Contains(result.Body.String(), "isolation unqualified") {
		t.Fatal("sameUID guard did not precede valid human approval", result.Code, result.Body.String())
	}
	if _, e = ledger.Get(ctx, p.ID); e == nil {
		t.Fatal("unqualified isolation wrote finite policy")
	}
	// The human/current-scope offer itself is valid: isolation is the precise refusal.
	authority := &effortauthority.Authority{Store: ledger, Owners: service, Approver: service, Scope: joinedAcceptedScope{p}}
	if e = authority.Approve(ctx, token, p); e != nil {
		t.Fatal("fixture did not offer valid self approval", e)
	}
}
