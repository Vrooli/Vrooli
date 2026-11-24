package facts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const (
	maxInvariantFiles = 120
	maxInvariantBytes = 8 << 20
	maxInvariantHits  = 10
)

var (
	invariantScenarioPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	invariantTypePattern           = regexp.MustCompile(`(?i)\b(type\s+[A-Z]\w*\s+(struct|interface)|interface\s+[A-Z]\w*|type\s+[A-Z]\w*\s*=)`)
	invariantDBPattern             = regexp.MustCompile(`(?i)\b(check\s*\(|unique|not\s+null|foreign\s+key|gorm:"[^"]*(unique|not null))`)
	invariantRuntimePattern        = regexp.MustCompile(`(?i)^\s*(if|switch)\b.*(<|>|==|!=|<=|>=|nil|empty|len\()`)
	invariantTestPattern           = regexp.MustCompile(`(?i)(assert|require|expect|should|testify|testing\.T|\bt\.(errorf|fatalf|error|fatal|run)\b|\b(it|test)\s*\()`)
	invariantIdentPattern          = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9_]{1,}\b`)
	invariantCamelPattern          = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	invariantZeroComparisonPattern = regexp.MustCompile(`[<>]=?\s*0`)
)

// EnforcementLayers are deliberately UNKNOWN: lexical source candidates do
// not prove that a domain invariant is enforced by a type, database, test, or
// runtime guard. Test files are surfaced only as candidate corroboration.
type EnforcementLayers struct {
	Type     string `json:"type"`
	Database string `json:"database"`
	Test     string `json:"test"`
	Runtime  string `json:"runtime"`
}

type InvariantCandidate struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Text         string            `json:"text"`
	Scenario     string            `json:"scenario"`
	Path         string            `json:"path"`
	Line         int               `json:"line"`
	SHA256       string            `json:"sha256"`
	EvidenceKind string            `json:"evidence_kind"`
	Score        float64           `json:"score"`
	Attestation  string            `json:"attestation"`
	Enforcement  EnforcementLayers `json:"enforcement"`
}

type InvariantCandidateResponse struct {
	Query        string                 `json:"query"`
	Scenario     string                 `json:"scenario"`
	Scope        string                 `json:"scope"`
	Results      []InvariantCandidate   `json:"results"`
	Degradations []InvariantDegradation `json:"degradations"`
}

type InvariantDegradation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type invariantFile struct {
	path string
	body []byte
}

type InvalidInvariantCandidateScopeError string

func (e InvalidInvariantCandidateScopeError) Error() string { return string(e) }

// SearchInvariantCandidates returns bounded lexical source locations. Every
// hit is a candidate, never an assertion that the invariant is enforced.
func (s *Service) SearchInvariantCandidates(_ context.Context, query, scenario, relativePath string, limit int) (*InvariantCandidateResponse, error) {
	query, scenario, relativePath = strings.TrimSpace(query), strings.TrimSpace(scenario), filepath.Clean(strings.TrimSpace(relativePath))
	if query == "" || !invariantScenarioPattern.MatchString(scenario) || relativePath == "." || filepath.IsAbs(relativePath) || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return nil, InvalidInvariantCandidateScopeError("query, scenario slug, and relative source path are required")
	}
	if limit <= 0 || limit > maxInvariantHits {
		limit = maxInvariantHits
	}
	target, err := resolveTarget(&factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario})
	if err != nil {
		return nil, err
	}
	base, err := filepath.EvalSymlinks(target.GetRootPath())
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(filepath.Join(base, relativePath))
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(base, root)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, InvalidInvariantCandidateScopeError("source path resolves outside selected scenario")
	}
	files, bounded, err := collectInvariantFiles(root)
	if err != nil {
		return nil, err
	}
	resp := &InvariantCandidateResponse{Query: query, Scenario: scenario, Scope: fmt.Sprintf("scenario:%s/%s; lexical candidate locations only; max %d files, %d MiB; no enforcement inference", scenario, filepath.ToSlash(relativePath), maxInvariantFiles, maxInvariantBytes>>20), Results: []InvariantCandidate{}, Degradations: []InvariantDegradation{}}
	if bounded {
		resp.Degradations = append(resp.Degradations, InvariantDegradation{Code: "source_scan_bounded", Message: "Source inventory reached its file or byte cap; candidates may be incomplete."})
	}
	terms := invariantTerms(query)
	rankedHits := map[string]InvariantCandidate{}
	for _, file := range files {
		lines := strings.Split(string(file.body), "\n")
		digest := sha256.Sum256(file.body)
		for i, line := range lines {
			kind := invariantEvidenceKind(file.path, line)
			if kind == "" {
				continue
			}
			start, end := i-12, i+13
			if start < 0 {
				start = 0
			}
			if end > len(lines) {
				end = len(lines)
			}
			contextText := strings.Join(lines[start:end], " ")
			score := invariantMatchScore(terms, contextText)
			if score == 0 {
				continue
			}
			relPath, _ := filepath.Rel(root, file.path)
			relPath = filepath.ToSlash(filepath.Join(relativePath, relPath))
			kindID := strings.NewReplacer("-", "_", " ", "_").Replace(strings.TrimSuffix(kind, " candidate"))
			id := fmt.Sprintf("code-facts:invariant-candidate:%s:%s:%s", scenario, kindID, relPath)
			hit := InvariantCandidate{ID: id, Title: fmt.Sprintf("%s source candidate · %s:%d", kind, relPath, i+1), Text: fmt.Sprintf("Source pattern candidate (%s). This location is not proof of invariant enforcement.", kind), Scenario: scenario, Path: relPath, Line: i + 1, SHA256: hex.EncodeToString(digest[:]), EvidenceKind: kind, Score: score, Attestation: "CANDIDATE_ONLY", Enforcement: EnforcementLayers{Type: "UNKNOWN", Database: "UNKNOWN", Test: "UNKNOWN", Runtime: "UNKNOWN"}}
			if old, exists := rankedHits[id]; !exists || hit.Score > old.Score {
				rankedHits[id] = hit
			}
		}
	}
	candidates := make([]InvariantCandidate, 0, len(rankedHits))
	for _, hit := range rankedHits {
		candidates = append(candidates, hit)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		if candidates[i].Path != candidates[j].Path {
			return candidates[i].Path < candidates[j].Path
		}
		return candidates[i].Line < candidates[j].Line
	})
	for _, hit := range candidates {
		if len(resp.Results) >= limit {
			break
		}
		resp.Results = append(resp.Results, hit)
	}
	if len(resp.Results) == 0 {
		resp.Degradations = append(resp.Degradations, InvariantDegradation{Code: "no_candidate_match", Message: "No source location matched the query and supported candidate patterns in the bounded scope."})
	}
	return resp, nil
}

func collectInvariantFiles(root string) ([]invariantFile, bool, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, false, err
	}
	if !info.IsDir() {
		return nil, false, fmt.Errorf("source scope must be a directory")
	}
	var files []invariantFile
	var size int64
	bounded := false
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == "build") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".sql" && ext != ".proto" && ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		if len(files) >= maxInvariantFiles {
			bounded = true
			return fs.SkipAll
		}
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		if fi.Size() > 1<<20 || size+fi.Size() > maxInvariantBytes {
			bounded = true
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		size += int64(len(body))
		files = append(files, invariantFile{path: path, body: body})
		return nil
	})
	if err != nil {
		return nil, bounded, err
	}
	return files, bounded, nil
}

func invariantEvidenceKind(path, line string) string {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, "_test.") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		if invariantTestPattern.MatchString(line) {
			return "test-source candidate"
		}
		return ""
	}
	if invariantDBPattern.MatchString(line) {
		return "database-constraint candidate"
	}
	if invariantTypePattern.MatchString(line) {
		return "type-shape candidate"
	}
	if invariantRuntimePattern.MatchString(line) {
		return "runtime-guard candidate"
	}
	return ""
}

func invariantTerms(query string) []string { return invariantTokens(query) }
func invariantMatchScore(terms []string, text string) float64 {
	if len(terms) == 0 {
		return 0
	}
	fields := map[string]bool{}
	for _, term := range invariantTokens(text) {
		fields[term] = true
	}
	match := 0
	for _, term := range terms {
		if fields[term] {
			match++
		}
	}
	return float64(match) / float64(len(terms))
}
func invariantTokens(value string) []string {
	value = invariantCamelPattern.ReplaceAllString(value, "$1 $2")
	seen := map[string]bool{}
	var out []string
	for _, token := range invariantIdentPattern.FindAllString(strings.ToLower(value), -1) {
		if len(token) < 2 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	if strings.Contains(value, "<") {
		out = append(out, "less")
	}
	if strings.Contains(value, ">") {
		out = append(out, "greater")
	}
	if invariantZeroComparisonPattern.MatchString(value) {
		out = append(out, "zero")
	}
	return out
}
