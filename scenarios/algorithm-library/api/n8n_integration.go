package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// N8nWorkflowClient handles integration with n8n workflows
type N8nWorkflowClient struct {
	baseURL     string
	webhookPath string
	httpClient  *http.Client
}

// N8nExecutionRequest represents a request to execute code via n8n
type N8nExecutionRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Input    string `json:"input"`
}

// N8nExecutionResponse represents the response from n8n workflow
type N8nExecutionResponse struct {
	Success bool    `json:"success"`
	Output  string  `json:"output"`
	Error   string  `json:"error"`
	Time    float64 `json:"time"`
	Memory  int     `json:"memory"`
	Status  string  `json:"status"`
}

// NewN8nWorkflowClient creates a new n8n workflow client
func NewN8nWorkflowClient() *N8nWorkflowClient {
	n8nHost := os.Getenv("N8N_HOST")
	if n8nHost == "" {
		n8nHost = "localhost"
	}

	n8nPort := os.Getenv("N8N_PORT")
	if n8nPort == "" {
		n8nPort = "5678"
	}

	return &N8nWorkflowClient{
		baseURL:     fmt.Sprintf("http://%s:%s", n8nHost, n8nPort),
		webhookPath: "/webhook/algorithm-executor",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsAvailable checks if n8n is available and the workflow is active
func (c *N8nWorkflowClient) IsAvailable() bool {
	// Check if n8n is healthy
	resp, err := c.httpClient.Get(c.baseURL + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ExecuteCode executes code through the n8n workflow
func (c *N8nWorkflowClient) ExecuteCode(language, code, input string) (*N8nExecutionResponse, error) {
	req := N8nExecutionRequest{
		Language: language,
		Code:     code,
		Input:    input,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+c.webhookPath,
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute workflow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workflow returned status %d", resp.StatusCode)
	}

	var result N8nExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// WrapCodeWithTestHarness wraps user code with test harness that calls it with input and prints output
func WrapCodeWithTestHarness(language, code, algorithmName, inputJSON string) string {
	switch language {
	case "python":
		return fmt.Sprintf(`%s

# Test harness
import json
test_input = json.loads('%s')
result = %s(test_input['arr']) if 'arr' in test_input else %s(test_input.get('n', test_input.get('array', None)))
print(json.dumps(result))
`, code, inputJSON, algorithmName, algorithmName)
	case "javascript":
		return fmt.Sprintf(`%s

// Test harness
const testInput = %s;
const result = %s(testInput.arr !== undefined ? testInput.arr : (testInput.n !== undefined ? testInput.n : testInput.array));
console.log(JSON.stringify(result));
`, code, inputJSON, algorithmName)
	case "go":
		return fmt.Sprintf(`package main
import (
	"encoding/json"
	"fmt"
)

%s

func main() {
	input := %s
	var testInput map[string]interface{}
	json.Unmarshal([]byte(input), &testInput)
	var arr []int
	if testInput["arr"] != nil {
		for _, v := range testInput["arr"].([]interface{}) {
			arr = append(arr, int(v.(float64)))
		}
	}
	result := %s(arr)
	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}
`, code, inputJSON, algorithmName)
	default:
		// For unsupported languages, return code as-is
		return code
	}
}

// ExecuteWithFallback tries to execute via n8n, falls back to local execution
func ExecuteWithFallback(language, code, input string, timeout time.Duration) (*LocalExecutionResult, error) {
	// Try n8n first if available
	n8nClient := NewN8nWorkflowClient()
	if n8nClient.IsAvailable() {
		n8nResult, err := n8nClient.ExecuteCode(language, code, input)
		if err == nil && n8nResult != nil {
			// Convert n8n result to LocalExecutionResult
			return &LocalExecutionResult{
				Success:       n8nResult.Success,
				Output:        n8nResult.Output,
				Error:         n8nResult.Error,
				ExecutionTime: n8nResult.Time,
			}, nil
		}
		// If n8n fails, fall back to local execution
	}

	// Fallback to local executor
	executor := NewLocalExecutor(timeout)

	switch language {
	case "python":
		return executor.ExecutePython(code, input)
	case "javascript":
		return executor.ExecuteJavaScript(code, input)
	case "go":
		return executor.ExecuteGo(code, input)
	case "java":
		return executor.ExecuteJava(code, input)
	case "cpp", "c++":
		return executor.ExecuteCPP(code, input)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}
