package verdicts_test

import (
	"strings"
	"testing"

	"image-tools/internal/models"
	"image-tools/internal/verdicts"
)

// TestEvidenceSeedMatchesTheCatalog is the conformance gate for the evidence
// seed: every claim names a real model and an operation that model serves, so a
// renamed model or op cannot leave an orphaned claim behind.
func TestEvidenceSeedMatchesTheCatalog(t *testing.T) {
	seed, err := verdicts.LoadSeed()
	if err != nil {
		t.Fatalf("load seed: %v", err)
	}
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if len(seed.Claims) == 0 {
		t.Fatal("the evidence seed has no claims")
	}
	for _, c := range seed.Claims {
		m, ok := registry.ByID(c.ModelID)
		if !ok {
			t.Errorf("claim %s/%s: model %q is not in the registry", c.ModelID, c.Operation, c.ModelID)
			continue
		}
		if c.Operation != verdicts.AllOperations && !m.ServesOperation(c.Operation) {
			t.Errorf("claim %s/%s: %q does not serve operation %q (serves %s)", c.ModelID, c.Operation, c.ModelID, c.Operation, strings.Join(m.Operations, ", "))
		}
	}
}

func TestSeedKnownVerdicts(t *testing.T) {
	seed, err := verdicts.LoadSeed()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		model, op string
		want      verdicts.Grade
	}{
		{"flux-2-klein-4b", "edit_instruct", verdicts.Recommended},
		{"flux-2-klein-4b", "text_to_image", verdicts.Recommended},
		{"flux-2-klein-4b", "image_to_image", verdicts.Recommended},
		{"instruct-pix2pix", "edit_instruct", verdicts.NotRecommended},
		{"sd-1.5", "text_to_image", verdicts.NotRecommended},
		{"sd-1.5", "image_to_image", verdicts.NotRecommended},
		{"depth-anything-v2-large", "depth_map", verdicts.Recommended},
		{"restoreformer-plus-plus", "face_restore", verdicts.Recommended},
		{"codeformer", "face_restore", verdicts.Works},
		{"openrouter-image", "edit_instruct", verdicts.Works},
	}
	for _, tc := range cases {
		v := verdicts.Derive(seed.ClaimsFor(tc.model, tc.op), verdicts.Signals{}, verdicts.DefaultPolicy)
		if v.Grade != tc.want {
			t.Errorf("%s/%s = %s, want %s", tc.model, tc.op, v.Grade, tc.want)
		}
	}
}

func TestParseSeedRejectsIncompleteClaims(t *testing.T) {
	const head = `{"schema_version":1,"claims":[`
	good := `"model_id":"m","operation":"op","grade":"works","reason":"r","kind":"qualification","ref":"x","date":"2026-11-21"`
	cases := map[string]string{
		"untested grade":    strings.Replace(good, `"works"`, `"untested"`, 1),
		"unknown grade":     strings.Replace(good, `"works"`, `"great"`, 1),
		"no reason":         strings.Replace(good, `"reason":"r"`, `"reason":""`, 1),
		"multi-line reason": strings.Replace(good, `"reason":"r"`, `"reason":"a\nb"`, 1),
		"no ref":            strings.Replace(good, `"ref":"x"`, `"ref":""`, 1),
		"live kind":         strings.Replace(good, `"qualification"`, `"job_history"`, 1),
		"bad date":          strings.Replace(good, `2026-11-21`, `21/11/2026`, 1),
		"no operation":      strings.Replace(good, `"operation":"op"`, `"operation":""`, 1),
	}
	for name, claim := range cases {
		if _, err := verdicts.ParseSeed([]byte(head + "{" + claim + "}]}")); err == nil {
			t.Errorf("%s: seed was accepted", name)
		}
	}
	if _, err := verdicts.ParseSeed([]byte(head + "{" + good + "}]}")); err != nil {
		t.Fatalf("a complete claim was rejected: %v", err)
	}
}
