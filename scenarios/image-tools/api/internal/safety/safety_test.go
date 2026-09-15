package safety

import (
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
)

func TestParseTier(t *testing.T) {
	cases := map[string]Tier{
		"":           TierLocal,
		"local":      TierLocal,
		"personal":   TierLocal,
		"PUBLIC":     TierPublic,
		" public ":   TierPublic,
		"prod":       TierPublic,
		"production": TierPublic,
		"monetized":  TierPublic,
		"saas":       TierPublic,
		"nonsense":   TierLocal,
	}
	for in, want := range cases {
		if got := ParseTier(in); got != want {
			t.Errorf("ParseTier(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveTierEnv(t *testing.T) {
	t.Setenv(TierEnv, "public")
	if got := ResolveTier(); got != TierPublic {
		t.Errorf("ResolveTier with env=public = %q, want public", got)
	}
	t.Setenv(TierEnv, "")
	if got := ResolveTier(); got != TierLocal {
		t.Errorf("ResolveTier with empty env = %q, want local", got)
	}
}

func TestOpWeight(t *testing.T) {
	cases := map[string]Weight{
		"naturalize":         WeightLow,
		"edit_instruct":      WeightHigh,
		"inpaint":            WeightHigh,
		"outpaint":           WeightHigh,
		"object_removal":     WeightHigh,
		"background_replace": WeightHigh,
		"image_to_image":     WeightHigh,
		"text_to_image":      WeightNone,
		"upscale":            WeightNone,
		"background_removal": WeightNone,
		"denoise":            WeightNone,
		"unknown_op":         WeightNone,
	}
	for op, want := range cases {
		if got := OpWeight(op); got != want {
			t.Errorf("OpWeight(%q) = %q, want %q", op, got, want)
		}
	}
	// OpWeights returns a copy — mutating it must not affect the canonical table.
	w := OpWeights()
	w["naturalize"] = WeightHigh
	if OpWeight("naturalize") != WeightLow {
		t.Error("OpWeights() must return a defensive copy")
	}
}

func TestPolicyFor(t *testing.T) {
	local := PolicyFor(TierLocal)
	if local.RequireConsent || local.ForceNSFWScan || local.RequireProvenance || local.RateLimitPerMin != 0 {
		t.Errorf("local policy should be fully permissive, got %+v", local)
	}
	pub := PolicyFor(TierPublic)
	if !pub.RequireConsent || !pub.ForceNSFWScan || !pub.RequireProvenance || pub.RateLimitPerMin <= 0 {
		t.Errorf("public policy should enforce all gates, got %+v", pub)
	}
	if local.Summary() == "" || pub.Summary() == "" {
		t.Error("policy summaries should be non-empty")
	}
}

func TestEvaluateLocalUnrestricted(t *testing.T) {
	p := PolicyFor(TierLocal)
	for _, op := range []string{"edit_instruct", "inpaint", "naturalize", "text_to_image"} {
		d := p.Evaluate(op, false)
		if !d.Allowed {
			t.Errorf("local Evaluate(%q) blocked; local should be unrestricted", op)
		}
		if d.ForceNSFWScan {
			t.Errorf("local Evaluate(%q) forced scan; local should not", op)
		}
		if d.RecordConsent {
			t.Errorf("local Evaluate(%q) recorded consent; local should not", op)
		}
	}
}

func TestEvaluatePublicHighWeightBlockedWithoutConsent(t *testing.T) {
	p := PolicyFor(TierPublic)
	d := p.Evaluate("edit_instruct", false)
	if d.Allowed {
		t.Fatal("public high-weight op without consent should be blocked")
	}
	if d.Reason == "" || d.RecoveryHint == "" {
		t.Error("a block must carry an actionable reason + recovery hint")
	}
	if d.Weight != WeightHigh {
		t.Errorf("weight = %q, want high", d.Weight)
	}
}

func TestEvaluatePublicHighWeightAllowedWithConsent(t *testing.T) {
	p := PolicyFor(TierPublic)
	d := p.Evaluate("inpaint", true)
	if !d.Allowed {
		t.Fatal("public high-weight op WITH consent should be allowed")
	}
	if !d.RecordConsent {
		t.Error("an allowed, affirmed high-weight op should be recorded")
	}
	if !d.ForceNSFWScan {
		t.Error("public tier should force the NSFW scan")
	}
}

func TestEvaluatePublicLowAndNoneWeightNeverGated(t *testing.T) {
	p := PolicyFor(TierPublic)
	// Naturalize (low) and text_to_image (none) never need consent, even public.
	for _, op := range []string{"naturalize", "text_to_image", "upscale"} {
		d := p.Evaluate(op, false)
		if !d.Allowed {
			t.Errorf("public Evaluate(%q) blocked; only high-weight ops are gated", op)
		}
		if d.RecordConsent {
			t.Errorf("public Evaluate(%q) recorded consent; only high-weight affirmed ops record", op)
		}
		if !d.ForceNSFWScan {
			t.Errorf("public Evaluate(%q) should still force the scan", op)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	base := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	cur := base
	rl := NewRateLimiter(3).withClock(func() time.Time { return cur })
	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Fatalf("event %d should be allowed (limit 3)", i)
		}
	}
	if rl.Allow() {
		t.Error("4th event within the window should be blocked")
	}
	// Advance past the window — the bucket clears.
	cur = base.Add(61 * time.Second)
	if !rl.Allow() {
		t.Error("after the window elapses the limiter should allow again")
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	rl := NewRateLimiter(0)
	for i := 0; i < 100; i++ {
		if !rl.Allow() {
			t.Fatal("a 0-limit limiter must always allow")
		}
	}
	var nilRL *RateLimiter
	if !nilRL.Allow() {
		t.Error("a nil limiter must allow")
	}
}

func TestGateAndConsentLog(t *testing.T) {
	d := db.NewSQLite(t)
	if _, err := d.Exec(Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	logStore := NewConsentLog(d)
	gate := NewGate(TierPublic, logStore)

	if gate.Policy().Tier != TierPublic {
		t.Fatalf("gate tier = %q, want public", gate.Policy().Tier)
	}

	// A blocked op must not record consent.
	if d := gate.Evaluate("edit_instruct", false); d.Allowed {
		t.Fatal("expected block without consent")
	}

	// An allowed, affirmed high-weight op records.
	dec := gate.Evaluate("edit_instruct", true)
	if !dec.Allowed || !dec.RecordConsent {
		t.Fatalf("expected allowed+record, got %+v", dec)
	}
	if err := gate.RecordConsent(t.Context(), "edit_instruct", dec.Weight); err != nil {
		t.Fatalf("RecordConsent: %v", err)
	}
	n, err := logStore.Count(t.Context())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("consent log count = %d, want 1", n)
	}

	if !gate.AllowRate() {
		t.Error("first submit should pass the rate limiter")
	}
}

func TestConsentLogNilSafe(t *testing.T) {
	var l *ConsentLog
	if err := l.Record(t.Context(), "inpaint", WeightHigh, TierPublic); err != nil {
		t.Errorf("nil consent log Record should be a no-op, got %v", err)
	}
	if n, err := l.Count(t.Context()); err != nil || n != 0 {
		t.Errorf("nil consent log Count = (%d, %v), want (0, nil)", n, err)
	}
}
