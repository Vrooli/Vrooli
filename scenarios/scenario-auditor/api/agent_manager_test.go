//go:build legacy_auditor_tests
// +build legacy_auditor_tests

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockHTTPClient struct {
	responses map[string]*http.Response
	mutex     sync.RWMutex
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", req.Method, req.URL.String())
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}

	return &http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader("Not found")),
		Header:     make(http.Header),
	}, nil
}

func (m *mockHTTPClient) addResponse(method, url string, statusCode int, body string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.responses == nil {
		m.responses = make(map[string]*http.Response)
	}

	key := fmt.Sprintf("%s:%s", method, url)
	m.responses[key] = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNewAgentManager(t *testing.T) {
	tempDir := t.TempDir()
	config := AgentConfig{
		Provider:   "openai",
		Model:      "gpt-4",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		LogDir:     tempDir,
	}

	manager := NewAgentManager(config)
	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.config.Provider != config.Provider {
		t.Errorf("Expected provider %s, got %s", config.Provider, manager.config.Provider)
	}

	if manager.config.Model != config.Model {
		t.Errorf("Expected model %s, got %s", config.Model, manager.config.Model)
	}

	if manager.agents == nil {
		t.Error("Expected agents map to be initialized")
	}
}

func TestAgentManager_CreateAgent(t *testing.T) {
	tempDir := t.TempDir()
	manager := &AgentManager{
		config: AgentConfig{
			Provider: "openai",
			Model:    "gpt-4",
			LogDir:   tempDir,
		},
		agents: make(map[string]*Agent),
		client: &http.Client{Timeout: 30 * time.Second},
	}

	tests := []struct {
		name        string
		agentType   AgentType
		wantErr     bool
		checkFields func(t *testing.T, agent *Agent)
	}{
		{
			name:      "create scanner agent",
			agentType: AgentScanner,
			wantErr:   false,
			checkFields: func(t *testing.T, agent *Agent) {
				if agent.Type != AgentScanner {
					t.Errorf("Expected type %v, got %v", AgentScanner, agent.Type)
				}
				if agent.SystemPrompt == "" {
					t.Error("Expected system prompt to be set")
				}
			},
		},
		{
			name:      "create standards agent",
			agentType: AgentStandards,
			wantErr:   false,
			checkFields: func(t *testing.T, agent *Agent) {
				if agent.Type != AgentStandards {
					t.Errorf("Expected type %v, got %v", AgentStandards, agent.Type)
				}
			},
		},
		{
			name:      "create analysis agent",
			agentType: AgentAnalysis,
			wantErr:   false,
		},
		{
			name:      "create fix agent",
			agentType: AgentFix,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := manager.CreateAgent(tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if agent == nil {
					t.Fatal("Expected agent to be created")
				}

				if agent.ID == "" {
					t.Error("Expected agent ID to be set")
				}

				if agent.State != AgentStateReady {
					t.Errorf("Expected state %v, got %v", AgentStateReady, agent.State)
				}

				if agent.CreatedAt.IsZero() {
					t.Error("Expected CreatedAt to be set")
				}

				if _, ok := manager.agents[agent.ID]; !ok {
					t.Error("Expected agent to be stored in manager")
				}

				logFile := filepath.Join(tempDir, fmt.Sprintf("%s.log", agent.ID))
				if _, err := os.Stat(logFile); os.IsNotExist(err) {
					t.Error("Expected log file to be created")
				}

				if tt.checkFields != nil {
					tt.checkFields(t, agent)
				}
			}
		})
	}
}

func TestAgentManager_ExecuteTask(t *testing.T) {
	mockClient := &mockHTTPClient{}

	manager := &AgentManager{
		config: AgentConfig{
			Provider: "openai",
			Model:    "gpt-4",
			LogDir:   t.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: mockClient,
	}

	agent, err := manager.CreateAgent(AgentAnalysis)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	tests := []struct {
		name           string
		task           AgentTask
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResult    func(t *testing.T, result *AgentResult)
	}{
		{
			name: "successful task execution",
			task: AgentTask{
				ID:      "test-task-1",
				AgentID: agent.ID,
				Type:    "analyze",
				Input: map[string]interface{}{
					"code": "func main() {}",
				},
			},
			mockResponse: `{
				"choices": [{
					"message": {
						"content": "Analysis complete: No issues found"
					}
				}]
			}`,
			mockStatusCode: 200,
			wantErr:        false,
			checkResult: func(t *testing.T, result *AgentResult) {
				if result.Success != true {
					t.Error("Expected success to be true")
				}
				if result.Output == nil {
					t.Error("Expected output to be set")
				}
			},
		},
		{
			name: "API error",
			task: AgentTask{
				ID:      "test-task-2",
				AgentID: agent.ID,
				Type:    "analyze",
				Input:   map[string]interface{}{},
			},
			mockResponse:   `{"error": {"message": "Invalid request"}}`,
			mockStatusCode: 400,
			wantErr:        true,
		},
		{
			name: "invalid agent ID",
			task: AgentTask{
				ID:      "test-task-3",
				AgentID: "invalid-id",
				Type:    "analyze",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockStatusCode > 0 {
				mockClient.addResponse("POST", "https://api.openai.com/v1/chat/completions",
					tt.mockStatusCode, tt.mockResponse)
			}

			ctx := context.Background()
			result, err := manager.ExecuteTask(ctx, tt.task)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestAgentManager_GetAgent(t *testing.T) {
	manager := &AgentManager{
		config: AgentConfig{
			LogDir: t.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: &http.Client{},
	}

	agent, _ := manager.CreateAgent(AgentScanner)

	tests := []struct {
		name    string
		agentID string
		want    *Agent
		wantErr bool
	}{
		{
			name:    "existing agent",
			agentID: agent.ID,
			want:    agent,
			wantErr: false,
		},
		{
			name:    "non-existent agent",
			agentID: "invalid-id",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty ID",
			agentID: "",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.GetAgent(tt.agentID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetAgent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgentManager_ListAgents(t *testing.T) {
	manager := &AgentManager{
		config: AgentConfig{
			LogDir: t.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: &http.Client{},
	}

	agents := manager.ListAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}

	agent1, _ := manager.CreateAgent(AgentScanner)
	agent2, _ := manager.CreateAgent(AgentStandards)

	agents = manager.ListAgents()
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}

	foundAgent1 := false
	foundAgent2 := false
	for _, a := range agents {
		if a.ID == agent1.ID {
			foundAgent1 = true
		}
		if a.ID == agent2.ID {
			foundAgent2 = true
		}
	}

	if !foundAgent1 || !foundAgent2 {
		t.Error("Not all agents were returned")
	}
}

func TestAgentManager_TerminateAgent(t *testing.T) {
	manager := &AgentManager{
		config: AgentConfig{
			LogDir: t.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: &http.Client{},
	}

	agent, _ := manager.CreateAgent(AgentScanner)
	agentID := agent.ID

	err := manager.TerminateAgent(agentID)
	if err != nil {
		t.Errorf("TerminateAgent() error = %v", err)
	}

	if _, exists := manager.agents[agentID]; exists {
		t.Error("Agent should have been removed from manager")
	}

	err = manager.TerminateAgent(agentID)
	if err == nil {
		t.Error("Expected error when terminating non-existent agent")
	}
}

func TestAgentManager_Cleanup(t *testing.T) {
	manager := &AgentManager{
		config: AgentConfig{
			LogDir: t.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: &http.Client{},
	}

	agent1, _ := manager.CreateAgent(AgentScanner)
	agent2, _ := manager.CreateAgent(AgentStandards)
	agent3, _ := manager.CreateAgent(AgentAnalysis)

	agent1.LastActivity = time.Now().Add(-2 * time.Hour)
	agent2.LastActivity = time.Now()
	agent3.State = AgentStateTerminated

	manager.Cleanup()

	if _, exists := manager.agents[agent1.ID]; exists {
		t.Error("Idle agent should have been cleaned up")
	}

	if _, exists := manager.agents[agent2.ID]; !exists {
		t.Error("Active agent should not have been cleaned up")
	}

	if _, exists := manager.agents[agent3.ID]; exists {
		t.Error("Terminated agent should have been cleaned up")
	}
}

func TestAgentManager_generatePrompt(t *testing.T) {
	manager := &AgentManager{
		config: AgentConfig{},
	}

	tests := []struct {
		name     string
		task     AgentTask
		agent    *Agent
		expected []string
	}{
		{
			name: "scanner agent prompt",
			task: AgentTask{
				Type: "scan",
				Input: map[string]interface{}{
					"target": "test-file.go",
				},
			},
			agent: &Agent{
				Type:         AgentScanner,
				SystemPrompt: "You are a security scanner",
			},
			expected: []string{"security scanner", "test-file.go"},
		},
		{
			name: "fix agent prompt",
			task: AgentTask{
				Type: "fix",
				Input: map[string]interface{}{
					"issue": "SQL injection vulnerability",
				},
			},
			agent: &Agent{
				Type:         AgentFix,
				SystemPrompt: "You fix security issues",
			},
			expected: []string{"fix security issues", "SQL injection"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := manager.generatePrompt(tt.task, tt.agent)

			for _, exp := range tt.expected {
				if !strings.Contains(prompt, exp) {
					t.Errorf("Expected prompt to contain '%s', got: %s", exp, prompt)
				}
			}
		})
	}
}

func TestAgent_UpdateState(t *testing.T) {
	agent := &Agent{
		State: AgentStateReady,
	}

	tests := []struct {
		name     string
		newState AgentState
		wantErr  bool
	}{
		{
			name:     "valid state transition",
			newState: AgentStateBusy,
			wantErr:  false,
		},
		{
			name:     "another valid transition",
			newState: AgentStateTerminated,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agent.UpdateState(tt.newState)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateState() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && agent.State != tt.newState {
				t.Errorf("Expected state %v, got %v", tt.newState, agent.State)
			}
		})
	}
}

func TestAgent_GetMetrics(t *testing.T) {
	agent := &Agent{
		Metrics: AgentMetrics{
			TasksCompleted: 10,
			TasksFailed:    2,
			TotalTime:      5 * time.Minute,
		},
	}

	metrics := agent.GetMetrics()

	if metrics.TasksCompleted != 10 {
		t.Errorf("Expected TasksCompleted to be 10, got %d", metrics.TasksCompleted)
	}

	if metrics.TasksFailed != 2 {
		t.Errorf("Expected TasksFailed to be 2, got %d", metrics.TasksFailed)
	}

	if metrics.TotalTime != 5*time.Minute {
		t.Errorf("Expected TotalTime to be 5 minutes, got %v", metrics.TotalTime)
	}
}

func TestConcurrentAgentOperations(t *testing.T) {
	manager := &AgentManager{
		config: AgentConfig{
			LogDir: t.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: &http.Client{},
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			agentType := AgentScanner
			if id%2 == 0 {
				agentType = AgentStandards
			}

			agent, err := manager.CreateAgent(agentType)
			if err != nil {
				t.Errorf("Failed to create agent: %v", err)
				return
			}

			time.Sleep(10 * time.Millisecond)

			agents := manager.ListAgents()
			if len(agents) == 0 {
				t.Error("Expected agents to exist")
			}

			if id%3 == 0 {
				manager.TerminateAgent(agent.ID)
			}
		}(i)
	}

	wg.Wait()

	agents := manager.ListAgents()
	t.Logf("Final agent count: %d", len(agents))
}

func TestAgentHTTPHandlers(t *testing.T) {
	manager := &AgentManager{
		config: AgentConfig{
			LogDir: t.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: &http.Client{},
	}

	t.Run("HandleCreateAgent", func(t *testing.T) {
		reqBody := `{"type": "scanner"}`
		req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		HandleCreateAgent(manager)(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["id"] == nil {
			t.Error("Expected agent ID in response")
		}
	})

	t.Run("HandleListAgents", func(t *testing.T) {
		manager.CreateAgent(AgentScanner)
		manager.CreateAgent(AgentStandards)

		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		w := httptest.NewRecorder()

		HandleListAgents(manager)(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if len(result) < 2 {
			t.Errorf("Expected at least 2 agents, got %d", len(result))
		}
	})
}

func BenchmarkAgentCreation(b *testing.B) {
	manager := &AgentManager{
		config: AgentConfig{
			LogDir: b.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: &http.Client{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent, _ := manager.CreateAgent(AgentScanner)
		manager.TerminateAgent(agent.ID)
	}
}

func BenchmarkConcurrentTaskExecution(b *testing.B) {
	mockClient := &mockHTTPClient{}
	mockClient.addResponse("POST", "https://api.openai.com/v1/chat/completions",
		200, `{"choices": [{"message": {"content": "test"}}]}`)

	manager := &AgentManager{
		config: AgentConfig{
			Provider: "openai",
			Model:    "gpt-4",
			LogDir:   b.TempDir(),
		},
		agents: make(map[string]*Agent),
		client: mockClient,
	}

	agent, _ := manager.CreateAgent(AgentScanner)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := context.Background()
			task := AgentTask{
				ID:      fmt.Sprintf("task-%d", time.Now().UnixNano()),
				AgentID: agent.ID,
				Type:    "analyze",
				Input:   map[string]interface{}{"test": "data"},
			}
			manager.ExecuteTask(ctx, task)
		}
	})
}
