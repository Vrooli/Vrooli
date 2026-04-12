package repocontract

import "testing"

func TestResolveProfile(t *testing.T) {
	contract := mustLoadDefault(t, repoRoot(t))

	resolved, err := contract.ResolveProfile("mini_vrooli_bundle", ResolveParams{
		Values: map[string]string{
			"scenario": "scenario-to-cloud",
		},
		Lists: map[string][]string{
			"resources": {"postgres", "redis"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}

	if !contains(resolved.Include, "scenarios/scenario-to-cloud") {
		t.Fatalf("resolved include missing scenario path: %v", resolved.Include)
	}
	if !contains(resolved.Include, "resources/postgres") || !contains(resolved.Include, "resources/redis") {
		t.Fatalf("resolved include missing resource paths: %v", resolved.Include)
	}
	if !contains(resolved.OptionalInclude, "go.mod") {
		t.Fatalf("resolved optional include missing go.mod: %v", resolved.OptionalInclude)
	}
}

func TestResolveProfileValidationBoundaries(t *testing.T) {
	contract := validContract()

	_, err := contract.ResolveProfile("mini_vrooli_bundle", ResolveParams{})
	assertErrorKind(t, err, ErrInvalidInput)

	_, err = contract.ResolveProfile("missing", ResolveParams{})
	assertErrorKind(t, err, ErrNotFound)

	_, err = contract.ResolveProfile("mini_vrooli_bundle", ResolveParams{
		Values: map[string]string{"unexpected": "x"},
	})
	assertErrorKind(t, err, ErrInvalidInput)

	_, err = contract.ResolveProfile("mini_vrooli_bundle", ResolveParams{
		Lists: map[string][]string{"unexpected": {"x"}},
	})
	assertErrorKind(t, err, ErrInvalidInput)
}

func TestExpandProfileEntry(t *testing.T) {
	params := ResolveParams{
		Values: map[string]string{
			"scenario": "demo",
		},
		Lists: map[string][]string{
			"resources": {"postgres", "redis"},
		},
	}

	got, err := expandProfileEntry("scenarios/{scenario}", params)
	if err != nil || len(got) != 1 || got[0] != "scenarios/demo" {
		t.Fatalf("expandProfileEntry(scalar) = %v, %v", got, err)
	}

	got, err = expandProfileEntry("resources/{resources[*]}", params)
	if err != nil {
		t.Fatalf("expandProfileEntry(list) error = %v", err)
	}
	if len(got) != 2 || got[0] != "resources/postgres" || got[1] != "resources/redis" {
		t.Fatalf("expandProfileEntry(list) = %v", got)
	}

	got, err = expandProfileEntry("plain/path", params)
	if err != nil || len(got) != 1 || got[0] != "plain/path" {
		t.Fatalf("expandProfileEntry(plain) = %v, %v", got, err)
	}
}

func TestExpandProfileEntryErrorCases(t *testing.T) {
	params := ResolveParams{
		Values: map[string]string{
			"scenario": "",
		},
	}

	_, err := expandProfileEntry("scenarios/{scenario}", params)
	assertErrorKind(t, err, ErrInvalidInput)

	got, err := expandProfileEntry("resources/{resources[*]}", ResolveParams{})
	if err != nil {
		t.Fatalf("expandProfileEntry(empty list) error = %v", err)
	}
	if got != nil {
		t.Fatalf("expandProfileEntry(empty list) = %v, want nil", got)
	}
}

func TestExpandProfileEntriesDedupes(t *testing.T) {
	got, err := expandProfileEntries([]string{
		"scenarios/{scenario}",
		"scenarios/{scenario}",
	}, ResolveParams{
		Values: map[string]string{"scenario": "demo"},
	})
	if err != nil {
		t.Fatalf("expandProfileEntries() error = %v", err)
	}
	if len(got) != 1 || got[0] != "scenarios/demo" {
		t.Fatalf("expandProfileEntries() = %v", got)
	}
}

func TestDedupePreserveOrder(t *testing.T) {
	got := dedupePreserveOrder([]string{"a", "b", "a", "c", "b"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("dedupePreserveOrder() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dedupePreserveOrder()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
