package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Contribution represents a user-submitted algorithm or implementation
type Contribution struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"` // "algorithm" or "implementation"
	AlgorithmID      string    `json:"algorithm_id,omitempty"`
	ContributorName  string    `json:"contributor_name"`
	ContributorEmail string    `json:"contributor_email,omitempty"`
	Status           string    `json:"status"` // "pending", "approved", "rejected"
	SubmittedAt      time.Time `json:"submitted_at"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes      string    `json:"review_notes,omitempty"`
	Content          json.RawMessage `json:"content"`
}

// AlgorithmContribution represents a new algorithm submission
type AlgorithmContribution struct {
	Name            string   `json:"name"`
	DisplayName     string   `json:"display_name"`
	Category        string   `json:"category"`
	Description     string   `json:"description"`
	ComplexityTime  string   `json:"complexity_time"`
	ComplexitySpace string   `json:"complexity_space"`
	Difficulty      string   `json:"difficulty"`
	Tags            []string `json:"tags"`
	Implementation  struct {
		Language string `json:"language"`
		Code     string `json:"code"`
	} `json:"implementation"`
	TestCases []struct {
		Name           string      `json:"name"`
		Input          interface{} `json:"input"`
		ExpectedOutput interface{} `json:"expected_output"`
		Description    string      `json:"description,omitempty"`
	} `json:"test_cases"`
}

// ImplementationContribution represents a new implementation for existing algorithm
type ImplementationContribution struct {
	AlgorithmID string `json:"algorithm_id"`
	Language    string `json:"language"`
	Code        string `json:"code"`
	Notes       string `json:"notes,omitempty"`
}

// submitAlgorithmHandler handles new algorithm submissions
func submitAlgorithmHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContributorName  string                `json:"contributor_name"`
		ContributorEmail string                `json:"contributor_email,omitempty"`
		Algorithm        AlgorithmContribution `json:"algorithm"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.ContributorName == "" || req.Algorithm.Name == "" || req.Algorithm.Category == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	// Create contribution record
	contributionID := uuid.New().String()
	contentJSON, _ := json.Marshal(req.Algorithm)
	
	_, err := db.Exec(`
		INSERT INTO contributions (
			id, type, contributor_name, contributor_email, 
			status, submitted_at, content
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, contributionID, "algorithm", req.ContributorName, req.ContributorEmail,
		"pending", time.Now(), contentJSON)
	
	if err != nil {
		// If table doesn't exist, create it
		createContributionsTable()
		
		// Try again
		_, err = db.Exec(`
			INSERT INTO contributions (
				id, type, contributor_name, contributor_email, 
				status, submitted_at, content
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, contributionID, "algorithm", req.ContributorName, req.ContributorEmail,
			"pending", time.Now(), contentJSON)
		
		if err != nil {
			log.Printf("Failed to save contribution: %v", err)
			http.Error(w, "Failed to save contribution", http.StatusInternalServerError)
			return
		}
	}
	
	// Validate the algorithm implementation
	validationResult := validateContribution(req.Algorithm)
	
	response := map[string]interface{}{
		"contribution_id": contributionID,
		"status":         "pending",
		"validation":     validationResult,
		"message":        "Thank you for your contribution! It will be reviewed soon.",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// submitImplementationHandler handles new implementation submissions
func submitImplementationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContributorName   string                      `json:"contributor_name"`
		ContributorEmail  string                      `json:"contributor_email,omitempty"`
		Implementation    ImplementationContribution `json:"implementation"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.ContributorName == "" || req.Implementation.AlgorithmID == "" || 
	   req.Implementation.Language == "" || req.Implementation.Code == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	// Verify algorithm exists
	var algoExists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM algorithms WHERE id = $1)", 
		req.Implementation.AlgorithmID).Scan(&algoExists)
	
	if err != nil || !algoExists {
		http.Error(w, "Algorithm not found", http.StatusNotFound)
		return
	}
	
	// Create contribution record
	contributionID := uuid.New().String()
	contentJSON, _ := json.Marshal(req.Implementation)
	
	_, err = db.Exec(`
		INSERT INTO contributions (
			id, type, algorithm_id, contributor_name, contributor_email, 
			status, submitted_at, content
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, contributionID, "implementation", req.Implementation.AlgorithmID,
		req.ContributorName, req.ContributorEmail, "pending", time.Now(), contentJSON)
	
	if err != nil {
		// If table doesn't exist, create it
		createContributionsTable()
		
		// Try again
		_, err = db.Exec(`
			INSERT INTO contributions (
				id, type, algorithm_id, contributor_name, contributor_email, 
				status, submitted_at, content
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, contributionID, "implementation", req.Implementation.AlgorithmID,
			req.ContributorName, req.ContributorEmail, "pending", time.Now(), contentJSON)
		
		if err != nil {
			log.Printf("Failed to save contribution: %v", err)
			http.Error(w, "Failed to save contribution", http.StatusInternalServerError)
			return
		}
	}
	
	// Test the implementation against existing test cases
	testResult := testImplementation(req.Implementation.AlgorithmID, 
		req.Implementation.Language, req.Implementation.Code)
	
	response := map[string]interface{}{
		"contribution_id": contributionID,
		"status":         "pending",
		"test_results":   testResult,
		"message":        "Thank you for your contribution! It will be reviewed soon.",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listContributionsHandler lists pending contributions
func listContributionsHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	
	rows, err := db.Query(`
		SELECT id, type, algorithm_id, contributor_name, status, 
		       submitted_at, reviewed_at, review_notes
		FROM contributions
		WHERE status = $1
		ORDER BY submitted_at DESC
		LIMIT 100
	`, status)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Table might not exist yet
			createContributionsTable()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"contributions": []Contribution{},
				"total": 0,
			})
			return
		}
		log.Printf("Failed to list contributions: %v", err)
		http.Error(w, "Failed to list contributions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var contributions []Contribution
	for rows.Next() {
		var c Contribution
		var algorithmID sql.NullString
		var reviewedAt sql.NullTime
		var reviewNotes sql.NullString
		
		err := rows.Scan(&c.ID, &c.Type, &algorithmID, &c.ContributorName,
			&c.Status, &c.SubmittedAt, &reviewedAt, &reviewNotes)
		
		if err != nil {
			log.Printf("Failed to scan contribution: %v", err)
			continue
		}
		
		if algorithmID.Valid {
			c.AlgorithmID = algorithmID.String
		}
		if reviewedAt.Valid {
			c.ReviewedAt = &reviewedAt.Time
		}
		if reviewNotes.Valid {
			c.ReviewNotes = reviewNotes.String
		}
		
		contributions = append(contributions, c)
	}
	
	response := map[string]interface{}{
		"contributions": contributions,
		"total":        len(contributions),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// reviewContributionHandler handles contribution review (approve/reject)
func reviewContributionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contributionID := vars["id"]
	
	var req struct {
		Action string `json:"action"` // "approve" or "reject"
		Notes  string `json:"notes,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.Action != "approve" && req.Action != "reject" {
		http.Error(w, "Invalid action. Must be 'approve' or 'reject'", http.StatusBadRequest)
		return
	}
	
	status := "approved"
	if req.Action == "reject" {
		status = "rejected"
	}
	
	// Update contribution status
	result, err := db.Exec(`
		UPDATE contributions 
		SET status = $1, reviewed_at = $2, review_notes = $3
		WHERE id = $4
	`, status, time.Now(), req.Notes, contributionID)
	
	if err != nil {
		log.Printf("Failed to update contribution: %v", err)
		http.Error(w, "Failed to update contribution", http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Contribution not found", http.StatusNotFound)
		return
	}
	
	// If approved, add to main database
	if req.Action == "approve" {
		applyContribution(contributionID)
	}
	
	response := map[string]interface{}{
		"contribution_id": contributionID,
		"status":         status,
		"message":        fmt.Sprintf("Contribution has been %s", status),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func createContributionsTable() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS contributions (
			id VARCHAR(36) PRIMARY KEY,
			type VARCHAR(20) NOT NULL,
			algorithm_id VARCHAR(36),
			contributor_name VARCHAR(100) NOT NULL,
			contributor_email VARCHAR(100),
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			reviewed_at TIMESTAMP,
			review_notes TEXT,
			content JSONB NOT NULL,
			CONSTRAINT fk_algorithm 
				FOREIGN KEY(algorithm_id) 
				REFERENCES algorithms(id) 
				ON DELETE CASCADE
		)
	`)
	
	if err != nil {
		log.Printf("Failed to create contributions table: %v", err)
	}
}

func validateContribution(algo AlgorithmContribution) map[string]interface{} {
	result := map[string]interface{}{
		"valid": true,
		"errors": []string{},
	}
	
	// Basic validation
	if algo.Name == "" {
		result["valid"] = false
		result["errors"] = append(result["errors"].([]string), "Algorithm name is required")
	}
	
	if algo.Category == "" {
		result["valid"] = false
		result["errors"] = append(result["errors"].([]string), "Category is required")
	}
	
	if algo.Implementation.Code == "" {
		result["valid"] = false
		result["errors"] = append(result["errors"].([]string), "At least one implementation is required")
	}
	
	if len(algo.TestCases) == 0 {
		result["valid"] = false
		result["errors"] = append(result["errors"].([]string), "At least one test case is required")
	}
	
	return result
}

func testImplementation(algorithmID, language, code string) map[string]interface{} {
	// Get test cases for the algorithm
	rows, err := db.Query(`
		SELECT input, expected_output 
		FROM test_cases 
		WHERE algorithm_id = $1
		LIMIT 5
	`, algorithmID)
	
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Failed to get test cases",
		}
	}
	defer rows.Close()
	
	passed := 0
	failed := 0
	
	for rows.Next() {
		var inputJSON, expectedJSON string
		if err := rows.Scan(&inputJSON, &expectedJSON); err != nil {
			continue
		}
		
		// Execute the code with the test input
		executor := NewLocalExecutor(5 * time.Second)
		var result *LocalExecutionResult
		
		switch language {
		case "python":
			result, err = executor.ExecutePython(code, inputJSON)
		case "javascript":
			result, err = executor.ExecuteJavaScript(code, inputJSON)
		case "go":
			result, err = executor.ExecuteGo(code, inputJSON)
		case "java":
			result, err = executor.ExecuteJava(code, inputJSON)
		case "cpp":
			result, err = executor.ExecuteCPP(code, inputJSON)
		default:
			continue
		}
		
		if err == nil && result.Success {
			// Compare output with expected
			if normalizeOutput(result.Output) == normalizeOutput(expectedJSON) {
				passed++
			} else {
				failed++
			}
		} else {
			failed++
		}
	}
	
	return map[string]interface{}{
		"total_tests": passed + failed,
		"passed":      passed,
		"failed":      failed,
		"success":     failed == 0,
	}
}

func applyContribution(contributionID string) {
	// Get contribution details
	var contrib Contribution
	var content json.RawMessage
	
	err := db.QueryRow(`
		SELECT type, algorithm_id, content 
		FROM contributions 
		WHERE id = $1
	`, contributionID).Scan(&contrib.Type, &contrib.AlgorithmID, &content)
	
	if err != nil {
		log.Printf("Failed to get contribution: %v", err)
		return
	}
	
	if contrib.Type == "algorithm" {
		// Add new algorithm
		var algo AlgorithmContribution
		if err := json.Unmarshal(content, &algo); err != nil {
			log.Printf("Failed to unmarshal algorithm: %v", err)
			return
		}
		
		// Insert algorithm
		algorithmID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO algorithms (
				id, name, display_name, category, description,
				complexity_time, complexity_space, difficulty
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, algorithmID, algo.Name, algo.DisplayName, algo.Category,
			algo.Description, algo.ComplexityTime, algo.ComplexitySpace, algo.Difficulty)
		
		if err != nil {
			log.Printf("Failed to insert algorithm: %v", err)
			return
		}
		
		// Add implementation
		if algo.Implementation.Code != "" {
			implID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO implementations (
					id, algorithm_id, language, code, is_primary, validated
				) VALUES ($1, $2, $3, $4, $5, $6)
			`, implID, algorithmID, algo.Implementation.Language,
				algo.Implementation.Code, true, false)
		}
		
		// Add test cases
		for i, tc := range algo.TestCases {
			tcID := uuid.New().String()
			inputJSON, _ := json.Marshal(tc.Input)
			outputJSON, _ := json.Marshal(tc.ExpectedOutput)
			
			_, err = db.Exec(`
				INSERT INTO test_cases (
					id, algorithm_id, name, input, expected_output, 
					description, sequence_order
				) VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, tcID, algorithmID, tc.Name, inputJSON, outputJSON,
				tc.Description, i+1)
		}
		
	} else if contrib.Type == "implementation" {
		// Add new implementation
		var impl ImplementationContribution
		if err := json.Unmarshal(content, &impl); err != nil {
			log.Printf("Failed to unmarshal implementation: %v", err)
			return
		}
		
		implID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO implementations (
				id, algorithm_id, language, code, is_primary, validated
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, implID, impl.AlgorithmID, impl.Language, impl.Code, false, false)
		
		if err != nil {
			log.Printf("Failed to insert implementation: %v", err)
		}
	}
}

func normalizeOutput(output string) string {
	// Remove trailing whitespace and newlines for comparison
	return strings.TrimSpace(output)
}