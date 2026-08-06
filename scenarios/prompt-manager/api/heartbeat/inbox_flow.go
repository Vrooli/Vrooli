// Inbox Flow heartbeat section.
//
// This file generates the "Inbox Flow" section that appears in a member's
// heartbeat prompt when the member declares one or more `intake[]` entries
// in topics.json. The section is generated from structural data (topics.json
// + the taxonomy JSON sidecar). Skills, RESPONSIBILITIES.md, and
// HEARTBEAT.md must NOT duplicate any content rendered here.
//
// DOC: docs/agent-system/INTAKE_PIPELINE.md
package heartbeat

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"prompt-manager/memberflow"
)

// inboxFlowInputs is the deterministic input bundle for RenderInboxFlow.
// All I/O happens in the caller (LoadInboxFlowInputs); the renderer is
// pure so snapshot tests can run without filesystem access.
type inboxFlowInputs struct {
	teamID     string
	agentID    string
	memberFlow memberflow.MemberTopics
	taxonomies map[string]*memberflow.Taxonomy
}

// LoadInboxFlowInputs reads the member's topics.json from configDir and the
// taxonomy registry from repoRoot, returning everything RenderInboxFlow
// needs. Returns ok=false (without error) when the member has no intake;
// callers should skip the section in that case.
func LoadInboxFlowInputs(configDir, repoRoot, teamID, agentID string) (*inboxFlowInputs, bool, error) {
	mt, err := memberflow.LoadMember(configDir, teamID, agentID)
	if err != nil {
		return nil, false, fmt.Errorf("load topics.json: %w", err)
	}
	if len(mt.Topics.Intake) == 0 {
		return nil, false, nil
	}
	registry, err := memberflow.LoadAllTaxonomies(repoRoot)
	if err != nil {
		return nil, false, fmt.Errorf("load taxonomies: %w", err)
	}
	return &inboxFlowInputs{
		teamID:     teamID,
		agentID:    agentID,
		memberFlow: mt,
		taxonomies: registry,
	}, true, nil
}

// RenderInboxFlow returns the markdown for the heartbeat's Inbox Flow
// section. Returns empty string when the member declares no intake; callers
// should skip the section in that case.
//
// The renderer is pure (no I/O). Tests inject the loaded structures.
func RenderInboxFlow(in *inboxFlowInputs) string {
	if in == nil || len(in.memberFlow.Topics.Intake) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(promptHeading(promptSectionKindInboxFlow) + "\n\n")
	b.WriteString("You drain one or more team-knowledge inboxes. The mechanics, destinations, decisions, and dispatch below are generated from your topics.json and the taxonomies it cites. Do not paraphrase from memory; the generated text is the source of truth.\n")

	for _, intake := range in.memberFlow.Topics.Intake {
		renderInboxBlock(&b, in, intake)
	}

	renderUniversalDrainProcedure(&b, in.teamID, &in.memberFlow.Topics)
	renderDestinationsBlock(&b, in)
	renderDecisionsBlock(&b, &in.memberFlow.Topics)
	renderDispatchBlock(&b, in)

	return strings.TrimRight(b.String(), "\n")
}

func renderInboxBlock(b *strings.Builder, in *inboxFlowInputs, intake memberflow.IntakeEntry) {
	b.WriteString("\n## Inbox: `" + intake.Prefix + "`\n\n")
	b.WriteString("| | |\n|---|---|\n")
	b.WriteString("| Team | `" + in.teamID + "` |\n")
	b.WriteString("| Prefix | `" + intake.Prefix + "` |\n")

	taxonomyID := strings.TrimSpace(intake.Taxonomy)
	if taxonomyID == "" {
		b.WriteString("| Taxonomy | _none declared_ (transitional; set `intake[].taxonomy` in topics.json) |\n")
	} else if tx, ok := in.taxonomies[taxonomyID]; ok && tx != nil {
		porPath := tx.PoRPath
		if strings.TrimSpace(porPath) == "" {
			porPath = derivePoRPath(tx.SourcePath)
		}
		if strings.TrimSpace(porPath) == "" {
			b.WriteString(fmt.Sprintf("| Taxonomy | `%s` |\n", taxonomyID))
		} else {
			b.WriteString(fmt.Sprintf("| Taxonomy | `%s` (PoR: `%s`) |\n", taxonomyID, porPath))
		}
	} else {
		b.WriteString(fmt.Sprintf("| Taxonomy | `%s` _(NOT FOUND in registry — fix `intake[].taxonomy`)_ |\n", taxonomyID))
	}

	if classifier := strings.TrimSpace(intake.ClassifierSkill); classifier != "" {
		b.WriteString(fmt.Sprintf("| Classifier | `%s` (load with `prompt-manager skill read %s`) |\n", classifier, classifier))
	} else {
		b.WriteString("| Classifier | _none_ — the topic prefix is taken as the deterministic signal-type |\n")
	}
	if intake.SourceTeam != nil && strings.TrimSpace(*intake.SourceTeam) != "" {
		if *intake.SourceTeam == memberflow.SourceTeamWildcard {
			b.WriteString("| Source team | `*` (universal — any team's members may write) |\n")
		} else {
			b.WriteString(fmt.Sprintf("| Source team | `%s` |\n", *intake.SourceTeam))
		}
	}

	b.WriteString("\nView unrouted entries:\n")
	b.WriteString("```bash\n")
	b.WriteString(fmt.Sprintf("prompt-manager team knowledge-list %s --topic-prefix=%s\n", in.teamID, prefixForList(intake.Prefix)))
	b.WriteString(fmt.Sprintf("prompt-manager team knowledge-list %s --topic-prefix=%s<signal-type>/\n", in.teamID, prefixForList(intake.Prefix)))
	b.WriteString("```\n")
}

func renderUniversalDrainProcedure(b *strings.Builder, teamID string, topics *memberflow.Topics) {
	if len(topics.ExternalProducers) > 0 {
		b.WriteString("\nExternal producers that may write into your inbox(es): ")
		b.WriteString("`" + strings.Join(topics.ExternalProducers, "`, `") + "`.\n")
	}
	b.WriteString("\n## Drain procedure (universal)\n\n")
	b.WriteString("For each entry:\n\n")
	b.WriteString("1. If a classifier is declared, apply it to identify `signal_type`, `evidence_strength`, and `honesty_flags`. If no classifier is declared, take the topic-prefix segment after the inbox prefix as the signal_type.\n")
	b.WriteString("2. Choose the smallest useful action from the taxonomy's `actionSelection` set:\n")
	b.WriteString(fmt.Sprintf("   - **drop** — `prompt-manager team knowledge-delete %s <id>`\n", teamID))
	b.WriteString(fmt.Sprintf("   - **observe** — `prompt-manager team knowledge-update %s <id> --topic=\"<destination-prefix>\"`\n", teamID))
	b.WriteString("   - **promote-to-canon** — same as observe; pair with a decision when evidence converges\n")
	b.WriteString("   - **file-decision** — raise an owned context; delete the inbox row if the artifact lives elsewhere\n")
	b.WriteString("   - **capability-gap** — file `capability-gap` decision; leave the inbox entry until the gap is closed\n")
	b.WriteString("3. After routing, the entry must no longer carry an inbox topic-prefix. The inbox view is the unrouted set.\n")
}

func renderDestinationsBlock(b *strings.Builder, in *inboxFlowInputs) {
	outputs := in.memberFlow.Topics.Output
	if len(outputs) == 0 {
		return
	}
	b.WriteString("\n## Destinations\n\n")
	b.WriteString("| Prefix | Kind | Schema | Cross-team to |\n|---|---|---|---|\n")
	for _, o := range outputs {
		schema := strings.TrimSpace(o.Schema)
		if schema == "" {
			schema = "—"
		}
		crossTeam := "—"
		if o.DestinationTeam != nil && strings.TrimSpace(*o.DestinationTeam) != "" {
			crossTeam = "`" + *o.DestinationTeam + "`"
		}
		path := ""
		if o.DestinationPath != nil && strings.TrimSpace(*o.DestinationPath) != "" {
			path = " (`" + *o.DestinationPath + "`)"
		}
		b.WriteString(fmt.Sprintf("| `%s`%s | `%s` | `%s` | %s |\n", o.Prefix, path, o.DestinationKind, schema, crossTeam))
	}
	if hasAnySchema(outputs) {
		b.WriteString("\nFront-matter shapes for each schema are declared on the producer's taxonomy (see PoR links above).\n")
	}
}

func renderDecisionsBlock(b *strings.Builder, topics *memberflow.Topics) {
	if len(topics.DecisionsOwned) == 0 && len(topics.DecisionsConsumed) == 0 && !topics.RaisesCapabilityGaps {
		return
	}
	b.WriteString("\n## Decisions\n\n")
	b.WriteString("| Context | Role |\n|---|---|\n")
	owned := append([]string(nil), topics.DecisionsOwned...)
	sort.Strings(owned)
	for _, ctx := range owned {
		b.WriteString(fmt.Sprintf("| `%s` | own / propose |\n", ctx))
	}
	consumed := append([]string(nil), topics.DecisionsConsumed...)
	sort.Strings(consumed)
	for _, ctx := range consumed {
		b.WriteString(fmt.Sprintf("| `%s` | consume |\n", ctx))
	}
	if topics.RaisesCapabilityGaps {
		b.WriteString("| `capability-gap` | permitted to raise |\n")
	}
}

func renderDispatchBlock(b *strings.Builder, in *inboxFlowInputs) {
	// Group dispatch rows by taxonomy id so cross-intake members render
	// each taxonomy once.
	seen := map[string]bool{}
	first := true
	for _, intake := range in.memberFlow.Topics.Intake {
		taxonomyID := strings.TrimSpace(intake.Taxonomy)
		if taxonomyID == "" || seen[taxonomyID] {
			continue
		}
		seen[taxonomyID] = true
		tx, ok := in.taxonomies[taxonomyID]
		if !ok || tx == nil || len(tx.SignalTypes) == 0 {
			continue
		}
		if first {
			b.WriteString("\n## Default dispatch (from taxonomy)\n\n")
			b.WriteString("If the classifier returns a typed signal (or the topic-prefix names the type), the default method skill is:\n\n")
			first = false
		}
		b.WriteString(fmt.Sprintf("**Taxonomy `%s`:**\n\n", taxonomyID))
		b.WriteString("| signal_type | method skill | default destination |\n|---|---|---|\n")
		for _, st := range tx.SignalTypes {
			method := strings.TrimSpace(st.DefaultMethod)
			if method == "" {
				method = "_none_"
			} else {
				method = "`" + method + "`"
			}
			dest := strings.TrimSpace(st.DefaultDestinationPrefix)
			if dest == "" {
				dest = "—"
			} else {
				dest = "`" + dest + "`"
			}
			b.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", st.ID, method, dest))
		}
		if len(tx.PendingMethodSkills) > 0 {
			b.WriteString("\nPending method skills (referenced but not yet registered): `" + strings.Join(tx.PendingMethodSkills, "`, `") + "`. Apply inline guidance from the taxonomy PoR until the skill ships.\n")
		}
	}
	if !first {
		b.WriteString("\nOverride only when the classifier explicitly recommends a different method.\n")
	}
}

func prefixForList(prefix string) string {
	// Strip trailing /* for the list command (the CLI accepts the bare
	// prefix and treats it as a wildcard match).
	return strings.TrimSuffix(prefix, "*")
}

func derivePoRPath(taxonomyJSONPath string) string {
	if strings.TrimSpace(taxonomyJSONPath) == "" {
		return ""
	}
	dir := filepath.Dir(taxonomyJSONPath)
	base := strings.ToUpper(strings.TrimSuffix(filepath.Base(taxonomyJSONPath), ".json"))
	base = strings.ReplaceAll(base, "-", "_")
	return filepath.Join(dir, base+".md")
}

func hasAnySchema(outputs []memberflow.OutputEntry) bool {
	for _, o := range outputs {
		if strings.TrimSpace(o.Schema) != "" {
			return true
		}
	}
	return false
}
