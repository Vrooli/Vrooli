package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// mockCommandRunner implements CommandRunner for testing.
type mockCommandRunner struct {
	lookPathResult string
	lookPathErr    error
	versionOutput  string

	// For capturing command execution
	lastCmd  string
	lastArgs []string
}

func (m *mockCommandRunner) LookPath(file string) (string, error) {
	if m.lookPathErr != nil {
		return "", m.lookPathErr
	}
	if m.lookPathResult != "" {
		return m.lookPathResult, nil
	}
	return "/usr/local/bin/" + file, nil
}

func (m *mockCommandRunner) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	m.lastCmd = name
	m.lastArgs = args

	// For version check, create a real command that outputs the version
	if len(args) > 0 && args[len(args)-1] == "--version" {
		output := m.versionOutput
		if output == "" {
			output = "claude-code 1.0.18"
		}
		cmd := exec.CommandContext(ctx, "echo", "-n", output)
		return cmd
	}

	// For actual navigation, we can't easily mock exec.Cmd
	// so we return a command that will fail gracefully
	cmd := exec.CommandContext(ctx, "echo", "test")
	return cmd
}

func TestNewClaudeCodeVisionNavigator(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("creates with defaults", func(t *testing.T) {
		nav := NewClaudeCodeVisionNavigator(log)

		if nav == nil {
			t.Fatal("NewClaudeCodeVisionNavigator returned nil")
		}
		if nav.log != log {
			t.Error("logger not set correctly")
		}
		if nav.activeNavigations == nil {
			t.Error("activeNavigations map not initialized")
		}
		if nav.cmdRunner == nil {
			t.Error("cmdRunner not initialized")
		}
	})

	t.Run("applies options", func(t *testing.T) {
		wsHub := &mockWSHub{}
		cmdRunner := &mockCommandRunner{}
		var recordedActions []*RecordedNavigationAction
		callback := func(sessionID string, action *RecordedNavigationAction) {
			recordedActions = append(recordedActions, action)
		}

		nav := NewClaudeCodeVisionNavigator(log,
			WithClaudeCodeHub(wsHub),
			WithClaudeCodeCommandRunner(cmdRunner),
			WithClaudeCodeActionRecordCallback(callback),
		)

		if nav.wsHub == nil {
			t.Error("WebSocket hub option not applied")
		}
		if nav.cmdRunner != cmdRunner {
			t.Error("command runner option not applied")
		}
		if nav.onActionRecord == nil {
			t.Error("action record callback option not applied")
		}
	})
}

func TestClaudeCodeVisionNavigator_Type(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	if nav.Type() != NavigatorClaudeCode {
		t.Errorf("Type() = %v, want %v", nav.Type(), NavigatorClaudeCode)
	}
}

func TestClaudeCodeVisionNavigator_Description(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	desc := nav.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(desc, "Claude Code") {
		t.Errorf("Description() = %q, expected to contain 'Claude Code'", desc)
	}
}

func TestClaudeCodeVisionNavigator_CreditPolicy(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	policy := nav.CreditPolicy()

	if policy.RequiresCredits {
		t.Error("RequiresCredits should be false for local execution")
	}
	if policy.CreditsPerStep != 0 {
		t.Errorf("CreditsPerStep = %d, want 0", policy.CreditsPerStep)
	}
	if policy.PerStepCharging {
		t.Error("PerStepCharging should be false")
	}
	if len(policy.BypassConditions) != 1 || policy.BypassConditions[0] != BypassLocalExecution {
		t.Error("BypassConditions should only contain BypassLocalExecution")
	}
}

func TestClaudeCodeVisionNavigator_ClientSourcePolicy(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	policy := nav.ClientSourcePolicy()

	// Should only allow CLI
	if !policy.IsAllowed(ClientSourceCLI) {
		t.Error("should allow CLI")
	}
	if policy.IsAllowed(ClientSourceUI) {
		t.Error("should not allow UI")
	}
	if policy.IsAllowed(ClientSourceAPI) {
		t.Error("should not allow API")
	}
}

func TestClaudeCodeVisionNavigator_IsAvailable(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("not available when claude not in PATH", func(t *testing.T) {
		cmdRunner := &mockCommandRunner{
			lookPathErr: exec.ErrNotFound,
		}
		nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeCommandRunner(cmdRunner))

		if nav.IsAvailable(t.Context()) {
			t.Error("IsAvailable() should return false when claude not in PATH")
		}
	})

	t.Run("available with sufficient version", func(t *testing.T) {
		cmdRunner := &mockCommandRunner{
			lookPathResult: "/usr/local/bin/claude",
			versionOutput:  "claude-code 1.0.18",
		}
		nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeCommandRunner(cmdRunner))

		if !nav.IsAvailable(t.Context()) {
			t.Error("IsAvailable() should return true with version 1.0.18")
		}
	})

	t.Run("available with newer version", func(t *testing.T) {
		cmdRunner := &mockCommandRunner{
			lookPathResult: "/usr/local/bin/claude",
			versionOutput:  "claude-code 2.0.0",
		}
		nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeCommandRunner(cmdRunner))

		if !nav.IsAvailable(t.Context()) {
			t.Error("IsAvailable() should return true with version 2.0.0")
		}
	})

	t.Run("not available with old version", func(t *testing.T) {
		cmdRunner := &mockCommandRunner{
			lookPathResult: "/usr/local/bin/claude",
			versionOutput:  "claude-code 0.9.0",
		}
		nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeCommandRunner(cmdRunner))

		if nav.IsAvailable(t.Context()) {
			t.Error("IsAvailable() should return false with version 0.9.0")
		}
	})
}

func TestClaudeCodeVisionNavigator_UnavailableReason(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("not in PATH", func(t *testing.T) {
		cmdRunner := &mockCommandRunner{
			lookPathErr: exec.ErrNotFound,
		}
		nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeCommandRunner(cmdRunner))

		reason := nav.UnavailableReason(t.Context())
		if !strings.Contains(reason, "not found") {
			t.Errorf("UnavailableReason() = %q, expected to contain 'not found'", reason)
		}
	})

	t.Run("old version", func(t *testing.T) {
		cmdRunner := &mockCommandRunner{
			lookPathResult: "/usr/local/bin/claude",
			versionOutput:  "claude-code 0.9.0",
		}
		nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeCommandRunner(cmdRunner))

		reason := nav.UnavailableReason(t.Context())
		if !strings.Contains(reason, "too old") {
			t.Errorf("UnavailableReason() = %q, expected to contain 'too old'", reason)
		}
	})

	t.Run("empty when available", func(t *testing.T) {
		cmdRunner := &mockCommandRunner{
			lookPathResult: "/usr/local/bin/claude",
			versionOutput:  "claude-code 1.0.18",
		}
		nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeCommandRunner(cmdRunner))

		reason := nav.UnavailableReason(t.Context())
		if reason != "" {
			t.Errorf("UnavailableReason() = %q, want empty string", reason)
		}
	})
}

func TestParseClaudeVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"1.0.18", "1.0.18", false},
		{"v1.0.18", "1.0.18", false},
		{"claude-code 1.0.18", "1.0.18", false},
		{"claude 2.0.0", "2.0.0", false},
		{"1.0.18 (build 123)", "1.0.18", false},
		{"claude-code 1.0.18-beta", "1.0.18-beta", false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseClaudeVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseClaudeVersion(%q) error = nil, want error", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseClaudeVersion(%q) error = %v", tt.input, err)
				return
			}
			if got.String() != tt.want {
				t.Errorf("parseClaudeVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestClaudeCodeVisionNavigator_Navigate_NotAvailable(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	cmdRunner := &mockCommandRunner{
		lookPathErr: exec.ErrNotFound,
	}
	nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeCommandRunner(cmdRunner))

	_, err := nav.Navigate(t.Context(), NavigationRequest{
		SessionID: "session123",
		Prompt:    "Click button",
	})

	if err == nil {
		t.Error("Navigate() should return error when not available")
	}
	if !errors.Is(err, ErrNavigatorNotAvailable) {
		t.Errorf("Navigate() error = %v, want ErrNavigatorNotAvailable", err)
	}
}

func TestClaudeCodeVisionNavigator_MapMCPToolToAction(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	tests := []struct {
		name       string
		toolName   string
		input      map[string]interface{}
		wantType   string
		wantURL    string
		wantSelect string
	}{
		{
			name:     "navigate",
			toolName: "mcp__claude-in-chrome__navigate",
			input:    map[string]interface{}{"url": "https://example.com"},
			wantType: "navigate",
			wantURL:  "https://example.com",
		},
		{
			name:       "computer left_click",
			toolName:   "mcp__claude-in-chrome__computer",
			input:      map[string]interface{}{"action": "left_click", "coordinate": []interface{}{100.0, 200.0}},
			wantType:   "click",
			wantSelect: "[100, 200]",
		},
		{
			name:     "computer screenshot",
			toolName: "mcp__claude-in-chrome__computer",
			input:    map[string]interface{}{"action": "screenshot"},
			wantType: "screenshot",
		},
		{
			name:       "form_input",
			toolName:   "mcp__claude-in-chrome__form_input",
			input:      map[string]interface{}{"ref": "ref_1", "value": "test"},
			wantType:   "type",
			wantSelect: "ref_1",
		},
		{
			name:       "find",
			toolName:   "mcp__claude-in-chrome__find",
			input:      map[string]interface{}{"query": "search button"},
			wantType:   "find",
			wantSelect: "search button",
		},
		{
			name:     "read_page",
			toolName: "mcp__claude-in-chrome__read_page",
			input:    map[string]interface{}{},
			wantType: "read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, _ := json.Marshal(tt.input)
			event := &claudeStreamEvent{
				Name:  tt.toolName,
				Input: inputJSON,
			}

			gotType, gotURL, gotSelect := nav.mapMCPToolToAction(event)

			if gotType != tt.wantType {
				t.Errorf("actionType = %q, want %q", gotType, tt.wantType)
			}
			if gotURL != tt.wantURL {
				t.Errorf("url = %q, want %q", gotURL, tt.wantURL)
			}
			if gotSelect != tt.wantSelect {
				t.Errorf("selector = %q, want %q", gotSelect, tt.wantSelect)
			}
		})
	}
}

func TestClaudeCodeVisionNavigator_NormalizeComputerAction(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	tests := []struct {
		input string
		want  string
	}{
		{"left_click", "click"},
		{"right_click", "click"},
		{"double_click", "click"},
		{"triple_click", "click"},
		{"type", "type"},
		{"screenshot", "screenshot"},
		{"scroll", "scroll"},
		{"key", "keypress"},
		{"left_click_drag", "drag"},
		{"hover", "hover"},
		{"wait", "wait"},
		{"zoom", "zoom"},
		{"scroll_to", "scroll"},
		{"unknown_action", "unknown_action"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := nav.normalizeComputerAction(tt.input)
			if got != tt.want {
				t.Errorf("normalizeComputerAction(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClaudeCodeVisionNavigator_ExtractReasoning(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	t.Run("string content", func(t *testing.T) {
		content := json.RawMessage(`"This is my reasoning"`)
		got := nav.extractReasoning(content)
		if got != "This is my reasoning" {
			t.Errorf("extractReasoning() = %q, want %q", got, "This is my reasoning")
		}
	})

	t.Run("array of text blocks", func(t *testing.T) {
		content := json.RawMessage(`[{"type": "text", "text": "First part"}, {"type": "text", "text": "Second part"}]`)
		got := nav.extractReasoning(content)
		if got != "First part Second part" {
			t.Errorf("extractReasoning() = %q, want %q", got, "First part Second part")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		content := json.RawMessage(``)
		got := nav.extractReasoning(content)
		if got != "" {
			t.Errorf("extractReasoning() = %q, want empty string", got)
		}
	})
}

func TestClaudeCodeVisionNavigator_ResumeNotSupported(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	err := nav.ResumeNavigation(t.Context(), "nav_test")
	if err == nil {
		t.Error("ResumeNavigation() should return error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("ResumeNavigation() error = %v, expected to contain 'not supported'", err)
	}
}

func TestClaudeCodeVisionNavigator_AbortNotFound(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	err := nav.AbortNavigation(t.Context(), "nav_notfound")
	if err == nil {
		t.Error("AbortNavigation() should return error for non-existent navigation")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("AbortNavigation() error = %v, expected to contain 'not found'", err)
	}
}

func TestClaudeCodeVisionNavigator_GetSession(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	// Add a session
	session := &claudeCodeSession{
		NavigationSession: &NavigationSession{
			NavigationID: "nav_test",
			SessionID:    "session123",
		},
		doneChan: make(chan struct{}),
	}
	nav.mu.Lock()
	nav.activeNavigations["nav_test"] = session
	nav.mu.Unlock()

	t.Run("existing session", func(t *testing.T) {
		s, exists := nav.GetSession("nav_test")
		if !exists {
			t.Error("expected session to exist")
		}
		if s.SessionID != "session123" {
			t.Errorf("SessionID = %q, want %q", s.SessionID, "session123")
		}
	})

	t.Run("non-existent session", func(t *testing.T) {
		_, exists := nav.GetSession("nav_notfound")
		if exists {
			t.Error("expected session to not exist")
		}
	})
}

func TestClaudeCodeNavigationHandle(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	session := &claudeCodeSession{
		NavigationSession: &NavigationSession{
			NavigationID: "nav_handle",
			SessionID:    "session123",
			Status:       StatusNavigating,
		},
		doneChan: make(chan struct{}),
	}
	nav.mu.Lock()
	nav.activeNavigations["nav_handle"] = session
	nav.mu.Unlock()

	handle := &claudeCodeNavigationHandle{
		navigator: nav,
		session:   session,
	}

	t.Run("ID returns navigation ID", func(t *testing.T) {
		if handle.ID() != "nav_handle" {
			t.Errorf("ID() = %q, want %q", handle.ID(), "nav_handle")
		}
	})

	t.Run("SessionID returns session ID", func(t *testing.T) {
		if handle.SessionID() != "session123" {
			t.Errorf("SessionID() = %q, want %q", handle.SessionID(), "session123")
		}
	})

	t.Run("Status returns current status", func(t *testing.T) {
		if handle.Status() != StatusNavigating {
			t.Errorf("Status() = %v, want %v", handle.Status(), StatusNavigating)
		}
	})

	t.Run("Status returns Completed for cleaned up session", func(t *testing.T) {
		nav.mu.Lock()
		delete(nav.activeNavigations, "nav_handle")
		nav.mu.Unlock()

		if handle.Status() != StatusCompleted {
			t.Errorf("Status() = %v, want %v after cleanup", handle.Status(), StatusCompleted)
		}

		// Restore for other tests
		nav.mu.Lock()
		nav.activeNavigations["nav_handle"] = session
		nav.mu.Unlock()
	})

	t.Run("Wait returns on context cancel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := handle.Wait(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Wait() error = %v, want context.DeadlineExceeded", err)
		}
	})

	t.Run("Wait returns on done channel", func(t *testing.T) {
		// Create a new session with a done channel we can close
		testSession := &claudeCodeSession{
			NavigationSession: &NavigationSession{
				NavigationID: "nav_wait_test",
				SessionID:    "session_wait",
				Status:       StatusNavigating,
			},
			doneChan: make(chan struct{}),
		}
		nav.mu.Lock()
		nav.activeNavigations["nav_wait_test"] = testSession
		nav.mu.Unlock()

		testHandle := &claudeCodeNavigationHandle{
			navigator: nav,
			session:   testSession,
		}

		// Close done channel in background
		go func() {
			time.Sleep(10 * time.Millisecond)
			close(testSession.doneChan)
		}()

		err := testHandle.Wait(context.Background())
		if err != nil {
			t.Errorf("Wait() error = %v, want nil", err)
		}
	})

	t.Run("Resume returns not supported error", func(t *testing.T) {
		err := handle.Resume(t.Context())
		if err == nil {
			t.Error("Resume() should return error")
		}
	})
}

func TestClaudeCodeVisionNavigator_ReportActionToRecording(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	var recordedActions []*RecordedNavigationAction
	callback := func(sessionID string, action *RecordedNavigationAction) {
		recordedActions = append(recordedActions, action)
	}

	nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeActionRecordCallback(callback))

	session := &claudeCodeSession{
		NavigationSession: &NavigationSession{
			NavigationID: "nav_record",
			SessionID:    "session_record",
		},
	}

	inputJSON, _ := json.Marshal(map[string]interface{}{
		"action":     "left_click",
		"coordinate": []interface{}{100.0, 200.0},
	})
	event := &claudeStreamEvent{
		Name:  "mcp__claude-in-chrome__computer",
		Input: inputJSON,
	}

	nav.reportActionToRecording(session, event, 1, "Clicking the button")

	if len(recordedActions) != 1 {
		t.Fatalf("expected 1 recorded action, got %d", len(recordedActions))
	}

	action := recordedActions[0]
	if action.ActionType != "click" {
		t.Errorf("ActionType = %q, want %q", action.ActionType, "click")
	}
	if action.StepNumber != 1 {
		t.Errorf("StepNumber = %d, want 1", action.StepNumber)
	}
	if action.Reasoning != "Clicking the button" {
		t.Errorf("Reasoning = %q, want %q", action.Reasoning, "Clicking the button")
	}
	if action.Source != "ai" {
		t.Errorf("Source = %q, want %q", action.Source, "ai")
	}
}

func TestClaudeCodeVisionNavigator_BuildPrompt(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	req := NavigationRequest{
		Prompt:    "Find the login button and click it",
		SessionID: "session123",
	}

	prompt := nav.buildPrompt(req, 20)

	if !strings.Contains(prompt, "Find the login button and click it") {
		t.Error("prompt should contain the user's goal")
	}
	if !strings.Contains(prompt, "Maximum steps: 20") {
		t.Error("prompt should contain max steps")
	}
	if !strings.Contains(prompt, "mcp__claude-in-chrome__") {
		t.Error("prompt should mention MCP tools")
	}
}

func TestClaudeCodeVisionNavigator_ParseOutput(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	var recordedActions []*RecordedNavigationAction
	callback := func(sessionID string, action *RecordedNavigationAction) {
		recordedActions = append(recordedActions, action)
	}

	wsHub := &mockWSHub{}
	nav := NewClaudeCodeVisionNavigator(log,
		WithClaudeCodeActionRecordCallback(callback),
		WithClaudeCodeHub(wsHub),
	)

	session := &claudeCodeSession{
		NavigationSession: &NavigationSession{
			NavigationID: "nav_parse",
			SessionID:    "session_parse",
			Status:       StatusNavigating,
			StartedAt:    time.Now(),
		},
		doneChan: make(chan struct{}),
	}

	// Simulate stream-json output
	streamOutput := `{"type":"assistant","content":[{"type":"text","text":"I will navigate to the page"}]}
{"type":"tool_use","name":"mcp__claude-in-chrome__navigate","input":{"url":"https://example.com"}}
{"type":"tool_result","content":"navigated"}
{"type":"result","result":"completed"}
`

	reader := bytes.NewReader([]byte(streamOutput))
	nav.parseOutput(session, reader)

	// Verify action was recorded
	if len(recordedActions) != 1 {
		t.Errorf("expected 1 recorded action, got %d", len(recordedActions))
	}

	if len(recordedActions) > 0 {
		action := recordedActions[0]
		if action.ActionType != "navigate" {
			t.Errorf("ActionType = %q, want %q", action.ActionType, "navigate")
		}
		if action.URL != "https://example.com" {
			t.Errorf("URL = %q, want %q", action.URL, "https://example.com")
		}
	}

	// Verify WebSocket broadcasts
	if wsHub.broadcastCount < 1 {
		t.Error("expected at least one WebSocket broadcast")
	}

	// Verify session status updated
	session.mu.Lock()
	status := session.Status
	session.mu.Unlock()
	if status != StatusCompleted {
		t.Errorf("Status = %v, want %v", status, StatusCompleted)
	}
}

func TestClaudeCodeVisionNavigator_SetActionRecordCallback(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	if nav.onActionRecord != nil {
		t.Error("onActionRecord should be nil initially")
	}

	callback := func(sessionID string, action *RecordedNavigationAction) {}
	nav.SetActionRecordCallback(callback)

	if nav.onActionRecord == nil {
		t.Error("onActionRecord should be set after SetActionRecordCallback")
	}
}
