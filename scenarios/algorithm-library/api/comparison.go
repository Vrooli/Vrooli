package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ComparisonRequest represents a request to compare multiple implementations
type ComparisonRequest struct {
	AlgorithmID     string   `json:"algorithm_id"`
	Languages       []string `json:"languages"`
	TestCaseID      string   `json:"test_case_id,omitempty"`
	CustomInput     string   `json:"custom_input,omitempty"`
	IncludeBenchmark bool     `json:"include_benchmark"`
}

// ComparisonResult represents the result of comparing implementations
type ComparisonResult struct {
	Algorithm      Algorithm                `json:"algorithm"`
	Implementations []ImplementationComparison `json:"implementations"`
	TestCase       *ComparisonTestCase      `json:"test_case,omitempty"`
	Summary        ComparisonSummary        `json:"summary"`
}

// ImplementationComparison represents a single implementation in comparison
type ImplementationComparison struct {
	Language       string                 `json:"language"`
	Code           string                 `json:"code"`
	ExecutionTime  float64                `json:"execution_time_ms"`
	MemoryUsage    int                    `json:"memory_bytes"`
	Output         string                 `json:"output"`
	Success        bool                   `json:"success"`
	Error          string                 `json:"error,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
}

// ComparisonSummary provides a summary of the comparison
type ComparisonSummary struct {
	FastestLanguage    string  `json:"fastest_language"`
	FastestTime        float64 `json:"fastest_time_ms"`
	SlowestLanguage    string  `json:"slowest_language"`
	SlowestTime        float64 `json:"slowest_time_ms"`
	AverageTime        float64 `json:"average_time_ms"`
	SuccessCount       int     `json:"success_count"`
	FailureCount       int     `json:"failure_count"`
	SpeedupFactor      float64 `json:"speedup_factor"`
}

// ComparisonTestCase represents a test case for algorithm comparison
type ComparisonTestCase struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}

// compareAlgorithmsHandler handles side-by-side comparison of algorithm implementations
func compareAlgorithmsHandler(w http.ResponseWriter, r *http.Request) {
	var req ComparisonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get algorithm details
	var algo Algorithm
	
	// First try to parse as UUID, then fall back to name search
	query := `
		SELECT id, name, display_name, category, description, 
		       complexity_time, complexity_space, difficulty
		FROM algorithms 
		WHERE id = $1::uuid
		LIMIT 1
	`
	
	// Try as UUID first
	err := db.QueryRow(query, req.AlgorithmID).Scan(
		&algo.ID, &algo.Name, &algo.DisplayName, &algo.Category,
		&algo.Description, &algo.ComplexityTime, &algo.ComplexitySpace,
		&algo.Difficulty,
	)
	
	// If UUID fails, try as name
	if err != nil {
		query = `
			SELECT id, name, display_name, category, description, 
			       complexity_time, complexity_space, difficulty
			FROM algorithms 
			WHERE LOWER(name) = LOWER($1)
			LIMIT 1
		`
		err = db.QueryRow(query, req.AlgorithmID).Scan(
			&algo.ID, &algo.Name, &algo.DisplayName, &algo.Category,
			&algo.Description, &algo.ComplexityTime, &algo.ComplexitySpace,
			&algo.Difficulty,
		)
	}
	
	if err != nil {
		log.Printf("Algorithm not found: %v", err)
		http.Error(w, "Algorithm not found", http.StatusNotFound)
		return
	}

	// Get test case or use custom input
	var testCase *ComparisonTestCase
	var testInput string
	
	if req.TestCaseID != "" {
		testCase = &ComparisonTestCase{}
		err := db.QueryRow(`
			SELECT id, name, input, expected_output 
			FROM test_cases 
			WHERE id = $1 AND algorithm_id = $2
			LIMIT 1
		`, req.TestCaseID, algo.ID).Scan(
			&testCase.ID, &testCase.Name, 
			&testCase.Input, &testCase.ExpectedOutput,
		)
		if err != nil {
			log.Printf("Test case not found: %v", err)
			http.Error(w, "Test case not found", http.StatusNotFound)
			return
		}
		testInput = testCase.Input
	} else if req.CustomInput != "" {
		testInput = req.CustomInput
	} else {
		// Get the first test case as default
		testCase = &ComparisonTestCase{}
		err := db.QueryRow(`
			SELECT id, name, input, expected_output 
			FROM test_cases 
			WHERE algorithm_id = $1
			ORDER BY sequence_order
			LIMIT 1
		`, algo.ID).Scan(
			&testCase.ID, &testCase.Name, 
			&testCase.Input, &testCase.ExpectedOutput,
		)
		if err != nil {
			log.Printf("No test cases found: %v", err)
			testInput = "{}"
		} else {
			testInput = testCase.Input
		}
	}

	// Get implementations for requested languages
	implementations := []ImplementationComparison{}
	
	for _, lang := range req.Languages {
		// Get the implementation code
		var code string
		err := db.QueryRow(`
			SELECT code FROM implementations 
			WHERE algorithm_id = $1 AND language = $2
			LIMIT 1
		`, algo.ID, lang).Scan(&code)
		
		if err != nil {
			// No implementation for this language
			implementations = append(implementations, ImplementationComparison{
				Language: lang,
				Code:     "",
				Success:  false,
				Error:    "No implementation available for this language",
			})
			continue
		}

		// Execute the implementation
		startTime := time.Now()
		execResult, execErr := ExecuteWithFallback(lang, code, testInput, 5*time.Second)
		execTime := time.Since(startTime).Seconds() * 1000 // Convert to ms
		
		impl := ImplementationComparison{
			Language:      lang,
			Code:          code,
			ExecutionTime: execTime,
		}
		
		if execErr != nil {
			impl.Success = false
			impl.Error = execErr.Error()
		} else {
			impl.Success = execResult.Success
			impl.Output = strings.TrimSpace(execResult.Output)
			impl.Error = execResult.Error
			
			// Add metrics if benchmarking is requested
			if req.IncludeBenchmark {
				impl.Metrics = map[string]interface{}{
					"raw_execution_time": execResult.ExecutionTime,
					"total_time_ms":      execTime,
				}
			}
		}
		
		implementations = append(implementations, impl)
	}

	// Calculate summary statistics
	summary := calculateComparisonSummary(implementations)
	
	// Prepare response
	response := ComparisonResult{
		Algorithm:       algo,
		Implementations: implementations,
		TestCase:        testCase,
		Summary:         summary,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// calculateComparisonSummary calculates summary statistics from comparison
func calculateComparisonSummary(implementations []ImplementationComparison) ComparisonSummary {
	summary := ComparisonSummary{}
	
	if len(implementations) == 0 {
		return summary
	}
	
	var totalTime float64
	var minTime float64 = 999999
	var maxTime float64 = 0
	
	for _, impl := range implementations {
		if impl.Success {
			summary.SuccessCount++
			
			if impl.ExecutionTime > 0 {
				totalTime += impl.ExecutionTime
				
				if impl.ExecutionTime < minTime {
					minTime = impl.ExecutionTime
					summary.FastestLanguage = impl.Language
					summary.FastestTime = impl.ExecutionTime
				}
				
				if impl.ExecutionTime > maxTime {
					maxTime = impl.ExecutionTime
					summary.SlowestLanguage = impl.Language
					summary.SlowestTime = impl.ExecutionTime
				}
			}
		} else {
			summary.FailureCount++
		}
	}
	
	if summary.SuccessCount > 0 {
		summary.AverageTime = totalTime / float64(summary.SuccessCount)
		
		if minTime > 0 {
			summary.SpeedupFactor = maxTime / minTime
		}
	}
	
	return summary
}

// compareVisualizationHandler provides a visualization of algorithm comparison
func compareVisualizationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	algorithmID := vars["id"]
	
	// Get algorithm details
	var algo Algorithm
	err := db.QueryRow(`
		SELECT id, name, display_name, category, description, 
		       complexity_time, complexity_space, difficulty
		FROM algorithms 
		WHERE id = $1 OR LOWER(name) = LOWER($1)
		LIMIT 1
	`, algorithmID).Scan(
		&algo.ID, &algo.Name, &algo.DisplayName, &algo.Category,
		&algo.Description, &algo.ComplexityTime, &algo.ComplexitySpace,
		&algo.Difficulty,
	)
	
	if err != nil {
		log.Printf("Algorithm not found: %v", err)
		http.Error(w, "Algorithm not found", http.StatusNotFound)
		return
	}
	
	// Get available languages for this algorithm
	rows, err := db.Query(`
		SELECT DISTINCT language 
		FROM implementations 
		WHERE algorithm_id = $1
		ORDER BY language
	`, algo.ID)
	if err != nil {
		log.Printf("Failed to get languages: %v", err)
		http.Error(w, "Failed to retrieve languages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var languages []string
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err == nil {
			languages = append(languages, lang)
		}
	}
	
	// Return visualization data
	response := map[string]interface{}{
		"algorithm": algo,
		"available_languages": languages,
		"comparison_endpoint": fmt.Sprintf("/api/v1/algorithms/compare"),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}