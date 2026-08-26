// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"react-component-library/internal/components"
	"react-component-library/internal/graphreconcile"
)

// Finding is one gate defect. Message states what is wrong; Remediation states
// what to do about it. The split is deliberate: an agent acting on a finding
// needs the fix, and a message that ends in "use a declared semantic token"
// when no such token exists is worse than no guidance at all.
//
// File/Line locate the defect. Both are optional because some gates inspect
// declarations rather than source (a missing story specimen has no line), but
// a gate that can resolve a location and does not is a gate that costs its
// reader a grep.
type Finding struct {
	Code        string
	Category    string
	AssetID     string
	Message     string
	File        string
	Line        int
	Remediation string
	DocsRef     string
}

// Result makes runner coverage observable. A gate that reports no findings
// after inspecting zero inputs is not a passing gate; it is a broken runner.
type Result struct {
	Findings []Finding
	// InformationalFindings are auditable observations that do not affect the
	// gate verdict. They are used for planned-but-not-applied lifecycle work,
	// where hiding the observation would make the run look cleaner than it is.
	InformationalFindings []Finding
	Inspected             int
	InspectedAssets       []string
	RunnerError           []Finding
	SurfaceCounts         map[string]int
	UnmeasuredAssets      []string
	// UnstampedAssets and UncapturedAssets partition UnmeasuredAssets by root
	// cause. An unstamped asset rendered without an identity marker, so its
	// evidence exists but cannot be attributed — a build-config fix. An
	// uncaptured asset carries a marker but no browser ever recorded it — a
	// capture-coverage fix. Collapsing the two hides which lever to pull.
	UnstampedAssets   []string
	UncapturedAssets  []string
	CompositionScores map[string]float64
	CompositionMedian float64
	BespokeEscapes    []CompositionEscape
	// Status is an explicit runner outcome. In particular, unmeasured means
	// that no runner produced an observation and must never be interpreted as
	// a clean result by an evidence consumer.
	Status string
}

// CompositionEscape records a rendered node exempted from the raw-node
// denominator. The reason is author-visible evidence, not a hidden waiver.
type CompositionEscape struct {
	AssetID string
	Reason  string
}

// NormalizeResult enforces the gate identity boundary at the runner seam.
// Source paths and workbench labels are useful diagnostics, but they are not
// catalog identity. Findings that cannot be resolved are runner errors and
// are deliberately excluded from per-asset scoring.
func NormalizeResult(root string, result Result) Result {
	ids := catalogAssetIDs(root)
	normalized := make([]Finding, 0, len(result.Findings))
	for _, finding := range result.Findings {
		if finding.AssetID == "" {
			result.RunnerError = append(result.RunnerError, finding)
			continue
		}
		// Corpus-level gates use stable pseudo-assets instead of attributing a
		// distribution finding to an arbitrary catalog entry. Preserve those
		// findings as findings; they are measured observations, not runner
		// failures caused by an unresolvable asset identity.
		if strings.HasPrefix(finding.AssetID, "workbench.") || strings.HasPrefix(finding.AssetID, "__corpus__") {
			normalized = append(normalized, finding)
			continue
		}
		if ids[finding.AssetID] {
			if finding.AssetID != "" {
				result.InspectedAssets = appendUnique(result.InspectedAssets, finding.AssetID)
			}
			normalized = append(normalized, finding)
			continue
		}
		if resolved := implementationName(filepath.Join(root, finding.File)); resolved != "" && ids[resolved] {
			finding.AssetID = resolved
			result.InspectedAssets = appendUnique(result.InspectedAssets, resolved)
			normalized = append(normalized, finding)
			continue
		}
		finding.AssetID = ""
		result.RunnerError = append(result.RunnerError, finding)
	}
	result.Findings = normalized
	return result
}

func catalogAssetIDs(root string) map[string]bool {
	ids := map[string]bool{}
	paths, _ := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		if json.Unmarshal(data, &doc) == nil && doc.Asset.ID != "" {
			ids[doc.Asset.ID] = true
		}
	}
	return ids
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// lineOf resolves the 1-indexed line containing the byte offset of the first
// occurrence of match in data. Gates that scan with regexes know the matched
// text but not where it sat; this recovers the location without threading an
// index through every scan loop.
func lineOf(data []byte, match string) int {
	idx := bytes.Index(data, []byte(match))
	if idx < 0 {
		return 0
	}
	return bytes.Count(data[:idx], []byte("\n")) + 1
}

// lineAt resolves the 1-indexed line for an absolute byte offset.
func lineAt(data []byte, offset int) int {
	if offset < 0 || offset > len(data) {
		return 0
	}
	return bytes.Count(data[:offset], []byte("\n")) + 1
}

// repoRel trims the repository root from an absolute path so findings carry a
// stable, clickable location rather than a machine-specific one.
func repoRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

// ValidateGraphReconciled is intentionally non-blocking in catalog/config.json.
// It makes pre-existing dependency drift measurable without pretending this
// reporting phase repaired any manifest.
func ValidateGraphReconciled(root string) (Result, error) {
	report, err := graphreconcile.Reconcile(context.Background(), root)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(report.Assets)}
	for _, row := range report.Assets {
		if row.Verdict != graphreconcile.ImportsUnavailable {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.graph_reconciled_unavailable", AssetID: "__corpus__.graph-reconciled",
			Message:     fmt.Sprintf("imports-unavailable: %s", row.Cause),
			Remediation: "Confirm typescript-code-graph is running and has indexed scenarios/react-component-library; until it does, dependency reconciliation is unmeasured rather than evidence of drift.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
		})
		return result, nil
	}
	for _, row := range report.Assets {
		if row.Verdict == graphreconcile.Reconciled {
			continue
		}
		// ImportsUnavailable is a runner fault, not asset drift: the third view
		// simply did not load, so no conclusion about this asset's edges is
		// available either way. Reporting it with the drift remediation would
		// send a reader auditing 410 healthy assets against a comparison that
		// never ran.
		remediation := "Bring the three dependency views into agreement: the requires edges in catalog/assets/, the dependencies[] pins in the asset's library/**/component.json, and the imports the source actually makes. Whichever two agree usually identifies the stale one. This gate is non-blocking because the drift predates it — it reports, and never edits library/ on your behalf."
		switch row.Verdict {
		case graphreconcile.ImportsUnavailable:
			remediation = "This is a runner fault, not drift in this asset. The reconciler could not obtain an import graph from typescript-code-graph, so the source-import view is missing and no reconciliation verdict is possible. Confirm typescript-code-graph is running and has indexed scenarios/react-component-library; until it does, every asset reports this and none of them are evidence of a dependency problem."
		case graphreconcile.NotImplemented:
			// The catalog is desired state, so this is the expected resting
			// verdict for anything not built yet. It is reported for census,
			// not as drift, and carries no reconciliation advice: there is only
			// one dependency view to read.
			remediation = "No library implementation exists for this catalog asset, so there is nothing to reconcile against its requires edges. This is a construction gap rather than dependency drift; use `react-component-library catalog next` to decide whether this asset is worth building."
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.graph_reconciled", AssetID: row.AssetID,
			Message:     fmt.Sprintf("%s: %s", row.Verdict, row.Cause),
			Remediation: remediation,
			DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
		})
	}
	return result, nil
}

var (
	cssVarRefGateRE         = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
	cssVarDeclGateRE        = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
	versionLivenessImportRE = regexp.MustCompile(`@vrooli/react-component-library/([^/\s'\";]+)/([^/\s'\";]+)`)
)

// ValidateVersionLiveness protects the published version boundary. Every
// library import must resolve to a surviving version entry, and released
// source must not retain a relative edge into another version directory.
// Package compilation catches the former late; this gate makes the contract
// visible to catalog evidence and calibration before a consumer build.
func ValidateVersionLiveness(root string) (Result, error) {
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	var sources []string
	err := filepath.WalkDir(libraryRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext == ".ts" || ext == ".tsx" {
			sources = append(sources, path)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	sort.Strings(sources)
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(data)
		assetID := implementationName(path)
		for _, match := range versionLivenessImportRE.FindAllStringSubmatchIndex(text, -1) {
			name := text[match[2]:match[3]]
			version := text[match[4]:match[5]]
			if !versionEntryExists(libraryRoot, name, version) {
				line := lineAt(data, match[0])
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.version_liveness", AssetID: assetID, File: repoRel(root, path), Line: line,
					Message:     fmt.Sprintf("imports retired or missing library version %s@%s", name, version),
					Remediation: fmt.Sprintf("Point this import at a surviving %s version and run `react-component-library versions plan-cleanup` before retiring any further versions. Published package subpaths must resolve to a live version entry.", name),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#version-lifecycle",
				})
			}
		}
		for _, match := range regexp.MustCompile(`(?:from\s*|import\s*)[\"']([^\"']+)[\"']`).FindAllStringSubmatchIndex(text, -1) {
			specifier := text[match[2]:match[3]]
			if strings.HasPrefix(specifier, ".") && strings.Contains(specifier, "/versions/") {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.version_liveness", AssetID: assetID, File: repoRel(root, path), Line: lineAt(data, match[0]),
					Message:     fmt.Sprintf("retains a relative import into a version directory: %s", specifier),
					Remediation: "Use the published @vrooli/react-component-library/<asset>/<version> entry for a dependency, or move a shared helper into the importing version's own closure before retiring versions.",
					DocsRef:     "docs/concepts/ARCHITECTURE.md#version-lifecycle",
				})
			}
		}
	}
	return nonEmpty(result, "version-liveness"), nil
}

func versionEntryExists(libraryRoot, name, version string) bool {
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		dir := filepath.Join(libraryRoot, kind, name, "versions", version)
		for _, entry := range []string{name, pascalCaseGate(name)} {
			for _, ext := range []string{".ts", ".tsx"} {
				if _, err := os.Stat(filepath.Join(dir, entry+ext)); err == nil {
					return true
				}
			}
		}
	}
	return false
}

func pascalCaseGate(value string) string {
	parts := strings.Split(value, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

// ValidateTokenVocabulary rejects the retired app-prefixed CSS vocabulary in
// active library source. The consumer-side token-map vocabulary is separate
// and is intentionally not inspected here.
func ValidateTokenVocabulary(root string) (Result, error) {
	sources, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		if strings.Contains(string(raw), "--app-") {
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog.token_vocabulary",
				AssetID:     implementationName(path),
				File:        repoRel(root, path),
				Line:        lineOf(raw, "--app-"),
				Message:     "references the retired --app-* CSS custom property vocabulary",
				Remediation: "Replace each --app-<name> reference with its --color-<name> / --space-<name> equivalent from the canonical ramp at ui/src/design-tokens.css. The --app-* family is defined only for the workspace application shell and is not published to library consumers, so a library asset referencing it renders unstyled in every adopting scenario.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
			})
		}
	}
	return nonEmpty(result, "token-vocabulary"), nil
}

// ValidateTokenRampComplete verifies that every external literal custom
// property used by active library source is published by the canonical RCL
// ramp. Self-defined --rcl-* properties and dynamic families are excluded.
func ValidateTokenRampComplete(root string) (Result, error) {
	sources, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	rampRaw, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "ui", "src", "design-tokens.css"))
	if err != nil {
		return Result{}, err
	}
	ramp := map[string]struct{}{}
	for _, match := range cssVarDeclGateRE.FindAllStringSubmatch(string(rampRaw), -1) {
		ramp[match[1]] = struct{}{}
	}
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(raw)
		declared := map[string]struct{}{}
		for _, match := range cssVarDeclGateRE.FindAllStringSubmatch(text, -1) {
			declared[match[1]] = struct{}{}
		}
		for _, match := range cssVarRefGateRE.FindAllStringSubmatch(text, -1) {
			property := match[1]
			if _, local := declared[property]; local || strings.HasPrefix(property, "--rcl-") || strings.HasSuffix(property, "-") {
				continue
			}
			if _, published := ramp[property]; !published {
				result.Findings = append(result.Findings, Finding{
					Code:        "catalog.token_ramp_complete",
					AssetID:     implementationName(path),
					File:        repoRel(root, path),
					Line:        lineOf(raw, match[0]),
					Message:     fmt.Sprintf("consumes %s, which the canonical ramp does not publish", property),
					Remediation: fmt.Sprintf("Either publish %s in ui/src/design-tokens.css so every adopting scenario inherits it, or switch this reference to a property the ramp already declares. An unpublished property resolves to nothing in a consumer that has not copied this file, so the asset silently loses the styling this reference was meant to apply.", property),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
				})
			}
		}
	}
	return nonEmpty(result, "token-ramp-complete"), nil
}

// ValidateReleasedVersionImmutable compares every indexed released version
// with its current on-disk entry and companion files. It is intentionally a
// corpus gate, independent of the indexer's write path, so direct filesystem
// edits remain observable.
func ValidateReleasedVersionImmutable(root string) (Result, error) {
	dbPath := filepath.Join(root, "scenarios", "react-component-library", "data", "react-component-library.db")
	db, err := openGateDB(context.Background(), dbPath)
	if err != nil {
		return Result{}, fmt.Errorf("open component index: %w", err)
	}
	defer db.Close()
	return ValidateReleasedVersionImmutableWithDB(root, db)
}

// queryContexter is the smallest database seam needed by the immutable gate.
// The production catalog coverage path supplies the already-open routed
// scenario database; the root-only runner above remains useful for isolated
// calibration fixtures.
type queryContexter interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// ValidateReleasedVersionImmutableWithDB compares released versions using an
// already-open scenario database. Runtime callers must use this form so the
// gate observes the same routed storage and schema as the rest of the API.
func ValidateReleasedVersionImmutableWithDB(root string, db queryContexter) (Result, error) {
	if db == nil {
		return Result{}, fmt.Errorf("component index database is not configured")
	}
	rows, err := db.QueryContext(context.Background(), `SELECT v.status, v.source_path, v.content_sha256 FROM component_versions v WHERE v.status = 'released'`)
	if err != nil {
		return Result{}, fmt.Errorf("read released version hashes: %w", err)
	}
	defer rows.Close()
	result := Result{}
	for rows.Next() {
		var status, sourcePath, recorded string
		if err := rows.Scan(&status, &sourcePath, &recorded); err != nil {
			return Result{}, err
		}
		if status != "released" {
			continue
		}
		result.Inspected++
		raw, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "library", sourcePath))
		if err != nil {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.released_version_immutable", AssetID: implementationName(filepath.Join(root, "scenarios", "react-component-library", "library", sourcePath)), File: filepath.Join("library", sourcePath),
				Message:     fmt.Sprintf("released source cannot be read: %v", err),
				Remediation: "Restore the file at this path, or if the version was withdrawn on purpose, remove its row from the versions table rather than deleting the source. A released version is a published contract: adopting scenarios pin it by hash, so a source that has vanished cannot be re-verified by anyone who already depends on it.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#versioning",
			})
			continue
		}
		sum := sha256.Sum256(raw)
		current := hex.EncodeToString(sum[:])
		// A source checkout may materialize one terminal LF even when the
		// release record was captured without one. Treat that representation
		// change as canonical whitespace, while still rejecting every other
		// byte-level mutation.
		if recorded != "" && recorded != current && !(bytes.HasSuffix(raw, []byte("\n")) && func() bool {
			withoutTerminalLF := sha256.Sum256(bytes.TrimSuffix(raw, []byte("\n")))
			return recorded == hex.EncodeToString(withoutTerminalLF[:])
		}()) {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.released_version_immutable", AssetID: implementationName(filepath.Join(root, "scenarios", "react-component-library", "library", sourcePath)), File: filepath.Join("library", sourcePath),
				Message:     fmt.Sprintf("released source changed after release: recorded %s, current %s", recorded[:12], current[:12]),
				Remediation: "Revert this file to its released content and publish the change as a new version instead. Released versions are immutable because consumers pin them by hash; editing one in place means two scenarios can hold the same version number and get different code, which makes every downstream bug report unreproducible.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#versioning",
			})
		}
	}
	if err := rows.Err(); err != nil {
		return Result{}, err
	}
	return nonEmpty(result, "released-version-immutable"), nil
}

type assetDoc struct {
	Asset struct {
		ID, Kind, Name, Surface string
		Target                  struct {
			Maturity string `json:"maturity"`
		} `json:"target"`
	} `json:"asset"`
	Budgets struct {
		MountMS float64 `json:"mountMs"`
	} `json:"budgets"`
	API *struct {
		Variants map[string][]string `json:"variants"`
		Modes    []string            `json:"modes"`
		Parts    []json.RawMessage   `json:"parts"`
	} `json:"api"`
	Fixture *struct {
		DataShapes []string `json:"dataShapes"`
		Satisfies  *struct {
			Capability    string   `json:"capability"`
			TypeArguments []string `json:"typeArguments"`
		} `json:"satisfies"`
	} `json:"fixture"`
	Provides map[string]json.RawMessage `json:"provides"`
	Consumes map[string]json.RawMessage `json:"consumes"`
}

func loadAssets(root string) ([]assetDoc, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []assetDoc
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc assetDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		out = append(out, doc)
	}
	return out, nil
}

// ValidateAPI checks declared API vocabulary against the implementation
// source selected by catalogId. Missing implementations are not failures of
// this runner; coverage keeps those assets at missing/scaffolded.
func ValidateAPI(root string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.API == nil {
			continue
		}
		manifest, source, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if !ok {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		text := string(data)
		apiRemediation := func(kind, value string) string {
			return fmt.Sprintf("Either implement %s %q in %s, or remove it from the asset's api block in catalog/assets/. The catalog entry is the contract adopting scenarios read before they call this component; a declared %s that the source does not handle is a promise the library cannot keep, and it fails at the consumer rather than here.", kind, value, repoRel(root, source), kind)
		}
		for group, values := range asset.API.Variants {
			for _, value := range values {
				if !strings.Contains(text, value) {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog.api_mismatch", AssetID: asset.Asset.ID, File: repoRel(root, manifest),
						Message:     fmt.Sprintf("declared %s variant %q is absent from the implementation", group, value),
						Remediation: apiRemediation("variant", value),
						DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
					})
				}
			}
		}
		for _, value := range asset.API.Modes {
			if !strings.Contains(text, value) {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.api_mismatch", AssetID: asset.Asset.ID, File: repoRel(root, manifest),
					Message:     fmt.Sprintf("declared mode %q is absent from the implementation", value),
					Remediation: apiRemediation("mode", value),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
				})
			}
		}
		for _, rawPart := range asset.API.Parts {
			partID := ""
			if json.Unmarshal(rawPart, &partID) != nil {
				var part struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(rawPart, &part)
				partID = part.ID
			}
			if partID != "" && !strings.Contains(text, partID) {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.api_mismatch", AssetID: asset.Asset.ID, File: repoRel(root, manifest),
					Message:     fmt.Sprintf("declared part %q is absent from the implementation", partID),
					Remediation: apiRemediation("part", partID),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
				})
			}
		}
	}
	return nonEmpty(result, "api"), nil
}

var (
	i18nAttributeLiteral    = regexp.MustCompile(`(?m)\b(aria-label|placeholder|title|alt|label|description)\s*=\s*["']([^"'\r\n<>]{1,160})["']`)
	jsxTextLiteral          = regexp.MustCompile(`>\s*([[:alpha:]][^<>{}\n]{1,160})\s*</[A-Za-z]`)
	interactiveElementStart = regexp.MustCompile(`<((?:button|a|input|select|textarea))\b`)
	objectTestIDLiteral     = regexp.MustCompile(`["']data-testid["']\s*:\s*["']([^"'\r\n<>]+)["']`)
	legacyI18nBridge        = regexp.MustCompile(`__vrooliTranslate|library-locale-bridge|\btranslate\(\s*["']`)
	positionalStringKey     = regexp.MustCompile(`(?:useStrings|resolveStrings|defineStrings)\(\s*["'][^"']*\.[0-9]+["']|["'][^"']*\.[0-9]+["']\s*:`)
)

// ValidateI18n derives user-facing strings from component source. Literal
// labels are not a stable adoption contract: the host must supply their
// translation through the shared locale bridge.
func ValidateI18n(root string) (Result, error) {
	return validateActiveSources(root, "i18n", func(asset assetDoc, source string) defect {
		if legacyI18nBridge.MatchString(source) {
			return defect{
				Message:     "library source still uses the removed locale bridge or legacy translate call",
				Remediation: "Declare a named key in a co-located .strings.ts module and read it through useStrings.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			}
		}
		if positionalStringKey.MatchString(source) {
			return defect{
				Message:     "internationalization key is positional rather than semantic",
				Remediation: "Rename the key to describe the meaning of the copy and keep the English default in the strings declaration.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			}
		}
		for _, match := range i18nAttributeLiteral.FindAllStringSubmatch(source, -1) {
			return defect{
				Message:     fmt.Sprintf("user-facing %s literal %q is embedded in the library source", match[1], match[2]),
				Remediation: fmt.Sprintf("Replace the literal with the host locale bridge for key %s.%s. The English fallback belongs in the locale catalog, not in the adopted component source.", asset.Asset.ID, strings.ToLower(match[1])),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			}
		}
		if match := jsxTextLiteral.FindStringSubmatch(source); len(match) > 0 && strings.TrimSpace(match[1]) != "" {
			return defect{
				Message:     fmt.Sprintf("user-facing JSX text %q is embedded in the library source", strings.TrimSpace(match[1])),
				Remediation: fmt.Sprintf("Render a translated value from the host locale bridge using a key under %s.", asset.Asset.ID),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			}
		}
		return defect{}
	})
}

// ValidateSelectorCoverage requires every native interactive element to carry
// a stable test id rooted at the catalog asset identity. This keeps BAS flows
// portable after the asset is copied into an adopting scenario.
func ValidateSelectorCoverage(root string) (Result, error) {
	return validateActiveSources(root, "selector-coverage", func(asset assetDoc, source string) defect {
		rootSelector := false
		for _, tag := range jsxOpeningTags(source) {
			if regexp.MustCompile(`\bdata-testid\s*=`).MatchString(tag) && strings.Contains(tag, asset.Asset.ID) {
				rootSelector = true
				break
			}
		}
		if !rootSelector {
			for _, match := range objectTestIDLiteral.FindAllStringSubmatch(source, -1) {
				if len(match) > 1 && strings.Contains(match[1], asset.Asset.ID) {
					rootSelector = true
					break
				}
			}
		}
		if !rootSelector {
			return defect{
				Message:     fmt.Sprintf("exported component has no root data-testid derived from %s", asset.Asset.ID),
				Remediation: fmt.Sprintf("Add data-testid=%q to the outermost rendered element, or derive the value from the catalog id.", asset.Asset.ID),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#automation-selectors",
			}
		}
		for _, tag := range interactiveElements(source) {
			match := interactiveElementStart.FindStringSubmatch(tag)
			if len(match) == 0 {
				continue
			}
			testID := regexp.MustCompile(`data-testid\s*=`).FindStringIndex(tag)
			if testID == nil || !strings.Contains(tag, asset.Asset.ID) {
				return defect{
					Message:     fmt.Sprintf("interactive <%s> has no data-testid derived from %s", match[1], asset.Asset.ID),
					Remediation: fmt.Sprintf("Add data-testid=%q or a derived selector rooted at %s to the interactive element.", asset.Asset.ID, asset.Asset.ID),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#automation-selectors",
				}
			}
		}
		return defect{}
	})
}

// ValidateRestyleContract checks the public seam used by linked consumers.
// Native HTML attribute inheritance is accepted as an explicit className
// contract when the component forwards its remaining props to its root
// element. Components with bespoke props must name className and use it in
// rendered markup so a consumer never has to copy the implementation merely
// to change presentation.
func ValidateRestyleContract(root string) (Result, error) {
	return validateActiveSources(root, "restyle-contract", func(asset assetDoc, source string) defect {
		finding := analyzeRestyleSource(source)
		if finding.Message == "" {
			return ok()
		}
		finding.Message = fmt.Sprintf("%s: %s", asset.Asset.ID, finding.Message)
		return finding
	})
}

var (
	stylePropRE         = regexp.MustCompile(`(?m)(?:^|[;{]\s*)style\?\s*:\s*([^;},\n]+)`)
	classNamePropRE     = regexp.MustCompile(`(?m)\bclassName\??\s*:`)
	classNameUseRE      = regexp.MustCompile(`\bclassName\b`)
	forwardRefRE        = regexp.MustCompile(`\bforwardRef\b`)
	refAttributeRE      = regexp.MustCompile(`\bref\s*=\s*\{[^}]*\b(?:ref|forwardedRef)\b[^}]*\}`)
	imperativeRefRE     = regexp.MustCompile(`\buseImperativeHandle\s*\(`)
	assignRefRE         = regexp.MustCompile(`\bassignRef\s*\(`)
	classNameBoundaryRE = regexp.MustCompile(`\bwithClassName\s*\(`)
	exportedComponentRE = regexp.MustCompile(`(?m)^export\s+(?:function|const|class)\s+[A-Z]`)
)

func analyzeRestyleSource(source string) defect {
	if !exportedComponentRE.MatchString(source) {
		return ok()
	}
	hasClassName := classNamePropRE.MatchString(source) || classNameUseRE.MatchString(source) || strings.Contains(source, "HTMLAttributes<") || strings.Contains(source, "ComponentProps<") || classNameBoundaryRE.MatchString(source)
	if hasOverloadedStyleProp(source) && hasClassName {
		return defect{
			Message:     "overloads the standard style prop while exposing className",
			Remediation: "Remove the bespoke style prop; expose a named token selector and reserve style for consumer-owned computed values.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#restyle-contract",
		}
	}
	if hasClassName && hasInlineStyleOverride(source) && !strings.Contains(source, "/* computed-style-ok */") {
		return defect{
			Message:     "accepts className but also assigns an inline style object",
			Remediation: "Move token-derived declarations to a co-located stylesheet keyed by data attributes, then compose className with cn() on the root.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#restyle-contract",
		}
	}
	hasForwardedRef := forwardRefRE.MatchString(source) && refAttributeRE.MatchString(source)
	if classNameBoundaryRE.MatchString(source) || imperativeRefRE.MatchString(source) || assignRefRE.MatchString(source) {
		hasForwardedRef = true
	}
	if hasClassName && !hasForwardedRef {
		return defect{
			Message:     "does not forward a consumer ref to its root element",
			Remediation: "Wrap the exported component in forwardRef and pass ref to the outermost rendered element.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#restyle-contract",
		}
	}
	if !hasClassName {
		return defect{
			Message:     "does not expose a className pass-through on its public component surface",
			Remediation: "Add className?: string to the exported props, merge it with cn(), and forward it to the outermost rendered element.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#restyle-contract",
		}
	}
	return ok()
}

func hasOverloadedStyleProp(source string) bool {
	for _, match := range stylePropRE.FindAllStringSubmatch(source, -1) {
		if len(match) < 2 {
			continue
		}
		typeName := strings.TrimSpace(match[1])
		if typeName != "CSSProperties" && typeName != "React.CSSProperties" {
			return true
		}
	}
	return false
}

// hasInlineStyleOverride only considers one JSX opening element at a time.
// A component can legitimately use a computed inline value on a nested
// layout child while still exposing a className-controlled root. Matching the
// whole source would incorrectly reject that case (and even match
// data-* attributes such as data-text-style).
func hasInlineStyleOverride(source string) bool {
	for _, tag := range jsxOpeningTags(source) {
		if !jsxInlineStyleObjectRE.MatchString(tag) {
			continue
		}
		if jsxConsumerClassRE.MatchString(tag) || jsxSpreadRE.MatchString(tag) {
			return true
		}
	}
	return false
}

var (
	jsxSpreadRE            = regexp.MustCompile(`\.\.\.(?:props|rest)\b`)
	jsxConsumerClassRE     = regexp.MustCompile(`(?m)(?:^|\s)className\s*=\s*\{[^}]*\bclassName\b`)
	jsxInlineStyleObjectRE = regexp.MustCompile(`(?m)(?:^|\s)style\s*=\s*\{\s*\{`)
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
func ValidateStoryGrammar(root string) (Result, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(paths)}
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return Result{}, readErr
		}
		_, diagnostics := components.ParseStoryContract(raw)
		contract, _ := components.ParseStoryContract(raw)
		for _, diagnostic := range components.StoryContractErrors(diagnostics) {
			if diagnostic.Rule != "raw_node_tag_name" {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.raw_node_tag_name", AssetID: implementationName(path), File: repoRel(root, path),
				Message: diagnostic.Detail, Remediation: "Replace the tag-name $text value with a $node and meaningful children, or move the React composition into story.tsx.", DocsRef: "docs/guides/asset-preview-composition.md",
			})
		}
		if contract == nil {
			continue
		}
		assetID := implementationName(path)
		anatomyCount := 0
		axes := map[string]bool{}
		for _, story := range contract.Stories {
			if story.Role == "anatomy" {
				anatomyCount++
			}
			if story.Role == "axis" {
				axes[story.Axis] = true
			}
			if len(story.Expect) == 0 {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_expectation_missing", AssetID: assetID, File: repoRel(root, path),
					Message:     fmt.Sprintf("story %q declares no expectation", story.ID),
					Remediation: "Add at least one rendered expectation to the story, such as a role, text, attribute, or layout assertion.",
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			}
		}
		if anatomyCount != 1 {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.story_anatomy_missing", AssetID: assetID, File: repoRel(root, path),
				Message:     fmt.Sprintf("contract has %d anatomy frames; exactly one is required", anatomyCount),
				Remediation: "Mark exactly one default rendered story with role=anatomy and keep its specimen free of matrix or boundary-only variation.",
				DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
			})
		}
		for _, field := range contract.Args.Fields {
			if field.Kind == components.StoryFieldEnum && !axes[field.Path] {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_axis_missing", AssetID: assetID, File: repoRel(root, path),
					Message:     fmt.Sprintf("enum axis %q has no axis frame", field.Path),
					Remediation: fmt.Sprintf("Add one story with role=axis, axis=%q, and covers listing the rendered options.", field.Path),
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			}
		}
	}
	return nonEmpty(result, "story-grammar"), nil
}

// ValidateStoryDistinctness rejects exact duplicate frames and the old
// one-specimen-per-option shape. Axis stories are intentionally allowed to
// share a specimen because their declared covers matrix is the variation.
func ValidateStoryDistinctness(root string) (Result, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(paths)}
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return Result{}, readErr
		}
		contract, diagnostics := components.ParseStoryContract(raw)
		if contract == nil || len(components.StoryContractErrors(diagnostics)) > 0 || len(contract.Stories) < 2 {
			continue
		}
		seen := map[string]string{}
		specimenExports := map[string]int{}
		for _, story := range contract.Stories {
			if story.Composition != nil && story.Composition.Specimen != nil {
				specimenExports[story.Composition.Specimen.Export]++
			}
			fingerprintStory := story
			fingerprintStory.ID = ""
			fingerprintStory.Name = ""
			canonical, marshalErr := json.Marshal(fingerprintStory)
			if marshalErr != nil {
				return Result{}, marshalErr
			}
			hash := sha256.Sum256(canonical)
			key := hex.EncodeToString(hash[:])
			if previous, exists := seen[key]; exists {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_sibling_duplicate", AssetID: implementationName(path), File: repoRel(root, path),
					Message:     fmt.Sprintf("story %q duplicates sibling story %q", story.ID, previous),
					Remediation: "Give each sibling frame a distinct rendered question, or collapse the duplicate into the matrix story that declares its covers.",
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			} else {
				seen[key] = story.ID
			}
		}
		if len(contract.Stories) >= 3 && len(specimenExports) == 1 {
			for exportName, count := range specimenExports {
				if exportName == "" || count != len(contract.Stories) {
					continue
				}
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_sibling_duplicate", AssetID: implementationName(path), File: repoRel(root, path),
					Message:     fmt.Sprintf("all %d sibling frames reuse specimen %q", count, exportName),
					Remediation: "Give anatomy, axis, and boundary questions distinct specimen compositions; an axis matrix may share one specimen only when it is the sole frame for that axis.",
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			}
		}
		if len(seen) == 1 && contract.Stories[0].Role == "" {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.story_sibling_duplicate", AssetID: implementationName(path), File: repoRel(root, path),
				Message:     "multiple stories render one indistinguishable legacy specimen",
				Remediation: "Migrate the contract to role=anatomy plus explicit axis or boundary frames with distinct rendered coverage.",
				DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
			})
		}
	}
	return nonEmpty(result, "story-distinctness"), nil
}

// Evidence freshness is a filesystem gate so the catalog can distinguish a
// current story contract from an older component-test observation.
func ValidateEvidenceFreshness(root string) (Result, error) {
	return validateEvidenceFreshness(root)
}

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
func ValidateTypes(root string) (Result, error) {
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

	result := Result{Inspected: countCatalogSources(root)}
	if result.Inspected == 0 {
		return nonEmpty(result, "types"), nil
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		result.Findings = append(result.Findings, Finding{Code: "catalog.types_runner_unavailable", Message: "pnpm is unavailable; the declared types runner could not execute", Remediation: "Install or expose pnpm through the scenario dependency analyzer before running catalog conformance."})
		return result, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "pnpm", "run", "catalog:check")
	command.Dir = uiDir
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog.types_timeout",
			AssetID:     "",
			Message:     "catalog conformance timed out after 3m before the declared types gate completed",
			Remediation: "Run `pnpm run catalog:check` in scenarios/react-component-library/ui directly to see where it stalls. This is a runner fault, not an asset defect — no type conclusion can be drawn from this run either way.",
			DocsRef:     "docs/internal/TESTING.md",
		})
		return result, nil
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 4000 {
			message = message[len(message)-4000:]
		}
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog.types_failed",
			AssetID:     "",
			Message:     "catalog conformance failed: " + message,
			Remediation: "Reproduce with `pnpm run catalog:check` in scenarios/react-component-library/ui; the output above is that command's tail. Fix the reported type or lint errors at their source files — this gate deliberately reports the real toolchain's output rather than re-deriving its own verdict, so the failure it shows is the failure to fix.",
			DocsRef:     "docs/internal/TESTING.md",
		})
	}
	return result, nil
}

func countCatalogSources(root string) int {
	sources, _ := activeLibrarySources(root)
	return len(sources)
}

// activeLibrarySources returns the files represented by each manifest's
// latest and draft pointers. Historical versions remain available to callers
// that pin them explicitly, but corpus-wide quality gates should measure the
// active catalog surface consistently with indexing, coverage, and the type
// gate rather than double-counting retired implementations.
func activeLibrarySources(root string) ([]string, error) {
	var sources []string
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		manifests, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return nil, err
			}
			var doc struct {
				Latest string `json:"latest"`
				Draft  string `json:"draft"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return nil, err
			}
			versions := []string{doc.Latest}
			if doc.Draft != "" && doc.Draft != doc.Latest {
				versions = append(versions, doc.Draft)
			}
			for _, version := range versions {
				if version == "" {
					continue
				}
				for _, extension := range []string{"*.ts", "*.tsx"} {
					matches, err := filepath.Glob(filepath.Join(filepath.Dir(manifest), "versions", version, extension))
					if err != nil {
						return nil, err
					}
					sources = append(sources, matches...)
				}
			}
		}
	}
	if len(sources) > 0 {
		sort.Strings(sources)
		return sources, nil
	}

	// Keep the unit-level gate contract useful for isolated fixtures that do
	// not need a full component manifest. Real repositories always take the
	// manifest-backed path above.
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		for _, extension := range []string{"*.ts", "*.tsx"} {
			matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "versions", "*", extension))
			if err != nil {
				return nil, err
			}
			sources = append(sources, matches...)
		}
	}
	sort.Strings(sources)
	return sources, nil
}

func implementationSource(root, catalogID string) (string, string, bool, error) {
	paths := []string{}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return "", "", false, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	for _, manifest := range paths {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return "", "", false, err
		}
		var doc struct {
			CatalogID string `json:"catalogId"`
			Latest    string `json:"latest"`
		}
		if json.Unmarshal(data, &doc) != nil || doc.CatalogID != catalogID {
			continue
		}
		if doc.Latest == "" {
			return manifest, "", false, nil
		}
		rootDir := filepath.Dir(manifest)
		versionDir := filepath.Join(rootDir, "versions", doc.Latest)
		source := filepath.Join(versionDir, filepath.Base(rootDir)+".tsx")
		if _, err := os.Stat(source); err != nil {
			matches := versionSources(versionDir)
			if len(matches) == 0 {
				versionDir = filepath.Join(rootDir, doc.Latest)
				matches = versionSources(versionDir)
			}
			if len(matches) == 0 {
				return manifest, "", false, nil
			}
			source = matches[0]
		}
		return manifest, source, true, nil
	}
	return "", "", false, nil
}

func versionSources(versionDir string) []string {
	var matches []string
	for _, extension := range []string{"*.tsx", "*.ts"} {
		found, _ := filepath.Glob(filepath.Join(versionDir, extension))
		matches = append(matches, found...)
	}
	sort.Strings(matches)
	return matches
}

var (
	pxValue = regexp.MustCompile(`--space-[a-z0-9-]+\s*:\s*([0-9.]+)px`)
	// literalDimension splits into two families because they have different
	// fixes. Box spacing has a token ramp to point at; sizing does not — the
	// ramp publishes no icon-size property, so telling an author to "use a
	// semantic token" for w-4/h-4 sends them looking for something that does
	// not exist. Sizing belongs to the Icon primitive's size scale instead.
	literalSpacing = regexp.MustCompile(`\b(?:p|px|py|pt|pr|pb|pl|m|mx|my|mt|mr|mb|ml|gap)-[0-9]+(?:\.[0-9]+)?\b`)
	literalSizing  = regexp.MustCompile(`\b[wh]-[0-9]+(?:\.[0-9]+)?\b`)
	arbitraryPx    = regexp.MustCompile(`\[[0-9.]+px\]`)
)

// literalDimensionFindings classifies a raw dimension by the fix it needs.
// Each family names a different remediation because each has a different
// correct answer; a single shared message would be wrong for two of the three.
func literalDimensionFindings(root, path string, data []byte) []Finding {
	assetID := implementationName(path)
	file := repoRel(root, path)
	var out []Finding
	for _, loc := range literalSpacing.FindAllIndex(data, -1) {
		match := string(data[loc[0]:loc[1]])
		out = append(out, Finding{
			Code: "catalog.tokens_literal", AssetID: assetID, File: file, Line: lineAt(data, loc[0]),
			Message:     fmt.Sprintf("box spacing %q is a raw value, not a ramp step", match),
			Remediation: fmt.Sprintf("Replace %q with the matching gap-space-*/p-space-*/m-space-* utility. The ramp publishes space-3xs through space-2xl on a 4px grid; a raw step does not move when the ramp is retuned, so this element drifts out of rhythm with every tokenized neighbour the first time density changes.", match),
			DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
		})
	}
	for _, loc := range literalSizing.FindAllIndex(data, -1) {
		// Do not treat the sizing suffix in layout utilities such as
		// min-w-0/max-w-0 as a standalone raw width utility.
		if loc[0] > 0 && data[loc[0]-1] == '-' {
			continue
		}
		match := string(data[loc[0]:loc[1]])
		out = append(out, Finding{
			Code: "catalog.tokens_literal", AssetID: assetID, File: file, Line: lineAt(data, loc[0]),
			Message:     fmt.Sprintf("element sized with raw dimension %q", match),
			Remediation: "Size icons through the Icon primitive's size scale (size=\"sm\" | \"md\" | \"lg\") rather than raw width/height utilities. The canonical ramp deliberately publishes no icon-size custom property, so there is no token to substitute here — the scale lives in the primitive's API. For non-icon boxes that genuinely need an intrinsic size, prefer a layout constraint (flex/grid sizing) over a fixed one.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
		})
	}
	for _, loc := range arbitraryPx.FindAllIndex(data, -1) {
		match := string(data[loc[0]:loc[1]])
		out = append(out, Finding{
			Code: "catalog.tokens_literal", AssetID: assetID, File: file, Line: lineAt(data, loc[0]),
			Message:     fmt.Sprintf("arbitrary pixel value %q bypasses the ramp entirely", match),
			Remediation: fmt.Sprintf("Remove the arbitrary value %q. If the nearest ramp step is visually acceptable, use it; if no step fits, the ramp is missing a rung — add it in ui/src/design-tokens.css so every consumer inherits it, rather than encoding the exception at one callsite where nothing can find it later.", match),
			DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
		})
	}
	return out
}

// ValidateTokens checks the shared ramp contract in every design kit and
// rejects non-grid spacing declarations.
func ValidateTokens(root string) (Result, error) {
	paths, err := filepath.Glob(filepath.Join(root, "templates", "design", "*", "adapters", "react-vite-tailwind", "tokens.css"))
	if err != nil {
		return Result{}, err
	}
	shared := []string{"space-3xs", "space-2xs", "space-xs", "space-sm", "space-md", "space-lg", "space-xl", "space-2xl", "text-display", "text-title", "text-heading", "text-body", "elev-flat", "elev-raised", "layer-base", "layer-modal", "dur-instant"}
	result := Result{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(data)
		kit := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
		for _, token := range shared {
			if !strings.Contains(text, "--"+token) {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.tokens_missing", AssetID: "", File: repoRel(root, path),
					Message:     fmt.Sprintf("design kit %q does not declare shared token --%s", kit, token),
					Remediation: fmt.Sprintf("Declare --%s in this kit's tokens.css. Every kit must publish the full shared vocabulary, because a component authored against one kit is expected to render in all of them; a kit missing a step silently drops that declaration to its initial value wherever the component lands.", token),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
				})
			}
		}
		for _, loc := range pxValue.FindAllSubmatchIndex(data, -1) {
			raw := string(data[loc[2]:loc[3]])
			value, _ := strconv.ParseFloat(raw, 64)
			if int(value)%4 != 0 {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.tokens_grid", AssetID: "", File: repoRel(root, path), Line: lineAt(data, loc[0]),
					Message:     fmt.Sprintf("design kit %q declares a spacing token at %spx, off the 4px grid", kit, raw),
					Remediation: fmt.Sprintf("Round %spx to the nearest multiple of 4. The grid is what makes steps from different kits interchangeable; an off-grid step produces half-pixel seams where a tokenized element sits next to an off-grid one, which reads as the blurry-edge misalignment that is very hard to attribute back to a token definition.", raw),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
				})
			}
		}
	}
	sources, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		result.Findings = append(result.Findings, literalDimensionFindings(root, path, data)...)
	}
	for index := range result.Findings {
		result.Findings[index].Category = "conformance"
	}
	return nonEmpty(result, "tokens"), nil
}

// ValidateLifecycle performs conservative static checks over hook/service/
// adapter/generator sources. It deliberately prefers a finding over a green
// result when cleanup evidence is absent.
func ValidateLifecycle(root string) (Result, error) {
	result := Result{}
	paths, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	for _, path := range paths {
		// Stories are browser-only specimens, not released runtime. Including
		// them here makes the lifecycle gate report demo timers and AbortSignal
		// listeners as component defects.
		if isStorySource(path) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		text := string(data)
		if strings.Contains(text, "addEventListener") && !strings.Contains(text, "removeEventListener") {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.lifecycle_cleanup", AssetID: implementationName(path), File: repoRel(root, path), Line: lineOf(data, "addEventListener"),
				Message:     "adds an event listener with no matching removeEventListener anywhere in the file",
				Remediation: "Return a cleanup function from the effect that registered this listener, calling removeEventListener with the same target, type, and handler reference. Without it the handler outlives the component: every mount adds another subscription, so the work done per event grows with the number of times the user has visited the surface.",
				DocsRef:     "docs/internal/SEAMS.md",
			})
		}
		if strings.Contains(text, "new MutationObserver") && !strings.Contains(text, ".disconnect(") {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.lifecycle_cleanup", AssetID: implementationName(path), File: repoRel(root, path), Line: lineOf(data, "new MutationObserver"),
				Message:     "constructs a MutationObserver with no .disconnect() anywhere in the file",
				Remediation: "Call .disconnect() in the cleanup of the effect that constructed this observer. An undisconnected observer keeps its target subtree alive and keeps firing after unmount, which shows up as work attributed to whatever surface is mounted next rather than to this one.",
				DocsRef:     "docs/internal/SEAMS.md",
			})
		}
		if hasBrowserAccessOutsideEffects(text) {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.lifecycle_ssr", AssetID: implementationName(path), File: repoRel(root, path),
				Message:     "reads a browser global during render or module scope, where no SSR guard applies",
				Remediation: "Move the access into useEffect/useLayoutEffect, or guard it with a typeof window !== \"undefined\" check. Render and module scope both execute during server rendering, so a bare window/document reference throws there — and because it throws at import time it takes down the whole route, not just this component.",
				DocsRef:     "docs/internal/SEAMS.md",
			})
		}
	}
	return nonEmpty(result, "lifecycle"), nil
}

func isStorySource(path string) bool {
	base := filepath.Base(path)
	return base == "story.ts" || base == "story.tsx"
}

// hasBrowserAccessOutsideEffects keeps the static SSR check conservative while
// understanding the one React lifecycle boundary that is guaranteed not to
// execute during server rendering. Browser access in render, module scope, or
// an arbitrary exported callback still requires an explicit guard.
func hasBrowserAccessOutsideEffects(text string) bool {
	remaining := []byte(text)
	for _, start := range effectCallbackRanges(text) {
		for index := start[0]; index < start[1] && index < len(remaining); index++ {
			remaining[index] = ' '
		}
	}
	textWithoutEffects := string(remaining)
	return (strings.Contains(textWithoutEffects, "window.") && !strings.Contains(textWithoutEffects, "typeof window")) ||
		(strings.Contains(textWithoutEffects, "document.") && !strings.Contains(textWithoutEffects, "typeof document"))
}

func effectCallbackRanges(text string) [][2]int {
	var ranges [][2]int
	for offset := 0; offset < len(text); {
		match := strings.Index(text[offset:], "useEffect")
		if match < 0 {
			break
		}
		start := offset + match
		after := start + len("useEffect")
		if after < len(text) && isIdentifierPart(text[after]) {
			offset = after
			continue
		}
		for after < len(text) && (text[after] == ' ' || text[after] == '\n' || text[after] == '\r' || text[after] == '\t') {
			after++
		}
		if after >= len(text) || text[after] != '(' {
			offset = after
			continue
		}
		arrow := strings.Index(text[after:], "=>")
		if arrow < 0 {
			break
		}
		arrow += after + 2
		body := arrow
		for body < len(text) && (text[body] == ' ' || text[body] == '\n' || text[body] == '\r' || text[body] == '\t') {
			body++
		}
		if body >= len(text) || text[body] != '{' {
			offset = arrow
			continue
		}
		end, ok := matchingBrace(text, body)
		if !ok {
			break
		}
		ranges = append(ranges, [2]int{body, end + 1})
		offset = end + 1
	}
	return ranges
}

func matchingBrace(text string, open int) (int, bool) {
	depth := 0
	for index := open; index < len(text); index++ {
		switch text[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index, true
			}
		}
	}
	return 0, false
}

func isIdentifierPart(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '_'
}

func implementationName(path string) string {
	path = filepath.Clean(path)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		path = filepath.Dir(path)
	}
	for {
		data, err := os.ReadFile(filepath.Join(path, "component.json"))
		if err == nil {
			var manifest struct {
				CatalogID string `json:"catalogId"`
			}
			if json.Unmarshal(data, &manifest) == nil && manifest.CatalogID != "" {
				return manifest.CatalogID
			}
		}
		parent := filepath.Dir(path)
		if parent == path {
			return ""
		}
		path = parent
	}
}

func ValidateFixtures(root string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.Asset.Kind != "fixture" || asset.Fixture == nil {
			continue
		}
		result.Inspected++
		if len(asset.Fixture.DataShapes) == 0 {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.fixture_adversarial", AssetID: asset.Asset.ID,
				Message:     "fixture declares no data shapes at all",
				Remediation: "Add a dataShapes array to this fixture's catalog entry naming the shapes it supplies (at minimum one of \"failure\" or \"overflow\"). A fixture that only supplies happy-path data lets every component consuming it pass its gates without ever rendering an error or a string long enough to wrap, which is exactly where layout defects hide.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
		if !contains(asset.Fixture.DataShapes, "failure") && !contains(asset.Fixture.DataShapes, "overflow") {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.fixture_adversarial", AssetID: asset.Asset.ID,
				Message:     fmt.Sprintf("fixture declares %v but neither \"failure\" nor \"overflow\"", asset.Fixture.DataShapes),
				Remediation: "Add a \"failure\" shape (what this data looks like when the source errors) or an \"overflow\" shape (values long or numerous enough to exceed their container). These are the two shapes that reveal layout and error-state defects; without one of them the fixture cannot drive a component into the states its experience contract claims to handle.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
		if asset.Fixture.Satisfies != nil && asset.Fixture.Satisfies.Capability == "data-source" && len(asset.Fixture.Satisfies.TypeArguments) == 0 {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.fixture_data_source", AssetID: asset.Asset.ID,
				Message:     "fixture satisfies the data-source capability without declaring a type argument",
				Remediation: "Add typeArguments to this fixture's satisfies block naming the row type it produces. The data-source capability is generic; without the type argument a consuming asset cannot tell whether this fixture supplies the shape it needs, so the port match succeeds structurally and then fails at render.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
		if consumers, err := fixtureConsumers(root, asset.Asset.ID); err != nil {
			return Result{}, err
		} else if consumers > 0 {
			assertions, err := fixtureFailureAssertions(root, asset.Asset.ID)
			if err != nil {
				return Result{}, err
			}
			if assertions == 0 {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.fixture_adversarial_render", AssetID: asset.Asset.ID,
					Message:     "fixture failure shape has consumers but no rendered error-state assertion",
					Remediation: "Add a failure-shaped consumer story and assert role=alert or data-fixture-state=failure. The preview harness accepts fixtureShape=failure and renders the fixture's failure record, so this check exercises the consumer rather than trusting fixture metadata.",
					DocsRef:     "docs/internal/TESTING.md",
				})
			}
		}
	}
	return nonEmpty(result, "fixture_adversarial"), nil
}

type fixtureStoryContract struct {
	Composition struct {
		Fixture struct {
			Asset string `json:"asset"`
		} `json:"fixture"`
	} `json:"composition"`
	Stories []struct {
		Expect []struct {
			Role      string `json:"role"`
			Selector  string `json:"selector"`
			Attribute string `json:"attribute"`
			Value     string `json:"value"`
		} `json:"expect"`
	} `json:"stories"`
}

func fixtureStoryContracts(root string) ([]fixtureStoryContract, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return nil, err
	}
	contracts := make([]fixtureStoryContract, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var contract fixtureStoryContract
		if err := json.Unmarshal(data, &contract); err != nil {
			return nil, err
		}
		contracts = append(contracts, contract)
	}
	return contracts, nil
}

func fixtureConsumers(root, fixtureID string) (int, error) {
	contracts, err := fixtureStoryContracts(root)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, contract := range contracts {
		if contract.Composition.Fixture.Asset == fixtureID {
			count++
		}
	}
	return count, nil
}

func fixtureFailureAssertions(root, fixtureID string) (int, error) {
	contracts, err := fixtureStoryContracts(root)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, contract := range contracts {
		if contract.Composition.Fixture.Asset != fixtureID {
			continue
		}
		for _, story := range contract.Stories {
			for _, expectation := range story.Expect {
				if expectation.Role == "alert" ||
					(expectation.Attribute == "data-fixture-state" && expectation.Value == "failure") ||
					expectation.Selector == `[data-fixture-state="failure"]` ||
					strings.Contains(strings.ToLower(expectation.Value), "failure") ||
					strings.Contains(strings.ToLower(expectation.Value), "error") {
					count++
				}
			}
		}
	}
	return count, nil
}

// ValidateExamples checks that renderable assets have a public story contract
// beside their released source. Enum completeness is validated by the story
// contract parser in the registry; this gate owns the filesystem-level
// requirement so coverage never promotes a primitive with no specimen.
func ValidateExamples(root string) (Result, error) {
	result := Result{}
	for _, kind := range []string{"components", "primitives"} {
		manifests, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		sort.Strings(manifests)
		for _, manifestPath := range manifests {
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return Result{}, err
			}
			var manifest struct {
				Latest string `json:"latest"`
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return Result{}, err
			}
			result.Inspected++
			storyPath := filepath.Join(filepath.Dir(manifestPath), "versions", manifest.Latest, "story.json")
			if _, err := os.Stat(storyPath); err != nil {
				if os.IsNotExist(err) {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog.examples_missing", AssetID: filepath.Base(filepath.Dir(manifestPath)), File: repoRel(root, storyPath),
						Message:     fmt.Sprintf("released version %s has no story.json specimen", manifest.Latest),
						Remediation: fmt.Sprintf("Author %s describing at least one rendering of this asset. The specimen is what the preview surface, the visual gate, and every adopting scenario's picker all read; a released renderable asset without one is invisible in the catalog UI and is skipped by every gate that needs something to render.", repoRel(root, storyPath)),
						DocsRef:     "docs/internal/TESTING.md",
					})
					continue
				}
				return Result{}, err
			}
		}
	}
	return nonEmpty(result, "examples"), nil
}

// ValidateStress requires every active renderable implementation to have an
// indexed story contract. The story contract is the stress fixture boundary:
// it is where long, empty, disabled, and large-value specimens are declared
// and version-pinned for the browser runner.
func ValidateStress(root string) (Result, error) {
	const stressDocs = "docs/internal/TESTING.md"
	return validateActiveSources(root, "stress", func(asset assetDoc, source string) defect {
		_ = source
		manifest, _, found, err := implementationSource(root, asset.Asset.ID)
		if err != nil || !found {
			return defect{
				Message:     "no active implementation is available to the stress runner",
				Remediation: "Point this catalog asset at a library implementation, or drop the stress gate from its appliesTo list. The stress runner has nothing to drive without one, and a gate that inspects nothing must not report a pass.",
				DocsRef:     stressDocs,
			}
		}
		data, readErr := os.ReadFile(manifest)
		if readErr != nil {
			return defect{
				Message:     fmt.Sprintf("implementation manifest could not be read: %v", readErr),
				Remediation: fmt.Sprintf("Restore or repair %s. Without a readable manifest the runner cannot resolve which version to stress.", repoRel(root, manifest)),
				DocsRef:     stressDocs,
			}
		}
		var doc struct {
			Latest string `json:"latest"`
		}
		if json.Unmarshal(data, &doc) != nil || doc.Latest == "" {
			return defect{
				Message:     "implementation manifest declares no released version",
				Remediation: fmt.Sprintf("Set \"latest\" in %s to the version this asset publishes. The stress runner resolves its specimens through the released version; with none declared there is nothing to stress.", repoRel(root, manifest)),
				DocsRef:     stressDocs,
			}
		}
		storyPath := filepath.Join(filepath.Dir(manifest), "versions", doc.Latest, "story.json")
		story, readErr := os.ReadFile(storyPath)
		if readErr != nil || len(bytes.TrimSpace(story)) == 0 {
			return defect{
				Message:     fmt.Sprintf("released version %s has no non-empty story contract", doc.Latest),
				Remediation: fmt.Sprintf("Author %s with the adversarial specimens this asset must survive: long strings, empty collections, disabled states, and large numeric values. The story contract is the stress fixture boundary — it is the only place those specimens are version-pinned, so an asset without one is never driven past its happy path.", repoRel(root, storyPath)),
				DocsRef:     stressDocs,
			}
		}
		return ok()
	})
}

// ValidateIntegration checks the source-level integration boundary shared by
// every released renderable asset. The actual manager/browser integration is
// recorded by component-test and Experience Manager evidence; this runner
// prevents a source-only asset from receiving an integration pass.
func ValidateIntegration(root string) (Result, error) {
	return validateActiveSources(root, "integration", func(asset assetDoc, source string) defect {
		if strings.TrimSpace(source) == "" {
			return defect{
				Message:     "released integration source is empty",
				Remediation: "Implement this asset, or move it back to draft. An empty released source passes every static gate by having nothing to inspect, which is how an asset reaches production-ready while rendering nothing.",
				DocsRef:     "docs/internal/SEAMS.md",
			}
		}
		// The active component manifest supplies the exact released version;
		// source identity may use the library marker or the established
		// adoption-facade marker. Both are valid integration boundaries, while
		// an unowned source is not.
		if !strings.Contains(source, "@libraryId") && !strings.Contains(source, "@vrooliComponentSource") {
			return defect{
				Message:     "released source carries neither @libraryId nor @vrooliComponentSource identity metadata",
				Remediation: "Add an @libraryId docblock tag naming this asset's library id (or @vrooliComponentSource if the file is an adoption facade). Identity metadata is how the adoption reconciler tells a library-owned file from a scenario-owned one; without it this file is invisible to drift detection and will silently diverge from the version it was copied from.",
				DocsRef:     "docs/internal/SEAMS.md",
			}
		}
		return ok()
	})
}

// defect is what a source check reports. Remediation is required alongside
// Message: a check that can describe a defect can describe its fix, and the
// pair is what makes the finding actionable without a second investigation.
type defect struct{ Message, Remediation, DocsRef string }

func ok() defect { return defect{} }

func validateActiveSources(root, gate string, check func(asset assetDoc, source string) defect) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.Asset.Kind != "component" && asset.Asset.Kind != "navigation" && asset.Asset.Kind != "primitive" && asset.Asset.Kind != "pattern" && asset.Asset.Kind != "page-template" {
			continue
		}
		_, source, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if !ok {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		if d := check(asset, string(data)); d.Message != "" {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog." + gate, AssetID: asset.Asset.ID, File: repoRel(root, source),
				Message: d.Message, Remediation: d.Remediation, DocsRef: d.DocsRef,
			})
		}
	}
	return nonEmpty(result, gate), nil
}

func nonEmpty(result Result, gate string) Result {
	if result.Inspected == 0 {
		result.RunnerError = append(result.RunnerError, Finding{
			Code:        "catalog." + gate + "_zero_inspected",
			AssetID:     "",
			Message:     "gate inspected zero inputs; runner configuration is stale or broken",
			Remediation: "Treat this as a runner fault, never as a pass. The most common cause is a source-glob that no longer matches the tree — check the path pattern this gate resolves against, and whether an asset kind or directory was renamed without updating it. A gate reporting no findings after inspecting nothing is indistinguishable from a clean corpus, which is exactly the failure this finding exists to make visible.",
			DocsRef:     "docs/internal/TESTING.md",
		})
	}
	return result
}

// UnmeasuredGate returns the built catalog asset set for a gate that has no
// runner. The set is only an attribution boundary; it is not an observation
// and therefore carries an explicit unmeasured status.
func UnmeasuredGate(root string) (Result, error) {
	result := Result{}
	kinds := []string{"foundations", "hooks", "services", "primitives", "components"}
	for _, kind := range kinds {
		manifests, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return Result{}, err
			}
			var doc struct {
				CatalogID string `json:"catalogId"`
				Latest    string `json:"latest"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return Result{}, fmt.Errorf("parse %s: %w", manifest, err)
			}
			if doc.CatalogID == "" || doc.Latest == "" {
				continue
			}
			result.Inspected++
			result.InspectedAssets = append(result.InspectedAssets, doc.CatalogID)
		}
	}
	result.Status = "unmeasured"
	return result, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
