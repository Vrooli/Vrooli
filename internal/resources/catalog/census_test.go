package catalog

import "testing"

func TestCensusReason(t *testing.T) {
	tests := []struct {
		name                             string
		inContract, enabled, declaresCLI bool
		installed                        bool
		want                             string
	}{
		{"absent", false, false, true, true, "absent_from_contract"},
		{"disabled", true, false, true, true, "resource_disabled"},
		{"not declared", true, true, false, false, "cli_not_declared"},
		{"declared missing", true, true, true, false, "declared_not_installed"},
		{"installed", true, true, true, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := censusReason(tt.inContract, tt.enabled, tt.declaresCLI, tt.installed); got != tt.want {
				t.Fatalf("censusReason() = %q, want %q", got, tt.want)
			}
		})
	}
}
