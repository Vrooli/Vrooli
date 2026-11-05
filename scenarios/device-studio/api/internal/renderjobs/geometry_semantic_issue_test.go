package renderjobs

import (
	"math"
	"testing"

	"device-studio/internal/geometry"
)

func TestValidGeometrySemanticIssueRequiresExactBevelBounds(t *testing.T) {
	issue := geometry.SemanticIssue{
		Code:                "bevel_exceeds_part_dimensions",
		PartID:              "lower_bezel",
		BevelMM:             1,
		MinimumDimensionMM:  2,
		MaxBevelExclusiveMM: 1,
		Message:             "bevel must be less than half the smallest part dimension",
	}
	if !validGeometrySemanticIssue(&issue) {
		t.Fatal("a finite typed bevel issue with its exact exclusive limit should be safely correctable")
	}
	for name, mutate := range map[string]func(*geometry.SemanticIssue){
		"altered bound":        func(i *geometry.SemanticIssue) { i.MaxBevelExclusiveMM = 1.01 },
		"valid bevel":          func(i *geometry.SemanticIssue) { i.BevelMM = .9 },
		"non-finite dimension": func(i *geometry.SemanticIssue) { i.MinimumDimensionMM = math.NaN() },
		"arbitrary message":    func(i *geometry.SemanticIssue) { i.Message = "provider supplied text" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := issue
			mutate(&candidate)
			if validGeometrySemanticIssue(&candidate) {
				t.Fatal("malformed issue must not be exposed through correction")
			}
		})
	}
}
