package adapters

import "testing"

func TestSeedDoctorAndLintClean(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if rep := r.DoctorCatalog(); !rep.OK {
		t.Fatalf("seed DoctorCatalog not OK: %+v", rep.Findings)
	}
	if rep := r.RegistryLint(); !rep.OK {
		t.Fatalf("seed RegistryLint not OK: %+v", rep.Findings)
	}
}

func TestDoctorFlagsEnabledWithoutStrategy(t *testing.T) {
	doc := `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"lora","architecture":"sd15","weight":"none","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"yes"},"enabled":true,"ready":false,"pending":"p"}]}`
	r, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rep := r.DoctorCatalog()
	if rep.OK {
		t.Fatal("expected doctor to fail for an enabled adapter with no fetch strategy")
	}
	if !hasCode(rep.Findings, "adapter_without_fetch_strategy") {
		t.Fatalf("expected adapter_without_fetch_strategy, got %+v", rep.Findings)
	}
}

func TestDoctorWarnsDisabledUnpinnedRepo(t *testing.T) {
	doc := `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"controlnet","architecture":"sd15","weight":"none","preprocessor":"canny","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"conditional","commercial_use_notes":"n"},"enabled":false,"ready":false,"pending":"p","source":{"repo":{"repo_id":"r","revision":""}}}]}`
	r, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rep := r.DoctorCatalog()
	if !rep.OK {
		t.Fatalf("disabled unpinned repo should be a warning, not error: %+v", rep.Findings)
	}
	f := findCode(rep.Findings, "repo_source_without_pinned_revision")
	if f == nil {
		t.Fatalf("expected repo_source_without_pinned_revision warning, got %+v", rep.Findings)
	}
	if f.Severity != FindingWarning {
		t.Fatalf("expected warning severity, got %q", f.Severity)
	}
}

func hasCode(fs []CatalogFinding, code string) bool { return findCode(fs, code) != nil }

func findCode(fs []CatalogFinding, code string) *CatalogFinding {
	for i := range fs {
		if fs[i].Code == code {
			return &fs[i]
		}
	}
	return nil
}

// A restrictive adapter license is informational for normal use: the shared
// rights evaluator allows every license class locally and keeps monetized,
// hosted and redistributed use refused, so it never fails the catalog.
func TestDoctorTreatsRestrictiveLicenseAsInformational(t *testing.T) {
	for _, tc := range []struct{ commercialUse, code string }{
		{"conditional", "enabled_conditional_commercial_use"},
		{"no", "enabled_non_commercial_license"},
	} {
		doc := `{"schema_version":"1","adapters":[{"id":"x","name":"X","kind":"controlnet","architecture":"sd15","weight":"none","preprocessor":"canny","scale_range":{"min":0,"max":1,"default":0.5},"capability_labels":{"commercial_use":"` + tc.commercialUse + `","commercial_use_notes":"n"},"enabled":true,"ready":true,"source":{"repo":{"repo_id":"r","revision":"0123456789abcdef0123456789abcdef01234567"}}}]}`
		r, err := Parse([]byte(doc))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		rep := r.DoctorCatalog()
		f := findCode(rep.Findings, tc.code)
		if f == nil || f.Severity != FindingInfo {
			t.Fatalf("%s: expected informational %s, got %+v", tc.commercialUse, tc.code, rep.Findings)
		}
		if !rep.OK {
			t.Fatalf("%s: a license finding must not fail the catalog: %+v", tc.commercialUse, rep.Findings)
		}
	}
}
