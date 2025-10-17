package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TaskParsingSession struct {
	ID                 uuid.UUID `json:"id"`
	AppID             uuid.UUID `json:"app_id"`
	RawText           string    `json:"raw_text"`
	InputType         string    `json:"input_type"`
	SubmittedBy       string    `json:"submitted_by"`
	Processed         bool      `json:"processed"`
	TasksExtracted    int       `json:"tasks_extracted"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type ParsedTask struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Priority       string   `json:"priority"`
	Tags           []string `json:"tags"`
	EstimatedHours float64  `json:"estimated_hours"`
}

type TaskParsingResult struct {
	Success      bool         `json:"success"`
	SessionID    uuid.UUID    `json:"session_id"`
	TasksCreated int          `json:"tasks_created"`
	Tasks        []Task       `json:"tasks"`
	Error        string       `json:"error,omitempty"`
}

func (s *TaskPlannerService) ParseText(w http.ResponseWriter, r *http.Request) {
	var req ParseTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON payload", http.StatusBadRequest, err)
		return
	}

	// Validate required fields
	if req.AppID == "" || req.RawText == "" || req.APIToken == "" {
		HTTPError(w, "app_id, raw_text, and api_token are required", http.StatusBadRequest, nil)
		return
	}

	// Validate app and token
	app, err := s.validateAppToken(req.AppID, req.APIToken)
	if err != nil {
		HTTPError(w, "Invalid app_id or api_token", http.StatusUnauthorized, err)
		return
	}

	// Create parsing session
	sessionID := uuid.New()
	session := TaskParsingSession{
		ID:          sessionID,
		AppID:       app.ID,
		RawText:     req.RawText,
		InputType:   req.InputType,
		SubmittedBy: req.SubmittedBy,
		Processed:   false,
		CreatedAt:   time.Now(),
	}

	err = s.createParsingSession(session)
	if err != nil {
		HTTPError(w, "Failed to create parsing session", http.StatusInternalServerError, err)
		return
	}

	// Parse tasks using AI
	ctx := context.Background()
	result, err := s.parseTasksWithAI(ctx, req.RawText, app.ID, sessionID)
	if err != nil {
		s.logger.Error("Failed to parse tasks with AI", err)
		result = &TaskParsingResult{
			Success:      false,
			SessionID:    sessionID,
			TasksCreated: 0,
			Error:        err.Error(),
		}
	}

	// Update session status
	err = s.markSessionComplete(sessionID, result.TasksCreated)
	if err != nil {
		s.logger.Error("Failed to update session status", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}
	json.NewEncoder(w).Encode(result)
}

func (s *TaskPlannerService) parseTasksWithAI(ctx context.Context, rawText string, appID, sessionID uuid.UUID) (*TaskParsingResult, error) {
	// Build prompt for task extraction
	prompt := fmt.Sprintf(`You are a task extraction AI. Parse the following unstructured text into individual, actionable tasks.

Text to parse:
%s

Extract tasks and return ONLY a JSON array with this format:
[
  {
    "title": "Task title (max 100 chars)",
    "description": "Detailed description",
    "priority": "high|medium|low",
    "tags": ["tag1", "tag2"],
    "estimated_hours": 2.5
  }
]

Rules:
- Extract all actionable items, TODOs, ideas that could become tasks
- Keep titles concise but descriptive
- Assign reasonable priorities based on context
- Add relevant tags for categorization
- Estimate completion time in hours
- Return ONLY the JSON array, no other text`, rawText)

	// Generate response using Ollama CLI
	response, err := s.callOllamaGenerate(ctx, prompt, "llama3.2", "reasoning")
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}

	// Parse AI response
	parsedTasks, err := s.parseAITaskResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	if len(parsedTasks) == 0 {
		return &TaskParsingResult{
			Success:      false,
			SessionID:    sessionID,
			TasksCreated: 0,
			Error:        "No actionable tasks found in the provided text",
		}, nil
	}

	// Insert tasks into database
	createdTasks := make([]Task, 0, len(parsedTasks))
	for i, parsedTask := range parsedTasks {
		task := Task{
			ID:          uuid.New(),
			Title:       parsedTask.Title,
			Description: parsedTask.Description,
			Status:      "backlog",
			Priority:    parsedTask.Priority,
			Tags:        parsedTask.Tags,
			AppID:       appID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := s.insertTask(task, sessionID, parsedTask.EstimatedHours, i)
		if err != nil {
			s.logger.Error("Failed to insert task", err)
			continue
		}

		// Generate embedding for semantic search
		go s.generateTaskEmbedding(context.Background(), task.ID, task.Title, task.Description)

		createdTasks = append(createdTasks, task)
	}

	return &TaskParsingResult{
		Success:      true,
		SessionID:    sessionID,
		TasksCreated: len(createdTasks),
		Tasks:        createdTasks,
	}, nil
}

func (s *TaskPlannerService) callOllamaGenerate(ctx context.Context, prompt, model, taskType string) (string, error) {
	// Use vrooli CLI to interact with Ollama
	cmd := exec.CommandContext(ctx, "bash", "/vrooli/cli/vrooli", "resource", "ollama", "generate", prompt, "--model", model, "--type", taskType, "--quiet")
	
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ollama command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func (s *TaskPlannerService) parseAITaskResponse(response string) ([]ParsedTask, error) {
	// Clean response and extract JSON
	cleanResponse := strings.TrimSpace(response)
	
	// Remove any markdown formatting
	cleanResponse = regexp.MustCompile("```json\\n?|```\\n?").ReplaceAllString(cleanResponse, "")
	
	// Try to find JSON array
	jsonMatch := regexp.MustCompile(`\[[\s\S]*\]`).FindString(cleanResponse)
	if jsonMatch == "" {
		return nil, fmt.Errorf("no JSON array found in AI response")
	}

	var parsedTasks []ParsedTask
	if err := json.Unmarshal([]byte(jsonMatch), &parsedTasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Validate and clean up tasks
	var validTasks []ParsedTask
	for _, task := range parsedTasks {
		if task.Title == "" {
			continue
		}

		// Ensure title length
		if len(task.Title) > 500 {
			task.Title = task.Title[:497] + "..."
		}

		// Validate priority
		validPriorities := map[string]bool{"critical": true, "high": true, "medium": true, "low": true}
		if !validPriorities[task.Priority] {
			task.Priority = "medium"
		}

		// Ensure estimated hours is reasonable
		if task.EstimatedHours <= 0 || task.EstimatedHours > 1000 {
			task.EstimatedHours = 1.0 // Default to 1 hour
		}

		validTasks = append(validTasks, task)
	}

	return validTasks, nil
}

func (s *TaskPlannerService) validateAppToken(appID, apiToken string) (*App, error) {
	var app App
	query := `SELECT id, name, display_name, type, created_at, updated_at FROM apps WHERE id = $1 AND api_token = $2`
	
	err := s.db.QueryRow(query, appID, apiToken).Scan(
		&app.ID, &app.Name, &app.DisplayName, &app.Type, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (s *TaskPlannerService) createParsingSession(session TaskParsingSession) error {
	query := `
		INSERT INTO unstructured_sessions (id, app_id, raw_text, input_type, submitted_by, processed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err := s.db.Exec(query, session.ID, session.AppID, session.RawText, 
		session.InputType, session.SubmittedBy, session.Processed, session.CreatedAt)
	return err
}

func (s *TaskPlannerService) insertTask(task Task, sessionID uuid.UUID, estimatedHours float64, parseIndex int) error {
	tagsJSON, _ := json.Marshal(task.Tags)
	metadata := map[string]interface{}{
		"source":                "text_parser",
		"parsing_index":         parseIndex,
		"original_text_snippet": task.Title,
		"session_id":           sessionID.String(),
	}
	metadataJSON, _ := json.Marshal(metadata)

	query := `
		INSERT INTO tasks (id, app_id, title, description, status, priority, 
		                  estimated_hours, tags, confidence_score, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	
	_, err := s.db.Exec(query, task.ID, task.AppID, task.Title, task.Description,
		task.Status, task.Priority, estimatedHours, tagsJSON, 0.85, metadataJSON, 
		task.CreatedAt, task.UpdatedAt)
	return err
}

func (s *TaskPlannerService) markSessionComplete(sessionID uuid.UUID, tasksExtracted int) error {
	query := `
		UPDATE unstructured_sessions 
		SET processed = true, tasks_extracted = $1, processed_at = CURRENT_TIMESTAMP 
		WHERE id = $2`
	
	_, err := s.db.Exec(query, tasksExtracted, sessionID)
	return err
}

func (s *TaskPlannerService) generateTaskEmbedding(ctx context.Context, taskID uuid.UUID, title, description string) {
	// Generate embedding for the task
	text := fmt.Sprintf("%s - %s", title, description)
	
	cmd := exec.CommandContext(ctx, "bash", "/vrooli/cli/vrooli", "resource", "ollama", "embed", text, "--model", "nomic-embed-text", "--quiet")
	
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		s.logger.Error("Failed to generate task embedding", err)
		return
	}

	embedding := stdout.String()
	if embedding != "" {
		// Update task with embedding
		query := `UPDATE tasks SET title_embedding = $1 WHERE id = $2`
		s.db.Exec(query, embedding, taskID)
	}
}