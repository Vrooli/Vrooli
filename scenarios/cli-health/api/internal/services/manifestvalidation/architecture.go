package manifestvalidation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// EvidenceArtifactStatus classifies the static primitive-evidence artifact the
// provider tried to read for a scenario. The zero value (EvidenceArtifactOK)
// means the Primitives are trustworthy — either a good artifact or the honest
// no-evidence rollout state (empty Primitives). Malformed/Stale mean the artifact
// exists but cannot be trusted, so its evidence is ignored and an explicit
// artifact-level finding is emitted.
type EvidenceArtifactStatus string

const (
	// EvidenceArtifactOK: trust Primitives as-is. An absent artifact is OK with
	// empty Primitives — declared primitives then classify as unverified debt.
	EvidenceArtifactOK EvidenceArtifactStatus = ""
	// EvidenceArtifactMalformed: the artifact file exists but is unparseable or
	// carries an unrecognized schema. Its evidence is ignored (declared →
	// unverified) and arch.evidence_malformed is emitted (gating error).
	EvidenceArtifactMalformed EvidenceArtifactStatus = "malformed"
	// EvidenceArtifactStale: the artifact parses but its recorded manifest hash no
	// longer matches the on-disk manifest, so its evidence describes an older CLI
	// surface. Its evidence is ignored (declared → unverified) and
	// arch.evidence_stale is emitted (advisory — regenerate the artifact).
	EvidenceArtifactStale EvidenceArtifactStatus = "stale"
)

// ArchitectureEvidence holds the cli-core primitive class each of a scenario's
// commands was ACTUALLY built from, keyed by normalized command path
// ("group command"). It is the unforgeable structural proof cli-health compares
// against the manifest's declared architecture.primitive: a declaration is
// forgeable text, but the observed primitive travels onto Command.PrimitiveEvidence
// by construction (only a cli-core primitive builder can produce it). Absence of
// an entry means "no evidence observed for this command" — declared-only debt,
// never verified.
type ArchitectureEvidence struct {
	// Primitives maps normalized command path -> observed cli-core PrimitiveClass.
	Primitives map[string]cliapp.PrimitiveClass
	// Status is the trust state of the artifact the evidence was read from.
	// Non-OK statuses mean Primitives must be ignored and an artifact-level
	// finding emitted; the zero value trusts Primitives.
	Status EvidenceArtifactStatus
	// ArtifactPath is where the provider looked for the artifact (for finding
	// locations / remediation). Empty when no artifact channel was consulted.
	ArtifactPath string
	// Detail carries the parse error text for a malformed artifact.
	Detail string
}

// trusted reports whether the observed Primitives may be used to verify declared
// primitives. Malformed/Stale artifacts are not trusted.
func (e ArchitectureEvidence) trusted() bool {
	return e.Status == EvidenceArtifactOK
}

// Primitive returns the observed primitive class for a command path (normalized
// on lookup), or "" when no evidence is available (or the artifact is not
// trusted).
func (e ArchitectureEvidence) Primitive(path string) cliapp.PrimitiveClass {
	if !e.trusted() || e.Primitives == nil {
		return ""
	}
	return e.Primitives[normalizeCommandPath(path)]
}

// commandArchObservation pairs what a manifest command DECLARES (forgeable text)
// with what cli-core OBSERVED when the handler was built (unforgeable evidence).
// Threading both onto one value is how the classifier separates verified
// primitive adoption from declaration-only maturity debt and from contradiction.
type commandArchObservation struct {
	group    string
	command  string
	declared cliapp.CommandArchitecture
	observed cliapp.PrimitiveClass
}

// This file implements CLI command-architecture maturity classification. It
// derives maturity purely from declared structure — the manifest's per-command
// architecture block, the top-level exceptions[] declarations, and (when
// execution is requested) the runtime command surface — with NO handler AST
// heuristics and NO live command-parity probing, per the plan constraints. See
// scenarios/cli-health/docs/reference/cli-architecture-maturity.md for the
// ladder and rollout policy.

// architectureStaticFindings classifies command architecture from the parsed
// manifest and the cli-core primitive evidence observed for the scenario (empty
// evidence during rollout — no runtime probe required). It emits, per bound
// command, exactly one of:
//   - arch.primitive_undeclared when the command declares no architecture at all
//     (the manifest/API-bound honest-debt signal; caps the rung at L3);
//   - arch.primitive_unverified when the command declares a primitive/exception
//     but cli-core reported no matching evidence — declared intent that is not
//     yet proven (advisory debt, caps at L3, never a verified L4);
//   - arch.primitive_mismatch when the declared primitive and the observed
//     cli-core primitive disagree (a contradiction — gating error);
//   - nothing when the declaration is verified by matching evidence (L4).
//
// It also emits arch.claimed_maturity_violation when a top-level exceptions[]
// entry names a command that is actually a normal manifest-bound proto command
// (a false special-case claim — gating).
func architectureStaticFindings(m *cliapp.Manifest, evidence ArchitectureEvidence, manifestPath string) []Finding {
	var findings []Finding

	for _, g := range m.Groups {
		for _, c := range g.Commands {
			obs := commandArchObservation{
				group:    g.Name,
				command:  c.Name,
				declared: c.Architecture.CommandArchitecture(),
				observed: evidence.Primitive(groupCmd(g.Name, c.Name)),
			}
			if f, ok := classifyCommandArchitecture(obs, manifestPath); ok {
				findings = append(findings, f)
			}
		}
	}

	bound := manifestCommandPaths(m)
	for _, e := range m.Exceptions {
		commandPath := normalizeCommandPath(e.Command)
		if bound[commandPath] {
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       CodeArchClaimedViolation,
				Location:   manifestPath + "#/exceptions",
				Message:    fmt.Sprintf("exceptions[] declares %q as a %s special case, but it is a normal manifest-bound proto command", e.Command, e.Class),
				Suggestion: "remove the exceptions[] entry (a plain proto command needs no exception), or move the command out of the manifest binding path if it truly has a special lifecycle",
			})
			continue
		}
		obs := commandArchObservation{
			command:  commandPath,
			declared: e.CommandArchitecture(),
			observed: evidence.Primitive(commandPath),
		}
		if f, ok := classifyCommandArchitecture(obs, manifestPath+"#/exceptions"); ok {
			findings = append(findings, f)
		}
	}

	findings = append(findings, evidenceArtifactFindings(evidence)...)

	return findings
}

// evidenceArtifactFindings surfaces an untrusted static evidence artifact as one
// explicit architecture finding. A malformed artifact is a gating error (the
// committed contract file is broken); a stale artifact is advisory (regenerate
// it). In both cases the per-command evidence is already ignored (Primitive()
// returns "" when the artifact is not trusted), so declared primitives correctly
// classify as unverified rather than falsely reaching verified L4.
func evidenceArtifactFindings(evidence ArchitectureEvidence) []Finding {
	loc := evidence.ArtifactPath
	if loc == "" {
		loc = cliapp.EvidenceArtifactRelPath
	}
	switch evidence.Status {
	case EvidenceArtifactMalformed:
		msg := "the static primitive-evidence artifact is malformed and cannot be trusted"
		if strings.TrimSpace(evidence.Detail) != "" {
			msg = fmt.Sprintf("%s: %s", msg, evidence.Detail)
		}
		return []Finding{{
			Severity:   SeverityError,
			Code:       CodeArchEvidenceMalformed,
			Location:   loc,
			Message:    msg,
			Suggestion: "regenerate the artifact from the scenario's evidence generator (build the command tree and cliapp.WritePrimitiveEvidence); do not hand-edit it",
		}}
	case EvidenceArtifactStale:
		return []Finding{{
			Severity:   SeverityWarning,
			Code:       CodeArchEvidenceStale,
			Location:   loc,
			Message:    "the static primitive-evidence artifact is stale — its recorded manifest hash no longer matches cli/manifest.json, so its evidence describes an older command surface",
			Suggestion: "regenerate the artifact after changing the manifest or handlers (run the scenario's evidence generator, e.g. its CLI test with UPDATE_CLI_EVIDENCE=1)",
		}}
	default:
		return nil
	}
}

// classifyCommandArchitecture turns one command's declared-vs-observed
// architecture into at most one Finding. It is the maturity heart of the
// verified-primitive contract: it reuses cliapp.ClassifyPrimitiveEvidence so
// cli-health and cli-core agree, byte-for-byte, on what "verified" means.
func classifyCommandArchitecture(obs commandArchObservation, manifestPath string) (Finding, bool) {
	loc := commandLocation(manifestPath, obs.group, obs.command)
	label := groupCmd(obs.group, obs.command)

	// No architecture declared at all: manifest/API-bound but unclassified.
	if obs.declared.IsZero() {
		return Finding{
			Severity:   SeverityWarning,
			Code:       CodeArchPrimitiveUndecl,
			Location:   loc,
			Message:    fmt.Sprintf("command %q is manifest/API-bound but declares no architecture.primitive, so renderer separation cannot be confirmed", label),
			Suggestion: "declare architecture.primitive (proto_list/proto_mutation/operational/action) and build the handler with the matching cli-core primitive (cliapp.ProtoList/ProtoMutation/ProtoOperational)",
		}, true
	}

	// Declared a normal or special-case primitive: compare against the primitive
	// cli-core actually built the handler from.
	if obs.declared.Primitive != "" {
		switch cliapp.ClassifyPrimitiveEvidence(obs.declared.Primitive, obs.observed) {
		case cliapp.EvidenceVerified:
			return Finding{}, false
		case cliapp.EvidenceContradiction:
			return Finding{
				Severity:   SeverityError,
				Code:       CodeArchPrimitiveMismatch,
				Location:   loc,
				Message:    fmt.Sprintf("command %q declares architecture.primitive %q but cli-core observed the %q primitive — the declaration and implementation disagree", label, obs.declared.Primitive, obs.observed),
				Suggestion: "fix the manifest declaration to match the primitive the handler is built from, or rebuild the handler with the declared cli-core primitive",
			}, true
		default: // EvidenceDeclaredOnly (no observed evidence)
			return Finding{
				Severity:   SeverityWarning,
				Code:       CodeArchPrimitiveUnverif,
				Location:   loc,
				Message:    fmt.Sprintf("command %q declares architecture.primitive %q but cli-core reported no matching primitive evidence, so renderer separation is declared but not yet verified", label, obs.declared.Primitive),
				Suggestion: "build the handler with the matching cli-core primitive builder and register it via cliapp.LoadFromManifestPrimitives so the observed primitive proves the declaration",
			}, true
		}
	}

	// Declared a per-command exception (special-case shape, no normal primitive).
	// The observed evidence must be the special-case primitive that satisfies the
	// declared exception class.
	switch {
	case obs.observed == "":
		return Finding{
			Severity:   SeverityWarning,
			Code:       CodeArchPrimitiveUnverif,
			Location:   loc,
			Message:    fmt.Sprintf("command %q declares a %s exception but cli-core reported no matching primitive evidence, so the special-case shape is declared but not yet verified", label, obs.declared.Exception),
			Suggestion: "build the handler with the matching special-case cli-core primitive and register it via cliapp.LoadFromManifestPrimitives so the observed primitive proves the exception",
		}, true
	case obs.observed.SatisfiesException() == obs.declared.Exception:
		return Finding{}, false
	default:
		return Finding{
			Severity:   SeverityError,
			Code:       CodeArchPrimitiveMismatch,
			Location:   loc,
			Message:    fmt.Sprintf("command %q declares a %s exception but cli-core observed the %q primitive, which satisfies exception %q — the declaration and implementation disagree", label, obs.declared.Exception, obs.observed, obs.observed.SatisfiesException()),
			Suggestion: "fix the manifest exception class to match the primitive the handler is built from, or rebuild the handler with the primitive that satisfies the declared exception",
		}, true
	}
}

// architectureParseFindings surfaces architecture-metadata problems as arch.*
// finding codes even when the manifest fails to parse for an architecture
// reason. cliapp.ParseManifest rejects a malformed architecture block as a
// generic parse error; without this, the command_architecture capability would
// see no arch.* finding at all and be mis-scored by the generic manifest.parse_error.
// It decodes permissively (no strict field/binding checks) purely to reach the
// architecture blocks and re-runs the canonical CommandArchitecture.Validate.
func architectureParseFindings(raw []byte, manifestPath string) []Finding {
	var m cliapp.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		// Not decodable at all — a generic parse problem, not an architecture one.
		return nil
	}
	var findings []Finding
	for _, g := range m.Groups {
		for _, c := range g.Commands {
			if err := c.Architecture.CommandArchitecture().Validate(); err != nil {
				findings = append(findings, Finding{
					Severity:   SeverityError,
					Code:       CodeArchMetadataInvalid,
					Location:   commandLocation(manifestPath, g.Name, c.Name),
					Message:    fmt.Sprintf("command %q architecture metadata is invalid: %v", groupCmd(g.Name, c.Name), err),
					Suggestion: "fix architecture.primitive/exception to a valid, non-contradictory combination (see .vrooli/schemas/cli-manifest.schema.json)",
				})
			}
		}
	}
	for _, e := range m.Exceptions {
		if err := e.CommandArchitecture().Validate(); err != nil {
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       CodeArchMetadataInvalid,
				Location:   manifestPath + "#/exceptions",
				Message:    fmt.Sprintf("exceptions[] entry for %q has invalid architecture metadata: %v", e.Command, err),
				Suggestion: "fix the exception class/reason to a valid combination (see .vrooli/schemas/cli-manifest.schema.json)",
			})
		}
	}
	return findings
}

// architectureRuntimeFindings adds the runtime-dependent architecture check:
// arch.metadata_invalid for a top-level exceptions[] entry whose declared
// command is not exposed at runtime (a stale/typo declaration — gating, opt-in).
// It runs only when the caller requested execution and the probe resolved the
// binary.
func architectureRuntimeFindings(m *cliapp.Manifest, obs RuntimeObservation, manifestPath string) []Finding {
	if !obs.Resolved || obs.HelpFailed {
		return nil
	}
	bound := manifestCommandPaths(m)
	var findings []Finding
	for _, e := range m.Exceptions {
		p := normalizeCommandPath(e.Command)
		if runtimeHasCommandPath(obs, p) || bound[p] {
			continue
		}
		findings = append(findings, Finding{
			Severity:   SeverityError,
			Code:       CodeArchMetadataInvalid,
			Location:   manifestPath + "#/exceptions",
			Message:    fmt.Sprintf("exceptions[] declares special-case command %q but the CLI does not expose it at runtime (stale declaration?)", e.Command),
			Suggestion: "fix the command path to match the runtime surface, or remove the stale exceptions[] entry",
		})
	}
	return findings
}

func runtimeHasCommandPath(obs RuntimeObservation, path string) bool {
	runtime := runtimeCommandPaths(obs)
	if runtime[path] {
		return true
	}
	// cli-core help groups top-level commands under display sections (for
	// example "Suites" or "Local"). A manifest exception path with no spaces is
	// a top-level command, so match it by leaf name as well as the parser's
	// section-qualified path.
	if strings.Contains(path, " ") {
		return false
	}
	for _, c := range obs.Commands {
		if normalizeCommandPath(c.Name) == path {
			return true
		}
	}
	return false
}

// manifestCommandPaths returns the set of normalized command paths declared in
// the manifest groups ("group name" joined, top-level commands are just "name").
func manifestCommandPaths(m *cliapp.Manifest) map[string]bool {
	out := map[string]bool{}
	for _, g := range m.Groups {
		for _, c := range g.Commands {
			out[normalizeCommandPath(groupCmd(g.Name, c.Name))] = true
		}
	}
	return out
}

// exceptionCommandPaths returns the set of normalized command paths declared as
// legitimate special cases in the manifest's top-level exceptions[]. The runtime
// command-surface reconciler treats these as declared (not "undeclared"), so a
// declared exception silences cli.command_undeclared for that command.
func exceptionCommandPaths(m *cliapp.Manifest) map[string]bool {
	out := map[string]bool{}
	if m == nil {
		return out
	}
	for _, e := range m.Exceptions {
		out[normalizeCommandPath(e.Command)] = true
	}
	return out
}

func runtimeCommandPaths(obs RuntimeObservation) map[string]bool {
	out := map[string]bool{}
	for _, c := range obs.Commands {
		out[normalizeCommandPath(groupCmd(c.Group, c.Name))] = true
	}
	return out
}

func normalizeCommandPath(path string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(path)), " ")
}
