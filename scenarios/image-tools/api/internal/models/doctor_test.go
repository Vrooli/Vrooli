package models

import "testing"

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
	  "blocklist": [{"id": "conditional", "operations": ["naturalize"], "license": "test", "reason": "blocked"}]
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
		"seed_blocklist_overlap",
		"operation_without_installable_enabled_model",
	} {
		if !codes[code] {
			t.Fatalf("expected doctor code %q in findings: %+v", code, report.Findings)
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
		  "blocklist": [{"id": "x", "operations": ["edit_instruct"], "license": "t", "reason": "r"}]
		}`
	}
	cases := []struct {
		name   string
		family string
		minRT  string
		want   string
	}{
		{"no family", "", "", "enabled_edit_model_without_family"},
		{"not ready family", "flux-2-klein", "diffusers>=1", "enabled_edit_model_family_not_ready"},
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
