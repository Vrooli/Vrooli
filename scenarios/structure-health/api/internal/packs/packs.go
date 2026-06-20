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
	"strings"

	"structure-health/internal/packs/auditrules"
	"structure-health/internal/packs/scan"
	"structure-health/internal/rules"

	developsteps "structure-health/internal/packs/configpack/developsteps"
	envvalidation "structure-health/internal/packs/configpack/envvalidation"
	hardcodedvalues "structure-health/internal/packs/configpack/hardcodedvalues"
	healthlifecycle "structure-health/internal/packs/configpack/healthlifecycle"
	ports "structure-health/internal/packs/configpack/ports"
	runtimestorage "structure-health/internal/packs/configpack/runtimestorage"
	setupconditions "structure-health/internal/packs/configpack/setupconditions"
	setupsteps "structure-health/internal/packs/configpack/setupsteps"
	teststeps "structure-health/internal/packs/configpack/teststeps"
	requiredlayout "structure-health/internal/packs/structurepack/requiredlayout"
	testcoverage "structure-health/internal/packs/structurepack/testcoverage"
	uilifecyclelaunch "structure-health/internal/packs/structurepack/uilifecyclelaunch"
	uistructure "structure-health/internal/packs/structurepack/uistructure"
	focusvisible "structure-health/internal/packs/uipack/focusvisible"
	helmetframe "structure-health/internal/packs/uipack/helmetframe"
	spatialnav "structure-health/internal/packs/uipack/spatialnav"
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

	// ---- config pack: service.json rules ----
	{
		Code: "PROFILE_DEVELOP_STEPS", Name: "Develop Lifecycle Steps",
		Feed: feedServiceJSON, Severity: "medium",
		run: func(c, p, s string) []auditrules.Violation {
			return developsteps.CheckDevelopLifecycleSteps([]byte(c), p)
		},
	},
	{
		Code: "PROFILE_HEALTH_LIFECYCLE", Name: "Health Lifecycle Event",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return healthlifecycle.CheckServiceHealthLifecycle([]byte(c), p)
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
		Code: "PROFILE_SETUP_CONDITIONS", Name: "Setup Conditions",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation {
			return setupconditions.CheckServiceSetupConditions([]byte(c), p)
		},
	},
	{
		Code: "PROFILE_TEST_STEPS", Name: "Test Lifecycle Steps",
		Feed: feedServiceJSON, Severity: "high",
		run: func(c, p, s string) []auditrules.Violation { return teststeps.CheckLifecycleTestSteps([]byte(c), p) },
	},
	{
		Code: "PROFILE_SETUP_STEPS", Name: "Setup Steps Configuration",
		Feed: feedServiceJSON, Severity: "medium",
		run: func(c, p, s string) []auditrules.Violation {
			return setupsteps.CheckSetupStepsConfiguration([]byte(c), p)
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
		Code: "PROFILE_HARDCODED_VALUES", Name: "No Hardcoded Values",
		Feed: feedPerFile, Target: scan.TargetTest, Severity: "high",
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

	// ---- ui pack: per-file ui rules ----
	{
		Code: "PROFILE_FOCUS_VISIBLE", Name: "Focus Visible Styles",
		Feed: feedPerFile, Target: scan.TargetUI, Severity: "low", Surface: "ui",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := focusvisible.Check(c, p, s)
			return v
		},
	},
	{
		Code: "PROFILE_HELMET_FRAME", Name: "Helmet Frame Ancestors",
		Feed: feedPerFile, Target: scan.TargetUI, Severity: "high", Surface: "ui",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := helmetframe.Check(c, p, s)
			return v
		},
	},
	{
		Code: "PROFILE_SPATIAL_NAV", Name: "Spatial Nav Provider",
		Feed: feedPerFile, Target: scan.TargetUI, Severity: "medium", Surface: "ui",
		run: func(c, p, s string) []auditrules.Violation {
			v, _ := spatialnav.Check(c, p, s)
			return v
		},
	},
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

func toFinding(e entry, v auditrules.Violation, sc *scan.Context, enforce bool) rules.Finding {
	sev := mapSeverity(firstNonEmpty(v.Severity, e.Severity))
	code := e.Code
	if !enforce {
		// Advisory for unrecognized profiles: never block, and surface that the
		// finding is a profile-convention check that may not apply.
		sev = "warning"
		code = "PROFILE_CONFORMANCE_VIOLATION"
	}
	return rules.Finding{
		Code:        code,
		Severity:    sev,
		Title:       firstNonEmpty(v.Title, e.Name),
		Message:     firstNonEmpty(v.Message, v.Description),
		Location:    relativize(firstNonEmpty(v.FilePath, v.File), sc.RootPath),
		Remediation: v.Recommendation,
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
