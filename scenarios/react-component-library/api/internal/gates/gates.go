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
	"react-component-library/internal/utilityclass"
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
	InspectedVersions     int
	InspectedAssets       []string
	Skipped               []string
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

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ValidateGraphReconciled is a blocking observation in catalog/config.json.
// It never repairs a manifest, but every unavailable or divergent dependency
// view must fail so the catalog cannot claim a trustworthy graph while its
// source evidence is missing.
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
			Remediation: "Start typescript-code-graph and confirm it has indexed scenarios/react-component-library; the gate is blocking because an unavailable import graph cannot prove dependency reconciliation.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
		})
		return result, nil
	}
	for _, row := range report.Assets {
		if row.Verdict == graphreconcile.Reconciled {
			continue
		}
		remediation := "Bring the three dependency views into agreement: the requires edges in catalog/assets/, the dependencies[] pins in the asset's library/**/component.json, and the imports the source actually makes. Whichever two agree usually identifies the stale one. The gate reports and never edits library/ on your behalf."
		switch row.Verdict {
		case graphreconcile.ImportsUnavailable:
			remediation = "The reconciler could not obtain an import graph from typescript-code-graph, so the source-import view is missing and no reconciliation verdict is possible. Start the graph service and confirm it has indexed scenarios/react-component-library."
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

// ValidateReleaseProvenance rejects release directories that did not pass
// through the draft publisher. Historical releases carry an explicit
// backfilled marker; new releases must name their draft and publication time.
func ValidateReleaseProvenance(root string) (Result, error) {
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	raw, err := os.ReadFile(filepath.Join(libraryRoot, "release-provenance.json"))
	if err != nil {
		return Result{RunnerError: []Finding{{
			Code: "catalog.release_provenance_unavailable", AssetID: "__corpus__.release-provenance",
			Message:     "release provenance ledger is unavailable: " + err.Error(),
			Remediation: "Restore library/release-provenance.json; the bypass-prevention gate cannot pass without its durable ledger.",
			DocsRef:     "docs/guides/asset-update-flow.md",
		}}}, nil
	}
	var ledger struct {
		Entries []struct {
			LibraryID    string `json:"libraryId"`
			Version      string `json:"version"`
			DraftVersion string `json:"draftVersion"`
			PublishedAt  string `json:"publishedAt"`
			Backfilled   bool   `json:"backfilled"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &ledger); err != nil {
		return Result{}, fmt.Errorf("decode release provenance ledger: %w", err)
	}
	recorded := map[string]bool{}
	for _, entry := range ledger.Entries {
		if strings.TrimSpace(entry.PublishedAt) == "" || (!entry.Backfilled && (!strings.Contains(entry.DraftVersion, "-") || strings.TrimSpace(entry.DraftVersion) == "")) {
			continue
		}
		recorded[entry.LibraryID+"@"+entry.Version] = true
	}
	result := Result{}
	manifests, err := filepath.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	if err != nil {
		return Result{}, err
	}
	for _, manifestPath := range manifests {
		manifestRaw, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Result{}, readErr
		}
		var manifest struct {
			LibraryID string `json:"libraryId"`
		}
		if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
			return Result{}, err
		}
		entries, readErr := os.ReadDir(filepath.Join(filepath.Dir(manifestPath), "versions"))
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(entry.Name()) {
				continue
			}
			result.Inspected++
			if recorded[manifest.LibraryID+"@"+entry.Name()] {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.release_provenance_missing", AssetID: "__corpus__.release-provenance",
				File:        repoRel(root, filepath.Join(filepath.Dir(manifestPath), "versions", entry.Name())),
				Message:     fmt.Sprintf("released directory %s@%s has no valid publish or backfill record", manifest.LibraryID, entry.Name()),
				Remediation: "Remove the bypass release and publish it through `react-component-library components draft publish`; never backfill a newly-created release.",
				DocsRef:     "docs/guides/asset-update-flow.md",
			})
		}
	}
	return nonEmpty(result, "release-provenance"), nil
}

// ValidateDependencyRank enforces the composition direction over generated
// per-version locks, which are the durable projection of real source imports.
func ValidateDependencyRank(root string) (Result, error) {
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	rankByKind := map[string]int{"foundations": 1, "hooks": 2, "services": 2, "adapters": 2, "primitives": 3, "components": 4, "patterns": 5, "navigation": 5, "page-templates": 6}
	type assetRank struct {
		rank int
		kind string
	}
	byLibraryID := map[string]assetRank{}
	manifests, _ := filepath.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	for _, manifestPath := range manifests {
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			return Result{}, err
		}
		var manifest struct {
			LibraryID string `json:"libraryId"`
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return Result{}, err
		}
		kind := filepath.Base(filepath.Dir(filepath.Dir(manifestPath)))
		byLibraryID[manifest.LibraryID] = assetRank{rank: rankByKind[kind], kind: kind}
	}
	locks, _ := filepath.Glob(filepath.Join(libraryRoot, "*", "*", "versions", "*", "dependencies.json"))
	result := Result{Inspected: len(locks)}
	for _, lockPath := range locks {
		raw, err := os.ReadFile(lockPath)
		if err != nil {
			return Result{}, err
		}
		var lock struct {
			LibraryID    string `json:"libraryId"`
			Version      string `json:"version"`
			Dependencies []struct {
				LibraryID string `json:"libraryId"`
				Version   string `json:"version"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(raw, &lock); err != nil {
			return Result{}, err
		}
		owner, ownerKnown := byLibraryID[lock.LibraryID]
		if !ownerKnown || owner.rank == 0 {
			continue
		}
		for _, dependency := range lock.Dependencies {
			target, known := byLibraryID[dependency.LibraryID]
			if !known {
				continue
			}
			if target.kind != "fixtures" && target.kind != "generators" && target.rank <= owner.rank {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.dependency_rank", AssetID: "__corpus__.dependency-rank", File: repoRel(root, lockPath),
				Message:     fmt.Sprintf("%s@%s (%s rank %d) imports %s@%s (%s rank %d)", lock.LibraryID, lock.Version, owner.kind, owner.rank, dependency.LibraryID, dependency.Version, target.kind, target.rank),
				Remediation: "Invert the dependency, extract a lower-rank seam, or remove the fixture/generator import from the composing asset; do not waive the edge.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-composition-ranks",
			})
		}
	}
	return nonEmpty(result, "dependency-rank"), nil
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
	manifests, _ := filepath.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	for _, manifestPath := range manifests {
		raw, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Result{}, readErr
		}
		var manifest struct {
			LibraryID       string   `json:"libraryId"`
			EvictedVersions []string `json:"evictedVersions"`
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return Result{}, err
		}
		for _, version := range manifest.EvictedVersions {
			versionDir := filepath.Join(filepath.Dir(manifestPath), "versions", version)
			if _, statErr := os.Stat(versionDir); statErr == nil {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.evicted_version_materialized", AssetID: "__corpus__.version-liveness",
					File:        repoRel(root, versionDir),
					Message:     fmt.Sprintf("evicted version remains materialized: %s@%s", manifest.LibraryID, version),
					Remediation: "Reconcile every dependency lock, then remove the exact evicted directory through the version lifecycle cleanup flow.",
					DocsRef:     "docs/concepts/ARCHITECTURE.md#version-lifecycle",
				})
			}
		}
	}
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
		return validateReleasedVersionHashLedger(root)
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
		return validateReleasedVersionHashLedger(root)
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
	if result.Inspected == 0 {
		return validateReleasedVersionHashLedger(root)
	}
	return nonEmpty(result, "released-version-immutable"), nil
}

type releasedVersionHashLedger struct {
	SchemaVersion int `json:"schemaVersion"`
	Entries       []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"entries"`
}

func validateReleasedVersionHashLedger(root string) (Result, error) {
	ledgerPath := filepath.Join(root, "scenarios", "react-component-library", "library", "released-version-hashes.json")
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		return Result{RunnerError: []Finding{{
			Code: "catalog.released_version_immutable", AssetID: "__corpus__.released-version-hashes", File: repoRel(root, ledgerPath),
			Message:     "component index is unavailable or empty and the released-version hash ledger cannot be read",
			Remediation: "Restore library/released-version-hashes.json; an immutability gate must never pass without a byte oracle.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning",
		}}}, nil
	}
	var ledger releasedVersionHashLedger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return Result{}, fmt.Errorf("decode released-version hash ledger: %w", err)
	}
	result := Result{Inspected: len(ledger.Entries)}
	if len(ledger.Entries) == 0 {
		result.RunnerError = append(result.RunnerError, Finding{
			Code: "catalog.released_version_immutable", AssetID: "__corpus__.released-version-hashes", File: repoRel(root, ledgerPath),
			Message: "released-version hash ledger contains zero entries", Remediation: "Regenerate the ledger from every released version file; zero inspected inputs is not immutability evidence.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning",
		})
		return result, nil
	}
	for _, entry := range ledger.Entries {
		path := filepath.Join(root, "scenarios", "react-component-library", "library", filepath.FromSlash(entry.Path))
		raw, err := os.ReadFile(path)
		if err != nil {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.released_version_immutable", AssetID: implementationName(path), File: filepath.ToSlash(filepath.Join("library", entry.Path)),
				Message: fmt.Sprintf("released file cannot be read: %v", err), Remediation: "Restore the released file or retire the version through the governed lifecycle.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning",
			})
			continue
		}
		sum := sha256.Sum256(raw)
		current := hex.EncodeToString(sum[:])
		if current == entry.SHA256 {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.released_version_immutable", AssetID: implementationName(path), File: filepath.ToSlash(filepath.Join("library", entry.Path)),
			Message:     fmt.Sprintf("released file changed after ledger capture: recorded %s, current %s", shortHash(entry.SHA256), shortHash(current)),
			Remediation: "Revert the released file and publish the change as a new version; never update the ledger to bless an in-place mutation.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning",
		})
	}
	return nonEmpty(result, "released-version-immutable"), nil
}

func shortHash(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
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

// loadLibraryAssets is the manifest-backed view used by source gates. The
// catalog projection can legitimately lag a newly authored manifest; source
// quality gates must not turn that lag into an uninspected implementation.
func loadLibraryAssets(root string) ([]assetDoc, error) {
	catalog, err := loadAssets(root)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]assetDoc, len(catalog))
	for _, asset := range catalog {
		byID[asset.Asset.ID] = asset
	}
	var result []assetDoc
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		manifests, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		sort.Strings(manifests)
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return nil, err
			}
			var metadata struct {
				CatalogID   string `json:"catalogId"`
				LibraryID   string `json:"libraryId"`
				DisplayName string `json:"displayName"`
				AssetKind   string `json:"assetKind"`
			}
			if err := json.Unmarshal(data, &metadata); err != nil {
				return nil, fmt.Errorf("parse %s: %w", manifest, err)
			}
			id := metadata.CatalogID
			if id == "" {
				id = metadata.LibraryID
			}
			if projected, ok := byID[id]; ok {
				result = append(result, projected)
				continue
			}
			assetKind := metadata.AssetKind
			if assetKind == "" {
				assetKind = strings.TrimSuffix(kind, "s")
			}
			result = append(result, assetDoc{Asset: struct {
				ID, Kind, Name, Surface string
				Target                  struct {
					Maturity string `json:"maturity"`
				} `json:"target"`
			}{ID: id, Kind: assetKind, Name: metadata.DisplayName}})
		}
	}
	return result, nil
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
	factsIndex, indexErr := readSourceFactsIndex(root)
	if indexErr != nil && !os.IsNotExist(indexErr) {
		return Result{}, indexErr
	}
	return validateActiveSourcesWithPath(root, "selector-coverage", func(asset assetDoc, path, source string) defect {
		factErr := indexErr
		facts := []sourceFacts{}
		if fact, ok := factsIndex[filepath.Clean(path)]; ok {
			facts = []sourceFacts{fact}
			factErr = nil
		}
		if factErr != nil && !os.IsNotExist(factErr) {
			return defect{Message: factErr.Error(), Remediation: "Keep the shared AST facts analyzer available to selector validation.", DocsRef: "docs/concepts/ARCHITECTURE.md#automation-selectors"}
		}
		rootSelector := false
		for _, fact := range facts {
			for _, element := range fact.Elements {
				for _, value := range element.Attributes["data-testid"] {
					if strings.Contains(value, asset.Asset.ID) {
						rootSelector = true
						break
					}
				}
			}
		}
		if !rootSelector && factErr != nil {
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
		for _, fact := range facts {
			for _, element := range fact.Elements {
				if element.Tag != "button" && element.Tag != "a" && element.Tag != "input" && element.Tag != "select" && element.Tag != "textarea" {
					continue
				}
				testIDs := element.Attributes["data-testid"]
				if len(testIDs) > 0 && strings.Contains(strings.Join(testIDs, " "), asset.Asset.ID) {
					continue
				}
				return defect{
					Message:     fmt.Sprintf("interactive <%s> has no data-testid derived from %s", element.Tag, asset.Asset.ID),
					Remediation: fmt.Sprintf("Add data-testid=%q or a derived selector rooted at %s to the interactive element.", asset.Asset.ID, asset.Asset.ID),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#automation-selectors",
				}
			}
		}
		if factErr == nil {
			return defect{}
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

// ValidateManifestIdentity keeps the source-manifest join explicit. Legacy
// domain catalog ids remain valid because they are the public catalog asset
// identity; library-prefixed ids are the identity form used by assets that do
// not yet have a domain projection and must equal libraryId exactly.
func ValidateManifestIdentity(root string) (Result, error) {
	result := Result{}
	catalog, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	knownCatalogIDs := make(map[string]bool, len(catalog))
	for _, asset := range catalog {
		knownCatalogIDs[asset.Asset.ID] = true
	}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		for _, manifest := range paths {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return Result{}, err
			}
			var doc struct {
				CatalogID string `json:"catalogId"`
				LibraryID string `json:"libraryId"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return Result{}, err
			}
			result.Inspected++
			if doc.CatalogID == "" {
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_identity", AssetID: doc.LibraryID, File: repoRel(root, manifest), Message: "manifest omits required catalogId", Remediation: "Add catalogId to the manifest and keep it stable for the asset's catalog projection.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
				continue
			}
			if strings.HasPrefix(doc.CatalogID, "react-component-library:") && doc.CatalogID != doc.LibraryID {
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_identity", AssetID: doc.CatalogID, File: repoRel(root, manifest), Message: fmt.Sprintf("catalogId %q does not equal libraryId %q", doc.CatalogID, doc.LibraryID), Remediation: "Use the manifest's libraryId as its library-prefixed catalogId.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
				continue
			}
			if !knownCatalogIDs[doc.CatalogID] && doc.CatalogID != doc.LibraryID {
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_identity", AssetID: doc.CatalogID, File: repoRel(root, manifest), Message: fmt.Sprintf("catalogId %q has no matching catalog projection or library identity", doc.CatalogID), Remediation: "Add the catalog projection or use the manifest's library-prefixed identity.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
			}
		}
	}
	return nonEmpty(result, "manifest-identity"), nil
}

// ValidateManifestMetadata keeps authored assets discoverable and prevents
// transitional catalog escape hatches from becoming permanent public state.
func ValidateManifestMetadata(root string) (Result, error) {
	result := Result{}
	for _, kind := range []string{"components"} {
		paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		for _, manifest := range paths {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return Result{}, err
			}
			var doc struct {
				LibraryID                 string `json:"libraryId"`
				CatalogID                 string `json:"catalogId"`
				Description               string `json:"description"`
				SupplementalJustification string `json:"x-supplementalJustification"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return Result{}, err
			}
			result.Inspected++
			assetID := strings.TrimSpace(doc.CatalogID)
			if assetID == "" {
				assetID = doc.LibraryID
			}
			switch {
			case strings.TrimSpace(doc.SupplementalJustification) != "":
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_metadata", AssetID: assetID, File: repoRel(root, manifest), Message: "manifest carries x-supplementalJustification", Remediation: "Register the asset against its catalog projection or remove the transitional justification.", DocsRef: "docs/reference/overlay-selection.md"})
			case strings.TrimSpace(doc.Description) == "":
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_metadata", AssetID: assetID, File: repoRel(root, manifest), Message: "manifest description is empty", Remediation: "Add a concise description of the asset's user-visible responsibility.", DocsRef: "docs/reference/overlay-selection.md"})
			case strings.HasPrefix(strings.TrimSpace(doc.CatalogID), "react-component-library:"):
				result.Findings = append(result.Findings, Finding{Code: "catalog.manifest_metadata", AssetID: doc.CatalogID, File: repoRel(root, manifest), Message: "manifest uses a self-referential catalogId", Remediation: "Use the stable domain catalog id, or clear catalogId when no projection exists.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection"})
			}
		}
	}
	return nonEmpty(result, "manifest-metadata"), nil
}

// ValidateOverlaySurfaceComposition keeps modal and menu behavior on the
// shared overlay substrate. An opt-out is permitted only when the manifest
// carries a non-empty reason, making the exception reviewable.
func ValidateOverlaySurfaceComposition(root string) (Result, error) {
	result := Result{}
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "components", "*", "component.json"))
	if err != nil {
		return Result{}, err
	}
	for _, manifest := range paths {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return Result{}, err
		}
		var doc struct {
			LibraryID                  string `json:"libraryId"`
			CatalogID                  string `json:"catalogId"`
			Category                   string `json:"category"`
			Latest                     string `json:"latest"`
			Draft                      string `json:"draft"`
			OverlaySurfaceOptOutReason string `json:"overlaySurfaceOptOutReason"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return Result{}, err
		}
		if doc.Category != "overlays" && !strings.HasPrefix(doc.CatalogID, "overlays.") {
			continue
		}
		assetID := strings.TrimSpace(doc.CatalogID)
		if assetID == "" {
			assetID = doc.LibraryID
		}
		for _, version := range []string{doc.Latest, doc.Draft} {
			version = strings.TrimSpace(version)
			if version == "" {
				continue
			}
			versionDir := filepath.Join(filepath.Dir(manifest), "versions", version)
			entries, readErr := os.ReadDir(versionDir)
			if readErr != nil {
				return Result{}, readErr
			}
			for _, entry := range entries {
				if entry.IsDir() || !(strings.HasSuffix(entry.Name(), ".tsx") || strings.HasSuffix(entry.Name(), ".ts")) {
					continue
				}
				path := filepath.Join(versionDir, entry.Name())
				source, readErr := os.ReadFile(path)
				if readErr != nil {
					return Result{}, readErr
				}
				result.Inspected++
				overlayRole, factErr := sourceHasOverlayRole(root, path, source)
				if factErr != nil {
					return Result{}, factErr
				}
				if overlayRole && !bytes.Contains(source, []byte("useOverlaySurface")) && strings.TrimSpace(doc.OverlaySurfaceOptOutReason) == "" {
					result.Findings = append(result.Findings, Finding{Code: "catalog.overlay_surface_composition", AssetID: assetID, File: repoRel(root, path), Message: "overlay role is implemented without useOverlaySurface", Remediation: "Compose useOverlaySurface, or add overlaySurfaceOptOutReason with a concrete non-overlay rationale.", DocsRef: "docs/reference/overlay-contract.md"})
				}
			}
		}
	}
	return nonEmpty(result, "overlay-surface-composition"), nil
}

var (
	styleTagRE             = regexp.MustCompile(`(?s)<style\b`)
	sharedMotionRE         = regexp.MustCompile(`(?is)@media\s*\(\s*prefers-reduced-motion\s*:`)
	sharedForcedColorsRE   = regexp.MustCompile(`(?is)@media\s*\(\s*forced-colors\s*:\s*active\s*\)`)
	sharedFocusRE          = regexp.MustCompile(`(?is)[^{}]*:focus-visible[^{}]*\{`)
	sharedVisuallyHiddenRE = regexp.MustCompile(`(?is)clip-path\s*:\s*inset\s*\(\s*50%`)
	componentSourceRE      = regexp.MustCompile(`(?m)@vrooliComponentSource\s+([^*\s]+)`)
	libraryImportRE        = regexp.MustCompile(`@vrooli/react-component-library/([^/\s'";]+)/([^/\s'";]+)`)
	scenarioRCLPinRE       = regexp.MustCompile(`@vrooli/react-component-library/([^/\s'";]+)/([0-9]+(?:\.[0-9]+){0,2})`)
)

// ValidateSharedStyleOwnership keeps cross-cutting rules in BaseStyles. An
// asset-local copy is not harmless duplication: it gives render order control
// over focus, motion, and forced-colors behavior.
func ValidateSharedStyleOwnership(root string) (Result, error) {
	return validateActiveSourceFiles(root, "shared-style-ownership", func(asset assetDoc, source string) defect {
		if strings.HasSuffix(asset.Asset.ID, ":BaseStyles") || strings.HasSuffix(asset.Asset.ID, ".base-styles") {
			return ok()
		}
		switch {
		case sharedMotionRE.MatchString(source):
			return defect{Message: "declares a local prefers-reduced-motion rule", Remediation: "Use the shared BaseStyles foundation; an asset must not redefine the library-wide motion policy.", DocsRef: "docs/reference/style-ownership.md"}
		case sharedForcedColorsRE.MatchString(source):
			return defect{Message: "declares a local forced-colors rule", Remediation: "Use the shared BaseStyles foundation; an asset must not redefine the library-wide forced-colors policy.", DocsRef: "docs/reference/style-ownership.md"}
		case sharedFocusRE.MatchString(source):
			return defect{Message: "declares a local focus-visible rule", Remediation: "Use the shared BaseStyles focus ring; keep asset-specific focus styling limited to named states.", DocsRef: "docs/reference/style-ownership.md"}
		case sharedVisuallyHiddenRE.MatchString(source):
			return defect{Message: "declares a local visually-hidden rule", Remediation: "Use the shared BaseStyles visually-hidden utility.", DocsRef: "docs/reference/style-ownership.md"}
		default:
			return ok()
		}
	})
}

// ValidateStyleInjection rejects style elements emitted from component output.
// The only supported runtime path is useLibraryStyleSheet, whose document-head
// ownership is independently tested by the foundation package.
func ValidateStyleInjection(root string) (Result, error) {
	return validateActiveSourceFiles(root, "style-injection", func(_ assetDoc, source string) defect {
		if styleTagRE.MatchString(source) {
			return defect{Message: "renders a style element from component output", Remediation: "Move the stylesheet into a module-level string and mount it with useLibraryStyleSheet so instances share one head node.", DocsRef: "docs/reference/style-ownership.md"}
		}
		return ok()
	})
}

// ValidateForeignTokenClasses is the compatibility name for the superseded
// palette-only gate. Keep it delegated so existing catalog evidence and
// calibration fixtures retain their stable gate identity.
func ValidateForeignTokenClasses(root string) (Result, error) {
	result, err := validateNoUtilityClasses(root, "foreign-token-classes")
	return result, err
}

type utilityClassAllowance struct {
	SchemaVersion int `json:"schemaVersion"`
	Expires       string
	Entries       []struct {
		Path     string
		Reason   string
		ClosedBy string
	}
}

// ValidateNoUtilityClasses enforces the package portability boundary across
// every released source file. The allowlist is explicit migration debt and is
// surfaced as informational evidence; any unlisted hit is blocking.
func ValidateNoUtilityClasses(root string) (Result, error) {
	return validateNoUtilityClasses(root, "utility-class")
}

func validateNoUtilityClasses(root, gate string) (Result, error) {
	allowlistPath := filepath.Join(root, "scenarios", "react-component-library", "library", "utility-class-allowlist.json")
	data, err := os.ReadFile(allowlistPath)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	var allowlist utilityClassAllowance
	if len(data) > 0 {
		if err := json.Unmarshal(data, &allowlist); err != nil {
			return Result{}, fmt.Errorf("decode utility-class allowlist: %w", err)
		}
	}
	allowed := make(map[string]bool, len(allowlist.Entries))
	for _, entry := range allowlist.Entries {
		allowed[filepath.ToSlash(filepath.Clean(entry.Path))] = true
	}
	sources, err := allLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	runtimeSources := sources[:0]
	for _, path := range sources {
		base := strings.ToLower(filepath.Base(path))
		if strings.HasPrefix(base, "story.") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
			continue
		}
		runtimeSources = append(runtimeSources, path)
	}
	result := Result{Inspected: len(runtimeSources)}
	observedAllowed := map[string]bool{}
	for _, path := range runtimeSources {
		source, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		rel := repoRel(root, path)
		versionPath := libraryVersionPath(rel)
		for _, hit := range utilityclass.EmitsAny(string(source)) {
			finding := Finding{
				Code:        "catalog." + gate,
				AssetID:     implementationName(path),
				File:        rel,
				Line:        lineOf(source, hit.Class),
				Category:    hit.Category,
				Message:     fmt.Sprintf("emits utility class %q (%s)", hit.Class, hit.Category),
				Remediation: "Replace library-owned utility classes with a module stylesheet and semantic custom properties; consumer-supplied className values may pass through unchanged.",
				DocsRef:     "docs/reference/style-ownership.md",
			}
			if allowed[versionPath] {
				if !observedAllowed[versionPath] {
					observedAllowed[versionPath] = true
					finding.Message = fmt.Sprintf("allow-listed utility-class debt remains in %s", versionPath)
					result.InformationalFindings = append(result.InformationalFindings, finding)
				}
				continue
			}
			result.Findings = append(result.Findings, finding)
		}
	}
	for path := range allowed {
		if observedAllowed[path] {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog." + gate + ".stale-allowance",
			AssetID:     "__corpus__.utility-class-allowlist",
			File:        repoRel(root, allowlistPath),
			Message:     fmt.Sprintf("allowlist entry %q no longer covers an emitted utility class", path),
			Remediation: "Remove the stale entry and lower utility-class-allowlist.max.",
			DocsRef:     "docs/reference/style-ownership.md",
		})
	}
	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].File == result.Findings[j].File {
			return result.Findings[i].Line < result.Findings[j].Line
		}
		return result.Findings[i].File < result.Findings[j].File
	})
	return nonEmpty(result, gate), nil
}

func libraryVersionPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for index := 0; index+4 < len(parts); index++ {
		if parts[index] == "library" && parts[index+3] == "versions" {
			return strings.Join(parts[index:index+5], "/")
		}
	}
	return ""
}

// ValidateDeprecatedImports checks the active catalog source against its own
// manifest's deprecatedVersions list. Released historical versions remain
// immutable and may still contain the dependency pins that were valid when
// they were published; active sources are the surface this gate governs.
func ValidateDeprecatedImports(root string) (Result, error) {
	deprecated, err := deprecatedLibraryVersions(root)
	if err != nil {
		return Result{}, err
	}
	sources, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		for _, match := range libraryImportRE.FindAllStringSubmatch(string(data), -1) {
			if len(match) < 3 || !contains(deprecated[match[1]], match[2]) {
				continue
			}
			result.Findings = append(result.Findings, Finding{Code: "catalog.deprecated-import", AssetID: implementationName(path), File: repoRel(root, path), Line: lineOf(data, match[0]), Message: fmt.Sprintf("imports deprecated %s@%s", match[1], match[2]), Remediation: fmt.Sprintf("Import %s at its non-deprecated published version instead of pinning %s.", match[1], match[1]+"/"+match[2]), DocsRef: "docs/concepts/ARCHITECTURE.md#version-lifecycle"})
		}
	}
	return nonEmpty(result, "deprecated-import"), nil
}

type consumerPin struct {
	Asset     string
	Version   string
	Scenarios map[string]bool
	Files     []string
}

type consumerPinManifest struct {
	CatalogID          string   `json:"catalogId"`
	LibraryID          string   `json:"libraryId"`
	Latest             string   `json:"latest"`
	DeprecatedVersions []string `json:"deprecatedVersions"`
	Root               string
}

// ValidateConsumerPins inspects the exact asset-version surface imported by
// scenarios and groups each defect with every affected consumer.
func ValidateConsumerPins(root string) (Result, error) {
	manifests, err := consumerPinManifests(root)
	if err != nil {
		return Result{}, err
	}
	pins, err := scenarioConsumerPins(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(pins)}
	for _, pin := range pins {
		manifest, exists := manifests[pin.Asset]
		assetID := "__corpus__.consumer-pins"
		if exists {
			assetID = manifest.CatalogID
			if assetID == "" {
				assetID = manifest.LibraryID
			}
		}
		scenarios := sortedStringMapKeys(pin.Scenarios)
		location := ""
		if len(pin.Files) > 0 {
			location = pin.Files[0]
		}
		add := func(suffix, message, remediation string) {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.consumer-pin." + suffix, AssetID: assetID, File: location,
				Message:     fmt.Sprintf("%s@%s consumed by %s: %s", pin.Asset, pin.Version, strings.Join(scenarios, ", "), message),
				Remediation: remediation, DocsRef: "docs/reference/style-ownership.md",
			})
		}
		if !exists {
			add("missing", "asset manifest does not exist", "Restore the published asset or migrate every named consumer to an existing asset.")
			continue
		}
		resolved, versionRoot := resolveConsumerPinVersion(manifest, pin.Version)
		if resolved == "" {
			add("missing", "published version does not exist", "Migrate every named consumer to a version present in the asset manifest and version tree.")
			continue
		}
		if contains(manifest.DeprecatedVersions, resolved) {
			add("deprecated", "version is deprecated", "Migrate every named consumer to a non-deprecated version in the same supported major.")
		}
		if versionMajor(resolved) < versionMajor(manifest.Latest) {
			add("stale-major", fmt.Sprintf("latest is %s", manifest.Latest), "Migrate every named consumer to the current supported major, then use a major-scoped import.")
		}
		emits := false
		if err := filepath.WalkDir(versionRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || emits || entry.IsDir() {
				return walkErr
			}
			base := strings.ToLower(filepath.Base(path))
			ext := strings.ToLower(filepath.Ext(path))
			if (ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx") || strings.HasPrefix(base, "story.") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			emits = len(utilityclass.EmitsAny(string(raw))) > 0
			return nil
		}); err != nil {
			return Result{}, err
		}
		if emits {
			add("utility-class", "version emits library-owned utility classes", "Publish a token-bound version and migrate every named consumer; a Tailwind content glob is only a temporary bridge.")
		}
	}
	return nonEmpty(result, "consumer-pin"), nil
}

func consumerPinManifests(root string) (map[string]consumerPinManifest, error) {
	result := map[string]consumerPinManifest{}
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, err := filepath.Glob(filepath.Join(libraryRoot, kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			var manifest consumerPinManifest
			if err := json.Unmarshal(raw, &manifest); err != nil {
				return nil, err
			}
			manifest.Root = filepath.Dir(path)
			result[filepath.Base(filepath.Dir(path))] = manifest
		}
	}
	return result, nil
}

func scenarioConsumerPins(root string) ([]consumerPin, error) {
	byKey := map[string]*consumerPin{}
	scenariosRoot := filepath.Join(root, "scenarios")
	err := filepath.WalkDir(scenariosRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".retired" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(scenariosRoot, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 4 || parts[1] != "ui" || parts[2] != "src" {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range scenarioRCLPinRE.FindAllStringSubmatch(string(raw), -1) {
			key := match[1] + "@" + match[2]
			pin := byKey[key]
			if pin == nil {
				pin = &consumerPin{Asset: match[1], Version: match[2], Scenarios: map[string]bool{}}
				byKey[key] = pin
			}
			pin.Scenarios[parts[0]] = true
			pin.Files = appendUnique(pin.Files, repoRel(root, path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]consumerPin, 0, len(keys))
	for _, key := range keys {
		result = append(result, *byKey[key])
	}
	return result, nil
}

func resolveConsumerPinVersion(manifest consumerPinManifest, pin string) (string, string) {
	versionsRoot := filepath.Join(manifest.Root, "versions")
	if strings.Count(pin, ".") == 0 {
		entries, _ := os.ReadDir(versionsRoot)
		var candidates []string
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), pin+".") {
				candidates = append(candidates, entry.Name())
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return semverParts(candidates[i]) > semverParts(candidates[j]) })
		if len(candidates) == 0 {
			return "", ""
		}
		pin = candidates[0]
	}
	path := filepath.Join(versionsRoot, pin)
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return "", ""
	}
	return pin, path
}

func semverParts(version string) int64 {
	parts := strings.Split(version, ".")
	var value int64
	for index := 0; index < 3; index++ {
		value *= 1_000_000
		if index < len(parts) {
			number, _ := strconv.ParseInt(parts[index], 10, 64)
			value += number
		}
	}
	return value
}

func versionMajor(version string) int {
	major, _ := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	return major
}

func sortedStringMapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func allLibrarySources(root string) ([]string, error) {
	var sources []string
	err := filepath.WalkDir(filepath.Join(root, "scenarios", "react-component-library", "library"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".retired" {
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
		return nil, err
	}
	sort.Strings(sources)
	return sources, nil
}

func deprecatedLibraryVersions(root string) (map[string][]string, error) {
	result := map[string][]string{}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			var doc struct {
				LibraryID          string   `json:"libraryId"`
				CatalogID          string   `json:"catalogId"`
				DeprecatedVersions []string `json:"deprecatedVersions"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return nil, err
			}
			for _, key := range []string{filepath.Base(filepath.Dir(path)), strings.TrimPrefix(doc.LibraryID, "react-component-library:"), strings.TrimPrefix(doc.CatalogID, "react-component-library:")} {
				if key != "" {
					result[key] = append(result[key], doc.DeprecatedVersions...)
				}
			}
		}
	}
	return result, nil
}

// ValidateProvenanceStamp ensures a source marker describes a real adoption
// edge. Library files are checked against their owning manifest; UI files must
// import or render the stamped asset instead of copying a stale label.
func ValidateProvenanceStamp(root string) (Result, error) {
	var paths []string
	for _, base := range []string{filepath.Join(root, "scenarios", "react-component-library", "library"), filepath.Join(root, "scenarios", "react-component-library", "ui")} {
		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return Result{}, err
		}
	}
	sort.Strings(paths)
	result := Result{Inspected: len(paths)}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		match := componentSourceRE.FindStringSubmatch(string(data))
		if len(match) < 2 {
			continue
		}
		stamp := strings.TrimSpace(match[1])
		key := stamp[strings.LastIndexAny(stamp, ".:")+1:]
		if pathWithin(filepath.Join(root, "scenarios", "react-component-library", "library"), path) {
			owner := implementationName(path)
			libraryID, catalogID := libraryManifestIdentities(path)
			valid := strings.EqualFold(stamp, libraryID) || strings.EqualFold(stamp, catalogID)
			if !valid && owner != "" {
				valid = strings.EqualFold(strings.TrimPrefix(owner, "react-component-library:"), key) || strings.Contains(strings.ToLower(owner), strings.ToLower(key))
			}
			if !valid {
				result.Findings = append(result.Findings, Finding{Code: "catalog.provenance-stamp", AssetID: owner, File: repoRel(root, path), Line: lineOf(data, match[0]), Message: fmt.Sprintf("component source stamp %q does not identify its owning library asset", stamp), Remediation: "Use the owning component's stable catalog identity in @vrooliComponentSource.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-provenance"})
			}
			continue
		}
		if !strings.Contains(strings.ToLower(string(data)), strings.ToLower(key)) {
			result.Findings = append(result.Findings, Finding{Code: "catalog.provenance-stamp", AssetID: stamp, File: repoRel(root, path), Line: lineOf(data, match[0]), Message: fmt.Sprintf("component source stamp %q is not imported or rendered by this file", stamp), Remediation: "Change the stamp to the library asset actually imported or rendered by this file.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-provenance"})
		}
	}
	return nonEmpty(result, "provenance-stamp"), nil
}

func libraryManifestIdentities(sourcePath string) (string, string) {
	// sourcePath is .../<asset>/versions/<version>/<entry>.tsx. The owning
	// manifest is three directories above the source: <asset>/component.json.
	assetPath := filepath.Dir(filepath.Dir(filepath.Dir(sourcePath)))
	manifestPath := filepath.Join(assetPath, "component.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", ""
	}
	var doc struct {
		LibraryID string `json:"libraryId"`
		CatalogID string `json:"catalogId"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return "", ""
	}
	return doc.LibraryID, doc.CatalogID
}

var (
	stylePropRE          = regexp.MustCompile(`(?m)(?:^|[;{]\s*)style\?\s*:\s*([^;},\n]+)`)
	classNamePropRE      = regexp.MustCompile(`(?m)\bclassName\??\s*:`)
	classNameUseRE       = regexp.MustCompile(`\bclassName\b`)
	forwardRefRE         = regexp.MustCompile(`\bforwardRef\b`)
	refAttributeRE       = regexp.MustCompile(`\bref\s*=\s*\{[^}]*\b(?:ref|forwardedRef)\b[^}]*\}`)
	imperativeRefRE      = regexp.MustCompile(`\buseImperativeHandle\s*\(`)
	assignRefRE          = regexp.MustCompile(`\bassignRef\s*\(`)
	classNameBoundaryRE  = regexp.MustCompile(`\bwithClassName\s*\(`)
	exportedComponentRE  = regexp.MustCompile(`(?m)(?:^|[;\n])\s*export\s+(?:function|const|class)\s+[A-Z]`)
	jsxIdentifierStyleRE = regexp.MustCompile(`style\s*=\s*\{\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*\}`)
	moduleObjectRE       = regexp.MustCompile(`(?m)(?:^|[;\n])\s*(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:[:][^=\n]+)?=\s*\{`)
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
	objects := map[string]bool{}
	for _, match := range moduleObjectRE.FindAllStringSubmatch(source, -1) {
		if len(match) > 1 {
			objects[match[1]] = true
		}
	}
	for _, tag := range jsxOpeningTags(source) {
		inlineObject := jsxInlineStyleObjectRE.MatchString(tag)
		if !inlineObject {
			match := jsxIdentifierStyleRE.FindStringSubmatch(tag)
			inlineObject = len(match) > 1 && objects[match[1]]
		}
		if !inlineObject {
			continue
		}
		if jsxConsumerClassRE.MatchString(tag) || jsxSpreadRE.MatchString(tag) {
			return true
		}
	}
	// Generic TypeScript syntax can look like a JSX opening tag to a small
	// scanner (for example forwardRef<HTMLDivElement, Props>). Preserve the
	// same root-class requirement while covering that valid source shape.
	for _, match := range jsxIdentifierStyleRE.FindAllStringSubmatch(source, -1) {
		if len(match) > 1 && (objects[match[1]] || strings.Contains(source, "const "+match[1]+" = {") || strings.Contains(source, "let "+match[1]+" = {") || strings.Contains(source, "var "+match[1]+" = {")) && strings.Contains(source, "className={className}") && strings.Contains(source, "return") {
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
	reportFile, reportErr := os.CreateTemp("", "rcl-catalog-report-*.json")
	if reportErr != nil {
		return Result{}, reportErr
	}
	reportPath := reportFile.Name()
	_ = reportFile.Close()
	defer func() { _ = os.Remove(reportPath) }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "pnpm", "run", "catalog:check")
	command.Dir = uiDir
	command.Env = append(os.Environ(), "RCL_CATALOG_REPORT="+reportPath)
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
			// gate cannot then claim any asset is clean, so this is a runner
			// fault rather than a finding — the evidence mapper fails every
			// asset closed on a runner fault, and passing them would be the one
			// outcome the run does not support.
			result.RunnerError = append(result.RunnerError, Finding{
				Code:        "catalog.types_failed",
				AssetID:     "__corpus__.types",
				Message:     "catalog conformance failed: " + message,
				Remediation: "Reproduce with `pnpm run catalog:check` in scenarios/react-component-library/ui; the output above is that command's tail. Fix the reported type or lint errors at their source files — this gate deliberately reports the real toolchain's output rather than re-deriving its own verdict, so the failure it shows is the failure to fix.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
	}
	return result, nil
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
				Remediation: "Fix the reported type or lint error at its source file, then re-run `pnpm run catalog:check` in scenarios/react-component-library/ui.",
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
func libraryAssetForPath(libraryRoot, file string) string {
	relative, err := filepath.Rel(libraryRoot, filepath.Clean(file))
	if err != nil || strings.HasPrefix(relative, "..") {
		return ""
	}
	segments := strings.Split(filepath.ToSlash(relative), "/")
	if len(segments) < 2 {
		return ""
	}
	return segments[1]
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
	sources, err := implementationSources(root, catalogID)
	if err != nil || len(sources) == 0 {
		return "", "", false, err
	}
	for _, source := range sources {
		if source.Version == source.Latest {
			return source.Manifest, source.Path, true, nil
		}
	}
	source := sources[len(sources)-1]
	return source.Manifest, source.Path, true, nil
}

type implementationSourceEntry struct {
	Manifest string
	Path     string
	Version  string
	Latest   string
}

// implementationSources returns every released version that consumers can
// reach through the published package exports map. Keeping this lookup next to
// implementationSource prevents a gate from silently regressing to latest-only
// coverage when a package publishes an older pinned entry.
func implementationSources(root, catalogID string) ([]implementationSourceEntry, error) {
	paths := make([]string, 0)
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	for _, manifest := range paths {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return nil, err
		}
		var doc struct {
			CatalogID  string   `json:"catalogId"`
			LibraryID  string   `json:"libraryId"`
			Latest     string   `json:"latest"`
			Deprecated []string `json:"deprecatedVersions"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		if doc.CatalogID != catalogID && doc.LibraryID != catalogID {
			continue
		}
		return exportedImplementationSources(root, manifest, doc.Latest, doc.Deprecated), nil
	}
	return nil, nil
}

func exportedImplementationSources(root, manifest, latest string, deprecated []string) []implementationSourceEntry {
	rootDir := filepath.Dir(manifest)
	name := filepath.Base(rootDir)
	exported := exportedVersions(root, name)
	var versions []string
	entries, _ := os.ReadDir(filepath.Join(rootDir, "versions"))
	for _, entry := range entries {
		if !entry.IsDir() || containsString(deprecated, entry.Name()) {
			continue
		}
		if len(exported) > 0 && !exported[entry.Name()] {
			continue
		}
		versions = append(versions, entry.Name())
	}
	sort.Slice(versions, func(i, j int) bool { return semverLikeLess(versions[i], versions[j]) })
	result := make([]implementationSourceEntry, 0, len(versions))
	for _, version := range versions {
		versionDir := filepath.Join(rootDir, "versions", version)
		source := filepath.Join(versionDir, name+".tsx")
		if _, err := os.Stat(source); err != nil {
			matches := versionSources(versionDir)
			if len(matches) == 0 {
				continue
			}
			source = matches[0]
		}
		result = append(result, implementationSourceEntry{Manifest: manifest, Path: source, Version: version, Latest: latest})
	}
	return result
}

func exportedVersions(root, name string) map[string]bool {
	path := filepath.Join(root, "packages", "react-component-library", "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Exports map[string]json.RawMessage `json:"exports"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return nil
	}
	result := map[string]bool{}
	prefix := "./" + name + "/"
	for key := range doc.Exports {
		if strings.HasPrefix(key, prefix) {
			result[strings.TrimPrefix(key, prefix)] = true
		}
	}
	return result
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func semverLikeLess(left, right string) bool {
	parse := func(value string) [3]int {
		parts := strings.Split(value, ".")
		var out [3]int
		for i := 0; i < len(parts) && i < 3; i++ {
			out[i], _ = strconv.Atoi(parts[i])
		}
		return out
	}
	l, r := parse(left), parse(right)
	for i := range l {
		if l[i] != r[i] {
			return l[i] < r[i]
		}
	}
	return left < right
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

// ValidateSelfHosting measures whether the catalog application exercises the
// published library surface. This is a corpus observation: the catalog app
// is the consumer and the library asset set is the denominator.
func ValidateSelfHosting(root string) (Result, error) {
	uiRoot := filepath.Join(root, "scenarios", "react-component-library", "ui", "src")
	policyPath := filepath.Join(root, "scenarios", "react-component-library", "catalog", "self-hosting-policy.json")
	policyData, err := os.ReadFile(policyPath)
	if err != nil {
		return Result{RunnerError: []Finding{{
			Code: "catalog.self_hosting_policy_unavailable", AssetID: "__corpus__.self-hosting",
			Message:     "self-hosting policy is unavailable: " + err.Error(),
			Remediation: "Restore catalog/self-hosting-policy.json with the reviewed coverage floor and explicit exemptions.",
			DocsRef:     "docs/reference/composition-validation.md",
		}}}, nil
	}
	var policy struct {
		MinimumCovered int `json:"minimumCovered"`
		Exemptions     []struct {
			Pattern string `json:"pattern"`
			Reason  string `json:"reason"`
		} `json:"exemptions"`
	}
	if err := json.Unmarshal(policyData, &policy); err != nil {
		return Result{}, fmt.Errorf("decode self-hosting policy: %w", err)
	}
	assets, err := loadLibraryAssets(root)
	if err != nil {
		return Result{}, err
	}
	consumed := map[string]bool{}
	known := map[string]string{}
	for _, asset := range assets {
		if asset.Asset.ID != "" {
			known[asset.Asset.Name] = asset.Asset.ID
		}
	}
	files := 0
	err = filepath.WalkDir(uiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		files++
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range libraryImportRE.FindAllStringSubmatch(string(data), -1) {
			if assetID := known[match[1]]; assetID != "" {
				consumed[assetID] = true
			}
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: 1}
	if files == 0 {
		result.RunnerError = append(result.RunnerError, Finding{
			Code: "catalog.self_hosting_no_sources", AssetID: "__corpus__.self-hosting",
			Message:     "catalog application source tree contains no inspectable files",
			Remediation: "Restore the catalog application source tree before measuring self-hosting.",
			DocsRef:     "docs/internal/TESTING.md",
		})
		return result, nil
	}
	result.InformationalFindings = append(result.InformationalFindings, Finding{
		Code: "catalog.self_hosting_measurement", AssetID: "__corpus__.self-hosting",
		Message:     fmt.Sprintf("catalog application imports %d of %d implemented catalog assets across %d source files; floor is %d; exemptions: %d", len(consumed), len(known), files, policy.MinimumCovered, len(policy.Exemptions)),
		Remediation: "Increase the reviewed coverage floor when a new asset is consumed. Add an exemption only for a documented surface outside the catalog application's remit.",
		DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
	})
	if len(consumed) < policy.MinimumCovered {
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.self_hosting_uncovered", AssetID: "__corpus__.self-hosting",
			Message:     fmt.Sprintf("catalog application consumes %d implemented assets, below the required floor of %d", len(consumed), policy.MinimumCovered),
			Remediation: "Add real catalog application usage paths or lower the floor only through a reviewed policy change with a documented reason; keep this gate blocking.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
		})
	}
	return result, nil
}

// ValidateBASGenericity keeps browser workflows capability-driven. Component
// names and version query parameters belong in story/runner data, not in a
// workflow file that must be copied for every asset.
func ValidateBASGenericity(root string) (Result, error) {
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	componentNames := map[string]bool{}
	manifests, err := filepath.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	if err != nil {
		return Result{}, err
	}
	for _, manifestPath := range manifests {
		data, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Result{}, readErr
		}
		var manifest struct{ DisplayName, LibraryID string }
		if json.Unmarshal(data, &manifest) != nil {
			continue
		}
		name := filepath.Base(filepath.Dir(manifestPath))
		if name != "" {
			componentNames[strings.ToLower(name)] = true
		}
		if manifest.DisplayName != "" {
			componentNames[strings.ToLower(manifest.DisplayName)] = true
		}
	}
	result := Result{}
	for _, directory := range []string{"cases", "calibration"} {
		base := filepath.Join(root, "scenarios", "react-component-library", "bas", directory)
		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if entry.IsDir() || (filepath.Ext(path) != ".json" && filepath.Ext(path) != ".js") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			result.Inspected++
			text := strings.ToLower(string(data))
			if strings.Contains(text, "version=") {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.bas_version_pin", AssetID: "__corpus__.bas-genericity", File: repoRel(root, path),
					Message: "BAS workflow embeds a version query parameter", Remediation: "Pass the version through the generic runner input contract instead of encoding it in the workflow.", DocsRef: "docs/reference/composition-validation.md",
				})
			}
			// Descriptions and labels are allowed to name a capability (for
			// example, "overlay" or "surface"). Only structured asset-selection
			// fields and package imports are identity-bearing BAS knowledge.
			identityText := basIdentityText(data, path)
			for name := range componentNames {
				if regexp.MustCompile(`(^|[^a-z0-9])` + regexp.QuoteMeta(name) + `([^a-z0-9]|$)`).MatchString(strings.ToLower(identityText)) {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog.bas_component_knowledge", AssetID: "__corpus__.bas-genericity", File: repoRel(root, path),
						Message: fmt.Sprintf("BAS workflow contains component-specific name %q", name), Remediation: "Move asset selection into story capabilities and runner parameters; keep the workflow universal.", DocsRef: "docs/reference/composition-validation.md",
					})
					break
				}
			}
			return nil
		})
		if err != nil {
			return Result{}, err
		}
	}
	return nonEmpty(result, "bas-genericity"), nil
}

func basIdentityText(data []byte, path string) string {
	if filepath.Ext(path) != ".json" {
		return string(data)
	}
	var value any
	if json.Unmarshal(data, &value) != nil {
		return ""
	}
	var values []string
	var walk func(any, string)
	walk = func(node any, key string) {
		switch typed := node.(type) {
		case map[string]any:
			for childKey, child := range typed {
				lower := strings.ToLower(childKey)
				if strings.Contains(lower, "component") || strings.Contains(lower, "asset") || strings.Contains(lower, "library") || strings.Contains(lower, "story") || strings.Contains(lower, "package") {
					walk(child, lower)
				}
			}
		case []any:
			for _, child := range typed {
				walk(child, key)
			}
		case string:
			values = append(values, typed)
		}
	}
	walk(value, "")
	return strings.Join(values, "\n")
}

// defect is what a source check reports. Remediation is required alongside
// Message: a check that can describe a defect can describe its fix, and the
// pair is what makes the finding actionable without a second investigation.
type defect struct{ Message, Remediation, DocsRef string }

func ok() defect { return defect{} }

func validateActiveSources(root, gate string, check func(asset assetDoc, source string) defect) (Result, error) {
	return validateActiveSourcesWithPath(root, gate, func(asset assetDoc, _ string, source string) defect {
		return check(asset, source)
	})
}

func validateActiveSourcesWithPath(root, gate string, check func(asset assetDoc, path, source string) defect) (Result, error) {
	assets, err := loadLibraryAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.Asset.Kind != "component" && asset.Asset.Kind != "navigation" && asset.Asset.Kind != "primitive" && asset.Asset.Kind != "pattern" && asset.Asset.Kind != "page-template" {
			continue
		}
		sources, err := implementationSources(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if len(sources) == 0 {
			result.Skipped = append(result.Skipped, asset.Asset.ID)
			result.RunnerError = append(result.RunnerError, Finding{
				Code:        "catalog." + gate + "_asset_unresolved",
				AssetID:     asset.Asset.ID,
				Message:     fmt.Sprintf("no exported, non-deprecated implementation resolved for asset %q", asset.Asset.ID),
				Remediation: "Add a catalogId/libraryId-matching manifest and publish a non-deprecated version in the package exports map.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
			})
			continue
		}
		result.Inspected++
		result.InspectedAssets = append(result.InspectedAssets, asset.Asset.ID)
		for _, source := range sources {
			data, err := os.ReadFile(source.Path)
			if err != nil {
				return Result{}, err
			}
			result.InspectedVersions++
			if d := check(asset, source.Path, string(data)); d.Message != "" {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog." + gate, AssetID: asset.Asset.ID, File: repoRel(root, source.Path),
					Message: fmt.Sprintf("%s (version %s)", d.Message, source.Version), Remediation: d.Remediation, DocsRef: d.DocsRef,
				})
			}
		}
	}
	return nonEmpty(result, gate), nil
}

// validateActiveSourceFiles is used by gates whose contract applies to the
// complete implementation package, not only the component entrypoint. Style
// declarations commonly live beside that entrypoint in styles.ts, so looking
// at the first source file would make those declarations invisible.
func validateActiveSourceFiles(root, gate string, check func(asset assetDoc, source string) defect) (Result, error) {
	assets, err := loadLibraryAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.Asset.Kind != "component" && asset.Asset.Kind != "navigation" && asset.Asset.Kind != "primitive" && asset.Asset.Kind != "pattern" && asset.Asset.Kind != "page-template" {
			continue
		}
		versions, err := implementationSources(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if len(versions) == 0 {
			result.Skipped = append(result.Skipped, asset.Asset.ID)
			result.RunnerError = append(result.RunnerError, Finding{
				Code:        "catalog." + gate + "_asset_unresolved",
				AssetID:     asset.Asset.ID,
				Message:     fmt.Sprintf("no exported, non-deprecated implementation resolved for asset %q", asset.Asset.ID),
				Remediation: "Add a catalogId/libraryId-matching manifest and publish a non-deprecated version in the package exports map.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
			})
			continue
		}
		result.Inspected++
		result.InspectedAssets = append(result.InspectedAssets, asset.Asset.ID)
		for _, version := range versions {
			result.InspectedVersions++
			for _, path := range versionSources(filepath.Dir(version.Path)) {
				data, err := os.ReadFile(path)
				if err != nil {
					return Result{}, err
				}
				if d := check(asset, string(data)); d.Message != "" {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog." + gate, AssetID: asset.Asset.ID, File: repoRel(root, path),
						Message: fmt.Sprintf("%s (version %s)", d.Message, version.Version), Remediation: d.Remediation, DocsRef: d.DocsRef,
					})
				}
			}
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
