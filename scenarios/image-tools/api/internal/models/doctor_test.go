package models

import (
	"strings"
	"testing"
)

func TestDoctorTreatsDisabledExperimentalOperationAsUnavailableWarning(t *testing.T) {
	const seed = `{"schema_version":"1.0.0","models":[{"id":"experimental","name":"Experimental","operations":["image_to_object"],"default_for":["image_to_object"],"tier":"default","backend":"python-sidecar","hardware":{"cpu_capable":true},"capability_labels":{"commercial_use":"yes"},"enabled":false,"source":{"assets":[{"url":"https://example.test/model.ckpt","filename":"model.ckpt","kind":"checkpoint","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","min_bytes":1}]}}]}`
	r, err := Parse([]byte(seed))
	if err != nil {
		t.Fatal(err)
	}
	report := r.DoctorCatalog()
	found := false
	for _, finding := range report.Findings {
		if finding.Operation == "image_to_object" {
			found = true
			if finding.Severity != FindingWarning || !strings.Contains(finding.Message, "intentionally unavailable") {
				t.Fatalf("unexpected finding: %+v", finding)
			}
		}
	}
	if !found {
		t.Fatal("expected an explicit unavailable warning")
	}
}

func TestDoctorCatalogReportsActionableFindings(t *testing.T) {
	const seed = `{
	  "schema_version": "1.0.0",
	  "models": [
	    {
	      "id": "missing-assets", "name": "Missing Assets", "operations": ["upscale"],
	      "default_for": ["upscale"], "tier": "default", "backend": "onnxruntime",
	      "hardware": {"cpu_capable": true}, "capability_labels": {"commercial_use": "yes"}, "enabled": true
	    },
	    {
	      "id": "bad-assets", "name": "Bad Assets", "operations": ["segment"],
	      "default_for": ["segment"], "tier": "default", "backend": "onnxruntime",
	      "hardware": {"cpu_capable": true}, "capability_labels": {"commercial_use": "yes"}, "enabled": true,
	      "source": {"assets": [{"url": "", "filename": "", "kind": "", "sha256": "abc", "min_bytes": 0}],
	                 "checksum": {"algo": "md5", "value": "abc", "status": "pinned"}}
	    },
	    {
	      "id": "conditional", "name": "Conditional", "operations": ["naturalize"],
	      "default_for": ["naturalize"], "tier": "default", "backend": "builtin",
	      "hardware": {"cpu_capable": true},
	      "capability_labels": {"commercial_use": "conditional", "commercial_use_notes": "requires review"},
	      "enabled": true
	    },
	    {
	      "id": "empty-pinned-checksum", "name": "Empty Pinned Checksum", "operations": ["upscale"],
	      "default_for": [], "tier": "nice-to-have", "backend": "onnxruntime",
	      "hardware": {"cpu_capable": true}, "capability_labels": {"commercial_use": "yes"}, "enabled": false,
	      "source": {"checksum": {"algo": "sha256", "status": "pinned"}}
	    }
	  ],
	  "blocklist": [{"id": "conditional", "operations": ["naturalize"], "license": "test", "reason": "blocked", "gate": "exclusion", "exclusion_kind": "technical"}]
	}`
	r, err := Parse([]byte(seed))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	report := r.DoctorCatalog()
	if report.OK {
		t.Fatalf("doctor should fail for malformed installability/policy state")
	}
	codes := make(map[string]bool)
	for _, f := range report.Findings {
		codes[f.Code] = true
	}
	for _, code := range []string{
		"enabled_model_without_assets",
		"asset_missing_url",
		"asset_missing_filename",
		"asset_missing_kind",
		"asset_missing_min_bytes",
		"asset_bad_sha256",
		"checksum_algo_not_sha256",
		"checksum_bad_sha256",
		"checksum_pinned_without_value",
		"enabled_conditional_commercial_use",
		"seed_exclusion_overlap",
		"operation_without_installable_enabled_model",
	} {
		if !codes[code] {
			t.Fatalf("expected doctor code %q in findings: %+v", code, report.Findings)
		}
	}
}

func TestDoctorRejectsNonHexSHA256WithCorrectLength(t *testing.T) {
	seed := `{"schema_version":"1.0.0","models":[{"id":"bad-hash","name":"Bad Hash","operations":["upscale"],"default_for":["upscale"],"tier":"default","backend":"onnxruntime","hardware":{"cpu_capable":true},"capability_labels":{"commercial_use":"yes"},"enabled":true,"source":{"assets":[{"url":"https://example.test/model.onnx","filename":"model.onnx","kind":"onnx","sha256":"` + strings.Repeat("g", 64) + `","min_bytes":1}],"checksum":{"algo":"sha256","value":"` + strings.Repeat("g", 64) + `","status":"pinned"}}}]}`
	r, err := Parse([]byte(seed))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	report := r.DoctorCatalog()
	codes := make(map[string]bool)
	for _, finding := range report.Findings {
		codes[finding.Code] = true
	}
	for _, code := range []string{"asset_bad_sha256", "checksum_bad_sha256"} {
		if !codes[code] {
			t.Fatalf("doctor accepted 64-character non-hex digest; missing %q in %+v", code, report.Findings)
		}
	}
}

// TestDoctorDiffusersEditRunnableInvariant pins the Phase-4 "enabled ⇒ runnable"
// invariant for diffusers edit_instruct models: an enabled such model must name a
// registered, proven family adapter and declare a min_runtime. Each failure mode
// is asserted on a deliberately-broken enabled fixture.
func TestDoctorDiffusersEditRunnableInvariant(t *testing.T) {
	base := func(id, family, minRuntime string) string {
		runtime := ""
		if family != "" || minRuntime != "" {
			runtime = `, "runtime": {"family": "` + family + `", "min_runtime": "` + minRuntime + `"}`
		}
		return `{
		  "schema_version": "1.0.0",
		  "models": [{
		    "id": "` + id + `", "name": "Edit", "operations": ["edit_instruct"],
		    "default_for": ["edit_instruct"], "tier": "quality", "backend": "diffusers",
		    "hardware": {"gpu_required": true}, "capability_labels": {"commercial_use": "yes"},
		    "source": {"local_path": "/tmp/x"}, "enabled": true` + runtime + `
		  }],
		  "blocklist": [{"id": "x", "operations": ["edit_instruct"], "license": "t", "reason": "r", "gate": "license", "gated_uses": ["monetized_use"]}]
		}`
	}
	cases := []struct {
		name   string
		family string
		minRT  string
		want   string
	}{
		{"no family", "", "", "enabled_edit_model_without_family"},
		{"not ready family", "longcat-image-edit", "diffusers>=1", "enabled_edit_model_family_not_ready"},
		{"no min_runtime", "instruct-pix2pix", "", "enabled_edit_model_without_min_runtime"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Parse([]byte(base("edit-x", tc.family, tc.minRT)))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			var got []string
			for _, f := range r.DoctorCatalog().Findings {
				if f.ModelID == "edit-x" {
					got = append(got, f.Code)
				}
			}
			if !containsCode(got, tc.want) {
				t.Fatalf("expected finding %q, got %v", tc.want, got)
			}
		})
	}

	// A correctly-declared enabled edit model (ready family + min_runtime) draws no
	// runnable-invariant finding.
	r, err := Parse([]byte(base("good-edit", "instruct-pix2pix", "diffusers>=0.21.0")))
	if err != nil {
		t.Fatalf("Parse good: %v", err)
	}
	for _, f := range r.DoctorCatalog().Findings {
		if f.ModelID == "good-edit" && f.Code != "checksum_pinned_without_value" {
			t.Errorf("unexpected finding on a well-formed edit model: %+v", f)
		}
	}
}

// A restrictive license on an enabled model is informational under the shared
// rights evaluator: normal local use is allowed for every license class, and
// only monetized, hosted and redistributed use stay refused. A license gate on a
// seeded row whose rights would grant the gated uses is a catalog error.
func TestDoctorLicenseFindingsFollowSharedRightsEvaluator(t *testing.T) {
	seed := `{
	  "schema_version": "1.0.0",
	  "models": [
	    {"id": "nc", "name": "NC", "operations": ["naturalize"], "default_for": ["naturalize"], "tier": "default", "backend": "builtin",
	     "hardware": {"cpu_capable": true}, "capability_labels": {"commercial_use": "no", "commercial_use_notes": "non-commercial"},
	     "rights": {"artifact_id": "nc", "classification": "prohibited"}, "enabled": true},
	    {"id": "cond", "name": "Cond", "operations": ["naturalize"], "default_for": [], "tier": "default", "backend": "builtin",
	     "hardware": {"cpu_capable": true}, "capability_labels": {"commercial_use": "conditional", "commercial_use_notes": "review"},
	     "rights": {"artifact_id": "cond", "classification": "unknown"}, "enabled": true},
	    {"id": "mislabelled", "name": "Mislabelled", "operations": ["naturalize"], "default_for": [], "tier": "default", "backend": "builtin",
	     "hardware": {"cpu_capable": true}, "capability_labels": {"commercial_use": "yes"},
	     "rights": {"artifact_id": "mislabelled", "classification": "unrestricted"}, "enabled": true}
	  ],
	  "blocklist": [
	    {"id": "nc", "operations": ["naturalize"], "license": "NC", "reason": "non-commercial", "gate": "license", "gated_uses": ["monetized_use", "hosted_inference"]},
	    {"id": "mislabelled", "operations": ["naturalize"], "license": "NC", "reason": "non-commercial", "gate": "license", "gated_uses": ["monetized_use"]}
	  ]
	}`
	r, err := Parse([]byte(seed))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	report := r.DoctorCatalog()
	got := map[string]CatalogFinding{}
	for _, f := range report.Findings {
		got[f.ModelID+"/"+f.Code] = f
	}
	for _, key := range []string{"nc/enabled_non_commercial_license", "cond/enabled_conditional_commercial_use"} {
		f, ok := got[key]
		if !ok {
			t.Fatalf("missing %s in %+v", key, report.Findings)
		}
		if f.Severity != FindingInfo || !strings.Contains(f.Message, "normal local use is allowed") {
			t.Fatalf("%s should be informational for normal use, got %+v", key, f)
		}
	}
	if _, ok := got["nc/license_gate_rights_disagree"]; ok {
		t.Fatalf("a prohibited row agrees with its license gate: %+v", report.Findings)
	}
	if f, ok := got["mislabelled/license_gate_rights_disagree"]; !ok || f.Severity != FindingError {
		t.Fatalf("a commercial-labelled row under a license gate must be an error, got %+v", report.Findings)
	}
	if report.OK {
		t.Fatal("doctor must fail while a license gate disagrees with its row's rights")
	}
}
