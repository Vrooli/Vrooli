package consumeridentity

import (
	"strings"
	"testing"
	"time"
)

func TestCreditAssertionBindsOwnerAppAudienceOperationAndSource(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	signer, err := GenerateSigner("lpbs-test", "lpbs.fixture", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	signer.Now = func() time.Time { return now }
	verifier := Verifier{Keys: NewKeySet(PublicKey{ID: signer.KeyID, Key: &signer.Private.PublicKey}), Issuer: "lpbs.fixture", Now: func() time.Time { return now }}
	expect := CreditAssertionExpectation{BusinessAccountID: "acct-1", ActorID: "actor-1", AppKey: "audio-tools", FeatureID: "audio-tools.summarize", MaxCredits: 500, Audience: CreditGatewayAudience, OperationID: "op-1", Source: CreditAssertionSource}
	token, expires, err := signer.SignCreditAssertion(expect, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if expires.Sub(now) != time.Minute {
		t.Fatalf("expiration = %s", expires.Sub(now))
	}
	if _, err := VerifyCreditAssertion(verifier, token, expect); err != nil {
		t.Fatalf("valid assertion refused: %v", err)
	}
	for name, change := range map[string]func(*CreditAssertionExpectation){
		"account":          func(e *CreditAssertionExpectation) { e.BusinessAccountID = "acct-2" },
		"actor":            func(e *CreditAssertionExpectation) { e.ActorID = "actor-2" },
		"app":              func(e *CreditAssertionExpectation) { e.AppKey = "web-console" },
		"feature":          func(e *CreditAssertionExpectation) { e.FeatureID = "web-console.chat" },
		"credit cap":       func(e *CreditAssertionExpectation) { e.MaxCredits++ },
		"audience":         func(e *CreditAssertionExpectation) { e.Audience = "scenario:other" },
		"replay operation": func(e *CreditAssertionExpectation) { e.OperationID = "op-2" },
		"source":           func(e *CreditAssertionExpectation) { e.Source = "caller-header" },
	} {
		t.Run(name, func(t *testing.T) {
			wrong := expect
			change(&wrong)
			if _, err := VerifyCreditAssertion(verifier, token, wrong); err == nil {
				t.Fatal("mismatched assertion accepted")
			}
		})
	}
	parts := strings.Split(token, ".")
	parts[1] = parts[1] + "x"
	if _, err := VerifyCreditAssertion(verifier, strings.Join(parts, "."), expect); err == nil {
		t.Fatal("forged assertion accepted")
	}
	verifier.Now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := VerifyCreditAssertion(verifier, token, expect); err == nil {
		t.Fatal("expired assertion accepted")
	}
}

func TestCreditAssertionRequiresExactScopeAndKnownSource(t *testing.T) {
	signer, err := GenerateSigner("lpbs-test", "lpbs.fixture", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	base := CreditAssertionExpectation{BusinessAccountID: "acct-1", ActorID: "actor-1", AppKey: "audio-tools", FeatureID: "audio-tools.summarize", MaxCredits: 500, Audience: CreditGatewayAudience, OperationID: "op-1", Source: CreditAssertionSource}
	for name, mutate := range map[string]func(*CreditAssertionExpectation){"missing app": func(e *CreditAssertionExpectation) { e.AppKey = "" }, "wrong audience": func(e *CreditAssertionExpectation) { e.Audience = "scenario:other" }, "wrong source": func(e *CreditAssertionExpectation) { e.Source = "untrusted" }} {
		t.Run(name, func(t *testing.T) {
			e := base
			mutate(&e)
			if _, _, err := signer.SignCreditAssertion(e, time.Second); err == nil {
				t.Fatal("invalid binding signed")
			}
		})
	}
}

func TestCreditAssertionAdmitsOnlyOwnerDeclaredMeteredSources(t *testing.T) {
	signer, err := GenerateSigner("test-key", "issuer", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	base := CreditAssertionExpectation{BusinessAccountID: "acct", ActorID: "actor", AppKey: "web-console", FeatureID: "agent.session", MaxCredits: 100, Audience: CreditGatewayAudience, OperationID: "op-1"}
	for _, source := range []string{CreditAssertionSourceAIDoor, CreditAssertionSourceMeteredInference} {
		t.Run(source, func(t *testing.T) {
			expect := base
			expect.Source = source
			if _, _, err := signer.SignCreditAssertion(expect, 20*time.Second); err != nil {
				t.Fatalf("owner-declared source was refused: %v", err)
			}
		})
	}
	base.Source = "caller-controlled-header"
	if _, _, err := signer.SignCreditAssertion(base, 20*time.Second); err == nil {
		t.Fatal("caller-controlled source was accepted")
	}
}

func TestCreditAssertionOwnerBindingRejectsWrongFeatureAndOperation(t *testing.T) {
	signer, err := GenerateSigner("owner-test", "issuer", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	expect := CreditAssertionExpectation{BusinessAccountID: "acct", ActorID: "actor", AppKey: "browser-automation-studio", FeatureID: "browser-automation-studio.ai-writing", MaxCredits: 200, Audience: CreditGatewayAudience, OperationID: "op-1", Source: CreditAssertionSourceMeteredInference}
	token, _, err := signer.SignCreditAssertion(expect, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewVerifier(NewKeySet(PublicKey{ID: signer.KeyID, Key: &signer.Private.PublicKey}), "issuer", DefaultLeeway)
	claims, err := verifier.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	valid := CreditAssertionOwnerBinding{AppKey: expect.AppKey, FeatureID: expect.FeatureID, OperationID: expect.OperationID, Source: expect.Source}
	if err := ValidateCreditAssertionOwnerClaims(claims, valid); err != nil {
		t.Fatalf("valid feature owner refused: %v", err)
	}
	for name, mutate := range map[string]func(*CreditAssertionOwnerBinding){
		"wrong app":          func(b *CreditAssertionOwnerBinding) { b.AppKey = "web-console" },
		"wrong feature":      func(b *CreditAssertionOwnerBinding) { b.FeatureID = "web-console.command-generation" },
		"replayed operation": func(b *CreditAssertionOwnerBinding) { b.OperationID = "op-2" },
		"wrong source":       func(b *CreditAssertionOwnerBinding) { b.Source = "caller-header" },
	} {
		t.Run(name, func(t *testing.T) {
			binding := valid
			mutate(&binding)
			if err := ValidateCreditAssertionOwnerClaims(claims, binding); err == nil {
				t.Fatal("owner mismatch was accepted")
			}
		})
	}
}
