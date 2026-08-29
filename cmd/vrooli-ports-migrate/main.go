// vrooli-ports-migrate shifts every scenario UI listener port out of the
// Linux ephemeral range (32768-60999) by applying a uniform newPort =
// oldPort - 15000 shift. It discovers every `.vrooli/service.json` in the
// repo (plus the two react-vite templates), rewrites fixed ports and ranges
// in-place, and prints a before→after table with a per-scenario "tunneled?"
// flag so external route configurations can be updated in the same pass.
//
// Default mode is dry-run. Use --apply to write changes. The tool is
// idempotent — running it twice after a successful apply is a no-op.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

const (
	portKeyColumnWidth = 9
)

const (
	mndMainNumberOctal644 = 0o644
	invalidRootExitCode   = 2
	collisionExitCode     = 3
	scenarioColumnWidth   = 41
	rangeBoundCount       = 2
)

const (
	oldRangeLow  = 35000
	oldRangeHigh = 39999
	shift        = -15000
	newRangeLow  = oldRangeLow + shift  // 20000
	newRangeHigh = oldRangeHigh + shift // 24999
)

type change struct {
	Scenario string `json:"scenario"`
	Path     string `json:"path"`
	PortKey  string `json:"port_key"`
	Field    string `json:"field"` // "port" or "range"
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
	Tunneled bool   `json:"tunneled,omitempty"`
}

type manifestResult struct {
	Path     string   `json:"path"`
	Scenario string   `json:"scenario"`
	Changes  []change `json:"changes,omitempty"`
	Tunneled bool     `json:"tunneled,omitempty"`
	Skipped  string   `json:"skipped,omitempty"` // populated when skipped with a reason
}

type report struct {
	Repo       string           `json:"repo"`
	Manifests  []manifestResult `json:"manifests"`
	Collisions []string         `json:"collisions,omitempty"`
	Applied    bool             `json:"applied"`
}

func main() {
	var (
		apply    = flag.Bool("apply", false, "write changes in place (default is dry-run)")
		asJSON   = flag.Bool("json", false, "emit JSON report instead of human-readable table")
		repoRoot = flag.String("root", "", "repository root (default: auto-detect from cwd)")
	)
	flag.Parse()

	root, err := resolveRepoRoot(*repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(invalidRootExitCode)
	}

	rep, err := runMigration(root, *apply)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	} else {
		printHumanReport(os.Stdout, rep, *apply)
	}

	if len(rep.Collisions) > 0 {
		os.Exit(collisionExitCode)
	}
}

// resolveRepoRoot returns the supplied path or uses the shared repo-contract
// resolver. Templates + scenarios both live at known relative locations so the
// root is enough to drive everything.
func resolveRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	return repocontract.FindRepoRootFromEnvOrCWD()
}

func runMigration(root string, apply bool) (report, error) {
	rep := report{Repo: root, Applied: apply}

	manifestPaths, err := discoverManifests(root)
	if err != nil {
		return rep, err
	}

	// Collect every (scenario, new port) mapping so we can flag collisions
	// before writing anything.
	pendingFixed := make(map[int][]string) // newPort -> scenario names

	for _, path := range manifestPaths {
		res, err := processManifest(root, path, apply)
		if err != nil {
			return rep, fmt.Errorf("process %s: %w", path, err)
		}
		rep.Manifests = append(rep.Manifests, res)
		for _, ch := range res.Changes {
			if ch.Field != "port" {
				continue
			}
			newPort, perr := strconv.Atoi(ch.NewValue)
			if perr != nil {
				continue
			}
			pendingFixed[newPort] = append(pendingFixed[newPort], res.Scenario)
		}
	}

	for port, scenarios := range pendingFixed {
		if len(scenarios) > 1 {
			sort.Strings(scenarios)
			rep.Collisions = append(rep.Collisions,
				fmt.Sprintf("new port %d claimed by: %s", port, strings.Join(scenarios, ", ")))
		}
	}
	sort.Strings(rep.Collisions)

	// If we detected collisions in --apply mode, caller exits non-zero;
	// we've already written the shifted files for the non-colliding ones,
	// but the collisions are left for the operator to resolve.
	return rep, nil
}

func discoverManifests(root string) ([]string, error) {
	var out []string

	// Scenarios
	scenariosDir := filepath.Join(root, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err == nil {
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			candidate := filepath.Join(scenariosDir, ent.Name(), ".vrooli", "service.json")
			if _, serr := os.Stat(candidate); serr == nil {
				out = append(out, candidate)
			}
		}
	}

	// Templates
	templatesDir := filepath.Join(root, "templates", "scenarios")
	if err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "service.json" && strings.Contains(path, ".vrooli"+string(filepath.Separator)) {
			out = append(out, path)
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	sort.Strings(out)
	return out, nil
}

func processManifest(root, path string, apply bool) (manifestResult, error) {
	res := manifestResult{Path: path}
	scenario := scenarioSlugFromPath(root, path)
	res.Scenario = scenario
	res.Tunneled = detectTunneled(root, scenario, path)

	raw, err := os.ReadFile(path)
	if err != nil {
		return res, err
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		res.Skipped = "unparseable JSON: " + err.Error()
		return res, nil
	}

	portsRaw, ok := doc["ports"]
	if !ok {
		res.Skipped = "no ports block"
		return res, nil
	}
	var ports map[string]map[string]json.RawMessage
	if err := json.Unmarshal(portsRaw, &ports); err != nil {
		res.Skipped = "ports not a keyed object"
		return res, nil
	}

	changes := collectChanges(scenario, path, ports, res.Tunneled)
	res.Changes = changes
	if len(changes) == 0 {
		return res, nil
	}

	if apply {
		newContent := applyChanges(raw, changes)
		if err := os.WriteFile(path, newContent, mndMainNumberOctal644); err != nil {
			return res, err
		}
	}
	return res, nil
}

// collectChanges walks one scenario's ports map and enumerates the shifts
// that would be applied. It only shifts fixed ports that sit strictly inside
// oldRangeLow..oldRangeHigh, and ranges whose low bound is inside the old
// range. Anything already below 32768 (already shifted, or custom non-UI
// ports) is left alone so the tool is idempotent.
func collectChanges(scenario, path string, ports map[string]map[string]json.RawMessage, tunneled bool) []change {
	var out []change

	keys := make([]string, 0, len(ports))
	for k := range ports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		entry := ports[key]
		if portRaw, ok := entry["port"]; ok {
			if oldPort, ok := parseIntJSON(portRaw); ok && oldPort >= oldRangeLow && oldPort <= oldRangeHigh {
				out = append(out, change{
					Scenario: scenario,
					Path:     path,
					PortKey:  key,
					Field:    "port",
					OldValue: strconv.Itoa(oldPort),
					NewValue: strconv.Itoa(oldPort + shift),
					Tunneled: tunneled,
				})
			}
		}
		if rangeRaw, ok := entry["range"]; ok {
			if oldRange, ok := parseStringJSON(rangeRaw); ok {
				if newRange, changed := shiftRange(oldRange); changed {
					out = append(out, change{
						Scenario: scenario,
						Path:     path,
						PortKey:  key,
						Field:    "range",
						OldValue: oldRange,
						NewValue: newRange,
						Tunneled: tunneled,
					})
				}
			}
		}
	}
	return out
}

func shiftRange(raw string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(raw), "-")
	if len(parts) != rangeBoundCount {
		return raw, false
	}
	lo, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return raw, false
	}
	hi, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return raw, false
	}
	// Only shift ranges whose low bound is in the old UI band. Partial
	// overlaps (e.g. 30000-40000) are ambiguous; flag them as collisions by
	// refusing to shift and surfacing them to the operator.
	if lo < oldRangeLow || lo > oldRangeHigh {
		return raw, false
	}
	if hi < oldRangeLow || hi > oldRangeHigh {
		return raw, false
	}
	return fmt.Sprintf("%d-%d", lo+shift, hi+shift), true
}

func parseIntJSON(raw json.RawMessage) (int, bool) {
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, true
	}
	return 0, false
}

func parseStringJSON(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, true
	}
	return "", false
}

// applyChanges rewrites the raw JSON bytes in place using surgical string
// replacement so whitespace, key order, and comments (if any) are preserved.
// Each change's OldValue is matched within the scope of its port key's object
// so we never touch a matching literal elsewhere in the file.
func applyChanges(original []byte, changes []change) []byte {
	content := string(original)
	for _, ch := range changes {
		content = rewriteOne(content, ch)
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return []byte(content)
}

// rewriteOne finds the block belonging to ch.PortKey and, within that block,
// replaces the relevant field's value. This avoids mis-matching a number or
// range that happens to appear elsewhere (e.g. a health endpoint key named
// "ui" on an unrelated line, or another port entry that coincidentally
// shares the same literal).
func rewriteOne(content string, ch change) string {
	portsStart, portsEnd, ok := findPortsBlock(content)
	if !ok {
		return content
	}
	portsBlock := content[portsStart : portsEnd+1]

	keyQuoted := `"` + ch.PortKey + `"`
	keyIdx := strings.Index(portsBlock, keyQuoted)
	if keyIdx < 0 {
		return content
	}
	braceIdx := strings.Index(portsBlock[keyIdx:], "{")
	if braceIdx < 0 {
		return content
	}
	braceIdx += keyIdx
	blockEnd := findMatchingBrace(portsBlock, braceIdx)
	if blockEnd < 0 {
		return content
	}
	block := portsBlock[braceIdx : blockEnd+1]
	var updated string
	if ch.Field == "port" {
		updated = replacePortField(block, ch.OldValue, ch.NewValue)
	} else {
		updated = replaceRangeField(block, ch.OldValue, ch.NewValue)
	}
	if updated == block {
		return content
	}
	newPortsBlock := portsBlock[:braceIdx] + updated + portsBlock[blockEnd+1:]
	return content[:portsStart] + newPortsBlock + content[portsEnd+1:]
}

// findPortsBlock returns the byte offsets of the outermost top-level
// "ports" object value in the service.json. Scans for `"ports"` keys at any
// depth but returns the first one whose value is an object.
func findPortsBlock(content string) (int, int, bool) {
	key := `"ports"`
	idx := 0
	for {
		rel := strings.Index(content[idx:], key)
		if rel < 0 {
			return 0, 0, false
		}
		keyPos := idx + rel
		// Walk forward past whitespace and the colon, then look for the
		// opening brace. If we find anything else (e.g. a string value),
		// this wasn't the right "ports" key; advance and keep searching.
		scan := keyPos + len(key)
		for scan < len(content) && (content[scan] == ' ' || content[scan] == '\t' || content[scan] == '\n' || content[scan] == '\r' || content[scan] == ':') {
			scan++
		}
		if scan >= len(content) || content[scan] != '{' {
			idx = keyPos + len(key)
			continue
		}
		end := findMatchingBrace(content, scan)
		if end < 0 {
			return 0, 0, false
		}
		return scan, end, true
	}
}

// findMatchingBrace returns the index of the `}` that closes the `{` at
// startBrace, respecting string literals. Returns -1 if no match is found.
func findMatchingBrace(content string, startBrace int) int {
	depth := 0
	inString := false
	escape := false
	for i := startBrace; i < len(content); i++ {
		c := content[i]
		if escape {
			escape = false
			continue
		}
		if inString {
			switch c {
			case '\\':
				escape = true
			case '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

var (
	portFieldPattern  = regexp.MustCompile(`("port"\s*:\s*)(\d+)`)
	rangeFieldPattern = regexp.MustCompile(`("range"\s*:\s*")([^"]+)(")`)
)

func replacePortField(block, oldVal, newVal string) string {
	return portFieldPattern.ReplaceAllStringFunc(block, func(m string) string {
		sub := portFieldPattern.FindStringSubmatch(m)
		if len(sub) != 3 || sub[2] != oldVal {
			return m
		}
		return sub[1] + newVal
	})
}

func replaceRangeField(block, oldVal, newVal string) string {
	return rangeFieldPattern.ReplaceAllStringFunc(block, func(m string) string {
		sub := rangeFieldPattern.FindStringSubmatch(m)
		if len(sub) != 4 || sub[2] != oldVal {
			return m
		}
		return sub[1] + newVal + sub[3]
	})
}

// scenarioSlugFromPath derives a human label from the manifest path. For
// `scenarios/<name>/.vrooli/service.json` this is `<name>`; for templates it
// is `template:<template-name>`.
func scenarioSlugFromPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) >= 4 && parts[0] == "scenarios" {
		return parts[1]
	}
	if len(parts) >= 4 && parts[0] == "templates" {
		return "template:" + parts[2]
	}
	return rel
}

// detectTunneled grep-scans the scenario's README + the repo-root CLAUDE.md
// for keywords that suggest the port is externally published (Cloudflare
// tunnel, fixed public URL, etc.). False positives are preferred over false
// negatives — the flag only surfaces a reminder, it never blocks the shift.
func detectTunneled(root, scenario, manifestPath string) bool {
	if strings.HasPrefix(scenario, "template:") {
		return false
	}
	candidates := []string{
		filepath.Join(filepath.Dir(filepath.Dir(manifestPath)), "README.md"),
		filepath.Join(filepath.Dir(filepath.Dir(manifestPath)), "CLAUDE.md"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lowered := strings.ToLower(string(data))
		if strings.Contains(lowered, "tunnel") ||
			strings.Contains(lowered, "cloudflare") ||
			strings.Contains(lowered, "public url") ||
			strings.Contains(lowered, "public-facing") {
			return true
		}
	}
	return false
}

func printHumanReport(out *os.File, rep report, applied bool) {
	header := "DRY-RUN"
	if applied {
		header = "APPLIED"
	}
	fmt.Fprintf(out, "vrooli-ports-migrate (%s)\n", header)
	fmt.Fprintf(out, "repo: %s\n\n", rep.Repo)

	totalChanges := 0
	tunneledChanged := 0

	// Sort manifests by scenario name for stable output.
	sort.Slice(rep.Manifests, func(i, j int) bool {
		return rep.Manifests[i].Scenario < rep.Manifests[j].Scenario
	})

	fmt.Fprintln(out, "SCENARIO                                  KEY       FIELD  OLD            NEW            TUNNELED")
	fmt.Fprintln(out, "----------------------------------------- --------- ------ -------------- -------------- --------")
	for _, m := range rep.Manifests {
		for _, ch := range m.Changes {
			totalChanges++
			if ch.Tunneled {
				tunneledChanged++
			}
			flag := ""
			if ch.Tunneled {
				flag = "yes"
			}
			fmt.Fprintf(out, "%-41s %-9s %-6s %-14s %-14s %s\n",
				truncate(m.Scenario, scenarioColumnWidth), truncate(ch.PortKey, portKeyColumnWidth), ch.Field, ch.OldValue, ch.NewValue, flag)
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "changes:    %d across %d manifest(s)\n", totalChanges, countManifestsWithChanges(rep.Manifests))
	fmt.Fprintf(out, "tunneled:   %d change(s) on scenarios with tunnel/cloudflare keywords — update external config after apply\n", tunneledChanged)
	if len(rep.Collisions) > 0 {
		fmt.Fprintln(out, "\nCOLLISIONS:")
		for _, c := range rep.Collisions {
			fmt.Fprintln(out, "  -", c)
		}
	}
	if !applied && totalChanges > 0 {
		fmt.Fprintln(out, "\nRun with --apply to write these changes.")
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func countManifestsWithChanges(ms []manifestResult) int {
	n := 0
	for _, m := range ms {
		if len(m.Changes) > 0 {
			n++
		}
	}
	return n
}
