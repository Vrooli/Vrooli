package migration

type Decision string

const (
	DecisionRedesigned Decision = "redesigned"
	DecisionKept       Decision = "kept"
	DecisionDelegated  Decision = "delegated"
	DecisionDeferred   Decision = "deferred"
	DecisionRejected   Decision = "rejected"
)

type Rule struct {
	File             string
	RuleID           string
	Decision         Decision
	Owner            string
	APIHealthMapping string
	Rationale        string
}

func Ledger() []Rule {
	return []Rule{
		{
			File:             "application_logging.go",
			RuleID:           "application_logging",
			Decision:         DecisionRedesigned,
			Owner:            "api-health runtime hygiene",
			APIHealthMapping: "api.unstructured_logging",
			Rationale:        "API Health preserves operational logging intent but scopes it to production API runtime paths and avoids generic format-string policing.",
		},
		{
			File:             "content_type_binary.go",
			RuleID:           "content_type_binary",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.content_type_missing",
			Rationale:        "Binary responses are handled through explicit writer evidence and route classification instead of independent regex rules.",
		},
		{
			File:             "content_type_csv.go",
			RuleID:           "content_type_csv",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.content_type_missing",
			Rationale:        "CSV response intent is kept only when a production API route clearly writes CSV/text content directly.",
		},
		{
			File:             "content_type_headers.go",
			RuleID:           "content_type_headers",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.content_type_missing",
			Rationale:        "The broad header rule becomes low-ambiguity content-type evidence for production REST handlers, excluding tests, Connect procedures, and framework-owned responses.",
		},
		{
			File:             "content_type_html.go",
			RuleID:           "content_type_html",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.content_type_missing",
			Rationale:        "HTML response checks are folded into the route-aware content-type analyzer so static/template paths are not treated as feature API defects.",
		},
		{
			File:             "content_type_pdf.go",
			RuleID:           "content_type_pdf",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.content_type_missing",
			Rationale:        "PDF/download checks are retained only for obvious direct writes where API Health can cite file and handler evidence.",
		},
		{
			File:             "content_type_streaming.go",
			RuleID:           "content_type_streaming",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.content_type_missing",
			Rationale:        "Streaming/SSE intent is represented by route classification and explicit content-type evidence rather than broad source scanning.",
		},
		{
			File:             "content_type_text.go",
			RuleID:           "content_type_text",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.content_type_missing",
			Rationale:        "Plain-text response checks are merged into the shared content-type analyzer with production API path filtering.",
		},
		{
			File:             "content_type_xml.go",
			RuleID:           "content_type_xml",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.content_type_missing",
			Rationale:        "XML response checks are retained only where direct writer evidence avoids helper and test false positives.",
		},
		{
			File:             "file_close.go",
			RuleID:           "file_close",
			Decision:         DecisionDelegated,
			Owner:            "generic resource hygiene provider",
			APIHealthMapping: "n/a",
			Rationale:        "Disk file-handle hygiene is not API readiness. API Health only owns HTTP response body closure because that is API runtime behavior.",
		},
		{
			File:             "goroutine_context.go",
			RuleID:           "goroutine_context",
			Decision:         DecisionRedesigned,
			Owner:            "api-health runtime hygiene",
			APIHealthMapping: "api.goroutine_uncancellable",
			Rationale:        "The legacy concern is kept for API startup and handler paths, using AST evidence for long-lived loops without cancellation.",
		},
		{
			File:             "health_check.go",
			RuleID:           "health_check",
			Decision:         DecisionRedesigned,
			Owner:            "api-health lifecycle and live health probe",
			APIHealthMapping: "api.health_metadata_missing, api.health_probe_*",
			Rationale:        "Health readiness now combines service metadata validation, endpoint descriptors, and optional bounded live /health probing.",
		},
		{
			File:             "http_client_timeout.go",
			RuleID:           "http_client_timeout",
			Decision:         DecisionRedesigned,
			Owner:            "api-health runtime hygiene",
			APIHealthMapping: "api.http_client_unbounded",
			Rationale:        "Outbound HTTP timeout intent is preserved for production API runtime paths and paired with request-context evidence.",
		},
		{
			File:             "http_response_close.go",
			RuleID:           "http_response_close",
			Decision:         DecisionRedesigned,
			Owner:            "api-health runtime hygiene",
			APIHealthMapping: "api.response_body_unclosed",
			Rationale:        "API Health keeps response-body close checks because they directly affect API runtime resource usage.",
		},
		{
			File:             "http_status_codes.go",
			RuleID:           "http_status_codes",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.raw_status_literal",
			Rationale:        "Raw numeric status checks are retained, but only for production API files with precise AST location evidence.",
		},
		{
			File:             "preflight.go",
			RuleID:           "preflight",
			Decision:         DecisionRedesigned,
			Owner:            "api-health lifecycle",
			APIHealthMapping: "api.preflight_missing, api.preflight_late, api.preflight_wrong_name",
			Rationale:        "Preflight validation now checks api/main.go structure and scenario name with Go AST instead of line-window heuristics.",
		},
		{
			File:             "security_headers.go",
			RuleID:           "security_headers",
			Decision:         DecisionDelegated,
			Owner:            "security-health",
			APIHealthMapping: "n/a",
			Rationale:        "CORS, HSTS, clickjacking, and browser security headers are security posture, not API readiness.",
		},
		{
			File:             "server_run.go",
			RuleID:           "server_run",
			Decision:         DecisionRedesigned,
			Owner:            "api-health lifecycle",
			APIHealthMapping: "api.direct_listen_and_serve, api.server_runner_missing",
			Rationale:        "Server lifecycle checks now assert api-core server runner adoption and reject direct ListenAndServe calls in production API entrypoints.",
		},
		{
			File:             "versioned_endpoints.go",
			RuleID:           "versioned_endpoints",
			Decision:         DecisionRedesigned,
			Owner:            "api-health HTTP semantics",
			APIHealthMapping: "api.rest_endpoint_unversioned",
			Rationale:        "REST versioning is preserved for feature endpoints while exempting /health, Connect procedures, static assets, and declared rest_exception routes.",
		},
	}
}
