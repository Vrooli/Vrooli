// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/librarywalk"
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
	// Typed attribution fields are populated at the runner boundary. AssetID
	// remains the compatibility identity consumed by older callers.
	CatalogID string
	LibraryID string
	Scope     FindingScope
	Blocking  bool
	Owner     string
	Severity  FindingSeverity
}

type FindingScope string

const (
	FindingScopeAsset  FindingScope = "asset"
	FindingScopeCorpus FindingScope = "corpus"
)

type FindingSeverity string

const (
	FindingSeverityBlocking FindingSeverity = "blocking"
	FindingSeverityWarning  FindingSeverity = "warning"
	FindingSeverityInfo     FindingSeverity = "info"
)

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
	for index := range result.InformationalFindings {
		result.InformationalFindings[index] = normalizeFinding(result.InformationalFindings[index])
	}
	for index := range result.RunnerError {
		result.RunnerError[index] = normalizeFinding(result.RunnerError[index])
	}
	ids := catalogAssetIDs(root)
	normalized := make([]Finding, 0, len(result.Findings))
	for _, finding := range result.Findings {
		finding = normalizeFinding(finding)
		// Corpus-level observations have no catalog identity. Older runners may
		// still stamp a __corpus__ compatibility value, but it must never cross
		// the typed finding boundary as an asset identity.
		if finding.AssetID != "" && (strings.HasPrefix(finding.AssetID, "workbench.") || strings.HasPrefix(finding.AssetID, "supplemental.") || strings.HasPrefix(finding.AssetID, "__corpus__")) {
			finding.AssetID = ""
			finding.CatalogID = ""
			finding.Scope = FindingScopeCorpus
			normalized = append(normalized, finding)
			continue
		}
		// Some runners intentionally leave AssetID blank because they only
		// have the source location at the point they emit the finding. Resolve
		// that location before declaring the observation unowned.
		if finding.AssetID == "" {
			if resolved := implementationName(filepath.Join(root, finding.File)); resolved != "" && ids[resolved] {
				finding.AssetID = resolved
				finding.Scope = FindingScopeAsset
				finding.CatalogID = resolved
				normalized = append(normalized, finding)
				continue
			}
			result.RunnerError = append(result.RunnerError, finding)
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
	paths, _ := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
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

func normalizeFinding(finding Finding) Finding {
	if finding.Owner == "" {
		finding.Owner = filepath.ToSlash(finding.File)
	}
	if strings.HasPrefix(finding.AssetID, "__corpus__") || strings.HasPrefix(finding.AssetID, "workbench.") || strings.HasPrefix(finding.AssetID, "supplemental.") || finding.AssetID == "" {
		finding.Scope = FindingScopeCorpus
		finding.CatalogID = ""
		if finding.Severity == "" {
			finding.Severity = FindingSeverityWarning
		}
		return finding
	}
	finding.Scope = FindingScopeAsset
	if finding.CatalogID == "" {
		finding.CatalogID = finding.AssetID
	}
	if strings.HasPrefix(finding.AssetID, "react-component-library:") && finding.LibraryID == "" {
		finding.LibraryID = finding.AssetID
	}
	if finding.Severity == "" {
		finding.Severity = FindingSeverityBlocking
	}
	return finding
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
	cssVarRefGateRE  = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
	cssVarDeclGateRE = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
)

// ValidateVersionLiveness protects the published version boundary. Every
// library import must resolve to a surviving version entry, and released
// source must not retain a relative edge into another version directory.
// Package compilation catches the former late; this gate makes the contract
// visible to catalog evidence and calibration before a consumer build.
func versionEntryExists(libraryRoot, name, version string) bool {
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		versionsRoot := filepath.Join(libraryRoot, kind, name, "versions")
		candidates := []string{version}
		if major := strings.Split(version, ".")[0]; major != version || (len(version) > 0 && !strings.Contains(version, ".")) {
			if entries, err := os.ReadDir(versionsRoot); err == nil {
				for _, candidate := range entries {
					if candidate.IsDir() && strings.Split(candidate.Name(), ".")[0] == major {
						candidates = append(candidates, candidate.Name())
					}
				}
			}
		}
		for _, candidate := range candidates {
			dir := filepath.Join(versionsRoot, candidate)
			for _, entry := range []string{name, pascalCaseGate(name)} {
				for _, ext := range []string{".ts", ".tsx"} {
					if _, err := os.Stat(filepath.Join(dir, entry+ext)); err == nil {
						return true
					}
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
func librarySourceFiles(root string, scope Scope) []string {
	if scope.Set.Files != nil {
		files := make([]string, 0, len(scope.Set.Files))
		for _, path := range scope.Set.Files {
			if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") {
				files = append(files, path)
			}
		}
		sort.Strings(files)
		return files
	}
	var files []string
	_ = librarywalk.Walk(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}
