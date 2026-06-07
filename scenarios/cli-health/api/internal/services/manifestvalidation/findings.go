// Package manifestvalidation runs the cli-health validators against a
// scenario's cli/manifest.json. It loads the manifest, schema-validates it,
// loads the scenario's proto descriptors via buf, and reports a Finding list
// describing schema errors, unresolved bindings, orphan methods, and stale
// omissions. The handler layer maps these onto the proto Finding/Severity
// types; this package stays transport-agnostic.
package manifestvalidation

// Severity classifies a Finding. The Connect handler maps these onto the
// proto Severity enum 1:1.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Finding codes — stable, machine-readable strings. Tooling filters by code.
const (
	CodeManifestMissing      = "manifest.missing"
	CodeManifestParseError   = "manifest.parse_error"
	CodeManifestSchemaError  = "manifest.schema_error"
	CodeProtoBuildFailed     = "proto.build_failed"
	CodeBindingUnknownSvc    = "binding.unknown_service"
	CodeBindingUnknownMethod = "binding.unknown_method"
	CodeBindingDuplicate     = "binding.duplicate"
	CodeProtoOrphanMethod    = "proto.orphan_method"
	CodeOmissionOrphan       = "omission.orphan"

	// Measure-block codes (Phase 2 of the measures plan). Static well-formedness
	// only — coverage/expected/waived domain grading is measures-health's job.
	CodeMeasureInvalid      = "measure.invalid"            // assembly/Validate failure (drift, bad result/effect)
	CodeMeasureUnknownType  = "measure.unknown_param_type" // manifest param `type` annotation is not a known canonical convention
	CodeMeasureSchemaUnread = "measure.schema_unread"      // proto param schema could not be resolved (descriptor unavailable)
	CodeMeasureTier         = "measure.tier"               // info: the graded adoption tier for a well-formed measure
)

// Finding is a single validation result.
type Finding struct {
	Severity   Severity
	Code       string
	Location   string
	Message    string
	Suggestion string
}

// Summary aggregates Finding counts by severity.
type Summary struct {
	Errors   int
	Warnings int
	Infos    int
}

// Report is the full result for one scenario.
type Report struct {
	Scenario string
	Passed   bool
	Findings []Finding
	Summary  Summary
}

func summarize(findings []Finding) Summary {
	var s Summary
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			s.Errors++
		case SeverityWarning:
			s.Warnings++
		case SeverityInfo:
			s.Infos++
		}
	}
	return s
}
