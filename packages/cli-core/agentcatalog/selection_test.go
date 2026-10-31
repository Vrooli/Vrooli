package agentcatalog

import (
	"errors"
	"reflect"
	"testing"
)

func selectionCatalog() CodingRoleCatalog {
	effort := "high"
	return CodingRoleCatalog{
		Runner: "opencode",
		ModelAliases: map[string]ModelAlias{
			"m1": {CanonicalModel: "vendor/m1", Provider: "provider-a"},
			"m2": {CanonicalModel: "vendor/m2", Provider: "provider-b"},
		},
		ExcludedModels:   []string{"blocked"},
		RestrictedModels: map[string][]string{"restricted": {"judgment.supervision"}},
		Roles: map[string]CodingRole{
			"code.default": {
				Model: "m1", CanonicalModel: "vendor/m1",
				Models: []RoleModel{
					{Model: "m1", Source: "provider-a", Effort: &effort},
					{Model: "m1", Source: "provider-b"},
					{Model: "m2", Source: "provider-b"},
					{Model: "blocked", Source: "provider-a"},
					{Model: "restricted", Source: "provider-a"},
				},
			},
			"judgment.supervision": {Model: "restricted"},
		},
	}
}

func TestRoleModelChoicesPreservesDeclaredSourceAndEffortAndAppliesRestrictions(t *testing.T) {
	choices, err := RoleModelChoices(selectionCatalog(), "code.default")
	if err != nil {
		t.Fatalf("RoleModelChoices: %v", err)
	}
	got := make([]string, 0, len(choices))
	for _, choice := range choices {
		got = append(got, choice.Model+"/"+choice.Source)
	}
	if want := []string{"m1/provider-a", "m1/provider-b", "m2/provider-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("choices = %#v, want %#v", got, want)
	}
	if choices[0].CanonicalModel != "vendor/m1" || choices[0].Provider != "provider-a" || choices[0].Effort == nil || *choices[0].Effort != "high" {
		t.Fatalf("choice evidence = %#v", choices[0])
	}
}

func TestResolveRoleSelectionKeepsDefaultModelButReportsAmbiguousSource(t *testing.T) {
	resolved, err := ResolveRoleSelection(selectionCatalog(), RoleSelectionIntent{Role: "code.default"})
	if err != nil {
		t.Fatalf("ResolveRoleSelection: %v", err)
	}
	if resolved.Model != "m1" || resolved.Source != "" || resolved.Provider != "" || resolved.SourceStatus != "ambiguous" || !resolved.IsDefault {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveRoleSelectionAcceptsExactModelAndSourceOverrides(t *testing.T) {
	resolved, err := ResolveRoleSelection(selectionCatalog(), RoleSelectionIntent{Role: "code.default", Model: "m1", Source: "provider-b"})
	if err != nil {
		t.Fatalf("ResolveRoleSelection: %v", err)
	}
	if resolved.Model != "m1" || resolved.Source != "provider-b" || resolved.Provider != "provider-a" || resolved.SourceStatus != "declared" || resolved.IsDefault {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveRoleSelectionRequiresModelWhenSourceHasMultipleModels(t *testing.T) {
	_, err := ResolveRoleSelection(selectionCatalog(), RoleSelectionIntent{Role: "code.default", Source: "provider-b"})
	if !errors.Is(err, ErrModelSelectionNeeded) {
		t.Fatalf("source-only error = %v, want model selection required", err)
	}

	resolved, err := ResolveRoleSelection(selectionCatalog(), RoleSelectionIntent{Role: "code.default", Source: "provider-a"})
	if err != nil {
		t.Fatalf("unique source choice: %v", err)
	}
	if resolved.Model != "m1" || resolved.Source != "provider-a" || resolved.IsDefault {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveRoleSelectionRejectsAmbiguousAndUnknownExplicitChoices(t *testing.T) {
	cases := []struct {
		name   string
		intent RoleSelectionIntent
		want   error
	}{
		{"missing source", RoleSelectionIntent{Role: "code.default", Model: "m1"}, ErrSourceSelectionNeeded},
		{"unknown model", RoleSelectionIntent{Role: "code.default", Model: "unlisted"}, ErrUnsupportedModel},
		{"unknown source", RoleSelectionIntent{Role: "code.default", Model: "m1", Source: "unknown"}, ErrUnsupportedSource},
		{"unknown role", RoleSelectionIntent{Role: "code.unknown"}, ErrUnknownCodingRole},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveRoleSelection(selectionCatalog(), tc.intent)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestResolveRoleSelectionReportsUnknownSourceForLegacyRole(t *testing.T) {
	catalog := selectionCatalog()
	catalog.Roles["code.default"] = CodingRole{Model: "m1", CanonicalModel: "vendor/m1"}
	resolved, err := ResolveRoleSelection(catalog, RoleSelectionIntent{Role: "code.default"})
	if err != nil {
		t.Fatalf("ResolveRoleSelection: %v", err)
	}
	if resolved.Model != "m1" || resolved.Source != "" || resolved.SourceStatus != "unknown" {
		t.Fatalf("resolved = %#v", resolved)
	}
	if _, err := ResolveRoleSelection(catalog, RoleSelectionIntent{Role: "code.default", Source: "provider-a"}); !errors.Is(err, ErrUnsupportedSource) {
		t.Fatalf("source override error = %v, want unsupported", err)
	}
}

func TestRoleModelChoicesRejectMalformedPolicyEvidence(t *testing.T) {
	catalog := selectionCatalog()
	role := catalog.Roles["code.default"]
	role.Models[0].Source = " provider-a"
	catalog.Roles["code.default"] = role
	if _, err := RoleModelChoices(catalog, "code.default"); !errors.Is(err, ErrInvalidRoleModel) {
		t.Fatalf("error = %v, want invalid model declaration", err)
	}
}
