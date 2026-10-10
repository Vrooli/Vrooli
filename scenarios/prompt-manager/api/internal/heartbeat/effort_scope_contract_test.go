package heartbeat

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"github.com/vrooli/api-core/effortauthority"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamcontract"
	"testing"
	"time"
)

func contractPrivateHTTP(t *testing.T, h http.Handler) (*http.Client, string) {
	t.Helper()
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "disposable contract fixture"}, DNSNames: []string{"fixture.invalid"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	pool := x509.NewCertPool()
	pool.AddCert(parsed)
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	pin := sha256.Sum256(der)
	dial := func(ctx context.Context, _, _ string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			conn := tls.Server(server, &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert})
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			if conn.HandshakeContext(ctx) != nil {
				return
			}
			r, e := http.ReadRequest(bufio.NewReader(conn))
			if e != nil {
				return
			}
			defer r.Body.Close()
			state := conn.ConnectionState()
			r.TLS = &state
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			resp := w.Result()
			defer resp.Body.Close()
			_ = resp.Write(conn)
		}()
		return client, nil
	}
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DialContext: dial, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{cert}}}}, hex.EncodeToString(pin[:])
}

type contractAcceptedOwner struct{ revoked bool }

func (o *contractAcceptedOwner) ReadFiniteAcceptance(ctx context.Context, p effortauthority.Policy) (effortauthority.AcceptanceContract, error) {
	if o.revoked {
		return effortauthority.AcceptanceContract{}, effortauthority.ErrRefused
	}
	return effortauthority.AcceptanceContract{Subject: effortauthority.CommissionSubjectFor(p), Actor: p.Owner, AcceptedAt: p.NotBefore, Generation: 1}, nil
}

type contractProfileOwner struct{ digest string }

func (o *contractProfileOwner) ReadFiniteProfileContract(context.Context, string) (effortauthority.ProfileContract, error) {
	return effortauthority.ProfileContract{ID: "fixture-profile", Key: "qualified-profile", Digest: o.digest}, nil
}
func TestContractsPMCommissionReadsDisabledTargetButAdmissionRefuses(t *testing.T) {
	ctx := context.Background()
	f, authority, _, p, _, _ := finiteAuthorityFixture(t, false)
	p.ID = "readonly-startup-fixture"
	if e := f.teams.Update(ctx, "committee", &store.Team{OperatingContract: teamcontract.Minimal("", "lead"), Enabled: false, EnabledSet: true}); e != nil {
		t.Fatal(e)
	}
	cfg, e := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
	if e != nil {
		t.Fatal(e)
	}
	cfg.Enabled = false
	if e = f.teams.SetHeartbeatConfig(ctx, "committee", "lead", cfg); e != nil {
		t.Fatal(e)
	}
	team, e := f.teams.Get(ctx, "committee")
	if e != nil {
		t.Fatal(e)
	}
	p.TeamDigest = effortauthority.Digest(team.Contract())
	p.BindingDigest = effortauthority.Digest(cfg.FiniteLeader)
	b := effortauthority.Binding{PolicyID: p.ID, PolicyDigest: effortauthority.Digest(p), Epoch: p.Epoch, Owner: p.Owner, Client: p.Client, Deadline: p.Deadline}
	acceptedOwner := &contractAcceptedOwner{}
	var acceptedHandler *effortauthority.AcceptanceContractHandler
	acceptanceHTTP, acceptancePin := contractPrivateHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { acceptedHandler.ServeHTTP(w, r) }))
	acceptedHandler, e = effortauthority.NewAcceptanceContractHandler(acceptedOwner, map[string]map[string]effortauthority.AcceptanceContractGrant{acceptancePin: {p.ID: {Policy: p}}})
	if e != nil {
		t.Fatal(e)
	}
	accepted, e := effortauthority.NewAcceptanceContractClient(effortauthority.AcceptanceContractClientConfig{URL: "https://fixture.invalid" + effortauthority.AcceptanceContractEndpoint, HTTP: acceptanceHTTP, ServerPin: acceptancePin, Binding: b, Policy: p})
	if e != nil {
		t.Fatal(e)
	}
	profileOwner := &contractProfileOwner{p.Profiles["qualified-profile"]}
	var profileHandler *effortauthority.ProfileContractHandler
	profileHTTP, profilePin := contractPrivateHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { profileHandler.ServeHTTP(w, r) }))
	profileHandler, e = effortauthority.NewProfileContractHandler(profileOwner, map[string]map[string]effortauthority.ProfileContractGrant{profilePin: {p.ID: {Binding: b, Profiles: map[string]string{"qualified-profile": "fixture-profile"}}}})
	if e != nil {
		t.Fatal(e)
	}
	profiles, e := effortauthority.NewProfileContractClient(effortauthority.ProfileContractClientConfig{URL: "https://fixture.invalid" + effortauthority.ProfileContractEndpoint, HTTP: profileHTTP, ServerPin: profilePin, Binding: b, Profiles: map[string]string{"qualified-profile": "fixture-profile"}})
	if e != nil {
		t.Fatal(e)
	}
	scope := &FiniteEffortScope{Teams: f.teams, Accepted: accepted, Profiles: profiles}
	var commissioningHandler, admissionHandler *effortauthority.ScopeContractHandler
	scopeHTTP, scopePin := contractPrivateHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == effortauthority.ScopeContractEndpoint {
			admissionHandler.ServeHTTP(w, r)
		} else {
			commissioningHandler.ServeHTTP(w, r)
		}
	}))
	peers := map[string]map[string]effortauthority.AcceptanceContractGrant{scopePin: {p.ID: {Policy: p}}}
	commissioningHandler, admissionHandler, e = PrepareFiniteScopeReadContracts(ctx, p, scope, peers)
	if e != nil {
		t.Fatal("disabled authored commissioning", e)
	}
	if _, e = PrepareFiniteScopeContract(ctx, p, scope, peers); e == nil {
		t.Fatal("disabled admission published")
	}
	commission, e := effortauthority.NewCommissionScopeContractClient(effortauthority.AcceptanceContractClientConfig{URL: "https://fixture.invalid" + effortauthority.CommissionScopeEndpoint, HTTP: scopeHTTP, ServerPin: scopePin, Binding: b, Policy: p})
	if e != nil {
		t.Fatal(e)
	}
	admission, e := effortauthority.NewScopeContractClient(effortauthority.AcceptanceContractClientConfig{URL: "https://fixture.invalid" + effortauthority.ScopeContractEndpoint, HTTP: scopeHTTP, ServerPin: scopePin, Binding: b, Policy: p})
	if e != nil {
		t.Fatal(e)
	}
	authority.Scope = admission
	authority.ApprovalScope = commission
	if e = authority.Approve(ctx, "disposable-authenticated-human", p); e != nil {
		t.Fatal("consent against exact disabled authoredtarget", e)
	}
	if _, e = authority.CheckBinding(ctx, b); e == nil {
		t.Fatal("approved policy bypassed disabled target")
	}
	beforeState, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	scope.Accepted = nil
	scope.Profiles = nil // published provider references are frozen.
	if e = commission.CheckEffortScope(ctx, p); e != nil {
		t.Fatal("current full private commissioning", e)
	}
	if commission.CheckInstallation(p) == nil || commission.CheckApprovalInstallation(p) != nil {
		t.Fatal("commission reader became admission authority")
	}
	// Enabling only this disposable fixture preserves authored digests; live
	// admission then reads the same approved current binding, without a new grant.
	if e = f.teams.Update(ctx, "committee", &store.Team{Enabled: true, EnabledSet: true}); e != nil {
		t.Fatal(e)
	}
	cfg.Enabled = true
	if e = f.teams.SetHeartbeatConfig(ctx, "committee", "lead", cfg); e != nil {
		t.Fatal(e)
	}
	if _, e = authority.CheckBinding(ctx, b); e != nil {
		t.Fatal("positive staged startup order", e)
	}
	acceptedOwner.revoked = true
	if commission.CheckEffortScope(ctx, p) == nil {
		t.Fatal("revoked commission authorized")
	}
	acceptedOwner.revoked = false
	profileOwner.digest = effortauthority.Digest("drift")
	if commission.CheckEffortScope(ctx, p) == nil {
		t.Fatal("profile drift authorized")
	}
	profileOwner.digest = p.Profiles["qualified-profile"]
	afterState, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if effortauthority.Digest(beforeState) != effortauthority.Digest(afterState) || f.queue.enqueues != 0 || len(f.agent.createTaskCalls) != 0 || len(f.agent.createRunCalls) != 0 {
		t.Fatal("read composition activated work")
	}
}
