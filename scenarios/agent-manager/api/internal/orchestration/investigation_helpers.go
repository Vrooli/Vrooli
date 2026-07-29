// Package orchestration: investigation helpers retain resume-only context and
// recurrence formatting outside the bounded workflow-dispatch implementation.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/findings"

	"github.com/google/uuid"
)

func recurrenceAttachment(runID uuid.UUID, prior []findings.Finding) domain.ContextAttachment {
	var content strings.Builder
	content.WriteString("### Prior findings and recurrence\n\n")
	for _, finding := range prior {
		applied := "no recorded decision"
		if finding.Decision != "" {
			applied = "decision=" + finding.Decision
		}
		fingerprint := finding.Fingerprint
		if len(fingerprint) > 12 {
			fingerprint = fingerprint[:12]
		}
		fmt.Fprintf(&content, "- fingerprint=%s occurrences=%d %s: %s\n", fingerprint, finding.Occurrences, applied, finding.Recommendation)
	}
	return domain.ContextAttachment{Type: "note", Key: "prior-findings-" + shortID(runID), Label: "Prior Findings " + shortID(runID), Content: content.String(), Format: "markdown", Priority: "medium", Summary: fmt.Sprintf("%d prior finding(s), ordered by recurrence", len(prior)), Tags: []string{"findings", "recurrence", "investigation"}}
}

func buildAgentSetupAttachment(profile *domain.AgentProfile, projectRoot string, short string) (domain.ContextAttachment, bool) {
	storeRoot := filepath.Join(projectRoot, "scenarios", "prompt-manager", "store")
	agentDir := filepath.Join(storeRoot, "agents", profile.ProfileKey)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### Agent Profile: `%s`\n\n**Name**: %s\n", profile.ProfileKey, profile.Name))
	if profile.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description**: %s\n", profile.Description))
	}
	sb.WriteString(fmt.Sprintf("**Role**: %s\n", profile.RoleRef))
	if len(profile.AllowedTools) > 0 {
		sb.WriteString(fmt.Sprintf("**Allowed Tools**: %s\n", strings.Join(profile.AllowedTools, ", ")))
	}
	if len(profile.DeniedTools) > 0 {
		sb.WriteString(fmt.Sprintf("**Denied Tools**: %s\n", strings.Join(profile.DeniedTools, ", ")))
	}
	sb.WriteString("\n")
	agentDirExists := false
	if _, err := os.Stat(agentDir); err == nil {
		agentDirExists = true
		sb.WriteString(fmt.Sprintf("**Agent Directory**: `%s/`\n", agentDir))
		for _, file := range []struct{ name, desc string }{{"agent.json", "Agent metadata and configuration"}, {"SOUL.md", "Core identity, boundaries, and domain focus"}, {"AGENTS.md", "Workflow procedures and coordination"}, {"TOOLS.md", "Available skills and resource access"}} {
			if _, err := os.Stat(filepath.Join(agentDir, file.name)); err == nil {
				sb.WriteString(fmt.Sprintf("- `%s` — %s\n", file.name, file.desc))
			}
		}
	} else {
		sb.WriteString(fmt.Sprintf("**No agent directory** at `%s/`\n", agentDir))
		sb.WriteString("This agent's prompt may be generated dynamically rather than stored as files.\nCheck the task description and runner configuration for how this agent receives its instructions.\n")
	}
	relationsDir := filepath.Join(storeRoot, "relations", "team-member")
	suffix := "__" + profile.ProfileKey + ".json"
	if entries, err := os.ReadDir(relationsDir); err == nil {
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), suffix) {
				continue
			}
			teamID := strings.TrimSuffix(entry.Name(), suffix)
			teamDir := filepath.Join(storeRoot, "teams", teamID)
			memberDir := filepath.Join(teamDir, "members", profile.ProfileKey)
			sb.WriteString(fmt.Sprintf("\n### Team: `%s`\n\n**Team Directory**: `%s/`\n", teamID, teamDir))
			for _, file := range []struct{ name, desc string }{{"team.json", "Team configuration and spawn mode"}, {"org.json", "Organizational hierarchy (reporting structure)"}, {"roles.json", "Role definitions within the team"}, {filepath.Join("shared", "TEAM.md"), "Team mission, strategy, and deployment model"}} {
				if _, err := os.Stat(filepath.Join(teamDir, file.name)); err == nil {
					sb.WriteString(fmt.Sprintf("- `%s` — %s\n", file.name, file.desc))
				}
			}
			if _, err := os.Stat(memberDir); err == nil {
				sb.WriteString(fmt.Sprintf("\n**Member Directory**: `%s/`\n", memberDir))
				for _, file := range []struct{ name, desc string }{{"heartbeat.json", "Execution schedule and last execution status"}, {"HEARTBEAT.md", "Checklist of tasks for scheduled runs"}, {"RESPONSIBILITIES.md", "Role-specific duties and deliverables"}} {
					if _, err := os.Stat(filepath.Join(memberDir, file.name)); err == nil {
						sb.WriteString(fmt.Sprintf("- `%s` — %s\n", file.name, file.desc))
					}
				}
				if entries, err := os.ReadDir(filepath.Join(memberDir, "logs")); err == nil && len(entries) > 0 {
					sb.WriteString(fmt.Sprintf("- `logs/` — %d execution log(s)\n", len(entries)))
				}
			}
			sb.WriteString(fmt.Sprintf("\n**Relation**: `%s`\n", filepath.Join(relationsDir, entry.Name())))
		}
	}
	if agentDirExists {
		sb.WriteString("\n*Read these files to understand the agent's identity, instructions, tools, team context, and scheduled responsibilities.*\n")
	}
	return domain.ContextAttachment{Type: "note", Key: fmt.Sprintf("agent-setup-%s", short), Label: fmt.Sprintf("Agent Setup %s", short), Content: sb.String(), Format: "markdown", Priority: "high", Summary: fmt.Sprintf("Profile metadata and prompt-manager paths for agent %q", profile.ProfileKey), Tags: []string{"agent", "setup", "investigation"}}, true
}

func (o *Orchestrator) buildHistoricalContext(ctx context.Context, currentRun *domain.Run, short string) (domain.ContextAttachment, bool) {
	if currentRun.AgentProfileID == nil {
		return domain.ContextAttachment{}, false
	}
	recentRuns, err := o.ListRuns(ctx, RunListOptions{ListOptions: ListOptions{Limit: 10}, AgentProfileID: currentRun.AgentProfileID})
	if err != nil || len(recentRuns) == 0 {
		return domain.ContextAttachment{}, false
	}
	var history []*domain.Run
	for _, run := range recentRuns {
		if run.ID != currentRun.ID && !strings.Contains(run.Tag, "investigation") {
			history = append(history, run)
		}
	}
	if len(history) == 0 {
		return domain.ContextAttachment{}, false
	}
	var sb strings.Builder
	sb.WriteString("### Recent Runs (same agent profile)\n\n| Run ID | Date | Tag | Status | Duration | Error |\n|--------|------|-----|--------|----------|-------|\n")
	var successCount, failCount int
	for _, run := range history {
		duration := "—"
		if run.StartedAt != nil && run.EndedAt != nil {
			duration = run.EndedAt.Sub(*run.StartedAt).Round(time.Second).String()
		}
		errMsg := run.ErrorMsg
		if errMsg == "" {
			errMsg = "—"
		} else if len(errMsg) > 60 {
			errMsg = errMsg[:60] + "..."
		}
		tag := run.Tag
		if len(tag) > 30 {
			tag = tag[:30] + "..."
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s | %s |\n", shortID(run.ID), run.CreatedAt.Format("Jan 02 15:04"), tag, run.Status, duration, errMsg)
		if run.Status == domain.RunStatusComplete {
			successCount++
		} else if run.Status == domain.RunStatusFailed {
			failCount++
		}
	}
	total := len(history)
	fmt.Fprintf(&sb, "\n**Pattern**: %d of last %d runs succeeded, %d failed\n", successCount, total, failCount)
	if failCount > 0 && successCount > 0 {
		sb.WriteString("*Compare successful vs failed runs to identify what changed.*\n")
	} else if failCount == total {
		sb.WriteString("*All recent runs failed — likely a persistent issue rather than a transient one.*\n")
	}
	return domain.ContextAttachment{Type: "note", Key: fmt.Sprintf("run-history-%s", short), Label: fmt.Sprintf("Historical Context %s", short), Content: sb.String(), Format: "markdown", Priority: "medium", Summary: fmt.Sprintf("%d recent runs: %d succeeded, %d failed", total, successCount, failCount), Tags: []string{"run", "history", "investigation"}}, true
}

func marshalJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	return string(data), err
}
func shortID(id uuid.UUID) string {
	value := id.String()
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}
func taskShortID() string { return shortID(uuid.New()) }
