package zones

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ArchitectureDocPath is the scenario-relative location of the human-authored
// architecture document whose "## Zone Map" table declares the intended
// code-layout layering. It is the DECLARED signal in zone convergence; the
// template manifest is the DERIVED signal and the import graph is the reality.
const ArchitectureDocPath = "docs/concepts/ARCHITECTURE.md"

// DeclaredZone is one row of the ARCHITECTURE.md "## Zone Map" table: a
// canonical zone, the verbatim layer label the author wrote, and the
// repo-relative path conventions the author assigned to it.
type DeclaredZone struct {
	Zone          string // canonical zone (Transport/Domain/...) or Unknown
	DeclaredLayer string // raw label as written, preserved for honest reporting
	Patterns      []string
}

// DeclaredZoneMap is the parsed ARCHITECTURE.md zone declaration. Present is
// false when the document has no "## Zone Map" table (zones are then derived-
// only and the map's authority confidence is LOW).
type DeclaredZoneMap struct {
	Present bool
	Zones   []DeclaredZone
}

// layerNameToZone normalizes the free-text "layer" label an author may write in
// the ARCHITECTURE.md Zone Map onto a canonical zone. Unknown labels map to
// Unknown (reported, never a hard fail).
var layerNameToZone = map[string]string{
	"transport":             Transport,
	"api surface":           Transport,
	"handler":               Transport,
	"handlers":              Transport,
	"domain":                Domain,
	"scenario core":         Domain,
	"core":                  Domain,
	"substrate":             Substrate,
	"shared infrastructure": Substrate,
	"infrastructure":        Substrate,
	"composition-root":      CompositionRoot,
	"composition root":      CompositionRoot,
	"bootstrap":             CompositionRoot,
	"wiring":                CompositionRoot,
	"cli":                   CLI,
	"operator wrapper":      CLI,
	"ui":                    UI,
	"browser presentation":  UI,
	"presentation":          UI,
}

// NormalizeLayer maps a declared layer label onto a canonical zone. ok is false
// when the label has no canonical mapping.
func NormalizeLayer(label string) (string, bool) {
	z, ok := layerNameToZone[strings.ToLower(strings.TrimSpace(label))]
	return z, ok
}

// LoadDeclaredZoneMap parses the "## Zone Map" table from a scenario's
// ARCHITECTURE.md. A missing file or section returns {Present: false} (absence
// is not failure).
func LoadDeclaredZoneMap(scenarioDir string) DeclaredZoneMap {
	data, err := os.ReadFile(filepath.Join(scenarioDir, ArchitectureDocPath))
	if err != nil {
		return DeclaredZoneMap{}
	}
	return ParseDeclaredZoneMap(string(data))
}

// LoadDeclaredZoneMapForScenario resolves a scenario name to its directory
// (mirroring LoadForScenarioName) and parses its ARCHITECTURE.md Zone Map.
func LoadDeclaredZoneMapForScenario(scenario string) DeclaredZoneMap {
	root := findRepoRoot("")
	if root == "" {
		return DeclaredZoneMap{}
	}
	return LoadDeclaredZoneMap(filepath.Join(root, "scenarios", strings.TrimSpace(scenario)))
}

// ParseDeclaredZoneMap is the pure parsing core. The expected table has a
// "Zone" (or "Layer"), an optional "Declared Layer", and a path-convention
// column ("Path Convention" / "Paths"). Rows are matched by header name so
// extra columns are tolerated.
func ParseDeclaredZoneMap(content string) DeclaredZoneMap {
	section, ok := sectionBody(content, "Zone Map")
	if !ok {
		return DeclaredZoneMap{}
	}
	var (
		out        DeclaredZoneMap
		headerSeen bool
		cols       zoneMapColumns
	)
	sc := bufio.NewScanner(strings.NewReader(section))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			if headerSeen {
				break
			}
			continue
		}
		cells := splitTableRow(line)
		if isSeparatorRow(cells) {
			continue
		}
		if !headerSeen {
			var ok bool
			if cols, ok = parseZoneMapHeader(cells); !ok {
				return DeclaredZoneMap{}
			}
			headerSeen = true
			continue
		}
		if dz, ok := cols.parseRow(cells); ok {
			out.Zones = append(out.Zones, dz)
		}
	}
	out.Present = headerSeen && len(out.Zones) > 0
	return out
}

// zoneMapColumns holds the resolved column indices of a Zone Map table.
type zoneMapColumns struct {
	zone  int
	layer int
	paths int
}

func parseZoneMapHeader(cells []string) (zoneMapColumns, bool) {
	cols := zoneMapColumns{zone: -1, layer: -1, paths: -1}
	for i, h := range cells {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "zone":
			cols.zone = i
		case "declared layer", "layer":
			cols.layer = i
		case "path convention", "paths", "path conventions", "source paths":
			cols.paths = i
		}
	}
	if cols.zone < 0 && cols.layer >= 0 {
		cols.zone = cols.layer
	}
	if cols.zone < 0 || cols.paths < 0 {
		return zoneMapColumns{}, false
	}
	return cols, true
}

func (c zoneMapColumns) parseRow(cells []string) (DeclaredZone, bool) {
	rawZone := cellAt(cells, c.zone)
	rawLayer := rawZone
	if c.layer >= 0 && c.layer != c.zone {
		if l := cellAt(cells, c.layer); l != "" {
			rawLayer = l
		}
	}
	patterns := parseZonePatternCell(cellAt(cells, c.paths))
	if rawZone == "" && len(patterns) == 0 {
		return DeclaredZone{}, false
	}
	zone, _ := NormalizeLayer(rawZone)
	return DeclaredZone{Zone: zone, DeclaredLayer: rawLayer, Patterns: patterns}, true
}

// ZoneFor returns the declared zone + raw layer label for a repo path, matching
// the path against each declared zone's path conventions. ok is false when no
// declared convention matches.
func (m DeclaredZoneMap) ZoneFor(repoPath string) (zone, declaredLayer string, ok bool) {
	path := strings.Trim(strings.TrimSpace(repoPath), "/")
	if path == "" || !m.Present {
		return "", "", false
	}
	for _, dz := range m.Zones {
		for _, pattern := range dz.Patterns {
			prefix := zonePatternPrefix(pattern)
			if prefix == "" {
				continue
			}
			if path == prefix || strings.HasPrefix(path, prefix+"/") || strings.HasPrefix(path+"/", prefix+"/") {
				return dz.Zone, dz.DeclaredLayer, true
			}
		}
	}
	return "", "", false
}

// zonePatternPrefix reduces a declared path convention to a matchable prefix by
// dropping the trailing <domain>/<segment> placeholder and surrounding slashes.
func zonePatternPrefix(pattern string) string {
	pattern = strings.Trim(strings.TrimSpace(strings.Trim(pattern, "`")), "/")
	for _, ph := range []string{"<domain>", "<segment>", "<name>"} {
		if i := strings.Index(pattern, ph); i >= 0 {
			pattern = strings.Trim(pattern[:i], "/")
		}
	}
	return pattern
}

func parseZonePatternCell(cell string) []string {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(cell, ",") {
		part = strings.Trim(strings.TrimSpace(part), "`")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func cellAt(cells []string, idx int) string {
	if idx < 0 || idx >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[idx])
}

// sectionBody returns the body lines under a "## <heading>" markdown section,
// stopping at the next heading of the same or higher level.
func sectionBody(content, heading string) (string, bool) {
	sc := bufio.NewScanner(strings.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var b strings.Builder
	in := false
	target := strings.ToLower(strings.TrimSpace(heading))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			title := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trimmed, "# ")))
			if !in && title == target {
				in = true
				continue
			}
			if in {
				break // next heading ends the section
			}
		}
		if in {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String(), in
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func isSeparatorRow(cells []string) bool {
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
