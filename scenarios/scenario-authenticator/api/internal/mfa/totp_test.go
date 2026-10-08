package mfa

import (
	"strings"
	"testing"
	"time"
)

func TestTOTPAllowsBoundedClockSkewOnly(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Unix(1_700_000_000, 0)
	code, err := Code(secret, now)
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(secret, code, now) || !Verify(secret, code, now.Add(30*time.Second)) {
		t.Fatal("expected current and adjacent TOTP windows to verify")
	}
	old, err := Code(secret, now.Add(-60*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if Verify(secret, old, now) {
		t.Fatal("two windows of clock skew must be rejected")
	}
	if Verify(secret, strings.TrimSpace(code)+"0", now) {
		t.Fatal("malformed code must be rejected")
	}
}

func TestRecoveryCodesHashAndReplayPreparation(t *testing.T) {
	codes, err := GenerateRecoveryCodes(4)
	if err != nil || len(codes) != 4 {
		t.Fatalf("generate recovery codes: %v", err)
	}
	hash, err := HashRecoveryCode(codes[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(hash, codes[0]) || !VerifyRecoveryCode(codes[0], hash) {
		t.Fatal("recovery code must verify only through its stored hash")
	}
	if VerifyRecoveryCode(codes[1], hash) {
		t.Fatal("a different recovery code must fail")
	}
	set, err := NewRecoverySet(codes)
	if err != nil {
		t.Fatal(err)
	}
	if !set.Consume(codes[0]) || set.Consume(codes[0]) {
		t.Fatal("recovery codes must be single-use")
	}
}

func TestProvisioningURI(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	uri, err := ProvisioningURI("Vrooli", "owner@example.com", secret)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"otpauth://totp/", "secret=" + secret, "issuer=Vrooli"} {
		if !strings.Contains(uri, part) {
			t.Fatalf("provisioning URI missing %q: %s", part, uri)
		}
	}
}
