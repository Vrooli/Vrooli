package phasepolicy

import "strings"

type (
	SelectionPolicy         string
	ProviderReadinessPolicy string
	ProviderLifecyclePolicy string
	FreshnessPolicy         string
	ResultGatingPolicy      string
	UnavailablePolicy       string
)

const (
	SelectionDefaultWhenApplicable       SelectionPolicy = "default_when_applicable"
	SelectionComprehensiveWhenApplicable SelectionPolicy = "comprehensive_when_applicable"
	SelectionExplicitOnly                SelectionPolicy = "explicit_only"
	SelectionNeverByDefault              SelectionPolicy = "never_by_default"

	ProviderReadinessNone                   ProviderReadinessPolicy = "none"
	ProviderReadinessRequiredWhenApplicable ProviderReadinessPolicy = "required_when_applicable"
	ProviderReadinessBestEffort             ProviderReadinessPolicy = "best_effort"

	ProviderLifecycleNone               ProviderLifecyclePolicy = "none"
	ProviderLifecycleCheckOnly          ProviderLifecyclePolicy = "check_only"
	ProviderLifecycleStartIfNeeded      ProviderLifecyclePolicy = "start_if_needed"
	ProviderLifecycleRestartBeforeProbe ProviderLifecyclePolicy = "restart_before_probe"

	FreshnessNone                FreshnessPolicy = "none"
	FreshnessRequireReachable    FreshnessPolicy = "require_reachable"
	FreshnessRequireLiveContract FreshnessPolicy = "require_live_contract"
	FreshnessRequireFreshBinary  FreshnessPolicy = "require_fresh_binary"

	ResultGatingGating             ResultGatingPolicy = "gating"
	ResultGatingAdvisory           ResultGatingPolicy = "advisory"
	ResultGatingHighConfidenceOnly ResultGatingPolicy = "high_confidence_gating"

	UnavailableFail               UnavailablePolicy = "fail"
	UnavailablePartial            UnavailablePolicy = "partial"
	UnavailableSkipWithoutFailing UnavailablePolicy = "skip_without_failing"
	UnavailableAdvisory           UnavailablePolicy = "advisory"
)

const (
	StatusPassed              = "passed"
	StatusFailed              = "failed"
	StatusSkipped             = "skipped"
	StatusMissing             = "missing"
	StatusNotExecutable       = "not_executable"
	StatusNotRun              = "not_run"
	StatusNotApplicable       = "not_applicable"
	StatusProviderUnavailable = "provider_unavailable"

	SuiteVerdictPass    = "pass"
	SuiteVerdictFail    = "fail"
	SuiteVerdictPartial = "partial"
)

type Policy struct {
	Selection         SelectionPolicy         `json:"selection"`
	ProviderReadiness ProviderReadinessPolicy `json:"providerReadiness"`
	ProviderLifecycle ProviderLifecyclePolicy `json:"providerLifecycle"`
	Freshness         FreshnessPolicy         `json:"freshness"`
	ResultGating      ResultGatingPolicy      `json:"resultGating"`
	Unavailable       UnavailablePolicy       `json:"unavailable"`
}

type ValidationError struct {
	Code    string
	Field   string
	Message string
}

type ExecutionInput struct {
	Phase                 string
	Status                string
	FailureClassification string
	Policy                Policy
}

func RequiredProviderPolicy() Policy {
	return Policy{
		Selection:         SelectionDefaultWhenApplicable,
		ProviderReadiness: ProviderReadinessRequiredWhenApplicable,
		ProviderLifecycle: ProviderLifecycleStartIfNeeded,
		Freshness:         FreshnessRequireLiveContract,
		ResultGating:      ResultGatingGating,
		Unavailable:       UnavailableFail,
	}
}

func BestEffortProviderPolicy() Policy {
	p := RequiredProviderPolicy()
	p.ProviderReadiness = ProviderReadinessBestEffort
	p.Unavailable = UnavailableSkipWithoutFailing
	return p
}

func AdvisoryProviderPolicy() Policy {
	p := BestEffortProviderPolicy()
	p.ResultGating = ResultGatingAdvisory
	p.Unavailable = UnavailableAdvisory
	return p
}

func HighConfidenceProviderPolicy() Policy {
	p := BestEffortProviderPolicy()
	p.ResultGating = ResultGatingHighConfidenceOnly
	return p
}

func FromLegacyCatalog(optional, advisory bool) Policy {
	if advisory {
		return AdvisoryProviderPolicy()
	}
	if optional {
		return BestEffortProviderPolicy()
	}
	return RequiredProviderPolicy()
}

func (p Policy) IsZero() bool {
	return p.Selection == "" &&
		p.ProviderReadiness == "" &&
		p.ProviderLifecycle == "" &&
		p.Freshness == "" &&
		p.ResultGating == "" &&
		p.Unavailable == ""
}

func (p Policy) Validate() []ValidationError {
	var out []ValidationError
	add := func(code, field, message string) {
		out = append(out, ValidationError{Code: code, Field: field, Message: message})
	}
	if !oneOf(string(p.Selection), SelectionDefaultWhenApplicable, SelectionComprehensiveWhenApplicable, SelectionExplicitOnly, SelectionNeverByDefault) {
		add("invalid_selection_policy", "selection", "unknown selection policy")
	}
	if !oneOf(string(p.ProviderReadiness), ProviderReadinessNone, ProviderReadinessRequiredWhenApplicable, ProviderReadinessBestEffort) {
		add("invalid_provider_readiness_policy", "providerReadiness", "unknown provider readiness policy")
	}
	if !oneOf(string(p.ProviderLifecycle), ProviderLifecycleNone, ProviderLifecycleCheckOnly, ProviderLifecycleStartIfNeeded, ProviderLifecycleRestartBeforeProbe) {
		add("invalid_provider_lifecycle_policy", "providerLifecycle", "unknown provider lifecycle policy")
	}
	if !oneOf(string(p.Freshness), FreshnessNone, FreshnessRequireReachable, FreshnessRequireLiveContract, FreshnessRequireFreshBinary) {
		add("invalid_freshness_policy", "freshness", "unknown freshness policy")
	}
	if !oneOf(string(p.ResultGating), ResultGatingGating, ResultGatingAdvisory, ResultGatingHighConfidenceOnly) {
		add("invalid_result_gating_policy", "resultGating", "unknown result gating policy")
	}
	if !oneOf(string(p.Unavailable), UnavailableFail, UnavailablePartial, UnavailableSkipWithoutFailing, UnavailableAdvisory) {
		add("invalid_unavailable_policy", "unavailable", "unknown unavailable policy")
	}
	if p.ProviderReadiness == ProviderReadinessNone && p.ProviderLifecycle != ProviderLifecycleNone {
		add("unsafe_policy", "providerLifecycle", "providerLifecycle must be none when providerReadiness is none")
	}
	return out
}

func SuiteVerdictForExecution(results []ExecutionInput) string {
	partial := false
	for _, result := range results {
		policy := result.Policy
		if policy.IsZero() {
			policy = RequiredProviderPolicy()
		}
		switch normalize(result.Status) {
		case StatusFailed:
			if policy.ResultGating != ResultGatingAdvisory {
				return SuiteVerdictFail
			}
		case StatusProviderUnavailable:
			switch policy.Unavailable {
			case UnavailableFail:
				return SuiteVerdictFail
			case UnavailablePartial:
				partial = true
			}
		case StatusSkipped, StatusMissing, StatusNotExecutable, StatusNotRun:
			switch policy.Unavailable {
			case UnavailableFail, UnavailablePartial:
				partial = true
			}
		}
	}
	if partial {
		return SuiteVerdictPartial
	}
	return SuiteVerdictPass
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func oneOf[T ~string](value string, allowed ...T) bool {
	for _, item := range allowed {
		if value == string(item) {
			return true
		}
	}
	return false
}
