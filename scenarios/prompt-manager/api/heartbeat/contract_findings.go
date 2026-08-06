// Contract findings — the return path from validation to the member that
// caused it.
//
// Declaration defects used to reach only the operator, through a manual
// `prompt-manager graph topics` sweep. Nothing put them in front of the agent
// whose declarations were wrong, so a member could repeat the same defect on
// every heartbeat indefinitely: monetization-contrarian wrote its snapshot
// topic as `contrarian-scan-<date>` — a hyphen where the prefix needs a path
// separator — on twenty consecutive runs while writing the correct form four
// other times. Every one of those runs was validated; none of them told the
// agent.
//
// This section closes that loop. It renders only findings attributed to the
// running member, and renders nothing at all when there are none, so a clean
// member's prompt does not grow.
//
// Attribution, not an allow-list, decides what appears here. A rule that
// names a member reaches that member automatically; adding a rule does not
// mean editing this file.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md
package heartbeat

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"prompt-manager/finding"

	"prompt-manager/memberflow"
	"prompt-manager/store"
)

// ContractFinding is one open validation finding against a member's own
// declarations, already collapsed to one entry per defect by
// memberflow's per-defect grouping.
// ContractFinding is the same type every other validation surface produces.
// It was a separate four-field shape whose Severity was a bare string, so a
// contract defect arriving here lost its catalogued identity on the way in.
type ContractFinding = finding.Finding

// ContractFindingsProvider supplies the open findings attributed to one
// member. Implementations are expected to be cheap to call once per prompt
// build; MemberflowContractFindings caches for that reason.
type ContractFindingsProvider interface {
	MemberContractFindings(ctx context.Context, teamID, memberID string) ([]ContractFinding, error)
}

// SetContractFindingsProvider wires the findings source. When unset, the
// prompt omits the section entirely rather than claiming a clean contract it
// has not checked — an unwired builder and a clean member must not look the
// same to a reader of the prompt.
func (b *PromptBuilder) SetContractFindingsProvider(provider ContractFindingsProvider) {
	if b == nil {
		return
	}
	b.contractFindings = provider
}

func (b *PromptBuilder) buildContractFindingsSection(ctx context.Context, teamID, agentID string) string {
	if b == nil || b.contractFindings == nil {
		return ""
	}
	findings, err := b.contractFindings.MemberContractFindings(ctx, teamID, agentID)
	if err != nil || len(findings) == 0 {
		return ""
	}
	return renderContractFindings(teamID, findings)
}

func renderContractFindings(teamID string, findings []ContractFinding) string {
	if len(findings) == 0 {
		return ""
	}
	sorted := make([]ContractFinding, len(findings))
	copy(sorted, findings)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Severity != sorted[j].Severity {
			// Errors first; both values are known, so a simple compare is
			// enough to keep "error" ahead of "warning".
			return sorted[i].Severity == finding.SeverityError
		}
		if sorted[i].Rule != sorted[j].Rule {
			return sorted[i].Rule < sorted[j].Rule
		}
		return sorted[i].Prefix < sorted[j].Prefix
	})

	var b strings.Builder
	b.WriteString(promptHeading(promptSectionKindContractFindings) + "\n\n")
	b.WriteString(fmt.Sprintf("Validation found %s open against your own declarations. ", pluralItems(len(sorted))))
	b.WriteString("These describe a gap between what you actually write and what your `topics.json` and member documents say you write. They are not this run's task, and clearing them is not a substitute for it.\n\n")

	for _, finding := range sorted {
		b.WriteString(fmt.Sprintf("- **%s** `%s`", strings.ToLower(string(finding.Severity)), finding.Rule))
		if finding.Prefix != "" {
			b.WriteString(fmt.Sprintf(" — `%s`", finding.Prefix))
		}
		b.WriteString("\n")
		if detail := strings.TrimSpace(finding.Detail); detail != "" {
			b.WriteString("  " + detail + "\n")
		}
	}

	b.WriteString("\n## What to do with these\n\n")
	b.WriteString("Take the smallest correcting action your contract already allows:\n\n")
	b.WriteString("1. If the finding says you wrote outside your declared prefixes, write to a prefix you already declare for the rest of this run.\n")
	b.WriteString("2. If the declaration itself is wrong, propose the change as a decision in one of your owned contexts. You do not edit `topics.json` or your own member contract directly.\n")
	b.WriteString("3. If neither fits — the right prefix does not exist, or the rule looks wrong — report it as friction rather than working around it silently.\n\n")
	b.WriteString(fmt.Sprintf("Full detail, including findings owned by your teammates: `prompt-manager graph topics --team %s`.\n", teamID))

	return strings.TrimRight(b.String(), "\n")
}

func pluralItems(n int) string {
	if n == 1 {
		return "1 item"
	}
	return fmt.Sprintf("%d items", n)
}

// MemberflowContractFindings adapts memberflow validation to the provider
// interface.
//
// Validation is a whole-corpus pass — cross-team prefixes have to resolve
// against every member before any single member's findings are known — so it
// is computed once and shared. The TTL exists for `team prompt-matrix`, which
// builds all 27 members back to back and would otherwise trigger 27 identical
// corpus scans. Heartbeats run far enough apart that a reading up to one TTL
// old is still the current one.
type MemberflowContractFindings struct {
	StoreDir       string
	RepoRoot       string
	RuntimeDataDir string
	TTL            time.Duration

	mu       sync.Mutex
	cachedAt time.Time
	cached   map[string][]ContractFinding
}

const defaultContractFindingsTTL = 60 * time.Second

// MemberContractFindings returns the findings attributed to one member.
func (m *MemberflowContractFindings) MemberContractFindings(_ context.Context, teamID, memberID string) ([]ContractFinding, error) {
	if m == nil {
		return nil, nil
	}
	byMember, err := m.snapshot()
	if err != nil {
		return nil, err
	}
	return byMember[memberflow.MemberRef{Team: teamID, Member: memberID}.String()], nil
}

func (m *MemberflowContractFindings) snapshot() (map[string][]ContractFinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ttl := m.TTL
	if ttl <= 0 {
		ttl = defaultContractFindingsTTL
	}
	if m.cached != nil && time.Since(m.cachedAt) < ttl {
		return m.cached, nil
	}

	members, err := memberflow.LoadAll(m.StoreDir)
	if err != nil {
		return nil, err
	}
	skillIDs, _ := memberflow.LoadSkillIDs(m.StoreDir)
	taxonomies, _ := memberflow.LoadAllTaxonomies(m.RepoRoot)
	result := memberflow.Validate(members, memberflow.ValidationOptions{
		RepoRoot:       m.RepoRoot,
		StoreDir:       m.StoreDir,
		RuntimeDataDir: m.RuntimeDataDir,
		SkillIDs:       skillIDs,
		Taxonomies:     taxonomies,
	})

	byMember := filterActionableFindings(result.Findings)
	teamStore := store.NewFileTeamStore(m.StoreDir, m.RuntimeDataDir, nil)
	if teams, err := teamStore.List(context.Background()); err == nil {
		mergeTeamValidationFindings(byMember, teams)
	}
	m.cached = byMember
	m.cachedAt = time.Now()
	return byMember, nil
}

// mergeTeamValidationFindings routes readable-but-invalid team contracts back
// to their members. Member-specific fields are delivered only to that member;
// a team-wide configuration defect is delivered to every declared contract
// member because each prompt is otherwise capable of claiming a clean
// operating policy while the team configuration is invalid.
func mergeTeamValidationFindings(byMember map[string][]ContractFinding, teams []store.Team) {
	for _, team := range teams {
		if len(team.ValidationFindings) == 0 || team.OperatingContract == nil {
			continue
		}
		for _, entry := range team.ValidationFindings {
			memberID := memberIDForTeamFinding(entry.Field)
			for id := range team.OperatingContract.Members {
				if memberID != "" && id != memberID {
					continue
				}
				key := memberflow.MemberRef{Team: team.ID, Member: id}.String()
				// The rule id comes from the contract catalog when the
				// producing family has one. Synthesizing "team_<source>_invalid"
				// here is what made a malformed team.json uncatalogued and
				// therefore unrankable and undocumentable.
				rule := entry.Rule
				if rule == "" {
					rule = "team_" + entry.Source + "_invalid"
				}
				byMember[key] = append(byMember[key], ContractFinding{
					Rule:     rule,
					Severity: finding.SeverityError,
					Kind:     finding.KindDeclaration,
					Team:     team.ID,
					Member:   id,
					Path:     entry.Field,
					Detail:   entry.Message,
				})
			}
		}
	}
}

func memberIDForTeamFinding(field string) string {
	const prefix = "operatingContract.members."
	if !strings.HasPrefix(field, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(field, prefix)
	memberID, _, _ := strings.Cut(rest, ".")
	return strings.TrimSpace(memberID)
}

// filterActionableFindings keeps the findings a specific member can act on and
// buckets them by `team/member`.
//
// Two things are dropped. Advisory findings come from heuristics that cannot
// separate a real defect from a lookalike; asking a member to correct one
// spends its judgment on the validator's mistake. Unattributed findings belong
// to the team or the corpus, and choosing a member to hand them to would be a
// guess — a prompt is the wrong place to guess.
func filterActionableFindings(findings []memberflow.Finding) map[string][]ContractFinding {
	byMember := map[string][]ContractFinding{}
	for _, finding := range findings {
		if finding.Advisory {
			continue
		}
		key := strings.TrimSpace(finding.Subject())
		if key == "" || key == "/" || finding.Member == "" {
			continue
		}
		byMember[key] = append(byMember[key], ContractFinding{
			Rule:     finding.Rule,
			Severity: finding.Severity,
			Prefix:   finding.Prefix,
			Detail:   finding.Detail,
		})
	}
	return byMember
}
