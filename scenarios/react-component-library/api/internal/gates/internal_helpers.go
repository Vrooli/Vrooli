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
	"react-component-library/internal/themes"
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
	Code           string
	Category       string
	AssetID        string
	Message        string
	File           string
	Line           int
	Remediation    string
	DocsRef        string
	RuleSource     RuleSource
	RuleDeclaredIn string
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
		if strings.HasPrefix(finding.AssetID, "workbench.") || strings.HasPrefix(finding.AssetID, "supplemental.") || strings.HasPrefix(finding.AssetID, "__corpus__") {
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
func hasImplementationSources(versionDir string) bool {
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), "story.") {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".ts", ".tsx", ".js", ".jsx":
			return true
		}
	}
	return false
}

func dependencyImportedByImplementation(versionDir, libraryID string) bool {
	name := strings.TrimPrefix(libraryID, "react-component-library:")
	if name == libraryID || name == "" {
		return false
	}
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return false
	}
	needle := "@vrooli/react-component-library/" + name + "/"
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), "story.") {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".ts", ".tsx", ".js", ".jsx":
		default:
			continue
		}
		raw, err := os.ReadFile(filepath.Join(versionDir, entry.Name()))
		if err == nil && strings.Contains(string(raw), needle) {
			return true
		}
	}
	return false
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
func firstRetiredTokenReference(raw []byte) int {
	declarations := map[int]bool{}
	for _, match := range cssVarDeclGateRE.FindAllSubmatchIndex(raw, -1) {
		if len(match) >= 4 {
			declarations[match[2]] = true
		}
	}
	retired := regexp.MustCompile(`--app-[A-Za-z0-9_-]+`)
	for _, match := range retired.FindAllIndex(raw, -1) {
		if !declarations[match[0]] {
			return match[0]
		}
	}
	return -1
}

// ValidateTokenRampComplete verifies that every external literal custom
// property used by active library source is published by the canonical RCL
// ramp. Self-defined --rcl-* properties and dynamic families are excluded.
func librarySourceFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

// queryContexter is the smallest database seam needed by the immutable gate.
// The production catalog coverage path supplies the already-open routed
// scenario database; the root-only runner above remains useful for isolated
// calibration fixtures.
type queryContexter interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// validateReleasedVersionImmutableWithDB compares released versions using an
// already-open scenario database. Scope-aware callers reach it through the
// single exported ValidateReleasedVersionImmutable runner.
func validateReleasedVersionImmutableWithDB(root string, db queryContexter) (Result, error) {
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
		if filepath.Base(sourcePath) == "dependencies.json" {
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
		// dependencies.json is generated from authored source and may be
		// regenerated when a major-line dependency resolves to a new release.
		// Released immutability protects authored source, not this derived lock.
		if !components.IsAuthoredReleaseFile(entry.Path) {
			result.Inspected--
			continue
		}
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
		ID, Kind, Name, Surface, Description string
		Target                               struct {
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

func loadAssets(scope Scope) ([]assetDoc, error) {
	root := scope.Root
	selected := make(map[string]bool, len(scope.Assets))
	for _, assetID := range scope.Assets {
		selected[assetID] = true
	}
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
		if !scope.IsFullCorpus() && !selected[doc.Asset.ID] {
			continue
		}
		out = append(out, doc)
	}
	return out, nil
}

// loadLibraryAssets is the manifest-backed view used by source gates. The
// catalog projection can legitimately lag a newly authored manifest; source
// quality gates must not turn that lag into an uninspected implementation.
func loadLibraryAssets(scope Scope) ([]assetDoc, error) {
	root := scope.Root
	catalog, err := loadAssets(scope)
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
				CatalogID    string `json:"catalogId"`
				LibraryID    string `json:"libraryId"`
				DisplayName  string `json:"displayName"`
				AssetKind    string `json:"assetKind"`
				Supplemental bool   `json:"supplemental"`
			}
			if err := json.Unmarshal(data, &metadata); err != nil {
				return nil, fmt.Errorf("parse %s: %w", manifest, err)
			}
			id := metadata.CatalogID
			if id == "" && metadata.Supplemental {
				id = "supplemental." + strings.ReplaceAll(metadata.LibraryID, ":", ".")
			}
			if projected, ok := byID[id]; ok {
				result = append(result, projected)
				continue
			}
			assetKind := metadata.AssetKind
			result = append(result, assetDoc{Asset: struct {
				ID, Kind, Name, Surface, Description string
				Target                               struct {
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
func validateNoUtilityClasses(scope Scope, gate string) (Result, error) {
	root := scope.Root
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
	sources, err := allLibrarySources(scope)
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

func allLibrarySources(scope Scope) ([]string, error) {
	root := scope.Root
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
			if !sourceInScope(root, path, scope) {
				return nil
			}
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
		manifests, _ := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
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

func countCatalogSources(scope Scope) int {
	sources, _ := activeLibrarySources(scope)
	return len(sources)
}

// activeLibrarySources returns the files represented by each manifest's
// latest and draft pointers. Historical versions remain available to callers
// that pin them explicitly, but corpus-wide quality gates should measure the
// active catalog surface consistently with indexing, coverage, and the type
// gate rather than double-counting retired implementations.
func activeLibrarySources(scope Scope) ([]string, error) {
	root := scope.Root
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
					for _, path := range matches {
						if sourceInScope(root, path, scope) {
							sources = append(sources, path)
						}
					}
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
			for _, path := range matches {
				if sourceInScope(root, path, scope) {
					sources = append(sources, path)
				}
			}
		}
	}
	sort.Strings(sources)
	return sources, nil
}

func sourceInScope(root, sourcePath string, scope Scope) bool {
	if scope.IsFullCorpus() {
		return true
	}
	selected := make(map[string]bool, len(scope.Assets))
	for _, assetID := range scope.Assets {
		selected[assetID] = true
	}
	versionDir := filepath.Dir(sourcePath)
	assetDir := filepath.Dir(filepath.Dir(versionDir))
	manifestPath := filepath.Join(assetDir, "component.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}
	var manifest struct {
		CatalogID    string `json:"catalogId"`
		LibraryID    string `json:"libraryId"`
		Supplemental bool   `json:"supplemental"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	assetID := manifest.CatalogID
	if assetID == "" && manifest.Supplemental {
		assetID = "supplemental." + strings.ReplaceAll(manifest.LibraryID, ":", ".")
	}
	return selected[assetID]
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
	path := filepath.Join(root, "packages", "react-component-library", "dist", "exports", "resolution.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return exportedVersionsFromPackage(root, name)
	}
	var resolutions map[string]struct {
		Source string `json:"source"`
	}
	if json.Unmarshal(data, &resolutions) != nil {
		return nil
	}
	result := map[string]bool{}
	prefix := "./" + name + "/"
	for key, resolution := range resolutions {
		if strings.HasPrefix(key, prefix) {
			version := strings.TrimPrefix(key, prefix)
			if isConcreteVersion(version) && resolution.Source != "" {
				result[version] = true
			}
		}
	}
	return result
}

func exportedVersionsFromPackage(root, name string) map[string]bool {
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
		version := strings.TrimPrefix(key, prefix)
		if strings.HasPrefix(key, prefix) && isConcreteVersion(version) {
			result[version] = true
		}
	}
	return result
}

func isConcreteVersion(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return false
			}
		}
	}
	return true
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
func compatibilityGateResult(census Census, affinityOnly bool) Result {
	result := Result{Inspected: census.ComponentsScanned}
	if affinityOnly {
		for _, overclaim := range census.AffinityOverclaims {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.affinity_compatible", AssetID: strings.TrimPrefix(overclaim.LibraryID, "react-component-library:"),
				Message:     fmt.Sprintf("declared affinity %s is broader than derived kit compatibility", overclaim.StyleID),
				Remediation: "Remove the aesthetic-fit claim until the named kit supplies every required token.",
				DocsRef:     "docs/design/TOKEN-DICTIONARY.md",
			})
		}
		return result
	}
	for _, component := range census.Components {
		if component.Verdict != CompatibilityUndefinedVocabulary && component.Verdict != CompatibilityUnsatisfiable {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.kit_compatibility", AssetID: strings.TrimPrefix(component.LibraryID, "react-component-library:"),
			Message:     fmt.Sprintf("derived kit compatibility is %s for required tokens %s", component.Verdict, strings.Join(component.RequiredTokens, ", ")),
			Remediation: "Publish the missing semantic vocabulary through the shared base or a deliberate kit override; do not broaden affinity metadata.",
			DocsRef:     "docs/design/TOKEN-DICTIONARY.md",
		})
	}
	return result
}

func tokenValue(tokens []themes.DesignToken, property string) string {
	for _, token := range tokens {
		if token.Name == property {
			return token.Value
		}
	}
	return ""
}

func normalizeTokenValue(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func parseTokenFallbacks(source string) []tokenFallback {
	var result []tokenFallback
	for offset := 0; offset < len(source); {
		relative := strings.Index(source[offset:], "var(")
		if relative < 0 {
			break
		}
		start := offset + relative
		depth, comma, end := 0, -1, -1
		for i := start + len("var("); i < len(source); i++ {
			switch source[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
				} else {
					depth--
				}
			case ',':
				if depth == 0 && comma < 0 {
					comma = i
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			break
		}
		if comma > 0 {
			property := strings.TrimSpace(source[start+len("var(") : comma])
			if strings.HasPrefix(property, "--") && !strings.Contains(property, "${") {
				result = append(result, tokenFallback{Property: property, Value: strings.TrimSpace(source[comma+1 : end]), Offset: start})
			}
		}
		offset = end + 1
	}
	return result
}

// ValidateLifecycle performs conservative static checks over hook/service/
// adapter/generator sources. It deliberately prefers a finding over a green
// result when cleanup evidence is absent.
func isStorySource(path string) bool {
	base := filepath.Base(path)
	return base == "story.ts" || base == "story.tsx"
}

func isTestSource(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx")
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
		for after < len(text) && (text[after] == ' ' || text[after] == '\n' || text[after] == '\r' || text[after] == '	') {
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
		for body < len(text) && (text[body] == ' ' || text[body] == '\n' || text[body] == '\r' || text[body] == '	') {
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

func validateActiveSources(scope Scope, gate string, check func(asset assetDoc, source string) defect) (Result, error) {
	return validateActiveSourcesWithPath(scope, gate, func(asset assetDoc, _ string, source string) defect {
		return check(asset, source)
	})
}

func validateActiveSourcesWithPath(scope Scope, gate string, check func(asset assetDoc, path, source string) defect) (Result, error) {
	root := scope.Root
	assets, err := loadLibraryAssets(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	selected := make(map[string]bool, len(scope.Assets))
	for _, assetID := range scope.Assets {
		selected[assetID] = true
	}
	for _, asset := range assets {
		if !scope.IsFullCorpus() && !selected[asset.Asset.ID] {
			continue
		}
		if strings.HasPrefix(asset.Asset.ID, "supplemental.") {
			// Supplemental manifests are durable implementation inputs, but are
			// intentionally outside the active catalog population. They are not
			// runner failures and should not inflate the gate's skipped count.
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
func validateActiveSourceFiles(scope Scope, gate string, check func(asset assetDoc, source string) defect) (Result, error) {
	root := scope.Root
	assets, err := loadLibraryAssets(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
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
