package validation

// Severity classifies provider findings before the handler projects them onto
// the shared maturity contract.
type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warning"
	SeverityInfo  Severity = "info"
)

const (
	CodeTargetUnresolved      = "api_health.target_unresolved"
	CodeAPISurfaceAbsent      = "api_health.api_surface_absent"
	CodeServiceHealthMissing  = "api_health.service_health_missing"
	CodePreflightMissingLate  = "api_health.preflight_missing_or_late"
	CodeServerRunnerMissing   = "api_health.server_runner_missing"
	CodeHealthEndpointMissing = "api_health.health_endpoint_missing"
	CodeHealthProbeFailed     = "api_health.health_probe_failed"
	CodeHealthSchemaInvalid   = "api_health.health_schema_invalid"
	CodeRawStatusCode         = "api_health.raw_status_code"
	CodeImplicitErrorSuccess  = "api_health.error_response_implicit_success"
	CodeContentTypeMissing    = "api_health.content_type_missing"
	CodeUnversionedEndpoint   = "api_health.unversioned_feature_endpoint"
	CodeHTTPClientUnbounded   = "api_health.http_client_unbounded"
	CodeResponseBodyUnclosed  = "api_health.response_body_unclosed"
	CodeRequestContextDrop    = "api_health.request_context_dropped"
	CodeGoroutineUncancelled  = "api_health.goroutine_not_cancellable"
	CodeUnstructuredLogging   = "api_health.unstructured_api_logging"
)

// Finding is one API Health validation result.
type Finding struct {
	Severity    Severity
	Code        string
	Title       string
	Location    string
	Message     string
	Remediation string
}

// Summary aggregates finding counts by severity.
type Summary struct {
	Errors   int
	Warnings int
	Infos    int
}

// Report is the full provider-native result for one target.
type Report struct {
	Scenario string
	Target   Target
	Passed   bool
	Findings []Finding
	Summary  Summary
}

func finalize(scenario string, target Target, findings []Finding) Report {
	var summary Summary
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			summary.Errors++
		case SeverityWarn:
			summary.Warnings++
		case SeverityInfo:
			summary.Infos++
		}
	}
	return Report{
		Scenario: scenario,
		Target:   target,
		Passed:   summary.Errors == 0 && summary.Warnings == 0,
		Findings: findings,
		Summary:  summary,
	}
}
