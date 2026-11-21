package protolabel

import (
	"testing"

	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
)

func TestArchetype(t *testing.T) {
	tests := []struct {
		name  string
		value *domainsv1.DomainArchetype
		want  string
	}{
		{"reporting", &domainsv1.DomainArchetype{Archetype: domainsv1.Archetype_ARCHETYPE_REPORTING}, "reporting"},
		{"service", &domainsv1.DomainArchetype{Archetype: domainsv1.Archetype_ARCHETYPE_SERVICE}, "service"},
		{"mutation", &domainsv1.DomainArchetype{Archetype: domainsv1.Archetype_ARCHETYPE_MUTATION}, "mutation"},
		{"classification", &domainsv1.DomainArchetype{Archetype: domainsv1.Archetype_ARCHETYPE_CLASSIFICATION}, "classification"},
		{"orchestration", &domainsv1.DomainArchetype{Archetype: domainsv1.Archetype_ARCHETYPE_ORCHESTRATION}, "orchestration"},
		{"scoring", &domainsv1.DomainArchetype{Archetype: domainsv1.Archetype_ARCHETYPE_SCORING}, "scoring"},
		{"query", &domainsv1.DomainArchetype{Archetype: domainsv1.Archetype_ARCHETYPE_QUERY}, "query"},
		{"declared fallback", &domainsv1.DomainArchetype{Archetype: domainsv1.Archetype_ARCHETYPE_UNSPECIFIED, DeclaredLabel: "custom"}, "custom"},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Archetype(tt.value); got != tt.want {
				t.Fatalf("Archetype(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestFindingClass(t *testing.T) {
	for _, tt := range []struct {
		value sharedv1.FindingClass
		want  string
	}{
		{sharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC, "deterministic"},
		{sharedv1.FindingClass_FINDING_CLASS_HEURISTIC, "heuristic"},
		{sharedv1.FindingClass_FINDING_CLASS_UNSPECIFIED, "unspecified"},
		{sharedv1.FindingClass(99), "unspecified"},
	} {
		if got := FindingClass(tt.value); got != tt.want {
			t.Errorf("FindingClass(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
}
