package xrefs

import (
	"context"
	"regexp"
	"strings"

	"prompt-manager/store"
)

// Regex patterns for reference extraction.
var (
	// Matches: prompt-manager skill read <ids> OR prompt-manager skills read <ids>
	cliReadRE = regexp.MustCompile("prompt-manager\\s+skills?\\s+read\\s+([^\n`]+)")

	// Matches: **kebab-case-id** (bold-listed in markdown)
	boldListedRE = regexp.MustCompile(`\*\*([a-z][a-z0-9]*(?:-[a-z0-9]+)*)\*\*`)

	// Matches: store/skills/packs/{pack}/{id}/ or /SKILL.md
	pathRefRE = regexp.MustCompile(`store/skills/packs/[a-z]+/([a-z][a-z0-9]*(?:-[a-z0-9]+)*)(?:/|/SKILL\.md)`)

	// Validates a kebab-case skill ID.
	validIDRE = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
)

// extractedRef is an intermediate result from scanning content.
type extractedRef struct {
	skillID    string
	refType    ReferenceType
	lineNumber int // 1-based
}

// Scanner extracts skill references from agents, teams, and skills.
type Scanner struct {
	agentStore *store.FileAgentStore
	teamStore  *store.FileTeamStore
	skillStore *store.FileSkillStore
}

// NewScanner creates a new cross-reference scanner.
func NewScanner(agentStore *store.FileAgentStore, teamStore *store.FileTeamStore, skillStore *store.FileSkillStore) *Scanner {
	return &Scanner{
		agentStore: agentStore,
		teamStore:  teamStore,
		skillStore: skillStore,
	}
}

// ScanAll scans all entities and returns all skill references found.
func (s *Scanner) ScanAll(ctx context.Context) ([]Reference, error) {
	// Build valid skill ID set
	skills, err := s.skillStore.List(ctx)
	if err != nil {
		return nil, err
	}
	validIDs := make(map[string]bool, len(skills))
	for _, sk := range skills {
		validIDs[sk.ID] = true
	}

	var refs []Reference

	// Scan agents
	agentRefs, err := s.scanAgents(ctx, validIDs)
	if err != nil {
		return nil, err
	}
	refs = append(refs, agentRefs...)

	// Scan teams
	teamRefs, err := s.scanTeams(ctx, validIDs)
	if err != nil {
		return nil, err
	}
	refs = append(refs, teamRefs...)

	// Scan skills (defaultScope + content cross-references)
	skillRefs := s.scanSkills(skills, validIDs)
	refs = append(refs, skillRefs...)

	return refs, nil
}

// scanAgents scans all agent files for skill references.
func (s *Scanner) scanAgents(ctx context.Context, validIDs map[string]bool) ([]Reference, error) {
	agents, err := s.agentStore.List(ctx)
	if err != nil {
		return nil, err
	}

	var refs []Reference
	for _, agent := range agents {
		files, err := s.agentStore.ListFiles(ctx, agent.ID)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir || !strings.HasSuffix(strings.ToLower(f.Path), ".md") {
				continue
			}
			content, err := s.agentStore.ReadFile(ctx, agent.ID, f.Path)
			if err != nil {
				continue
			}
			extracted := extractRefsFromContent(content, validIDs)
			for _, ext := range extracted {
				refs = append(refs, Reference{
					SkillID: ext.skillID,
					RefType: ext.refType,
					Source: ReferenceSource{
						EntityType: "agent",
						EntityID:   agent.ID,
						EntityName: agent.DisplayName,
						FilePath:   f.Path,
						LineNumber: ext.lineNumber,
					},
				})
			}
		}
	}
	return refs, nil
}

// scanTeams scans all team shared files for skill references.
func (s *Scanner) scanTeams(ctx context.Context, validIDs map[string]bool) ([]Reference, error) {
	teams, err := s.teamStore.List(ctx)
	if err != nil {
		return nil, err
	}

	var refs []Reference
	for _, team := range teams {
		files, err := s.teamStore.ListSharedFiles(ctx, team.ID)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir || !strings.HasSuffix(strings.ToLower(f.Path), ".md") {
				continue
			}
			content, err := s.teamStore.ReadSharedFile(ctx, team.ID, f.Path)
			if err != nil {
				continue
			}
			extracted := extractRefsFromContent(content, validIDs)
			for _, ext := range extracted {
				refs = append(refs, Reference{
					SkillID: ext.skillID,
					RefType: ext.refType,
					Source: ReferenceSource{
						EntityType: "team",
						EntityID:   team.ID,
						EntityName: team.DisplayName,
						FilePath:   f.Path,
						LineNumber: ext.lineNumber,
					},
				})
			}
		}
	}
	return refs, nil
}

// scanSkills scans skill metadata (defaultScope) and SKILL.md content for cross-references.
func (s *Scanner) scanSkills(skills []store.Skill, validIDs map[string]bool) []Reference {
	var refs []Reference
	for _, skill := range skills {
		// Check defaultScope
		if skill.DefaultScope != "" && validIDs[skill.DefaultScope] && skill.DefaultScope != skill.ID {
			refs = append(refs, Reference{
				SkillID: skill.DefaultScope,
				RefType: RefDefaultScope,
				Source: ReferenceSource{
					EntityType: "skill",
					EntityID:   skill.ID,
					EntityName: skill.Name,
					FilePath:   "skill.json",
					LineNumber: 0,
				},
			})
		}

		// Scan SKILL.md content for references to other skills
		_, content, err := s.skillStore.GetWithContent(context.Background(), skill.ID)
		if err != nil || content == "" {
			continue
		}
		extracted := extractRefsFromContent(content, validIDs)
		for _, ext := range extracted {
			// Skip self-references
			if ext.skillID == skill.ID {
				continue
			}
			refs = append(refs, Reference{
				SkillID: ext.skillID,
				RefType: ext.refType,
				Source: ReferenceSource{
					EntityType: "skill",
					EntityID:   skill.ID,
					EntityName: skill.Name,
					FilePath:   "SKILL.md",
					LineNumber: ext.lineNumber,
				},
			})
		}
	}
	return refs
}

// extractRefsFromContent extracts skill references from text content.
// This is a pure function for testability.
func extractRefsFromContent(content string, validIDs map[string]bool) []extractedRef {
	lines := strings.Split(content, "\n")
	type dedupeKey struct {
		skillID string
		refType ReferenceType
	}
	seen := make(map[dedupeKey]bool)
	var results []extractedRef

	for lineIdx, line := range lines {
		lineNum := lineIdx + 1 // 1-based

		// Pattern 1: CLI read commands
		for _, match := range cliReadRE.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			ids := strings.Fields(strings.TrimSpace(match[1]))
			for _, id := range ids {
				if !isValidSkillToken(id) || !validIDs[id] {
					continue
				}
				key := dedupeKey{id, RefCLIRead}
				if seen[key] {
					continue
				}
				seen[key] = true
				results = append(results, extractedRef{
					skillID:    id,
					refType:    RefCLIRead,
					lineNumber: lineNum,
				})
			}
		}

		// Pattern 2: Bold-listed names
		for _, match := range boldListedRE.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			id := match[1]
			if !validIDs[id] {
				continue
			}
			key := dedupeKey{id, RefBoldListed}
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, extractedRef{
				skillID:    id,
				refType:    RefBoldListed,
				lineNumber: lineNum,
			})
		}

		// Pattern 3: Path references
		for _, match := range pathRefRE.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			id := match[1]
			if !validIDs[id] {
				continue
			}
			key := dedupeKey{id, RefPathRef}
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, extractedRef{
				skillID:    id,
				refType:    RefPathRef,
				lineNumber: lineNum,
			})
		}
	}

	return results
}

// isValidSkillToken checks whether a token looks like a valid skill ID
// and is not a template/placeholder variable.
func isValidSkillToken(token string) bool {
	if !validIDRE.MatchString(token) {
		return false
	}
	// Reject tokens containing template/placeholder chars
	if strings.ContainsAny(token, "{}<>") {
		return false
	}
	return true
}
