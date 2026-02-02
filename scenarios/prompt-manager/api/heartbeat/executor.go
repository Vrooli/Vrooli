package heartbeat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"prompt-manager/store"
)

// ExecutionResult represents the result of a heartbeat execution
type ExecutionResult struct {
	TeamID    string
	AgentID   string
	RunID     string
	Status    string
	StartedAt time.Time
	EndedAt   *time.Time
	LogPath   string
	Error     error
}

// Executor handles the actual execution of heartbeats
type Executor struct {
	teamStore   *store.FileTeamStore
	agentStore  *store.FileAgentStore
	agentClient *AgentManagerClient
	vrooliRoot  string
}

// NewExecutor creates a new heartbeat executor
func NewExecutor(
	teamStore *store.FileTeamStore,
	agentStore *store.FileAgentStore,
	agentClient *AgentManagerClient,
	vrooliRoot string,
) *Executor {
	return &Executor{
		teamStore:   teamStore,
		agentStore:  agentStore,
		agentClient: agentClient,
		vrooliRoot:  vrooliRoot,
	}
}

// Execute runs a heartbeat for a team member
func (e *Executor) Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error) {
	startedAt := time.Now().UTC()
	result := &ExecutionResult{
		TeamID:    teamID,
		AgentID:   agentID,
		StartedAt: startedAt,
	}

	// Build the prompt
	prompt, err := e.buildPrompt(ctx, teamID, agentID)
	if err != nil {
		result.Error = fmt.Errorf("building prompt: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Update config with running status
	config, err := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		result.Error = fmt.Errorf("getting heartbeat config: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Generate log path
	timestamp := startedAt.Format("2006-01-02T15-04-05Z")
	logPath := e.teamStore.GetMemberLogPath(teamID, agentID, timestamp)
	result.LogPath = logPath

	// Update config to running
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt: startedAt.Format(time.RFC3339),
		Status:    store.HeartbeatStatusRunning,
		LogPath:   logPath,
	}
	if err := e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config); err != nil {
		result.Error = fmt.Errorf("updating config to running: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Create task in agent-manager
	task := &Task{
		Prompt:      prompt,
		WorkingDir:  e.vrooliRoot,
		Description: fmt.Sprintf("Heartbeat: %s/%s", teamID, agentID),
	}

	createdTask, err := e.agentClient.CreateTask(ctx, task)
	if err != nil {
		result.Error = fmt.Errorf("creating task: %w", err)
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error.Error())
		return result, result.Error
	}

	// Create run
	runTag := fmt.Sprintf("heartbeat-%s-%s-%s", teamID, agentID, timestamp)
	runReq := &CreateRunRequest{
		TaskID: createdTask.ID,
		ProfileRef: &ProfileRef{
			ProfileKey: profileKey,
		},
		Tag:     &runTag,
		RunMode: "RUN_MODE_IN_PLACE",
	}

	run, err := e.agentClient.CreateRun(ctx, runReq)
	if err != nil {
		result.Error = fmt.Errorf("creating run: %w", err)
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error.Error())
		return result, result.Error
	}

	result.RunID = run.ID
	config.LastExecution.RunID = run.ID
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)

	// Wait for completion asynchronously
	go e.waitForCompletion(context.Background(), teamID, agentID, run.ID, startedAt, logPath)

	result.Status = store.HeartbeatStatusRunning
	return result, nil
}

// buildPrompt constructs the full prompt for heartbeat execution
func (e *Executor) buildPrompt(ctx context.Context, teamID, agentID string) (string, error) {
	var parts []string

	// 1. Agent markdown files (global personality + notes)
	agentFiles, err := e.agentStore.ListFiles(ctx, agentID)
	if err == nil && len(agentFiles) > 0 {
		var markdownFiles []store.AgentFileEntry
		for _, entry := range agentFiles {
			if entry.IsDir {
				continue
			}
			if strings.HasSuffix(strings.ToLower(entry.Path), ".md") {
				markdownFiles = append(markdownFiles, entry)
			}
		}

		if len(markdownFiles) > 0 {
			sort.Slice(markdownFiles, func(i, j int) bool {
				a := strings.ToLower(markdownFiles[i].Path)
				b := strings.ToLower(markdownFiles[j].Path)
				if a == "soul.md" {
					return b != "soul.md"
				}
				if b == "soul.md" {
					return false
				}
				return a < b
			})

			section := "# Agent Files (Markdown)\n\n"
			for _, entry := range markdownFiles {
				content, err := e.agentStore.ReadFile(ctx, agentID, entry.Path)
				if err != nil {
					continue
				}
				section += fmt.Sprintf("## %s\n\n%s\n\n", entry.Path, content)
			}
			parts = append(parts, section)
		}
	}

	// 2. Team member RESPONSIBILITIES.md
	responsibilities, err := e.teamStore.GetResponsibilities(ctx, teamID, agentID)
	if err == nil && responsibilities != "" {
		parts = append(parts, "# Team Responsibilities (RESPONSIBILITIES.md)\n\n"+responsibilities)
	}

	// 3. Team relationships + coordination commands
	if section := e.buildRelationshipSection(ctx, teamID, agentID); section != "" {
		parts = append(parts, section)
	}

	// 4. Team inbox messages
	if section := e.buildInboxSection(ctx, teamID, agentID); section != "" {
		parts = append(parts, section)
	}

	// 5. HEARTBEAT.md (the specific task)
	heartbeatInstructions, err := e.teamStore.GetHeartbeatInstructions(ctx, teamID, agentID)
	if err == nil && heartbeatInstructions != "" {
		parts = append(parts, "# Heartbeat Task (HEARTBEAT.md)\n\n"+heartbeatInstructions)
	} else {
		// No heartbeat instructions - use default task
		parts = append(parts, "# Heartbeat Task\n\nNo specific heartbeat instructions defined. Please review your responsibilities and perform any pending work.")
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("no content available for heartbeat prompt")
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

func (e *Executor) buildRelationshipSection(ctx context.Context, teamID, agentID string) string {
	org, err := e.teamStore.GetOrgChart(ctx, teamID)
	if err != nil {
		return ""
	}

	teamName := teamID
	if team, err := e.teamStore.Get(ctx, teamID); err == nil && team.DisplayName != "" {
		teamName = team.DisplayName
	}

	var managerID string
	var reportIDs []string
	for _, edge := range org.Edges {
		if edge.ReportAgentID == agentID {
			managerID = edge.ManagerAgentID
		}
		if edge.ManagerAgentID == agentID {
			reportIDs = append(reportIDs, edge.ReportAgentID)
		}
	}

	resolveAgentLabel := func(id string) string {
		if id == "" {
			return ""
		}
		agent, err := e.agentStore.Get(ctx, id)
		if err != nil || agent.DisplayName == "" || agent.DisplayName == id {
			return id
		}
		return fmt.Sprintf("%s (%s)", agent.DisplayName, id)
	}

	managerLabel := "None"
	if managerID != "" {
		managerLabel = resolveAgentLabel(managerID)
	}

	var reportsLabel string
	if len(reportIDs) == 0 {
		reportsLabel = "None"
	} else {
		labels := make([]string, 0, len(reportIDs))
		for _, reportID := range reportIDs {
			labels = append(labels, resolveAgentLabel(reportID))
		}
		sort.Strings(labels)
		reportsLabel = strings.Join(labels, ", ")
	}

	section := "# Team Relationships\n\n"
	section += fmt.Sprintf("Team: %s (%s)\n", teamName, teamID)
	section += fmt.Sprintf("Your agent ID: %s\n\n", agentID)
	section += fmt.Sprintf("- Reports to: %s\n", managerLabel)
	section += fmt.Sprintf("- Direct reports: %s\n\n", reportsLabel)
	section += "## Coordination Commands\n\n"
	section += fmt.Sprintf("- Send a directive: `prompt-manager team message-send %s <recipient-agent-id> --from=%s --content \"...\"`\n", teamID, agentID)
	section += fmt.Sprintf("- Check your inbox: `prompt-manager team message-list %s %s`\n", teamID, agentID)
	section += fmt.Sprintf("- Delete a message: `prompt-manager team message-delete %s %s <message-id>`\n", teamID, agentID)
	section += fmt.Sprintf("- Clear inbox: `prompt-manager team message-clear %s %s`\n", teamID, agentID)
	return section
}

func (e *Executor) buildInboxSection(ctx context.Context, teamID, agentID string) string {
	inbox, err := e.teamStore.GetInbox(ctx, teamID, agentID)
	if err != nil || len(inbox.Messages) == 0 {
		return ""
	}

	messages := make([]store.TeamMessage, len(inbox.Messages))
	copy(messages, inbox.Messages)
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt < messages[j].CreatedAt
	})

	resolveAgentLabel := func(id string) string {
		agent, err := e.agentStore.Get(ctx, id)
		if err != nil || agent.DisplayName == "" || agent.DisplayName == id {
			return id
		}
		return fmt.Sprintf("%s (%s)", agent.DisplayName, id)
	}

	section := "# Team Inbox\n\n"
	section += fmt.Sprintf("You have %d pending message(s):\n\n", len(messages))
	for _, message := range messages {
		fromLabel := resolveAgentLabel(message.FromAgentID)
		section += fmt.Sprintf("## %s\n\n", message.ID)
		section += fmt.Sprintf("From: %s\n", fromLabel)
		section += fmt.Sprintf("Sent: %s\n\n", message.CreatedAt)
		section += message.Content + "\n\n"
	}
	return section
}

// waitForCompletion polls for run completion and updates config
func (e *Executor) waitForCompletion(ctx context.Context, teamID, agentID, runID string, startedAt time.Time, logPath string) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	run, err := e.agentClient.WaitForRun(timeoutCtx, runID, 5*time.Second)

	endedAt := time.Now().UTC()

	// Get current config
	config, configErr := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if configErr != nil {
		return
	}

	if err != nil {
		config.LastExecution = &store.HeartbeatExecResult{
			StartedAt: startedAt.Format(time.RFC3339),
			EndedAt:   endedAt.Format(time.RFC3339),
			Status:    store.HeartbeatStatusFailed,
			RunID:     runID,
			LogPath:   logPath,
			Error:     err.Error(),
		}
	} else {
		status := store.HeartbeatStatusCompleted
		errMsg := ""
		if run.Status == "RUN_STATUS_FAILED" || run.Status == "failed" {
			status = store.HeartbeatStatusFailed
			errMsg = run.Error
		}

		config.LastExecution = &store.HeartbeatExecResult{
			StartedAt: startedAt.Format(time.RFC3339),
			EndedAt:   endedAt.Format(time.RFC3339),
			Status:    status,
			RunID:     runID,
			LogPath:   logPath,
			Error:     errMsg,
		}
	}

	// Write log file
	logContent := fmt.Sprintf("Heartbeat execution for %s/%s\n", teamID, agentID)
	logContent += fmt.Sprintf("Started: %s\n", startedAt.Format(time.RFC3339))
	logContent += fmt.Sprintf("Ended: %s\n", endedAt.Format(time.RFC3339))
	logContent += fmt.Sprintf("Run ID: %s\n", runID)
	logContent += fmt.Sprintf("Status: %s\n", config.LastExecution.Status)
	if config.LastExecution.Error != "" {
		logContent += fmt.Sprintf("Error: %s\n", config.LastExecution.Error)
	}

	// Ensure logs directory exists and write log
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err == nil {
		_ = os.WriteFile(logPath, []byte(logContent), 0o644)
	}

	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
}

// updateConfigFailed updates config with failed status
func (e *Executor) updateConfigFailed(ctx context.Context, teamID, agentID string, config *store.HeartbeatConfig, startedAt time.Time, errMsg string) {
	endedAt := time.Now().UTC()
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt: startedAt.Format(time.RFC3339),
		EndedAt:   endedAt.Format(time.RFC3339),
		Status:    store.HeartbeatStatusFailed,
		Error:     errMsg,
	}
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
}

// TriggerManual manually triggers a heartbeat execution
func (e *Executor) TriggerManual(ctx context.Context, teamID, agentID string) (*ExecutionResult, error) {
	// Get heartbeat config to get profile key
	config, err := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		return nil, fmt.Errorf("getting heartbeat config: %w", err)
	}

	profileKey := "prompt-manager-heartbeat"
	if config != nil && config.ProfileKey != "" {
		profileKey = config.ProfileKey
	}

	return e.Execute(ctx, teamID, agentID, profileKey)
}
