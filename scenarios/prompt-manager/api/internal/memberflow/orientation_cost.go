// Per-team orientation cost: how much a reader must hold in their head to work
// inside a team.
//
// Why this exists: adding a capability is supposed to make the system smaller.
// A scenario absorbs work that previously lived as instructions, so the team
// that gained the scenario should end the cycle cheaper to orient in, not more
// expensive. Nothing measured that, so the ratchet could run backwards
// indefinitely — a team could absorb capability and still grow, and the only
// signal was an operator noticing that a team felt heavy.
//
// The composite is deliberately a *count of declared surfaces*, not a quality
// score. Every component is something a new reader must actually read before
// they can act inside the team, and every one is already declared somewhere, so
// the number is auditable rather than modelled.
//
// The band is a trend, not a level. An absolute ceiling on roster size or canon
// length would be arbitrary — a team that owns more is allowed to carry more.
// What is never allowed is for orientation cost to rise in the same cycle that
// scenario coverage grew.
//
// DOC: docs/agent-system/FRAMEWORK_HEALTH.md § Team orientation cost
package memberflow

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// OrientationComponents are the individually-readable quantities the composite
// sums. They are reported alongside the composite so a rise is attributable to
// a surface rather than to an opaque score.
type OrientationComponents struct {
	// Members is the team's roster size.
	Members int `json:"members"`
	// CanonLines is the line count of the team's declared plan-of-record
	// documents, **charged by consumer share**: a document its own manifest
	// entry declares for N consuming teams contributes lines/N here.
	//
	// Lines rather than files: two 40-line docs cost a reader less than one
	// 700-line doc, and file count would say the opposite.
	//
	// The consumer split exists because fleet canon has to live somewhere, and
	// billing all of it to whichever team happens to be its custodian made the
	// team that maintains the framework read as the fleet's heaviest team. The
	// custodian of `docs/agent-system/` was charged 4,241 lines that all six
	// teams consume, which buried the signal that actually distinguished it —
	// roster and topic sprawl. The consumer list is already in the manifest, so
	// this needs no new declaration.
	CanonLines int `json:"canonLines"`
	// SharedCanonLines is the full, unsplit line count of documents this team
	// declares for more than one consumer. It is reported and never charged, so
	// the split above cannot hide how much canon a custodian actually carries.
	SharedCanonLines int `json:"sharedCanonLines,omitempty"`
	// Topics is the count of declared topic families in the team catalog.
	Topics int `json:"topics"`
}

// OrientationCost is one team's reading.
type OrientationCost struct {
	TeamID     string                `json:"teamId"`
	Components OrientationComponents `json:"components"`
	// Composite is the weighted sum. Its absolute value carries no meaning;
	// only its movement between audit cycles does.
	Composite int `json:"composite"`
	// ScenarioCoverage counts the scenarios this team's declared instrument
	// aggregates (`team.json::instrument.coversScenarios`). It is the second
	// quantity in the band: a rise in composite is only a finding when this
	// rose too.
	//
	// It previously counted external nodes in the team's operating graph, on
	// the stated assumption that external nodes are how a team declares the
	// scenarios it composes with. They are not: they resolved to `operator`,
	// `vision-walk`, `swarm-manager-work` and `report-friction` — triggers and
	// people. The custodian of the fleet's own instrument did not have that
	// instrument in its list. The guard series therefore could not move for the
	// reason the band exists, and the band silently never fired.
	ScenarioCoverage int      `json:"scenarioCoverage"`
	Scenarios        []string `json:"scenarios,omitempty"`
	// DomainAddresses counts the scenarios this team's member files name that
	// its declared instrument does **not** account for. The target is zero: an
	// address the instrument already covers is a sanctioned depth read, not a
	// competing place to learn the team's state.
	//
	// The exclusion is load-bearing, and its absence made the first version of
	// this metric wrong. Counting every named scenario conflates two things the
	// target model deliberately separates: an *address* used to learn team state
	// (what deviation D2 is about) and a scenario *called to do work* (the
	// execute exit, which is what capability is for). meta-optimization read as
	// four addresses when three of the four were a workflow invocation, a run
	// investigation, and a depth read of a numerator source its own board
	// already declares — all correct, none a deviation.
	//
	// It still reads low for two opposite reasons — a consolidated team and an
	// unequipped one — so it is meaningless alone and must be read beside
	// ScenarioCoverage and the instrument declaration.
	DomainAddresses int      `json:"domainAddresses"`
	Addresses       []string `json:"addresses,omitempty"`
	// CoveredReads names scenarios the member files call that the instrument
	// declares it aggregates. Reported, never charged: they are the evidence
	// that depth reads are happening against declared sources rather than
	// arbitrary ones, and hiding them would make a consolidated team
	// indistinguishable from an idle one.
	CoveredReads []string `json:"coveredReads,omitempty"`
	// ExternalActors is the operating graph's external node set, retained under
	// an honest name after it stopped backing ScenarioCoverage. It is real
	// information about a team's trigger surface; it was only ever wrong as a
	// proxy for scenario coverage.
	ExternalActors []string `json:"externalActors,omitempty"`
	// MissingCanon names plan-of-record documents the team declares but that
	// do not exist on disk. They are excluded from CanonLines; reporting them
	// keeps a broken reference from reading as a cheaper team.
	MissingCanon []string `json:"missingCanon,omitempty"`
}

// Orientation-cost component weights.
//
// A member costs more than a topic because onboarding to a member means reading
// its responsibilities, heartbeat, and declarations, while a topic is one row
// in a catalog. Canon is divided by a nominal page so a long document does not
// swamp every other component. These are judgment calls, stated here rather
// than buried, and they only need to be *stable* — the band reads movement, so
// a consistent wrong weight still detects the ratchet running backwards.
const (
	orientationWeightMember      = 10
	orientationWeightTopic       = 2
	orientationCanonLinesPerPage = 50
)

// OrientationCostReport is the JSON shape for GET /orientation-cost.
type OrientationCostReport struct {
	Teams []OrientationCost `json:"teams"`
	// Note states the band's shape so a single reading is not mistaken for a
	// verdict. One reading cannot say whether cost moved.
	Note string `json:"note"`
}

const orientationTrendNote = "A single reading is a baseline, not a finding. The band is a trend: compare against the previous framework-health-audit record and raise a finding only when composite rose in a cycle where scenarioCoverage also rose. scenarioCoverage reads team.json::instrument.coversScenarios; a team that declares no instrument reports zero and can never trip the band. domainAddresses is reported beside it and is meaningless alone — it reads low both for a consolidated team and for an unequipped one."

// ComputeOrientationCost reads every team's declared surfaces and returns the
// composite for each.
func ComputeOrientationCost(configDir, repoRoot string) (OrientationCostReport, error) {
	report := OrientationCostReport{Teams: []OrientationCost{}, Note: orientationTrendNote}

	contracts, err := LoadAllTeamContracts(configDir)
	if err != nil {
		return report, err
	}
	// Model documents supply the external-actor set. A model that fails to load
	// costs that team its actor list, not the whole report.
	models, _ := LoadOperatingModelDocuments(repoRoot)
	externalByTeam := externalScenariosByTeam(models)

	// Instrument declarations supply the guard series. A store that cannot be
	// read leaves every team's coverage at zero rather than failing the report;
	// zero coverage can never make the trend band fire, so the degraded mode is
	// silent-safe rather than falsely alarming.
	instruments, _ := LoadTeamInstruments(configDir)
	coveredByTeam := instrumentCoveredScenarios(instruments)
	addressesByTeam, coveredByTeamReads := domainAddressesByTeam(configDir, repoRoot, sortedMapKeys(contracts), instruments)

	for _, teamID := range sortedMapKeys(contracts) {
		contract := contracts[teamID]
		cost := OrientationCost{TeamID: teamID}
		cost.Components.Members = countTeamMembers(configDir, teamID, contract)
		cost.Components.Topics = len(contract.TopicCatalog)
		charged, shared, missing := canonLineCount(repoRoot, contract)
		cost.Components.CanonLines = charged
		cost.Components.SharedCanonLines = shared
		cost.MissingCanon = missing
		cost.Composite = orientationComposite(cost.Components)
		cost.Scenarios = coveredByTeam[teamID]
		cost.ScenarioCoverage = len(cost.Scenarios)
		cost.Addresses = addressesByTeam[teamID]
		cost.DomainAddresses = len(cost.Addresses)
		cost.CoveredReads = coveredByTeamReads[teamID]
		cost.ExternalActors = externalByTeam[teamID]
		report.Teams = append(report.Teams, cost)
	}
	return report, nil
}

func orientationComposite(c OrientationComponents) int {
	return c.Members*orientationWeightMember +
		c.Topics*orientationWeightTopic +
		c.CanonLines/orientationCanonLinesPerPage
}

// countTeamMembers prefers the on-disk member directories over roles.json: the
// directories are what a reader actually opens, and a role declared without a
// member directory costs nothing to orient in.
func countTeamMembers(configDir, teamID string, contract *LoadedTeamContract) int {
	dir := filepath.Join(configDir, "teams", teamID, "members")
	entries, err := os.ReadDir(dir)
	if err == nil {
		count := 0
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				count++
			}
		}
		return count
	}
	if contract != nil && contract.Contract != nil {
		return len(contract.Contract.Members)
	}
	return 0
}

// canonLineCount sums the line counts of a team's declared plan-of-record
// documents.
//
// A declared path ending in "/" is a canon *root*: every .md beneath it counts,
// and no per-file declaration is kept. Roots exist because an enumerated file
// list is a second source of truth that drifts silently — a canon file added to
// the folder but not to the list was invisible here, so framework growth cost
// nothing and the ratchet could not fire on it. A root cannot drift: the folder
// is the declaration. The trade is that everything markdown under a root counts,
// including generated projections, so a folder is charged for what it contains.
//
// Returns (charged, shared, missing): charged is the consumer-split total that
// feeds the composite, shared is the full unsplit count of multi-consumer
// documents, and missing names declared paths that do not exist on disk.
func canonLineCount(repoRoot string, contract *LoadedTeamContract) (int, int, []string) {
	if contract == nil {
		return 0, 0, nil
	}
	total := 0
	shared := 0
	var missing []string
	seen := map[string]bool{}
	for _, doc := range contract.PlanOfRecordDocuments {
		// A document declared for N consuming teams is N teams' orientation
		// cost, not one team's. One consumer (or none declared) charges in
		// full — the common case, and the conservative default.
		consumers := len(doc.Consumers)
		if consumers < 1 {
			consumers = 1
		}
		for _, ref := range doc.Paths {
			// Only repo-root paths are resolvable here. A member-relative ref
			// is member documentation, which the roster component already
			// accounts for.
			if strings.TrimSpace(ref.Base) != "repo-root" {
				continue
			}
			path := filepath.ToSlash(strings.TrimSpace(ref.Path))
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			var lines int
			if strings.HasSuffix(path, "/") {
				rootLines, err := canonRootLineCount(repoRoot, path)
				if err != nil {
					missing = append(missing, path)
					continue
				}
				lines = rootLines
			} else {
				data, err := os.ReadFile(filepath.Join(repoRoot, path))
				if err != nil {
					missing = append(missing, path)
					continue
				}
				lines = strings.Count(string(data), "\n") + 1
			}
			total += lines / consumers
			if consumers > 1 {
				shared += lines
			}
		}
	}
	sort.Strings(missing)
	return total, shared, missing
}

// Address-scan vocabulary.
//
// universalSubstrateScenarios are excluded because every team calls them by
// construction — prompt-manager is the runtime the members run on and
// swarm-manager is the shared portfolio. Counting them would add a constant to
// every team and make the number less able to separate teams, which is the
// only job it has.
var universalSubstrateScenarios = map[string]bool{
	"prompt-manager": true,
	"swarm-manager":  true,
}

// addressScanNoise are scenario names that are also ordinary English words or
// generic test fixtures. Matching them produces false addresses from prose that
// was never naming a scenario ("record durable notes", "run the test"). They
// are skipped rather than resolved because a false address inflates the reading
// in the direction that hides consolidation, which is the failure that matters.
var addressScanNoise = map[string]bool{
	"notes":         true,
	"test":          true,
	"portal":        true,
	"calendar":      true,
	"progress":      true,
	"simple-test":   true,
	"test-scenario": true,
}

// domainAddressesByTeam scans each team's member files for named scenarios and
// splits them into uncovered addresses and covered depth reads.
//
// Member files are the right surface because they are what a member actually
// reads before acting: a scenario named in a heartbeat or in standing
// responsibilities is something that member must hold, whether or not the team
// contract mentions it.
//
// The split is what makes the count mean anything. A scenario the team's
// instrument declares in `scenario` or `coversScenarios` is a declared source —
// reading it for detail the board does not surface is the intended pattern, and
// the board itself reads it as a numerator. Only a scenario *outside* that
// declaration is a second place to learn the team's state.
func domainAddressesByTeam(configDir, repoRoot string, teamIDs []string, instruments map[string]*TeamInstrument) (map[string][]string, map[string][]string) {
	out := map[string][]string{}
	covered := map[string][]string{}
	scenarios := repoScenarioNames(repoRoot)
	if len(scenarios) == 0 {
		return out, covered
	}
	for _, teamID := range teamIDs {
		membersDir := filepath.Join(configDir, "teams", teamID, "members")
		entries, err := os.ReadDir(membersDir)
		if err != nil {
			continue
		}
		found := map[string]bool{}
		for _, me := range entries {
			if !me.IsDir() || strings.HasPrefix(me.Name(), ".") {
				continue
			}
			files, err := os.ReadDir(filepath.Join(membersDir, me.Name()))
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() || !strings.EqualFold(filepath.Ext(f.Name()), ".md") {
					continue
				}
				data, err := os.ReadFile(filepath.Join(membersDir, me.Name(), f.Name()))
				if err != nil {
					continue
				}
				for _, name := range scenarios {
					if !found[name] && containsScenarioToken(string(data), name) {
						found[name] = true
					}
				}
			}
		}
		accounted := instrumentAccountedScenarios(instruments[teamID])
		var names, reads []string
		for name := range found {
			if accounted[name] {
				reads = append(reads, name)
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		sort.Strings(reads)
		if len(names) > 0 {
			out[teamID] = names
		}
		if len(reads) > 0 {
			covered[teamID] = reads
		}
	}
	return out, covered
}

// instrumentAccountedScenarios is the set a team's instrument declares: the
// board itself plus everything it says it aggregates.
func instrumentAccountedScenarios(inst *TeamInstrument) map[string]bool {
	out := map[string]bool{}
	if inst == nil {
		return out
	}
	if name := strings.TrimSpace(inst.Scenario); name != "" {
		out[name] = true
	}
	for _, s := range inst.CoversScenarios {
		if name := strings.TrimSpace(s); name != "" {
			out[name] = true
		}
	}
	return out
}

// repoScenarioNames lists the scenario directory names worth scanning for.
func repoScenarioNames(repoRoot string) []string {
	if strings.TrimSpace(repoRoot) == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		name := e.Name()
		if universalSubstrateScenarios[name] || addressScanNoise[name] {
			continue
		}
		out = append(out, name)
	}
	// Longest first so a scan cannot match `agent-manager` inside
	// `agent-metareasoning-manager` before trying the longer name.
	sort.Slice(out, func(i, j int) bool {
		if len(out[i]) != len(out[j]) {
			return len(out[i]) > len(out[j])
		}
		return out[i] < out[j]
	})
	return out
}

// containsScenarioToken reports whether body names the scenario as a whole
// token. Scenario names are hyphenated, so a plain substring search matches
// `agent-manager` inside `agent-metareasoning-manager`; the boundary check
// treats a hyphen as a word character to prevent it.
func containsScenarioToken(body, name string) bool {
	for i := 0; ; {
		idx := strings.Index(body[i:], name)
		if idx < 0 {
			return false
		}
		start := i + idx
		end := start + len(name)
		if !scenarioTokenChar(body, start-1) && !scenarioTokenChar(body, end) {
			return true
		}
		i = start + 1
		if i >= len(body) {
			return false
		}
	}
}

// scenarioTokenChar reports whether the byte at index is part of a scenario
// token. Out-of-range indices are not, which makes start/end of file a
// boundary.
func scenarioTokenChar(body string, index int) bool {
	if index < 0 || index >= len(body) {
		return false
	}
	c := body[index]
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_':
		return true
	}
	return false
}

// canonRootLineCount sums every .md beneath a declared canon root. Only .md
// counts: sidecar JSON in a plan-of-record folder is machine config the reader
// never orients into, and the manifest already validates it.
func canonRootLineCount(repoRoot, root string) (int, error) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(strings.TrimSuffix(root, "/")))
	info, err := os.Stat(dir)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, fs.ErrInvalid
	}
	total := 0
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return nil //nolint:nilerr // an unreadable child is skipped, not fatal
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		total += strings.Count(string(data), "\n") + 1
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

// externalScenariosByTeam collects the distinct external nodes each team's
// operating graph names. External nodes are how a team declares the scenarios
// its work composes with, so their count is the closest declared proxy for
// "how much of this team's work a scenario now absorbs".
func externalScenariosByTeam(models []OperatingModelDocument) map[string][]string {
	out := map[string][]string{}
	for _, model := range models {
		team := strings.TrimSpace(model.Team)
		if team == "" {
			continue
		}
		seen := map[string]bool{}
		for _, block := range model.Graphs {
			for _, node := range block.Graph.Nodes {
				if node.Kind != OperatingGraphNodeKindExternal {
					continue
				}
				value := strings.TrimSpace(node.Value)
				if value == "" || seen[value] {
					continue
				}
				seen[value] = true
			}
		}
		names := make([]string, 0, len(seen))
		for name := range seen {
			names = append(names, name)
		}
		sort.Strings(names)
		out[team] = append(out[team], names...)
	}
	return out
}

// GetOrientationCost handles GET /orientation-cost.
func (h *Handlers) GetOrientationCost(w http.ResponseWriter, r *http.Request) {
	report, err := ComputeOrientationCost(h.configDir, h.repoRoot())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}
