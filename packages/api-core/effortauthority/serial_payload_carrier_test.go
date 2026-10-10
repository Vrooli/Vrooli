// serial_payload_carrier_test.go uses disposable SQL and in-process HTTP only.
package effortauthority

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func protectedSerialFixture(t *testing.T) (*Authority, *fixtureOwners, *fixtureScope, Policy, Binding, Intent, *time.Time, ed25519.PrivateKey) {
	t.Helper()
	a, owners, scope, p, key, now := authorityFixture(t)
	p.ID = "protected-serial-fixture"
	p.MaxConcurrent = 1
	p.Profiles = map[string]string{"leader-profile": Digest("leader complete"), "worker-profile": Digest("worker complete")}
	p.SerialEdges = []SerialEdge{{"leader", "leader-profile", "worker", "worker-profile"}, {"worker", "worker-profile", "leader", "leader-profile"}}
	if err := a.Approve(context.Background(), "fixture-human", p); err != nil {
		t.Fatal(err)
	}
	root := nativeIntent(p, "protected-parent")
	root.Profile = "leader-profile"
	root.ProfileDigest = p.Profiles[root.Profile]
	b := binding(p)
	startSerialRoot(t, a, b, root)
	return a, owners, scope, p, b, root, now, key
}
func protectedPayloadNext(p Policy, root Intent) (Intent, json.RawMessage) {
	payload := json.RawMessage(`{"task":{"title":"disposable worker","scope":"fixture"},"tag":"worker"}`)
	next := serialNext(p, root, "native-parent", "protected-worker", "worker", "worker-profile")
	next.SerialPayloadDigest = Digest(string(payload))
	return next, payload
}
func TestProtectedSerialPayloadDurableReplayCopiesAndUniqueRunIdentity(t *testing.T) {
	ctx := context.Background()
	a, _, _, p, b, root, _, _ := protectedSerialFixture(t)
	next, payload := protectedPayloadNext(p, root)
	expected := append(json.RawMessage(nil), payload...)
	h, replay, err := a.PrepareSerialHandoffWithPayload(ctx, b, root.IdempotencyKey, next, payload)
	if err != nil || replay || !bytes.Equal(h.Payload, expected) {
		t.Fatal("prepare", err)
	}
	payload[0] = '!'
	h.Payload[0] = '!'
	restarted := *a
	got, err := restarted.ReadSerialHandoff(ctx, b, next.IdempotencyKey)
	if err != nil || !bytes.Equal(got.Payload, expected) {
		t.Fatal("frozen payload read", err)
	}
	got.Payload[0] = '!'
	before, _ := a.Store.Get(ctx, p.ID)
	h, replay, err = a.PrepareSerialHandoffWithPayload(ctx, b, root.IdempotencyKey, next, expected)
	if err != nil || !replay || !bytes.Equal(h.Payload, expected) {
		t.Fatal("exact replay", err)
	}
	unchangedRecord(t, a, p.ID, Digest(before))
	if err = a.Settle(ctx, b, root.IdempotencyKey, "native-parent"); err != nil {
		t.Fatal(err)
	}
	slot, replay, err := restarted.ReserveSerialEpisode(ctx, b, next.IdempotencyKey)
	if err != nil || replay || slot.Intent.SerialPayloadDigest != next.SerialPayloadDigest {
		t.Fatal("reserve", err)
	}
	before, _ = a.Store.Get(ctx, p.ID)
	if err = a.Accept(ctx, b, next.IdempotencyKey, "native-parent"); err == nil {
		t.Fatal("reused predecessor run identity accepted")
	}
	unchangedRecord(t, a, p.ID, Digest(before))
	if err = a.Accept(ctx, b, next.IdempotencyKey, "native-worker"); err != nil {
		t.Fatal(err)
	}
	if err = a.Accept(ctx, b, next.IdempotencyKey, "native-worker"); err != nil {
		t.Fatal("accepted replay", err)
	}
}
func TestProtectedSerialPayloadRefusalsHaveZeroLedgerEffects(t *testing.T) {
	for _, name := range []string{"digest", "invalid-json", "raw-html", "oversized", "missing", "binding", "owner-grant", "scope", "expired", "cancelled", "changed-replay"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			a, owner, scope, p, b, root, now, _ := protectedSerialFixture(t)
			next, payload := protectedPayloadNext(p, root)
			if name == "changed-replay" {
				if _, _, err := a.PrepareSerialHandoffWithPayload(ctx, b, root.IdempotencyKey, next, payload); err != nil {
					t.Fatal(err)
				}
				payload = json.RawMessage(`{"task":"changed"}`)
				next.SerialPayloadDigest = Digest(string(payload))
			}
			before, _ := a.Store.Get(ctx, p.ID)
			switch name {
			case "digest":
				next.SerialPayloadDigest = Digest("foreign")
			case "invalid-json":
				payload = []byte(`{"broken"`)
				next.SerialPayloadDigest = Digest(string(payload))
			case "raw-html":
				payload = json.RawMessage(`{"task":"<owner>&fixture</owner>"}`)
				next.SerialPayloadDigest = Digest(string(payload))
			case "oversized":
				payload = []byte(`"` + strings.Repeat("x", 24576) + `"`)
				next.SerialPayloadDigest = Digest(string(payload))
			case "missing":
				payload = nil
			case "binding":
				b.Epoch++
			case "owner-grant":
				owner.state.Scopes = nil
			case "scope":
				scope.invalid = true
			case "expired":
				*now = p.Deadline
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, _, err := a.PrepareSerialHandoffWithPayload(ctx, b, root.IdempotencyKey, next, payload); err == nil {
				t.Fatal("refusal accepted")
			}
			unchangedRecord(t, a, p.ID, Digest(before))
		})
	}
}
func TestProtectedSerialCorruptedPayloadReadAndReserveRefuse(t *testing.T) {
	ctx := context.Background()
	a, _, _, p, b, root, _, _ := protectedSerialFixture(t)
	next, payload := protectedPayloadNext(p, root)
	if _, _, err := a.PrepareSerialHandoffWithPayload(ctx, b, root.IdempotencyKey, next, payload); err != nil {
		t.Fatal(err)
	}
	if err := a.Settle(ctx, b, root.IdempotencyKey, "native-parent"); err != nil {
		t.Fatal(err)
	}
	if err := a.Store.Update(ctx, p.ID, func(r *Record) error {
		h := r.SerialHandoffs[next.IdempotencyKey]
		h.Payload = json.RawMessage(`{"task":"corrupted"}`)
		r.SerialHandoffs[next.IdempotencyKey] = h
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, _ := a.Store.Get(ctx, p.ID)
	if _, err := a.ReadSerialHandoff(ctx, b, next.IdempotencyKey); err == nil {
		t.Fatal("corrupt read accepted")
	}
	if _, _, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey); err == nil {
		t.Fatal("corrupt reserve accepted")
	}
	unchangedRecord(t, a, p.ID, Digest(before))
}

// RawMessage SQL serialization must not acknowledge a handoff that cannot later
// be read. Noncanonical input may be refused before effects or preserved exactly.
func TestProtectedSerialWhitespacePayloadIsPreservedOrRefusedBeforeEffects(t *testing.T) {
	ctx := context.Background()
	a, _, _, p, b, root, _, _ := protectedSerialFixture(t)
	next, _ := protectedPayloadNext(p, root)
	payload := json.RawMessage("{\n  \"task\": \"fixture\"\n}")
	next.SerialPayloadDigest = Digest(string(payload))
	before, _ := a.Store.Get(ctx, p.ID)
	_, _, err := a.PrepareSerialHandoffWithPayload(ctx, b, root.IdempotencyKey, next, payload)
	if err != nil {
		unchangedRecord(t, a, p.ID, Digest(before))
		return
	}
	got, err := a.ReadSerialHandoff(ctx, b, next.IdempotencyKey)
	if err != nil || !bytes.Equal(got.Payload, payload) {
		t.Fatal("acknowledged payload lost exact durable bytes", err)
	}
}

// The TLS state below is a synthetic verified fixture, not a socket/host proof.
func TestProtectedSerialBrokerExactPeerOperationsChannelsAndCurrentBinding(t *testing.T) {
	for _, name := range []string{"positive", "absent-tls", "unverified", "wrong-peer", "wrong-operation", "Authorization", "Cookie", "X-Agent-Identity-Token", Header, "wrong-binding", "custody-missing", "owner-grant"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			a, owner, _, p, b, root, _, key := protectedSerialFixture(t)
			next, payload := protectedPayloadNext(p, root)
			cert, _, pin := transportCertificate(t)
			leaf, err := x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				t.Fatal(err)
			}
			handler := &BrokerHandler{Authority: a, Custody: &CustodySigner{Clients: map[string]crypto.Signer{p.Client: key}}, Peers: map[string]map[string]BrokerGrant{pin: {p.ID: {PolicyID: p.ID, Operations: map[string]bool{"prepare-serial-handoff": true, "read-serial-handoff": true}}}}}
			if name == "wrong-binding" {
				b.Epoch++
			}
			raw, _ := json.Marshal(transaction{Binding: b, Key: root.IdempotencyKey, Intent: next, Payload: payload})
			req := httptest.NewRequest(http.MethodPost, BrokerEndpoint+"prepare-serial-handoff", bytes.NewReader(raw))
			req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf}, VerifiedChains: [][]*x509.Certificate{{leaf}}}
			switch name {
			case "absent-tls":
				req.TLS = nil
			case "unverified":
				req.TLS.VerifiedChains = nil
			case "wrong-peer":
				handler.Peers = nil
			case "wrong-operation":
				handler.Peers[pin][p.ID] = BrokerGrant{PolicyID: p.ID, Operations: map[string]bool{"read-serial-handoff": true}}
			case "custody-missing":
				handler.Custody = nil
			case "owner-grant":
				owner.state.Scopes = nil
			case "Authorization", "Cookie", "X-Agent-Identity-Token", Header:
				req.Header.Add(name, "")
			}
			before, _ := a.Store.Get(ctx, p.ID)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if name != "positive" {
				if rec.Code != http.StatusForbidden {
					t.Fatal("wrong-channel/current refusal", rec.Code)
				}
				unchangedRecord(t, a, p.ID, Digest(before))
				return
			}
			if rec.Code != http.StatusOK {
				t.Fatal("protected prepare", rec.Code, rec.Body.String())
			}
			var out transactionResult
			if json.Unmarshal(rec.Body.Bytes(), &out) != nil || out.Handoff == nil || !bytes.Equal(out.Handoff.Payload, payload) {
				t.Fatal("protected payload reply")
			}
			before, _ = a.Store.Get(ctx, p.ID)
			raw, _ = json.Marshal(transaction{Binding: b, Key: next.IdempotencyKey})
			req = httptest.NewRequest(http.MethodPost, BrokerEndpoint+"read-serial-handoff", bytes.NewReader(raw))
			req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf}, VerifiedChains: [][]*x509.Certificate{{leaf}}}
			rec = httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatal("protected read", rec.Code)
			}
			unchangedRecord(t, a, p.ID, Digest(before))
		})
	}
}
