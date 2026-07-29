package experimentation

import (
	"strings"
	"testing"
)

func TestValidateSelectionEnforcesExperimentAssignmentPolicy(t *testing.T) {
	space := Space{
		Axes: map[string]Axis{
			"persona": {Variants: []string{"founder", "operator"}},
			"cta":     {Variants: []string{"demo", "trial"}},
		},
		DisallowedCombinations: []map[string]string{{"persona": "founder", "cta": "trial"}},
	}

	for name, tc := range map[string]struct {
		selection map[string]string
		wantError string
	}{
		"accepts complete allowed assignment": {selection: map[string]string{"persona": "operator", "cta": "trial"}},
		"rejects unknown axis":                {selection: map[string]string{"persona": "operator", "cta": "trial", "region": "us"}, wantError: "unknown axis region"},
		"rejects missing axis":                {selection: map[string]string{"persona": "operator"}, wantError: "axis cta is required"},
		"rejects invalid value":               {selection: map[string]string{"persona": "operator", "cta": "purchase"}, wantError: "invalid value 'purchase' for axis cta"},
		"rejects disallowed combination":      {selection: map[string]string{"persona": "founder", "cta": "trial"}, wantError: "is disallowed"},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateSelection(space, tc.selection)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("ValidateSelection() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ValidateSelection() error = %v, want %q", err, tc.wantError)
			}
		})
	}
}
