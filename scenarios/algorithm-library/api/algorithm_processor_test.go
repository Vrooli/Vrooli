package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAlgorithmProcessor(t *testing.T) {
	processor := NewAlgorithmProcessor("http://judge0.test")
	assert.NotNil(t, processor)
	assert.Equal(t, "http://judge0.test", processor.judge0URL)
	assert.NotNil(t, processor.localExecutor)
}

func TestExecuteAlgorithm(t *testing.T) {
	processor := NewAlgorithmProcessor("")
	ctx := context.Background()
	
	tests := []struct {
		name     string
		req      AlgorithmExecutionRequest
		wantErr  bool
	}{
		{
			name: "Python execution",
			req: AlgorithmExecutionRequest{
				Language: "python",
				Code:     `print("hello")`,
				Stdin:    "",
			},
			wantErr: false,
		},
		{
			name: "JavaScript execution",
			req: AlgorithmExecutionRequest{
				Language: "javascript",
				Code:     `console.log("test")`,
				Stdin:    "",
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ExecuteAlgorithm(ctx, tt.req)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestValidateBatch(t *testing.T) {
	processor := NewAlgorithmProcessor("")
	ctx := context.Background()
	
	req := BatchValidationRequest{
		AlgorithmID: "sort",
		Language:    "python",
		Implementation: `
def sort_array(arr):
    return sorted(arr)

import sys
import json
arr = json.loads(sys.argv[1] if len(sys.argv) > 1 else input())
print(json.dumps(sort_array(arr)))
`,
		TestCases: []TestCase{
			{
				Input: map[string]interface{}{
					"arr": []int{3, 1, 2},
				},
				Expected: []int{1, 2, 3},
			},
		},
	}
	
	result, err := processor.ValidateBatch(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.TestResults)
}

func TestGetLanguageID(t *testing.T) {
	processor := NewAlgorithmProcessor("")
	
	tests := []struct {
		language string
		wantID   int
		wantErr  bool
	}{
		{"python", 71, false},
		{"javascript", 63, false},
		{"go", 60, false},
		{"java", 62, false},
		{"cpp", 54, false},
		{"invalid", 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			id, err := processor.getLanguageID(tt.language)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestFormatExpectedOutput(t *testing.T) {
	processor := NewAlgorithmProcessor("")

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "String",
			input:    "test",
			expected: `"test"`, // JSON marshals strings with quotes
		},
		{
			name:     "Integer",
			input:    42,
			expected: "42",
		},
		{
			name:     "Array",
			input:    []int{1, 2, 3},
			expected: "[1,2,3]",
		},
		{
			name:     "Map",
			input:    map[string]int{"a": 1},
			expected: `{"a":1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.formatExpectedOutput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	str := generateRandomString(10)
	assert.Len(t, str, 10)
	
	// Should be different each time
	str2 := generateRandomString(10)
	assert.Len(t, str2, 10)
	// This might fail very rarely but probability is extremely low
	assert.NotEqual(t, str, str2)
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1.23", 1.23},
		{"0", 0},
		{"invalid", 0},
		{"", 0},
		{"-4.56", -4.56},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseFloat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseExecutionTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "With ms suffix and space",
			input:   "100 ms",
			want:    100,
			wantErr: false,
		},
		{
			name:    "With decimal and ms",
			input:   "123.45 ms",
			want:    123.45,
			wantErr: false,
		},
		{
			name:    "Invalid format - no space",
			input:   "100ms",
			want:    0,
			wantErr: true,
		},
		{
			name:    "Invalid format - no ms",
			input:   "123.45",
			want:    0,
			wantErr: true,
		},
		{
			name:    "Invalid format",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExecutionTime(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestJudge0MockServer(t *testing.T) {
	// Create a test server to simulate Judge0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/about":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version": "1.13.0",
				"status":  "operational",
			})
		case "/submissions":
			if r.Method == "POST" {
				// Create submission response
				json.NewEncoder(w).Encode(map[string]interface{}{
					"token": "test-token-123",
				})
			}
		case "/submissions/test-token-123":
			// Get submission result
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"id":          3,
					"description": "Accepted",
				},
				"stdout":   "Hello, World!\n",
				"stderr":   nil,
				"time":     "0.123",
				"memory":   1234,
				"token":    "test-token-123",
				"compile_output": nil,
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	
	// Test Judge0 availability check
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", server.URL+"/about", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}