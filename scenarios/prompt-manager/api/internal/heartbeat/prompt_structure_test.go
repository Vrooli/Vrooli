package heartbeat

import (
	"context"
	"sort"
	"strings"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamcontract"
)

// The prompt is the only artifact in this system that reaches a running agent.
// Every other validation surface checks a declaration. These two tests are the
// first sensors on the output itself: one proves that no injected document
// title escapes its section, and one proves that every section the precedence
// list ranks actually carries content.
//
// The defect they lock out was live on all six teams: `# Operating Policy`
// carried 61-67 bytes of source pointer while the real 7268-byte team contract
// sat under `# Meta Optimization Team`, and `# Heartbeat Task (HEARTBEAT.md)`
// — rank three in the prompt's own precedence order — carried nothing at all.

// livePromptSections builds every bundled member's prompt once, so both tests
// read the same roster the runtime does rather than a fixture approximation.
func livePromptSections(t *testing.T, visit func(t *testing.T, teamID, agentID string, prompt string, sections []PromptSection)) {
	t.Helper()
	ctx := context.Background()
	fileStore := newFileStore(t, paths.RootsForRepoStoreTest(t, "../../../store"))
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	builder := NewPromptBuilder(teamStore, agentStore)

	teams, err := teamStore.List(ctx)
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}
	sort.Slice(teams, func(i, j int) bool { return teams[i].ID < teams[j].ID })

	visited := 0
	for _, team := range teams {
		// Disabled teams are covered too. A team is re-enabled by flipping one
		// flag, and a prompt defect that only surfaces after that flip is a
		// defect this test exists to catch before the flip.
		if team.OperatingContract == nil {
			continue
		}
		members, err := teamStore.GetMembers(ctx, team.ID)
		if err != nil {
			t.Fatalf("list %s members: %v", team.ID, err)
		}
		sort.Slice(members, func(i, j int) bool { return members[i].AgentID < members[j].AgentID })
		for _, relation := range members {
			if relation.Status != "" && relation.Status != store.MemberStatusActive {
				continue
			}
			if _, ok := team.OperatingContract.Members[relation.AgentID]; !ok {
				continue
			}
			teamID, agentID := team.ID, relation.AgentID
			t.Run(teamID+"/"+agentID, func(t *testing.T) {
				req := PromptBuildRequest{TeamID: teamID, AgentID: agentID}
				prompt, err := builder.Build(ctx, req)
				if err != nil {
					t.Fatalf("build prompt: %v", err)
				}
				sections, err := builder.BuildStructured(ctx, req)
				if err != nil {
					t.Fatalf("build structured prompt: %v", err)
				}
				visit(t, teamID, agentID, prompt, sections)
			})
			visited++
		}
	}
	if visited == 0 {
		t.Fatal("no bundled member produced a prompt; the roster lookup is broken, not clean")
	}
}

// topLevelHeadings returns every level-one heading in an assembled prompt, in
// order, skipping fenced code blocks so a `# comment` in a shell example is
// not mistaken for a section.
func topLevelHeadings(prompt string) []string {
	var headings []string
	lines := strings.Split(prompt, "\n")
	forEachHeading(lines, func(_ int, level int, text string) {
		if level == 1 {
			headings = append(headings, "# "+text)
		}
	})
	return headings
}

func TestAssembledPromptEmitsOnlyRegisteredSectionHeadings(t *testing.T) {
	registered := promptSectionHeadings()
	livePromptSections(t, func(t *testing.T, teamID, agentID, prompt string, _ []PromptSection) {
		for _, heading := range topLevelHeadings(prompt) {
			if _, ok := registered[heading]; !ok {
				t.Errorf("prompt for %s/%s emits unregistered level-one heading %q; an injected document title has escaped its section",
					teamID, agentID, heading)
			}
		}
	})
}

// promptSectionBody returns the bytes a heading owns: everything from the
// heading to the next level-one heading. This is what the agent actually reads
// under that heading, which is why an empty body is a defect no matter what
// the builder intended to put there.
func promptSectionBody(prompt, heading string) (string, bool) {
	lines := strings.Split(prompt, "\n")
	start, end := -1, len(lines)
	forEachHeading(lines, func(index int, level int, text string) {
		if level != 1 {
			return
		}
		switch {
		case start == -1 && "# "+text == heading:
			start = index
		case start != -1 && index > start && end == len(lines):
			end = index
		}
	})
	if start == -1 {
		return "", false
	}
	return strings.TrimSpace(strings.Join(lines[start+1:end], "\n")), true
}

func TestPromptPrecedenceListNamesNonEmptySections(t *testing.T) {
	// The headings the prompt precedence list ranks, read from the registry
	// rather than restated here. A section that is ranked but
	// empty tells an agent to obey nothing.
	ranked := make([]string, 0, len(promptPrecedenceKinds))
	for _, kind := range promptPrecedenceKinds {
		ranked = append(ranked, promptHeading(kind))
	}
	livePromptSections(t, func(t *testing.T, teamID, agentID, prompt string, _ []PromptSection) {
		for _, heading := range ranked {
			body, ok := promptSectionBody(prompt, heading)
			if !ok {
				// A section the builder legitimately omits for this member is
				// not a defect; a section it emits empty is.
				continue
			}
			if body == "" {
				t.Errorf("prompt for %s/%s ranks %q in its precedence list but emits it empty",
					teamID, agentID, heading)
			}
		}
	})
}

// TestPrecedenceListHeadingsAreEmittableSections binds the hand-typed strings
// in the precedence list to the registry. Renaming a registry heading without
// updating the list fails here instead of silently pointing an agent at a
// heading that no longer exists.
func TestPrecedenceListHeadingsAreEmittableSections(t *testing.T) {
	registered := promptSectionHeadings()
	for _, kind := range promptPrecedenceKinds {
		heading := promptHeading(kind)
		if _, ok := registered[heading]; !ok {
			t.Errorf("precedence list names %q, which no registered prompt section emits", heading)
		}
	}
}

func TestShiftMarkdownHeadings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		body     string
		topLevel int
		want     string
	}{
		{
			name:     "document title demotes and children follow",
			body:     "# Title\n\ntext\n\n## Child\n\nmore",
			topLevel: 2,
			want:     "## Title\n\ntext\n\n### Child\n\nmore",
		},
		{
			name:     "already deep enough is untouched",
			body:     "## Section\n\ntext",
			topLevel: 2,
			want:     "## Section\n\ntext",
		},
		{
			name:     "shell comments inside a fence survive",
			body:     "# Title\n\n```bash\n# not a heading\nls\n```\n\n## Child",
			topLevel: 2,
			want:     "## Title\n\n```bash\n# not a heading\nls\n```\n\n### Child",
		},
		{
			name:     "tilde fences are honored too",
			body:     "# Title\n\n~~~\n# not a heading\n~~~",
			topLevel: 2,
			want:     "## Title\n\n~~~\n# not a heading\n~~~",
		},
		{
			name:     "a hash without a space is not a heading",
			body:     "# Title\n\n#hashtag",
			topLevel: 2,
			want:     "## Title\n\n#hashtag",
		},
		{
			name:     "level six clamps instead of overflowing",
			body:     "# Title\n\n###### Deep",
			topLevel: 3,
			want:     "### Title\n\n###### Deep",
		},
		{
			name:     "a document with no heading is untouched",
			body:     "just prose",
			topLevel: 2,
			want:     "just prose",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := shiftMarkdownHeadings(tc.body, tc.topLevel); got != tc.want {
				t.Errorf("shiftMarkdownHeadings()\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestEmittedSectionHeadingsMatchTheRegistry proves the assembled prompt uses
// the registry's heading for each section it emits, not a copy that has drifted
// from it.
func TestEmittedSectionHeadingsMatchTheRegistry(t *testing.T) {
	livePromptSections(t, func(t *testing.T, teamID, agentID, prompt string, sections []PromptSection) {
		for _, section := range sections {
			// Agent-file sections carry a `## path` sub-heading; their level-one
			// heading belongs to the merged block, checked by the prompt walk.
			if section.Kind == promptSectionKindAgentFile {
				continue
			}
			want := promptHeading(section.Kind)
			first := strings.SplitN(strings.TrimSpace(section.Content), "\n", 2)[0]
			if strings.TrimSpace(first) != want {
				t.Errorf("%s/%s section %q emits heading %q, registry says %q",
					teamID, agentID, section.Kind, first, want)
			}
		}
	})
}

// TestStorageMapFollowsMemberContract changes one declared write and proves
// the rendered storage capability changes with it. The task is no longer a
// second policy surface, so there is no duplicate brief to keep in sync.
func TestStorageMapFollowsMemberContract(t *testing.T) {
	ctx := context.Background()
	fileStore := newFileStore(t, paths.RootsForRepoStoreTest(t, "../../../store"))
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	builder := NewPromptBuilder(teamStore, agentStore)

	const teamID, agentID = "director-swarm", "portfolio-manager"
	req := PromptBuildRequest{TeamID: teamID, AgentID: agentID}

	before, err := builder.Build(ctx, req)
	if err != nil {
		t.Fatalf("build baseline prompt: %v", err)
	}
	storageBefore, _ := promptSectionBody(before, promptHeading(promptSectionKindStorageMap))
	if storageBefore == "" {
		t.Fatal("fixture member has no storage map")
	}

	team, err := teamStore.Get(ctx, teamID)
	if err != nil {
		t.Fatalf("get team: %v", err)
	}
	member := team.OperatingContract.Members[agentID]
	member.ForbiddenWrites = append(member.ForbiddenWrites, teamcontract.WriteRef{Kind: "knowledge"})
	team.OperatingContract.Members[agentID] = member

	storageAfter, err := builder.buildStorageMapSection(team, agentID)
	if err != nil {
		t.Fatalf("rebuild storage map: %v", err)
	}

	if storageAfter == storageBefore {
		t.Error("forbidding a knowledge write did not change the Storage Map write surface")
	}
}
