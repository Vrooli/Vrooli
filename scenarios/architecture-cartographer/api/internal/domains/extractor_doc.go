package domains

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DomainsDocPath is the scenario-relative location of the canonical
// structured domain inventory.
const DomainsDocPath = "docs/concepts/DOMAINS.md"

// archetypeVocabulary is the fixed set of primary archetypes the
// structured contract recognizes. The parser takes the first token of a
// compound cell (e.g., "service / orchestration" -> "service") and matches
// it case-insensitively. Unknown archetypes are preserved verbatim so
// convergence reporting (Phase 3) can flag them rather than the parser
// silently dropping intent.
var archetypeVocabulary = map[string]string{
	"service":          "service",
	"reporting":        "reporting",
	"validation":       "validation",
	"mutation":         "mutation",
	"orchestration":    "orchestration",
	"infrastructure":   "infrastructure",
	"composition-root": "composition-root",
}

// DomainsDocExtractor parses the structured Domain Inventory table in
// docs/concepts/DOMAINS.md. It is the interim top rung of the ladder until
// an API manifest exists.
//
// The parser is header-driven: it locates columns by header text rather
// than position, so extra human-facing columns (Purpose, Owns Data,
// Surfaces, Requirements) are ignored without breaking extraction. The
// required columns are "Domain", an archetype column ("Primary Archetype"
// or "Archetype"), and a source-paths column ("Source Paths"...). A
// "Glossary" column is optional.
type DomainsDocExtractor struct{}

// NewDomainsDocExtractor returns the production DOMAINS.md extractor.
func NewDomainsDocExtractor() *DomainsDocExtractor { return &DomainsDocExtractor{} }

// Source identifies this rung.
func (*DomainsDocExtractor) Source() Source { return SourceDomainsDoc }

// Extract reads and parses the scenario's DOMAINS.md. A missing file
// returns an empty extraction (absence is not failure); a present but
// malformed table returns an error.
func (e *DomainsDocExtractor) Extract(_ context.Context, scenarioDir string) (Extraction, error) {
	path := filepath.Join(scenarioDir, DomainsDocPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Extraction{Source: SourceDomainsDoc}, nil
		}
		return Extraction{}, fmt.Errorf("read %s: %w", DomainsDocPath, err)
	}
	return e.parse(string(data))
}

// parse is the pure parsing core, split out so tests can drive it with
// golden markdown without touching the filesystem.
func (e *DomainsDocExtractor) parse(content string) (Extraction, error) {
	out := Extraction{Source: SourceDomainsDoc}

	domains, warns, err := parseDomainInventory(content)
	if err != nil {
		return Extraction{}, err
	}
	out.Domains = domains
	out.Warnings = warns
	out.SharedSubstrate, out.NonDomains = parseNonDomains(content)
	return out, nil
}

// parseDomainInventory finds the "## Domain Inventory" section and parses
// its markdown table by header name.
//
// Returns the parsed domains plus a list of non-fatal warnings (rows
// skipped for structural reasons). The caller propagates warnings up to
// the audit pipeline so the operator sees what was silently dropped
// rather than guessing why a domain count looks off.
func parseDomainInventory(content string) ([]ExtractedDomain, []ExtractionWarning, error) {
	section, ok := sectionBody(content, "Domain Inventory")
	if !ok {
		return nil, nil, fmt.Errorf("%s: missing '## Domain Inventory' section", DomainsDocPath)
	}

	var headerCells []string
	headerSeen := false
	expectedCols := 0
	var domains []ExtractedDomain
	var warns []ExtractionWarning
	lineNo := 0
	var (
		idxDomain    = -1
		idxArchetype = -1
		idxPaths     = -1
		idxGlossary  = -1
	)

	sc := bufio.NewScanner(strings.NewReader(section))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			if headerSeen {
				break // table ended
			}
			continue
		}
		cells := splitTableRow(line)
		if isSeparatorRow(cells) {
			continue
		}
		if !headerSeen {
			headerCells = cells
			expectedCols = len(headerCells)
			idxDomain = headerIndex(headerCells, func(h string) bool { return h == "domain" })
			idxArchetype = headerIndex(headerCells, func(h string) bool {
				return strings.Contains(h, "archetype")
			})
			idxPaths = headerIndex(headerCells, func(h string) bool {
				return strings.HasPrefix(h, "source paths")
			})
			idxGlossary = headerIndex(headerCells, func(h string) bool { return h == "glossary" })
			if idxDomain < 0 || idxPaths < 0 {
				return nil, nil, fmt.Errorf("%s: Domain Inventory table must have 'Domain' and 'Source Paths' columns", DomainsDocPath)
			}
			headerSeen = true
			continue
		}

		if expectedCols > 0 && len(cells) != expectedCols {
			warns = append(warns, ExtractionWarning{
				Kind:    "domains_doc.row_shape",
				Path:    DomainsDocPath,
				Line:    lineNo,
				Summary: fmt.Sprintf("row has %d columns, header has %d — skipped", len(cells), expectedCols),
			})
			continue
		}
		name := strings.TrimSpace(cell(cells, idxDomain))
		if name == "" {
			warns = append(warns, ExtractionWarning{
				Kind:    "domains_doc.empty_name",
				Path:    DomainsDocPath,
				Line:    lineNo,
				Summary: "row has empty Domain cell — skipped",
			})
			continue
		}
		d := ExtractedDomain{
			Name:      name,
			Paths:     parsePathList(cell(cells, idxPaths)),
			Archetype: normalizeArchetype(cell(cells, idxArchetype)),
			Glossary:  parseTermList(cell(cells, idxGlossary)),
		}
		if len(d.Paths) == 0 {
			warns = append(warns, ExtractionWarning{
				Kind:    "domains_doc.no_paths",
				Path:    DomainsDocPath,
				Line:    lineNo,
				Summary: fmt.Sprintf("domain %q declares no source paths — skipped", name),
			})
			continue
		}
		domains = append(domains, d)
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("%s: scan Domain Inventory: %w", DomainsDocPath, err)
	}
	if !headerSeen {
		return nil, nil, fmt.Errorf("%s: Domain Inventory section has no table", DomainsDocPath)
	}

	sort.Slice(domains, func(i, j int) bool { return domains[i].Name < domains[j].Name })
	return domains, warns, nil
}

// parseNonDomains finds the "## Non-Domains" section and extracts the
// backtick-wrapped paths from its bulleted list. These become the derived
// shared-substrate set; the path's last segment becomes a non-domain name.
func parseNonDomains(content string) (shared, names []string) {
	section, ok := sectionBody(content, "Non-Domains")
	if !ok {
		return nil, nil
	}
	sc := bufio.NewScanner(strings.NewReader(section))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
			continue
		}
		path := firstBacktickToken(line)
		if path == "" {
			continue
		}
		path = NormalizePath(path)
		if path == "" {
			continue
		}
		shared = append(shared, path)
		names = append(names, lastPathSegment(path))
	}
	sort.Strings(shared)
	sort.Strings(names)
	return shared, names
}

// sectionBody returns the lines under a "## <title>" heading, up to the
// next heading of the same or higher level (## or #).
func sectionBody(content, title string) (string, bool) {
	lines := strings.Split(content, "\n")
	want := "## " + title
	start := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.EqualFold(t, want) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return "", false
	}
	var b strings.Builder
	for i := start; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "## ") || (strings.HasPrefix(t, "# ") && !strings.HasPrefix(t, "## ")) {
			break
		}
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	return b.String(), true
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func isSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if strings.Trim(c, "-: ") != "" {
			return false
		}
	}
	return true
}

func headerIndex(headers []string, match func(string) bool) int {
	for i, h := range headers {
		if match(strings.ToLower(strings.TrimSpace(h))) {
			return i
		}
	}
	return -1
}

func cell(cells []string, idx int) string {
	if idx < 0 || idx >= len(cells) {
		return ""
	}
	return cells[idx]
}

// parsePathList splits a Source Paths cell into normalized path tokens.
// Cells separate globs with commas and wrap each in backticks.
func parsePathList(cellVal string) []string {
	return splitNormalized(cellVal, NormalizePath)
}

// parseTermList splits a Glossary cell into trimmed, backtick-stripped
// terms.
func parseTermList(cellVal string) []string {
	return splitNormalized(cellVal, func(s string) string {
		return strings.Trim(strings.TrimSpace(s), "`")
	})
}

func splitNormalized(cellVal string, norm func(string) string) []string {
	cellVal = strings.TrimSpace(cellVal)
	if cellVal == "" || cellVal == "—" || cellVal == "-" {
		return nil
	}
	// Accept comma and <br> separators (DOMAINS.md tables sometimes wrap).
	cellVal = strings.ReplaceAll(cellVal, "<br>", ",")
	cellVal = strings.ReplaceAll(cellVal, "<br/>", ",")
	var out []string
	for _, raw := range strings.Split(cellVal, ",") {
		v := norm(raw)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func normalizeArchetype(cellVal string) string {
	cellVal = strings.TrimSpace(cellVal)
	if cellVal == "" {
		return ""
	}
	// Take the first token of a compound cell ("service / orchestration").
	first := cellVal
	if i := strings.IndexAny(cellVal, "/,"); i >= 0 {
		first = cellVal[:i]
	}
	first = strings.ToLower(strings.TrimSpace(first))
	if canonical, ok := archetypeVocabulary[first]; ok {
		return canonical
	}
	return first
}

func firstBacktickToken(line string) string {
	start := strings.Index(line, "`")
	if start < 0 {
		return ""
	}
	rest := line[start+1:]
	end := strings.Index(rest, "`")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func lastPathSegment(path string) string {
	path = strings.TrimSuffix(path, "/")
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
