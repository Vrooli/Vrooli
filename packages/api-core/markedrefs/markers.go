package markedrefs

// MarkerSpec describes a supported marked-reference marker.
type MarkerSpec struct {
	Name        string
	Description string
}

// QualifierSpec describes a supported marked-reference qualifier.
type QualifierSpec struct {
	Name        string
	Description string
}

const (
	MarkerPath     = "path"
	MarkerDoc      = "doc"
	MarkerURL      = "url"
	MarkerTopic    = "topic"
	MarkerSkill    = "skill"
	MarkerAgent    = "agent"
	MarkerTeam     = "team"
	MarkerScenario = "scenario"
	MarkerResource = "resource"
	MarkerAction   = "action"
	MarkerDecision = "decision"
	MarkerCLI      = "cli"
	MarkerEnv      = "env"
	MarkerPlatform = "platform"
	MarkerMIME     = "mime"
	MarkerRoute    = "route"
	MarkerPackage  = "package"
	MarkerLiteral  = "literal"
	MarkerNum      = "num"
)

const (
	QualifierExample  = "example"
	QualifierOld      = "old"
	QualifierFuture   = "future"
	QualifierOptional = "optional"
	QualifierExternal = "external"
	QualifierLiteral  = "literal"
)

var markerSpecs = []MarkerSpec{
	{MarkerPath, "Repo-relative file or directory path."},
	{MarkerDoc, "Documentation page or path intended for docs navigation."},
	{MarkerURL, "External URL."},
	{MarkerTopic, "Prompt-manager knowledge topic prefix or entry topic."},
	{MarkerSkill, "Prompt-manager skill id."},
	{MarkerAgent, "Prompt-manager agent id."},
	{MarkerTeam, "Prompt-manager team id."},
	{MarkerScenario, "Scenario id."},
	{MarkerResource, "Resource id."},
	{MarkerAction, "Prompt-manager Action id."},
	{MarkerDecision, "Decision context or decision id."},
	{MarkerCLI, "CLI command or subcommand."},
	{MarkerEnv, "Environment variable."},
	{MarkerPlatform, "OS / architecture / runtime target."},
	{MarkerMIME, "MIME or media type."},
	{MarkerRoute, "HTTP, API, or UI route path."},
	{MarkerPackage, "Package, module, or import path."},
	{MarkerLiteral, "Intentionally literal value that should not be semantically validated."},
	{MarkerNum, "Intentional, owner-backed number in prose (target/threshold/price/version/decision/sot) that should not be flagged as a drift-prone count."},
}

var qualifierSpecs = []QualifierSpec{
	{QualifierExample, "Illustrative only; existence is not required."},
	{QualifierOld, "Historical or deprecated; current existence is not required."},
	{QualifierFuture, "Planned; absence is allowed."},
	{QualifierOptional, "May depend on installation, config, or platform."},
	{QualifierExternal, "Outside this repo or local Vrooli installation."},
	{QualifierLiteral, "Intentionally not semantic despite using a marker-shaped form."},
}

var (
	knownMarkers    = buildMarkerSet(markerSpecs)
	knownQualifiers = buildQualifierSet(qualifierSpecs)
)

// KnownMarkers returns the supported marker registry in display order.
func KnownMarkers() []MarkerSpec {
	out := make([]MarkerSpec, len(markerSpecs))
	copy(out, markerSpecs)
	return out
}

// KnownQualifiers returns the supported qualifier registry in display order.
func KnownQualifiers() []QualifierSpec {
	out := make([]QualifierSpec, len(qualifierSpecs))
	copy(out, qualifierSpecs)
	return out
}

// IsKnownMarker reports whether marker is in the project-level registry.
func IsKnownMarker(marker string) bool {
	return knownMarkers[marker]
}

// IsKnownQualifier reports whether qualifier is in the project-level registry.
func IsKnownQualifier(qualifier string) bool {
	return knownQualifiers[qualifier]
}

func buildMarkerSet(specs []MarkerSpec) map[string]bool {
	out := make(map[string]bool, len(specs))
	for _, spec := range specs {
		out[spec.Name] = true
	}
	return out
}

func buildQualifierSet(specs []QualifierSpec) map[string]bool {
	out := make(map[string]bool, len(specs))
	for _, spec := range specs {
		out[spec.Name] = true
	}
	return out
}
