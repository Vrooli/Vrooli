package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type TaskResearchRequest struct {
	TaskID       string `json:"task_id"`
	ForceRefresh bool   `json:"force_refresh,omitempty"`
}

type TaskResearchResult struct {
	Success          bool                   `json:"success"`
	TaskID          uuid.UUID              `json:"task_id"`
	ResearchSummary string                 `json:"research_summary"`
	Requirements    []string               `json:"requirements"`
	Dependencies    []string               `json:"dependencies"`
	Recommendations []string               `json:"recommendations"`
	EstimatedHours  float64                `json:"estimated_hours"`
	Complexity      string                 `json:"complexity"`
	ResearchData    map[string]interface{} `json:"research_data"`
	Error           string                 `json:"error,omitempty"`
}

type TaskResearchContext struct {
	Task           *Task  `json:"task"`
	AppContext     *App   `json:"app_context"`
	RelatedTasks   []Task `json:"related_tasks"`
	CompletedTasks []Task `json:"completed_tasks"`
}

func (s *TaskPlannerService) ResearchTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskIDStr := vars["taskId"]
	if taskIDStr == "" {
		// Try to get from request body
		var req TaskResearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.TaskID != "" {
			taskIDStr = req.TaskID
		}
	}

	if taskIDStr == "" {
		HTTPError(w, "task_id is required", http.StatusBadRequest, nil)
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		HTTPError(w, "Invalid task ID format", http.StatusBadRequest, err)
		return
	}

	// Get task details
	task, err := s.getTaskByID(taskID)
	if err != nil {
		HTTPError(w, "Task not found", http.StatusNotFound, err)
		return
	}

	// Only research tasks in backlog status
	if task.Status != "backlog" {
		HTTPError(w, "Task must be in backlog status to research", http.StatusBadRequest, nil)
		return
	}

	// Perform research
	ctx := context.Background()
	result, err := s.performTaskResearch(ctx, task)
	if err != nil {
		s.logger.Error("Failed to research task", err)
		result = &TaskResearchResult{
			Success: false,
			TaskID:  taskID,
			Error:   err.Error(),
		}
	}

	// Update task with research results
	if result.Success {
		err = s.updateTaskWithResearch(task.ID, result)
		if err != nil {
			s.logger.Error("Failed to update task with research", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(result)
}

func (s *TaskPlannerService) performTaskResearch(ctx context.Context, task *Task) (*TaskResearchResult, error) {
	s.logger.Info(fmt.Sprintf("Starting research for task: %s", task.Title))

	// Gather research context
	context, err := s.gatherResearchContext(task)
	if err != nil {
		return nil, fmt.Errorf("failed to gather research context: %w", err)
	}

	// Build research prompt
	prompt := s.buildResearchPrompt(task, context)

	// Get AI research
	aiResponse, err := s.callOllamaGenerate(ctx, prompt, "llama3.2", "analysis")
	if err != nil {
		return nil, fmt.Errorf("failed to get AI research: %w", err)
	}

	// Parse research results
	result, err := s.parseResearchResults(aiResponse, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse research results: %w", err)
	}

	// Enhance with additional context
	s.enhanceResearchWithContext(result, context)

	return result, nil
}

func (s *TaskPlannerService) gatherResearchContext(task *Task) (*TaskResearchContext, error) {
	// Get app context
	app, err := s.getAppByID(task.AppID)
	if err != nil {
		return nil, err
	}

	// Get related tasks (similar tags or keywords)
	relatedTasks, err := s.findRelatedTasks(task)
	if err != nil {
		s.logger.Warn("Failed to find related tasks", err)
		relatedTasks = []Task{}
	}

	// Get recently completed tasks from the same app
	completedTasks, err := s.getRecentCompletedTasks(task.AppID, 5)
	if err != nil {
		s.logger.Warn("Failed to get completed tasks", err)
		completedTasks = []Task{}
	}

	return &TaskResearchContext{
		Task:           task,
		AppContext:     app,
		RelatedTasks:   relatedTasks,
		CompletedTasks: completedTasks,
	}, nil
}

func (s *TaskPlannerService) buildResearchPrompt(task *Task, context *TaskResearchContext) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf(`You are a technical research assistant. Analyze this task and provide detailed research to help with implementation.

TASK TO RESEARCH:
Title: %s
Description: %s
Priority: %s
Current Status: %s
Tags: %s

APPLICATION CONTEXT:
App Name: %s
App Type: %s`, 
		task.Title, task.Description, task.Priority, task.Status, 
		strings.Join(task.Tags, ", "), context.AppContext.Name, context.AppContext.Type))

	if len(context.RelatedTasks) > 0 {
		prompt.WriteString("\n\nRELATED TASKS:\n")
		for i, relTask := range context.RelatedTasks {
			if i >= 3 { // Limit to top 3 related tasks
				break
			}
			prompt.WriteString(fmt.Sprintf("- %s (Status: %s)\n", relTask.Title, relTask.Status))
		}
	}

	if len(context.CompletedTasks) > 0 {
		prompt.WriteString("\n\nRECENT COMPLETED TASKS:\n")
		for i, compTask := range context.CompletedTasks {
			if i >= 3 { // Limit to top 3 completed tasks
				break
			}
			prompt.WriteString(fmt.Sprintf("- %s\n", compTask.Title))
		}
	}

	prompt.WriteString(`

RESEARCH REQUIREMENTS:
Provide a comprehensive analysis in JSON format with these fields:

{
  "research_summary": "2-3 sentence summary of what this task involves",
  "requirements": ["req1", "req2", "req3"],
  "dependencies": ["dependency1", "dependency2"],
  "recommendations": ["recommendation1", "recommendation2", "recommendation3"],
  "estimated_hours": 4.5,
  "complexity": "low|medium|high|very_high",
  "technical_considerations": ["consideration1", "consideration2"],
  "potential_challenges": ["challenge1", "challenge2"],
  "success_criteria": ["criteria1", "criteria2"]
}

Focus on:
- Technical requirements and dependencies
- Implementation approach recommendations
- Realistic time estimates
- Potential roadblocks or challenges
- Success criteria for completion

Return ONLY the JSON object, no additional text.`)

	return prompt.String()
}

func (s *TaskPlannerService) parseResearchResults(aiResponse string, taskID uuid.UUID) (*TaskResearchResult, error) {
	// Clean and extract JSON
	cleanResponse := strings.TrimSpace(aiResponse)
	cleanResponse = strings.Trim(cleanResponse, "`")
	
	// Try to parse as JSON
	var researchData map[string]interface{}
	if err := json.Unmarshal([]byte(cleanResponse), &researchData); err != nil {
		// If direct parsing fails, try to extract JSON from response
		jsonStart := strings.Index(cleanResponse, "{")
		jsonEnd := strings.LastIndex(cleanResponse, "}") + 1
		if jsonStart >= 0 && jsonEnd > jsonStart {
			jsonStr := cleanResponse[jsonStart:jsonEnd]
			if err := json.Unmarshal([]byte(jsonStr), &researchData); err != nil {
				return nil, fmt.Errorf("failed to parse JSON from AI response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in AI response")
		}
	}

	// Extract fields with defaults
	result := &TaskResearchResult{
		Success:        true,
		TaskID:        taskID,
		ResearchData:  researchData,
	}

	// Parse specific fields
	if summary, ok := researchData["research_summary"].(string); ok {
		result.ResearchSummary = summary
	}

	if requirements, ok := researchData["requirements"].([]interface{}); ok {
		for _, req := range requirements {
			if reqStr, ok := req.(string); ok {
				result.Requirements = append(result.Requirements, reqStr)
			}
		}
	}

	if dependencies, ok := researchData["dependencies"].([]interface{}); ok {
		for _, dep := range dependencies {
			if depStr, ok := dep.(string); ok {
				result.Dependencies = append(result.Dependencies, depStr)
			}
		}
	}

	if recommendations, ok := researchData["recommendations"].([]interface{}); ok {
		for _, rec := range recommendations {
			if recStr, ok := rec.(string); ok {
				result.Recommendations = append(result.Recommendations, recStr)
			}
		}
	}

	if hours, ok := researchData["estimated_hours"].(float64); ok {
		result.EstimatedHours = hours
	} else if hours, ok := researchData["estimated_hours"].(int); ok {
		result.EstimatedHours = float64(hours)
	} else {
		result.EstimatedHours = 2.0 // Default
	}

	if complexity, ok := researchData["complexity"].(string); ok {
		validComplexities := map[string]bool{"low": true, "medium": true, "high": true, "very_high": true}
		if validComplexities[complexity] {
			result.Complexity = complexity
		} else {
			result.Complexity = "medium"
		}
	} else {
		result.Complexity = "medium"
	}

	return result, nil
}

func (s *TaskPlannerService) enhanceResearchWithContext(result *TaskResearchResult, context *TaskResearchContext) {
	// Add context-specific recommendations
	if context.AppContext.Type == "web-application" {
		result.Recommendations = append(result.Recommendations, "Consider web accessibility standards")
		result.Recommendations = append(result.Recommendations, "Ensure responsive design compatibility")
	}

	// Adjust estimates based on related tasks
	if len(context.CompletedTasks) > 0 {
		// If we have similar completed tasks, we might have better estimates
		avgEstimate := 0.0
		count := 0
		for _, task := range context.CompletedTasks {
			// This would need metadata about actual time taken
			// For now, just use a heuristic
			avgEstimate += 3.0 // Placeholder
			count++
		}
		if count > 0 && avgEstimate/float64(count) > 0 {
			// Adjust estimate slightly toward historical average
			historicalAvg := avgEstimate / float64(count)
			result.EstimatedHours = (result.EstimatedHours*0.7 + historicalAvg*0.3)
		}
	}
}

func (s *TaskPlannerService) getTaskByID(taskID uuid.UUID) (*Task, error) {
	query := `
		SELECT t.id, t.title, t.description, t.status, t.priority, 
		       COALESCE(t.tags, '[]'::jsonb) as tags, t.app_id, a.name as app_name,
		       t.created_at, t.updated_at
		FROM tasks t
		JOIN apps a ON t.app_id = a.id
		WHERE t.id = $1`

	var task Task
	var tagsJSON []byte
	
	err := s.db.QueryRow(query, taskID).Scan(
		&task.ID, &task.Title, &task.Description, &task.Status, 
		&task.Priority, &tagsJSON, &task.AppID, &task.AppName,
		&task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse tags
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &task.Tags)
	}

	return &task, nil
}

func (s *TaskPlannerService) getAppByID(appID uuid.UUID) (*App, error) {
	query := `
		SELECT id, name, display_name, type, total_tasks, completed_tasks, created_at, updated_at
		FROM apps WHERE id = $1`

	var app App
	err := s.db.QueryRow(query, appID).Scan(
		&app.ID, &app.Name, &app.DisplayName, &app.Type,
		&app.TotalTasks, &app.CompletedTasks, &app.CreatedAt, &app.UpdatedAt,
	)
	
	return &app, err
}

func (s *TaskPlannerService) findRelatedTasks(task *Task) ([]Task, error) {
	// Simple keyword-based matching for now
	// In a real implementation, you'd use vector similarity
	keywords := strings.Fields(strings.ToLower(task.Title + " " + task.Description))
	if len(keywords) == 0 {
		return []Task{}, nil
	}

	// Build a basic text search query
	searchTerms := strings.Join(keywords[:min(len(keywords), 3)], " | ")
	query := `
		SELECT t.id, t.title, t.description, t.status, t.priority,
		       COALESCE(t.tags, '[]'::jsonb) as tags, t.app_id, a.name as app_name,
		       t.created_at, t.updated_at
		FROM tasks t
		JOIN apps a ON t.app_id = a.id
		WHERE t.id != $1 AND (
			t.title ILIKE $2 OR 
			t.description ILIKE $2 OR
			EXISTS (SELECT 1 FROM jsonb_array_elements_text(t.tags) tag WHERE tag ILIKE ANY($3))
		)
		ORDER BY t.updated_at DESC
		LIMIT 5`

	// Create array of tag patterns
	tagPatterns := make([]string, len(task.Tags))
	for i, tag := range task.Tags {
		tagPatterns[i] = "%" + tag + "%"
	}

	rows, err := s.db.Query(query, task.ID, "%"+searchTerms+"%", tagPatterns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var tagsJSON []byte
		
		err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&tagsJSON, &t.AppID, &t.AppName, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &t.Tags)
		}

		tasks = append(tasks, t)
	}

	return tasks, nil
}

func (s *TaskPlannerService) getRecentCompletedTasks(appID uuid.UUID, limit int) ([]Task, error) {
	query := `
		SELECT t.id, t.title, t.description, t.status, t.priority,
		       COALESCE(t.tags, '[]'::jsonb) as tags, t.app_id, a.name as app_name,
		       t.created_at, t.updated_at
		FROM tasks t
		JOIN apps a ON t.app_id = a.id
		WHERE t.app_id = $1 AND t.status = 'completed'
		ORDER BY t.updated_at DESC
		LIMIT $2`

	rows, err := s.db.Query(query, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var tagsJSON []byte
		
		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
			&tagsJSON, &task.AppID, &task.AppName, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &task.Tags)
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *TaskPlannerService) updateTaskWithResearch(taskID uuid.UUID, result *TaskResearchResult) error {
	// Store research results in task metadata
	researchMetadata := map[string]interface{}{
		"research_completed_at": time.Now().Unix(),
		"research_summary":      result.ResearchSummary,
		"requirements":         result.Requirements,
		"dependencies":         result.Dependencies,
		"recommendations":      result.Recommendations,
		"complexity":           result.Complexity,
		"ai_estimated_hours":   result.EstimatedHours,
		"research_data":        result.ResearchData,
	}

	metadataJSON, _ := json.Marshal(researchMetadata)
	
	query := `
		UPDATE tasks 
		SET metadata = COALESCE(metadata, '{}'::jsonb) || $1::jsonb,
		    estimated_hours = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $3`
	
	_, err := s.db.Exec(query, metadataJSON, result.EstimatedHours, taskID)
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}