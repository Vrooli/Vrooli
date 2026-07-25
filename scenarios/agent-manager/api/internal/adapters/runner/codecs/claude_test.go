package codecs

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// Test helpers
// =============================================================================

// decodeOne is a thin shim over Claude.DecodeStreamLine that allocates a
// fresh state each call. Convenient for tests that don't care about
// state continuity across lines.
func decodeOne(t *testing.T, c *Claude, line string) []*domain.RunEvent {
	t.Helper()
	state := c.NewState()
	events, err := c.DecodeStreamLine(state, uuid.New(), line)
	if err != nil {
		t.Fatalf("DecodeStreamLine error: %v", err)
	}
	return events
}

// claudeCodeSamples holds real Claude Code v2.0.70 stream-json output
// captured 2025-12-19 (verify periodically as the format evolves).
var claudeCodeSamples = map[string]string{
	"system_init":             `{"type":"system","subtype":"init","cwd":"/home/user/project","session_id":"751b5a53-bc44-4484-943d-8851ccfdfda1","tools":["Task","Bash","Read","Write"],"mcp_servers":[],"model":"claude-opus-4-5-20251101","permissionMode":"bypassPermissions","claude_code_version":"2.0.70","uuid":"70d9d8c9-20ae-40b4-bfcf-48c30cbf7dc5"}`,
	"assistant_text":          `{"type":"assistant","message":{"model":"claude-opus-4-5-20251101","id":"msg_012wEys6PjZFVz3rJyuY8o4g","type":"message","role":"assistant","content":[{"type":"text","text":"Hello! I'm ready to help you."}],"stop_reason":null,"usage":{"input_tokens":2,"cache_creation_input_tokens":7192,"cache_read_input_tokens":14135,"output_tokens":5,"service_tier":"standard"}},"session_id":"751b5a53-bc44-4484-943d-8851ccfdfda1"}`,
	"assistant_tool_use":      `{"type":"assistant","message":{"model":"claude-opus-4-5-20251101","id":"msg_015m1Y7p5a3BZwxStfp7HQbK","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_01KKEKfCADiJJApRkJF86t8R","name":"Write","input":{"file_path":"/tmp/test.txt","content":"hello"}}],"stop_reason":null,"usage":{"input_tokens":2,"output_tokens":5}},"session_id":"fa4d44b2-1aa6-4509-b6d4-97611b779f04"}`,
	"assistant_text_and_tool": `{"type":"assistant","message":{"model":"claude-opus-4-5-20251101","id":"msg_mixed","type":"message","role":"assistant","content":[{"type":"text","text":"Let me create that file."},{"type":"tool_use","id":"toolu_abc123","name":"Write","input":{"file_path":"/tmp/file.txt","content":"data"}}],"stop_reason":null},"session_id":"session123"}`,
	"user_tool_result":        `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_01KKEKfCADiJJApRkJF86t8R","type":"tool_result","content":"File created successfully at: /tmp/test-sample.txt"}]},"session_id":"fa4d44b2-1aa6-4509-b6d4-97611b779f04","tool_use_result":{"type":"create","filePath":"/tmp/test-sample.txt","content":"sample content"}}`,
	"result_success":          `{"type":"result","subtype":"success","is_error":false,"duration_ms":2892,"duration_api_ms":5284,"num_turns":1,"result":"Hello! I'm ready to help.","session_id":"751b5a53-bc44-4484-943d-8851ccfdfda1","total_cost_usd":0.08424875,"usage":{"input_tokens":2,"cache_creation_input_tokens":7192,"cache_read_input_tokens":14135,"output_tokens":40,"server_tool_use":{"web_search_requests":0},"service_tier":"standard"}}`,
	"result_error":            `{"type":"result","subtype":"error","is_error":true,"duration_ms":100,"num_turns":0,"result":"Claude AI usage limit reached|1755806400","session_id":"session123","total_cost_usd":0}`,
	"error":                   `{"type":"error","error":{"code":"rate_limit","message":"Rate limit exceeded"}}`,
	"usage":                   `{"type":"usage","usage":{"input_tokens":100,"output_tokens":50}}`,
}

// diagnosticSamples — result-event shapes whose error-classification we
// pin via the formatErrorMessage / buildErrorDetails enrichment path.
var diagnosticSamples = map[string]string{
	"result_error_max_turns": `{"type":"result","subtype":"error_max_turns","is_error":true,"duration_ms":450000,"num_turns":60,"session_id":"s1","result":"","total_cost_usd":0}`,
	"result_error_empty":     `{"type":"result","subtype":"","is_error":true,"duration_ms":100,"num_turns":0,"session_id":"s2","result":"","total_cost_usd":0}`,
	"result_error_during":    `{"type":"result","subtype":"error_during_execution","is_error":true,"duration_ms":5000,"num_turns":12,"session_id":"s3","result":"internal API error","total_cost_usd":0}`,
	"system_auto_compact":    `{"type":"system","subtype":"auto-compacting","result":"Auto-compacting conversation to free tokens","session_id":"s4"}`,
}

// =============================================================================
// BuildArgs / BuildEnv / BuildPrompt
// =============================================================================

func TestClaude_BuildArgs_EnableBrowser(t *testing.T) {
	c := NewClaudeForTest()
	tests := []struct {
		name       string
		config     *domain.RunConfig
		wantChrome bool
	}{
		{"true adds --chrome", &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, Features: domain.FeatureFlags{EnableBrowser: true}}, true},
		{"false omits --chrome", &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, Features: domain.FeatureFlags{EnableBrowser: false}}, false},
		{"zero features omits --chrome", &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := runner.ExecuteRequest{RunID: uuid.New(), ResolvedConfig: tt.config}
			args := c.BuildArgs(c.NewState(), req)
			has := false
			for _, a := range args {
				if a == "--chrome" {
					has = true
					break
				}
			}
			if has != tt.wantChrome {
				t.Errorf("hasChrome=%v want=%v args=%v", has, tt.wantChrome, args)
			}
		})
	}
}

func TestClaudeBuildArgsTranslatesCanonicalAllowedTools(t *testing.T) {
	c := NewClaudeForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{RunID: uuid.New(), ResolvedConfig: &domain.RunConfig{
		RunnerType:   domain.RunnerTypeClaudeCode,
		AllowedTools: []string{"read", "shell", "web_search"},
	}})
	for index, arg := range args {
		if arg == "--allowedTools" {
			if index+1 >= len(args) || args[index+1] != "Read,Bash,WebSearch" {
				t.Fatalf("translated allowed tools = %v", args)
			}
			return
		}
	}
	t.Fatalf("--allowedTools absent from %v", args)
}

func TestClaudeBuildArgsTranslatesCanonicalDeniedTools(t *testing.T) {
	c := NewClaudeForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{RunID: uuid.New(), ResolvedConfig: &domain.RunConfig{
		RunnerType:  domain.RunnerTypeClaudeCode,
		DeniedTools: []string{"shell", "web_fetch"},
	}})
	for index, arg := range args {
		if arg == "--disallowedTools" && index+1 < len(args) && args[index+1] == "Bash,WebFetch" {
			return
		}
	}
	t.Fatalf("--disallowedTools absent or untranslated: %v", args)
}

func TestClaude_BuildArgs_ExtraFlags(t *testing.T) {
	c := NewClaudeForTest()
	tests := []struct {
		name      string
		config    *domain.RunConfig
		wantFlags []string
	}{
		{"flags appended", &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, ExtraFlags: domain.RunnerExtraFlags{domain.RunnerTypeClaudeCode: {"--verbose", "--allowedTools=Read,Write"}}}, []string{"--verbose", "--allowedTools=Read,Write"}},
		{"flags for other runner ignored", &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, ExtraFlags: domain.RunnerExtraFlags{domain.RunnerTypeCodex: {"--verbose"}}}, nil},
		{"nil extras", &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, ExtraFlags: nil}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{RunID: uuid.New(), ResolvedConfig: tt.config})
			for _, want := range tt.wantFlags {
				found := false
				for _, a := range args {
					if a == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing flag %q in %v", want, args)
				}
			}
		})
	}
}

func TestClaude_BuildArgs_FeaturesAndExtraFlags(t *testing.T) {
	c := NewClaudeForTest()
	cfg := &domain.RunConfig{
		RunnerType: domain.RunnerTypeClaudeCode,
		Features:   domain.FeatureFlags{EnableBrowser: true},
		ExtraFlags: domain.RunnerExtraFlags{domain.RunnerTypeClaudeCode: {"--verbose"}},
	}
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{RunID: uuid.New(), ResolvedConfig: cfg})
	hasChrome, hasVerbose := false, false
	for _, a := range args {
		switch a {
		case "--chrome":
			hasChrome = true
		case "--verbose":
			hasVerbose = true
		}
	}
	if !hasChrome {
		t.Errorf("expected --chrome in %v", args)
	}
	if !hasVerbose {
		t.Errorf("expected --verbose in %v", args)
	}
}

func TestClaude_BuildArgs_SystemPrompt(t *testing.T) {
	c := NewClaudeForTest()
	t.Run("includes --append-system-prompt when set", func(t *testing.T) {
		req := runner.ExecuteRequest{RunID: uuid.New(), SystemPrompt: "You are an investigation agent.", Profile: &domain.AgentProfile{RoleRef: "code.default"}}
		args := c.BuildArgs(c.NewState(), req)
		found := false
		for i, a := range args {
			if a == "--append-system-prompt" && i+1 < len(args) {
				found = true
				if args[i+1] != "You are an investigation agent." {
					t.Errorf("--append-system-prompt value=%q", args[i+1])
				}
			}
		}
		if !found {
			t.Error("expected --append-system-prompt")
		}
	})
	t.Run("omits --append-system-prompt when empty", func(t *testing.T) {
		req := runner.ExecuteRequest{RunID: uuid.New(), Profile: &domain.AgentProfile{RoleRef: "code.default"}}
		args := c.BuildArgs(c.NewState(), req)
		for _, a := range args {
			if a == "--append-system-prompt" {
				t.Error("unexpected --append-system-prompt for empty system prompt")
			}
		}
	})
}

func TestClaude_BuildArgs_DefaultMaxTurnsAndStdinSentinel(t *testing.T) {
	c := NewClaudeForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{RunID: uuid.New(), Profile: &domain.AgentProfile{RoleRef: "code.default"}})
	// Default max-turns should be 30 when no override.
	gotMaxTurns := ""
	for i, a := range args {
		if a == "--max-turns" && i+1 < len(args) {
			gotMaxTurns = args[i+1]
		}
	}
	if gotMaxTurns != "30" {
		t.Errorf("default --max-turns=%s want 30", gotMaxTurns)
	}
	if args[len(args)-1] != "-" {
		t.Errorf("expected stdin-sentinel '-' as last arg, got %q", args[len(args)-1])
	}
}

func TestClaude_BuildContinueArgs(t *testing.T) {
	c := NewClaudeForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{RunID: uuid.New(), SessionID: "sess-xyz"})
	hasResume, sessionMatch := false, false
	for i, a := range args {
		if a == "--resume" {
			hasResume = true
			if i+1 < len(args) && args[i+1] == "sess-xyz" {
				sessionMatch = true
			}
		}
	}
	if !hasResume || !sessionMatch {
		t.Errorf("expected --resume sess-xyz in %v", args)
	}
}

func TestClaudeBuildContinueArgsKeepsCanonicalToolRestriction(t *testing.T) {
	c := NewClaudeForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{RunID: uuid.New(), SessionID: "sess-xyz", ResolvedConfig: &domain.RunConfig{AllowedTools: []string{"read", "shell"}}})
	for index, arg := range args {
		if arg == "--allowedTools" && index+1 < len(args) && args[index+1] == "Read,Bash" {
			return
		}
	}
	t.Fatalf("continuation omitted translated restriction: %v", args)
}

func TestClaudeBuildContinueArgsKeepsCanonicalDeniedTools(t *testing.T) {
	c := NewClaudeForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{RunID: uuid.New(), SessionID: "sess-xyz", ResolvedConfig: &domain.RunConfig{DeniedTools: []string{"shell"}}})
	for index, arg := range args {
		if arg == "--disallowedTools" && index+1 < len(args) && args[index+1] == "Bash" {
			return
		}
	}
	t.Fatalf("continuation omitted translated denied restriction: %v", args)
}

func TestClaude_BuildEnv_AgentTag(t *testing.T) {
	c := NewClaudeForTest()
	t.Run("tag present", func(t *testing.T) {
		env := c.BuildEnv("heartbeat-team1-agent1-2026-01-01T00-00-00Z", nil)
		found := false
		for _, e := range env {
			if strings.HasPrefix(e, "CLAUDE_CODE_AGENT_TAG=") {
				found = true
				if strings.TrimPrefix(e, "CLAUDE_CODE_AGENT_TAG=") != "heartbeat-team1-agent1-2026-01-01T00-00-00Z" {
					t.Errorf("unexpected tag value: %s", e)
				}
			}
		}
		if !found {
			t.Error("expected CLAUDE_CODE_AGENT_TAG in env")
		}
	})
}

func TestClaude_Capabilities_SupportedFeatures(t *testing.T) {
	c := NewClaudeForTest()
	caps := c.Capabilities()
	found := false
	for _, f := range caps.SupportedFeatures {
		if f == "EnableBrowser" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected EnableBrowser in %v", caps.SupportedFeatures)
	}
}

func TestClaude_Capabilities_DeniedToolsIsFirstClass(t *testing.T) {
	c := NewClaudeForTest()
	caps := c.Capabilities()
	found := false
	for _, f := range caps.AllowedExtraFlags {
		if f == "--disallowedTools" {
			found = true
		}
	}
	if found {
		t.Errorf("--disallowedTools must not be an extra flag: %v", caps.AllowedExtraFlags)
	}
}

func TestClaude_BuildPrompt_NoAttachments(t *testing.T) {
	c := NewClaudeForTest()
	prompt := "Fix the bug in main.go"
	if got := c.BuildPrompt(prompt, nil); got != prompt {
		t.Errorf("got %q want %q", got, prompt)
	}
	if got := c.BuildPrompt(prompt, []runner.Attachment{}); got != prompt {
		t.Errorf("empty slice: got %q want %q", got, prompt)
	}
}

func TestClaude_BuildPrompt_SingleAttachment(t *testing.T) {
	c := NewClaudeForTest()
	got := c.BuildPrompt("Describe this screenshot", []runner.Attachment{
		{ID: "att-1", FileName: "screen.png", FilePath: "/tmp/uploads/screen.png"},
	})
	want := "/tmp/uploads/screen.png\n\nDescribe this screenshot"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestClaude_BuildPrompt_MultipleAttachments(t *testing.T) {
	c := NewClaudeForTest()
	got := c.BuildPrompt("Compare these images", []runner.Attachment{
		{ID: "att-1", FileName: "before.png", FilePath: "/tmp/uploads/before.png"},
		{ID: "att-2", FileName: "after.png", FilePath: "/tmp/uploads/after.png"},
		{ID: "att-3", FileName: "diff.jpg", FilePath: "/data/diff.jpg"},
	})
	want := "/tmp/uploads/before.png\n/tmp/uploads/after.png\n/data/diff.jpg\n\nCompare these images"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestClaude_BuildPrompt_EmptyPrompt(t *testing.T) {
	c := NewClaudeForTest()
	got := c.BuildPrompt("", []runner.Attachment{
		{ID: "att-1", FileName: "image.png", FilePath: "/tmp/image.png"},
	})
	want := "/tmp/image.png\n\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// =============================================================================
// Stream parsing
// =============================================================================

func TestClaude_DecodeStreamLine_SystemInit(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, claudeCodeSamples["system_init"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	logData, ok := events[0].Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("expected LogEventData, got %T", events[0].Data)
	}
	if logData.Level != "debug" {
		t.Errorf("level=%s want debug", logData.Level)
	}
	if logData.Message != "System context received" {
		t.Errorf("message=%q", logData.Message)
	}
}

func TestClaude_DecodeStreamLine_AssistantText(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, claudeCodeSamples["assistant_text"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	if events[0].EventType != domain.EventTypeMessage {
		t.Errorf("type=%s want message", events[0].EventType)
	}
	msg := events[0].Data.(*domain.MessageEventData)
	if msg.Role != "assistant" {
		t.Errorf("role=%s", msg.Role)
	}
	if msg.Content != "Hello! I'm ready to help you." {
		t.Errorf("content=%q", msg.Content)
	}
}

func TestClaude_DecodeStreamLine_AssistantToolUse(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, claudeCodeSamples["assistant_tool_use"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	if events[0].EventType != domain.EventTypeToolCall {
		t.Errorf("type=%s want tool_call", events[0].EventType)
	}
	tool := events[0].Data.(*domain.ToolCallEventData)
	if tool.ToolName != "Write" {
		t.Errorf("tool=%s", tool.ToolName)
	}
	if fp, ok := tool.Input["file_path"].(string); !ok || fp != "/tmp/test.txt" {
		t.Errorf("file_path=%v", tool.Input["file_path"])
	}
}

func TestClaude_DecodeStreamLine_AssistantTextAndTool(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, claudeCodeSamples["assistant_text_and_tool"])
	if len(events) < 2 {
		t.Fatalf("got %d events, want >=2", len(events))
	}
	// First event should be the text message
	msg := events[0].Data.(*domain.MessageEventData)
	if msg.Content != "Let me create that file." {
		t.Errorf("first event content=%q", msg.Content)
	}
}

func TestClaude_DecodeStreamLine_UserToolResult(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, claudeCodeSamples["user_tool_result"])
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	if events[0].EventType != domain.EventTypeToolResult {
		t.Errorf("type=%s want tool_result", events[0].EventType)
	}
	r := events[0].Data.(*domain.ToolResultEventData)
	if r.Output != "File created successfully at: /tmp/test-sample.txt" {
		t.Errorf("output=%q", r.Output)
	}
}

func TestClaude_DecodeStreamLine_UserToolResultError(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_error","type":"tool_result","content":"Exit code 1\nboom","is_error":true}]}}`)
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	r := events[0].Data.(*domain.ToolResultEventData)
	if r.Success {
		t.Fatal("tool_result Success = true, want false for is_error content block")
	}
	if r.Error != "Exit code 1\nboom" {
		t.Fatalf("tool_result Error = %q", r.Error)
	}
}

func TestClaude_DecodeStreamLine_ResultSuccess(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, claudeCodeSamples["result_success"])
	// Should produce at least a cost event; success result with non-empty
	// result text + lastAssistant=="" emits a synthetic assistant message.
	hasCost := false
	for _, e := range events {
		if _, ok := e.Data.(*domain.CostEventData); ok {
			hasCost = true
		}
	}
	if !hasCost {
		t.Errorf("expected CostEventData in %d events", len(events))
	}
}

func TestClaude_DecodeStreamLine_ResultError_RateLimit(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	events, err := c.DecodeStreamLine(state, uuid.New(), claudeCodeSamples["result_error"])
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	hasRL := false
	for _, e := range events {
		if _, ok := e.Data.(*domain.RateLimitEventData); ok {
			hasRL = true
		}
	}
	if !hasRL {
		t.Error("expected RateLimitEventData")
	}
	// Codec captures rate-limit on state for PostClassify to flip later.
	cs := state.(*claudeState)
	if cs.rateLimit == nil {
		t.Error("expected state.rateLimit to be captured")
	}
}

func TestClaude_DecodeStreamLine_Error(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, claudeCodeSamples["error"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	errData := events[0].Data.(*domain.ErrorEventData)
	if errData.Code != "rate_limit" {
		t.Errorf("code=%s", errData.Code)
	}
	if errData.Message != "Rate limit exceeded" {
		t.Errorf("message=%s", errData.Message)
	}
}

func TestClaude_DecodeStreamLine_Usage(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, claudeCodeSamples["usage"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	if _, ok := events[0].Data.(*domain.MetricEventData); !ok {
		t.Errorf("expected MetricEventData, got %T", events[0].Data)
	}
}

func TestClaude_DecodeStreamLine_NonJsonLines(t *testing.T) {
	c := NewClaudeForTest()
	cases := []string{
		"Initializing claude code...",
		"[Info] Starting up",
		"[INFO] Ready",
		"plain text without JSON",
	}
	for _, line := range cases {
		events := decodeOne(t, c, line)
		if len(events) != 0 {
			t.Errorf("non-JSON line %q produced events: %v", line, events)
		}
	}
}

func TestClaude_DecodeStreamLine_MalformedJson(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"unterminated`)
	if len(events) != 0 {
		t.Errorf("malformed JSON should be skipped silently, got %v", events)
	}
}

func TestClaude_DecodeStreamLine_SilentlySkipped(t *testing.T) {
	c := NewClaudeForTest()
	cases := []string{"", "   ", "\t\n", `{"type":"init"}`, `{"type":"start"}`, `{"type":"ping"}`, `{"type":"heartbeat"}`}
	for _, line := range cases {
		events := decodeOne(t, c, line)
		if len(events) != 0 {
			t.Errorf("line %q should be silent, got %v", line, events)
		}
	}
}

func TestClaude_DecodeStreamLine_MessageDelta_LogsWithoutText(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	events, _ := c.DecodeStreamLine(state, uuid.New(), `{"type":"message_delta","delta":{"type":"text_delta","text":""}}`)
	if len(events) == 0 {
		t.Fatal("expected debug log event")
	}
	logData, ok := events[0].Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("expected LogEventData, got %T", events[0].Data)
	}
	if logData.Level != "debug" {
		t.Errorf("level=%s", logData.Level)
	}
}

func TestClaude_DecodeStreamLine_MessageDelta_TextBuffer(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	runID := uuid.New()
	for _, chunk := range []string{"Hel", "lo, ", "world!"} {
		line := `{"type":"message_delta","delta":{"type":"text_delta","text":"` + chunk + `"}}`
		events, _ := c.DecodeStreamLine(state, runID, line)
		if len(events) != 0 {
			t.Errorf("expected 0 events for partial delta, got %d", len(events))
		}
	}
	// message_stop flushes the buffer.
	events, _ := c.DecodeStreamLine(state, runID, `{"type":"message_stop"}`)
	if len(events) != 1 {
		t.Fatalf("expected 1 event from flush, got %d", len(events))
	}
	msg := events[0].Data.(*domain.MessageEventData)
	if msg.Content != "Hello, world!" {
		t.Errorf("flushed content=%q", msg.Content)
	}
}

func TestClaude_DecodeStreamLine_ContentBlockStart_ToolUse(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	events, _ := c.DecodeStreamLine(state, uuid.New(), `{"type":"content_block_start","content_block":{"type":"tool_use","id":"toolu_1","name":"Bash"}}`)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
	cs := state.(*claudeState)
	if !cs.toolUseActive {
		t.Error("expected toolUseActive=true")
	}
	if cs.toolUseName != "Bash" {
		t.Errorf("toolUseName=%s", cs.toolUseName)
	}
}

func TestClaude_DecodeStreamLine_ContentBlockDelta_TextAndJSON(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	runID := uuid.New()
	// Start tool use
	_, _ = c.DecodeStreamLine(state, runID, `{"type":"content_block_start","content_block":{"type":"tool_use","id":"t1","name":"Bash"}}`)
	_, _ = c.DecodeStreamLine(state, runID, `{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{\"command\":\"ls\"}"}}`)
	events, _ := c.DecodeStreamLine(state, runID, `{"type":"content_block_stop"}`)
	if len(events) != 1 {
		t.Fatalf("expected 1 tool_call event, got %d", len(events))
	}
	tc := events[0].Data.(*domain.ToolCallEventData)
	if tc.ToolName != "Bash" {
		t.Errorf("tool=%s", tc.ToolName)
	}
	if cmd, ok := tc.Input["command"].(string); !ok || cmd != "ls" {
		t.Errorf("command=%v", tc.Input["command"])
	}
}

func TestClaude_DecodeStreamLine_MessageStop_FlushesText(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	runID := uuid.New()
	_, _ = c.DecodeStreamLine(state, runID, `{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`)
	events, _ := c.DecodeStreamLine(state, runID, `{"type":"message_stop"}`)
	if len(events) == 0 {
		t.Fatal("expected flush event")
	}
	if events[0].EventType != domain.EventTypeMessage {
		t.Errorf("type=%s want message", events[0].EventType)
	}
}

func TestClaude_DecodeStreamLine_ContinuesAfterResult(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	runID := uuid.New()
	// Process a result event first
	_, _ = c.DecodeStreamLine(state, runID, claudeCodeSamples["result_success"])
	// Then another assistant message — we should still parse it.
	events, _ := c.DecodeStreamLine(state, runID, claudeCodeSamples["assistant_text"])
	if len(events) == 0 {
		t.Error("expected events after result")
	}
}

// =============================================================================
// UpdateMetrics
// =============================================================================

func TestClaude_UpdateMetrics_MessageEvent(t *testing.T) {
	c := NewClaudeForTest()
	metrics := runner.ExecutionMetrics{}
	last := ""
	c.UpdateMetrics(domain.NewMessageEvent(uuid.New(), "assistant", "hello"), &metrics, &last)
	if metrics.TurnsUsed != 1 {
		t.Errorf("TurnsUsed=%d want 1", metrics.TurnsUsed)
	}
	if last != "hello" {
		t.Errorf("last=%q", last)
	}
}

func TestClaude_UpdateMetrics_ToolCallEvent(t *testing.T) {
	c := NewClaudeForTest()
	metrics := runner.ExecutionMetrics{}
	last := ""
	c.UpdateMetrics(domain.NewToolCallEvent(uuid.New(), "Bash", "t1", nil), &metrics, &last)
	if metrics.ToolCallCount != 1 {
		t.Errorf("ToolCallCount=%d want 1", metrics.ToolCallCount)
	}
}

func TestClaude_UpdateMetrics_CostEvent(t *testing.T) {
	c := NewClaudeForTest()
	metrics := runner.ExecutionMetrics{}
	last := ""
	costEvent := &domain.RunEvent{
		ID:        uuid.New(),
		EventType: domain.EventTypeMetric,
		Data: &domain.CostEventData{
			InputTokens: 100, OutputTokens: 50,
			CacheCreationTokens: 10, CacheReadTokens: 20,
			TotalCostUSD: 0.05,
		},
	}
	c.UpdateMetrics(costEvent, &metrics, &last)
	if metrics.TokensInput != 100 || metrics.TokensOutput != 50 {
		t.Errorf("tokens in/out = %d/%d", metrics.TokensInput, metrics.TokensOutput)
	}
	if metrics.CostEstimateUSD != 0.05 {
		t.Errorf("cost=%v", metrics.CostEstimateUSD)
	}
}

func TestClaude_UpdateMetrics_NilEvent(t *testing.T) {
	c := NewClaudeForTest()
	metrics := runner.ExecutionMetrics{}
	last := ""
	c.UpdateMetrics(nil, &metrics, &last) // must not panic
}

// =============================================================================
// PostClassify (rate-limit flip)
// =============================================================================

func TestClaude_PostClassify_FlipsOnRateLimit(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState().(*claudeState)
	state.rateLimit = &domain.RateLimitEventData{Message: "Claude AI usage limit reached"}
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Summary:  &domain.RunSummary{Description: "ok"},
	}
	c.PostClassify(state, result)
	if result.Success {
		t.Error("expected Success=false after rate-limit")
	}
	if result.ExitCode != 429 {
		t.Errorf("exitCode=%d want 429", result.ExitCode)
	}
	if result.Summary != nil {
		t.Error("expected Summary=nil after flip")
	}
	if !strings.Contains(result.ErrorMessage, "limit") {
		t.Errorf("errorMessage=%q", result.ErrorMessage)
	}
}

func TestClaude_PostClassify_NoOpWhenNoRateLimit(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	result := &runner.ExecuteResult{Success: true, ExitCode: 0}
	c.PostClassify(state, result)
	if !result.Success || result.ExitCode != 0 {
		t.Errorf("PostClassify should be a no-op without rate-limit, got %+v", result)
	}
}

func TestClaude_ClassifyTerminalError(t *testing.T) {
	c := NewClaudeForTest()
	cases := []struct {
		name     string
		stderr   string
		wantCode domain.ErrorCode // empty = expect nil
	}{
		{"session-expired bare", "session not found", domain.ErrCodeRunnerSessionExpired},
		{"session-expired prefixed", "the session abc was not found", domain.ErrCodeRunnerSessionExpired},
		{"unrelated error", "some other error", ""},
		{"session-invalid is not classified", "session abc is invalid", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.ClassifyTerminalError(tc.stderr, 1)
			if tc.wantCode == "" {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %s, got nil", tc.wantCode)
			}
			if got.Code() != tc.wantCode {
				t.Errorf("Code() = %s, want %s", got.Code(), tc.wantCode)
			}
		})
	}
}

// =============================================================================
// Rate-limit detection (detectClaudeRateLimit)
// =============================================================================

func TestDetectClaudeRateLimit_UsageLimitReached(t *testing.T) {
	rl := detectClaudeRateLimit("Claude AI usage limit reached|1755806400")
	if !rl.Detected {
		t.Error("expected Detected")
	}
	if rl.LimitType != "5_hour" {
		t.Errorf("limitType=%s want 5_hour", rl.LimitType)
	}
}

func TestDetectClaudeRateLimit_DailyLimit(t *testing.T) {
	rl := detectClaudeRateLimit("You've reached your daily limit")
	if !rl.Detected {
		t.Error("expected Detected")
	}
	if rl.LimitType != "daily" {
		t.Errorf("limitType=%s want daily", rl.LimitType)
	}
}

func TestDetectClaudeRateLimit_HitYourLimit(t *testing.T) {
	rl := detectClaudeRateLimit("You hit your weekly limit")
	if !rl.Detected {
		t.Error("expected Detected")
	}
}

func TestDetectClaudeRateLimit_NoLimit(t *testing.T) {
	rl := detectClaudeRateLimit("Everything is fine")
	if rl.Detected {
		t.Error("did not expect Detected")
	}
}

func TestDetectClaudeRateLimit_DocumentedPhrases(t *testing.T) {
	phrases := []string{
		"usage limit reached",
		"rate limit reached",
		"rate limit exceeded",
		"request rejected (429)",
		"server is temporarily limiting requests",
	}
	for _, p := range phrases {
		if !detectClaudeRateLimit(p).Detected {
			t.Errorf("expected Detected for %q", p)
		}
	}
}

func TestDetectClaudeRateLimit_LengthCap(t *testing.T) {
	long := strings.Repeat("rate limit reached ", 50) // > cap
	rl := detectClaudeRateLimit(long)
	if rl.Detected {
		t.Errorf("oversized message should not be classified, got Detected=%v", rl.Detected)
	}
}

func TestDetectClaudeRateLimit_BareRateLimitIgnored(t *testing.T) {
	rl := detectClaudeRateLimit("we discussed rate limit handling")
	if rl.Detected {
		t.Error("bare 'rate limit' phrase should NOT be classified")
	}
}

// =============================================================================
// parseClaudeResultEvent — error enrichment
// =============================================================================

func TestParseClaudeResultEvent_ErrorMaxTurns(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	events, err := c.DecodeStreamLine(state, uuid.New(), diagnosticSamples["result_error_max_turns"])
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	errEvent := findErrorEvent(events)
	if errEvent == nil {
		t.Fatal("expected error event")
	}
	errData := errEvent.Data.(*domain.ErrorEventData)
	if errData.Code != "execution_error" {
		t.Errorf("code=%s", errData.Code)
	}
	if !strings.Contains(errData.Message, "subtype=error_max_turns") {
		t.Errorf("missing subtype, got %q", errData.Message)
	}
	if !strings.Contains(errData.Message, "turns=60") || !strings.Contains(errData.Message, "duration_ms=450000") {
		t.Errorf("missing counters, got %q", errData.Message)
	}
}

func TestParseClaudeResultEvent_ErrorEmpty(t *testing.T) {
	c := NewClaudeForTest()
	events, _ := c.DecodeStreamLine(c.NewState(), uuid.New(), diagnosticSamples["result_error_empty"])
	errEvent := findErrorEvent(events)
	if errEvent == nil {
		t.Fatal("expected error event")
	}
	errData := errEvent.Data.(*domain.ErrorEventData)
	if strings.TrimSpace(errData.Message) == "" {
		t.Fatal("message must not be empty")
	}
	if !strings.Contains(errData.Message, "turns=0") || !strings.Contains(errData.Message, "duration_ms=100") {
		t.Errorf("missing counters: %q", errData.Message)
	}
}

func TestParseClaudeResultEvent_ErrorDuringExecution_CarriesResultText(t *testing.T) {
	c := NewClaudeForTest()
	events, _ := c.DecodeStreamLine(c.NewState(), uuid.New(), diagnosticSamples["result_error_during"])
	errEvent := findErrorEvent(events)
	if errEvent == nil {
		t.Fatal("expected error event")
	}
	errData := errEvent.Data.(*domain.ErrorEventData)
	if !strings.Contains(errData.Message, "internal API error") {
		t.Errorf("expected result_text in message: %q", errData.Message)
	}
	if errData.Details["result_text"] != "internal API error" {
		t.Errorf("result_text in details=%v", errData.Details["result_text"])
	}
}

func TestParseClaudeResultEvent_RateLimitPrecedence(t *testing.T) {
	c := NewClaudeForTest()
	events, _ := c.DecodeStreamLine(c.NewState(), uuid.New(), claudeCodeSamples["result_error"])
	for _, e := range events {
		if _, ok := e.Data.(*domain.ErrorEventData); ok {
			t.Errorf("rate-limit shape should not produce ErrorEventData: %+v", e.Data)
		}
	}
	hasRL := false
	for _, e := range events {
		if _, ok := e.Data.(*domain.RateLimitEventData); ok {
			hasRL = true
		}
	}
	if !hasRL {
		t.Error("expected RateLimitEventData")
	}
}

// =============================================================================
// Compaction helpers
// =============================================================================

func TestParseCompactCommand(t *testing.T) {
	tests := []struct {
		input     string
		isCompact bool
		focus     string
	}{
		{"/compact", true, ""},
		{"/compact focus on auth", true, "auth"},
		{"/compact focus on API changes", true, "API changes"},
		{"/compact authentication flow", true, "authentication flow"},
		{"  /compact  ", true, ""},
		{"regular message", false, ""},
		{"/compacting", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			isCompact, focus := parseCompactCommand(tt.input)
			if isCompact != tt.isCompact || focus != tt.focus {
				t.Errorf("got (%v, %q) want (%v, %q)", isCompact, focus, tt.isCompact, tt.focus)
			}
		})
	}
}

func TestIsCompactionSummary(t *testing.T) {
	cases := map[string]bool{
		"<summary>We worked on auth...</summary>": true,
		"Summary of the conversation so far...":   true,
		"Here is what we did...":                  false,
	}
	for input, want := range cases {
		if got := isCompactionSummary(input); got != want {
			t.Errorf("isCompactionSummary(%q)=%v want %v", input, got, want)
		}
	}
}

func TestExtractSummaryContent(t *testing.T) {
	cases := map[string]string{
		"<summary>Auth bug was fixed by updating token validation</summary>":  "Auth bug was fixed by updating token validation",
		"Some preamble\n<summary>The actual summary</summary>\nSome epilogue": "The actual summary",
		"No tags here, just plain text":                                       "No tags here, just plain text",
	}
	for input, want := range cases {
		if got := extractSummaryContent(input); got != want {
			t.Errorf("extractSummaryContent(%q)=%q want %q", input, got, want)
		}
	}
}

func TestDecodeStreamLine_CompactionFlow(t *testing.T) {
	c := NewClaudeForTest()
	state := c.NewState()
	runID := uuid.New()

	events1, _ := c.DecodeStreamLine(state, runID, `{"type":"message","message":{"role":"user","content":"/compact focus on auth"}}`)
	if len(events1) != 0 {
		t.Errorf("/compact command should produce 0 events, got %d", len(events1))
	}

	events2, _ := c.DecodeStreamLine(state, runID, `{"type":"message","message":{"role":"assistant","content":"<summary>We fixed the auth bug...</summary>"}}`)
	if len(events2) != 1 {
		t.Fatalf("expected 1 compaction event, got %d", len(events2))
	}
	if events2[0].EventType != domain.EventTypeCompaction {
		t.Errorf("type=%s want compaction", events2[0].EventType)
	}
	cd := events2[0].Data.(*domain.CompactionEventData)
	if cd.Summary != "We fixed the auth bug..." {
		t.Errorf("summary=%q", cd.Summary)
	}
	if cd.Trigger != "manual" {
		t.Errorf("trigger=%s", cd.Trigger)
	}
	if cd.Focus != "auth" {
		t.Errorf("focus=%s", cd.Focus)
	}
	if cd.OriginalCommand != "/compact focus on auth" {
		t.Errorf("originalCommand=%s", cd.OriginalCommand)
	}
}

func TestDecodeStreamLine_AutoCompactSystem(t *testing.T) {
	c := NewClaudeForTest()
	events := decodeOne(t, c, diagnosticSamples["system_auto_compact"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	cd, ok := events[0].Data.(*domain.CompactionEventData)
	if !ok {
		t.Fatalf("expected CompactionEventData, got %T", events[0].Data)
	}
	if cd.Trigger != "auto" {
		t.Errorf("trigger=%s want auto", cd.Trigger)
	}
}

// =============================================================================
// ANSI handling
// =============================================================================

func TestDecodeStreamLine_ANSIInMessageContent(t *testing.T) {
	c := NewClaudeForTest()
	line := "{\"type\":\"message\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"\\u001b[31mhello\\u001b[0m world\"}]}}"
	events := decodeOne(t, c, line)
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	msg := events[0].Data.(*domain.MessageEventData)
	if msg.Content != "hello world" {
		t.Errorf("ANSI not stripped, got %q", msg.Content)
	}
}

func TestDecodeStreamLine_ANSIInToolResult(t *testing.T) {
	c := NewClaudeForTest()
	line := "{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"tool_use_id\":\"t1\",\"type\":\"tool_result\",\"content\":\"\\u001b[32mfile created\\u001b[0m\"}]}}"
	events := decodeOne(t, c, line)
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	r := events[0].Data.(*domain.ToolResultEventData)
	if r.Output != "file created" {
		t.Errorf("ANSI not stripped, got %q", r.Output)
	}
}

func TestDecodeStreamLine_UserTextSuppressed(t *testing.T) {
	c := NewClaudeForTest()
	line := `{"type":"user","message":{"role":"user","content":"plain user text"}}`
	events := decodeOne(t, c, line)
	for _, e := range events {
		if e.EventType == domain.EventTypeMessage {
			t.Errorf("user text should be suppressed, got %v", e)
		}
	}
}

// =============================================================================
// ParseTranscriptLine — terminal extraction
// =============================================================================

func TestParseTranscriptLine_TerminalSuccess(t *testing.T) {
	c := NewClaudeForTest()
	r := c.ParseTranscriptLine(uuid.New(), claudeCodeSamples["result_success"])
	if r.Terminal == nil {
		t.Fatal("expected Terminal")
	}
	if !r.Terminal.Success {
		t.Errorf("expected success=true")
	}
	if r.Terminal.ExitCode != 0 {
		t.Errorf("exitCode=%d", r.Terminal.ExitCode)
	}
}

func TestParseTranscriptLine_TerminalRateLimit(t *testing.T) {
	c := NewClaudeForTest()
	r := c.ParseTranscriptLine(uuid.New(), claudeCodeSamples["result_error"])
	if r.Terminal == nil {
		t.Fatal("expected Terminal")
	}
	if r.Terminal.Success {
		t.Errorf("expected success=false")
	}
	if r.Terminal.ExitCode != 429 {
		t.Errorf("exitCode=%d want 429", r.Terminal.ExitCode)
	}
}

func TestParseTranscriptLine_SessionID(t *testing.T) {
	c := NewClaudeForTest()
	r := c.ParseTranscriptLine(uuid.New(), claudeCodeSamples["assistant_text"])
	if r.SessionID != "751b5a53-bc44-4484-943d-8851ccfdfda1" {
		t.Errorf("sessionID=%q", r.SessionID)
	}
}

func TestClaudeTranscriptDoesNotDuplicateFinalAssistant(t *testing.T) {
	c := NewClaudeForTest()
	runID := uuid.New()
	parser := c.NewTranscriptParser()

	assistant := parser.ParseTranscriptLine(runID, `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"done"}]},"session_id":"session-1"}`)
	result := parser.ParseTranscriptLine(runID, `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"session-1"}`)

	got := 0
	for _, res := range []runner.TranscriptParseResult{assistant, result} {
		if res.Err != nil {
			t.Fatalf("ParseTranscriptLine: %v", res.Err)
		}
		for _, event := range res.Events {
			if event.EventType == domain.EventTypeMessage {
				msg := event.Data.(*domain.MessageEventData)
				if msg.Role == "assistant" && msg.Content == "done" {
					got++
				}
			}
		}
	}
	if got != 1 {
		t.Fatalf("assistant message count = %d, want 1", got)
	}
}

func TestClaudeTranscriptResultSynthesizesAssistantWhenNoAssistantEventExists(t *testing.T) {
	c := NewClaudeForTest()
	res := c.ParseTranscriptLine(uuid.New(), `{"type":"result","subtype":"success","is_error":false,"result":"fallback summary","session_id":"session-1"}`)
	if res.Err != nil {
		t.Fatalf("ParseTranscriptLine: %v", res.Err)
	}
	got := 0
	for _, event := range res.Events {
		if event.EventType == domain.EventTypeMessage {
			msg := event.Data.(*domain.MessageEventData)
			if msg.Role == "assistant" && msg.Content == "fallback summary" {
				got++
			}
		}
	}
	if got != 1 {
		t.Fatalf("fallback assistant message count = %d, want 1", got)
	}
}

// =============================================================================
// ClaudeMessage extraction
// =============================================================================

func TestClaudeMessage_ExtractTextContent_StringContent(t *testing.T) {
	msg := &ClaudeMessage{Role: "assistant", Content: json.RawMessage(`"Hello, world!"`)}
	if got := msg.ExtractTextContent(); got != "Hello, world!" {
		t.Errorf("got %q", got)
	}
}

func TestClaudeMessage_ExtractTextContent_ArrayContent(t *testing.T) {
	content := `[{"type": "text", "text": "First part"},{"type": "text", "text": "Second part"}]`
	msg := &ClaudeMessage{Role: "assistant", Content: json.RawMessage(content)}
	if got := msg.ExtractTextContent(); got != "First part\nSecond part" {
		t.Errorf("got %q", got)
	}
}

func TestClaudeMessage_ExtractTextContent_MixedContent(t *testing.T) {
	content := `[{"type":"text","text":"Let me help you with that."},{"type":"tool_use","id":"t","name":"Bash","input":{"command":"ls"}},{"type":"text","text":"Here are the results."}]`
	msg := &ClaudeMessage{Role: "assistant", Content: json.RawMessage(content)}
	if got := msg.ExtractTextContent(); got != "Let me help you with that.\nHere are the results." {
		t.Errorf("got %q", got)
	}
}

func TestClaudeMessage_ExtractTextContent_EmptyShapes(t *testing.T) {
	cases := []struct {
		name    string
		content json.RawMessage
	}{
		{"nil", nil},
		{"empty", json.RawMessage(``)},
		{"empty string", json.RawMessage(`""`)},
		{"empty array", json.RawMessage(`[]`)},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			msg := &ClaudeMessage{Content: tt.content}
			if got := msg.ExtractTextContent(); got != "" {
				t.Errorf("got %q", got)
			}
		})
	}
}

func TestClaudeMessage_ExtractToolUses(t *testing.T) {
	content := `[
		{"type":"text","text":"intro"},
		{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}},
		{"type":"tool_use","id":"t2","name":"Read","input":{"file_path":"/tmp"}}
	]`
	msg := &ClaudeMessage{Content: json.RawMessage(content)}
	tools := msg.ExtractToolUses()
	if len(tools) != 2 {
		t.Fatalf("got %d tools", len(tools))
	}
	if tools[0].Name != "Bash" || tools[1].Name != "Read" {
		t.Errorf("unexpected names: %s, %s", tools[0].Name, tools[1].Name)
	}
}

// =============================================================================
// Diagnostic helpers
// =============================================================================

func TestTailBytesUTF8Safe_TruncatesAtRuneBoundary(t *testing.T) {
	input := "aaaaaaé"
	got := tailBytesUTF8Safe(input, 3)
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8: %q", got)
	}
	if len(got) > 3 {
		t.Fatalf("got len %d > 3", len(got))
	}
}

func TestTailBytesUTF8Safe_ShortInputUnchanged(t *testing.T) {
	if got := tailBytesUTF8Safe("abc", 100); got != "abc" {
		t.Errorf("got %q", got)
	}
}

func TestRedactSecrets(t *testing.T) {
	in := "Authorization: Bearer abcdef123456 and api_key=supersecretvalue and sk-abcdefgh"
	out := redactSecrets(in)
	for _, secret := range []string{"abcdef123456", "supersecretvalue", "sk-abcdefgh"} {
		if strings.Contains(out, secret) {
			t.Errorf("secret %q not redacted: %q", secret, out)
		}
	}
	if !strings.Contains(out, "<redacted>") {
		t.Error("expected at least one <redacted>")
	}
}

func TestIsAutoCompactMarker(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"regular log line", false},
		{"Auto-compacting conversation to free tokens", true},
		{"  CONVERSATION HISTORY HAS BEEN COMPACTED  ", true},
		{"context has been compacted", true},
		{"Automatic compaction begins now", true},
		{"auto-compact", false},
	}
	for _, c := range cases {
		if got := isAutoCompactMarker(c.in); got != c.want {
			t.Errorf("isAutoCompactMarker(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestFormatErrorMessage_NonEmpty(t *testing.T) {
	got := formatErrorMessage("", 0, 0, "")
	if got == "" {
		t.Error("expected non-empty even for all-empty inputs")
	}
}

func TestBuildErrorDetails_OmitsEmptyStringFields(t *testing.T) {
	details := buildErrorDetails("", 5, 1000, "", "", "")
	if _, ok := details["session_id"]; ok {
		t.Error("session_id should be omitted when empty")
	}
	if _, ok := details["stderr_tail"]; ok {
		t.Error("stderr_tail should be omitted when empty")
	}
	if details["num_turns"] != 5 {
		t.Errorf("num_turns=%v", details["num_turns"])
	}
}

// =============================================================================
// Misc helpers
// =============================================================================

func findErrorEvent(events []*domain.RunEvent) *domain.RunEvent {
	for _, e := range events {
		if e == nil {
			continue
		}
		if _, ok := e.Data.(*domain.ErrorEventData); ok {
			return e
		}
	}
	return nil
}
