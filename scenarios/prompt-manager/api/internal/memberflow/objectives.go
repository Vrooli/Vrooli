// Objective registry: the join between the operator's stated intent and the
// team roster.
//
// Why this file exists: every other relationship in the agent system is
// declared and validated — topic flows, cross-team outputs, operating-graph
// nodes, work types. The objective edge was the exception. It lived as
// prose on both ends: a `Served by` column in OBJECTIVES.md and a bolded
// sentence in six operating models. Editing an objective therefore fired
// nothing: no validator, no sweep target, no decision. The only reader was an
// agent performing a document read by judgment.
//
// The objective *set* stays operator-authored prose — that is deliberate and
// canon (OBJECTIVES.md is the only operator-specific layer in the system). What
// this file adds is a parse of that prose into ids, and a declared counterpart
// on team.json, so the join can be checked in both directions.
//
// DOC: docs/director-swarm/strategy/OBJECTIVES.md § The coverage rule
// DOC: docs/agent-system/FRAMEWORK_HEALTH.md § Objective coverage
package memberflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ObjectivesDocPath is the canon location of the objective set, relative to the
// repository root. The set is authored by the operator and is never written by
// this package.
const ObjectivesDocPath = "docs/director-swarm/strategy/OBJECTIVES.md"

// objectivesTableHeading is the section whose table carries the objective set.
// Parsing is anchored to the heading rather than to "the first table in the
// file" so that adding narrative tables above it cannot silently change what is
// parsed.
const objectivesTableHeading = "## The objectives"

// Objective role vocabulary. A team may declare a role only where the objective
// table itself distinguishes one; see ObjectiveTeamRef.Role.
const (
	ObjectiveRolePrimary    = "primary"
	ObjectiveRoleSupporting = "supporting"
)

// Objective coverage vocabulary.
const (
	ObjectiveCoverageFull    = "full"
	ObjectiveCoveragePartial = "partial"
)

// Objective classes.
const (
	ObjectiveClassTerminal     = "terminal"
	ObjectiveClassInstrumental = "instrumental"
)

// ObjectiveTeamRef is one (team, role) pair read from the objective table's
// `Served by` column, or declared on a team.json.
type ObjectiveTeamRef struct {
	TeamID string `json:"teamId"`
	// Role is "primary" or "supporting", or empty when the source does not
	// distinguish. Empty is a legitimate value, not a missing one: the table
	// qualifies roles for T1 and leaves I1/I2 unqualified, and inventing a
	// role where canon states none would assert something the operator did
	// not write.
	Role string `json:"role,omitempty"`
	// Coverage is "partial" when the source qualifies the contribution as
	// incomplete (T3's "OSS surface only"), otherwise "full".
	Coverage string `json:"coverage,omitempty"`
}

// Objective is one row of the objective table.
type Objective struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Class string `json:"class"`
	// ServedBy is the teams the table names. Empty when the table declares
	// the objective unserved.
	ServedBy []ObjectiveTeamRef `json:"servedBy,omitempty"`
	// GapMarker is the parenthesised marker the table carries in place of a
	// team (`pending-capability`, `pending-operator-input`). Its presence is
	// what separates a *declared* hole from an undeclared one — the coverage
	// rule treats the first as reported and the second as a finding.
	GapMarker string `json:"gapMarker,omitempty"`
	// EvidenceSource is the table's last column verbatim. An objective with
	// no evidence source cannot be scored; the rule set reports it.
	EvidenceSource string `json:"evidenceSource,omitempty"`
	// HasEvidence is false when the evidence cell declares none.
	HasEvidence bool `json:"hasEvidence"`
	Line        int  `json:"line"`
	// Revision is a short digest of this objective's *meaning* — title, class,
	// serving teams and evidence source, normalised. It exists because the
	// objective join was blind to the one change that matters most: editing an
	// objective's statement while keeping its id changed what every serving
	// team is obliged to do, and no surface asked whether the obligations still
	// followed. The id set, the team set and the evidence cell were all
	// checked; the words were not.
	//
	// The digest is computed, never authored. OBJECTIVES.md stays operator
	// prose and must not become a config file — so the version lives in the
	// parser and the *acknowledgement* lives on the team, which is the side
	// that has to act.
	Revision string `json:"revision,omitempty"`
}

// Unserved reports whether the table names no team for this objective.
func (o Objective) Unserved() bool { return len(o.ServedBy) == 0 }

// ObjectiveRegistry indexes the parsed objective set by id.
type ObjectiveRegistry struct {
	SourcePath string      `json:"sourcePath"`
	Objectives []Objective `json:"objectives"`
}

// Get returns the objective with the given id.
func (r ObjectiveRegistry) Get(id string) (Objective, bool) {
	for _, o := range r.Objectives {
		if strings.EqualFold(o.ID, id) {
			return o, true
		}
	}
	return Objective{}, false
}

// TeamsFor returns the team refs the table names for an objective.
func (r ObjectiveRegistry) TeamsFor(id string) []ObjectiveTeamRef {
	o, ok := r.Get(id)
	if !ok {
		return nil
	}
	return o.ServedBy
}

var (
	objectiveIDPattern   = regexp.MustCompile("`([TI][0-9]+)`")
	objectiveTeamPattern = regexp.MustCompile("`team:([a-z0-9][a-z0-9-]*)`\\s*(?:\\(([^)]*)\\))?")
	objectiveGapPattern  = regexp.MustCompile("`(pending-[a-z-]+)`")
	objectiveTitlePrefix = regexp.MustCompile(`^\*\*(.+?)\.?\*\*`)
)

// LoadObjectives parses the objective table out of OBJECTIVES.md.
//
// A missing document is not an error: it returns an empty registry, matching
// the LoadAllTeamContracts convention, so a checkout without the director-swarm
// plan of record validates rather than crashes.
func LoadObjectives(repoRoot string) (ObjectiveRegistry, error) {
	reg := ObjectiveRegistry{SourcePath: ObjectivesDocPath}
	if strings.TrimSpace(repoRoot) == "" {
		return reg, nil
	}
	path := filepath.Join(repoRoot, ObjectivesDocPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return reg, nil
		}
		return reg, fmt.Errorf("memberflow: read %q: %w", path, err)
	}
	reg.Objectives = parseObjectiveTable(strings.Split(string(data), "\n"))
	return reg, nil
}

// parseObjectiveTable walks the lines of OBJECTIVES.md and reads the table
// under objectivesTableHeading. Rows that carry no id in the first cell are
// skipped, which covers the header and separator rows without matching on
// their exact text.
func parseObjectiveTable(lines []string) []Objective {
	var out []Objective
	inSection := false
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			inSection = strings.EqualFold(line, objectivesTableHeading)
			continue
		}
		if !inSection || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitMarkdownRow(line)
		if len(cells) < 4 {
			continue
		}
		ids := objectiveIDPattern.FindStringSubmatch(cells[0])
		if ids == nil {
			continue
		}
		obj := Objective{ID: ids[1], Line: i + 1}
		if m := objectiveTitlePrefix.FindStringSubmatch(strings.TrimSpace(cells[1])); m != nil {
			obj.Title = strings.TrimSpace(m[1])
		} else {
			obj.Title = strings.TrimSpace(cells[1])
		}
		obj.Class = strings.ToLower(strings.TrimSpace(cells[2]))
		obj.ServedBy = parseObjectiveTeams(cells[3])
		if len(obj.ServedBy) == 0 {
			if g := objectiveGapPattern.FindStringSubmatch(cells[3]); g != nil {
				obj.GapMarker = g[1]
			}
		}
		if len(cells) > 4 {
			obj.EvidenceSource = strings.TrimSpace(cells[4])
			obj.HasEvidence = !objectiveCellDeclaresNone(cells[4])
		}
		obj.Revision = objectiveRevision(obj)
		out = append(out, obj)
	}
	return out
}

// objectiveRevision digests the parts of an objective row that change what a
// serving team is obliged to do.
//
// The line number is deliberately excluded: moving a row up the table does not
// change what it asks of anyone, and a digest that churned on reordering would
// train teams to re-acknowledge without reading. Whitespace is normalised for
// the same reason.
func objectiveRevision(obj Objective) string {
	teams := make([]string, 0, len(obj.ServedBy))
	for _, ref := range obj.ServedBy {
		teams = append(teams, strings.ToLower(strings.TrimSpace(ref.TeamID+"/"+ref.Role+"/"+ref.Coverage)))
	}
	sort.Strings(teams)
	parts := []string{
		strings.ToUpper(strings.TrimSpace(obj.ID)),
		normalizeObjectiveText(obj.Title),
		strings.ToLower(strings.TrimSpace(obj.Class)),
		strings.Join(teams, ","),
		strings.ToLower(strings.TrimSpace(obj.GapMarker)),
		normalizeObjectiveText(obj.EvidenceSource),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])[:12]
}

// normalizeObjectiveText collapses runs of whitespace so a reflowed line does
// not read as a changed objective.
func normalizeObjectiveText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

// parseObjectiveTeams reads the `Served by` cell. Each `team:<id>` reference
// may carry a parenthesised qualifier; the qualifier supplies the role when it
// names one and the coverage when it says "partial".
func parseObjectiveTeams(cell string) []ObjectiveTeamRef {
	var out []ObjectiveTeamRef
	for _, m := range objectiveTeamPattern.FindAllStringSubmatch(cell, -1) {
		ref := ObjectiveTeamRef{TeamID: m[1], Coverage: ObjectiveCoverageFull}
		qualifier := strings.ToLower(strings.TrimSpace(m[2]))
		switch {
		case strings.HasPrefix(qualifier, ObjectiveRolePrimary):
			ref.Role = ObjectiveRolePrimary
		case strings.HasPrefix(qualifier, ObjectiveRoleSupporting):
			ref.Role = ObjectiveRoleSupporting
		}
		if strings.Contains(qualifier, ObjectiveCoveragePartial) {
			ref.Coverage = ObjectiveCoveragePartial
		}
		out = append(out, ref)
	}
	return out
}

// objectiveCellDeclaresNone reports whether a table cell says "none" rather
// than naming a value. The table writes this as `*none*`, optionally followed
// by a parenthesised gap marker, so emphasis markers are stripped before the
// comparison rather than trimmed from the ends.
func objectiveCellDeclaresNone(cell string) bool {
	normalized := strings.ToLower(strings.TrimSpace(cell))
	normalized = strings.NewReplacer("*", "", "_", "").Replace(normalized)
	return strings.HasPrefix(strings.TrimSpace(normalized), "none")
}

// splitMarkdownRow splits a pipe-delimited table row into trimmed cells,
// dropping the empty leading and trailing fields the delimiters produce.
func splitMarkdownRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// TeamObjectiveDeclaration is the `objectivesServed` block on a team.json —
// the upward half of the join.
type TeamObjectiveDeclaration struct {
	ID       string `json:"id"`
	Role     string `json:"role,omitempty"`
	Coverage string `json:"coverage,omitempty"`
	// Note carries any qualifier the team wants recorded. It is never
	// validated; it exists so a team can say *why* its coverage is partial
	// without that reason having to live in prose the validator cannot see.
	Note string `json:"note,omitempty"`
	// AcknowledgedRevision is the objective revision this team last confirmed
	// its obligation list follows from. When it differs from the objective's
	// current revision — including when it is absent, which is the state every
	// team starts in — the team carries `objective_restatement_pending` until
	// its contrarian re-derives the obligations and records the new value.
	//
	// It is the actuator for the slowest loop in the target model: intent
	// changed, so the setpoint derived from it has to be re-derived. Nothing
	// else in the system fires on that event.
	AcknowledgedRevision string `json:"acknowledgedRevision,omitempty"`
}

// teamObjectivesFile is the minimal slice of team.json this parser reads.
type teamObjectivesFile struct {
	ID                string                     `json:"id"`
	ObjectivesServed  []TeamObjectiveDeclaration `json:"objectivesServed,omitempty"`
	OperatingContract json.RawMessage            `json:"operatingContract,omitempty"`
}

// LoadTeamObjectives reads every team.json's objectivesServed block, keyed by
// team id. Teams that declare none map to a nil slice, which the rules treat as
// "undeclared" rather than as "serves nothing".
func LoadTeamObjectives(configDir string) (map[string][]TeamObjectiveDeclaration, map[string]string, error) {
	declared := map[string][]TeamObjectiveDeclaration{}
	paths := map[string]string{}
	if strings.TrimSpace(configDir) == "" {
		return declared, paths, nil
	}
	teamsDir := filepath.Join(configDir, "teams")
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return declared, paths, nil
		}
		return nil, nil, fmt.Errorf("memberflow: read teams dir %q: %w", teamsDir, err)
	}
	for _, te := range entries {
		if !te.IsDir() || strings.HasPrefix(te.Name(), ".") {
			continue
		}
		path := filepath.Join(teamsDir, te.Name(), "team.json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("memberflow: read %q: %w", path, err)
		}
		var tf teamObjectivesFile
		if err := json.Unmarshal(data, &tf); err != nil {
			return nil, nil, fmt.Errorf("memberflow: parse %q: %w", path, err)
		}
		id := strings.TrimSpace(tf.ID)
		if id == "" {
			continue
		}
		declared[id] = tf.ObjectivesServed
		paths[id] = path
	}
	return declared, paths, nil
}

// objectiveProsePattern finds the ids a team's operating model claims in its
// "Objective served" paragraph.
var objectiveProseHeading = regexp.MustCompile(`(?i)^\*\*objective served\.?\*\*`)

// ProseObjectiveIDs extracts the objective ids named in an operating model's
// **Objective served.** paragraph. It returns nil when the paragraph is absent,
// which the rules distinguish from a paragraph that names no ids.
func ProseObjectiveIDs(body []string) ([]string, bool) {
	for _, line := range body {
		trimmed := strings.TrimSpace(line)
		if !objectiveProseHeading.MatchString(trimmed) {
			continue
		}
		var ids []string
		seen := map[string]bool{}
		for _, m := range objectiveIDPattern.FindAllStringSubmatch(trimmed, -1) {
			if seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
		return ids, true
	}
	return nil, false
}
