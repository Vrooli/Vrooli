// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"react-component-library/internal/librarywalk"
	"sort"
	"strings"
	"time"
)

func jsxOpeningTags(source string) []string {
	var tags []string
	for index := 0; index < len(source); index++ {
		if source[index] != '<' || index+1 >= len(source) || !isJSXNameStart(source[index+1]) {
			continue
		}
		braceDepth := 0
		quote := byte(0)
		for end := index + 1; end < len(source); end++ {
			char := source[end]
			if quote != 0 {
				if char == '\\' {
					end++
				} else if char == quote {
					quote = 0
				}
				continue
			}
			if char == '"' || char == '\'' || char == '`' {
				quote = char
				continue
			}
			switch char {
			case '{':
				braceDepth++
			case '}':
				if braceDepth > 0 {
					braceDepth--
				}
			case '>':
				if braceDepth == 0 {
					tags = append(tags, source[index:end+1])
					index = end
					end = len(source)
				}
			}
		}
	}
	return tags
}

func isJSXNameStart(char byte) bool {
	return (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z')
}

// ValidateStoryGrammar is the catalog-level counterpart to the story parser.
// It reads every story contract so the node DSL is checked even when a caller
// bypasses the component indexer.
func interactiveElements(source string) []string {
	starts := interactiveElementStart.FindAllStringIndex(source, -1)
	result := make([]string, 0, len(starts))
	for _, start := range starts {
		quote := byte(0)
		braceDepth := 0
		for index := start[0]; index < len(source); index++ {
			char := source[index]
			if quote != 0 {
				if char == quote && (index == 0 || source[index-1] != '\\') {
					quote = 0
				}
				continue
			}
			if char == '\'' || char == '"' || char == '`' {
				quote = char
				continue
			}
			switch char {
			case '{':
				braceDepth++
			case '}':
				if braceDepth > 0 {
					braceDepth--
				}
			case '>':
				if braceDepth == 0 {
					result = append(result, source[start[0]:index+1])
					index = len(source)
				}
			}
		}
	}
	return result
}

// ValidateTypes runs the same catalog conformance command declared by the
// catalog registry. Types are intentionally not inferred from the presence of
// source files: a released asset only earns this gate after the real
// TypeScript/ESLint boundary has executed successfully.
func validateTypes(root string, assets []string) (Result, error) {
	uiDir := filepath.Join(root, "scenarios", "react-component-library", "ui")
	if _, err := os.Stat(filepath.Join(uiDir, "package.json")); err != nil {
		if os.IsNotExist(err) {
			return Result{Findings: []Finding{{
				Code:        "catalog.types_zero_inputs",
				AssetID:     "",
				File:        repoRel(root, filepath.Join(uiDir, "package.json")),
				Message:     "catalog UI package is missing; the declared types runner could not execute",
				Remediation: "This is a runner fault, not an asset defect: the gate could not execute at all. Confirm the scenario tree is intact at scenarios/react-component-library/ui and that dependencies are installed. Do not interpret the absence of findings from this run as a passing types gate.",
				DocsRef:     "docs/internal/TESTING.md",
			}}}, nil
		}
		return Result{}, err
	}

	result := Result{Inspected: countCatalogSources(Scope{Root: root, Assets: assets})}
	if result.Inspected == 0 {
		return nonEmpty(result, "types"), nil
	}
	if _, err := exec.LookPath("node"); err != nil {
		result.Findings = append(result.Findings, Finding{Code: "catalog.types_runner_unavailable", Message: "node is unavailable; the declared types runner could not execute", Remediation: "Install or expose node through the scenario dependency analyzer before running catalog conformance."})
		return result, nil
	}
	reportFile, reportErr := os.CreateTemp("", "rcl-catalog-report-*.json")
	if reportErr != nil {
		return Result{}, reportErr
	}
	reportPath := reportFile.Name()
	_ = reportFile.Close()
	defer func() { _ = os.Remove(reportPath) }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "node", filepath.Join(uiDir, "scripts", "catalog-conformance.mjs"), "type-check")
	command.Dir = uiDir
	command.Env = append(os.Environ(), "RCL_CATALOG_REPORT="+reportPath)
	if len(assets) > 0 {
		scope := catalogScopeNames(root, assets)
		sort.Strings(scope)
		command.Env = append(command.Env, "RCL_CATALOG_ASSETS="+strings.Join(scope, ","))
	}
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog.types_timeout",
			AssetID:     "",
			Message:     "catalog conformance timed out after 3m before the declared types gate completed",
			Remediation: "Run `node scripts/catalog-conformance.mjs type-check` in scenarios/react-component-library/ui directly to see where it stalls. This is a runner fault, not an asset defect — no type conclusion can be drawn from this run either way.",
			DocsRef:     "docs/internal/TESTING.md",
		})
		return result, nil
	}
	// Report what was actually inspected rather than what the corpus contains,
	// so a scoped pass cannot describe itself as a full one. A scope that
	// matched no asset directory makes the script fail rather than report zero,
	// because zero inspected files exiting clean is indistinguishable from a
	// passing corpus.
	if inspected, ok := inspectedFromReport(reportPath); ok {
		result.Inspected = inspected
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 4000 {
			message = message[len(message)-4000:]
		}
		// Attribute the toolchain's diagnostics to the assets whose files they
		// name. Without this every types finding carried an empty AssetID, and
		// the evidence mapper matches a finding to an asset by exact id — so no
		// asset ever matched, and a failing catalog:check was recorded as
		// `types: pass` for the entire corpus.
		attributed, unattributed := attributeCatalogDiagnostics(root, reportPath)
		result.Findings = append(result.Findings, attributed...)
		if len(attributed) == 0 || unattributed {
			// Something failed that no single asset owns: the chain died before
			// the compiler ran, or the diagnostics point outside library/. The
			// gate cannot then claim any asset is clean, so this goes to
			// RunnerError, which the evidence mapper fails every asset closed
			// on. Passing them is the one outcome the run does not support.
			corpus := Finding{
				Code:        "catalog.types_failed",
				AssetID:     "",
				Message:     "catalog conformance failed: " + message,
				Remediation: "Reproduce with `node scripts/catalog-conformance.mjs type-check` in scenarios/react-component-library/ui; the output above is that command's tail. Fix the reported type errors at their source files — this gate deliberately reports the real toolchain's output rather than re-deriving its own verdict, so the failure it shows is the failure to fix.",
				DocsRef:     "docs/internal/TESTING.md",
			}
			result.RunnerError = append(result.RunnerError, corpus)
			// The same entry is repeated in Findings, and only for the
			// calibration harness, which scans Findings alone and matches an
			// empty AssetID. Removing it would flip this gate to
			// non-discriminating and quarantine it, dropping every asset's
			// types evidence to unmeasured.
			//
			// That quarantine would arguably be truthful: the types fixture
			// cannot exercise this runner at all. Its overlay writes a 0-byte
			// ui/package.json and omits packages/react-component-library
			// entirely, so `pnpm run catalog:check` dies before the compiler
			// starts, and the fixture has always been satisfied by that startup
			// failure rather than by the type error it plants. Making the gate
			// genuinely calibratable means giving the overlay a workspace the
			// toolchain can run in, which is a change with its own cost — a
			// full catalog:check inside every calibration pass — and its own
			// decision to make. Recorded here rather than resolved silently in
			// either direction.
			result.Findings = append(result.Findings, corpus)
		}
	}
	return result, nil
}

// catalogScopeNames translates public catalog ids into the directory names
// consumed by the UI conformance script. The API gate scope is expressed in
// catalog ids (for example controls.button), while the script intentionally
// scopes its file walk by authored library directory (Button).
func catalogScopeNames(root string, assets []string) []string {
	if len(assets) == 0 {
		return nil
	}
	ids := make(map[string]bool, len(assets))
	for _, asset := range assets {
		ids[asset] = true
	}
	names := make([]string, 0, len(assets))
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		manifests, _ := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				continue
			}
			var metadata struct {
				CatalogID string `json:"catalogId"`
				LibraryID string `json:"libraryId"`
			}
			if json.Unmarshal(data, &metadata) != nil {
				continue
			}
			if ids[metadata.CatalogID] || ids[metadata.LibraryID] {
				names = append(names, filepath.Base(filepath.Dir(manifest)))
			}
		}
	}
	if len(names) == 0 {
		return append([]string(nil), assets...)
	}
	return names
}

// inspectedFromReport reads the file count the conformance script actually
// checked. It reports ok=false when the report is unusable, leaving the
// caller's corpus-wide count in place rather than substituting a zero.
func inspectedFromReport(reportPath string) (int, bool) {
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return 0, false
	}
	var report struct {
		Inspected *int `json:"inspected"`
	}
	if err := json.Unmarshal(raw, &report); err != nil || report.Inspected == nil {
		return 0, false
	}
	return *report.Inspected, true
}

// catalogDiagnostic is one tsc or ESLint message, normalized by the conformance
// script at the point where its absolute path is still unambiguous.
type catalogDiagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source"`
}

// attributeCatalogDiagnostics maps each error-severity diagnostic onto the
// library asset that owns its file. It returns one finding per affected asset
// and reports whether any error could not be attributed, which is what forces
// the fail-closed corpus path.
func attributeCatalogDiagnostics(root, reportPath string) ([]Finding, bool) {
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, true
	}
	var report struct {
		Diagnostics []catalogDiagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, true
	}
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	byAsset := map[string]*Finding{}
	order := []string{}
	unattributed := false
	for _, diagnostic := range report.Diagnostics {
		// Warnings do not fail the toolchain, so they must not fail an asset.
		if !strings.EqualFold(diagnostic.Severity, "error") {
			continue
		}
		name := libraryAssetForPath(libraryRoot, diagnostic.File)
		if name == "" {
			unattributed = true
			continue
		}
		if _, ok := byAsset[name]; !ok {
			byAsset[name] = &Finding{
				Code:        "catalog.types_failed",
				AssetID:     name,
				File:        repoRel(root, diagnostic.File),
				Line:        diagnostic.Line,
				Message:     fmt.Sprintf("catalog conformance failed for %s: %s", name, diagnostic.Message),
				Remediation: "Fix the reported type error at its source file, then re-run `node scripts/catalog-conformance.mjs type-check` in scenarios/react-component-library/ui.",
				DocsRef:     "docs/internal/TESTING.md",
			}
			order = append(order, name)
		}
	}
	findings := make([]Finding, 0, len(order))
	for _, name := range order {
		findings = append(findings, *byAsset[name])
	}
	return findings, unattributed
}

// libraryAssetForPath returns the asset directory name owning a file under
// library/<kind>/<name>/…, or "" when the file belongs to something else — the
// catalog app, a script, a config. The evidence mapper matches a finding to an
// asset by catalog id or by this implementation name, so returning the
// directory name is what lets the mapping succeed.
