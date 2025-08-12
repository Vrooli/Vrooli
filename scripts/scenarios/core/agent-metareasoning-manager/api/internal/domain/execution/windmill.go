package execution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	
	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/pkg/interfaces"
)

// WindmillStrategy implements ExecutionStrategy for Windmill platform
type WindmillStrategy struct {
	baseURL         string
	workspace       string
	httpClients     interfaces.HTTPClientFactory
	circuitBreakers interfaces.CircuitBreakerManager
}

// NewWindmillStrategy creates a new Windmill execution strategy
func NewWindmillStrategy(baseURL, workspace string, httpClients interfaces.HTTPClientFactory, circuitBreakers interfaces.CircuitBreakerManager) ExecutionStrategy {
	return &WindmillStrategy{
		baseURL:         baseURL,
		workspace:       workspace,
		httpClients:     httpClients,
		circuitBreakers: circuitBreakers,
	}
}

// Execute runs the workflow on Windmill
func (w *WindmillStrategy) Execute(wf *WorkflowEntity, req *ExecutionRequest) (*ExecutionResponse, error) {
	if wf.JobPath == nil {
		return nil, fmt.Errorf("windmill workflow missing job path")
	}
	
	url := fmt.Sprintf("%s/api/w/%s/jobs/run/p/%s",
		w.baseURL, w.workspace, *wf.JobPath)
	
	// Prepare request body for Windmill
	body := map[string]interface{}{
		"args": map[string]interface{}{
			"input":   req.Input,
			"context": req.Context,
			"model":   req.Model,
		},
	}
	
	if req.Metadata != nil {
		body["args"].(map[string]interface{})["metadata"] = req.Metadata
	}
	
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal windmill request: %w", err)
	}
	
	// Execute with circuit breaker protection
	var result interface{}
	breaker := w.circuitBreakers.GetBreaker("windmill")
	
	err = breaker.Call(func() error {
		client := w.httpClients.GetClient(interfaces.WorkflowClient)
		
		httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to create windmill request: %w", err)
		}
		
		httpReq.Header.Set("Content-Type", "application/json")
		// TODO: Add proper authentication headers for Windmill
		
		resp, err := client.Do(httpReq)
		if err != nil {
			return fmt.Errorf("failed to call windmill: %w", err)
		}
		defer resp.Body.Close()
		
		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read windmill response: %w", err)
		}
		
		// Check HTTP status
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("windmill returned status %d: %s", resp.StatusCode, string(respBody))
		}
		
		// Parse JSON response
		if err := json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("failed to parse windmill response: %w", err)
		}
		
		return nil
	})
	
	if err != nil {
		return &ExecutionResponse{
			ID:     uuid.New(),
			Status: common.StatusFailed,
			Error:  err.Error(),
		}, nil // Return response with error, not error itself
	}
	
	return &ExecutionResponse{
		ID:     uuid.New(),
		Status: common.StatusSuccess,
		Data:   result,
	}, nil
}

// Validate checks if the workflow is valid for Windmill
func (w *WindmillStrategy) Validate(wf *WorkflowEntity) error {
	if wf.Platform != common.PlatformWindmill && wf.Platform != common.PlatformBoth {
		return fmt.Errorf("workflow platform %s is not compatible with windmill strategy", wf.Platform)
	}
	
	if wf.JobPath == nil || *wf.JobPath == "" {
		return fmt.Errorf("windmill workflow must have a job path")
	}
	
	// Validate job path format
	jobPath := *wf.JobPath
	if len(jobPath) < 3 {
		return fmt.Errorf("job path too short: %s", jobPath)
	}
	
	// Job path should not contain spaces or special characters
	if strings.ContainsAny(jobPath, " \t\n\r") {
		return fmt.Errorf("job path contains invalid characters: %s", jobPath)
	}
	
	return nil
}

// Export exports workflow to Windmill format
func (w *WindmillStrategy) Export(wf *WorkflowEntity, format string) (interface{}, error) {
	// Parse the workflow config
	var config map[string]interface{}
	if err := json.Unmarshal(wf.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to parse workflow config: %w", err)
	}
	
	// Create Windmill export format
	export := map[string]interface{}{
		"format":   format,
		"platform": "windmill",
		"data": map[string]interface{}{
			"summary":     wf.Name,
			"description": config["description"],
			"language":    config["language"],
			"content":     config["content"],
			"schema":      config["schema"],
		},
	}
	
	if wf.JobPath != nil {
		export["job_path"] = *wf.JobPath
	}
	
	return export, nil
}

// Import imports workflow from Windmill format
func (w *WindmillStrategy) Import(data json.RawMessage, name string) (*WorkflowEntity, error) {
	var importData struct {
		Data struct {
			Summary     string                 `json:"summary"`
			Description interface{}            `json:"description"`
			Language    string                 `json:"language"`
			Content     string                 `json:"content"`
			Schema      map[string]interface{} `json:"schema"`
		} `json:"data"`
		JobPath string `json:"job_path,omitempty"`
	}
	
	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, fmt.Errorf("failed to parse windmill import data: %w", err)
	}
	
	// Use provided name or fall back to imported summary
	workflowName := name
	if workflowName == "" {
		workflowName = importData.Data.Summary
	}
	
	// Reconstruct workflow config
	config := map[string]interface{}{
		"description": importData.Data.Description,
		"language":    importData.Data.Language,
		"content":     importData.Data.Content,
		"schema":      importData.Data.Schema,
	}
	
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow config: %w", err)
	}
	
	// Create workflow entity
	wf := &WorkflowEntity{
		ID:       uuid.New(),
		Name:     workflowName,
		Platform: common.PlatformWindmill,
		Config:   configJSON,
	}
	
	// Set job path if provided
	if importData.JobPath != "" {
		wf.JobPath = &importData.JobPath
	}
	
	return wf, nil
}

// GetPlatform returns the platform this strategy handles
func (w *WindmillStrategy) GetPlatform() common.Platform {
	return common.PlatformWindmill
}