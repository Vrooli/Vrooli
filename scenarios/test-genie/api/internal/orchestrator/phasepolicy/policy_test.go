package phasepolicy

import "testing"

func TestPolicyValidation(t *testing.T) {
	tests := []struct {
		name string
		in   Policy
		want string
	}{
		{
			name: "valid required provider policy",
			in:   RequiredProviderPolicy(),
		},
		{
			name: "invalid selection",
			in: Policy{
				Selection:         "sometimes",
				ProviderReadiness: ProviderReadinessRequiredWhenApplicable,
				ProviderLifecycle: ProviderLifecycleStartIfNeeded,
				Freshness:         FreshnessRequireLiveContract,
				ResultGating:      ResultGatingGating,
				Unavailable:       UnavailableFail,
			},
			want: "invalid_selection_policy",
		},
		{
			name: "provider lifecycle without provider readiness",
			in: Policy{
				Selection:         SelectionDefaultWhenApplicable,
				ProviderReadiness: ProviderReadinessNone,
				ProviderLifecycle: ProviderLifecycleStartIfNeeded,
				Freshness:         FreshnessNone,
				ResultGating:      ResultGatingGating,
				Unavailable:       UnavailableFail,
			},
			want: "unsafe_policy",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Validate()
			if tc.want == "" && len(got) != 0 {
				t.Fatalf("Validate returned errors: %#v", got)
			}
			if tc.want != "" && !hasCode(got, tc.want) {
				t.Fatalf("Validate errors = %#v, want %s", got, tc.want)
			}
		})
	}
}

func TestSuiteVerdictForExecution(t *testing.T) {
	tests := []struct {
		name    string
		results []ExecutionInput
		want    string
	}{
		{
			name: "required unavailable provider fails",
			results: []ExecutionInput{{
				Phase:  "structure",
				Status: StatusProviderUnavailable,
				Policy: RequiredProviderPolicy(),
			}},
			want: SuiteVerdictFail,
		},
		{
			name: "best effort unavailable provider passes",
			results: []ExecutionInput{{
				Phase:  "performance",
				Status: StatusProviderUnavailable,
				Policy: BestEffortProviderPolicy(),
			}},
			want: SuiteVerdictPass,
		},
		{
			name: "partial unavailable provider produces partial suite",
			results: []ExecutionInput{{
				Phase:  "docs",
				Status: StatusProviderUnavailable,
				Policy: withUnavailable(RequiredProviderPolicy(), UnavailablePartial),
			}},
			want: SuiteVerdictPartial,
		},
		{
			name: "non-applicable phase does not affect verdict",
			results: []ExecutionInput{{
				Phase:  "search",
				Status: StatusNotApplicable,
				Policy: RequiredProviderPolicy(),
			}},
			want: SuiteVerdictPass,
		},
		{
			name: "advisory failed result does not fail suite",
			results: []ExecutionInput{{
				Phase:  "measures",
				Status: StatusFailed,
				Policy: AdvisoryProviderPolicy(),
			}},
			want: SuiteVerdictPass,
		},
		{
			name: "high confidence gating result fails once provider marks failed",
			results: []ExecutionInput{{
				Phase:  "architecture",
				Status: StatusFailed,
				Policy: HighConfidenceProviderPolicy(),
			}},
			want: SuiteVerdictFail,
		},
		{
			name: "legacy required skip remains partial",
			results: []ExecutionInput{{
				Phase:  "unit",
				Status: StatusSkipped,
				Policy: FromLegacyCatalog(false, false),
			}},
			want: SuiteVerdictPartial,
		},
		{
			name: "legacy optional skip remains pass",
			results: []ExecutionInput{{
				Phase:  "performance",
				Status: StatusSkipped,
				Policy: FromLegacyCatalog(true, false),
			}},
			want: SuiteVerdictPass,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SuiteVerdictForExecution(tc.results); got != tc.want {
				t.Fatalf("SuiteVerdictForExecution = %q, want %q", got, tc.want)
			}
		})
	}
}

func withUnavailable(policy Policy, unavailable UnavailablePolicy) Policy {
	policy.Unavailable = unavailable
	return policy
}

func hasCode(errors []ValidationError, code string) bool {
	for _, err := range errors {
		if err.Code == code {
			return true
		}
	}
	return false
}
