package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchmarkHandler(t *testing.T) {

	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		validate       func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Method not allowed - GET",
			method:         "GET",
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			validate:       nil,
		},
		{
			name:           "Invalid JSON body",
			method:         "POST",
			body:           "invalid-json",
			expectedStatus: http.StatusBadRequest,
			validate:       nil,
		},
		{
			name:   "Unsupported language",
			method: "POST",
			body: BenchmarkRequest{
				AlgorithmID: "quicksort",
				Language:    "ruby",
				Code:        "def quicksort(arr); arr.sort; end",
			},
			expectedStatus: http.StatusBadRequest,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Python")
			},
		},
		{
			name:   "Valid Python benchmark request",
			method: "POST",
			body: BenchmarkRequest{
				AlgorithmID: "quicksort",
				Language:    "python",
				Code:        "def quicksort(arr):\n    return sorted(arr)",
				InputSizes:  []int{10, 50},
			},
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result BenchmarkResult
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, "quicksort", result.AlgorithmID)
				assert.Equal(t, "python", result.Language)
				assert.NotEmpty(t, result.Results)
				assert.NotEmpty(t, result.ComplexityFit)
			},
		},
		{
			name:   "Benchmark with default input sizes",
			method: "POST",
			body: BenchmarkRequest{
				AlgorithmID: "bubblesort",
				Language:    "python",
				Code:        "def bubblesort(arr):\n    return sorted(arr)",
			},
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result BenchmarkResult
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				// Default sizes: 10, 50, 100, 500, 1000
				assert.GreaterOrEqual(t, len(result.Results), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					body = bytes.NewBufferString(str)
				} else {
					bodyBytes, _ := json.Marshal(tt.body)
					body = bytes.NewBuffer(bodyBytes)
				}
			} else {
				body = bytes.NewBuffer([]byte{})
			}

			req := httptest.NewRequest(tt.method, "/api/v1/algorithms/benchmark", body)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			benchmarkHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validate != nil {
				tt.validate(t, w)
			}
		})
	}
}

func TestGenerateTestInput(t *testing.T) {
	tests := []struct {
		name        string
		algorithmID string
		size        int
		wantType    string
	}{
		{
			name:        "Sorting algorithm",
			algorithmID: "quicksort",
			size:        5,
			wantType:    "array",
		},
		{
			name:        "Another sorting algorithm",
			algorithmID: "mergesort",
			size:        10,
			wantType:    "array",
		},
		{
			name:        "Non-sorting algorithm",
			algorithmID: "binary_search",
			size:        20,
			wantType:    "array",
		},
		{
			name:        "Empty size",
			algorithmID: "quicksort",
			size:        0,
			wantType:    "array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTestInput(tt.algorithmID, tt.size)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "[")
			assert.Contains(t, result, "]")
		})
	}
}

func TestGetFunctionName(t *testing.T) {
	tests := []struct {
		algorithmID string
		expected    string
	}{
		{"quicksort", "quicksort"},
		{"mergesort", "mergesort"},
		{"bubblesort", "bubblesort"},
		{"binary_search", "binary_search"},
		{"linear_search", "linear_search"},
		{"unknown_algorithm", "unknown_algorithm"},
		{"custom_func", "custom_func"},
	}

	for _, tt := range tests {
		t.Run(tt.algorithmID, func(t *testing.T) {
			result := getFunctionName(tt.algorithmID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzeComplexity(t *testing.T) {
	tests := []struct {
		name     string
		results  []BenchmarkDataPoint
		expected string
	}{
		{
			name:     "Insufficient data - empty",
			results:  []BenchmarkDataPoint{},
			expected: "insufficient data",
		},
		{
			name: "Insufficient data - single point",
			results: []BenchmarkDataPoint{
				{InputSize: 10, ExecutionTime: 1.0},
			},
			expected: "insufficient data",
		},
		{
			name: "Linear complexity - O(n)",
			results: []BenchmarkDataPoint{
				{InputSize: 10, ExecutionTime: 1.0},
				{InputSize: 100, ExecutionTime: 10.0},
			},
			expected: "O(n) - Linear",
		},
		{
			name: "Linearithmic complexity - O(n log n)",
			results: []BenchmarkDataPoint{
				{InputSize: 10, ExecutionTime: 1.0},
				{InputSize: 100, ExecutionTime: 20.0},
			},
			expected: "O(n log n) - Linearithmic",
		},
		{
			name: "Quadratic complexity - O(n²)",
			results: []BenchmarkDataPoint{
				{InputSize: 10, ExecutionTime: 1.0},
				{InputSize: 100, ExecutionTime: 100.0},
			},
			expected: "O(n²) - Quadratic",
		},
		{
			name: "Super-quadratic complexity",
			results: []BenchmarkDataPoint{
				{InputSize: 10, ExecutionTime: 1.0},
				{InputSize: 100, ExecutionTime: 10000.0},
			},
			expected: "O(n²+) - Super-quadratic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzeComplexity(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBenchmarkDataPoint(t *testing.T) {
	// Test BenchmarkDataPoint structure
	dp := BenchmarkDataPoint{
		InputSize:     100,
		ExecutionTime: 1.5,
		MemoryUsed:    1024,
	}

	assert.Equal(t, 100, dp.InputSize)
	assert.Equal(t, 1.5, dp.ExecutionTime)
	assert.Equal(t, 1024, dp.MemoryUsed)
}

func TestBenchmarkRequest(t *testing.T) {
	// Test BenchmarkRequest marshaling/unmarshaling
	req := BenchmarkRequest{
		AlgorithmID: "quicksort",
		Language:    "python",
		Code:        "def quicksort(arr): return sorted(arr)",
		InputSizes:  []int{10, 50, 100},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded BenchmarkRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.AlgorithmID, decoded.AlgorithmID)
	assert.Equal(t, req.Language, decoded.Language)
	assert.Equal(t, req.Code, decoded.Code)
	assert.Equal(t, len(req.InputSizes), len(decoded.InputSizes))
}

func TestBenchmarkResult(t *testing.T) {
	// Test BenchmarkResult marshaling/unmarshaling
	result := BenchmarkResult{
		AlgorithmID:   "quicksort",
		Language:      "python",
		Results:       []BenchmarkDataPoint{{InputSize: 10, ExecutionTime: 1.0, MemoryUsed: 1024}},
		AverageTime:   1.0,
		ComplexityFit: "O(n log n)",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded BenchmarkResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.AlgorithmID, decoded.AlgorithmID)
	assert.Equal(t, result.Language, decoded.Language)
	assert.Equal(t, result.ComplexityFit, decoded.ComplexityFit)
	assert.Equal(t, len(result.Results), len(decoded.Results))
}
