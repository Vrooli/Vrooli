// Package packs runs the profile-keyed conformance rule packs migrated out of
// scenario-auditor (its structure/config/ui rules) plus test-genie's native
// structure checks. The default profile (react-vite-go) runs every pack with
// its original severity so verdicts match the pre-migration combined output;
// an unrecognized profile downgrades every pack finding to advisory so a
// non-react-vite/Go scenario is never falsely failed by react-vite-specific
// conventions.
package packs

import (
	"path/filepath"
	"regexp"
	"strings"

	"structure-health/internal/packs/auditrules"
	"structure-health/internal/packs/scan"
	"structure-health/internal/packs/targetpack"
	"structure-health/internal/rules"

	envvalidation "structure-health/internal/packs/configpack/envvalidation"
	hardcodedvalues "structure-health/internal/packs/configpack/hardcodedvalues"
	healthlifecycle "structure-health/internal/packs/configpack/healthlifecycle"
	manifestschema "structure-health/internal/packs/configpack/manifestschema"
	ports "structure-health/internal/packs/configpack/ports"
	runtimestorage "structure-health/internal/packs/configpack/runtimestorage"
	requiredlayout "structure-health/internal/packs/structurepack/requiredlayout"
	testcoverage "structure-health/internal/packs/structurepack/testcoverage"
	uilifecyclelaunch "structure-health/internal/packs/structurepack/uilifecyclelaunch"
	uistructure "structure-health/internal/packs/structurepack/uistructure"
)

// DefaultProfileID is the profile whose conformance packs reproduce the
// pre-migration scenario-auditor + test-genie structure verdicts verbatim.
const DefaultProfileID = "react-vite-go"

type feed int

const (
	// feedStructure runs once with the {scenario, files} payload + the scenario
	// root path (the auditor's structure-rule contract).
	feedStructure feed = iota
	// feedServiceJSON runs once with .vrooli/service.json content.
	feedServiceJSON
	// feedPerFile runs per file matching Target (api/cli/ui/test).
	feedPerFile
)

// runner adapts a migrated rule's (non-uniform) signature to a single shape.
type runner func(content string, path string, scenario string) []auditrules.Violation

// entry is one migrated rule wired with its structure-health finding code,
// content-feed strategy, and default severity (used when a violation omits one).
type entry struct {
	Code     string
	Name     string
	Feed     feed
	Target   string
	Severity string
	Surface  string
	run      runner
}

// registry is the full migrated rule set. Order is stable for deterministic
// finding output.
var registry = []entry{
	// ---- structure pack (file-list payload + scenario root) ----
	{
		Code: "PROFILE_REQUIRED_STRUCTURE", Name: "Scenario Required Structure",
		Feed: feedStructure, Severity: "critical",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := requiredlayout.Check(c, p, s)
			return v
		},
	},
	{
		Code: "PROFILE_TEST_COVERAGE", Name: "Test Package Coverage",
		Feed: feedStructure, Severity: "medium",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := testcoverage.CheckTestFileCoverage([]byte(c), p, s)
			return v
		},
	},
	{
		Code: "PROFILE_UI_STRUCTURE", Name: "Scenario UI Structure",
		Feed: feedStructure, Severity: "high", Surface: "ui",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := uistructure.CheckUIStructure([]byte(c), p, s)
			return v
		},
	},
	{
		Code: "PROFILE_UI_LIFECYCLE_LAUNCH", Name: "UI Lifecycle Launch",
		Feed: feedStructure, Severity: "high", Surface: "ui",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := uilifecyclelaunch.CheckUISharedLifecycleLaunch([]byte(c), p, s)
			return v
		},
	},
	{
		Code: "SCENARIO_SHARED_PACKAGE_BYPASS", Name: "Shared Package Source Bypass",
		Feed: feedPerFile, Target: scan.TargetUI, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return checkSharedPackageBypass(c, p)
		},
	},

	// ---- config pack: service.json rules ----
	{
		Code: "PROFILE_HEALTH_LIFECYCLE", Name: "Health Lifecycle Event",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return healthlifecycle.CheckServiceHealthLifecycle([]byte(c), p)
		},
	},
	{
		Code: "PROFILE_STOP_PROCESS_OWNERSHIP", Name: "Stop Process Ownership",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return healthlifecycle.CheckStopProcessOwnership([]byte(c), p)
		},
	},
	{
		Code: "PROFILE_PORTS", Name: "Ports Configuration",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return ports.CheckServicePortConfiguration([]byte(c), p, s)
		},
	},
	{
		Code: "SCENARIO_PORT_ENV_CONVENTION", Name: "Scenario Port Environment Convention",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return ports.CheckPortEnvConvention([]byte(c), p)
		},
	},
	{
		// The only whole-document rule for a scenario manifest. Every sibling
		// rule here inspects one hand-picked shape; this one runs the schema.
		Code: "SCENARIO_MANIFEST_INVALID", Name: "Scenario Manifest Schema",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckServiceManifestSchema([]byte(c), p)
		},
	},
	{
		Code: "SCENARIO_SHELL_FORBIDDEN", Name: "Scenario Shell-Free Declaration",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckScenarioShellInvocations([]byte(c), p)
		},
	},
	{
		Code: "SCENARIO_COMPONENT_INVALID", Name: "Scenario Component Contract",
		Feed: feedServiceJSON, Severity: "warning",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckScenarioComponents([]byte(c), p)
		},
	},
	{
		Code: "SCENARIO_PEER_BINDING_INVALID", Name: "Scenario Peer Binding",
		Feed: feedServiceJSON, Severity: "warning",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckScenarioPeerBindings([]byte(c), p)
		},
	},
	{
		Code: "SCENARIO_HARDCODED_PEER_ADDRESS", Name: "Scenario Hardcoded Peer Address",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckScenarioHardcodedPeerAddress([]byte(c), p)
		},
	},
	{
		Code: "SCENARIO_SECRET_LITERAL", Name: "Scenario Secret Literal",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckScenarioSecretLiteral([]byte(c), p)
		},
	},
	{
		Code: "SCENARIO_REDECLARES_RESOURCE_ENV", Name: "Scenario Resource Environment Redeclaration",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckScenarioRedeclaresResourceEnv([]byte(c), p)
		},
	},
	{
		Code: "SCENARIO_UI_SERVES_BUILD", Name: "Scenario UI Serves Production Build",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckScenarioUIServesBuild([]byte(c), p)
		},
	},
	{
		Code: "SCENARIO_BUILD_KIND_UNKNOWN", Name: "Scenario Build Kind",
		Feed: feedServiceJSON, Severity: "warning",
		run: func(c, p, s string) []auditrules.Violation {
			return manifestschema.CheckScenarioBuildKinds([]byte(c), p)
		},
	},

	// ---- config pack: per-file source rules ----
	{
		Code: "PROFILE_ENV_VALIDATION", Name: "Environment Variable Validation",
		Feed: feedPerFile, Target: scan.TargetAPI, Severity: "medium",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := (&envvalidation.EnvValidationRule{}).Check(c, p)
			return v
		},
	},
	{
		Code: "PROFILE_ENV_VALIDATION", Name: "Environment Variable Validation",
		Feed: feedPerFile, Target: scan.TargetCLI, Severity: "medium",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := (&envvalidation.EnvValidationRule{}).Check(c, p)
			return v
		},
	},
	{
		Code: "PROFILE_ENV_VALIDATION", Name: "Environment Variable Validation",
		Feed: feedPerFile, Target: scan.TargetUI, Severity: "medium",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := (&envvalidation.EnvValidationRule{}).Check(c, p)
			return v
		},
	},
	{
		Code: "PROFILE_HARDCODED_VALUES", Name: "No Hardcoded Values",
		Feed: feedPerFile, Target: scan.TargetAPI, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation { return hardcodedvalues.CheckHardcodedValues([]byte(c), p) },
	},
	{
		Code: "PROFILE_HARDCODED_VALUES", Name: "No Hardcoded Values",
		Feed: feedPerFile, Target: scan.TargetCLI, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation { return hardcodedvalues.CheckHardcodedValues([]byte(c), p) },
	},
	{
		Code: "PROFILE_HARDCODED_VALUES", Name: "No Hardcoded Values",
		Feed: feedPerFile, Target: scan.TargetUI, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation { return hardcodedvalues.CheckHardcodedValues([]byte(c), p) },
	},
	{
		Code: "PROFILE_RUNTIME_STORAGE", Name: "No Legacy Runtime Storage Paths",
		Feed: feedPerFile, Target: scan.TargetAPI, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return runtimestorage.CheckRepoLocalRuntimeStorage([]byte(c), p, s)
		},
	},
	{
		Code: "PROFILE_RUNTIME_STORAGE", Name: "No Legacy Runtime Storage Paths",
		Feed: feedPerFile, Target: scan.TargetCLI, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return runtimestorage.CheckRepoLocalRuntimeStorage([]byte(c), p, s)
		},
	},
	{
		Code: "PROFILE_RUNTIME_STORAGE", Name: "No Legacy Runtime Storage Paths",
		Feed: feedPerFile, Target: scan.TargetUI, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return runtimestorage.CheckRepoLocalRuntimeStorage([]byte(c), p, s)
		},
	},
	{
		Code: "PROFILE_RUNTIME_STORAGE", Name: "No Legacy Runtime Storage Paths",
		Feed: feedPerFile, Target: scan.TargetTest, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return runtimestorage.CheckRepoLocalRuntimeStorage([]byte(c), p, s)
		},
	},
	// NOTE: the per-file UI pack rules (focus-visible, helmet-frame-ancestors,
	// spatial-nav-provider) moved to the ui-health scenario, which is now the
	// single authority for all static UI-interop validation. structure-health
	// keeps only generic skeleton facts.
}

var sharedPackageSourcePath = regexp.MustCompile("packages/[^/\\s\\\"']+/src(?:/|[\\\"'])")

func checkSharedPackageBypass(content, path string) []auditrules.Violation {
	if !sharedPackageSourcePath.MatchString(content) {
		return nil
	}
	return []auditrules.Violation{{
		Severity:       "high",
		Title:          "Shared package source bypass",
		Message:        "scenario UI resolves a shared package directly from packages/*/src instead of consuming its governed package output",
		FilePath:       path,
		Recommendation: "Remove the source-tree alias and consume the package through its declared file dependency and compiled exports.",
	}}
}

// Evaluate runs every conformance pack against the scanned scenario and returns
// the findings. When the profile is not the recognized default, findings are
// emitted as advisory (severity downgraded to warning) so an unrecognized
// scenario shape is never falsely failed.
func Evaluate(profileID string, recognized bool, sc *scan.Context) []rules.Finding {
	enforce := recognized && profileID == DefaultProfileID
	var out []rules.Finding
	for _, e := range registry {
		switch e.Feed {
		case feedStructure:
			for _, v := range e.run(sc.StructurePayload(), sc.RootPath, sc.Scenario) {
				out = append(out, toFinding(e, v, sc, enforce))
			}
		case feedServiceJSON:
			if f, ok := sc.ServiceJSON(); ok {
				for _, v := range e.run(f.Content, f.AbsPath, sc.Scenario) {
					out = append(out, toFinding(e, v, sc, enforce))
				}
			}
		case feedPerFile:
			for _, f := range sc.FilesForTarget(e.Target) {
				for _, v := range e.run(f.Content, f.AbsPath, sc.Scenario) {
					out = append(out, toFinding(e, v, sc, enforce))
				}
			}
		}
	}
	return out
}

// EvaluateTarget runs the kind-keyed source-target packs. It is deliberately
// separate from profile evaluation so scenario findings cannot leak into a
// resource, tool, or safeguard verdict.
func EvaluateTarget(kind, root, id string) []rules.Finding {
	return targetpack.Evaluate(kind, root, id)
}

// EvaluateTargetWithParseUnits keeps Code Facts evidence attached to package
// validation without making targetpack depend on the profile client.
func EvaluateTargetWithParseUnits(kind, root, id string, units []targetpack.ParseUnit) []rules.Finding {
	return targetpack.EvaluateWithParseUnits(kind, root, id, units)
}

func toFinding(e entry, v auditrules.Violation, sc *scan.Context, enforce bool) rules.Finding {
	sev := mapSeverity(firstNonEmpty(v.Severity, e.Severity))
	code := e.Code
	remediation := v.Recommendation
	if !enforce {
		// Advisory for unrecognized profiles: never block, and surface that the
		// finding is a profile-convention check that may not apply.
		sev = "warning"
		code = "PROFILE_CONFORMANCE_VIOLATION"
	}
	if catalogEntry, ok := rules.Lookup(code); ok {
		sev = catalogEntry.Severity
		remediation = catalogEntry.Remediation
	}
	return rules.Finding{
		Code:        code,
		Severity:    sev,
		Title:       firstNonEmpty(v.Title, e.Name),
		Message:     firstNonEmpty(v.Message, v.Description),
		Location:    relativize(firstNonEmpty(v.FilePath, v.File), sc.RootPath),
		Remediation: remediation,
		Surface:     e.Surface,
	}
}

// mapSeverity maps the auditor severity vocabulary to structure-health's.
// critical/high block (error); medium/low/info are advisory (warning).
func mapSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "high", "error", "blocker":
		return "error"
	default:
		return "warning"
	}
}

// relativize converts an absolute path under root to a scenario-relative slash
// path; non-absolute or out-of-tree paths are returned slash-normalized as-is.
func relativize(p, root string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		if rel, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(p)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
