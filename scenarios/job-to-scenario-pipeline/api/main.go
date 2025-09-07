package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

// Job represents a job opportunity
type Job struct {
	ID               string            `yaml:"id" json:"id"`
	Source           string            `yaml:"source" json:"source"`
	Title            string            `yaml:"title" json:"title"`
	Description      string            `yaml:"description" json:"description"`
	Budget           Budget            `yaml:"budget" json:"budget"`
	SkillsRequired   []string          `yaml:"skills_required" json:"skills_required"`
	Timeline         string            `yaml:"timeline" json:"timeline"`
	State            string            `yaml:"state" json:"state"`
	Metadata         Metadata          `yaml:"metadata" json:"metadata"`
	ResearchReport   *ResearchReport   `yaml:"research_report,omitempty" json:"research_report,omitempty"`
	Proposal         *Proposal         `yaml:"proposal,omitempty" json:"proposal,omitempty"`
	History          []HistoryEntry    `yaml:"history" json:"history"`
}

type Budget struct {
	Min      float64 `yaml:"min" json:"min"`
	Max      float64 `yaml:"max" json:"max"`
	Currency string  `yaml:"currency" json:"currency"`
}

type Metadata struct {
	ImportedAt      string `yaml:"imported_at" json:"imported_at"`
	SourceURL       string `yaml:"source_url" json:"source_url"`
	ClientLocation  string `yaml:"client_location" json:"client_location"`
	JobType         string `yaml:"job_type" json:"job_type"`
	ExperienceLevel string `yaml:"experience_level" json:"experience_level"`
}

type ResearchReport struct {
	ID                 string   `yaml:"id" json:"id"`
	Evaluation         string   `yaml:"evaluation" json:"evaluation"`
	FeasibilityScore   float64  `yaml:"feasibility_score" json:"feasibility_score"`
	ExistingScenarios  []string `yaml:"existing_scenarios" json:"existing_scenarios"`
	RequiredScenarios  []string `yaml:"required_scenarios" json:"required_scenarios"`
	EstimatedHours     int      `yaml:"estimated_hours" json:"estimated_hours"`
	TechnicalAnalysis  string   `yaml:"technical_analysis" json:"technical_analysis"`
	CreatedAt          string   `yaml:"created_at" json:"created_at"`
}

type Proposal struct {
	ID                 string       `yaml:"id" json:"id"`
	CoverLetter        string       `yaml:"cover_letter" json:"cover_letter"`
	TechnicalApproach  string       `yaml:"technical_approach" json:"technical_approach"`
	Timeline           []string     `yaml:"timeline" json:"timeline"`
	Deliverables       []string     `yaml:"deliverables" json:"deliverables"`
	Price              float64      `yaml:"price" json:"price"`
	GeneratedScenarios []string     `yaml:"generated_scenarios" json:"generated_scenarios"`
	CreatedAt          string       `yaml:"created_at" json:"created_at"`
}

type HistoryEntry struct {
	Timestamp string `yaml:"timestamp" json:"timestamp"`
	State     string `yaml:"state" json:"state"`
	Action    string `yaml:"action" json:"action"`
	Notes     string `yaml:"notes" json:"notes"`
}

type ImportRequest struct {
	Source string `json:"source"`
	Data   string `json:"data"`
}

type ResearchRequest struct {
	JobID string `json:"job_id"`
}

var dataDir = "../data"

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "15500"
	}

	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Job endpoints
	r.HandleFunc("/api/v1/jobs", listJobsHandler).Methods("GET")
	r.HandleFunc("/api/v1/jobs/import", importJobHandler).Methods("POST")
	r.HandleFunc("/api/v1/jobs/{id}", getJobHandler).Methods("GET")
	r.HandleFunc("/api/v1/jobs/{id}/research", researchJobHandler).Methods("POST")
	r.HandleFunc("/api/v1/jobs/{id}/approve", approveJobHandler).Methods("POST")
	r.HandleFunc("/api/v1/jobs/{id}/proposal", generateProposalHandler).Methods("POST")

	// Enable CORS
	r.Use(corsMiddleware)

	log.Printf("Job Pipeline API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "job-to-scenario-pipeline",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func listJobsHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "*"
	}

	var jobs []Job
	states := []string{"pending", "researching", "evaluated", "approved", "building", "completed"}
	
	for _, s := range states {
		if state != "*" && s != state {
			continue
		}

		files, err := filepath.Glob(filepath.Join(dataDir, s, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}

			var job Job
			if err := yaml.Unmarshal(data, &job); err != nil {
				continue
			}

			jobs = append(jobs, job)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs": jobs,
		"total": len(jobs),
	})
}

func importJobHandler(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobID := fmt.Sprintf("JOB-%s-%03d", 
		time.Now().Format("20060102-150405"),
		time.Now().Nanosecond()%1000)

	var job Job
	
	switch req.Source {
	case "manual":
		job = Job{
			ID:          jobID,
			Source:      "manual",
			Title:       extractTitle(req.Data),
			Description: req.Data,
			State:       "pending",
			Budget:      Budget{Currency: "USD"},
			Metadata: Metadata{
				ImportedAt: time.Now().UTC().Format(time.RFC3339),
			},
			History: []HistoryEntry{{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				State:     "pending",
				Action:    "imported",
				Notes:     "Manual import",
			}},
		}

	case "screenshot":
		// Process screenshot with OCR using browserless
		text, err := processScreenshot(req.Data)
		if err != nil {
			http.Error(w, fmt.Sprintf("OCR failed: %v", err), http.StatusInternalServerError)
			return
		}

		job = Job{
			ID:          jobID,
			Source:      "screenshot",
			Title:       extractTitle(text),
			Description: text,
			State:       "pending",
			Budget:      Budget{Currency: "USD"},
			Metadata: Metadata{
				ImportedAt: time.Now().UTC().Format(time.RFC3339),
			},
			History: []HistoryEntry{{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				State:     "pending",
				Action:    "imported",
				Notes:     "Screenshot import with OCR",
			}},
		}

	case "upwork":
		// Parse structured data from Huginn
		var upworkData map[string]interface{}
		if err := json.Unmarshal([]byte(req.Data), &upworkData); err != nil {
			// If not JSON, treat as plain text
			job = Job{
				ID:          jobID,
				Source:      "upwork",
				Title:       extractTitle(req.Data),
				Description: req.Data,
				State:       "pending",
				Budget:      Budget{Currency: "USD"},
				Metadata: Metadata{
					ImportedAt: time.Now().UTC().Format(time.RFC3339),
				},
			}
		} else {
			// Parse structured Upwork data
			job = parseUpworkData(jobID, upworkData)
		}

		job.History = []HistoryEntry{{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			State:     "pending",
			Action:    "imported",
			Notes:     "Upwork RSS import",
		}}

	default:
		http.Error(w, "Invalid source", http.StatusBadRequest)
		return
	}

	// Save job to file
	if err := saveJob(&job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"state":  "pending",
	})
}

func getJobHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := loadJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func researchJobHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := loadJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Move to researching state
	if err := moveJob(jobID, job.State, "researching"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Perform research asynchronously
	go performResearch(jobID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "researching",
		"job_id": jobID,
	})
}

func approveJobHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := loadJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if job.State != "evaluated" {
		http.Error(w, "Job must be evaluated before approval", http.StatusBadRequest)
		return
	}

	// Move to approved state
	if err := moveJob(jobID, job.State, "approved"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Start building scenarios asynchronously
	go buildScenarios(jobID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"state": "building",
		"estimated_completion": time.Now().Add(30 * time.Minute).Format(time.RFC3339),
	})
}

func generateProposalHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := loadJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if job.State != "building" && job.State != "completed" {
		http.Error(w, "Job must be built before generating proposal", http.StatusBadRequest)
		return
	}

	// Generate proposal using Ollama
	proposal, err := generateProposal(job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	job.Proposal = proposal
	if err := saveJob(job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(proposal)
}

// Helper functions

func extractTitle(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		title := strings.TrimSpace(lines[0])
		if len(title) > 100 {
			return title[:100] + "..."
		}
		return title
	}
	return "Untitled Job"
}

func processScreenshot(base64Data string) (string, error) {
	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	// Save to temp file
	tempFile := fmt.Sprintf("/tmp/screenshot-%s.png", uuid.New().String())
	if err := ioutil.WriteFile(tempFile, imageData, 0644); err != nil {
		return "", err
	}
	defer os.Remove(tempFile)

	// Use browserless OCR
	cmd := exec.Command("resource-browserless", "ocr", tempFile)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("OCR failed: %v", err)
	}

	return string(output), nil
}

func parseUpworkData(jobID string, data map[string]interface{}) Job {
	job := Job{
		ID:     jobID,
		Source: "upwork",
		State:  "pending",
		Budget: Budget{Currency: "USD"},
		Metadata: Metadata{
			ImportedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	if title, ok := data["title"].(string); ok {
		job.Title = title
	}
	if desc, ok := data["description"].(string); ok {
		job.Description = desc
	}
	if budget, ok := data["budget"].(map[string]interface{}); ok {
		if min, ok := budget["min"].(float64); ok {
			job.Budget.Min = min
		}
		if max, ok := budget["max"].(float64); ok {
			job.Budget.Max = max
		}
	}
	if skills, ok := data["skills_required"].([]interface{}); ok {
		for _, skill := range skills {
			if s, ok := skill.(string); ok {
				job.SkillsRequired = append(job.SkillsRequired, s)
			}
		}
	}
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		if url, ok := metadata["source_url"].(string); ok {
			job.Metadata.SourceURL = url
		}
		if jobType, ok := metadata["job_type"].(string); ok {
			job.Metadata.JobType = jobType
		}
		if exp, ok := metadata["experience_level"].(string); ok {
			job.Metadata.ExperienceLevel = exp
		}
	}

	return job
}

func saveJob(job *Job) error {
	data, err := yaml.Marshal(job)
	if err != nil {
		return err
	}

	filename := filepath.Join(dataDir, job.State, job.ID+".yaml")
	return ioutil.WriteFile(filename, data, 0644)
}

func loadJob(jobID string) (*Job, error) {
	states := []string{"pending", "researching", "evaluated", "approved", "building", "completed", "archive"}
	
	for _, state := range states {
		filename := filepath.Join(dataDir, state, jobID+".yaml")
		if _, err := os.Stat(filename); err == nil {
			data, err := ioutil.ReadFile(filename)
			if err != nil {
				return nil, err
			}

			var job Job
			if err := yaml.Unmarshal(data, &job); err != nil {
				return nil, err
			}

			return &job, nil
		}
	}

	return nil, fmt.Errorf("job not found")
}

func moveJob(jobID, fromState, toState string) error {
	oldPath := filepath.Join(dataDir, fromState, jobID+".yaml")
	newPath := filepath.Join(dataDir, toState, jobID+".yaml")
	
	// Load job to update state
	job, err := loadJob(jobID)
	if err != nil {
		return err
	}

	// Update state and history
	job.State = toState
	job.History = append(job.History, HistoryEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		State:     toState,
		Action:    fmt.Sprintf("moved from %s", fromState),
		Notes:     "State transition",
	})

	// Save with new state
	if err := saveJob(job); err != nil {
		return err
	}

	// Remove old file
	os.Remove(oldPath)
	
	return nil
}

func performResearch(jobID string) {
	job, err := loadJob(jobID)
	if err != nil {
		log.Printf("Failed to load job %s: %v", jobID, err)
		return
	}

	// Use Ollama for research analysis
	prompt := fmt.Sprintf(`Analyze this job opportunity and determine:
1. Can we build this with our existing capabilities? (scenarios: scenario-generator-v1, research-assistant, knowledge-observatory, etc.)
2. What would need to be built?
3. Feasibility score (0-1)
4. Evaluation: RECOMMENDED, NOT_RECOMMENDED, ALREADY_DONE, or NO_ACTION

Job Title: %s
Description: %s
Budget: $%v-%v
Skills: %v

Respond in JSON format:
{
  "evaluation": "RECOMMENDED|NOT_RECOMMENDED|ALREADY_DONE|NO_ACTION",
  "feasibility_score": 0.85,
  "existing_scenarios": ["scenario1", "scenario2"],
  "required_scenarios": ["new_scenario1"],
  "estimated_hours": 40,
  "technical_analysis": "detailed analysis..."
}`, job.Title, job.Description, job.Budget.Min, job.Budget.Max, strings.Join(job.SkillsRequired, ", "))

	// Call Ollama directly
	cmd := exec.Command("resource-ollama", "query", "--json", prompt)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Ollama query failed: %v", err)
		moveJob(jobID, "researching", "evaluated")
		return
	}

	// Parse response
	var report ResearchReport
	if err := json.Unmarshal(output, &report); err != nil {
		log.Printf("Failed to parse research response: %v", err)
		// Create default report
		report = ResearchReport{
			ID:               uuid.New().String(),
			Evaluation:       "NOT_RECOMMENDED",
			FeasibilityScore: 0.3,
			TechnicalAnalysis: "Unable to analyze",
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		}
	}

	report.ID = uuid.New().String()
	report.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// Update job with research report
	job.ResearchReport = &report
	saveJob(job)

	// Move to evaluated state
	moveJob(jobID, "researching", "evaluated")
}

func buildScenarios(jobID string) {
	job, err := loadJob(jobID)
	if err != nil {
		log.Printf("Failed to load job %s: %v", jobID, err)
		return
	}

	// Move to building state
	moveJob(jobID, "approved", "building")

	// Check if we need to build new scenarios
	if job.ResearchReport != nil && len(job.ResearchReport.RequiredScenarios) > 0 {
		for _, scenario := range job.ResearchReport.RequiredScenarios {
			// Call scenario-generator-v1
			cmd := exec.Command("vrooli", "scenario", "run", "scenario-generator-v1", 
				"--input", fmt.Sprintf("Create a scenario called %s for: %s", scenario, job.Description))
			
			if output, err := cmd.Output(); err != nil {
				log.Printf("Failed to generate scenario %s: %v", scenario, err)
			} else {
				log.Printf("Generated scenario %s: %s", scenario, string(output))
			}
		}
	}

	// Move to completed state
	moveJob(jobID, "building", "completed")
}

func generateProposal(job *Job) (*Proposal, error) {
	prompt := fmt.Sprintf(`Generate a professional Upwork proposal for this job:

Title: %s
Description: %s
Budget: $%v-%v
Our Solution: Using scenarios %v

Create a compelling proposal that:
1. Shows understanding of their needs
2. Explains our automated solution approach
3. Provides timeline and deliverables
4. Suggests a fair price

Format as JSON:
{
  "cover_letter": "...",
  "technical_approach": "...",
  "timeline": ["Week 1: ...", "Week 2: ..."],
  "deliverables": ["...", "..."],
  "price": 0000
}`, job.Title, job.Description, job.Budget.Min, job.Budget.Max, 
		append(job.ResearchReport.ExistingScenarios, job.ResearchReport.RequiredScenarios...))

	cmd := exec.Command("resource-ollama", "query", "--json", prompt)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("proposal generation failed: %v", err)
	}

	var proposal Proposal
	if err := json.Unmarshal(output, &proposal); err != nil {
		return nil, fmt.Errorf("failed to parse proposal: %v", err)
	}

	proposal.ID = uuid.New().String()
	proposal.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	proposal.GeneratedScenarios = job.ResearchReport.RequiredScenarios

	return &proposal, nil
}