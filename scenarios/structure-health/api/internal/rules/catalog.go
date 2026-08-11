package rules

import (
	"fmt"
	"sort"
)

// EnforcementLevel describes how a registered policy participates in a
// verdict. Keeping this explicit prevents an unimplemented policy from
// disappearing from coverage reports.
type EnforcementLevel string

const (
	Enforced   EnforcementLevel = "enforced"
	Advisory   EnforcementLevel = "advisory"
	Unenforced EnforcementLevel = "none"
)

// CatalogEntry is the machine-readable record for one structural policy.
// Claims are stable identifiers used by documentation to link prose to the
// rule that actually proves it; an empty claim is valid for implementation
// details that are not presented as a prose enforcement claim.
type CatalogEntry struct {
	Code         string           `json:"code"`
	TargetKind   string           `json:"target_kind"`
	Severity     string           `json:"severity"`
	Enforcement  EnforcementLevel `json:"enforcement"`
	WhatItChecks string           `json:"what_it_checks"`
	Remediation  string           `json:"remediation"`
	Claim        string           `json:"claim,omitempty"`
}

type CoverageRow struct {
	TargetKind  string `json:"target_kind"`
	RuleCount   int    `json:"rule_count"`
	Enforced    int    `json:"enforced"`
	Advisory    int    `json:"advisory"`
	Unenforced  int    `json:"none"`
	Reachable   bool   `json:"reachable"`
	CallerCount int    `json:"caller_count"`
}

var targetKinds = []string{"scenario", "resource", "tool", "safeguard", "team", "package", "control-plane", "docs", "project"}

var callersByTargetKind = map[string][]string{
	"scenario":      {"ValidationService.ValidateTarget", "cli validate scenario"},
	"resource":      {"ValidationService.ValidateTarget", "cli validate resource"},
	"tool":          {"ValidationService.ValidateTarget", "cli validate tool"},
	"safeguard":     {"ValidationService.ValidateTarget", "cli validate safeguard"},
	"team":          {"ValidationService.ValidateTarget", "cli validate team"},
	"package":       {"ValidationService.ValidateTarget", "cli validate package"},
	"control-plane": {"ValidationService.ValidateTarget", "cli validate control-plane"},
	"docs":          {"ValidationService.ValidateTarget", "cli validate docs"},
	"project":       {"ValidationService.ValidateTarget", "cli validate project"},
}

// Coverage computes the catalog view used by the rules coverage command. The
// Caller reachability is derived from the declared API and CLI callers for
// each kind. Hygiene is a transport caller of the same API contract, not a
// second structural implementation.
func Coverage() []CoverageRow {
	rows := make([]CoverageRow, 0, len(targetKinds))
	for _, kind := range targetKinds {
		callers := callersByTargetKind[kind]
		row := CoverageRow{TargetKind: kind, CallerCount: len(callers), Reachable: len(callers) >= 2}
		for _, entry := range catalog {
			if entry.TargetKind != kind {
				continue
			}
			row.RuleCount++
			switch entry.Enforcement {
			case Enforced:
				row.Enforced++
			case Advisory:
				row.Advisory++
			case Unenforced:
				row.Unenforced++
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// TargetKinds returns the contract's declared target-kind vocabulary.
func TargetKinds() []string { return append([]string(nil), targetKinds...) }

var catalog = []CatalogEntry{
	{Code: "PACKAGE_MANIFEST_MISSING", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "Every package has a .vrooli/package.json manifest.", Remediation: "Add a valid .vrooli/package.json package governance manifest.", Claim: "package.manifest"},
	{Code: "PACKAGE_MANIFEST_INVALID", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "Package governance manifests have the required shape.", Remediation: "Repair .vrooli/package.json against the package schema.", Claim: "package.manifest-shape"},
	{Code: "PACKAGE_LAYOUT_MISSING", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "A package has README.md and a language configuration at its root.", Remediation: "Provide README.md and a go.mod or package.json at the package root.", Claim: "package.layout"},
	{Code: "PACKAGE_MODULE_PATH_MISMATCH", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "Declared package module identifiers match discovered module paths.", Remediation: "Add the discovered module path to package.module_identifiers.", Claim: "package.module-identifiers"},
	{Code: "PACKAGE_NAME_MISMATCH", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "The package manifest name matches the target identifier.", Remediation: "Set package.name to the canonical package id.", Claim: "package.identity"},
	{Code: "PACKAGE_OWN_MODULE_MISSING", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "A module-language parse unit rooted at a package has its own module configuration.", Remediation: "Add a module configuration at the package root or record an intentional exception.", Claim: "package.own-module"},
	{Code: "PACKAGE_INTERNAL_IMPORT", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "Packages do not import the control plane's private internal packages.", Remediation: "Promote or duplicate the shared capability and remove the root internal import.", Claim: "package.no-root-internal"},
	{Code: "PACKAGE_GO_REPLACE_MISSING", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "Modules depending on local Vrooli modules repeat the required local replace directives.", Remediation: "Use Scenario Dependency Analyzer to reconcile the module's local replaces.", Claim: "package.go-replaces"},
	{Code: "PACKAGE_BUILD_OUTPUTS_UNDECLARED", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "Every package generate or build lifecycle command declares the files it produces.", Remediation: "Add non-empty repository-relative outputs globs to each generate or build command in .vrooli/package.json.", Claim: "package.build-output-declarations"},
	{Code: "PACKAGE_BUILD_OUTPUTS_COMMITTED", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "Generated package build outputs are not committed to the repository.", Remediation: "Remove generated outputs from version control and ignore the declared output paths.", Claim: "package.no-committed-build-outputs"},
	{Code: "PACKAGE_SOURCE_ENTRYPOINT", TargetKind: "package", Severity: "error", Enforcement: Enforced, WhatItChecks: "JavaScript package entrypoints resolve to compiled output rather than source files.", Remediation: "Point package metadata exports at dist/ or another declared generated output directory.", Claim: "package.compiled-entrypoints"},
	{Code: "PROFILE_CONFORMANCE_VIOLATION", TargetKind: "scenario", Severity: "warning", Enforcement: Advisory, WhatItChecks: "Unrecognized scenario profiles report profile-specific conventions without blocking.", Remediation: "Review the profile-specific finding and either satisfy the convention or use a recognized profile.", Claim: "scenario.profile-advisory"},
	{Code: "PROFILE_DEVELOP_STEPS", TargetKind: "scenario", Severity: "warning", Enforcement: Enforced, WhatItChecks: "Scenario develop lifecycle steps are structurally valid.", Remediation: "Repair the scenario develop lifecycle steps.", Claim: "scenario.develop-steps"},
	{Code: "PROFILE_ENV_VALIDATION", TargetKind: "scenario", Severity: "warning", Enforcement: Enforced, WhatItChecks: "Scenario environment validation follows the profile convention.", Remediation: "Validate environment variables at the scenario boundary.", Claim: "scenario.environment-validation"},
	{Code: "PROFILE_HARDCODED_VALUES", TargetKind: "scenario", Severity: "warning", Enforcement: Enforced, WhatItChecks: "Scenario source does not hardcode runtime configuration values.", Remediation: "Move runtime configuration to validated environment or service configuration.", Claim: "scenario.no-hardcoded-values"},
	{Code: "PROFILE_HEALTH_LIFECYCLE", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario health lifecycle wiring is present and valid.", Remediation: "Repair the scenario health lifecycle configuration.", Claim: "scenario.health-lifecycle"},
	{Code: "PROFILE_PORTS", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario ports follow the declared lifecycle contract.", Remediation: "Repair scenario port declarations and references.", Claim: "scenario.ports"},
	{Code: "PROFILE_REQUIRED_STRUCTURE", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario required structure is present.", Remediation: "Restore the required scenario layout.", Claim: "scenario.required-structure"},
	{Code: "PROFILE_RUNTIME_STORAGE", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario runtime storage uses governed locations.", Remediation: "Move runtime state to the governed runtime-home storage seam.", Claim: "scenario.runtime-storage"},
	{Code: "PROFILE_SETUP_CONDITIONS", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario setup conditions are declared safely.", Remediation: "Repair setup conditions so lifecycle prerequisites are explicit.", Claim: "scenario.setup-conditions"},
	{Code: "PROFILE_SETUP_STEPS", TargetKind: "scenario", Severity: "warning", Enforcement: Enforced, WhatItChecks: "Scenario setup steps are structurally valid.", Remediation: "Repair the scenario setup steps.", Claim: "scenario.setup-steps"},
	{Code: "PROFILE_TEST_COVERAGE", TargetKind: "scenario", Severity: "warning", Enforcement: Enforced, WhatItChecks: "Scenario source has the required test coverage shape.", Remediation: "Add or repair tests for the scenario surface.", Claim: "scenario.test-coverage"},
	{Code: "PROFILE_TEST_STEPS", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario test lifecycle steps are declared.", Remediation: "Repair the scenario test lifecycle steps.", Claim: "scenario.test-steps"},
	{Code: "PROFILE_UI_LIFECYCLE_LAUNCH", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario UI launch wiring follows the lifecycle contract.", Remediation: "Repair UI launch wiring and lifecycle ownership.", Claim: "scenario.ui-lifecycle-launch"},
	{Code: "PROFILE_UI_STRUCTURE", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario UI structure follows the profile convention.", Remediation: "Repair the scenario UI structure.", Claim: "scenario.ui-structure"},
	{Code: "RESOURCE_IMAGE_UNPINNED", TargetKind: "resource", Severity: "error", Enforcement: Enforced, WhatItChecks: "Resource container images are digest pinned.", Remediation: "Pin container images with a sha256 digest.", Claim: "resource.image-pinning"},
	{Code: "RESOURCE_SHELL_FORBIDDEN", TargetKind: "resource", Severity: "error", Enforcement: Enforced, WhatItChecks: "Resource lifecycle is not owned by shell scripts.", Remediation: "Remove shell-owned resource lifecycle files.", Claim: "resource.go-native-lifecycle"},
	{Code: "RESOURCE_HEALTH_KIND_MISSING", TargetKind: "resource", Severity: "error", Enforcement: Enforced, WhatItChecks: "Resources declare valid readiness or liveness health checks.", Remediation: "Declare at least one readiness or liveness health check.", Claim: "resource.health-checks"},
	{Code: "RESOURCE_MANIFEST_INVALID", TargetKind: "resource", Severity: "error", Enforcement: Enforced, WhatItChecks: "Resource manifests are valid and complete.", Remediation: "Repair resource.json so it is valid and complete.", Claim: "resource.manifest"},
	{Code: "TOOL_HANDLER_MISSING", TargetKind: "tool", Severity: "error", Enforcement: Enforced, WhatItChecks: "Declared tool handlers exist.", Remediation: "Add the Go handler declared by tool.json.", Claim: "tool.handler"},
	{Code: "TOOL_MANIFEST_INVALID", TargetKind: "tool", Severity: "error", Enforcement: Enforced, WhatItChecks: "Tool manifests are valid and complete.", Remediation: "Repair tool.json so it is valid and complete.", Claim: "tool.manifest"},
	{Code: "TOOL_NAME_MISMATCH", TargetKind: "tool", Severity: "error", Enforcement: Enforced, WhatItChecks: "Tool identity matches its target identifier.", Remediation: "Set tool.json.name to the canonical tool id.", Claim: "tool.identity"},
	{Code: "SAFEGUARD_HANDLER_MISSING", TargetKind: "safeguard", Severity: "error", Enforcement: Enforced, WhatItChecks: "Declared safeguard handlers exist.", Remediation: "Add the Go handler declared by safeguard.json.", Claim: "safeguard.handler"},
	{Code: "SAFEGUARD_MANIFEST_INVALID", TargetKind: "safeguard", Severity: "error", Enforcement: Enforced, WhatItChecks: "Safeguard manifests are valid and complete.", Remediation: "Repair safeguard.json so it is valid and complete.", Claim: "safeguard.manifest"},
	{Code: "SAFEGUARD_NAME_MISMATCH", TargetKind: "safeguard", Severity: "error", Enforcement: Enforced, WhatItChecks: "Safeguard identity matches its target identifier.", Remediation: "Set safeguard.json.name to the canonical safeguard id.", Claim: "safeguard.identity"},
	{Code: "TEAM_LAYOUT_MISSING", TargetKind: "team", Severity: "error", Enforcement: Enforced, WhatItChecks: "Team records have a README and declared sections.", Remediation: "Provide README.md and at least one declared manifest section.", Claim: "team.layout"},
	{Code: "TEAM_MANIFEST_INVALID", TargetKind: "team", Severity: "error", Enforcement: Enforced, WhatItChecks: "Team manifests are valid and complete.", Remediation: "Add a valid team plan-of-record manifest.", Claim: "team.manifest"},
	{Code: "TEAM_OWNER_MISMATCH", TargetKind: "team", Severity: "error", Enforcement: Enforced, WhatItChecks: "Team ownership matches the target identifier.", Remediation: "Align contract.team with the enumerated team target id.", Claim: "team.identity"},
	{Code: "TEAM_OWNER_MISSING", TargetKind: "team", Severity: "error", Enforcement: Enforced, WhatItChecks: "Team manifests declare a stable owner identifier.", Remediation: "Declare contract.team as the stable team target id.", Claim: "team.owner"},
	{Code: "DOCS_LAYOUT_MISSING", TargetKind: "docs", Severity: "warning", Enforcement: Enforced, WhatItChecks: "The documentation target has a README hub.", Remediation: "Add the documentation hub README.md.", Claim: "docs.layout"},
	{Code: "DOCS_MANIFEST_INVALID", TargetKind: "docs", Severity: "error", Enforcement: Enforced, WhatItChecks: "Documentation manifests are valid and complete.", Remediation: "Repair manifest.json with version, title, and sections.", Claim: "docs.manifest"},
	{Code: "CONTROL_PLANE_LAYOUT_MISSING", TargetKind: "control-plane", Severity: "warning", Enforcement: Enforced, WhatItChecks: "Control-plane targets are backed by Go source.", Remediation: "Keep control-plane cmd/internal targets backed by Go source files.", Claim: "control-plane.go-native"},
	{Code: "PROJECT_CONFIG_SURFACE", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "The project configuration surface matches the repository contract.", Remediation: "Remove unapproved entries from .vrooli.", Claim: "project.config-surface"},
	{Code: "PROJECT_CONTRACT_INVALID", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "The repository contract is valid and readable.", Remediation: "Repair .vrooli/repo-contract.json.", Claim: "project.contract"},
	{Code: "PROJECT_ROOT_PNPM_LOCK", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "The repository root has no stray pnpm lockfile.", Remediation: "Remove the root pnpm-lock.yaml; scenario UIs own their lockfiles.", Claim: "project.no-root-pnpm-lock"},
	{Code: "PROJECT_ORPHAN_GO_WORK_SUM", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "go.work.sum is present only with its go.work owner.", Remediation: "Remove the orphaned go.work.sum or restore its intentional go.work owner.", Claim: "project.go-work-pair"},
	{Code: "PROJECT_ROOT_NPMRC", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "The repository root has no npm configuration that leaks across scenario boundaries.", Remediation: "Remove the root .npmrc or move configuration to its owning boundary.", Claim: "project.no-root-npmrc"},
	{Code: "PROJECT_PNPM_WORKSPACE_INVALID", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "The root pnpm workspace owns packages only and keeps isolated workspace settings.", Remediation: "Keep pnpm-workspace.yaml scoped to packages/* with the required isolation settings.", Claim: "project.pnpm-workspace"},
	{Code: "SCENARIO_UI_BOUNDARY_MISSING", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario UIs carry a pnpm workspace boundary file.", Remediation: "Add ui/pnpm-workspace.yaml to stop package-manager discovery at the scenario boundary.", Claim: "scenario.ui-boundary"},
	{Code: "SCENARIO_UI_LOCKFILE_MISSING", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario UIs carry a committed lockfile.", Remediation: "Generate and commit the UI lockfile through Scenario Dependency Analyzer.", Claim: "scenario.ui-lockfile"},
	{Code: "SCENARIO_WORKSPACE_DEPENDENCY", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario UI dependencies do not use unsupported workspace protocol declarations.", Remediation: "Use an explicit package version or a governed local dependency declaration.", Claim: "scenario.no-workspace-star"},
	{Code: "SCENARIO_SHARED_PACKAGE_BYPASS", TargetKind: "scenario", Severity: "error", Enforcement: Enforced, WhatItChecks: "Scenario UIs consume shared packages through governed compiled outputs instead of package source trees.", Remediation: "Remove packages/*/src aliases and consume the package through its declared file dependency and compiled exports.", Claim: "scenario.shared-package-boundary"},
	{Code: "PROJECT_CLAIM_UNRESOLVED", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "Every marked enforcement claim resolves to a catalog rule.", Remediation: "Add the referenced rule to the catalog or remove the enforcement claim.", Claim: "project.claim-resolution"},
	{Code: "PROJECT_BUNDLE_PROFILE", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "The repository bundle profile includes and excludes canonical roots.", Remediation: "Restore the mini_vrooli_bundle include, exclude, and parameter policy.", Claim: "project.bundle-profile"},
	{Code: "PROJECT_CANONICAL_LAYOUT", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "Canonical repository layout markers match the contract.", Remediation: "Restore the canonical repository markers and paths.", Claim: "project.canonical-layout"},
	{Code: "PROJECT_EXCLUDED_LEGACY", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "Retired repository paths and contract entries remain excluded.", Remediation: "Remove retired paths and legacy contract entries.", Claim: "project.excluded-legacy"},
	{Code: "PROJECT_LIVE_STRUCTURE", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "Required repository directories, files, and manifests exist.", Remediation: "Restore the required repository directories, files, and manifests.", Claim: "project.live-structure"},
	{Code: "PROJECT_PHASE1_SEMANTICS", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "Phase-one repository contract semantics are canonical.", Remediation: "Restore the phase-one repository contract semantics.", Claim: "project.phase1-semantics"},
	{Code: "PROJECT_PROFILE_ROOTS", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "Repository profile includes stay inside canonical roots.", Remediation: "Keep profile includes inside canonical repository roots.", Claim: "project.profile-roots"},
	{Code: "PROJECT_RESOURCE_ARTIFACTS", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "Generated resource schema artifacts are present and valid.", Remediation: "Regenerate resource schema artifacts and repair missing resource references.", Claim: "project.resource-artifacts"},
	{Code: "PROJECT_RUNTIME_HOME", TargetKind: "project", Severity: "error", Enforcement: Enforced, WhatItChecks: "The runtime-home contract is canonical.", Remediation: "Restore the runtime-home structural authority.", Claim: "project.runtime-home"},
}

// Catalog returns a stable copy sorted by rule code.
func Catalog() []CatalogEntry {
	out := append([]CatalogEntry(nil), catalog...)
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// Lookup returns a rule's authoritative metadata.
func Lookup(code string) (CatalogEntry, bool) {
	for _, entry := range catalog {
		if entry.Code == code {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}

// ValidateCatalog protects the generated surfaces from duplicate or malformed
// rule records.
func ValidateCatalog() error {
	seen := make(map[string]struct{}, len(catalog))
	for _, entry := range catalog {
		if entry.Code == "" || entry.TargetKind == "" || entry.Severity == "" || entry.Enforcement == "" || entry.WhatItChecks == "" || entry.Remediation == "" {
			return fmt.Errorf("catalog entry %q is incomplete", entry.Code)
		}
		if _, ok := seen[entry.Code]; ok {
			return fmt.Errorf("duplicate catalog rule %q", entry.Code)
		}
		seen[entry.Code] = struct{}{}
	}
	return nil
}
