package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

type BenchmarkRequest struct {
	AlgorithmID string `json:"algorithm_id"`
	Language    string `json:"language"`
	Code        string `json:"code"`
	InputSizes  []int  `json:"input_sizes"`
}

type BenchmarkResult struct {
	AlgorithmID   string               `json:"algorithm_id"`
	Language      string               `json:"language"`
	Results       []BenchmarkDataPoint `json:"results"`
	AverageTime   float64              `json:"average_time_ms"`
	ComplexityFit string               `json:"complexity_fit"`
}

type BenchmarkDataPoint struct {
	InputSize     int     `json:"input_size"`
	ExecutionTime float64 `json:"execution_time_ms"`
	MemoryUsed    int     `json:"memory_used_bytes"`
}

func benchmarkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For now, only support Python benchmarking locally
	if req.Language != "python" {
		http.Error(w, "Currently only Python is supported for benchmarking", http.StatusBadRequest)
		return
	}

	// Use default input sizes if none provided
	if len(req.InputSizes) == 0 {
		req.InputSizes = []int{10, 50, 100, 500, 1000}
	}

	results := []BenchmarkDataPoint{}
	totalTime := 0.0

	// Run benchmarks for different input sizes
	for _, size := range req.InputSizes {
		// Generate test input based on size
		testInput := generateTestInput(req.AlgorithmID, size)

		// Create a Python script that includes timing
		benchmarkCode := fmt.Sprintf(`
import time
import sys
import json

%s

# Parse input
input_data = %s

# Time the execution
start_time = time.perf_counter()
result = %s(input_data)
end_time = time.perf_counter()

# Output timing
execution_time_ms = (end_time - start_time) * 1000
print(json.dumps({"time_ms": execution_time_ms}))
`, req.Code, testInput, getFunctionName(req.AlgorithmID))

		// Execute the benchmark
		cmd := exec.Command("python3", "-c", benchmarkCode)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		// Parse the output
		var result struct {
			TimeMs float64 `json:"time_ms"`
		}
		if err := json.Unmarshal(output, &result); err == nil {
			dataPoint := BenchmarkDataPoint{
				InputSize:     size,
				ExecutionTime: result.TimeMs,
				MemoryUsed:    0, // Memory tracking would require more sophisticated tooling
			}
			results = append(results, dataPoint)
			totalTime += result.TimeMs
		}
	}

	// Calculate average time
	avgTime := 0.0
	if len(results) > 0 {
		avgTime = totalTime / float64(len(results))
	}

	// Analyze complexity (simplified)
	complexityFit := analyzeComplexity(results)

	response := BenchmarkResult{
		AlgorithmID:   req.AlgorithmID,
		Language:      req.Language,
		Results:       results,
		AverageTime:   avgTime,
		ComplexityFit: complexityFit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateTestInput(algorithmID string, size int) string {
	// Generate appropriate test input based on algorithm type
	if strings.Contains(algorithmID, "sort") {
		// Generate random array for sorting algorithms
		return fmt.Sprintf("[%s]", strings.Trim(strings.Repeat(fmt.Sprintf("%d,", size), size), ","))
	}
	// Default: return array of integers
	arr := make([]string, size)
	for i := 0; i < size; i++ {
		arr[i] = fmt.Sprintf("%d", i)
	}
	return fmt.Sprintf("[%s]", strings.Join(arr, ","))
}

func getFunctionName(algorithmID string) string {
	// Map algorithm IDs to function names
	funcMap := map[string]string{
		"quicksort":     "quicksort",
		"mergesort":     "mergesort",
		"bubblesort":    "bubblesort",
		"binary_search": "binary_search",
		"linear_search": "linear_search",
	}
	if name, ok := funcMap[algorithmID]; ok {
		return name
	}
	return algorithmID
}

func analyzeComplexity(results []BenchmarkDataPoint) string {
	if len(results) < 2 {
		return "insufficient data"
	}

	// Simple complexity analysis based on growth rate
	// This is a simplified version - real analysis would be more sophisticated
	firstTime := results[0].ExecutionTime
	lastTime := results[len(results)-1].ExecutionTime
	firstSize := float64(results[0].InputSize)
	lastSize := float64(results[len(results)-1].InputSize)

	ratio := (lastTime / firstTime) / (lastSize / firstSize)

	if ratio < 1.5 {
		return "O(n) - Linear"
	} else if ratio < 3 {
		return "O(n log n) - Linearithmic"
	} else if ratio < lastSize/firstSize {
		return "O(n²) - Quadratic"
	}
	return "O(n²+) - Super-quadratic"
}
