package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"git-control-tower/internal/testutil/fixtures"
)

func TestResolveChangeGroupsManualRulesPrecedeContractTargets(t *testing.T) {
	repoDir := t.TempDir()
	fixtures.WriteRepoContract(t, repoDir)
	fixtures.WriteFile(t, filepath.Join(repoDir, "internal", "tools", "compiler", "tool.json"), `{}`)

	groups := ResolveChangeGroups(repoDir, RepoFilesStatus{Untracked: []string{
		"internal/tools/compiler/main.go",
		"internal/other.go",
	}}, GroupingRulesConfig{Rules: []GroupingRule{
		{ID: "tooling", Label: "Tooling", Prefixes: []string{"internal/tools/"}, Mode: "segment"},
	}})

	if len(groups) != 2 {
		t.Fatalf("groups = %#v, want manual and contract groups", groups)
	}
	if groups[0].Key != "tooling:compiler" || groups[0].Source != groupSourceManual || len(groups[0].Files) != 1 {
		t.Fatalf("manual group = %#v", groups[0])
	}
	if groups[1].Key != "contract:control-plane:internal" || groups[1].Kind != "control-plane" || groups[1].ID != "internal" {
		t.Fatalf("contract group = %#v", groups[1])
	}
	if strings.Contains(groups[0].Key, "contract:") {
		t.Fatal("manual group was replaced by a contract group")
	}
}

func TestResolveChangeGroupsFallsBackToOtherWithoutContract(t *testing.T) {
	groups := ResolveChangeGroups(t.TempDir(), RepoFilesStatus{Untracked: []string{
		"README.md",
		"scenarios/example/api/main.go",
	}}, GroupingRulesConfig{Rules: []GroupingRule{
		{ID: "scenarios", Label: "Scenarios", Prefixes: []string{"scenarios/"}, Mode: "prefix"},
	}})

	if len(groups) != 2 {
		t.Fatalf("groups = %#v, want manual and Other groups", groups)
	}
	if groups[0].Key != "scenarios" || groups[0].Source != groupSourceManual || len(groups[0].Files) != 1 {
		t.Fatalf("manual group = %#v", groups[0])
	}
	if groups[1].Key != "other" || groups[1].Source != groupSourceBuiltin || groups[1].Kind != "" || groups[1].ID != "" {
		t.Fatalf("Other group = %#v", groups[1])
	}
}

func TestResolveChangeGroupsOmitsEmptyContractTargets(t *testing.T) {
	repoDir := t.TempDir()
	fixtures.WriteRepoContract(t, repoDir)

	groups := ResolveChangeGroups(repoDir, RepoFilesStatus{Untracked: []string{"README.md"}}, GroupingRulesConfig{})
	if len(groups) != 1 {
		t.Fatalf("groups = %#v, want one group", groups)
	}
	if groups[0].Source != groupSourceContract && groups[0].Source != groupSourceBuiltin {
		t.Fatalf("unexpected source = %#v", groups[0])
	}
	for _, group := range groups {
		if len(group.Files) == 0 {
			t.Fatalf("empty group should be omitted: %#v", group)
		}
	}
}

func TestRepoGroupsResponseWithoutContractMetadata(t *testing.T) {
	body, err := json.Marshal(RepoGroupsResponse{Groups: []ChangeGroup{{
		Key: "other", Label: "Other", Source: groupSourceBuiltin, Files: []string{"README.md"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	if strings.Contains(raw, `"kind"`) || strings.Contains(raw, `"id"`) || strings.Contains(raw, `"root"`) {
		t.Fatalf("fallback response leaked contract metadata: %s", raw)
	}
}

func TestResolveChangeGroupsOrdersByKind(t *testing.T) {
	groups := []resolvedChangeGroup{
		{group: ChangeGroup{Label: "docs"}, sourceRank: 1, kindRank: changeGroupKindRank("docs")},
		{group: ChangeGroup{Label: "resources"}, sourceRank: 1, kindRank: changeGroupKindRank("resource")},
		{group: ChangeGroup{Label: "scenarios"}, sourceRank: 1, kindRank: changeGroupKindRank("scenario")},
		{group: ChangeGroup{Label: "packages"}, sourceRank: 1, kindRank: changeGroupKindRank("package")},
		{group: ChangeGroup{Label: "manual"}, sourceRank: 0, order: 0},
		{group: ChangeGroup{Label: "Other"}, sourceRank: 2, order: 0},
	}
	sortResolvedChangeGroups(groups)
	labels := make([]string, 0, len(groups))
	for _, group := range groups {
		labels = append(labels, group.group.Label)
	}
	want := []string{"manual", "scenarios", "resources", "packages", "docs", "Other"}
	if strings.Join(labels, ",") != strings.Join(want, ",") {
		t.Fatalf("labels = %v, want %v", labels, want)
	}
}
