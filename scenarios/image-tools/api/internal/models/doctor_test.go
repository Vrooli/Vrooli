package models

import "testing"

func TestDoctorCatalogReportsActionableFindings(t *testing.T) {
	const seed = `{
	  "schema_version": "1.0.0",
	  "operations_vocabulary": ["upscale", "segment", "naturalize"],
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
