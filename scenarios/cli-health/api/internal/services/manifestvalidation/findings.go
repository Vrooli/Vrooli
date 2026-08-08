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
	CodeManifestRequired     = "manifest.required"
	CodeManifestParseError   = "manifest.parse_error"
	CodeManifestSchemaError  = "manifest.schema_error"
	CodeProtoBuildFailed     = "proto.build_failed"
	CodeBindingUnknownSvc    = "binding.unknown_service"
	CodeBindingUnknownMethod = "binding.unknown_method"
	CodeBindingDuplicate     = "binding.duplicate"
	CodeBindingArgUnmapped   = "binding.arg_unmapped"
	CodeBindingAmbiguousSvc  = "binding.ambiguous_service"
	CodeBindingFieldCollision = "binding.field_collision"
	CodeBindingControlFlagBound = "binding.control_flag_bound"
	CodeBindingRequiredFieldUnpopulated = "binding.required_field_unpopulated"
	CodeBindingBindWhereRenameSuffices = "binding.bind_where_rename_suffices"
	// CodeBindingScalarBoundToMessage fires when a single CLI value targets a
	// structured proto field with no structured decoder. The argument resolves,
	// so callability checks pass, but protojson cannot build the message from a
	// bare scalar and the call fails or silently drops the value at runtime.
	CodeBindingScalarBoundToMessage = "binding.scalar_bound_to_message"
	CodeProtoOrphanMethod    = "proto.orphan_method"
	CodeOmissionOrphan       = "omission.orphan"

	// Measure-block codes (Phase 2 of the measures plan). Static well-formedness
	// only — coverage/expected/waived domain grading is measures-health's job.
	CodeMeasureInvalid      = "measure.invalid"            // assembly/Validate failure (drift, bad result/effect)
	CodeMeasureUnknownType  = "measure.unknown_param_type" // manifest param `type` annotation is not a known canonical convention
	CodeMeasureSchemaUnread = "measure.schema_unread"      // proto param schema could not be resolved (descriptor unavailable)
	CodeMeasureTier         = "measure.tier"               // info: the graded adoption tier for a well-formed measure

	// Runtime CLI-probe codes. Emitted only when the caller requests execution
	// (include_execution) and the scenario declares a CLI surface; the probe
	// resolves and execs the scenario's binary rather than reading the manifest
	// statically. Degrades to a warning when the binary is simply absent in the
	// run context (see the runtimeprobe seam) — never hard-fails a scenario whose
	// CLI is not installed here.
	CodeCLIBinaryUnrunnable        = "cli.binary_unrunnable"            // declared CLI surface but binary cannot be resolved in this run context
	CodeCLIHelpFailed              = "cli.help_failed"                  // binary resolves but `--help` errors / produces nothing
	CodeCLICommandUndeclared       = "cli.command_undeclared"           // runtime command surface diverges from the manifest
	CodeCLIDiscoveryCoverage       = "cli.discovery_coverage_low"       // manifest does not cover the observed runtime surface
	CodeOmissionContradictsCommand = "cli.omission_contradicts_command" // omission says a live command is absent

	// CLI entrypoint-structure codes. These are static Go checks over cli/main.go,
	// scoped to the process boundary only. They do not replace architecture or
	// tidiness providers; they enforce the CLI-specific contract that main()
	// delegates to the scenario app/command runner instead of owning business or
	// server setup directly.
	CodeCLIMainUnreadable = "cli.main_unreadable"
	CodeCLIMainHeavy      = "cli.main_heavy"

	// Command-architecture maturity codes. Classify a CLI's convergence on
	// cli-core renderer-separated primitives from declared manifest metadata and
	// structural evidence — never handler AST heuristics or live command-parity
	// probing. See scenarios/cli-health/docs/reference/cli-architecture-maturity.md.
	// The un-migrated-fleet codes are WARNING + clean_requirement=required: they
	// cap the capability rung and count as honest maturity debt, but do NOT fail
	// the phase (only ERROR/BLOCKER do). The two ERROR codes fire only after a
	// scenario opts into architecture metadata and declares it wrong.
	CodeArchUnclassifiable    = "arch.unclassifiable"             // has a CLI/proto surface but no manifest to classify from
	CodeArchPrimitiveUndecl   = "arch.primitive_undeclared"       // manifest-bound command declares no architecture.primitive
	CodeArchPrimitiveUnverif  = "arch.primitive_unverified"       // command declares a primitive but cli-core reported no matching evidence
	CodeArchPrimitiveMismatch = "arch.primitive_mismatch"         // declared primitive and cli-core-observed primitive disagree (gating)
	CodeArchMetadataInvalid   = "arch.metadata_invalid"           // exceptions[]/architecture metadata is stale or malformed
	CodeArchClaimedViolation  = "arch.claimed_maturity_violation" // exceptions[] entry contradicts a normal manifest-bound proto command
	CodeArchEvidenceMalformed = "arch.evidence_malformed"         // the static primitive-evidence artifact exists but is unparseable/wrong-schema (gating)
	CodeArchEvidenceStale     = "arch.evidence_stale"             // the artifact's manifest hash no longer matches cli/manifest.json (advisory; regenerate)
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
