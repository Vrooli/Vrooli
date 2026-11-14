package profile

import "testing"

// [REQ:SH-GT-002]
func TestDeriveDefaultProfile(t *testing.T) {
	f := Facts{
		Scenario: "demo",
		Surfaces: []Surface{
			{ID: "api", Kind: "api", Language: "go"},
			{ID: "ui", Kind: "ui", Framework: "react-vite"},
			{ID: "cli", Kind: "cli", Language: "go"},
		},
	}
	p := Derive(f)
	if p.ID != DefaultProfileID {
		t.Fatalf("id = %q, want %q", p.ID, DefaultProfileID)
	}
	if !p.Recognized {
		t.Fatal("default profile must be recognized")
	}
	if p.BackendLanguage != "go" || p.UIFramework != "react-vite" {
		t.Fatalf("backend/ui = %q/%q", p.BackendLanguage, p.UIFramework)
	}
	if !p.HasUI() {
		t.Fatal("HasUI must be true")
	}
}

// [REQ:SH-GT-002]
func TestDeriveAPIOnlyProfile(t *testing.T) {
	f := Facts{Surfaces: []Surface{{ID: "api", Kind: "api", Language: "go"}}}
	p := Derive(f)
	if p.HasUI() {
		t.Fatal("api-only profile must not report UI")
	}
	if p.Recognized {
		t.Fatalf("api-only profile %q should be unrecognized (advisory)", p.ID)
	}
	if p.ID != "go" {
		t.Fatalf("id = %q, want go", p.ID)
	}
}

// [REQ:SH-GT-002] [REQ:SH-PROF-002]
func TestDeriveNonGoBackendUnrecognized(t *testing.T) {
	f := Facts{Surfaces: []Surface{
		{ID: "api", Kind: "api", Language: "python"},
		{ID: "ui", Kind: "ui", Framework: "vue"},
	}}
	p := Derive(f)
	if p.Recognized {
		t.Fatal("python/vue profile must be unrecognized → advisory")
	}
	if p.ID != "vue-python" {
		t.Fatalf("id = %q, want vue-python", p.ID)
	}
}
