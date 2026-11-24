// Package changeintent relates one explicit change statement to checked-in
// requirement declarations and source references. It never reads generated
// PRD artifacts or treats a source pointer as proof of behavior.
package changeintent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	maxIntentBytes           = 4096
	maxResults               = 20
	maxModules               = 64
	maxModuleBytes           = 1 << 20
	maxIndexBytes            = 64 << 10
	maxFiles                 = 240
	maxFileBytes             = 1 << 20
	maxTotalBytes            = 8 << 20
	maxRequirements          = 2000
	maxAnchorsPerRequirement = 20
	maxTitleBytes            = 200
	maxDescriptionBytes      = 1200
)

var (
	requirementID     = regexp.MustCompile(`[A-Z][A-Z0-9]+-[A-Z0-9]+-[0-9]{3}`)
	requirementLineID = regexp.MustCompile(`"id"\s*:\s*"([A-Z][A-Z0-9]+-[A-Z0-9]+-[0-9]{3})"`)
	tagPattern        = regexp.MustCompile(`\[REQ:([A-Z][A-Z0-9]+-[A-Z0-9]+-[0-9]{3})\]`)
)

type Request struct {
	Scenario string `json:"scenario"`
	Intent   string `json:"intent"`
	Limit    int    `json:"limit,omitempty"`
}

type SourceAnchor struct {
	Path        string `json:"path"`
	Line        int    `json:"line"`
	SHA256      string `json:"sha256"`
	Kind        string `json:"kind"`
	Attestation string `json:"attestation"`
}

type Result struct {
	Scenario      string         `json:"scenario"`
	RequirementID string         `json:"requirement_id"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Status        string         `json:"declared_status"`
	Relation      string         `json:"relation"`
	RelationBasis string         `json:"relation_basis"`
	Summary       string         `json:"snippet"`
	Score         int            `json:"lexical_overlap"`
	Requirement   SourceAnchor   `json:"requirement_source"`
	Anchors       []SourceAnchor `json:"implementation_anchors"`
}

type Response struct {
	Scenario     string   `json:"scenario"`
	Intent       string   `json:"intent"`
	Scope        string   `json:"scope"`
	Results      []Result `json:"results"`
	Limitations  []string `json:"limitations"`
	Degradations []string `json:"degradations,omitempty"`
}

type moduleFile struct {
	Requirements []requirementFile `json:"requirements"`
}

type requirementFile struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	anchor      SourceAnchor
}

type candidate struct {
	requirement requirementFile
	score       int
	anchors     []SourceAnchor
	relation    string
	basis       string
}

// Assess is intentionally restricted to one named scenario and its checked-in
// requirements plus api/cli/ui source. PRD.md, generated mirrors, and runtime
// state are not read.
func Assess(repoRoot string, request Request) (Response, error) {
	request.Scenario = strings.TrimSpace(request.Scenario)
	request.Intent = strings.TrimSpace(request.Intent)
	if request.Scenario != "business-health" {
		return Response{}, errors.New("scenario must be business-health for this bounded source contract")
	}
	if request.Intent == "" || len(request.Intent) > maxIntentBytes {
		return Response{}, fmt.Errorf("intent must contain 1..%d bytes", maxIntentBytes)
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 10
	}
	requestedLimitCapped := limit > maxResults
	if limit > maxResults {
		limit = maxResults
	}

	scenarioRoot := filepath.Join(repoRoot, "scenarios", request.Scenario)
	reqs, err := loadRequirements(scenarioRoot)
	if err != nil {
		return Response{}, err
	}
	sourceRefs, degradations := scanRequirementTags(scenarioRoot)
	intentTokens := tokens(request.Intent)
	ids := make(map[string]bool)
	for _, id := range exactRequirementIDs(request.Intent) {
		ids[id] = true
	}

	items := make([]candidate, 0, len(reqs))
	for _, r := range reqs {
		score := overlap(intentTokens, tokens(r.Title+" "+r.Description+" "+strings.Join(r.Tags, " ")))
		if score == 0 && !ids[r.ID] {
			continue
		}
		relation, basis := "unknown", "lexical_candidate_only"
		switch {
		case explicitContradiction(request.Intent, r.ID):
			relation, basis = "contradiction", "explicit_contradiction_phrase_with_requirement_id"
		case ids[r.ID]:
			relation, basis = "support", "explicit_requirement_id_reference_in_statement"
		}
		items = append(items, candidate{requirement: r, score: score, anchors: sourceRefs[r.ID], relation: relation, basis: basis})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].requirement.ID < items[j].requirement.ID
	})
	resultsCapped := len(items) > limit
	if len(items) > limit {
		items = items[:limit]
	}

	out := Response{
		Scenario: request.Scenario,
		Intent:   request.Intent,
		Scope:    "one scenario: checked-in requirements/*.json plus api/, cli/, ui/ source; no PRD or runtime state",
		Results:  make([]Result, 0, len(items)),
		Limitations: []string{
			"support means the statement explicitly names an exact requirement ID; it is not an endorsement and does not mean the requirement is implemented or accepted",
			"contradiction is emitted only when an exact requirement ID immediately follows an unnegated explicit contradiction/conflict phrase",
			"lexical matches are unknown candidates, not verified change rationale",
			"REQ source tags are traceability pointers; they do not prove behavior, execution, or passing tests",
			"malformed or ambiguous requirement declarations fail the request; incomplete bounded source scans are reported in degradations",
			"source scan excludes generated, dependency, and symlinked entries; no generated PRD, protected source session, test-run receipt, or live service state was consulted",
		},
		Degradations: degradations,
	}
	if requestedLimitCapped {
		out.Degradations = appendOnce(out.Degradations, "requested_limit_capped: maximum response result limit is 20")
	}
	if resultsCapped {
		out.Degradations = appendOnce(out.Degradations, "results_capped: response limited to requested top results")
	}
	for _, item := range items {
		r := item.requirement
		if len(r.Title) > maxTitleBytes || len(r.Description) > maxDescriptionBytes {
			out.Degradations = appendOnce(out.Degradations, "requirement_text_truncated: title/description output cap applied")
		}
		if len(item.anchors) > maxAnchorsPerRequirement {
			out.Degradations = appendOnce(out.Degradations, "source_anchors_capped: per-requirement source reference cap applied")
			item.anchors = item.anchors[:maxAnchorsPerRequirement]
		}
		out.Results = append(out.Results, Result{
			Scenario: request.Scenario, RequirementID: r.ID, Title: truncateUTF8(r.Title, maxTitleBytes), Description: truncateUTF8(r.Description, maxDescriptionBytes), Status: r.Status,
			Relation: item.relation, RelationBasis: item.basis, Summary: "Relation " + item.relation + " (" + item.basis + "); source references are not verification.", Score: item.score,
			Requirement: r.anchor, Anchors: item.anchors,
		})
	}
	return out, nil
}

func loadRequirements(scenarioRoot string) ([]requirementFile, error) {
	reqRoot := filepath.Join(scenarioRoot, "requirements")
	indexPath := filepath.Join(reqRoot, "index.json")
	indexInfo, err := os.Lstat(indexPath)
	if err != nil || !indexInfo.Mode().IsRegular() || indexInfo.Size() > maxIndexBytes {
		return nil, errors.New("requirements index is missing, non-regular, or exceeds 64 KiB bound")
	}
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("read checked-in requirements index: %w", err)
	}
	var index struct {
		Imports []string `json:"imports"`
	}
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		return nil, fmt.Errorf("parse requirements index: %w", err)
	}
	if len(index.Imports) > maxModules {
		return nil, errors.New("requirements index exceeds module-count bound")
	}
	var out []requirementFile
	var total int64
	seenIDs := make(map[string]bool)
	for _, imported := range index.Imports {
		clean := filepath.Clean(imported)
		parts := strings.Split(clean, string(filepath.Separator))
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.Ext(clean) != ".json" || len(parts) != 2 {
			return nil, fmt.Errorf("invalid requirements module path %q", imported)
		}
		dirInfo, err := os.Lstat(filepath.Join(reqRoot, parts[0]))
		if err != nil || !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("requirements module directory %q is missing or unsafe", parts[0])
		}
		path := filepath.Join(reqRoot, clean)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("stat requirements module %s: %w", clean, err)
		}
		if !info.Mode().IsRegular() || info.Size() > maxModuleBytes || total+info.Size() > maxTotalBytes {
			return nil, fmt.Errorf("requirements module %s violates file/byte bounds", clean)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read requirements module %s: %w", clean, err)
		}
		total += int64(len(b))
		var module moduleFile
		if err := json.Unmarshal(b, &module); err != nil {
			return nil, fmt.Errorf("parse requirements module %s: %w", clean, err)
		}
		h := sha256.Sum256(b)
		linesByID := requirementLines(b)
		for _, r := range module.Requirements {
			if strings.TrimSpace(r.ID) == "" {
				continue
			}
			if !isExactRequirementID(r.ID) {
				return nil, fmt.Errorf("requirements module %s contains an invalid requirement ID", clean)
			}
			if seenIDs[r.ID] {
				return nil, fmt.Errorf("requirements registry contains duplicate ID %s; relation identity is ambiguous", r.ID)
			}
			seenIDs[r.ID] = true
			line := linesByID[r.ID]
			if line == 0 {
				return nil, fmt.Errorf("requirements module %s parsed %s but could not bind its declaration line", clean, r.ID)
			}
			r.anchor = SourceAnchor{Path: filepath.ToSlash(filepath.Join("requirements", clean)), Line: line, SHA256: hex.EncodeToString(h[:]), Kind: "requirement_declaration", Attestation: "SOURCE_ONLY"}
			out = append(out, r)
			if len(out) > maxRequirements {
				return nil, errors.New("requirements registry exceeds 2000-record bound")
			}
		}
	}
	return out, nil
}

func requirementLines(data []byte) map[string]int {
	out := make(map[string]int)
	for lineNo, line := range strings.Split(string(data), "\n") {
		match := requirementLineID.FindStringSubmatch(line)
		if len(match) == 2 && out[match[1]] == 0 {
			out[match[1]] = lineNo + 1
		}
	}
	return out
}

func scanRequirementTags(scenarioRoot string) (map[string][]SourceAnchor, []string) {
	out := make(map[string][]SourceAnchor)
	var degradations []string
	files, total := 0, int64(0)
	for _, subtree := range []string{"api", "cli", "ui"} {
		root := filepath.Join(scenarioRoot, subtree)
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				degradations = append(degradations, "source_scan_partial: "+filepath.ToSlash(filepath.Join(subtree, "...")))
				return fs.SkipDir
			}
			if entry.IsDir() {
				if entry.Type()&fs.ModeSymlink != 0 || entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == "generated" {
					return fs.SkipDir
				}
				return nil
			}
			if entry.Type()&fs.ModeSymlink != 0 || !entry.Type().IsRegular() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				degradations = appendOnce(degradations, "source_scan_partial: source entry metadata could not be read")
				return nil
			}
			files++
			if files > maxFiles || info.Size() > maxFileBytes || total+info.Size() > maxTotalBytes {
				degradations = append(degradations, "source_scan_bounded: source reference scan reached configured file/byte bounds")
				return fs.SkipAll
			}
			data, err := os.ReadFile(path)
			if err != nil {
				degradations = appendOnce(degradations, "source_scan_partial: one or more source files could not be read")
				return nil
			}
			total += int64(len(data))
			h := sha256.Sum256(data)
			for lineNo, line := range strings.Split(string(data), "\n") {
				for _, match := range tagPattern.FindAllStringSubmatch(line, -1) {
					if len(out[match[1]]) >= maxAnchorsPerRequirement+1 {
						continue
					}
					kind := "implementation_reference"
					if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".test.ts") || strings.HasSuffix(path, ".test.tsx") || strings.HasSuffix(path, ".spec.ts") || strings.HasSuffix(path, ".spec.tsx") {
						kind = "test_reference"
					}
					out[match[1]] = append(out[match[1]], SourceAnchor{Path: filepath.ToSlash(strings.TrimPrefix(path, scenarioRoot+string(filepath.Separator))), Line: lineNo + 1, SHA256: hex.EncodeToString(h[:]), Kind: kind, Attestation: "REFERENCE_ONLY"})
				}
			}
			return nil
		})
	}
	for id := range out {
		sort.Slice(out[id], func(i, j int) bool {
			if out[id][i].Path != out[id][j].Path {
				return out[id][i].Path < out[id][j].Path
			}
			return out[id][i].Line < out[id][j].Line
		})
	}
	if len(degradations) == 0 {
		for _, anchors := range out {
			if len(anchors) > maxAnchorsPerRequirement {
				degradations = append(degradations, "source_anchors_capped: per-requirement source reference cap applied")
				break
			}
		}
	}
	sort.Strings(degradations)
	return out, degradations
}

func explicitContradiction(statement, id string) bool {
	for _, candidate := range exactRequirementIDs(statement) {
		if candidate != id {
			continue
		}
		for offset := 0; offset+len(id) <= len(statement); {
			relative := strings.Index(statement[offset:], id)
			if relative < 0 {
				break
			}
			start := offset + relative
			end := start + len(id)
			if isIDBoundary(statement, start-1) && isIDBoundary(statement, end) {
				prefix := statement[:start]
				phrase := regexp.MustCompile(`(?i)(?:^|[^[:alnum:]])(?:contradicts?|conflicts with)\s+$`)
				if loc := phrase.FindStringIndex(prefix); loc != nil {
					beforePhrase := prefix[:loc[0]]
					negation := regexp.MustCompile(`(?i)\b(?:not|never|no longer|cannot|can't|isn't|aren't|doesn't|don't|didn't|fails to)\b`)
					if !negation.MatchString(beforePhrase) {
						return true
					}
				}
			}
			offset = end
		}
	}
	return false
}

func exactRequirementIDs(s string) []string {
	var ids []string
	for _, bounds := range requirementID.FindAllStringIndex(s, -1) {
		if isIDBoundary(s, bounds[0]-1) && isIDBoundary(s, bounds[1]) {
			ids = append(ids, s[bounds[0]:bounds[1]])
		}
	}
	return ids
}

func isExactRequirementID(id string) bool {
	return len(requirementID.FindString(id)) == len(id) && requirementID.FindString(id) == id
}

func isIDBoundary(s string, index int) bool {
	if index < 0 || index >= len(s) {
		return true
	}
	b := s[index]
	return !(b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-')
}

func tokens(s string) map[string]bool {
	out := map[string]bool{}
	var b strings.Builder
	flush := func() {
		token := strings.ToLower(b.String())
		if len(token) > 2 {
			out[token] = true
		}
		b.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func overlap(a, b map[string]bool) int {
	n := 0
	for token := range a {
		if b[token] {
			n++
		}
	}
	return n
}

func truncateUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	for max > 0 && (s[max]&0xc0) == 0x80 {
		max--
	}
	return s[:max]
}

func appendOnce(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}
