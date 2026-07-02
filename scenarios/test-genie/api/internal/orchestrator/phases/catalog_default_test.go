package phases

import "testing"

func TestValidPhaseNamesMatchCatalog(t *testing.T) {
	catalog := DefaultCatalog()
	want := make([]string, 0, len(catalog.All()))
	for _, spec := range catalog.All() {
		want = append(want, spec.Name.String())
	}

	got := ValidPhaseNames()
	if len(got) != len(want) {
		t.Fatalf("ValidPhaseNames length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ValidPhaseNames[%d] = %q, want %q (catalog order must be preserved)", i, got[i], want[i])
		}
	}
}

func TestAllPhasesMirrorsValidPhaseNames(t *testing.T) {
	names := AllPhases()
	strs := ValidPhaseNames()
	if len(names) != len(strs) {
		t.Fatalf("AllPhases length = %d, ValidPhaseNames length = %d", len(names), len(strs))
	}
	for i := range names {
		if names[i].String() != strs[i] {
			t.Fatalf("AllPhases[%d] = %q, ValidPhaseNames[%d] = %q", i, names[i], i, strs[i])
		}
	}
}

func TestIsValidPhase(t *testing.T) {
	for _, name := range ValidPhaseNames() {
		if !IsValidPhase(name) {
			t.Errorf("IsValidPhase(%q) = false, want true", name)
		}
		// Case-insensitive lookup must hold for every canonical phase.
		if !IsValidPhase(upper(name)) {
			t.Errorf("IsValidPhase(%q) = false, want true (case-insensitive)", upper(name))
		}
	}
	for _, bad := range []string{"", "  ", "nonexistent-phase", "all"} {
		if IsValidPhase(bad) {
			t.Errorf("IsValidPhase(%q) = true, want false", bad)
		}
	}
}

func TestNormalizeKeyResolvesAliases(t *testing.T) {
	tests := map[string]string{
		" E2E ":      "workflow",
		"unit-test":  "unit",
		"playbook":   "workflow",
		"playbooks":  "workflow",
		"STRUCT":     "structure",
		"custom-one": "custom-one",
	}
	for input, want := range tests {
		if got := NormalizeKey(input); got != want {
			t.Fatalf("NormalizeKey(%q) = %q, want %q", input, got, want)
		}
	}
	name, ok := NormalizeName("e2e")
	if !ok {
		t.Fatal("NormalizeName(e2e) returned !ok")
	}
	if name != Workflow {
		t.Fatalf("NormalizeName(e2e) = %q, want %q", name, Workflow)
	}
}

func upper(s string) string {
	out := []byte(s)
	for i := range out {
		if out[i] >= 'a' && out[i] <= 'z' {
			out[i] -= 32
		}
	}
	return string(out)
}
