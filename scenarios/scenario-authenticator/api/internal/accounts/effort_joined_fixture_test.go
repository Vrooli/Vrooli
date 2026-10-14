package accounts

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"
	"strings"
	"testing"
	"time"
)

type joinedAcceptedScope struct{ policy effortauthority.Policy }

func (s joinedAcceptedScope) CheckEffortScope(_ context.Context, p effortauthority.Policy) error {
	if effortauthority.Digest(p) != effortauthority.Digest(s.policy) {
		return effortauthority.ErrRefused
	}
	return nil
}

// Temporary account, human session, grant issuer and mTLS current-owner server.
func TestFiniteJoinedOwnerHTTPFixture(t *testing.T) {
	root := os.Getenv("VROOLI_DISPOSABLE_FINITE_JOINED_ROOT")
	if root == "" {
		t.Skip("joined disposable helper")
	}
	ctx := context.Background()
	repo, _ := newRepo(t)
	acc, e := repo.Create(ctx, CreateInput{RealmID: "default", Email: "joined@example.test", PasswordHash: "unused-fixture"})
	if e != nil {
		t.Fatal(e)
	}
	grants := authorization.NewService(repo.(authorization.ScopeStore), nil)
	for _, scope := range []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"} {
		if _, e = grants.Grant(ctx, acc.ID, scope, authorization.Meta{}); e != nil {
			t.Fatal(e)
		}
	}
	keys, e := authcrypto.LoadOrGenerate(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour})
	sessions := sessions.NewManager(redisstate.NewMemory(), nil)
	service := NewService(ServiceConfig{Repo: repo, Signer: signer, Sessions: sessions, Authorization: grants, Clock: schedule.System()})
	raw, e := os.ReadFile(filepath.Join(root, "policy-input.json"))
	if e != nil {
		t.Fatal(e)
	}
	var p effortauthority.Policy
	if json.Unmarshal(raw, &p) != nil {
		t.Fatal("policy input")
	}
	p.Owner = acc.ID
	db, e := sql.Open("sqlite", filepath.Join(root, "ledger.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	store := &effortauthority.SQLStore{DB: db}
	if e = store.Ensure(ctx); e != nil {
		t.Fatal(e)
	}
	now := func() time.Time {
		b, e := os.ReadFile(filepath.Join(root, "clock.json"))
		if e != nil {
			t.Fatal(e)
		}
		var n time.Time
		if json.Unmarshal(b, &n) != nil {
			t.Fatal("clock")
		}
		return n
	}
	authority := &effortauthority.Authority{Store: store, Owners: service, Approver: service, Scope: joinedAcceptedScope{p}, Now: now}
	token, e := signer.Sign(authcrypto.TokenInput{UserID: acc.ID, Audience: "scenario-authenticator:default", Scopes: p.Scopes})
	if e != nil {
		t.Fatal(e)
	}
	consent := &effortauthority.ConsentHandler{Authority: authority, Proposals: map[string]effortauthority.Policy{p.ID: p}, Origin: "https://issuer.example", CookieName: "vrooli_access_token"}
	preview := httptest.NewRequest("GET", "https://issuer.example"+effortauthority.ConsentEndpoint+"?id="+p.ID, nil)
	preview.AddCookie(&http.Cookie{Name: "vrooli_access_token", Value: token})
	shown := httptest.NewRecorder()
	consent.ServeHTTP(shown, preview)
	if shown.Code != 200 || !strings.Contains(shown.Body.String(), effortauthority.Digest(p)) || strings.Contains(shown.Body.String(), token) {
		t.Fatal("actual provider exact consent preview failed")
	}
	form := url.Values{"id": {p.ID}, "digest": {effortauthority.Digest(p)}, "action": {"approve"}, "surviveLogout": {"yes"}}
	post := httptest.NewRequest("POST", "https://issuer.example"+effortauthority.ConsentEndpoint, strings.NewReader(form.Encode()))
	post.AddCookie(&http.Cookie{Name: "vrooli_access_token", Value: token})
	post.Header.Set("Origin", "https://issuer.example")
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	accepted := httptest.NewRecorder()
	consent.ServeHTTP(accepted, post)
	if accepted.Code != 200 {
		t.Fatal("actual provider human consent failed", accepted.Body.String())
	}
	// Session logout cannot approve again; the explicitly finite grant survives.
	if e = sessions.BlacklistAccess(ctx, token, time.Now().Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	if _, e = service.ApproveEffortOwner(ctx, token); e == nil {
		t.Fatal("logout did not close human approval")
	}
	cert, e := tls.LoadX509KeyPair(filepath.Join(root, "peer.crt"), filepath.Join(root, "peer.key"))
	if e != nil {
		t.Fatal(e)
	}
	pem, e := os.ReadFile(filepath.Join(root, "peer.crt"))
	if e != nil {
		t.Fatal(e)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		t.Fatal("fixture trust")
	}
	sum := sha256.Sum256(cert.Certificate[0])
	pin := hex.EncodeToString(sum[:])
	handler := &effortauthority.OwnerReadHandler{Store: store, Owners: service, Peers: map[string]map[string]bool{pin: {p.ID: true}}, Now: now}
	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
	server.StartTLS()
	defer server.Close()
	meta, _ := json.Marshal(struct {
		URL, Pin string
		Policy   effortauthority.Policy
	}{server.URL + effortauthority.OwnerReadEndpoint, pin, p})
	if e = os.WriteFile(filepath.Join(root, "owner.json"), meta, 0600); e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(45 * time.Second)
	revoked := false
	for time.Now().Before(deadline) {
		if _, e = os.Stat(filepath.Join(root, "done")); e == nil {
			return
		}
		if !revoked {
			if _, e = os.Stat(filepath.Join(root, "revoke-current-grant")); e == nil {
				if _, e = grants.Revoke(ctx, acc.ID, "agent-manager:orchestrate", authorization.Meta{}); e != nil {
					t.Fatal(e)
				}
				revoked = true
				os.WriteFile(filepath.Join(root, "grant-revoked"), []byte("done"), 0600)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("joined parent exceeded fixture deadline")
}
