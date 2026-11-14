package aimoneyfixtures

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFrontDoorContractAndFundedBoundary(t *testing.T) {
	gateway := &FakeGatewayPrivate{PlanValue: GatewayPlan{RouteID: "route-fixture", Supplier: "fixture", Model: "m"}, Result: GatewayResult{Output: "ok", InputTokens: 2, OutputTokens: 3}}
	front := &FakeFrontDoor{Gateway: gateway, UseClass: "bounded_feature"}
	body := `{"instruction":"answer safely","turn":{"role":"user","content":"hello"},"schema":{"type":"object"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/features/demo.reply", bytes.NewBufferString(body))
	req.Header.Set("Idempotency-Key", "op-1")
	rr := httptest.NewRecorder()
	front.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(gateway.Plans) != 1 || gateway.Plans[0].UseClass != "bounded_feature" {
		t.Fatalf("server-side use class not supplied: %#v", gateway.Plans)
	}
	if len(gateway.Executes) != 1 || !gateway.Executes[0].Funded {
		t.Fatalf("funded marker not set behind door: %#v", gateway.Executes)
	}
	// Replaying the same operation returns the prior result without executing again.
	req = httptest.NewRequest(http.MethodPost, "/v1/features/demo.reply", bytes.NewBufferString(body))
	req.Header.Set("Idempotency-Key", "op-1")
	rr = httptest.NewRecorder()
	front.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || len(gateway.Executes) != 1 {
		t.Fatalf("replay status=%d executes=%d", rr.Code, len(gateway.Executes))
	}
}

func TestFrontDoorCreditRefusalUses402Contract(t *testing.T) {
	front := &FakeFrontDoor{CreditRefusal: true}
	req := httptest.NewRequest(http.MethodPost, "/v1/features/demo.reply", bytes.NewBufferString(`{"instruction":"i","turn":{"role":"user","content":"x"},"schema":{"type":"object"}}`))
	rr := httptest.NewRecorder()
	front.ServeHTTP(rr, req)
	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("status=%d", rr.Code)
	}
	var body PublicError
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != ErrorCreditsRequired || body.ErrorType != ErrorCreditsRequired {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestFrontDoorNoRouteUses402Contract(t *testing.T) {
	gateway := &FakeGatewayPrivate{PlanErr: ErrNoEligibleRoute}
	front := &FakeFrontDoor{Gateway: gateway, UseClass: "in_app"}
	req := httptest.NewRequest(http.MethodPost, "/v1/features/demo.reply", bytes.NewBufferString(`{"instruction":"i","turn":{"role":"user","content":"x"},"schema":{"type":"object"}}`))
	rr := httptest.NewRecorder()
	front.ServeHTTP(rr, req)
	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("status=%d", rr.Code)
	}
	var body PublicError
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != ErrorNoEligibleRoute || body.ErrorType != ErrorNoEligibleRoute {
		t.Fatalf("unexpected body: %+v", body)
	}
	if got := CreditsRequiredBody(); got.Error != ErrorCreditsRequired || NoEligibleRouteBody().Error != ErrorNoEligibleRoute {
		t.Fatalf("public 402 bodies incorrect: %+v", got)
	}
}

func TestGatewayRequiresFundedMarkerAndReceiptIsIdempotent(t *testing.T) {
	gateway := &FakeGatewayPrivate{Result: GatewayResult{Output: "once"}}
	if _, err := gateway.Execute(context.Background(), GatewayExecuteRequest{OperationID: "op"}); err == nil {
		t.Fatal("unfunded request accepted")
	}
	request := GatewayExecuteRequest{OperationID: "op", Funded: true}
	first, err := gateway.Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	gateway.Result.Output = "changed"
	second, err := gateway.Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Output != second.Output || len(gateway.Executes) != 1 {
		t.Fatalf("replay changed result or executed twice: %#v %#v", first, second)
	}
	receipt, err := gateway.Receipt(context.Background(), GatewayReceiptRequest{OperationID: "op"})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != "complete" {
		t.Fatalf("state=%q", receipt.State)
	}
}

func TestFakeRouterUsesEphemeralLoopbackPort(t *testing.T) {
	router, err := NewFakeRouter(map[string]any{"id": "fixture", "choices": []any{map[string]any{"message": map[string]any{"content": "ok"}}}})
	if err != nil {
		t.Fatal(err)
	}
	defer router.Close()
	resp, err := http.Post(router.URL()+"/v1/chat/completions", "application/json", bytes.NewBufferString(`{"model":"fixture","messages":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(b, []byte(`"id":"fixture"`)) {
		t.Fatalf("body=%s", b)
	}
	router.mu.Lock()
	defer router.mu.Unlock()
	if len(router.Requests) != 1 {
		t.Fatalf("requests=%d", len(router.Requests))
	}
}

func TestFakeStripeEventStreamAndReplay(t *testing.T) {
	stream := &FakeStripe{}
	event := StripeEvent{ID: "evt_async", Type: StripeAsyncPaymentSucceeded, ObjectID: "cs_1"}
	if err := stream.Publish(event); err != nil {
		t.Fatal(err)
	}
	if err := stream.Publish(event); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{StripeDisputeCreated, StripeDisputeClosed, StripeEarlyFraudWarning, StripeRefunded, StripeAsyncPaymentFailed} {
		if err := stream.Publish(StripeEvent{ID: "evt_" + kind, Type: kind, ObjectID: "ch_1"}); err != nil {
			t.Fatal(err)
		}
	}
	if len(stream.Events()) != 6 {
		t.Fatalf("duplicate event not idempotent: %d", len(stream.Events()))
	}
	ctx := context.Background()
	for i := 0; i < 6; i++ {
		if _, err := stream.Next(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := stream.Next(ctx); err == nil {
		t.Fatal("expected drained stream")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := stream.Next(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestFixtureKeyIssuerExpiryAndRevocation(t *testing.T) {
	issuer := &StaticKeyIssuer{}
	now := time.Now().UTC()
	token, err := issuer.Issue("account-1", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !issuer.Valid(token, now) || issuer.Valid(token, now.Add(time.Minute)) {
		t.Fatal("expiry semantics incorrect")
	}
	issuer.Revoke(token)
	if issuer.Valid(token, now) {
		t.Fatal("revoked token valid")
	}
}

func TestRuleRecordFixtureDoesNotAutoApprove(t *testing.T) {
	record := RuleRecord{Supplier: "fixture", Plan: "payg", UseClasses: map[string]string{"in_app": "allowed"}, Review: Review{Status: "pending_review"}}
	eligible := StubEligible(false, "zero approved records")
	if allowed, _ := eligible(record, nil); allowed {
		t.Fatal("fixture auto-approved a rule")
	}
	if record.Review.Status != "pending_review" {
		t.Fatal("record changed")
	}
}

func TestLifecycleBackendSelectsExplicitBackend(t *testing.T) {
	local := &FakeLifecycleBackend{Name: "local", Snapshots: map[string]LifecycleSnapshot{"codex": {State: "managed", Version: "1"}}}
	server := &FakeLifecycleBackend{Name: "server", Snapshots: map[string]LifecycleSnapshot{"codex": {State: "found", Version: "2"}}}
	backends := TwoBackends{Local: local, Server: server}
	got, err := backends.Backend(true).Observe(context.Background(), "codex")
	if err != nil || got.Backend != "local" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	got, err = backends.Backend(false).Observe(context.Background(), "codex")
	if err != nil || got.Backend != "server" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestSignedComponentFixtureAndHelloDesktopPayload(t *testing.T) {
	bundle, err := NewSignedComponentFixture()
	if err != nil {
		t.Fatal(err)
	}
	if err := bundle.Verify(); err != nil {
		t.Fatal(err)
	}
	serviceJSON, ok := bundle.Payload["hello-desktop/.vrooli/service.json"]
	if !ok {
		t.Fatal("hello-desktop service.json missing from fixture")
	}
	var service struct {
		Service struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"service"`
	}
	if err := json.Unmarshal(serviceJSON, &service); err != nil {
		t.Fatalf("hello-desktop service.json is invalid: %v", err)
	}
	if service.Service.Name != "hello-desktop" || service.Service.Version != "1.0.0" {
		t.Fatalf("unexpected hello-desktop service manifest: %+v", service.Service)
	}
	if len(bundle.PublicKey) != ed25519.PublicKeySize {
		t.Fatalf("public key length=%d", len(bundle.PublicKey))
	}
	bundle.Payload["hello-desktop/README.md"] = []byte("tampered")
	if err := bundle.Verify(); err == nil {
		t.Fatal("tampered payload accepted")
	}
}

func TestChatRequestAcceptsOnlyOneUserTurn(t *testing.T) {
	gateway := &FakeGatewayPrivate{PlanValue: GatewayPlan{RouteID: "r", Supplier: "s", Model: "m"}, Result: GatewayResult{Output: "ok"}}
	front := &FakeFrontDoor{Gateway: gateway, UseClass: "harness_access"}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"fixture-model","messages":[{"role":"user","content":"hello"}]}`))
	rr := httptest.NewRecorder()
	front.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(gateway.Plans) != 1 || gateway.Plans[0].FeatureID != "coding-agent" || gateway.Plans[0].Model != "fixture-model" {
		t.Fatalf("plan=%+v", gateway.Plans)
	}
	bad := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"messages":[{"role":"user","content":"one"},{"role":"assistant","content":"two"}]}`))
	brr := httptest.NewRecorder()
	front.ServeHTTP(brr, bad)
	if brr.Code != http.StatusBadRequest {
		t.Fatalf("multi-turn status=%d", brr.Code)
	}
}

func TestUsageReceiptGoldenAndSyntheticCorpusLabel(t *testing.T) {
	receipt := loadGoldenReceipt(t)
	assertReceiptAttribution(t, receipt)
	assertSyntheticCorpus(t)
}

func loadGoldenReceipt(t *testing.T) UsageReceipt {
	t.Helper()
	golden, err := os.ReadFile(filepath.Join("testdata", "usage-receipt.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt UsageReceipt
	if err := json.Unmarshal(golden, &receipt); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(encoded, '\n'), golden) {
		t.Fatalf("receipt serialization drifted\nwant: %s\ngot: %s", golden, append(encoded, '\n'))
	}
	return receipt
}

func assertReceiptAttribution(t *testing.T, receipt UsageReceipt) {
	t.Helper()
	if len(receipt.LotIDs) != 1 || len(receipt.SourceKinds) != 1 || receipt.UpstreamCostMicros >= receipt.CustomerChargeMicros {
		t.Fatalf("bad attribution receipt: %+v", receipt)
	}
}

func assertSyntheticCorpus(t *testing.T) {
	t.Helper()
	var corpus struct {
		Label    string            `json:"label"`
		Verified bool              `json:"verified"`
		Cases    []json.RawMessage `json:"cases"`
	}
	data, err := os.ReadFile(filepath.Join("testdata", "limits", "corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Label != "synthetic_unverified" || corpus.Verified || len(corpus.Cases) == 0 {
		t.Fatalf("corpus incorrectly represents evidence: %+v", corpus)
	}
}
