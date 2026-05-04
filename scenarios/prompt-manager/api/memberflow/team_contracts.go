// Team-contract loader and registry. Memberflow validation rules that need
// to cross-check declarations against team-level state (decision contexts,
// knowledge-topic registries, etc.) consult this registry rather than
// reaching into the broader store package — it keeps memberflow a leaf
// dependency and lets validation tests fixture team contracts directly.
//
// Today the only consumer is ruleDanglingEvidenceDecision (Phase 1.2),
// which verifies that every `evidence_consumed[].for_decisions[]` id on
// every member's topics.json resolves against some team's
// `team.json::operatingContract.decisionContexts`.
//
// Loader semantics mirror LoadAllTaxonomies (taxonomy.go): a missing
// teams/ directory yields an empty registry without error so that callers
// can treat the absence as "skip the cross-check" rather than as failure.
package memberflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"prompt-manager/teamcontract"
)

// LoadedTeamContract pairs a team id with its parsed operating contract
// and the source path the contract was read from. Used by validation rules
// that need to attribute findings back to the originating team.json.
type LoadedTeamContract struct {
	TeamID     string
	Contract   *teamcontract.OperatingContract
	SourcePath string
}

// TeamContractRegistry indexes loaded contracts by team id. Methods on the
// registry are read-only convenience wrappers around the underlying map;
// callers may iterate the map directly when they need full traversal.
//
// A nil registry is a valid empty registry (every cross-check method
// reports "not found"); validation rules that depend on the registry
// gracefully skip when it is empty.
type TeamContractRegistry map[string]*LoadedTeamContract

// teamFile is a minimal slice of team.json — only the fields the
// memberflow validators need. Decoding into this rather than store.Team
// keeps memberflow free of store dependencies.
type teamFile struct {
	ID                string                          `json:"id"`
	OperatingContract *teamcontract.OperatingContract `json:"operatingContract"`
}

// LoadAllTeamContracts walks <storeDir>/teams/*/team.json and returns a
// registry of every parseable team contract, indexed by team id. The
// `id` field on the file (not the directory name) is authoritative.
//
// Returns an empty registry without error when teams/ is absent so callers
// can treat that as "no contracts to cross-check," matching the
// LoadAllTaxonomies / LoadSkillIDs convention.
//
// Errors only when teams/ exists but a specific team.json fails to parse
// or omits a required field — partial registries are not returned, so the
// caller gets either every team or an explicit failure.
func LoadAllTeamContracts(storeDir string) (TeamContractRegistry, error) {
	out := make(TeamContractRegistry)
	if strings.TrimSpace(storeDir) == "" {
		return out, nil
	}
	teamsDir := filepath.Join(storeDir, "teams")
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("memberflow: read teams dir %q: %w", teamsDir, err)
	}
	for _, te := range entries {
		if !te.IsDir() || strings.HasPrefix(te.Name(), ".") {
			continue
		}
		teamJSONPath := filepath.Join(teamsDir, te.Name(), "team.json")
		if _, err := os.Stat(teamJSONPath); err != nil {
			if os.IsNotExist(err) {
				// A team directory without team.json is a partial
				// scaffold (e.g., mid-creation); skip rather than
				// reject the entire registry.
				continue
			}
			return nil, fmt.Errorf("memberflow: stat %q: %w", teamJSONPath, err)
		}
		entry, err := parseTeamFile(teamJSONPath)
		if err != nil {
			return nil, fmt.Errorf("memberflow: %w", err)
		}
		if entry == nil {
			continue
		}
		if existing, ok := out[entry.TeamID]; ok {
			return nil, fmt.Errorf("memberflow: duplicate team id %q in %q and %q", entry.TeamID, existing.SourcePath, entry.SourcePath)
		}
		out[entry.TeamID] = entry
	}
	return out, nil
}

func parseTeamFile(path string) (*LoadedTeamContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	var tf teamFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}
	if strings.TrimSpace(tf.ID) == "" {
		return nil, fmt.Errorf("team %q is missing required id field", path)
	}
	// operatingContract may legitimately be nil while a team is being
	// scaffolded; the validator treats that as "no decision contexts to
	// match," which is preferable to crashing. The dangling-evidence rule
	// will then report unresolved references in detail.
	return &LoadedTeamContract{
		TeamID:     tf.ID,
		Contract:   tf.OperatingContract,
		SourcePath: path,
	}, nil
}

// HasDecisionContext reports whether any team in the registry declares the
// given decision-context id. Empty/whitespace ids never match.
func (r TeamContractRegistry) HasDecisionContext(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, lt := range r {
		if lt == nil || lt.Contract == nil {
			continue
		}
		if _, ok := lt.Contract.DecisionContext[id]; ok {
			return true
		}
	}
	return false
}

// TeamsForDecisionContext returns every team id whose contract declares
// the given decision-context id, in lexical order. Useful for diagnostic
// detail strings that point the operator at the right team.json.
func (r TeamContractRegistry) TeamsForDecisionContext(id string) []string {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	var teams []string
	for teamID, lt := range r {
		if lt == nil || lt.Contract == nil {
			continue
		}
		if _, ok := lt.Contract.DecisionContext[id]; ok {
			teams = append(teams, teamID)
		}
	}
	sort.Strings(teams)
	return teams
}

// IDs returns every loaded team id in lexical order.
func (r TeamContractRegistry) IDs() []string {
	ids := make([]string, 0, len(r))
	for id := range r {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
