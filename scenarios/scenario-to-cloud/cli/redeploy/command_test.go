package redeploy

import (
	"strings"
	"testing"
)

func TestToSelectorValidationAndTrim(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		scenarioID string
		domain     string
		target     string
		wantErr    string
	}{
		{
			name:    "reject target with host",
			host:    "203.0.113.10",
			target:  "vrooli.com",
			wantErr: "--target cannot be combined with --host or --domain",
		},
		{
			name:    "reject target with domain",
			domain:  "vrooli.com",
			target:  "target.example.com",
			wantErr: "--target cannot be combined with --host or --domain",
		},
		{
			name:    "require at least one selector",
			wantErr: "at least one selector is required",
		},
		{
			name:       "accept and trim valid selector",
			host:       " 203.0.113.10 ",
			scenarioID: " landing-page-business-suite ",
			domain:     " vrooli.com ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := toSelector(tc.host, tc.scenarioID, tc.domain, tc.target)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("toSelector returned error: %v", err)
			}
			if got.Host != "203.0.113.10" || got.ScenarioID != "landing-page-business-suite" || got.Domain != "vrooli.com" {
				t.Fatalf("unexpected selector: %+v", got)
			}
		})
	}
}

func TestRunSelectorModeRequiresIfNeeded(t *testing.T) {
	err := Run(nil, []string{"--domain", "vrooli.com", "--scenario", "landing-page-business-suite"})
	if err == nil {
		t.Fatal("expected selector mode to require --if-needed")
	}
	if !strings.Contains(err.Error(), "selector mode requires --if-needed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAllowsJSONAndWaitTogether(t *testing.T) {
	err := Run(nil, []string{
		"--domain", "vrooli.com",
		"--scenario", "landing-page-business-suite",
		"--wait",
		"--json",
	})
	if err == nil {
		t.Fatal("expected selector mode to require --if-needed")
	}
	if strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("unexpected incompatible-flags error: %v", err)
	}
	if !strings.Contains(err.Error(), "selector mode requires --if-needed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
