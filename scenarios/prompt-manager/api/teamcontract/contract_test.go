package teamcontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRejectsMissingOperatingContract(t *testing.T) {
	err := Validate(nil, ValidationInput{TeamID: "team-1", DecisionMode: DecisionModeApproval})
	if err == nil || !strings.Contains(err.Error(), "operatingContract is required") {
		t.Fatalf("expected missing contract error, got %v", err)
	}
}

func TestValidateRejectsUnknownOwnedContext(t *testing.T) {
	contract := Minimal(DecisionModeApproval, "agent-1")
	member := contract.Members["agent-1"]
	member.OwnedDecisionContexts = append(member.OwnedDecisionContexts, "unknown-context")
	contract.Members["agent-1"] = member

	err := Validate(contract, ValidationInput{TeamID: "team-1", DecisionMode: DecisionModeApproval})
	if err == nil || !strings.Contains(err.Error(), "unknown-context") {
		t.Fatalf("expected unknown context error, got %v", err)
	}
}

func TestValidateRejectsMissingActiveMemberContract(t *testing.T) {
	contract := Minimal(DecisionModeApproval, "agent-1")
	err := Validate(contract, ValidationInput{
		TeamID:       "team-1",
		DecisionMode: DecisionModeApproval,
		MemberIDs:    []string{"agent-1", "agent-2"},
	})
	if err == nil || !strings.Contains(err.Error(), `missing active member "agent-2"`) {
		t.Fatalf("expected missing active member error, got %v", err)
	}
}

func TestNormalizePathRendersRepoRootRelative(t *testing.T) {
	got, err := NormalizePath(PathRef{Base: BaseTeamShared, Path: "RUN_LESSONS.md"}, ValidationInput{TeamID: "meta-optimization"}, "run-introspector")
	if err != nil {
		t.Fatalf("NormalizePath: %v", err)
	}
	want := "scenarios/prompt-manager/store/teams/meta-optimization/shared/RUN_LESSONS.md"
	if got != want {
		t.Fatalf("NormalizePath = %q, want %q", got, want)
	}
}

func TestNormalizePathRejectsTraversal(t *testing.T) {
	_, err := NormalizePath(PathRef{Base: BaseTeamShared, Path: "../TEAM.md"}, ValidationInput{TeamID: "meta-optimization"}, "agent-1")
	if err == nil || !strings.Contains(err.Error(), "must not escape") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestRenderMemberIncludesResolvedPolicy(t *testing.T) {
	contract := Minimal(DecisionModeApproval, "agent-1")
	rendered, err := RenderMember(contract, RenderInput{
		TeamID:       "team-1",
		TeamName:     "Team One",
		DecisionMode: DecisionModeApproval,
		MemberID:     "agent-1",
	})
	if err != nil {
		t.Fatalf("RenderMember: %v", err)
	}
	for _, want := range []string{
		"# Resolved Operating Contract",
		"Team: team-1",
		"Agent ID: agent-1",
		"Owned decision contexts:",
		"Required knowledge topics:",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered contract missing %q:\n%s", want, rendered)
		}
	}
}

func TestBundledMetaOptimizationContractValidatesAndRendersRepoRootPaths(t *testing.T) {
	type teamFile struct {
		DecisionMode      string             `json:"decisionMode"`
		OperatingContract *OperatingContract `json:"operatingContract"`
	}

	storeDir := filepath.Clean("../../store")
	repoRoot := filepath.Clean("../../../..")
	data, err := os.ReadFile(filepath.Join(storeDir, "teams", "meta-optimization", "team.json"))
	if err != nil {
		t.Fatalf("read bundled team: %v", err)
	}
	var team teamFile
	if err := json.Unmarshal(data, &team); err != nil {
		t.Fatalf("unmarshal bundled team: %v", err)
	}

	err = Validate(team.OperatingContract, ValidationInput{
		TeamID:       "meta-optimization",
		DecisionMode: team.DecisionMode,
		StoreDir:     storeDir,
		RepoRoot:     repoRoot,
	})
	if err != nil {
		t.Fatalf("Validate bundled contract: %v", err)
	}

	rendered, err := RenderMember(team.OperatingContract, RenderInput{
		TeamID:       "meta-optimization",
		TeamName:     "Meta Optimization",
		DecisionMode: team.DecisionMode,
		MemberID:     "team-agent-optimizer",
		StoreDir:     storeDir,
		RepoRoot:     repoRoot,
	})
	if err != nil {
		t.Fatalf("RenderMember bundled contract: %v", err)
	}
	want := "scenarios/prompt-manager/store/teams/meta-optimization/shared/TEAM_AUDIT.md"
	if !strings.Contains(rendered, want) {
		t.Fatalf("rendered contract missing repo-root TEAM_AUDIT path %q:\n%s", want, rendered)
	}
	if strings.Contains(rendered, "\n- shared/TEAM_AUDIT.md") {
		t.Fatalf("rendered contract contains ambiguous TEAM_AUDIT path:\n%s", rendered)
	}
}

func TestBundledTeamContractsValidate(t *testing.T) {
	type teamFile struct {
		DecisionMode      string             `json:"decisionMode"`
		OperatingContract *OperatingContract `json:"operatingContract"`
	}

	storeDir := filepath.Clean("../../store")
	repoRoot := filepath.Clean("../../../..")
	teamsDir := filepath.Join(storeDir, "teams")
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		t.Fatalf("read teams dir: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		teamID := entry.Name()
		data, err := os.ReadFile(filepath.Join(teamsDir, teamID, "team.json"))
		if err != nil {
			t.Fatalf("read %s team: %v", teamID, err)
		}
		var team teamFile
		if err := json.Unmarshal(data, &team); err != nil {
			t.Fatalf("unmarshal %s team: %v", teamID, err)
		}
		memberIDs := relationMemberIDs(t, storeDir, teamID)
		if err := Validate(team.OperatingContract, ValidationInput{
			TeamID:       teamID,
			DecisionMode: team.DecisionMode,
			MemberIDs:    memberIDs,
			StoreDir:     storeDir,
			RepoRoot:     repoRoot,
		}); err != nil {
			t.Fatalf("Validate bundled %s contract: %v", teamID, err)
		}
	}
}

func TestBundledPromptProseDoesNotRestateContractPolicy(t *testing.T) {
	storeDir := filepath.Clean("../../store")
	files := appendGlob(t, nil, filepath.Join(storeDir, "teams", "*", "shared", "TEAM.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "teams", "*", "members", "*", "RESPONSIBILITIES.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "teams", "*", "members", "*", "HEARTBEAT.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "agents", "*", "SOUL.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "agents", "*", "AGENTS.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "agents", "*", "TOOLS.md"))

	forbidden := []string{
		"Team-ceiling",
		"Team ceiling",
		"Own-context cap",
		"Decision Contexts",
		"Decision Queue Discipline",
		"Shared State",
		"Knowledge supersession",
		"14 heartbeats",
		"4+ decisions",
		"3+ decisions",
		"max new decisions",
		"Team queue at capacity",
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read prompt prose %s: %v", file, err)
		}
		text := string(data)
		for _, phrase := range forbidden {
			if strings.Contains(text, phrase) {
				t.Fatalf("%s restates contract-owned policy phrase %q", file, phrase)
			}
		}
	}
}

func relationMemberIDs(t *testing.T, storeDir, teamID string) []string {
	t.Helper()
	relationsDir := filepath.Join(storeDir, "relations", "team-member")
	entries, err := os.ReadDir(relationsDir)
	if err != nil {
		t.Fatalf("read relations dir: %v", err)
	}
	var memberIDs []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), teamID+"__") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(relationsDir, entry.Name()))
		if err != nil {
			t.Fatalf("read relation %s: %v", entry.Name(), err)
		}
		var relation struct {
			AgentID string `json:"agentId"`
			Status  string `json:"status"`
		}
		if err := json.Unmarshal(data, &relation); err != nil {
			t.Fatalf("unmarshal relation %s: %v", entry.Name(), err)
		}
		if relation.Status == "" || relation.Status == "active" {
			memberIDs = append(memberIDs, relation.AgentID)
		}
	}
	return memberIDs
}

func appendGlob(t *testing.T, files []string, pattern string) []string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	return append(files, matches...)
}
