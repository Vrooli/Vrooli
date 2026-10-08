package heartbeat

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	"math/big"
	_ "modernc.org/sqlite"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamcontract"
	"testing"
	"time"
)

func joinedRead(t *testing.T, path string, out any) {
	t.Helper()
	until := time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		raw, e := os.ReadFile(path)
		if e == nil {
			if out == nil {
				return
			}
			if json.Unmarshal(raw, out) == nil {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("bounded disposable fixture readiness failed: %s", filepath.Base(path))
}
func joinedWrite(t *testing.T, path string, value any) {
	t.Helper()
	raw, e := json.Marshal(value)
	if e != nil {
		t.Fatal(e)
	}
	temp := path + ".tmp"
	if e = os.WriteFile(temp, raw, 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(temp, path); e != nil {
		t.Fatal(e)
	}
}
func joinedPeer(t *testing.T, root string) (tls.Certificate, *x509.CertPool, string) {
	t.Helper()
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "disposable joined finite peer"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, e := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := x509.MarshalPKCS8PrivateKey(key)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(root, "peer.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600)
	os.WriteFile(filepath.Join(root, "peer.key"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}), 0600)
	parsed, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	pool := x509.NewCertPool()
	pool.AddCert(parsed)
	sum := sha256.Sum256(der)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool, hex.EncodeToString(sum[:])
}
func joinedHelper(t *testing.T, root, scenario, test string) {
	t.Helper()
	cwd, e := os.Getwd()
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command("go", "test", "-count=1", "-timeout=50s", "./internal/"+map[string]string{"agent-manager": "handlers", "scenario-authenticator": "accounts"}[scenario], "-run", "^"+test+"$")
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatal("module root missing")
		}
		cwd = parent
	}
	cmd.Dir = filepath.Join(cwd, "..", "..", scenario, "api")
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_DISPOSABLE_FINITE_JOINED_ROOT="+root)
	output, e := os.Create(filepath.Join(root, scenario+".log"))
	if e != nil {
		t.Fatal(e)
	}
	cmd.Stdout = output
	cmd.Stderr = output
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait(); output.Close() }()
	t.Cleanup(func() {
		os.WriteFile(filepath.Join(root, "done"), []byte("done"), 0600)
		select {
		case e := <-done:
			if e != nil {
				b, _ := os.ReadFile(filepath.Join(root, scenario+".log"))
				t.Errorf("disposable %s helper failed: %v\n%s", scenario, e, b)
			}
		case <-time.After(5 * time.Second):
			cmd.Process.Kill()
			<-done
			t.Error("disposable helper did not finish")
		}
	})
}

// The fixture's approved commission is an issuer-protected record, not a
// task label or WorkReference. This does not accept any live Planner review.
type joinedAcceptedReader struct {
	ledger   effortauthority.Store
	policyID string
}

func (r joinedAcceptedReader) CheckAcceptedEffort(ctx context.Context, owner, effort, revision, digest string) error {
	record, e := r.ledger.Get(ctx, r.policyID)
	if e != nil || record.Revoked || record.Policy.Owner != owner || record.Policy.Effort != effort || record.Policy.Revision != revision || record.Policy.ContentDigest != digest {
		return effortauthority.ErrRefused
	}
	return nil
}

type joinedNativeProfileReader struct{ client *AgentManagerClient }

func (r joinedNativeProfileReader) CheckNativeProfile(ctx context.Context, key, digest string) error {
	if key != "qualified-profile" {
		return effortauthority.ErrRefused
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, r.client.testBaseURL+"/fixture/profile-digest", nil)
	if e != nil {
		return effortauthority.ErrRefused
	}
	resp, e := r.client.httpClient.Do(req)
	if e != nil {
		return effortauthority.ErrRefused
	}
	defer resp.Body.Close()
	var actual string
	if resp.StatusCode != 200 || json.NewDecoder(resp.Body).Decode(&actual) != nil || actual != digest {
		return effortauthority.ErrRefused
	}
	return nil
}
func TestFiniteJoinedActualPMNativeAndPrivateOwnerAfterLogout(t *testing.T) {
	runFiniteJoinedActual(t, false)
}
func TestFiniteJoinedPrivateBrokerActualAdaptersAfterLogout(t *testing.T) {
	runFiniteJoinedActual(t, true)
}
func runFiniteJoinedActual(t *testing.T, useBroker bool) {
	ctx := context.Background()
	root := t.TempDir()
	cert, pool, pin := joinedPeer(t, root)
	if useBroker {
		joinedWrite(t, filepath.Join(root, "broker-enabled"), true)
	}
	now := time.Now().UTC().Truncate(time.Second)
	joinedWrite(t, filepath.Join(root, "clock.json"), now)
	joinedHelper(t, root, "agent-manager", "TestFiniteJoinedNativeHTTPFixture")
	var native struct{ URL, ProfileDigest string }
	joinedRead(t, filepath.Join(root, "native.json"), &native)
	f := newFiniteFixtureWithKeepAlive(t, true)
	if e := f.teams.Update(ctx, "committee", &store.Team{OperatingContract: teamcontract.Minimal("", "lead")}); e != nil {
		t.Fatal(e)
	}
	cfg, _ := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
	team, _ := f.teams.Get(ctx, "committee")
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	p := effortauthority.Policy{ID: "joined-protected-policy", Client: "joined-pm-custody", ClientKey: pub, Epoch: 1, Repository: root, Effort: cfg.FiniteLeader.EffortRef, Revision: cfg.FiniteLeader.AcceptedRevision, ContentDigest: effortauthority.Digest("disposable exact accepted commission"), Team: "committee", TeamDigest: effortauthority.Digest(team.Contract()), BindingDigest: effortauthority.Digest(cfg.FiniteLeader), Members: []string{"lead"}, Profiles: map[string]string{"qualified-profile": native.ProfileDigest}, Scopes: []string{"agent-manager:read", "agent-manager:write", "agent-manager:orchestrate"}, Effects: []string{"run.create"}, NotBefore: now, Deadline: now.Add(8 * time.Hour), MaxStarts: 2, MaxConcurrent: 1, MaxTurns: 10, MaxToolCalls: 20, MaxRunSeconds: 3600, TotalTurns: 20, TotalToolCalls: 40, TotalRunSeconds: 7200, SurviveLogout: true}
	joinedWrite(t, filepath.Join(root, "policy-input.json"), p)
	joinedHelper(t, root, "scenario-authenticator", "TestFiniteJoinedOwnerHTTPFixture")
	var owner struct {
		URL, Pin string
		Policy   effortauthority.Policy
	}
	joinedRead(t, filepath.Join(root, "owner.json"), &owner)
	p = owner.Policy
	joinedRead(t, filepath.Join(root, "native-ready"), nil)
	db, e := sql.Open("sqlite", filepath.Join(root, "ledger.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ledger := &effortauthority.SQLStore{DB: db}
	ownerClient := &effortauthority.OwnerReadClient{URL: owner.URL, ServerPin: pin, HTTP: &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{cert}}}}}
	client := &AgentManagerClient{httpClient: &http.Client{Timeout: time.Second}, testBaseURL: native.URL}
	authority := &effortauthority.Authority{Store: ledger, Owners: ownerClient, Scope: &FiniteEffortScope{Teams: f.teams, Accepted: joinedAcceptedReader{ledger, p.ID}, Profiles: joinedNativeProfileReader{client}}, Now: func() time.Time { return now }}
	intent := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: "qualified-profile", ProfileDigest: p.Profiles["qualified-profile"], Effect: "run.create", IdempotencyKey: "setup", InputDigest: effortauthority.Digest("setup"), Turns: 10, ToolCalls: 20, RunSeconds: 3600}
	proof, _ := effortauthority.Sign(effortauthority.Proof{Intent: intent, Nonce: "joined-setup-nonce", IssuedAt: now}, key)
	binding, e := authority.CheckProof(ctx, proof)
	if e != nil {
		t.Fatal(e)
	}
	efforts := &FiniteEffortAuthority{Authority: authority, Signer: &effortauthority.CustodySigner{Clients: map[string]crypto.Signer{p.Client: key}}, Bindings: map[string]effortauthority.Binding{"committee/lead": binding}, Tasks: client, Now: func() time.Time { return now }}
	if useBroker {
		broker := &effortauthority.BrokerHandler{Authority: authority, Custody: &effortauthority.CustodySigner{Clients: map[string]crypto.Signer{p.Client: key}}, Peers: map[string]map[string]effortauthority.BrokerGrant{}}
		pmCert, pmPin := joinedClientPeer(t, cert, "pm", root)
		_, amPin := joinedClientPeer(t, cert, "native", root)
		broker.Peers[pmPin] = map[string]effortauthority.BrokerGrant{p.ID: {PolicyID: p.ID, Operations: map[string]bool{"check-binding": true, "check-proof": true, "sign": true, "prepare": true, "observe-ingress": true}}}
		broker.Peers[amPin] = map[string]effortauthority.BrokerGrant{p.ID: {PolicyID: p.ID, Operations: map[string]bool{"check-binding": true, "check-proof": true, "reserve": true, "accept": true, "settle": true, "read-reservation": true}}}
		protected := httptest.NewUnstartedServer(broker)
		protected.TLS = &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
		protected.StartTLS()
		defer protected.Close()
		remote := &effortauthority.BrokerClient{URL: protected.URL + effortauthority.BrokerEndpoint, ServerPin: pin, Bindings: map[string]effortauthority.Binding{p.ID: binding}, HTTP: &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{pmCert}}}}}
		efforts.Authority = remote
		efforts.Signer = remote
		joinedWrite(t, filepath.Join(root, "broker.json"), struct{ URL, Pin string }{protected.URL + effortauthority.BrokerEndpoint, pin})
		joinedRead(t, filepath.Join(root, "broker-native-ready"), nil)
	}
	client.efforts = efforts
	f.runtime.Efforts = efforts
	f.runtime.Executor.agentClient = client
	f.runtime.Executor.vrooliRoot = root
	queueDir := t.TempDir()
	queues := NewTeamExecutionStore(f.teams, f.runtime.Executor, queueDir, client)
	f.runtime.Queue = queues
	f.runtime.Executor.teamExecStore = queues
	// Actual account approval has been logged out; only retained B signs this new leader.
	now = now.Add(5 * time.Hour)
	joinedWrite(t, filepath.Join(root, "clock.json"), now)
	// Admit through the actual external finite HTTP handler into a busy queue.
	initialQueue := queues.GetOrCreate("committee")
	initialQueue.running["disposable-existing-blocker"] = runningEntry{ProfileKey: "fixture-blocker"}
	ingress := intent
	ingress.Endpoint = EffortStartEndpoint("committee", "lead")
	ingress.IdempotencyKey = "finite-leader-" + uuid.NewString()
	ingress.InputDigest = effortauthority.Digest(struct{ Team, Member string }{"committee", "lead"})
	ingressProof, _ := effortauthority.Sign(effortauthority.Proof{Intent: ingress, Nonce: "joined-external-ingress-nonce", IssuedAt: now}, key)
	raw, _ := json.Marshal(ingressProof)
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	pmHTTP := httptest.NewServer(f.runtime.EffortStartHandler("committee", "lead"))
	defer pmHTTP.Close()
	beforeRecord, _ := ledger.Get(ctx, p.ID)
	beforeState, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	for _, mode := range []string{"absent", "invalid", "duplicate", "human-mix", "wrong-path"} {
		path := EffortStartEndpoint("committee", "lead")
		if mode == "wrong-path" {
			path = "/wrong"
		}
		request, _ := http.NewRequest(http.MethodPost, pmHTTP.URL+path, nil)
		if mode != "absent" {
			request.Header.Set(effortauthority.Header, encoded)
		}
		switch mode {
		case "invalid":
			request.Header.Set(effortauthority.Header, "invalid")
		case "duplicate":
			request.Header.Add(effortauthority.Header, encoded)
		case "human-mix":
			request.Header.Set("Authorization", "Bearer invalid")
		}
		response, err := pmHTTP.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode < 400 {
			t.Fatal("invalid external ingress admitted", mode)
		}
		r, _ := ledger.Get(ctx, p.ID)
		st, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
		if effortauthority.Digest(r) != effortauthority.Digest(beforeRecord) || effortauthority.Digest(st) != effortauthority.Digest(beforeState) {
			t.Fatal("invalid external ingress wrote state or ledger", mode)
		}
	}
	request, _ := http.NewRequest(http.MethodPost, pmHTTP.URL+EffortStartEndpoint("committee", "lead"), nil)
	request.Header.Set(effortauthority.Header, encoded)
	response, err := pmHTTP.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("valid external ingress failed", response.StatusCode)
	}
	queuedState, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	prepared, _ := ledger.Get(ctx, p.ID)
	if queuedState.TaskStarted || len(initialQueue.queue) != 1 || len(prepared.Reservations) != 1 || prepared.Reservations[ingress.IdempotencyKey].NativeBound {
		t.Fatal("busy queue dispatched or lost original preparation")
	}
	// Simulate PM process loss before any task/run existed; recover the actual
	// queued binding with explicit original custody, then release fixture blocker.
	queues = NewTeamExecutionStore(f.teams, f.runtime.Executor, queueDir, client)
	f.runtime.Queue = queues
	f.runtime.Executor.teamExecStore = queues
	recoveredQueue := queues.GetOrCreate("committee")
	recoveredQueue.Recover(ctx)
	if len(recoveredQueue.queue) != 1 || recoveredQueue.queue[0].caller == nil {
		t.Fatal("queued finite authority did not restore")
	}
	recoveredQueue.OnMemberComplete("disposable-existing-blocker")
	queues.Shutdown()
	state, e := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if e != nil {
		t.Fatal(e)
	}
	record, e := ledger.Get(ctx, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	receipt := record.Reservations["finite-leader-"+state.ID]
	if !receipt.NativeBound || receipt.RunID == "" || len(record.Reservations) != 1 || receipt.Preparation == nil {
		t.Fatalf("joined native acceptance missing: %#v", receipt)
	}
	task, e := client.GetTask(ctx, state.TaskID)
	if e != nil || task == nil || task.Status == "cancelled" {
		t.Fatal("uncertain finite receipt cancelled native task", e)
	}
	// Crash/restart queue, restoring original custody and exact intent over real HTTP.
	restarted := newTeamExecutionContext("committee", &captureExecutor{}, queueDir, client)
	restarted.Recover(ctx)
	after, _ := ledger.Get(ctx, p.ID)
	if len(after.Reservations) != 1 || after.Reservations["finite-leader-"+state.ID].RunID != receipt.RunID {
		t.Fatal("restart duplicated native acceptance")
	}
	// Bind the lost response from actual owner lookup, then settle its exact row.
	if _, e = f.runtime.Tick(ctx, "committee", "lead"); e != nil {
		t.Fatal(e)
	}
	state, _ = f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if state.RunID != receipt.RunID {
		t.Fatal("owner lookup did not restore exact lost response")
	}
	resp, e := client.httpClient.Post(native.URL+"/fixture/settle/"+receipt.RunID, "application/json", nil)
	if e != nil {
		t.Fatal(e)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatal("native terminal fixture transition failed", resp.StatusCode)
	}
	if _, e = f.runtime.Tick(ctx, "committee", "lead"); e != nil {
		t.Fatal("single-slot successor after exact native settlement", e)
	}
	queues.Shutdown()
	state, _ = f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	after, _ = ledger.Get(ctx, p.ID)
	successor := after.Reservations["finite-leader-"+state.ID]
	if state.ID == receipt.Preparation.IdempotencyKey[len("finite-leader-"):] || len(state.RestartHistory) != 1 || len(after.Reservations) != 2 || !after.Reservations[receipt.Intent.IdempotencyKey].Terminal || successor.RunID == "" {
		t.Fatal("single-slot joined callback lost successor or renewed budget")
	}
	// Old same-key external replay observes its original native receipt, not the successor.
	oldBefore := effortauthority.Digest(after)
	response, err = pmHTTP.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("original ingress replay refused")
	}
	after, _ = ledger.Get(ctx, p.ID)
	if effortauthority.Digest(after) != oldBefore {
		t.Fatal("old external replay duplicated or renewed successor")
	}
	// Actual current coarse-grant removal blocks PM before state/task/queue effects.
	joinedWrite(t, filepath.Join(root, "revoke-current-grant"), true)
	joinedRead(t, filepath.Join(root, "grant-revoked"), nil)
	beforeStateDigest := effortauthority.Digest(state)
	beforeLedger := effortauthority.Digest(after)
	if _, e = f.runtime.Tick(ctx, "committee", "lead"); e == nil {
		t.Fatal("actual current-grant revoke admitted")
	}
	state, _ = f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	after, _ = ledger.Get(ctx, p.ID)
	if effortauthority.Digest(state) != beforeStateDigest || effortauthority.Digest(after) != beforeLedger {
		t.Fatal("revoked joined caller wrote state or ledger")
	}
	t.Logf("joined actual PM task/run HTTP, private mTLS actual account owner, late B, single retained native receipt and queue restart qualified; fixture run=%s", receipt.RunID)
}

func joinedClientPeer(t *testing.T, ca tls.Certificate, role, root string) (tls.Certificate, string) {
	t.Helper()
	issuer, e := x509.ParseCertificate(ca.Certificate[0])
	if e != nil {
		t.Fatal(e)
	}
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	spec := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "disposable " + role}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, e := x509.CreateCertificate(rand.Reader, spec, issuer, &key.PublicKey, ca.PrivateKey)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := x509.MarshalPKCS8PrivateKey(key)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, role+"-peer.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, role+"-peer.key"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: raw}), 0600); e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(der)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, hex.EncodeToString(sum[:])
}
