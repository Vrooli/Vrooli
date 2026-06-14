package hygiene

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	pnpmWorkspaceFile = "pnpm-workspace.yaml"
	rootNpmrcFile     = ".npmrc"
	// pnpmWorkspaceCommentMarker is a stable sentinel that must survive in the
	// governance comment block. pnpm strips comments whenever it rewrites the
	// file, so hygiene restores the full block whenever this marker is missing.
	pnpmWorkspaceCommentMarker = "Do NOT add scenarios"
	// pnpmDefaultPackageManager is only used to seed a freshly-created file when
	// no packageManager pin is present to preserve.
	pnpmDefaultPackageManager = "pnpm@10.14.0"
)

// pnpmWorkspaceComment is the canonical governance header. It is intentionally
// kept inside the file (not in a doc) so any agent reading pnpm-workspace.yaml
// sees the isolation policy immediately. No backticks so it stays a raw literal.
const pnpmWorkspaceComment = `# Root workspace configuration - shared packages only (packages/*).
# IMPORTANT: Do NOT add scenarios/**/ui or any scenarios/* path here.
# Scenarios must stay fully isolated to prevent dependency conflicts; each
# scenario ui/ carries its own pnpm-workspace.yaml BOUNDARY file (react-vite
# template >= 1.1.0) that stops pnpm's upward workspace walk, so a plain
# "pnpm install" there never joins this workspace ("--ignore-workspace" stays
# equivalent and is what lifecycle commands pass). A pnpm-lock.yaml must never
# exist at the repo root - it means an install ran in root-workspace scope.
# Shared-package adoption is governed by "vrooli package ..." plus package
# manifests under packages/*/.vrooli/package.json.
# This file is the single source of truth for root workspace settings; do not
# reintroduce a root .npmrc. pnpm rewrites this file and strips these comments;
# run "vrooli hygiene --fix-safe" to restore them. Enforced by "vrooli hygiene".`

// pnpmManagedScalars are the scalar settings hygiene enforces to a fixed value.
// Order here is the canonical write order.
var pnpmManagedScalarOrder = []string{
	"autoInstallPeers",
	"link-workspace-packages",
	"shared-workspace-lockfile",
}

var pnpmManagedScalars = map[string]string{
	"autoInstallPeers":          "false",
	"link-workspace-packages":   "false",
	"shared-workspace-lockfile": "false",
}

// pnpmRequiredHoist are the public-hoist-pattern entries migrated out of the
// root .npmrc so pnpm-workspace.yaml is the single source of truth.
var pnpmRequiredHoist = []string{"*eslint*", "*prettier*"}

// pnpmManagedKeys is the full set of keys hygiene owns; everything else in the
// file is preserved verbatim during a heal (e.g. onlyBuiltDependencies that
// pnpm may legitimately add).
var pnpmManagedKeys = map[string]bool{
	"packages":                  true,
	"public-hoist-pattern":      true,
	"packageManager":            true,
	"autoInstallPeers":          true,
	"link-workspace-packages":   true,
	"shared-workspace-lockfile": true,
}

// pnpmWorkspaceDoc is a lenient, comment-free view of pnpm-workspace.yaml. We
// parse line-based (no YAML dependency) which both avoids a new dependency and
// lets us reason about the comment block, which a YAML parser would discard.
type pnpmWorkspaceDoc struct {
	scalars map[string]string
	lists   map[string][]string
	order   []string // top-level key order as encountered
}

func parsePnpmWorkspace(data []byte) pnpmWorkspaceDoc {
	doc := pnpmWorkspaceDoc{scalars: map[string]string{}, lists: map[string][]string{}}
	currentList := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Indented line: only meaningful as a list item under the current key.
		if line[0] == ' ' || line[0] == '\t' {
			item := strings.TrimSpace(line)
			if currentList != "" && strings.HasPrefix(item, "-") {
				val := strings.TrimSpace(strings.TrimPrefix(item, "-"))
				val = strings.Trim(val, `"'`)
				doc.lists[currentList] = append(doc.lists[currentList], val)
			}
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			currentList = ""
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			currentList = ""
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if value == "" {
			// List parent (e.g. packages:, public-hoist-pattern:).
			currentList = key
			if _, ok := doc.lists[key]; !ok {
				doc.lists[key] = []string{}
			}
			doc.order = appendOnce(doc.order, key)
			continue
		}
		doc.scalars[key] = strings.Trim(value, `"'`)
		doc.order = appendOnce(doc.order, key)
		currentList = ""
	}
	return doc
}

// checkPnpmConfig validates (and optionally heals) the root pnpm-workspace.yaml
// plus the redundant-root-.npmrc invariant. When fixSafe is set, the file is
// healed first so findings reflect the post-heal state.
func (s Service) checkPnpmConfig(report *Report, fixSafe bool) {
	root := report.Root
	path := filepath.Join(root, pnpmWorkspaceFile)

	data, err := os.ReadFile(path)
	missing := false
	if err != nil {
		if !os.IsNotExist(err) {
			report.addCheck("pnpm_config", false, SeverityError, err.Error())
			report.addFinding(Finding{
				Severity:   SeverityError,
				Code:       "pnpm_workspace_read",
				Path:       pnpmWorkspaceFile,
				Message:    err.Error(),
				Fixability: FixabilityManual,
			})
			return
		}
		missing = true
		data = nil
	}

	if fixSafe {
		healed, changed := healPnpmWorkspace(data, missing)
		if changed {
			if werr := os.WriteFile(path, healed, 0o644); werr != nil {
				report.addFinding(Finding{
					Severity:   SeverityError,
					Code:       "pnpm_workspace_heal",
					Path:       pnpmWorkspaceFile,
					Message:    fmt.Sprintf("failed to write healed pnpm-workspace.yaml: %v", werr),
					Fixability: FixabilityManual,
				})
			} else {
				data = healed
				missing = false
				report.ConfigFixes = append(report.ConfigFixes, "restored canonical pnpm-workspace.yaml (comment block + workspace settings)")
				report.addCheck("pnpm_config_healed", true, SeverityInfo, "restored canonical pnpm-workspace.yaml")
			}
		}
	}

	var findings []Finding

	if missing {
		findings = append(findings, Finding{
			Severity:    SeverityError,
			Code:        "pnpm_workspace_missing",
			Path:        pnpmWorkspaceFile,
			Message:     "root pnpm-workspace.yaml is missing",
			Why:         "The root workspace file defines the packages/* workspace and the isolation settings scenarios rely on.",
			Fixability:  FixabilityAutomatic,
			NextActions: []Action{pnpmHealAction()},
		})
		s.finalizePnpmFindings(report, findings)
		return
	}

	doc := parsePnpmWorkspace(data)

	if !bytes.Contains(data, []byte(pnpmWorkspaceCommentMarker)) {
		findings = append(findings, Finding{
			Severity:    SeverityWarning,
			Code:        "pnpm_workspace_comment",
			Path:        pnpmWorkspaceFile,
			Message:     "governance comment block is missing (pnpm likely rewrote the file)",
			Why:         "The isolation policy lives in this comment so any agent editing the file sees it immediately; pnpm strips comments when it rewrites the file.",
			Fixability:  FixabilityAutomatic,
			NextActions: []Action{pnpmHealAction()},
		})
	}

	if !containsString(doc.lists["packages"], "packages/*") {
		findings = append(findings, Finding{
			Severity:    SeverityError,
			Code:        "pnpm_workspace_packages",
			Path:        pnpmWorkspaceFile,
			Message:     "packages: must include packages/*",
			Why:         "Shared packages live under packages/* and must be part of the root workspace.",
			Fixability:  FixabilityAutomatic,
			NextActions: []Action{pnpmHealAction()},
		})
	}
	var leaked []string
	for _, p := range doc.lists["packages"] {
		if strings.HasPrefix(p, "scenarios/") || strings.Contains(p, "/scenarios/") {
			leaked = append(leaked, p)
		}
	}
	if len(leaked) > 0 {
		findings = append(findings, Finding{
			Severity:    SeverityError,
			Code:        "pnpm_workspace_scenario_leak",
			Path:        pnpmWorkspaceFile,
			Locations:   leaked,
			Message:     fmt.Sprintf("packages: must not include scenario paths (%s); scenarios stay isolated and install with --ignore-workspace", strings.Join(leaked, ", ")),
			Why:         "Adding a scenario to the root workspace collapses scenario isolation and causes dependency conflicts.",
			Fixability:  FixabilityAutomatic,
			NextActions: []Action{pnpmHealAction()},
		})
	}

	for _, key := range pnpmManagedScalarOrder {
		want := pnpmManagedScalars[key]
		got, ok := doc.scalars[key]
		if ok && got == want {
			continue
		}
		severity := SeverityError
		if key == "autoInstallPeers" {
			severity = SeverityWarning
		}
		findings = append(findings, Finding{
			Severity:    severity,
			Code:        "pnpm_workspace_" + normalizePnpmCode(key),
			Path:        pnpmWorkspaceFile,
			Message:     fmt.Sprintf("%s must be %s (found %q)", key, want, got),
			Why:         "These settings keep the root workspace from linking or sharing lockfiles in ways that break scenario isolation.",
			Fixability:  FixabilityAutomatic,
			NextActions: []Action{pnpmHealAction()},
		})
	}

	if _, ok := doc.scalars["packageManager"]; !ok {
		findings = append(findings, Finding{
			Severity:   SeverityWarning,
			Code:       "pnpm_workspace_package_manager",
			Path:       pnpmWorkspaceFile,
			Message:    "packageManager should be pinned (e.g. pnpm@10.14.0) so corepack uses a deterministic pnpm version",
			Fixability: FixabilityGuided,
		})
	}

	if !containsAll(doc.lists["public-hoist-pattern"], pnpmRequiredHoist) {
		findings = append(findings, Finding{
			Severity:    SeverityWarning,
			Code:        "pnpm_workspace_public_hoist",
			Path:        pnpmWorkspaceFile,
			Message:     "public-hoist-pattern should include *eslint* and *prettier* (migrated from the removed root .npmrc)",
			Why:         "ESLint/Prettier plugins must be hoisted so resolvers can find them; this config moved out of .npmrc into the single source of truth.",
			Fixability:  FixabilityAutomatic,
			NextActions: []Action{pnpmHealAction()},
		})
	}

	// A root pnpm-lock.yaml must never exist: per-package locks live in
	// packages/*, scenario locks in scenarios/*/ui. A root lock is the
	// signature of an install that ran in root-workspace scope (e.g. a
	// scenario ui install without its boundary file) and was deliberately
	// removed from the repo.
	rootLock := filepath.Join(root, "pnpm-lock.yaml")
	if _, lerr := os.Stat(rootLock); lerr == nil {
		if fixSafe {
			if rerr := os.Remove(rootLock); rerr != nil {
				findings = append(findings, Finding{
					Severity:   SeverityError,
					Code:       "pnpm_root_lockfile_stray",
					Path:       "pnpm-lock.yaml",
					Message:    fmt.Sprintf("failed to remove stray root pnpm-lock.yaml: %v", rerr),
					Fixability: FixabilityManual,
				})
			} else {
				report.ConfigFixes = append(report.ConfigFixes, "removed stray root pnpm-lock.yaml")
				report.addCheck("pnpm_root_lockfile_healed", true, SeverityInfo, "removed stray root pnpm-lock.yaml")
			}
		} else {
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       "pnpm_root_lockfile_stray",
				Path:       "pnpm-lock.yaml",
				Message:    "stray pnpm-lock.yaml at the repo root; an install ran in root-workspace scope",
				Why:        "Per-package locks live in packages/* and scenario locks in scenarios/*/ui; a root lock means some install joined the root workspace (likely a scenario ui missing its pnpm-workspace.yaml boundary file).",
				Fixability: FixabilityAutomatic,
				NextActions: []Action{{
					Code:       "remove_stray_root_lockfile",
					Message:    "Delete the stray root pnpm-lock.yaml and add the boundary file to whichever ui/ produced it.",
					Command:    "vrooli hygiene --pnpm-only --fix-safe",
					Fixability: FixabilityAutomatic,
				}},
			})
		}
	}

	if npmrcData, nerr := os.ReadFile(filepath.Join(root, rootNpmrcFile)); nerr == nil {
		if npmrcHasWorkspaceKeys(npmrcData) {
			findings = append(findings, Finding{
				Severity:   SeverityWarning,
				Code:       "pnpm_npmrc_redundant",
				Path:       rootNpmrcFile,
				Message:    "root .npmrc duplicates workspace settings now owned by pnpm-workspace.yaml; remove it so there is a single source of truth",
				Why:        "Duplicated config across .npmrc and pnpm-workspace.yaml is what let pnpm silently rewrite the workspace file in the first place.",
				Fixability: FixabilityGuided,
				NextActions: []Action{{
					Code:       "remove_redundant_npmrc",
					Message:    "Move any non-workspace keys into pnpm-workspace.yaml, then delete the root .npmrc.",
					Command:    "rm .npmrc",
					Fixability: FixabilityGuided,
				}},
			})
		}
	}

	s.finalizePnpmFindings(report, findings)
}

func (s Service) finalizePnpmFindings(report *Report, findings []Finding) {
	passed := len(findings) == 0
	severity := SeverityInfo
	message := "root pnpm workspace config valid"
	if !passed {
		severity = highestSeverity(findings)
		message = fmt.Sprintf("%d pnpm workspace issue(s)", len(findings))
	}
	report.addCheck("pnpm_config", passed, severity, message)
	for _, finding := range findings {
		report.addFinding(finding)
	}
}

// checkScenarioPnpm enforces the per-scenario pnpm contract: scenario UIs must
// use file: references (never workspace:*) and should commit their lockfile.
func (s Service) checkScenarioPnpm(report *Report) {
	root := report.Root
	matches, err := filepath.Glob(filepath.Join(root, "scenarios", "*", "ui", "package.json"))
	if err != nil {
		report.addCheck("scenario_pnpm", false, SeverityError, err.Error())
		report.addFinding(Finding{
			Severity:   SeverityError,
			Code:       "scenario_pnpm_scan",
			Message:    err.Error(),
			Fixability: FixabilityManual,
		})
		return
	}
	sort.Strings(matches)

	var starViolations []string
	var missingLock []string
	var missingBoundary []string
	for _, pkgPath := range matches {
		data, rerr := os.ReadFile(pkgPath)
		if rerr != nil {
			continue
		}
		if bytes.Contains(data, []byte(`"workspace:`)) {
			starViolations = append(starViolations, relPathFromRoot(root, pkgPath))
		}
		lockPath := filepath.Join(filepath.Dir(pkgPath), "pnpm-lock.yaml")
		if _, lerr := os.Stat(lockPath); os.IsNotExist(lerr) {
			missingLock = append(missingLock, relPathFromRoot(root, filepath.Dir(pkgPath)))
		}
		boundaryPath := filepath.Join(filepath.Dir(pkgPath), pnpmWorkspaceFile)
		if _, berr := os.Stat(boundaryPath); os.IsNotExist(berr) {
			missingBoundary = append(missingBoundary, relPathFromRoot(root, filepath.Dir(pkgPath)))
		}
	}

	passed := len(starViolations) == 0 && len(missingLock) == 0 && len(missingBoundary) == 0
	severity := SeverityInfo
	message := fmt.Sprintf("%d scenario UIs checked, all isolated", len(matches))
	if len(starViolations) > 0 {
		severity = SeverityError
		message = fmt.Sprintf("%d scenario UIs use workspace:* dependencies", len(starViolations))
	} else if len(missingLock) > 0 {
		severity = SeverityWarning
		message = fmt.Sprintf("%d scenario UIs missing a committed pnpm-lock.yaml", len(missingLock))
	} else if len(missingBoundary) > 0 {
		severity = SeverityWarning
		message = fmt.Sprintf("%d scenario UIs missing the pnpm-workspace.yaml boundary file", len(missingBoundary))
	}
	report.addCheck("scenario_pnpm", passed, severity, message)

	if len(starViolations) > 0 {
		report.addFinding(Finding{
			Severity:   SeverityError,
			Code:       "scenario_workspace_star",
			Locations:  starViolations,
			Message:    "scenario UI package.json uses workspace:* dependencies; scenarios are isolated and must reference shared packages with file:",
			Why:        "workspace:* only resolves inside a pnpm workspace, but scenarios install with --ignore-workspace, so it breaks the install.",
			Fixability: FixabilityManual,
			NextActions: []Action{{
				Code:       "convert_workspace_star_to_file",
				Message:    "Replace workspace:* with a file: reference (e.g. file:../../../packages/<name>).",
				Fixability: FixabilityManual,
			}},
		})
	}
	if len(missingLock) > 0 {
		report.addFinding(Finding{
			Severity:   SeverityWarning,
			Code:       "scenario_missing_lockfile",
			Locations:  missingLock,
			Message:    "scenario UI has no committed pnpm-lock.yaml; installs are non-reproducible",
			Why:        "A committed lockfile keeps scenario UI installs deterministic across machines and rebuilds.",
			Fixability: FixabilityGuided,
			NextActions: []Action{{
				Code:       "commit_scenario_lockfile",
				Message:    "Run corepack pnpm install --ignore-workspace in the scenario ui/ and commit the generated pnpm-lock.yaml.",
				Fixability: FixabilityGuided,
			}},
		})
	}
	if len(missingBoundary) > 0 {
		report.addFinding(Finding{
			Severity:   SeverityWarning,
			Code:       "scenario_missing_workspace_boundary",
			Locations:  missingBoundary,
			Message:    "scenario UI has no pnpm-workspace.yaml boundary file; a plain pnpm install there joins the root workspace",
			Why:        "pnpm resolves its workspace by walking up to the first pnpm-workspace.yaml; without a local boundary, installs run in root-workspace scope, ignore the scenario lockfile/overrides, and regenerate a stray root lock (react-vite template >= 1.1.0 ships the boundary).",
			Fixability: FixabilityGuided,
			NextActions: []Action{{
				Code:       "copy_workspace_boundary",
				Message:    "Copy templates/scenarios/react-vite/ui/pnpm-workspace.yaml into the scenario ui/ directory.",
				Command:    "cp templates/scenarios/react-vite/ui/pnpm-workspace.yaml <scenario>/ui/pnpm-workspace.yaml",
				Fixability: FixabilityGuided,
			}},
		})
	}
}

// healPnpmWorkspace returns the canonical file bytes and whether they differ
// from the input. The canonical form is deterministic, so healing is
// idempotent: healing already-canonical bytes reports no change.
func healPnpmWorkspace(data []byte, missing bool) ([]byte, bool) {
	var doc pnpmWorkspaceDoc
	if missing || len(bytes.TrimSpace(data)) == 0 {
		doc = pnpmWorkspaceDoc{scalars: map[string]string{}, lists: map[string][]string{}}
	} else {
		doc = parsePnpmWorkspace(data)
	}
	canonical := renderCanonicalPnpmWorkspace(doc)
	return canonical, !bytes.Equal(data, canonical)
}

func renderCanonicalPnpmWorkspace(doc pnpmWorkspaceDoc) []byte {
	var b strings.Builder
	b.WriteString(pnpmWorkspaceComment)
	b.WriteString("\n\n")

	b.WriteString("packages:\n")
	for _, pkg := range canonicalPackages(doc.lists["packages"]) {
		b.WriteString("  - " + pkg + "\n")
	}
	b.WriteString("\n")

	for _, key := range pnpmManagedScalarOrder {
		b.WriteString(key + ": " + pnpmManagedScalars[key] + "\n")
	}

	pm := doc.scalars["packageManager"]
	if pm == "" {
		pm = pnpmDefaultPackageManager
	}
	b.WriteString("packageManager: " + pm + "\n")

	b.WriteString("public-hoist-pattern:\n")
	for _, pattern := range canonicalHoist(doc.lists["public-hoist-pattern"]) {
		b.WriteString("  - \"" + pattern + "\"\n")
	}

	// Preserve any keys hygiene does not own (e.g. onlyBuiltDependencies).
	for _, key := range doc.order {
		if pnpmManagedKeys[key] {
			continue
		}
		if val, ok := doc.scalars[key]; ok {
			b.WriteString(key + ": " + val + "\n")
			continue
		}
		if items, ok := doc.lists[key]; ok {
			b.WriteString(key + ":\n")
			for _, item := range items {
				b.WriteString("  - " + item + "\n")
			}
		}
	}

	return []byte(b.String())
}

// canonicalPackages guarantees packages/* is present and drops any scenario
// paths that must never join the root workspace, preserving other entries.
func canonicalPackages(existing []string) []string {
	out := []string{"packages/*"}
	seen := map[string]bool{"packages/*": true}
	for _, pkg := range existing {
		if pkg == "" || seen[pkg] {
			continue
		}
		if strings.HasPrefix(pkg, "scenarios/") || strings.Contains(pkg, "/scenarios/") {
			continue
		}
		seen[pkg] = true
		out = append(out, pkg)
	}
	return out
}

func canonicalHoist(existing []string) []string {
	out := make([]string, 0, len(existing)+len(pnpmRequiredHoist))
	seen := map[string]bool{}
	for _, pattern := range pnpmRequiredHoist {
		if !seen[pattern] {
			seen[pattern] = true
			out = append(out, pattern)
		}
	}
	for _, pattern := range existing {
		if pattern == "" || seen[pattern] {
			continue
		}
		seen[pattern] = true
		out = append(out, pattern)
	}
	return out
}

func npmrcHasWorkspaceKeys(data []byte) bool {
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key := strings.TrimSpace(strings.SplitN(line, "=", 2)[0])
		key = strings.TrimSuffix(key, "[]")
		switch key {
		case "link-workspace-packages", "shared-workspace-lockfile", "public-hoist-pattern", "auto-install-peers":
			return true
		}
	}
	return false
}

func pnpmHealAction() Action {
	return Action{
		Code:       "heal_pnpm_workspace",
		Message:    "Restore the canonical pnpm-workspace.yaml (comment block + workspace settings).",
		Command:    "vrooli hygiene --pnpm-only --fix-safe",
		Fixability: FixabilityAutomatic,
	}
}

func normalizePnpmCode(key string) string {
	return strings.ReplaceAll(key, "-", "_")
}

func highestSeverity(findings []Finding) Severity {
	severity := SeverityInfo
	for _, finding := range findings {
		switch finding.Severity {
		case SeverityError:
			return SeverityError
		case SeverityWarning:
			severity = SeverityWarning
		}
	}
	return severity
}

func appendOnce(items []string, value string) []string {
	if containsString(items, value) {
		return items
	}
	return append(items, value)
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func containsAll(items, required []string) bool {
	for _, req := range required {
		if !containsString(items, req) {
			return false
		}
	}
	return true
}

func relPathFromRoot(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
