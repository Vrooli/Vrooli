package safety

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedReviewProfilesAreValidAndComplete(t *testing.T) {
	profiles, err := EmbeddedReviewProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{ProfileLocal, ProfilePublic} {
		profile, ok := profiles[name]
		if !ok {
			t.Fatalf("missing profile %q", name)
		}
		if err := profile.Validate(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := profile.ClassifierIDs(); len(got) != 2 || got[0] != "falconsai-nsfw" || got[1] != "adamcodd-vit-nsfw" {
			t.Errorf("%s classifier ids = %v", name, got)
		}
		if got := profile.EscalationIDs(); len(got) != 2 || got[0] != "yunet" || got[1] != "fairface-vit-age" {
			t.Errorf("%s escalation ids = %v", name, got)
		}
		if len(profile.SecondOpinion.Models) == 0 || profile.SecondOpinion.JudgeRole == "" {
			t.Errorf("%s second-opinion judge config is incomplete: %+v", name, profile.SecondOpinion)
		}
	}
	local := profiles[ProfileLocal]
	public := profiles[ProfilePublic]
	if local.Mode != ReviewModeEnforce || local.OnRefusal != RefusalDropVariation {
		t.Errorf("local defaults = mode %q, refusal %q", local.Mode, local.OnRefusal)
	}
	if public.Mode != ReviewModeEnforce || public.OnRefusal != RefusalFailJob {
		t.Errorf("public defaults = mode %q, refusal %q", public.Mode, public.OnRefusal)
	}
	if local.QuarantineHours != 168 || public.QuarantineHours != 0 {
		t.Errorf("quarantine defaults = local %dh public %dh, want 168h/0h", local.QuarantineHours, public.QuarantineHours)
	}
}

func TestPolicyForCarriesTheProfileAndSummary(t *testing.T) {
	p := PolicyFor(TierPublic)
	if p.ReviewProfile.Name == "" || len(p.ReviewProfile.Classifiers) == 0 {
		t.Fatalf("policy has no review profile: %+v", p.ReviewProfile)
	}
	if !strings.Contains(p.Summary(), "review profile") || !strings.Contains(p.Summary(), "sexual") {
		t.Errorf("summary does not describe the profile: %q", p.Summary())
	}
}

func TestResolveReviewProfileHonorsOverrides(t *testing.T) {
	raw, err := json.Marshal(profileSet{SchemaVersion: 1, Profiles: map[string]ReviewProfile{
		"custom": {
			SchemaVersion: 1, Name: "custom", Mode: ReviewModeEnforce, OnRefusal: RefusalFailJob,
			Classifiers:   []ClassifierProfile{{ID: "one", Threshold: 0.4}},
			Aggregate:     "any",
			Face:          FaceProfile{ID: "yunet", ScoreThreshold: 0.5, MinPx: 40, CropMargin: 1.35, MaxFaces: 16},
			Age:           AgeProfile{ID: "fairface-vit-age", Threshold: 0.2},
			SecondOpinion: SecondOpinionProfile{Enabled: true, MinAge: 25, MinConfidence: "medium"},
			OnMinorFace:   "refuse", OnUnassessable: UnassessableSecondOpinion,
			MaxImagePixels: 1 << 20, ReviewerVersion: "test",
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	lookup := mapLookup(map[string]string{
		ReviewProfileEnv:     path,
		ReviewProfileNameEnv: "custom",
		ReviewModeEnv:        "shadow",
		ReviewOnRefusalEnv:   "drop_variation",
	})
	got, err := ResolveReviewProfile(TierPublic, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "custom" || got.Mode != ReviewModeShadow || got.OnRefusal != RefusalDropVariation {
		t.Fatalf("resolved profile = %+v", got)
	}
	if len(got.Classifiers) != 1 || got.Classifiers[0].ID != "one" {
		t.Fatalf("classifiers = %+v", got.Classifiers)
	}
}

func TestResolveReviewProfileRejectsMalformedOverrides(t *testing.T) {
	base := DefaultReviewProfile(TierLocal)
	raw, _ := json.Marshal(profileSet{SchemaVersion: 1, Profiles: map[string]ReviewProfile{"local": base}})
	path := filepath.Join(t.TempDir(), "profiles.json")
	_ = os.WriteFile(path, raw, 0o600)

	for name, env := range map[string]map[string]string{
		"unknown profile": {ReviewProfileEnv: path, ReviewProfileNameEnv: "missing"},
		"bad mode":        {ReviewProfileEnv: path, ReviewModeEnv: "maybe"},
		"bad refusal":     {ReviewProfileEnv: path, ReviewOnRefusalEnv: "always"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ResolveReviewProfile(TierLocal, mapLookup(env)); err == nil {
				t.Fatal("malformed override was accepted")
			}
		})
	}
}

func TestDefaultReviewProfileIsACopy(t *testing.T) {
	a := DefaultReviewProfile(TierLocal)
	a.Classifiers[0].Threshold = 0.99
	*a.Classifiers[0].StrictThreshold = 0.98
	b := DefaultReviewProfile(TierLocal)
	if b.Classifiers[0].Threshold == 0.99 || *b.Classifiers[0].StrictThreshold == 0.98 {
		t.Fatal("mutating one profile changed the embedded defaults")
	}
}

func TestReviewProfileValidationRejectsUnsafeConfigurations(t *testing.T) {
	base := DefaultReviewProfile(TierLocal)
	cases := map[string]func(*ReviewProfile){
		"no classifiers": func(p *ReviewProfile) { p.Classifiers = nil },
		"strict above normal": func(p *ReviewProfile) {
			v := 0.9
			p.Classifiers[0].Threshold = 0.2
			p.Classifiers[0].StrictThreshold = &v
		},
		"minor face overridable": func(p *ReviewProfile) { p.OnMinorFace = "flag" },
		"second opinion off":     func(p *ReviewProfile) { p.OnUnassessable = UnassessableSecondOpinion; p.SecondOpinion.Enabled = false },
		"age below adult":        func(p *ReviewProfile) { p.SecondOpinion.MinAge = 17 },
		"low confidence clears":  func(p *ReviewProfile) { p.SecondOpinion.MinConfidence = "low" },
		"zero pixels":            func(p *ReviewProfile) { p.MaxImagePixels = 0 },
		"negative quarantine":    func(p *ReviewProfile) { p.QuarantineHours = -1 },
		"quarantine over a year": func(p *ReviewProfile) { p.QuarantineHours = 8761 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := base
			p.Classifiers = append([]ClassifierProfile(nil), base.Classifiers...)
			mutate(&p)
			if err := p.Validate(); err == nil {
				t.Fatal("unsafe profile was accepted")
			}
		})
	}
}

func TestResolveReviewProfileHonorsTheQuarantineOverride(t *testing.T) {
	lookup := mapLookup(map[string]string{ReviewQuarantineHoursEnv: "24"})
	got, err := ResolveReviewProfile(TierLocal, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if got.QuarantineHours != 24 {
		t.Fatalf("quarantine hours = %d, want 24", got.QuarantineHours)
	}
	if _, err := ResolveReviewProfile(TierLocal, mapLookup(map[string]string{ReviewQuarantineHoursEnv: "-1"})); err == nil {
		t.Fatal("a negative quarantine override was accepted")
	}
}

func TestReviewProfileCanRefuseUnassessableWithoutAnAgeJudge(t *testing.T) {
	p := DefaultReviewProfile(TierLocal)
	p.OnUnassessable = UnassessableRefuse
	p.SecondOpinion = SecondOpinionProfile{}
	if err := p.Validate(); err != nil {
		t.Fatalf("an explicit refuse policy must not need a judge: %v", err)
	}
}

func TestEffectiveThresholdUsesStrictOnlyWhenSet(t *testing.T) {
	strict := 0.1
	c := ClassifierProfile{ID: "x", Threshold: 0.3, StrictThreshold: &strict}
	if got := c.EffectiveThreshold(false); got != 0.3 {
		t.Fatalf("normal threshold = %v", got)
	}
	if got := c.EffectiveThreshold(true); got != 0.1 {
		t.Fatalf("strict threshold = %v", got)
	}
	plain := ClassifierProfile{ID: "x", Threshold: 0.3}
	if got := plain.EffectiveThreshold(true); got != 0.3 {
		t.Fatalf("unset strict threshold = %v, want the normal bar", got)
	}
}

func TestClearsAdultWithProfileConfidence(t *testing.T) {
	j := AgeJudgement{ApparentAge: 30, Confidence: "medium"}
	if !j.ClearsAdultWith(25, "medium") {
		t.Error("medium confidence at the floor should clear a medium profile")
	}
	if j.ClearsAdultWith(25, "high") {
		t.Error("medium confidence must not clear a high-confidence profile")
	}
	if (AgeJudgement{ApparentAge: 40, Confidence: "low"}).ClearsAdultWith(25, "low") {
		t.Error("low confidence must never count as evidence of an adult")
	}
	if (AgeJudgement{ApparentAge: 24, Confidence: "high"}).ClearsAdultWith(25, "medium") {
		t.Error("an age below the floor must stay refused")
	}
}

func TestGateHealthReportsAnUnavailableReviewer(t *testing.T) {
	gate := NewGateFromPolicy(PolicyFor(TierPublic), nil)
	gate.SetMinorReviewer(UnavailableReviewer(os.ErrNotExist))
	if gate.Policy().MinorSafetyDetector {
		t.Fatal("an unavailable reviewer must not be reported as a configured detector")
	}
	health := gate.ReviewerHealth()
	if health.Available || !health.Configured {
		t.Fatalf("health = %+v", health)
	}
	if !strings.Contains(gate.Summary(), "unavailable") {
		t.Fatalf("summary = %q", gate.Summary())
	}
}
