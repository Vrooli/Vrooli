package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/promptmanager"
	"github.com/google/uuid"
)

type profilePackSource struct {
	skills map[string]promptmanager.SkillSourceSnapshot
}

type assignedProfilePackSource struct{ profilePackSource }

func (s assignedProfilePackSource) AssignExperimentPrompt(_ context.Context, req promptmanager.AssignmentRequest) (promptmanager.SkillSourceSnapshot, error) {
	snapshot, ok := s.skills[req.SkillID]
	if !ok {
		return promptmanager.SkillSourceSnapshot{}, os.ErrNotExist
	}
	snapshot.ExperimentID, snapshot.VariantID = req.ExperimentID, "control"
	return snapshot, nil
}

func (s profilePackSource) ReadSkillSource(_ context.Context, id, _ string, _ map[string]string, _ bool) (promptmanager.SkillSourceSnapshot, error) {
	if skill, ok := s.skills[id]; ok {
		return skill, nil
	}
	return promptmanager.SkillSourceSnapshot{}, os.ErrNotExist
}

func TestProjectProfileSkillsIsolatesConcurrentRunsAndReapsGeneratedSkills(t *testing.T) {
	root := t.TempDir()
	source := profilePackSource{skills: map[string]promptmanager.SkillSourceSnapshot{
		"alpha": {SkillID: "alpha", Description: "alpha skill", Content: "alpha instructions"},
		"beta":  {SkillID: "beta", Description: "beta skill", Content: "beta instructions"},
	}}
	first, second := uuid.New(), uuid.New()
	firstRoot := filepath.Join(root, first.String(), "skill-scope", "claude-code")
	secondRoot := filepath.Join(root, second.String(), "skill-scope", "claude-code")
	if _, err := ProjectProfileSkills(context.Background(), root, first, firstRoot, []string{"alpha"}, source, 1500); err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectProfileSkills(context.Background(), root, second, secondRoot, []string{"beta"}, source, 1500); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(firstRoot, "skills", "alpha", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(firstRoot, "skills", "beta")); !os.IsNotExist(err) {
		t.Fatalf("first run observed second profile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(secondRoot, "skills", "alpha")); !os.IsNotExist(err) {
		t.Fatalf("second run observed first profile: %v", err)
	}
	if _, err := ProjectProfileSkills(context.Background(), root, first, firstRoot, []string{"beta"}, source, 1500); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(firstRoot, "skills", "alpha")); !os.IsNotExist(err) {
		t.Fatalf("stale generated skill survived refresh: %v", err)
	}
}

func TestProjectProfileSkillsRefusesSharedRootAndBudgetOverflow(t *testing.T) {
	root := t.TempDir()
	runID := uuid.New()
	source := profilePackSource{skills: map[string]promptmanager.SkillSourceSnapshot{
		"large": {SkillID: "large", Description: strings.Repeat("x", 8000), Content: "body"},
	}}
	if _, err := ProjectProfileSkills(context.Background(), root, runID, root, []string{"large"}, source, 1500); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("shared root was accepted: %v", err)
	}
	private := filepath.Join(root, runID.String(), "skill-scope", "claude-code")
	if _, err := ProjectProfileSkills(context.Background(), root, runID, private, []string{"large"}, source, 10); err == nil || !strings.Contains(err.Error(), "exceeds resident ceiling") {
		t.Fatalf("budget overflow was accepted: %v", err)
	}
}

func TestUpdateProjectedProfileSkillChangesOnlyOneRun(t *testing.T) {
	root := t.TempDir()
	runID := uuid.New()
	runtimeRoot := filepath.Join(root, runID.String(), "skill-scope", "claude-code")
	source := profilePackSource{skills: map[string]promptmanager.SkillSourceSnapshot{
		"alpha": {SkillID: "alpha", Description: "alpha", Content: "alpha"},
		"beta":  {SkillID: "beta", Description: "beta", Content: "beta"},
	}}
	if _, err := ProjectProfileSkills(context.Background(), root, runID, runtimeRoot, []string{"alpha"}, source, 1500); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateProjectedProfileSkill(context.Background(), root, runID, runtimeRoot, "beta", true, source, 1500); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(runtimeRoot, "skills", "beta", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateProjectedProfileSkill(context.Background(), root, runID, runtimeRoot, "alpha", false, source, 1500); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(runtimeRoot, "skills", "alpha")); !os.IsNotExist(err) {
		t.Fatalf("removed skill survived heartbeat update: %v", err)
	}
}

func TestProjectProfileSkillsAssignedUsesStableAssignmentSurface(t *testing.T) {
	root := t.TempDir()
	runID := uuid.New()
	runtimeRoot := filepath.Join(root, runID.String(), "skill-scope", "claude-code")
	source := assignedProfilePackSource{profilePackSource{skills: map[string]promptmanager.SkillSourceSnapshot{
		"alpha": {SkillID: "alpha", Description: "assigned", Content: "frozen treatment"},
	}}}
	if _, err := ProjectProfileSkillsAssigned(context.Background(), root, runID, runtimeRoot, "exp-1", []string{"alpha"}, source, 1500); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(runtimeRoot, "skills", "alpha", "SKILL.md"))
	if err != nil || !strings.Contains(string(body), "frozen treatment") {
		t.Fatalf("assigned profile content missing: %v", err)
	}
}
