// Team-contract loader and registry. Memberflow validation rules that need
// to cross-check declarations against team-level state (decision contexts,
// knowledge-topic registries, etc.) consult this registry rather than
// reaching into the broader store package — it keeps memberflow a leaf
// dependency and lets validation tests fixture team contracts directly.
//
// Today the only consumer is ruleDanglingEvidenceDecision, which
// verifies that every `evidence_consumed[].for_decisions[]` id on every
// member's topics.json resolves against some team's
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
//
// The runtime-attribution Pillar 3 fields (AttributionValidFrom,
// FlagExternalWritesPerWeek) are projected from the team.json top-level
// shape (`attributionValidFrom`, `policy.flagExternalWritesPerWeek`). They
// are kept as direct fields rather than nested under a Policy struct so
// rules consume them without per-rule nil checks; absence on disk is
// canonicalized to zero values here. See docs/agent-system/RUNTIME_ATTRIBUTION.md
// § Per-team `attributionValidFrom` for the contract.
type LoadedTeamContract struct {
	TeamID     string
	Contract   *teamcontract.OperatingContract
	SourcePath string
	// TopicCatalog is team-level metadata for topic families: status and
	// purpose text shared by every member relationship that touches a prefix.
	// Member topics.json files remain the source of per-member read/write
	// relationships; this catalog supplies the common family description.
	TopicCatalog []TopicCatalogEntry
	// PlanOfRecordDocuments is the team's plan-of-record registry projected
	// from team.json::operatingContract.documents.planOfRecord.
	PlanOfRecordDocuments []teamcontract.PlanOfRecordDocument
	// AttributionValidFrom is the ISO-8601 (YYYY-MM-DD) cutoff for
	// Pillar 3 attribution checks. Empty when the team has not adopted
	// the runtime contract; ruleActualWriterUndeclared skips such teams
	// entirely so a still-migrating team never produces drift findings.
	AttributionValidFrom string
	// FlagExternalWritesPerWeek is the per-team opt-in threshold for
	// kind="external" knowledge writes, sourced from
	// `policy.flagExternalWritesPerWeek` on team.json. Zero means "track
	// but never flag" (the expected default); positive values enable
	// per-ISO-week count-vs-threshold findings on
	// ruleActualWriterUndeclared.
	FlagExternalWritesPerWeek int
	// RoleIDs are the role ids declared in the team's sibling roles.json,
	// in file order. Nil when the team declares no roles.json at all, which
	// ruleTeamRoleMemberDrift treats as "not adopted" rather than as drift.
	RoleIDs []string
	// RolesSourcePath is the roles.json path findings attribute to. Empty
	// when the team declares no roles.json.
	RolesSourcePath string
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
//
// Pillar 3 fields (AttributionValidFrom and the nested Policy) are
// optional on disk; they are zero-valued when omitted. The struct mirrors
// the on-disk JSON shape; LoadedTeamContract flattens it for validators.
type teamFile struct {
	ID                   string                          `json:"id"`
	OperatingContract    *teamcontract.OperatingContract `json:"operatingContract"`
	TopicCatalog         []TopicCatalogEntry             `json:"topicCatalog,omitempty"`
	AttributionValidFrom string                          `json:"attributionValidFrom,omitempty"`
	Policy               *teamFilePolicy                 `json:"policy,omitempty"`
}

// TopicCatalogEntry is the structured source of truth for a team-level topic
// family's status and purpose. It deliberately omits readers/writers; those
// live in member topics.json files and in the operating graph contract.
type TopicCatalogEntry struct {
	Prefix    string `json:"prefix"`
	Qualifier string `json:"qualifier,omitempty"`
	Status    string `json:"status"`
	Purpose   string `json:"purpose"`
}

// teamFilePolicy mirrors store.TeamPolicy under memberflow's narrower
// JSON-only view of team.json. Kept in sync with store.TeamPolicy by the
// runtime_attribution_test.go drift-detector test.
type teamFilePolicy struct {
	FlagExternalWritesPerWeek int `json:"flagExternalWritesPerWeek,omitempty"`
}

// LoadAllTeamContracts walks <configDir>/teams/*/team.json and returns a
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
func LoadAllTeamContracts(configDir string) (TeamContractRegistry, error) {
	out := make(TeamContractRegistry)
	if strings.TrimSpace(configDir) == "" {
		return out, nil
	}
	teamsDir := filepath.Join(configDir, "teams")
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
	if err := ValidateTopicCatalog(tf.TopicCatalog); err != nil {
		return nil, fmt.Errorf("team %q has invalid topicCatalog: %w", path, err)
	}
	// operatingContract may legitimately be nil while a team is being
	// scaffolded; the validator treats that as "no decision contexts to
	// match," which is preferable to crashing. The dangling-evidence rule
	// will then report unresolved references in detail.
	flag := 0
	if tf.Policy != nil {
		flag = tf.Policy.FlagExternalWritesPerWeek
	}
	roleIDs, rolesPath, err := loadTeamRoleIDs(filepath.Join(filepath.Dir(path), "roles.json"))
	if err != nil {
		return nil, err
	}
	return &LoadedTeamContract{
		TeamID:                    tf.ID,
		Contract:                  tf.OperatingContract,
		SourcePath:                path,
		TopicCatalog:              tf.TopicCatalog,
		PlanOfRecordDocuments:     operatingContractPlanOfRecord(tf.OperatingContract),
		AttributionValidFrom:      strings.TrimSpace(tf.AttributionValidFrom),
		FlagExternalWritesPerWeek: flag,
		RoleIDs:                   roleIDs,
		RolesSourcePath:           rolesPath,
	}, nil
}

// rolesFile is the minimal slice of roles.json the parity rule reads.
type rolesFile struct {
	Roles []struct {
		ID string `json:"id"`
	} `json:"roles"`
}

// loadTeamRoleIDs reads a team's roles.json. A missing file returns nil ids
// and an empty path, which the parity rule reads as "this team has not adopted
// roles.json" rather than as drift.
func loadTeamRoleIDs(path string) ([]string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("read %q: %w", path, err)
	}
	var rf rolesFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, "", fmt.Errorf("parse %q: %w", path, err)
	}
	ids := make([]string, 0, len(rf.Roles))
	for _, role := range rf.Roles {
		if id := strings.TrimSpace(role.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, path, nil
}

func operatingContractPlanOfRecord(contract *teamcontract.OperatingContract) []teamcontract.PlanOfRecordDocument {
	if contract == nil {
		return nil
	}
	return append([]teamcontract.PlanOfRecordDocument(nil), contract.Documents.PlanOfRecord...)
}

// ValidateTopicCatalog checks topic-family metadata shape without consulting
// member runtime relationships. Relationship parity belongs to operating graph
// rules; this function only protects the team-level data model.
func ValidateTopicCatalog(entries []TopicCatalogEntry) error {
	seen := map[string]struct{}{}
	for i, entry := range entries {
		prefix := strings.TrimSpace(entry.Prefix)
		if prefix == "" {
			return fmt.Errorf("entry[%d].prefix is required", i)
		}
		if !validPrefix(prefix) {
			return fmt.Errorf("entry[%d].prefix %q is malformed", i, entry.Prefix)
		}
		status := ParseOperatingTopicCatalogStatus(entry.Status)
		if status == OperatingTopicStatusUnknown {
			return fmt.Errorf("entry[%d].status %q is unknown", i, entry.Status)
		}
		qualifier := strings.TrimSpace(entry.Qualifier)
		expectedQualifier, hasExpectedQualifier := expectedTopicCatalogQualifier(status)
		if qualifier != "" {
			if !hasExpectedQualifier {
				return fmt.Errorf("entry[%d].qualifier %q cannot be validated for status %q", i, qualifier, entry.Status)
			}
			if qualifier != expectedQualifier {
				return fmt.Errorf("entry[%d].status %q expects qualifier %q, got %q", i, entry.Status, expectedQualifier, qualifier)
			}
		} else {
			qualifier = expectedQualifier
		}
		if operatingTopicCatalogStatusIsCurrent(status) && strings.TrimSpace(entry.Purpose) == "" {
			return fmt.Errorf("entry[%d].purpose is required for current status %q", i, entry.Status)
		}
		key := qualifiedTopicKey(qualifier, prefix)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("entry[%d] duplicates topic catalog prefix %q", i, displayQualifiedTopic(qualifier, prefix))
		}
		seen[key] = struct{}{}
	}
	return nil
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

// HasTeamDecisionContext reports whether a specific team contract declares the
// given decision-context id. Empty/whitespace ids never match.
func (r TeamContractRegistry) HasTeamDecisionContext(teamID, id string) bool {
	teamID = strings.TrimSpace(teamID)
	id = strings.TrimSpace(id)
	if teamID == "" || id == "" {
		return false
	}
	lt := r[teamID]
	if lt == nil || lt.Contract == nil {
		return false
	}
	_, ok := lt.Contract.DecisionContext[id]
	return ok
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

// TopicCatalog returns the team-level topic catalog for a team. The returned
// slice is copied so callers can sort or filter without mutating the registry.
func (r TeamContractRegistry) TopicCatalog(teamID string) []TopicCatalogEntry {
	lt := r[strings.TrimSpace(teamID)]
	if lt == nil || len(lt.TopicCatalog) == 0 {
		return nil
	}
	out := append([]TopicCatalogEntry(nil), lt.TopicCatalog...)
	return out
}

func (r TeamContractRegistry) HasPlanOfRecordPath(teamID, path string) bool {
	lt := r[strings.TrimSpace(teamID)]
	if lt == nil {
		return false
	}
	path = filepath.ToSlash(strings.TrimSpace(path))
	for _, doc := range lt.PlanOfRecordDocuments {
		if documentRefPathMatches(doc.Hub, path) {
			return true
		}
		for _, ref := range doc.Paths {
			ref := ref
			if documentRefPathMatches(&ref, path) {
				return true
			}
		}
	}
	return false
}

func documentRefPathMatches(ref *teamcontract.PathRef, path string) bool {
	if ref == nil {
		return false
	}
	return ref.Base == "repo-root" && filepath.ToSlash(strings.TrimSpace(ref.Path)) == path
}
