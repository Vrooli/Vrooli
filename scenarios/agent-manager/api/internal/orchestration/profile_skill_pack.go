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

// ProjectProfileSkills resolves and writes a profile's skills into the run's
// private native skills directory (RunnerSkillsDir). It accepts only identifiers and
// refuses paths that are not visibly owned by the run, preventing a profile
// from widening its write scope to the shared checkout.
func ProjectProfileSkills(ctx context.Context, root string, runID uuid.UUID, skillsDir string, skillIDs []string, source promptmanager.SourceClient, maxTokens int) (int, error) {
	if source == nil {
		return 0, fmt.Errorf("profile skill projection requires prompt-manager source")
	}
	if maxTokens <= 0 {
		maxTokens = defaultProfileSkillTokenCeiling
	}
	skillDir := filepath.Clean(skillsDir)
	if skillDir == "." || !strings.Contains(filepath.ToSlash(skillDir), "/"+runID.String()+"/") {
		return 0, fmt.Errorf("refusing profile skill projection outside run %s: %s", runID, skillsDir)
	}
	if filepath.Clean(root) == skillDir {
		return 0, fmt.Errorf("refusing profile skill projection into shared project directory: %s", skillsDir)
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
		snapshot, err := source.ReadSkillSource(ctx, id, nil, false)
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
// profile pack. The caller passes the same private skills directory used at
// launch; this function never discovers or writes a project-level path.
func UpdateProjectedProfileSkill(ctx context.Context, root string, runID uuid.UUID, skillsDir, skillID string, add bool, source promptmanager.SourceClient, maxTokens int) (int, error) {
	if !profileSkillIDPattern.MatchString(skillID) {
		return 0, fmt.Errorf("invalid profile skill identifier %q", skillID)
	}
	skillDir := filepath.Clean(skillsDir)
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
	return ProjectProfileSkills(ctx, root, runID, skillsDir, ids, source, maxTokens)
}
