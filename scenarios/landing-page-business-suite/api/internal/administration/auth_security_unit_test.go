package administration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTOTPMatchesRFC6238Vectors(t *testing.T) {
	// RFC 6238 appendix B SHA-1 secret "12345678901234567890", truncated to six digits.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	for _, vector := range []struct {
		unix int64
		code string
	}{{59, "287082"}, {1111111109, "081804"}, {1234567890, "005924"}, {2000000000, "279037"}} {
		got, err := TOTPCode(secret, time.Unix(vector.unix, 0))
		if err != nil || got != vector.code {
			t.Fatalf("TOTP at %d = %q, %v; want %q", vector.unix, got, err, vector.code)
		}
	}
}

func TestTOTPAcceptsOneStepSkewButNeverReplaysAStep(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Unix(1111111109, 0)
	previous, _ := TOTPCode(secret, now.Add(-30*time.Second))
	step, ok := matchTOTP(secret, previous, now, -1)
	if !ok {
		t.Fatal("previous-step code rejected")
	}
	if _, ok := matchTOTP(secret, previous, now, step); ok {
		t.Fatal("same step accepted twice")
	}
	stale, _ := TOTPCode(secret, now.Add(-2*time.Minute))
	if _, ok := matchTOTP(secret, stale, now, -1); ok {
		t.Fatal("code outside the skew window accepted")
	}
}

func TestRecoveryCodesAreUniqueAndNormalize(t *testing.T) {
	codes, hashes, err := newRecoveryCodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != recoveryCodeCount || len(hashes) != recoveryCodeCount {
		t.Fatalf("codes=%d hashes=%d", len(codes), len(hashes))
	}
	seen := map[string]bool{}
	for _, code := range codes {
		if seen[code] || len(normalizeRecoveryCode(strings.ToUpper(" "+code+" "))) != 10 {
			t.Fatalf("bad recovery code %q", code)
		}
		seen[code] = true
	}
}

func TestSignInContextOnlyAcceptsLocalCallbacks(t *testing.T) {
	for _, raw := range []string{
		`{"redirect_uri":"https://evil.example/cb"}`,
		`{"redirect_uri":"http://192.168.1.4:3000/cb"}`,
		`{"redirect_uri":"javascript:alert(1)"}`,
		`["not","an","object"]`,
		`{"app":"` + strings.Repeat("x", maxSignInContextSize) + `"}`,
	} {
		if _, err := normalizeSignInContext(json.RawMessage(raw)); !errors.Is(err, ErrContextInvalid) {
			t.Fatalf("context %.60s accepted", raw)
		}
	}
	for _, raw := range []string{`{"redirect_uri":"http://127.0.0.1:4100/cb","desktop_link":true}`, `{"redirect_uri":"vrooli://auth"}`, ``} {
		if _, err := normalizeSignInContext(json.RawMessage(raw)); err != nil {
			t.Fatalf("context %s rejected: %v", raw, err)
		}
	}
}

func TestSignInFlowAndEmailHelpers(t *testing.T) {
	if signInFlow(json.RawMessage(`{"desktop_link":true,"redirect_uri":"http://127.0.0.1:1/cb"}`)) != SignInFlowDesktopLink ||
		signInFlow(json.RawMessage(`{"redirect_uri":"http://127.0.0.1:1/cb"}`)) != SignInFlowNativeApp ||
		signInFlow(nil) != SignInFlowBrowser {
		t.Fatal("flow detection is wrong")
	}
	if MaskEmail("jordan@example.com") != "jo•••@example.com" || MaskEmail("al@x.io") != "a•••@x.io" {
		t.Fatalf("masks: %q %q", MaskEmail("jordan@example.com"), MaskEmail("al@x.io"))
	}
	for email, want := range map[string]bool{"a@b.co": true, "a@b": false, "@b.co": false, "a b@c.co": false, "a@b.co.": false} {
		if LooksLikeEmail(email) != want {
			t.Fatalf("LooksLikeEmail(%q) != %t", email, want)
		}
	}
	code, err := randomDigits(6)
	if err != nil || len(code) != 6 || strings.Trim(code, "0123456789") != "" {
		t.Fatalf("code=%q err=%v", code, err)
	}
	if bindingHash("short") != nil {
		t.Fatal("weak browser binding accepted")
	}
}

func TestAuthThrottleFallbackBlocksWithoutStore(t *testing.T) {
	throttle := NewAuthThrottle(nil)
	now := time.Unix(1000, 0)
	throttle.UseClock(func() time.Time { return now })
	rule := ThrottleRule{Limit: 2, Window: time.Minute}
	for i := 0; i < 2; i++ {
		if ok, _ := throttle.Allow(context.Background(), "b", rule); !ok {
			t.Fatalf("attempt %d blocked", i)
		}
	}
	if ok, _ := throttle.Allow(context.Background(), "b", rule); ok {
		t.Fatal("third attempt allowed")
	}
	now = now.Add(2 * time.Minute)
	if ok, _ := throttle.Allow(context.Background(), "b", rule); !ok {
		t.Fatal("window did not recover")
	}
}
