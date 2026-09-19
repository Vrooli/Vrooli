// Responsibility: compose bounded, assigned skill packs into private run scopes.
package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"agent-manager/internal/promptmanager"
	"github.com/google/uuid"
)

const defaultProfileSkillTokenCeiling = 1500

var profileSkillIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// ProjectProfileSkills resolves and writes a profile's skills into the
// already-private runtime root for one run. It accepts only identifiers and
// refuses paths that are not visibly owned by the run, preventing a profile
// from widening its write scope to the shared checkout.
func ProjectProfileSkills(ctx context.Context, root string, runID uuid.UUID, runtimeRoot string, skillIDs []string, source promptmanager.SourceClient, maxTokens int) (int, error) {
	if source == nil {
		return 0, fmt.Errorf("profile skill projection requires prompt-manager source")
	}
	if maxTokens <= 0 {
		maxTokens = defaultProfileSkillTokenCeiling
	}
	cleanRoot := filepath.Clean(runtimeRoot)
	if cleanRoot == "." || !strings.Contains(filepath.ToSlash(cleanRoot), "/"+runID.String()+"/") {
		return 0, fmt.Errorf("refusing profile skill projection outside run %s: %s", runID, runtimeRoot)
	}
	if filepath.Clean(root) == cleanRoot {
		return 0, fmt.Errorf("refusing profile skill projection into shared project directory: %s", runtimeRoot)
	}

	ids := append([]string(nil), skillIDs...)
	sort.Strings(ids)
	ids = uniqueProfileSkillIDs(ids)
	type resolved struct{ id, description, content string }
	resolvedSkills := make([]resolved, 0, len(ids))
	resident := 0
	for _, id := range ids {
		if !profileSkillIDPattern.MatchString(id) {
			return 0, fmt.Errorf("invalid profile skill identifier %q", id)
		}
		snapshot, err := source.ReadSkillSource(ctx, id, "", nil, false)
		if err != nil {
			return 0, fmt.Errorf("resolve profile skill %q: %w", id, err)
		}
		description := snapshot.Description
		if strings.TrimSpace(description) == "" {
			description = id
		}
		resident += (len(id) + len(description) + 3) / 4
		if resident > maxTokens {
			return 0, fmt.Errorf("profile skill pack exceeds resident ceiling %d tokens at %q (%d)", maxTokens, id, resident)
		}
		resolvedSkills = append(resolvedSkills, resolved{id: id, description: description, content: snapshot.Content})
	}

	skillDir := filepath.Join(cleanRoot, "skills")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		return 0, fmt.Errorf("create profile skill directory: %w", err)
	}
	wanted := make(map[string]bool, len(resolvedSkills))
	for _, skill := range resolvedSkills {
		wanted[skill.id] = true
		dir := filepath.Join(skillDir, skill.id)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return 0, err
		}
		if err := os.WriteFile(filepath.Join(dir, ".vrooli-generated-profile"), []byte(runID.String()+"\n"), 0o600); err != nil {
			return 0, err
		}
		body := fmt.Sprintf("---\nname: %q\ndescription: %q\n---\n\n%s", skill.id, skill.description, skill.content)
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
			return 0, err
		}
	}
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || wanted[entry.Name()] {
			continue
		}
		marker := filepath.Join(skillDir, entry.Name(), ".vrooli-generated-profile")
		if _, markerErr := os.Stat(marker); markerErr == nil {
			if err := os.RemoveAll(filepath.Join(skillDir, entry.Name())); err != nil {
				return 0, err
			}
		}
	}
	return resident, nil
}

// ProjectProfileSkillsAssigned composes a profile through the durable
// assignment surface when an experiment is configured. Each skill receives a
// stable idempotency key for this run; retries read the frozen assignment
// rather than selecting a new arm. Without an experiment it preserves the
// observational source-read behavior of ProjectProfileSkills.
func ProjectProfileSkillsAssigned(ctx context.Context, root string, runID uuid.UUID, runtimeRoot, experimentID string, skillIDs []string, source promptmanager.SourceClient, maxTokens int) (int, error) {
	if strings.TrimSpace(experimentID) == "" {
		return ProjectProfileSkills(ctx, root, runID, runtimeRoot, skillIDs, source, maxTokens)
	}
	assigner, ok := source.(promptmanager.AssignmentClient)
	if !ok {
		return 0, fmt.Errorf("skill experiment %q requires an assignment-capable prompt-manager source", experimentID)
	}
	assigned := &assignedSkillSource{ctx: ctx, source: source, assigner: assigner, experimentID: experimentID, runID: runID}
	return ProjectProfileSkills(ctx, root, runID, runtimeRoot, skillIDs, assigned, maxTokens)
}

type assignedSkillSource struct {
	ctx          context.Context
	source       promptmanager.SourceClient
	assigner     promptmanager.AssignmentClient
	experimentID string
	runID        uuid.UUID
}

func (s *assignedSkillSource) ReadSkillSource(ctx context.Context, skillID, _ string, variables map[string]string, withScope bool) (promptmanager.SkillSourceSnapshot, error) {
	return s.assigner.AssignExperimentPrompt(ctx, promptmanager.AssignmentRequest{
		ExperimentID: s.experimentID, SkillID: skillID, ExecutionID: s.runID.String(), NodeID: "profile", AttemptKey: "1",
		IdempotencyKey: "skill-profile/" + s.runID.String() + "/" + skillID, Variables: variables, WithScope: withScope,
	})
}

func uniqueProfileSkillIDs(ids []string) []string {
	result := ids[:0]
	for _, id := range ids {
		if len(result) == 0 || result[len(result)-1] != id {
			result = append(result, id)
		}
	}
	return result
}

// UpdateProjectedProfileSkill applies one heartbeat delta to a run's private
// profile pack. The caller passes the same private runtime root used at launch;
// this function never discovers or writes a project-level path.
func UpdateProjectedProfileSkill(ctx context.Context, root string, runID uuid.UUID, runtimeRoot, skillID string, add bool, source promptmanager.SourceClient, maxTokens int) (int, error) {
	if !profileSkillIDPattern.MatchString(skillID) {
		return 0, fmt.Errorf("invalid profile skill identifier %q", skillID)
	}
	skillDir := filepath.Join(filepath.Clean(runtimeRoot), "skills")
	entries, err := os.ReadDir(skillDir)
	if err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	ids := make([]string, 0, len(entries)+1)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == skillID {
			continue
		}
		if _, markerErr := os.Stat(filepath.Join(skillDir, entry.Name(), ".vrooli-generated-profile")); markerErr == nil {
			ids = append(ids, entry.Name())
		}
	}
	if add {
		ids = append(ids, skillID)
	}
	return ProjectProfileSkills(ctx, root, runID, runtimeRoot, ids, source, maxTokens)
}
