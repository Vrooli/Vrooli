package domains

import (
	"bufio"
	"context"
	"encoding/json"
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
	contract := loadDomainInventoryContract(scenarioDir)
	return e.parseWithContract(string(data), contract)
}

// parse is the pure parsing core, split out so tests can drive it with
// golden markdown without touching the filesystem.
func (e *DomainsDocExtractor) parse(content string) (Extraction, error) {
	return e.parseWithContract(content, domainInventoryContract{})
}

func (e *DomainsDocExtractor) parseWithContract(content string, contract domainInventoryContract) (Extraction, error) {
	out := Extraction{Source: SourceDomainsDoc}

	domains, warns, err := parseDomainInventory(content, contract)
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
func parseDomainInventory(content string, contract domainInventoryContract) ([]ExtractedDomain, []ExtractionWarning, error) {
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
		idxDomain          = -1
		idxResponsibility  = -1
		idxPurpose         = -1
		idxOwnsData        = -1
		idxPrimary         = -1
		idxSecondaryTraits = -1
		idxGlossary        = -1
		idxPaths           = -1
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
			idxDomain = contract.headerIndex(headerCells, "Domain")
			idxResponsibility = contract.headerIndex(headerCells, "Responsibility")
			idxPurpose = contract.headerIndex(headerCells, "Purpose")
			idxOwnsData = contract.headerIndex(headerCells, "Owns Data")
			idxPrimary = contract.headerIndex(headerCells, "Primary Archetype")
			idxSecondaryTraits = contract.headerIndex(headerCells, "Secondary Traits")
			idxGlossary = contract.headerIndex(headerCells, "Glossary")
			idxPaths = contract.headerIndex(headerCells, "Source Paths")
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
			Name:            name,
			Paths:           parsePathList(cell(cells, idxPaths)),
			Glossary:        parseTermList(cell(cells, idxGlossary)),
			Responsibility:  cleanTextCell(cell(cells, idxResponsibility)),
			Purpose:         cleanTextCell(cell(cells, idxPurpose)),
			OwnsData:        cleanTextCell(cell(cells, idxOwnsData)),
			SecondaryTraits: parseArchetypeNames(cell(cells, idxSecondaryTraits)),
		}
		primary := parseArchetypeNames(cell(cells, idxPrimary))
		d.Archetypes = DeclaredArchetypes(append(primary, d.SecondaryTraits...)...)
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

type domainInventoryContract struct {
	Columns []tableColumnContract
}

type tableColumnContract struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	EnumValues []string `json:"enumValues"`
	Aliases    []string `json:"aliases"`
}

func (c domainInventoryContract) headerIndex(headers []string, name string) int {
	if len(c.Columns) == 0 {
		return fallbackHeaderIndex(headers, name)
	}
	names := map[string]struct{}{normalizeHeaderName(name): {}}
	for _, col := range c.Columns {
		if !strings.EqualFold(strings.TrimSpace(col.Name), name) {
			continue
		}
		names[normalizeHeaderName(col.Name)] = struct{}{}
		for _, alias := range col.Aliases {
			names[normalizeHeaderName(alias)] = struct{}{}
		}
	}
	for i, h := range headers {
		if _, ok := names[normalizeHeaderName(h)]; ok {
			return i
		}
	}
	return -1
}

func fallbackHeaderIndex(headers []string, name string) int {
	switch name {
	case "Domain":
		return headerIndex(headers, func(h string) bool { return h == "domain" })
	case "Responsibility":
		return headerIndex(headers, func(h string) bool { return h == "responsibility" })
	case "Purpose":
		return headerIndex(headers, func(h string) bool { return h == "purpose" })
	case "Owns Data":
		return headerIndex(headers, func(h string) bool { return h == "owns data" })
	case "Primary Archetype":
		return headerIndex(headers, func(h string) bool { return strings.Contains(h, "archetype") })
	case "Secondary Traits":
		return headerIndex(headers, func(h string) bool { return h == "secondary traits" })
	case "Glossary":
		return headerIndex(headers, func(h string) bool { return h == "glossary" })
	case "Source Paths":
		return headerIndex(headers, func(h string) bool { return strings.HasPrefix(h, "source paths") })
	default:
		return -1
	}
}

func normalizeHeaderName(header string) string {
	header = strings.TrimSpace(strings.ToLower(header))
	if i := strings.Index(header, "("); i >= 0 {
		header = strings.TrimSpace(header[:i])
	}
	return strings.Join(strings.Fields(header), " ")
}

func loadDomainInventoryContract(scenarioDir string) domainInventoryContract {
	repoRoot := findRepoRoot(scenarioDir)
	if repoRoot == "" {
		return domainInventoryContract{}
	}
	templateID := templateIDForScenario(scenarioDir)
	scenarioManifest := filepath.Join(scenarioDir, "docs", "manifest.json")
	if c, ok := loadContractFromManifest(scenarioManifest); ok {
		return c
	}
	templateManifest := filepath.Join(repoRoot, "templates", "scenarios", templateID, "docs", "manifest.json")
	if c, ok := loadContractFromManifest(templateManifest); ok {
		return c
	}
	return domainInventoryContract{}
}

func loadContractFromManifest(path string) (domainInventoryContract, bool) {
	type manifest struct {
		Sections []struct {
			Documents []struct {
				Path       string `json:"path"`
				Validation struct {
					TableContracts []struct {
						AnchorHeading string                `json:"anchorHeading"`
						Columns       []tableColumnContract `json:"columns"`
					} `json:"tableContracts"`
				} `json:"validation"`
			} `json:"documents"`
		} `json:"sections"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return domainInventoryContract{}, false
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return domainInventoryContract{}, false
	}
	for _, section := range m.Sections {
		for _, doc := range section.Documents {
			if !isDomainsDocManifestPath(doc.Path) {
				continue
			}
			for _, contract := range doc.Validation.TableContracts {
				if strings.EqualFold(strings.TrimSpace(contract.AnchorHeading), "Domain Inventory") {
					return domainInventoryContract{Columns: contract.Columns}, true
				}
			}
		}
	}
	return domainInventoryContract{}, false
}

func isDomainsDocManifestPath(path string) bool {
	normalized := NormalizePath(path)
	return normalized == DomainsDocPath || normalized == strings.TrimPrefix(DomainsDocPath, "docs/")
}

func findRepoRoot(start string) string {
	start = filepath.Clean(start)
	for dir := start; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "templates", "scenarios", "react-vite", "docs", "manifest.json")); err == nil {
			return dir
		}
	}
	return ""
}

func templateIDForScenario(scenarioDir string) string {
	type serviceConfig struct {
		Generation struct {
			Template struct {
				ID string `json:"id"`
			} `json:"template"`
		} `json:"generation"`
	}
	data, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return "react-vite"
	}
	var cfg serviceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "react-vite"
	}
	if id := strings.TrimSpace(cfg.Generation.Template.ID); id != "" {
		return id
	}
	return "react-vite"
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

func cleanTextCell(cellVal string) string {
	cellVal = strings.TrimSpace(strings.Trim(cellVal, "`"))
	if cellVal == "" || cellVal == "-" || cellVal == "—" {
		return ""
	}
	return cellVal
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

// parseArchetypeNames splits an archetype cell into individual labels. The
// labels are returned verbatim (lowercased); canonical normalization onto the
// fixed Archetype vocabulary happens in DeclaredArchetypes, which preserves
// non-canonical labels for honest drift reporting.
func parseArchetypeNames(cellVal string) []string {
	cellVal = strings.TrimSpace(cellVal)
	if cellVal == "" {
		return nil
	}
	parts := strings.FieldsFunc(cellVal, func(r rune) bool {
		return r == ',' || r == '/'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		name := normalizeArchetypeName(part)
		if name == "" || name == "-" || name == "—" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func normalizeArchetypeName(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "`"))
	value = strings.ToLower(value)
	return strings.Join(strings.Fields(value), " ")
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
