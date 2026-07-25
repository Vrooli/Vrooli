// DOC: docs/concepts/GRAPH.md#reference-detection
package graph

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"prompt-manager/store"
)

// Regex patterns for reference extraction (ported from xrefs/scanner.go).
var (
	// Matches: prompt-manager skill read <ids> OR prompt-manager skills read <ids>
	cliReadRE = regexp.MustCompile("prompt-manager\\s+skills?\\s+read\\s+([^\n`]+)")

	// Matches: prompt-manager action run <id> OR prompt-manager actions run <id>.
	actionRunRE = regexp.MustCompile(`prompt-manager\s+actions?\s+run\s+([a-z][a-z0-9]*(?:[.-][a-z0-9]+)*)`)

	// Matches explicit prose references such as action:scenario.status.show.
	actionRefRE = regexp.MustCompile(`\baction:([a-z][a-z0-9]*(?:[.-][a-z0-9]+)*)\b`)

	// Matches: **kebab-case-id** (bold-listed in markdown)
	boldListedRE = regexp.MustCompile(`\*\*([a-z][a-z0-9]*(?:-[a-z0-9]+)*)\*\*`)

	// Matches: store/skills/packs/{pack}/{id}/ or /SKILL.md
	pathRefRE = regexp.MustCompile(`store/skills/packs/[a-z]+/([a-z][a-z0-9]*(?:-[a-z0-9]+)*)(?:/|/SKILL\.md)`)

	// Validates a kebab-case skill ID.
	validIDRE = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

	// Matches the prompt reference owned by an Agent Manager workflow
	// definition. The skill is delivered to the run even though no shell
	// command appears in the JSON declaration.
	workflowPromptRefRE = regexp.MustCompile(`"skillId"\s*:\s*"([a-z][a-z0-9]*(?:-[a-z0-9]+)*)"`)
)

// extractedRef is an intermediate result from scanning content.
type extractedRef struct {
	skillID    string
	edgeKind   EdgeKind
	lineNumber int // 1-based
}

// agentLister provides agent scanning methods.
type agentLister interface {
	List(ctx context.Context) ([]store.Agent, error)
	ListFiles(ctx context.Context, agentID string) ([]store.AgentFileEntry, error)
	ReadFile(ctx context.Context, agentID, relPath string) (string, error)
}

// teamLister provides team scanning methods.
type teamLister interface {
	List(ctx context.Context) ([]store.Team, error)
	ListSharedFiles(ctx context.Context, teamID string) ([]store.TeamFileEntry, error)
	ReadSharedFile(ctx context.Context, teamID, relPath string) (string, error)
}

// skillLister provides skill scanning methods.
type skillLister interface {
	List(ctx context.Context) ([]store.Skill, error)
	GetWithContent(ctx context.Context, id string) (*store.Skill, string, error)
}

// actionLister provides Action metadata for action-use reference extraction.
type actionLister interface {
	List(ctx context.Context) ([]store.Action, error)
}

// codeDetector detects code references in content.
// The concrete *CLIDetector satisfies this interface.
type codeDetector interface {
	Detect(content string) []CodeReference
}

// generatedPromptProvider supplies the assembled, runtime prompt for a team
// member. Generated prompt sections are a real skill-reference surface even
// when no source markdown file contains the reference directly.
type generatedPromptProvider func(ctx context.Context, teamID, agentID string) (string, error)

// Scanner extracts graph edges from agents, teams, and skills.
type Scanner struct {
	agentStore     agentLister
	teamStore      teamLister
	skillStore     skillLister
	actionStore    actionLister
	relationStore  store.RelationStore
	cliDetector    codeDetector
	repositoryRoot string
	promptProvider generatedPromptProvider
}

// SetRepositoryRoot enables scanning repository-level instruction and workflow
// sources. It is deliberately explicit so unit tests and embedded callers do
// not accidentally read the process working directory.
func (s *Scanner) SetRepositoryRoot(root string) {
	s.repositoryRoot = root
}

// SetGeneratedPromptProvider registers the composition boundary used to scan
// generated member prompt sections.
func (s *Scanner) SetGeneratedPromptProvider(provider generatedPromptProvider) {
	s.promptProvider = provider
}

// NewScanner creates a new graph scanner.
func NewScanner(
	agentStore agentLister,
	teamStore teamLister,
	skillStore skillLister,
	relationStore store.RelationStore,
	cliDetector codeDetector,
	actionStores ...actionLister,
) *Scanner {
	s := &Scanner{
		agentStore:    agentStore,
		teamStore:     teamStore,
		skillStore:    skillStore,
		relationStore: relationStore,
		cliDetector:   cliDetector,
	}
	if len(actionStores) > 0 {
		s.actionStore = actionStores[0]
	}
	return s
}

// ScanAll scans all entities and returns all edges found.
func (s *Scanner) ScanAll(ctx context.Context) ([]Edge, error) {
	// Build valid skill ID set
	skills, err := s.skillStore.List(ctx)
	if err != nil {
		return nil, err
	}
	validIDs := make(map[string]bool, len(skills))
	for _, sk := range skills {
		validIDs[sk.ID] = true
	}
	validActionIDs := make(map[string]bool)
	if s.actionStore != nil {
		actions, err := s.actionStore.List(ctx)
		if err != nil {
			return nil, err
		}
		validActionIDs = make(map[string]bool, len(actions))
		for _, action := range actions {
			validActionIDs[action.ID] = true
		}
	}

	var edges []Edge

	// Scan agents for skill references
	agentEdges, err := s.scanAgents(ctx, validIDs, validActionIDs)
	if err != nil {
		return nil, err
	}
	edges = append(edges, agentEdges...)

	// Scan teams for skill references
	teamEdges, err := s.scanTeams(ctx, validIDs, validActionIDs)
	if err != nil {
		return nil, err
	}
	edges = append(edges, teamEdges...)

	// Scan skills for cross-references
	skillEdges := s.scanSkills(skills, validIDs, validActionIDs)
	edges = append(edges, skillEdges...)

	// Scan repository-level instruction and workflow definitions. These are
	// genuine references but are not owned by an individual store entity.
	edges = append(edges, s.scanRepositoryReferences(validIDs)...)

	// Generated prompt sections are the effective instructions received by
	// members, so they must participate in reachability queries.
	promptEdges, err := s.scanGeneratedPrompts(ctx, validIDs)
	if err != nil {
		return nil, err
	}
	edges = append(edges, promptEdges...)

	// Scan team-agent membership from relations
	memberEdges, err := s.scanMemberships(ctx)
	if err != nil {
		return nil, err
	}
	edges = append(edges, memberEdges...)

	return edges, nil
}

func (s *Scanner) scanRepositoryReferences(validIDs map[string]bool) []Edge {
	if s.repositoryRoot == "" {
		return nil
	}

	var edges []Edge
	if content, err := os.ReadFile(filepath.Join(s.repositoryRoot, "AGENTS.md")); err == nil {
		edges = append(edges, skillReferenceEdges("system:agents", "AGENTS.md", string(content), validIDs)...)
	}

	for _, workflowRoot := range s.workflowRoots() {
		_ = filepath.WalkDir(workflowRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			rel, err := filepath.Rel(s.repositoryRoot, path)
			if err != nil {
				return nil
			}
			sourceFile := filepath.ToSlash(rel)
			source := "workflow:" + sourceFile
			switch filepath.Ext(path) {
			case ".json":
				edges = append(edges, workflowPromptReferenceEdges(source, sourceFile, string(content), validIDs)...)
			case ".go":
				if cliReadRE.Match(content) {
					edges = append(edges, skillReferenceEdges(source, sourceFile, string(content), validIDs)...)
				}
			}
			return nil
		})
	}
	return edges
}

// workflowRoots returns every directory that can hold Agent Manager workflow
// definitions. Any scenario may own workflows that dispatch a skill, so the
// scan covers `scenarios/*/.vrooli/agent-manager` rather than one scenario.
// Skills reached only through workflow dispatch are genuinely reachable, and
// omitting a scenario reports them as orphans.
func (s *Scanner) workflowRoots() []string {
	roots := []string{filepath.Join(s.repositoryRoot, "scenarios", "swarm-manager", "api")}

	scenariosDir := filepath.Join(s.repositoryRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return roots
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		roots = append(roots, filepath.Join(scenariosDir, entry.Name(), ".vrooli", "agent-manager"))
	}
	return roots
}

func workflowPromptReferenceEdges(sourceID, sourceFile, content string, validIDs map[string]bool) []Edge {
	seen := map[string]bool{}
	var edges []Edge
	for _, match := range workflowPromptRefRE.FindAllStringSubmatchIndex(content, -1) {
		if len(match) < 4 {
			continue
		}
		id := content[match[2]:match[3]]
		if !validIDs[id] || seen[id] {
			continue
		}
		seen[id] = true
		edges = append(edges, Edge{
			From:       sourceID,
			To:         id,
			Kind:       EdgeCLIRead,
			SourceFile: sourceFile,
			LineNumber: strings.Count(content[:match[0]], "\n") + 1,
		})
	}
	return edges
}

func (s *Scanner) scanGeneratedPrompts(ctx context.Context, validIDs map[string]bool) ([]Edge, error) {
	if s.promptProvider == nil || s.relationStore == nil {
		return nil, nil
	}
	teams, err := s.teamStore.List(ctx)
	if err != nil {
		return nil, err
	}
	var edges []Edge
	for _, team := range teams {
		members, err := s.relationStore.ListTeamMembers(ctx, team.ID)
		if err != nil {
			continue
		}
		for _, member := range members {
			content, err := s.promptProvider(ctx, team.ID, member.AgentID)
			if err != nil {
				continue
			}
			edges = append(edges, skillReferenceEdges(member.AgentID, "generated-prompt", content, validIDs)...)
		}
	}
	return edges, nil
}

func skillReferenceEdges(sourceID, sourceFile, content string, validIDs map[string]bool) []Edge {
	extracted := extractRefsFromContent(content, validIDs)
	edges := make([]Edge, 0, len(extracted))
	for _, ext := range extracted {
		edges = append(edges, Edge{
			From:       sourceID,
			To:         ext.skillID,
			Kind:       ext.edgeKind,
			SourceFile: sourceFile,
			LineNumber: ext.lineNumber,
		})
	}
	return edges
}

// scanAgents scans all agent files for skill references and code usage.
func (s *Scanner) scanAgents(ctx context.Context, validIDs map[string]bool, validActionIDs map[string]bool) ([]Edge, error) {
	agents, err := s.agentStore.List(ctx)
	if err != nil {
		return nil, err
	}

	var edges []Edge
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

			edges = append(edges, skillReferenceEdges(agent.ID, f.Path, content, validIDs)...)
			edges = append(edges, extractActionUseEdges(agent.ID, f.Path, content, validActionIDs)...)

			// Code usage edges
			edges = append(edges, s.codeUsageEdgesFromContent(agent.ID, f.Path, content)...)
		}
	}
	return edges, nil
}

// scanTeams scans all team shared files for skill references and code usage.
func (s *Scanner) scanTeams(ctx context.Context, validIDs map[string]bool, validActionIDs map[string]bool) ([]Edge, error) {
	teams, err := s.teamStore.List(ctx)
	if err != nil {
		return nil, err
	}

	var edges []Edge
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

			edges = append(edges, skillReferenceEdges(team.ID, f.Path, content, validIDs)...)
			edges = append(edges, extractActionUseEdges(team.ID, f.Path, content, validActionIDs)...)

			// Code usage edges
			edges = append(edges, s.codeUsageEdgesFromContent(team.ID, f.Path, content)...)
		}
	}
	return edges, nil
}

// scanSkills scans skill metadata and content for cross-references.
func (s *Scanner) scanSkills(skills []store.Skill, validIDs map[string]bool, validActionIDs map[string]bool) []Edge {
	var edges []Edge
	for _, skill := range skills {
		// Default scope edge
		if skill.DefaultScope != "" && validIDs[skill.DefaultScope] && skill.DefaultScope != skill.ID {
			edges = append(edges, Edge{
				From:       skill.ID,
				To:         skill.DefaultScope,
				Kind:       EdgeDefaultScope,
				SourceFile: "skill.json",
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
			edges = append(edges, Edge{
				From:       skill.ID,
				To:         ext.skillID,
				Kind:       ext.edgeKind,
				SourceFile: "SKILL.md",
				LineNumber: ext.lineNumber,
			})
		}
		edges = append(edges, extractActionUseEdges(skill.ID, "SKILL.md", content, validActionIDs)...)

		// Code usage edges from skill content
		edges = append(edges, s.codeUsageEdgesFromContent(skill.ID, "SKILL.md", content)...)
	}
	return edges
}

func extractActionUseEdges(sourceID, sourceFile, content string, validActionIDs map[string]bool) []Edge {
	if len(validActionIDs) == 0 {
		return nil
	}
	lines := strings.Split(content, "\n")
	seen := make(map[string]bool)
	var edges []Edge
	for lineIdx, line := range lines {
		lineNum := lineIdx + 1
		for _, match := range actionRunRE.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			id := match[1]
			if !validActionIDs[id] || seen[id] {
				continue
			}
			seen[id] = true
			edges = append(edges, Edge{
				From:       sourceID,
				To:         actionNodeID(id),
				Kind:       EdgeActionUse,
				SourceFile: sourceFile,
				LineNumber: lineNum,
			})
		}
		for _, match := range actionRefRE.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			id := match[1]
			if !validActionIDs[id] || seen[id] {
				continue
			}
			seen[id] = true
			edges = append(edges, Edge{
				From:       sourceID,
				To:         actionNodeID(id),
				Kind:       EdgeActionUse,
				SourceFile: sourceFile,
				LineNumber: lineNum,
			})
		}
	}
	return edges
}

// codeUsageEdgesFromContent runs the code detector on content and returns
// edges for CodeScenarioCLI, CodeExternalTool, and CodeScript references.
// CodeAPICall is intentionally excluded (documentation, not tool invocation).
// "prompt-manager skill read" commands are skipped — those are Skill→Skill
// relations handled as EdgeCLIRead by extractRefsFromContent.
func (s *Scanner) codeUsageEdgesFromContent(sourceID, sourceFile, content string) []Edge {
	if s.cliDetector == nil {
		return nil
	}
	type dedupeKey struct {
		from, to string
		cat      CodeCategory
	}
	seen := make(map[dedupeKey]bool)
	var edges []Edge
	for _, cr := range s.cliDetector.Detect(content) {
		switch cr.Category {
		case CodeScenarioCLI:
			// Skip "prompt-manager skill read" — handled as EdgeCLIRead
			if cliReadRE.MatchString(cr.Value) {
				continue
			}
		case CodeExternalTool, CodeScript:
			// Allow
		default:
			continue // CodeAPICall intentionally excluded
		}
		to := cliNodeID(cr.Value)
		command, subcommand := parseCommandParts(cr.Value)
		key := dedupeKey{sourceID, to, cr.Category}
		if seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, Edge{
			From:        sourceID,
			To:          to,
			Kind:        EdgeCodeUsage,
			Category:    cr.Category,
			Command:     command,
			Subcommand:  subcommand,
			CommandText: cr.Value,
			SourceFile:  sourceFile,
			LineNumber:  cr.Line,
		})
	}
	return edges
}

// scanMemberships creates edges for team-agent membership relations.
func (s *Scanner) scanMemberships(ctx context.Context) ([]Edge, error) {
	if s.relationStore == nil {
		return nil, nil
	}

	teams, err := s.teamStore.List(ctx)
	if err != nil {
		return nil, err
	}

	var edges []Edge
	for _, team := range teams {
		members, err := s.relationStore.ListTeamMembers(ctx, team.ID)
		if err != nil {
			continue
		}
		for _, m := range members {
			edges = append(edges, Edge{
				From: team.ID,
				To:   m.AgentID,
				Kind: EdgeMembership,
			})
		}
	}
	return edges, nil
}

// extractRefsFromContent extracts skill references from text content.
// This is a pure function for testability (ported from xrefs/scanner.go).
func extractRefsFromContent(content string, validIDs map[string]bool) []extractedRef {
	lines := strings.Split(content, "\n")
	type dedupeKey struct {
		skillID  string
		edgeKind EdgeKind
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
				key := dedupeKey{id, EdgeCLIRead}
				if seen[key] {
					continue
				}
				seen[key] = true
				results = append(results, extractedRef{
					skillID:    id,
					edgeKind:   EdgeCLIRead,
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
			key := dedupeKey{id, EdgeBoldListed}
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, extractedRef{
				skillID:    id,
				edgeKind:   EdgeBoldListed,
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
			key := dedupeKey{id, EdgePathRef}
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, extractedRef{
				skillID:    id,
				edgeKind:   EdgePathRef,
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
	if strings.ContainsAny(token, "{}<>") {
		return false
	}
	return true
}

// cliNodeID derives a node ID for a CLI reference from its command string.
// It uses the first word (the command name) prefixed with "cli:".
func cliNodeID(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return "cli:unknown"
	}
	return "cli:" + fields[0]
}

// parseCommandParts returns the command name and first non-flag argument.
func parseCommandParts(cmd string) (command string, subcommand string) {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return "", ""
	}
	command = fields[0]
	for _, field := range fields[1:] {
		if strings.HasPrefix(field, "-") {
			continue
		}
		return command, field
	}
	return command, ""
}
