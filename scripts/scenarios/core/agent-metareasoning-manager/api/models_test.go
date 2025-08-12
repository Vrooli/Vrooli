package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWorkflowStructValidation(t *testing.T) {
	tests := []struct {
		name    string
		wf      Workflow
		wantErr bool
	}{
		{
			name: "valid workflow",
			wf: Workflow{
				ID:          uuid.New(),
				Name:        "Test Workflow",
				Description: "Test description",
				Type:        "data-analysis",
				Platform:    "n8n",
				Config:      json.RawMessage(`{"nodes": []}`),
				Version:     1,
				IsActive:    true,
				IsBuiltin:   false,
				Tags:        []string{"test", "analysis"},
				CreatedBy:   "testuser",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "workflow with all platforms",
			wf: Workflow{
				ID:       uuid.New(),
				Name:     "Multi-Platform",
				Platform: "both",
				Config:   json.RawMessage(`{}`),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			data, err := json.Marshal(tt.wf)
			if err != nil {
				t.Fatalf("Failed to marshal workflow: %v", err)
			}

			var decoded Workflow
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal workflow: %v", err)
			}

			// Verify key fields
			if decoded.ID != tt.wf.ID {
				t.Errorf("ID mismatch: got %v, want %v", decoded.ID, tt.wf.ID)
			}
			if decoded.Name != tt.wf.Name {
				t.Errorf("Name mismatch: got %v, want %v", decoded.Name, tt.wf.Name)
			}
		})
	}
}

func TestWorkflowCreateValidation(t *testing.T) {
	tests := []struct {
		name    string
		wc      WorkflowCreate
		wantErr bool
	}{
		{
			name: "valid create request",
			wc: WorkflowCreate{
				Name:        "New Workflow",
				Description: "Description",
				Type:        "automation",
				Platform:    "windmill",
				Config:      json.RawMessage(`{"script": "console.log('test')"}`),
				Tags:        []string{"new", "test"},
			},
			wantErr: false,
		},
		{
			name: "minimal create request",
			wc: WorkflowCreate{
				Name:     "Minimal",
				Type:     "test",
				Platform: "n8n",
				Config:   json.RawMessage(`{}`),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.wc)
			if err != nil {
				t.Fatalf("Failed to marshal WorkflowCreate: %v", err)
			}

			var decoded WorkflowCreate
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal WorkflowCreate: %v", err)
			}

			if decoded.Name != tt.wc.Name {
				t.Errorf("Name mismatch: got %v, want %v", decoded.Name, tt.wc.Name)
			}
		})
	}
}

func TestExecutionRequestValidation(t *testing.T) {
	tests := []struct {
		name string
		req  ExecutionRequest
	}{
		{
			name: "request with input data",
			req: ExecutionRequest{
				Input:    json.RawMessage(`{"data": "test input"}`),
				Context:  "test context",
				Model:    "llama2",
				Metadata: map[string]interface{}{"timeout": 30},
			},
		},
		{
			name: "empty request",
			req:  ExecutionRequest{},
		},
		{
			name: "request with complex metadata",
			req: ExecutionRequest{
				Metadata: map[string]interface{}{
					"model":       "gpt-4",
					"temperature": 0.7,
					"max_tokens":  1000,
					"options":     []string{"opt1", "opt2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Failed to marshal ExecutionRequest: %v", err)
			}

			var decoded ExecutionRequest
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal ExecutionRequest: %v", err)
			}

			// Verify metadata if present
			if tt.req.Metadata != nil {
				if len(decoded.Metadata) != len(tt.req.Metadata) {
					t.Errorf("Metadata count mismatch: got %d, want %d",
						len(decoded.Metadata), len(tt.req.Metadata))
				}
			}
		})
	}
}

func TestExecutionResponseStructure(t *testing.T) {
	resp := ExecutionResponse{
		ID:          uuid.New(),
		WorkflowID:  uuid.New(),
		Status:      "success",
		Data:        json.RawMessage(`{"output": "test result"}`),
		ExecutionMS: 1500,
		Timestamp:   time.Now(),
		Error:       "",
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ExecutionResponse: %v", err)
	}

	var decoded ExecutionResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ExecutionResponse: %v", err)
	}

	if decoded.Status != resp.Status {
		t.Errorf("Status mismatch: got %v, want %v", decoded.Status, resp.Status)
	}
	if decoded.ExecutionMS != resp.ExecutionMS {
		t.Errorf("ExecutionMS mismatch: got %v, want %v", decoded.ExecutionMS, resp.ExecutionMS)
	}
}

func TestExecutionHistoryStructure(t *testing.T) {
	history := ExecutionHistory{
		ID:            uuid.New(),
		Status:        "completed",
		ExecutionTime: 2500,
		ModelUsed:     "llama2",
		CreatedAt:     time.Now(),
		InputSummary:  "test input summary",
		HasOutput:     true,
	}

	// Test with different status
	historyFailed := ExecutionHistory{
		ID:            uuid.New(),
		Status:        "failed",
		ExecutionTime: 1000,
		ModelUsed:     "gpt-4",
		CreatedAt:     time.Now(),
		InputSummary:  "failed execution",
		HasOutput:     false,
	}

	tests := []ExecutionHistory{history, historyFailed}

	for i, h := range tests {
		t.Run(fmt.Sprintf("history_%d", i), func(t *testing.T) {
			data, err := json.Marshal(h)
			if err != nil {
				t.Fatalf("Failed to marshal ExecutionHistory: %v", err)
			}

			var decoded ExecutionHistory
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal ExecutionHistory: %v", err)
			}

			if decoded.Status != h.Status {
				t.Errorf("Status mismatch: got %v, want %v", decoded.Status, h.Status)
			}
			if decoded.ExecutionTime != h.ExecutionTime {
				t.Errorf("ExecutionTime mismatch: got %v, want %v", decoded.ExecutionTime, h.ExecutionTime)
			}
			if decoded.HasOutput != h.HasOutput {
				t.Errorf("HasOutput mismatch: got %v, want %v", decoded.HasOutput, h.HasOutput)
			}
		})
	}
}

func TestMetricsResponseStructure(t *testing.T) {
	lastExec := time.Now()
	metrics := MetricsResponse{
		TotalExecutions:  100,
		SuccessCount:     85,
		FailureCount:     15,
		AvgExecutionTime: 1250,
		MinExecutionTime: 500,
		MaxExecutionTime: 3000,
		LastExecution:    &lastExec,
		SuccessRate:      85.0,
		ModelsUsed:       []string{"llama2", "gpt-4"},
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal MetricsResponse: %v", err)
	}

	var decoded MetricsResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal MetricsResponse: %v", err)
	}

	if decoded.TotalExecutions != metrics.TotalExecutions {
		t.Errorf("TotalExecutions mismatch: got %v, want %v",
			decoded.TotalExecutions, metrics.TotalExecutions)
	}
	if decoded.SuccessRate != metrics.SuccessRate {
		t.Errorf("SuccessRate mismatch: got %v, want %v",
			decoded.SuccessRate, metrics.SuccessRate)
	}
}

func TestOllamaStructures(t *testing.T) {
	// Test OllamaGenerateRequest
	genReq := OllamaGenerateRequest{
		Model:       "llama2",
		Prompt:      "Test prompt",
		Temperature: 0.7,
		Stream:      false,
	}

	data, err := json.Marshal(genReq)
	if err != nil {
		t.Fatalf("Failed to marshal OllamaGenerateRequest: %v", err)
	}

	var decodedReq OllamaGenerateRequest
	err = json.Unmarshal(data, &decodedReq)
	if err != nil {
		t.Fatalf("Failed to unmarshal OllamaGenerateRequest: %v", err)
	}

	if decodedReq.Model != genReq.Model {
		t.Errorf("Model mismatch: got %v, want %v", decodedReq.Model, genReq.Model)
	}

	// Test OllamaModel
	model := OllamaModel{
		Name:       "llama2:latest",
		ModifiedAt: "2024-01-15T10:30:00Z",
		Size:       1234567890,
	}

	data, err = json.Marshal(model)
	if err != nil {
		t.Fatalf("Failed to marshal OllamaModel: %v", err)
	}

	var decodedModel OllamaModel
	err = json.Unmarshal(data, &decodedModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal OllamaModel: %v", err)
	}

	if decodedModel.Name != model.Name {
		t.Errorf("Name mismatch: got %v, want %v", decodedModel.Name, model.Name)
	}
}

func TestPlatformInfoStructure(t *testing.T) {
	platform := PlatformInfo{
		Name:        "n8n",
		Description: "n8n automation platform",
		Status:      true,
		URL:         "http://localhost:5678",
	}

	data, err := json.Marshal(platform)
	if err != nil {
		t.Fatalf("Failed to marshal PlatformStatus: %v", err)
	}

	var decoded PlatformInfo
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal PlatformStatus: %v", err)
	}

	if decoded.Name != platform.Name {
		t.Errorf("Name mismatch: got %v, want %v", decoded.Name, platform.Name)
	}
	if decoded.Status != platform.Status {
		t.Errorf("Status mismatch: got %v, want %v", decoded.Status, platform.Status)
	}
}

func TestStatsResponseStructure(t *testing.T) {
	lastExec := time.Now()
	stats := StatsResponse{
		TotalWorkflows:    50,
		ActiveWorkflows:   45,
		TotalExecutions:   1000,
		SuccessfulExecs:   925,
		FailedExecs:       75,
		AvgExecutionTime:  1250,
		MostUsedWorkflow:  "Data Analysis Pipeline",
		MostUsedModel:     "llama2",
		LastExecution:     &lastExec,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal SystemStats: %v", err)
	}

	var decoded StatsResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal SystemStats: %v", err)
	}

	if decoded.TotalWorkflows != stats.TotalWorkflows {
		t.Errorf("TotalWorkflows mismatch: got %v, want %v",
			decoded.TotalWorkflows, stats.TotalWorkflows)
	}
	if decoded.MostUsedWorkflow != stats.MostUsedWorkflow {
		t.Errorf("MostUsedWorkflow mismatch: got %v, want %v",
			decoded.MostUsedWorkflow, stats.MostUsedWorkflow)
	}
	if decoded.AvgExecutionTime != stats.AvgExecutionTime {
		t.Errorf("AvgExecutionTime mismatch: got %v, want %v",
			decoded.AvgExecutionTime, stats.AvgExecutionTime)
	}
}

func TestListResponsePagination(t *testing.T) {
	workflows := []Workflow{
		{ID: uuid.New(), Name: "Workflow 1"},
		{ID: uuid.New(), Name: "Workflow 2"},
	}

	resp := ListResponse{
		Workflows: workflows,
		Total:     100,
		Page:      1,
		PageSize:  20,
	}

	// Verify pagination calculations
	totalPages := (resp.Total + resp.PageSize - 1) / resp.PageSize
	if totalPages != 5 {
		t.Errorf("Expected 5 pages for 100 items with page size 20, got %d", totalPages)
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ListResponse: %v", err)
	}

	var decoded ListResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ListResponse: %v", err)
	}

	if len(decoded.Workflows) != len(resp.Workflows) {
		t.Errorf("Workflows count mismatch: got %d, want %d",
			len(decoded.Workflows), len(resp.Workflows))
	}
}

// fmt is already imported at the top of the file