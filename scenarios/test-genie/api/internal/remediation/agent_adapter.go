package remediation

import (
	"context"
	"fmt"
	"path"
	"strings"

	"test-genie/agentmanager"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Launcher is the only agent-execution seam remediation needs. It does not
// model sandbox, network, tool, merge, or review policy because those belong
// to Agent Manager profiles.
type Launcher interface {
	Launch(context.Context, Job, string) (Attribution, error)
	Cancel(context.Context, Job) error
}

type AgentManagerAdapter struct{ agents *agentmanager.AgentService }

func NewAgentManagerAdapter(agents *agentmanager.AgentService) *AgentManagerAdapter {
	return &AgentManagerAdapter{agents: agents}
}

func (a *AgentManagerAdapter) Launch(ctx context.Context, job Job, roleRef string) (Attribution, error) {
	if a == nil || a.agents == nil {
		return Attribution{}, fmt.Errorf("agent-manager remediation adapter is unavailable")
	}
	if !a.agents.IsAvailable(ctx) {
		return Attribution{}, fmt.Errorf("agent-manager is not available")
	}
	roleRef = strings.TrimSpace(roleRef)
	if roleRef == "" {
		return Attribution{}, fmt.Errorf("roleRef is required")
	}
	task := &domainpb.Task{
		Title: "Remediate Test Genie findings for " + job.Scenario, Description: taskPacket(job),
		ScopePath: path.Join("scenarios", job.Scenario), CreatedBy: "test-genie",
	}
	result, err := a.agents.SpawnRemediation(ctx, agentmanager.RemediationSpawnRequest{Task: task, Tag: fmt.Sprintf("test-genie-remediation-%s-%d", job.ID, job.LaunchAttempt), RoleRef: roleRef, IdempotencyKey: launchIdempotencyKey(job)})
	if err != nil {
		return Attribution{}, err
	}
	return Attribution{TaskID: result.TaskID, RunID: result.RunID, RoleRef: roleRef, ResolvedProfile: a.agents.GetProfileID()}, nil
}

func (a *AgentManagerAdapter) Cancel(ctx context.Context, job Job) error {
	if a == nil || a.agents == nil || strings.TrimSpace(job.Attribution.RunID) == "" {
		return nil
	}
	return a.agents.StopRun(ctx, job.Attribution.RunID)
}

func taskPacket(job Job) string {
	selected := make(map[string]struct{}, len(job.SelectedFindingIDs))
	for _, id := range job.SelectedFindingIDs {
		selected[id] = struct{}{}
	}
	var b strings.Builder
	b.WriteString("Remediate the selected findings using the immutable Test Genie execution evidence below.\n\n")
	fmt.Fprintf(&b, "Scenario: %s\nSource execution: %s\nSource run: %s\n", job.Scenario, job.Source.SourceExecutionID, job.Source.SourceRunID)
	b.WriteString("\nSelected findings:\n")
	for _, finding := range job.Source.Findings {
		if _, ok := selected[finding.StableID]; !ok {
			continue
		}
		fmt.Fprintf(&b, "- [%s] %s (%s, %s)\n", finding.StableID, finding.Message, finding.Severity, finding.Class)
		if len(finding.Locations) > 0 {
			fmt.Fprintf(&b, "  Locations: %s\n", strings.Join(finding.Locations, ", "))
		}
		if finding.Suggestion != "" {
			fmt.Fprintf(&b, "  Suggested remediation: %s\n", finding.Suggestion)
		}
	}
	phaseNames := make(map[string]struct{})
	for _, finding := range job.Source.Findings {
		if _, ok := selected[finding.StableID]; ok {
			phaseNames[finding.Phase] = struct{}{}
		}
	}
	b.WriteString("\nRelevant phase evidence:\n")
	for _, phase := range job.Source.Phases {
		if _, ok := phaseNames[phase.Name]; !ok {
			continue
		}
		fmt.Fprintf(&b, "- %s", phase.Name)
		if phase.DocsPath != "" {
			fmt.Fprintf(&b, " (docs: %s)", phase.DocsPath)
		}
		b.WriteByte('\n')
		if phase.Remediation != "" {
			fmt.Fprintf(&b, "  Phase guidance: %s\n", phase.Remediation)
		}
		if phase.RunnabilityVerdict != "" {
			fmt.Fprintf(&b, "  Runnability: %s", phase.RunnabilityVerdict)
			if phase.RunnabilityReason != "" {
				fmt.Fprintf(&b, " (%s)", phase.RunnabilityReason)
			}
			b.WriteByte('\n')
		}
	}
	if len(job.SelectedRequirementIDs) > 0 {
		selectedRequirements := make(map[string]struct{}, len(job.SelectedRequirementIDs))
		for _, id := range job.SelectedRequirementIDs {
			selectedRequirements[id] = struct{}{}
		}
		b.WriteString("\nSelected requirement evidence:\n")
		for _, requirement := range job.Source.Requirements {
			if _, ok := selectedRequirements[requirement.ID]; !ok {
				continue
			}
			fmt.Fprintf(&b, "- [%s] %s (%s)\n", requirement.ID, requirement.Title, requirement.LiveStatus)
			if len(requirement.Validations) > 0 {
				fmt.Fprintf(&b, "  Validations: %s\n", strings.Join(requirement.Validations, ", "))
			}
		}
	}
	b.WriteString("\nConstraints: work only within the scenario scope. Do not treat this task completion as verification. Test Genie will run a fresh server-owned verification execution and compare stable finding IDs.\n")
	if job.AdditionalContext != "" {
		fmt.Fprintf(&b, "\nOperator context: %s\n", job.AdditionalContext)
	}
	return b.String()
}
