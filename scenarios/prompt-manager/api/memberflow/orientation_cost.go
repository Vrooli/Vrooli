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
	// CanonLines is the total line count of the team's declared
	// plan-of-record documents. Lines rather than files: two 40-line docs
	// cost a reader less than one 700-line doc, and file count would say the
	// opposite.
	CanonLines int `json:"canonLines"`
	// Topics is the count of declared topic families in the team catalog.
	Topics int `json:"topics"`
	// DecisionContexts is the count of declared decision contexts — each is
	// a separate lifecycle a member must know when to enter.
	DecisionContexts int `json:"decisionContexts"`
}

// OrientationCost is one team's reading.
type OrientationCost struct {
	TeamID     string                `json:"teamId"`
	Components OrientationComponents `json:"components"`
	// Composite is the weighted sum. Its absolute value carries no meaning;
	// only its movement between audit cycles does.
	Composite int `json:"composite"`
	// ScenarioCoverage counts the distinct external scenarios the team's
	// operating graph composes with. It is the second quantity in the band:
	// a rise in composite is only a finding when this rose too.
	ScenarioCoverage int      `json:"scenarioCoverage"`
	Scenarios        []string `json:"scenarios,omitempty"`
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
	orientationWeightMember          = 10
	orientationWeightTopic           = 2
	orientationWeightDecisionContext = 3
	orientationCanonLinesPerPage     = 50
)

// OrientationCostReport is the JSON shape for GET /orientation-cost.
type OrientationCostReport struct {
	Teams []OrientationCost `json:"teams"`
	// Note states the band's shape so a single reading is not mistaken for a
	// verdict. One reading cannot say whether cost moved.
	Note string `json:"note"`
}

const orientationTrendNote = "A single reading is a baseline, not a finding. The band is a trend: compare against the previous framework-health-audit record and raise a finding only when composite rose in a cycle where scenarioCoverage also rose."

// ComputeOrientationCost reads every team's declared surfaces and returns the
// composite for each.
func ComputeOrientationCost(configDir, repoRoot string) (OrientationCostReport, error) {
	report := OrientationCostReport{Teams: []OrientationCost{}, Note: orientationTrendNote}

	contracts, err := LoadAllTeamContracts(configDir)
	if err != nil {
		return report, err
	}
	// Model documents supply scenario coverage. A model that fails to load
	// costs that team its coverage number, not the whole report.
	models, _ := LoadOperatingModelDocuments(repoRoot)
	scenariosByTeam := externalScenariosByTeam(models)

	for _, teamID := range sortedMapKeys(contracts) {
		contract := contracts[teamID]
		cost := OrientationCost{TeamID: teamID}
		cost.Components.Members = countTeamMembers(configDir, teamID, contract)
		cost.Components.Topics = len(contract.TopicCatalog)
		if contract.Contract != nil {
			cost.Components.DecisionContexts = len(contract.Contract.DecisionContext)
		}
		lines, missing := canonLineCount(repoRoot, contract)
		cost.Components.CanonLines = lines
		cost.MissingCanon = missing
		cost.Composite = orientationComposite(cost.Components)
		cost.Scenarios = scenariosByTeam[teamID]
		cost.ScenarioCoverage = len(cost.Scenarios)
		report.Teams = append(report.Teams, cost)
	}
	return report, nil
}

func orientationComposite(c OrientationComponents) int {
	return c.Members*orientationWeightMember +
		c.Topics*orientationWeightTopic +
		c.DecisionContexts*orientationWeightDecisionContext +
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
func canonLineCount(repoRoot string, contract *LoadedTeamContract) (int, []string) {
	if contract == nil {
		return 0, nil
	}
	total := 0
	var missing []string
	seen := map[string]bool{}
	for _, doc := range contract.PlanOfRecordDocuments {
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
			data, err := os.ReadFile(filepath.Join(repoRoot, path))
			if err != nil {
				missing = append(missing, path)
				continue
			}
			total += strings.Count(string(data), "\n") + 1
		}
	}
	sort.Strings(missing)
	return total, missing
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
