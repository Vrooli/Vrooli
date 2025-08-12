package execution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	
	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/pkg/interfaces"
)

// N8nStrategy implements ExecutionStrategy for n8n platform
type N8nStrategy struct {
	baseURL         string
	httpClients     interfaces.HTTPClientFactory
	circuitBreakers interfaces.CircuitBreakerManager
}

// NewN8nStrategy creates a new n8n execution strategy
func NewN8nStrategy(baseURL string, httpClients interfaces.HTTPClientFactory, circuitBreakers interfaces.CircuitBreakerManager) ExecutionStrategy {
	return &N8nStrategy{
		baseURL:         baseURL,
		httpClients:     httpClients,
		circuitBreakers: circuitBreakers,
	}
}

// Execute runs the workflow on n8n
func (n *N8nStrategy) Execute(wf *WorkflowEntity, req *ExecutionRequest) (*ExecutionResponse, error) {
	if wf.WebhookPath == nil {
		return nil, fmt.Errorf("n8n workflow missing webhook path")
	}
	
	url := fmt.Sprintf("%s/webhook/%s", n.baseURL, *wf.WebhookPath)
	
	// Prepare request body for n8n
	body := map[string]interface{}{
		"input":   req.Input,
		"context": req.Context,
		"model":   req.Model,
	}
	
	if req.Metadata != nil {
		body["metadata"] = req.Metadata
	}
	
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal n8n request: %w", err)
	}
	
	// Execute with circuit breaker protection
	var result interface{}
	breaker := n.circuitBreakers.GetBreaker("n8n")
	
	err = breaker.Call(func() error {
		client := n.httpClients.GetClient(interfaces.WorkflowClient)
		
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to call n8n: %w", err)
		}
		defer resp.Body.Close()
		
		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read n8n response: %w", err)
		}
		
		// Check HTTP status
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("n8n returned status %d: %s", resp.StatusCode, string(respBody))
		}
		
		// Parse JSON response
		if err := json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("failed to parse n8n response: %w", err)
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

// Validate checks if the workflow is valid for n8n
func (n *N8nStrategy) Validate(wf *WorkflowEntity) error {
	if wf.Platform != common.PlatformN8n && wf.Platform != common.PlatformBoth {
		return fmt.Errorf("workflow platform %s is not compatible with n8n strategy", wf.Platform)
	}
	
	if wf.WebhookPath == nil || *wf.WebhookPath == "" {
		return fmt.Errorf("n8n workflow must have a webhook path")
	}
	
	// Validate webhook path format
	webhookPath := *wf.WebhookPath
	if len(webhookPath) < 3 {
		return fmt.Errorf("webhook path too short: %s", webhookPath)
	}
	
	return nil
}

// Export exports workflow to n8n format
func (n *N8nStrategy) Export(wf *WorkflowEntity, format string) (interface{}, error) {
	// Parse the workflow config
	var config map[string]interface{}
	if err := json.Unmarshal(wf.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to parse workflow config: %w", err)
	}
	
	// Create n8n export format
	export := map[string]interface{}{
		"format":   format,
		"platform": "n8n",
		"data": map[string]interface{}{
			"name":        wf.Name,
			"nodes":       config["nodes"],
			"connections": config["connections"],
			"settings": map[string]interface{}{
				"executionOrder": "v1",
			},
		},
	}
	
	if wf.WebhookPath != nil {
		export["webhook_path"] = *wf.WebhookPath
	}
	
	return export, nil
}

// Import imports workflow from n8n format
func (n *N8nStrategy) Import(data json.RawMessage, name string) (*WorkflowEntity, error) {
	var importData struct {
		Data struct {
			Name        string                 `json:"name"`
			Nodes       []interface{}          `json:"nodes"`
			Connections map[string]interface{} `json:"connections"`
			Settings    map[string]interface{} `json:"settings"`
		} `json:"data"`
		WebhookPath string `json:"webhook_path,omitempty"`
	}
	
	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, fmt.Errorf("failed to parse n8n import data: %w", err)
	}
	
	// Use provided name or fall back to imported name
	workflowName := name
	if workflowName == "" {
		workflowName = importData.Data.Name
	}
	
	// Reconstruct workflow config
	config := map[string]interface{}{
		"nodes":       importData.Data.Nodes,
		"connections": importData.Data.Connections,
		"settings":    importData.Data.Settings,
	}
	
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow config: %w", err)
	}
	
	// Create workflow entity
	wf := &WorkflowEntity{
		ID:       uuid.New(),
		Name:     workflowName,
		Platform: common.PlatformN8n,
		Config:   configJSON,
	}
	
	// Set webhook path if provided
	if importData.WebhookPath != "" {
		wf.WebhookPath = &importData.WebhookPath
	}
	
	return wf, nil
}

// GetPlatform returns the platform this strategy handles
func (n *N8nStrategy) GetPlatform() common.Platform {
	return common.PlatformN8n
}