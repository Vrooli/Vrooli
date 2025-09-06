package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

type Workflow struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	FolderPath   string                 `json:"folder_path"`
	FlowDef      map[string]interface{} `json:"flow_definition"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Tags         []string               `json:"tags"`
}

type Execution struct {
	ID          string    `json:"id"`
	WorkflowID  string    `json:"workflow_id"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
	Progress    int       `json:"progress"`
	CurrentStep string    `json:"current_step,omitempty"`
}

type CreateWorkflowRequest struct {
	Name           string                 `json:"name"`
	FolderPath     string                 `json:"folder_path"`
	FlowDefinition map[string]interface{} `json:"flow_definition,omitempty"`
	AIPrompt       string                 `json:"ai_prompt,omitempty"`
}

type ExecuteWorkflowRequest struct {
	Parameters         map[string]interface{} `json:"parameters,omitempty"`
	WaitForCompletion bool                   `json:"wait_for_completion"`
}

type Server struct {
	workflows  map[string]*Workflow
	executions map[string]*Execution
}

func NewServer() *Server {
	return &Server{
		workflows:  make(map[string]*Workflow),
		executions: make(map[string]*Execution),
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now(),
	})
}

func (s *Server) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workflow := &Workflow{
		ID:         uuid.New().String(),
		Name:       req.Name,
		FolderPath: req.FolderPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Tags:       []string{},
	}

	if req.AIPrompt != "" {
		// Generate workflow from AI prompt
		workflow.FlowDef = s.generateWorkflowFromPrompt(req.AIPrompt)
	} else if req.FlowDefinition != nil {
		workflow.FlowDef = req.FlowDefinition
	} else {
		workflow.FlowDef = map[string]interface{}{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		}
	}

	s.workflows[workflow.ID] = workflow

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflow_id": workflow.ID,
		"status":      "created",
		"nodes":       workflow.FlowDef["nodes"],
		"edges":       workflow.FlowDef["edges"],
	})
}

func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows := make([]*Workflow, 0, len(s.workflows))
	for _, wf := range s.workflows {
		workflows = append(workflows, wf)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": workflows,
	})
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workflow, exists := s.workflows[id]
	if !exists {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflow)
}

func (s *Server) handleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, exists := s.workflows[id]
	if !exists {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	var req ExecuteWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	execution := &Execution{
		ID:          uuid.New().String(),
		WorkflowID:  id,
		Status:      "started",
		StartedAt:   time.Now(),
		Progress:    0,
		CurrentStep: "Initializing",
	}

	s.executions[execution.ID] = execution

	// Start async execution
	go s.executeWorkflow(execution)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"execution_id": execution.ID,
		"status":       execution.Status,
	})
}

func (s *Server) handleGetExecutionScreenshots(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	// Mock screenshots for demo
	screenshots := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-30 * time.Second),
			"url":       fmt.Sprintf("/screenshots/%s/1.png", id),
			"step_name": "Navigate to page",
		},
		{
			"timestamp": time.Now().Add(-20 * time.Second),
			"url":       fmt.Sprintf("/screenshots/%s/2.png", id),
			"step_name": "Click login button",
		},
		{
			"timestamp": time.Now().Add(-10 * time.Second),
			"url":       fmt.Sprintf("/screenshots/%s/3.png", id),
			"step_name": "Fill form",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"screenshots": screenshots,
	})
}

func (s *Server) generateWorkflowFromPrompt(prompt string) map[string]interface{} {
	// Mock AI generation - in real implementation, call Ollama or Claude API
	return map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{
				"id":       "node-1",
				"type":     "navigate",
				"position": map[string]int{"x": 100, "y": 100},
				"data":     map[string]interface{}{"url": "https://example.com"},
			},
			map[string]interface{}{
				"id":       "node-2",
				"type":     "screenshot",
				"position": map[string]int{"x": 100, "y": 200},
				"data":     map[string]interface{}{"name": "homepage"},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{
				"id":     "edge-1",
				"source": "node-1",
				"target": "node-2",
			},
		},
	}
}

func (s *Server) executeWorkflow(execution *Execution) {
	// Simulate workflow execution
	steps := []string{
		"Starting browser",
		"Navigating to URL",
		"Waiting for page load",
		"Taking screenshot",
		"Extracting data",
		"Saving results",
	}

	for i, step := range steps {
		time.Sleep(2 * time.Second)
		execution.CurrentStep = step
		execution.Progress = (i + 1) * 100 / len(steps)
		
		if i == len(steps)-1 {
			execution.Status = "completed"
			now := time.Now()
			execution.CompletedAt = &now
		}
	}
}

func main() {
	port := os.Getenv("BROWSER_AUTOMATION_API_PORT")
	if port == "" {
		port = "8090"
	}

	server := NewServer()
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3090", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:          300,
	}))

	// Routes
	r.Get("/health", server.handleHealth)
	
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/workflows/create", server.handleCreateWorkflow)
		r.Get("/workflows", server.handleListWorkflows)
		r.Get("/workflows/{id}", server.handleGetWorkflow)
		r.Post("/workflows/{id}/execute", server.handleExecuteWorkflow)
		r.Get("/executions/{id}/screenshots", server.handleGetExecutionScreenshots)
	})

	log.Printf("Browser Automation Studio API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}