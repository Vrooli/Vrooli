package effortauthority

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPrivateBrokerSemanticTransactionsAndSignOnly(t *testing.T) {
	a, owner, scope, p, key, now := authorityFixture(t)
	ctx := context.Background()
	b := binding(p)
	cert, pool, pin := transportCertificate(t)
	grant := BrokerGrant{PolicyID: p.ID, Operations: map[string]bool{"sign": true, "check-binding": true, "check-proof": true, "reserve": true, "accept": true, "settle": true, "read-reservation": true, "observe-ingress": true, "prepare": true}}
	h := &BrokerHandler{Authority: a, Custody: &CustodySigner{Clients: map[string]crypto.Signer{p.Client: key}}, Peers: map[string]map[string]BrokerGrant{pin: {p.ID: grant}}}
	var handlerCalls atomic.Int64
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handlerCalls.Add(1); h.ServeHTTP(w, r) }))
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
	server.StartTLS()
	defer server.Close()
	client := &BrokerClient{URL: server.URL + BrokerEndpoint, ServerPin: pin, Bindings: map[string]Binding{p.ID: b}, HTTP: &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{cert}}}}}
	intent := nativeIntent(p, "broker-native")
	proof, e := client.SignEffort(ctx, p.Client, Proof{Intent: intent, Nonce: "client supplied", IssuedAt: now.Add(24 * time.Hour)})
	if e != nil || proof.Nonce == "client supplied" || !proof.IssuedAt.Equal(*now) {
		t.Fatalf("issuer signing failed: %v", e)
	}
	verified, e := client.CheckProof(ctx, proof)
	if e != nil || verified != b {
		t.Fatal("signed proof not accepted", e)
	}
	slot, replay, e := client.Reserve(ctx, b, intent, &proof)
	if e != nil || replay || !slot.NativeBound {
		t.Fatal("native reservation", e)
	}
	if e = client.Accept(ctx, b, intent.IdempotencyKey, "native-run"); e != nil {
		t.Fatal(e)
	}
	slot, e = client.ReadReservation(ctx, b, intent.IdempotencyKey)
	if e != nil || slot.RunID != "native-run" {
		t.Fatal("receipt lost", e)
	}
	before, _ := a.Store.Get(ctx, p.ID)
	for _, name := range []string{"wrong-policy", "wrong-intent", "wrong-client", "wrong-server", "wrong-operation", "owner-grant-loss", "scope-drift", "custody-missing", "wrong-enrolled-key", "expired"} {
		t.Run(name, func(t *testing.T) {
			offered := intent
			offeredClient := p.Client
			oldPin := client.ServerPin
			oldNow := *now
			oldScopes := owner.state.Scopes
			oldGrant := h.Peers[pin][p.ID]
			oldCustody := h.Custody
			oldScope := scope.invalid
			switch name {
			case "wrong-policy":
				offered.PolicyID = "foreign"
			case "wrong-intent":
				offered.Turns = p.MaxTurns + 1
			case "wrong-client":
				offeredClient = "foreign"
			case "wrong-server":
				client.ServerPin = "foreign"
			case "wrong-operation":
				g := grant
				g.Operations = map[string]bool{"check-binding": true}
				h.Peers[pin][p.ID] = g
			case "owner-grant-loss":
				owner.state.Scopes = nil
			case "scope-drift":
				scope.invalid = true
			case "custody-missing":
				h.Custody = nil
			case "wrong-enrolled-key":
				_, other, e := ed25519.GenerateKey(rand.Reader)
				if e != nil {
					t.Fatal(e)
				}
				h.Custody = &CustodySigner{Clients: map[string]crypto.Signer{p.Client: other}}
			case "expired":
				*now = p.Deadline
			}
			if _, e = client.SignEffort(ctx, offeredClient, Proof{Intent: offered}); e == nil {
				t.Fatal("invalid sign accepted")
			}
			client.ServerPin = oldPin
			*now = oldNow
			owner.state.Scopes = oldScopes
			h.Peers[pin][p.ID] = oldGrant
			h.Custody = oldCustody
			scope.invalid = oldScope
			unchangedRecord(t, a, p.ID, Digest(before))
		})
	}
	// Pin refusal must happen before an effectful HTTP request reaches issuer.
	badIntent := nativeIntent(p, "wrong-server-effect")
	badProof := signedFixture(t, a, p, key, badIntent, "wrong-server-effect-nonce")
	callsBefore := handlerCalls.Load()
	client.ServerPin = "foreign"
	if _, _, err := client.Reserve(ctx, b, badIntent, &badProof); err == nil {
		t.Fatal("wrong pinned server reserve accepted")
	}
	client.ServerPin = pin
	if handlerCalls.Load() != callsBefore {
		t.Fatal("wrong pin reached HTTP authority")
	}
	unchangedRecord(t, a, p.ID, Digest(before))
	transport := client.HTTP.Transport.(*http.Transport)
	var priorCalls atomic.Int64
	transport.TLSClientConfig.VerifyConnection = func(tls.ConnectionState) error { priorCalls.Add(1); return ErrRefused }
	if _, _, err := client.Reserve(ctx, b, badIntent, &badProof); err == nil {
		t.Fatal("prior verification refusal bypassed")
	}
	transport.TLSClientConfig.VerifyConnection = nil
	if priorCalls.Load() == 0 || handlerCalls.Load() != callsBefore {
		t.Fatal("prior callback or zero HTTP refusal unproved")
	}
	unchangedRecord(t, a, p.ID, Digest(before))
	raw, _ := json.Marshal(transaction{Binding: b})
	for _, channel := range []string{"Authorization", "Cookie", "X-Agent-Identity-Token", Header} {
		request, _ := http.NewRequest("POST", client.URL+"check-binding", bytes.NewReader(raw))
		request.Header.Set(channel, "offered wrong channel")
		response, e := client.HTTP.Do(request)
		if e != nil {
			t.Fatal(e)
		}
		response.Body.Close()
		if response.StatusCode != 403 {
			t.Fatal("wrong public channel admitted", channel)
		}
		unchangedRecord(t, a, p.ID, Digest(before))
	}
	if _, e := client.call(ctx, "raw-sign", transaction{Binding: b}); e == nil {
		t.Fatal("raw signing oracle exists")
	}
	if _, e := client.call(ctx, "update-record", transaction{Binding: b}); e == nil {
		t.Fatal("generic ledger mutation exists")
	}
	unchangedRecord(t, a, p.ID, Digest(before))
	// PM cannot report native completion when its peer lacks that operation.
	g := grant
	g.Operations = map[string]bool{"sign": true}
	h.Peers[pin][p.ID] = g
	if client.Settle(ctx, b, intent.IdempotencyKey, "native-run") == nil {
		t.Fatal("wrong-role settlement")
	}
	unchangedRecord(t, a, p.ID, Digest(before))
	h.Peers[pin][p.ID] = grant
	if client.Settle(ctx, b, intent.IdempotencyKey, "native-run") != nil {
		t.Fatal("native settlement")
	}
}

func TestFiniteConsentExactHumanProposalAndZeroEffects(t *testing.T) {
	a, owner, scope, p, _, _ := authorityFixture(t)
	p.ID = "new-human-consent"
	h := &ConsentHandler{Authority: a, Proposals: map[string]Policy{p.ID: p}, Origin: "https://issuer.example", CookieName: "vrooli_access_token"}
	request := func(method, body, origin, token string) *httptest.ResponseRecorder {
		target := ConsentEndpoint
		if method == "GET" {
			target += "?id=" + p.ID
		}
		r := httptest.NewRequest(method, "https://issuer.example"+target, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if token != "" {
			r.AddCookie(&http.Cookie{Name: h.CookieName, Value: token})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	duplicate := httptest.NewRequest("GET", "https://issuer.example"+ConsentEndpoint+"?id="+p.ID, nil)
	duplicate.Header.Set("Cookie", h.CookieName+"=; "+h.CookieName+"=synthetic-human")
	duplicateResult := httptest.NewRecorder()
	h.ServeHTTP(duplicateResult, duplicate)
	if duplicateResult.Code != 403 {
		t.Fatal("ambiguous empty-first cookie accepted")
	}
	if _, e := a.Store.Get(context.Background(), p.ID); e == nil {
		t.Fatal("ambiguous cookie wrote grant")
	}
	page := request("GET", "", "", "synthetic-human")
	if page.Code != 200 || !strings.Contains(page.Body.String(), Digest(p)) || !strings.Contains(page.Body.String(), "survives sign-out") || strings.Contains(page.Body.String(), "synthetic-human") {
		t.Fatal("review surface missing or leaked bearer")
	}
	form := url.Values{"id": {p.ID}, "digest": {Digest(p)}, "action": {"approve"}, "surviveLogout": {"yes"}}
	for _, name := range []string{"no-consent", "wrong-digest", "wrong-origin", "missing-human", "expired-human", "foreign-owner"} {
		t.Run(name, func(t *testing.T) {
			v := url.Values{}
			for k, x := range form {
				v[k] = append([]string{}, x...)
			}
			origin := h.Origin
			token := "synthetic-human"
			subject := owner.state.Subject
			switch name {
			case "no-consent":
				v.Del("surviveLogout")
			case "wrong-digest":
				v.Set("digest", "foreign")
			case "wrong-origin":
				origin = "https://foreign.example"
			case "missing-human":
				token = ""
			case "expired-human":
				owner.expired = true
			case "foreign-owner":
				owner.state.Subject = "foreign"
			}
			if request("POST", v.Encode(), origin, token).Code != 403 {
				t.Fatal("invalid consent accepted")
			}
			owner.expired = false
			owner.state.Subject = subject
			if _, e := a.Store.Get(context.Background(), p.ID); e == nil {
				t.Fatal("invalid consent created grant")
			}
		})
	}
	if request("POST", form.Encode(), h.Origin, "synthetic-human").Code != 200 {
		t.Fatal("valid human approval refused")
	}
	b := binding(p)
	if _, e := a.CheckBinding(context.Background(), b); e != nil {
		t.Fatal(e)
	}
	form.Set("action", "revoke")
	form.Del("surviveLogout")
	owner.state.Scopes = nil
	scope.invalid = true
	if request("GET", "", "", "synthetic-human").Code != 200 {
		t.Fatal("owner could not view own revoke surface after drift")
	}
	if request("POST", form.Encode(), h.Origin, "synthetic-human").Code != 200 {
		t.Fatal("self revocation refused after grant/scope drift")
	}
	record, e := a.Store.Get(context.Background(), p.ID)
	if e != nil || !record.Revoked {
		t.Fatal("revoke after drift not persisted", e)
	}
	if _, e := a.CheckBinding(context.Background(), b); e == nil {
		t.Fatal("revocation ignored")
	}
}
