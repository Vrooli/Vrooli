// effort_contract_combined_test.go qualifies actual private broker, custody and native admission without a host launch.
package orchestration

import (
	"agent-manager/internal/orchestration/spawn"
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/vrooli/api-core/effortauthority"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"github.com/vrooli/vrooli/packages/credential-authority-go/finitesigning"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// This disposable store is never the host credential authority. Writes/deletes
// deliberately fail; fixture enrollment bytes are built only in test memory.
type combinedPurposeStore struct {
	record  string
	reads   int
	revoked bool
}

func (s *combinedPurposeStore) Put(string, string, string) error { return effortauthority.ErrRefused }
func (s *combinedPurposeStore) Delete(string, string) error      { return effortauthority.ErrRefused }
func (s *combinedPurposeStore) Get(service, key string) (string, error) {
	s.reads++
	if s.revoked {
		return "", effortauthority.ErrRefused
	}
	return s.record, nil
}

// Genuine mutual TLS over net.Pipe exercises the real private HTTP adapter.
// It opens no listener/socket and does not qualify host isolation or model IO.
func combinedPrivateHTTP(t *testing.T, h http.Handler) (*http.Client, string) {
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

func TestContractsCombinedPurposePrivateBrokerAndNativeCreateRun(t *testing.T) {
	ctx := context.Background()
	o, a, p, task, profile, key, clock := nativeEffortFixture(t)
	record, _ := json.Marshal(map[string]any{"version": 1, "purpose": finitesigning.Purpose, "handleId": "disposable-finite-handle", "client": p.Client, "policyDigest": effortauthority.Digest(p), "publicKey": base64.RawStdEncoding.EncodeToString(p.ClientKey), "privateKey": base64.RawStdEncoding.EncodeToString(key), "revoked": false})
	memory := &combinedPurposeStore{record: string(record)}
	owner, e := credentialauthority.NewAuthority(memory)
	if e != nil {
		t.Fatal(e)
	}
	custody, e := finitesigning.New(finitesigning.Config{Authority: owner, Descriptor: finitesigning.Descriptor{Identity: "finite-effort/disposable", HandleID: "disposable-finite-handle", PublicKey: p.ClientKey, Policy: p}})
	if e != nil {
		t.Fatal(e)
	}
	h := &effortauthority.BrokerHandler{Authority: a, Custody: &effortauthority.CustodySigner{Purpose: custody}}
	httpClient, pin := combinedPrivateHTTP(t, h)
	h.Peers = map[string]map[string]effortauthority.BrokerGrant{pin: {p.ID: {PolicyID: p.ID, Operations: map[string]bool{"sign": true, "check-proof": true, "check-binding": true, "reserve": true, "accept": true, "read-reservation": true, "observe-ingress": true}}}}
	binding := effortauthority.Binding{PolicyID: p.ID, PolicyDigest: effortauthority.Digest(p), Epoch: p.Epoch, Owner: p.Owner, Client: p.Client, Deadline: p.Deadline}
	broker, e := effortauthority.NewBrokerClient(effortauthority.BrokerClientConfig{URL: "https://fixture.invalid" + effortauthority.BrokerEndpoint, HTTP: httpClient, ServerPin: pin, Bindings: map[string]effortauthority.Binding{p.ID: binding}})
	if e != nil {
		t.Fatal(e)
	}
	o.effortAuthority = broker
	req := effortRequest(t, o, p, task, profile, key, "combined-source")
	bytes, e := base64.RawURLEncoding.DecodeString(req.EffortProof)
	if e != nil {
		t.Fatal(e)
	}
	var original effortauthority.Proof
	if json.Unmarshal(bytes, &original) != nil {
		t.Fatal("proof fixture")
	}
	signed, e := broker.SignEffort(ctx, p.Client, original)
	if e != nil {
		t.Fatal("private purpose signing", e)
	}
	bytes, _ = json.Marshal(signed)
	req.EffortProof = base64.RawURLEncoding.EncodeToString(bytes)
	*clock = time.Now().UTC() // current owner read follows custody issuance, avoiding a frozen-clock future proof.
	run, e := o.CreateRun(ctx, req)
	if !errors.Is(e, spawn.ErrDispatcherClosed) {
		t.Fatal("expected isolated dispatcher boundary", run, e)
	}
	accepted, e := o.runs.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if e != nil || accepted == nil || accepted.OwnerSubject != p.Owner {
		t.Fatal("actual native owner admission", e)
	}
	rec, _ := a.Store.Get(ctx, p.ID)
	if rec.Reservations[req.IdempotencyKey].RunID != accepted.ID.String() {
		t.Fatal("private ledger lost exact native receipt")
	}
	replay, e := o.CreateRun(ctx, req)
	if e != nil || replay.ID != accepted.ID {
		t.Fatal("same-key positive compatibility", e)
	}

	// Reserve preserves canonical child/recovery effects for authenticated adapters;
	// the dedicated custody signer still refuses to create those proof purposes.
	for _, effect := range []string{"run.child", "run.recover"} {
		t.Run("private-reserve-"+effect, func(t *testing.T) {
			intent := original.Intent
			intent.IdempotencyKey = "compat-" + effect
			intent.Effect = effect
			if effect == "run.child" {
				intent.ParentRunID = accepted.ID.String()
			} else {
				intent.SourceRunID = accepted.ID.String()
				intent.Endpoint = "/api/v1/runs/resume-from-failed"
			}
			if _, err := broker.SignEffort(ctx, p.Client, effortauthority.Proof{Intent: intent}); err == nil {
				t.Fatal("dedicated custody signed wider effect")
			}
			rec, replay, err := broker.Reserve(ctx, binding, intent, nil)
			if err != nil || replay || rec.Intent.Effect != effect {
				t.Fatal("canonical private reserve compatibility", err)
			}
			before, _ := a.Store.Get(ctx, p.ID)
			memory.revoked = true
			intent.IdempotencyKey = "revoked-" + effect
			_, _, err = broker.Reserve(ctx, binding, intent, nil)
			memory.revoked = false
			after, _ := a.Store.Get(ctx, p.ID)
			if err == nil || effortauthority.Digest(before) != effortauthority.Digest(after) {
				t.Fatal("revoked purpose reserve changed ledger")
			}
		})
	}
	for _, mode := range []string{"absent", "invalid", "mixed-human", "revoked-custody", "revoked-signed-proof", "wrong-server"} {
		t.Run(mode, func(t *testing.T) {
			before, _ := a.Store.Get(ctx, p.ID)
			offer := req
			offer.IdempotencyKey = "reject-" + mode
			offer.EffortProof = ""
			reads := memory.reads
			switch mode {
			case "invalid":
				offer.EffortProof = "invalid"
			case "mixed-human":
				offer.EffortProof = req.EffortProof
				offer.OwnerToken = "invalid"
			case "revoked-custody":
				memory.revoked = true
				if _, e := broker.SignEffort(ctx, p.Client, original); e == nil {
					t.Fatal("revoked custody signed")
				}
				memory.revoked = false
			case "revoked-signed-proof":
				fresh := effortRequest(t, o, p, task, profile, key, offer.IdempotencyKey)
				raw, _ := base64.RawURLEncoding.DecodeString(fresh.EffortProof)
				var intent effortauthority.Proof
				if json.Unmarshal(raw, &intent) != nil {
					t.Fatal("fresh proof fixture")
				}
				proof, err := broker.SignEffort(ctx, p.Client, intent)
				if err != nil {
					t.Fatal(err)
				}
				raw, _ = json.Marshal(proof)
				offer.EffortProof = base64.RawURLEncoding.EncodeToString(raw)
				*clock = time.Now().UTC()
				memory.revoked = true
				defer func() { memory.revoked = false }()
			case "wrong-server":
				bad, e := effortauthority.NewBrokerClient(effortauthority.BrokerClientConfig{URL: "https://fixture.invalid" + effortauthority.BrokerEndpoint, HTTP: httpClient, ServerPin: effortauthority.Digest("wrong-server"), Bindings: map[string]effortauthority.Binding{p.ID: binding}})
				if e != nil {
					t.Fatal(e)
				}
				if _, e := bad.SignEffort(ctx, p.Client, original); e == nil {
					t.Fatal("wrong server sent signing request")
				}
				if memory.reads != reads {
					t.Fatal("wrong server reached custody")
				}
			}
			if r, e := o.CreateRun(ctx, offer); e == nil || r != nil {
				t.Fatal("invalid caller admitted", e)
			}
			after, _ := a.Store.Get(ctx, p.ID)
			if effortauthority.Digest(before) != effortauthority.Digest(after) {
				t.Fatal("rejection changed ledger")
			}
			if r, _ := o.runs.GetByIdempotencyKey(ctx, offer.IdempotencyKey); r != nil {
				t.Fatal("rejection created row")
			}
		})
	}
}
