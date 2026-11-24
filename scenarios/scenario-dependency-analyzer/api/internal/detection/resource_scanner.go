package detection

import (
	"context"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// resource_scanner.go - Resource dependency detection
//
// This file contains the logic for scanning scenario files to detect
// resource dependencies through CLI commands and heuristics.

// resourceScanner handles resource dependency detection
type resourceScanner struct {
	catalog *catalogManager
	// Immutable patterns are shared across scans. Cache only canonical catalog
	// names, never source text; catalog removal still stops matching that name.
	heuristicPatterns sync.Map
}

// newResourceScanner creates a scanner for detecting resource dependencies
func newResourceScanner(catalog *catalogManager) *resourceScanner {
	return &resourceScanner{
		catalog: catalog,
	}
}

// scan walks the scenario directory and detects all resource dependencies
func (s *resourceScanner) scan(scenarioPath, scenarioName string, cfg *types.Manifest) ([]types.ScenarioDependency, error) {
	return s.scanContext(context.Background(), scenarioPath, scenarioName, cfg)
}

func (s *resourceScanner) scanContext(ctx context.Context, scenarioPath, scenarioName string, cfg *types.Manifest) ([]types.ScenarioDependency, error) {
	results := map[string]types.ScenarioDependency{}
	// Catalogs are immutable during one scan. Snapshot once, rather than copying
	// the same catalog and taking its read lock for every source file.
	resourceCatalog := s.catalog.getResourceCatalog()

	err := filepath.WalkDir(scenarioPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			return relErr
		}
		if shouldIgnoreDetectionFile(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories that should be ignored
		if entry.IsDir() {
			if path != scenarioPath && shouldSkipDirectoryEntry(entry) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		// Only scan relevant file types
		ext := strings.ToLower(filepath.Ext(path))
		if !contains(resourceDetectionExtensions, ext) {
			return nil
		}

		// Read file content
		content, err := readDetectionSource(ctx, path, entry) // #nosec G304,G122 -- path comes from WalkDir under scenarioPath and symlink entries are skipped.
		if err != nil {
			return err
		}

		// Scan this file for resource dependencies
		s.scanFileWithCatalog(string(content), rel, scenarioName, results, resourceCatalog)

		return nil
	})

	// Convert map to sorted slice
	deps := make([]types.ScenarioDependency, 0, len(results))
	for _, dep := range results {
		deps = append(deps, dep)
	}
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].DependencyName < deps[j].DependencyName
	})

	return deps, err
}

// scanFile scans a single file for resource references
func (s *resourceScanner) scanFile(content, relPath, scenarioName string, results map[string]types.ScenarioDependency) {
	s.scanFileWithCatalog(content, relPath, scenarioName, results, s.catalog.getResourceCatalog())
}

func (s *resourceScanner) scanFileWithCatalog(content, relPath, scenarioName string, results map[string]types.ScenarioDependency, resourceCatalog map[string]struct{}) {
	// Detect explicit resource CLI commands (e.g., "resource-postgres")
	s.detectResourceCLICommands(content, relPath, scenarioName, results, resourceCatalog)

	// Detect resources via heuristics (e.g., connection strings, env vars)
	s.detectResourceHeuristicsWithCatalog(content, relPath, scenarioName, results, resourceCatalog)
}

// detectResourceCLICommands finds explicit resource-* CLI command usage
func (s *resourceScanner) detectResourceCLICommands(content, relPath, scenarioName string, results map[string]types.ScenarioDependency, resourceCatalog map[string]struct{}) {
	// Only look for CLI commands in allowed directories
	if !isAllowedResourceCLIPath(relPath) {
		return
	}

	matches := resourceCommandPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			resourceName := normalizeName(match[1])
			s.recordDetection(
				results,
				resourceCatalog,
				scenarioName,
				resourceName,
				"resource_cli",
				"resource-cli",
				relPath,
				resourceName,
			)
		}
	}
}

// detectResourceHeuristics is an interim resource signal retained until resource
// usage is delegated to code-facts/go-code-graph in the
// scenario-dependency-analyzer-code-evidence-via-ast-facts follow-up.
func (s *resourceScanner) detectResourceHeuristics(content, relPath, scenarioName string, results map[string]types.ScenarioDependency) {
	s.detectResourceHeuristicsWithCatalog(content, relPath, scenarioName, results, s.catalog.getResourceCatalog())
}

func (s *resourceScanner) detectResourceHeuristicsWithCatalog(content, relPath, scenarioName string, results map[string]types.ScenarioDependency, resourceCatalog map[string]struct{}) {
	// The resource literals are ASCII. Folding each source rune to its
	// equivalent ASCII letter (when one exists) preserves regexp (?i) matches
	// such as long-s while allowing the common path to use literal scans.
	foldedContent := foldForASCIIMatching(content)
	urls := httpURLSpans(foldedContent)
	for resourceName := range resourceCatalog {
		if !isASCII(resourceName) {
			for _, pattern := range s.patternsForResource(resourceName) {
				if pattern.MatchString(content) {
					s.recordDetection(results, resourceCatalog, scenarioName, resourceName, "heuristic", pattern.String(), relPath, "declared-resource")
					break
				}
			}
			continue
		}
		name := normalizeName(resourceName)
		envName := strings.ReplaceAll(name, "-", "_")
		if !strings.Contains(foldedContent, name) && !strings.Contains(foldedContent, envName) {
			continue
		}
		if pattern := asciiResourceReferencePattern(content, foldedContent, name, urls); pattern >= 0 {
			patterns := s.patternsForResource(resourceName)
			s.recordDetection(results, resourceCatalog, scenarioName, resourceName, "heuristic", patterns[pattern].String(), relPath, "declared-resource")
		}
	}
}

func foldForASCIIMatching(content string) string {
	var folded strings.Builder
	folded.Grow(len(content))
	for _, r := range content {
		if r < 0x80 {
			folded.WriteRune(unicode.ToLower(r))
			continue
		}
		matchedASCII := false
		for candidate := unicode.SimpleFold(r); candidate != r; candidate = unicode.SimpleFold(candidate) {
			if candidate < 0x80 && ((candidate >= 'A' && candidate <= 'Z') || (candidate >= 'a' && candidate <= 'z')) {
				folded.WriteRune(unicode.ToLower(candidate))
				matchedASCII = true
				break
			}
		}
		if !matchedASCII {
			folded.WriteRune(unicode.ToLower(r))
		}
	}
	return folded.String()
}

func asciiResourceReferencePattern(content, lowerContent, name string, urls [][2]int) int {
	if containsBoundaryAfter(content, "resource-"+name) {
		return 0
	}
	if containsBoundaryBefore(lowerContent, name+"://") {
		return 1
	}
	envName := strings.ReplaceAll(name, "-", "_")
	if containsEnvReference(content, strings.ToUpper(envName)) {
		return 2
	}
	for _, span := range urls {
		if containsBoundarySubstring(lowerContent, span[0], span[1], name) {
			return 3
		}
	}
	return -1
}

func containsBoundaryBefore(text, needle string) bool {
	for offset := 0; offset <= len(text)-len(needle); {
		index := strings.Index(text[offset:], needle)
		if index < 0 {
			return false
		}
		start := offset + index
		if asciiWordBoundary(text, start) {
			return true
		}
		offset = start + 1
	}
	return false
}

func containsBoundaryAfter(text, needle string) bool {
	for offset := 0; offset <= len(text)-len(needle); {
		index := strings.Index(text[offset:], needle)
		if index < 0 {
			return false
		}
		end := offset + index + len(needle)
		if asciiWordBoundary(text, end) {
			return true
		}
		offset = end - len(needle) + 1
	}
	return false
}

func containsBoundarySubstring(text string, start, end int, needle string) bool {
	for offset := start; offset <= end-len(needle); {
		index := strings.Index(text[offset:end], needle)
		if index < 0 {
			return false
		}
		matchStart := offset + index
		matchEnd := matchStart + len(needle)
		if asciiWordBoundary(text, matchStart) && asciiWordBoundary(text, matchEnd) {
			return true
		}
		offset = matchStart + 1
	}
	return false
}

func asciiWordBoundary(text string, position int) bool {
	beforeWord := position > 0 && isASCIIWord(text[position-1])
	afterWord := position < len(text) && isASCIIWord(text[position])
	return beforeWord != afterWord
}

func isASCIIWord(value byte) bool {
	return value == '_' || (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9')
}

func containsEnvReference(content, upperName string) bool {
	needle := upperName + "_"
	for offset := 0; offset <= len(content)-len(needle); {
		index := strings.Index(content[offset:], needle)
		if index < 0 {
			return false
		}
		start := offset + index
		if asciiWordBoundary(content, start) {
			for _, suffix := range []string{"HOST", "URL", "API", "BASE_URL", "ENDPOINT", "WEBHOOK", "KEY"} {
				end := start + len(needle) + len(suffix)
				if end <= len(content) && content[start+len(needle):end] == suffix && asciiWordBoundary(content, end) {
					return true
				}
			}
		}
		offset = start + 1
	}
	return false
}

func httpURLSpans(lowerContent string) [][2]int {
	var spans [][2]int
	for offset := 0; offset < len(lowerContent); {
		rest := lowerContent[offset:]
		httpIndex := strings.Index(rest, "http://")
		httpsIndex := strings.Index(rest, "https://")
		if httpIndex < 0 && httpsIndex < 0 {
			break
		}
		start, schemeLen := httpIndex, len("http://")
		if start < 0 || (httpsIndex >= 0 && httpsIndex < start) {
			start, schemeLen = httpsIndex, len("https://")
		}
		urlStart := offset + start + schemeLen
		urlEnd := urlStart
		for urlEnd < len(lowerContent) && !isURLDelimiter(lowerContent[urlEnd]) {
			urlEnd++
		}
		spans = append(spans, [2]int{urlStart, urlEnd})
		if urlEnd == len(lowerContent) {
			break
		}
		offset = urlEnd + 1
	}
	return spans
}

func isURLDelimiter(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r' || value == '\f' || value == '\'' || value == '"'
}

func isASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] >= 128 {
			return false
		}
	}
	return true
}

func (s *resourceScanner) patternsForResource(name string) []*regexp.Regexp {
	name = normalizeName(name)
	if patterns, ok := s.heuristicPatterns.Load(name); ok {
		return patterns.([]*regexp.Regexp)
	}
	patterns := resourceHeuristicPatterns(name)
	actual, loaded := s.heuristicPatterns.LoadOrStore(name, patterns)
	if loaded {
		return actual.([]*regexp.Regexp)
	}
	return patterns
}

func resourceHeuristicPatterns(resourceName string) []*regexp.Regexp {
	name := regexp.QuoteMeta(normalizeName(resourceName))
	// Resource slugs use hyphens while exported environment variables use
	// underscores (for example, `kale-tts` exports `KALE_TTS_URL`). Match the
	// conventional environment spelling without changing URL or CLI matching.
	envName := strings.NewReplacer("-", "_").Replace(normalizeName(resourceName))
	upper := regexp.QuoteMeta(strings.ToUpper(envName))
	return []*regexp.Regexp{
		regexp.MustCompile(`resource-` + name + `\b`),
		regexp.MustCompile(`(?i)\b` + name + `://`),
		regexp.MustCompile(`\b` + upper + `_(?:HOST|URL|API|BASE_URL|ENDPOINT|WEBHOOK|KEY)\b`),
		regexp.MustCompile(`(?i)https?://[^"'\s]*\b` + name + `\b`),
	}
}

// recordDetection records a resource dependency detection, merging with existing entries
func (s *resourceScanner) recordDetection(
	results map[string]types.ScenarioDependency,
	resourceCatalog map[string]struct{},
	scenarioName, name, method, pattern, file, resourceType string,
) {
	canonical := normalizeName(name)

	// Skip if empty or not a known resource
	if canonical == "" {
		return
	}
	if len(resourceCatalog) > 0 {
		if _, ok := resourceCatalog[canonical]; !ok {
			return
		}
	}

	entry := ensureResourceEntry(results, scenarioName, canonical, "Detected via static analysis", method, map[string]interface{}{"source": "detected"})

	// Set resource type
	entry.Configuration["resource_type"] = resourceType

	entry = appendMatch(entry, pattern, method, file)

	results[canonical] = entry
}

func ensureResourceEntry(
	results map[string]types.ScenarioDependency,
	scenarioName, canonical, purpose, method string,
	baseConfig map[string]interface{},
) types.ScenarioDependency {
	if entry, ok := results[canonical]; ok {
		if entry.Configuration == nil {
			entry.Configuration = map[string]interface{}{}
		}
		return entry
	}

	config := map[string]interface{}{}
	for k, v := range baseConfig {
		config[k] = v
	}

	return types.ScenarioDependency{
		ID:             uuid.New().String(),
		ScenarioName:   scenarioName,
		DependencyType: "resource",
		DependencyName: canonical,
		Required:       true,
		Purpose:        purpose,
		AccessMethod:   method,
		Configuration:  config,
		DiscoveredAt:   time.Now(),
		LastVerified:   time.Now(),
	}
}

func appendMatch(entry types.ScenarioDependency, pattern, method, file string) types.ScenarioDependency {
	if entry.Configuration == nil {
		entry.Configuration = map[string]interface{}{}
	}

	matches := existingMatches(entry.Configuration["matches"])
	entry.Configuration["matches"] = append(matches, map[string]interface{}{
		"pattern": pattern,
		"method":  method,
		"file":    file,
	})
	return entry
}

func existingMatches(raw interface{}) []map[string]interface{} {
	if cast, ok := raw.([]map[string]interface{}); ok {
		return cast
	}
	return []map[string]interface{}{}
}
