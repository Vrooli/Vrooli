package vision

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/credits"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// minClaudeVersion is the minimum Claude CLI version required for Chrome MCP support.
var minClaudeVersion = semver.MustParse("1.0.18")

const resourceClaudeCommand = "resource-claude-code"

// CommandRunner abstracts command execution for testing.
type CommandRunner interface {
	LookPath(file string) (string, error)
	CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd
}

// defaultCommandRunner implements CommandRunner using the standard library.
type defaultCommandRunner struct{}

func (r *defaultCommandRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (r *defaultCommandRunner) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// ClaudeCodeVisionNavigator implements VisionNavigator using Claude Code CLI.
// It spawns the Claude CLI as a subprocess with Chrome MCP tools for browser automation.
type ClaudeCodeVisionNavigator struct {
	log   *logrus.Logger
	wsHub wsHub.HubInterface

	// Recording callback for unified action capture.
	// When set, all AI navigation actions are reported for recording.
	onActionRecord ActionRecordCallback

	// Session tracking
	mu                sync.RWMutex
	activeNavigations map[string]*claudeCodeSession

	// For testability
	cmdRunner CommandRunner
}

// claudeCodeSession tracks an active Claude Code navigation.
type claudeCodeSession struct {
	*NavigationSession
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	doneChan   chan struct{}
	stderrBuf  strings.Builder
	stderrPipe io.ReadCloser
	mu         sync.Mutex
}

// claudeStreamEvent represents an event from Claude CLI's stream-json output.
type claudeStreamEvent struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	Message   json.RawMessage `json:"message,omitempty"`
	Name      string          `json:"name,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Error     string          `json:"error,omitempty"`
	Result    string          `json:"result,omitempty"`
	Duration  float64         `json:"duration_ms,omitempty"`
}

// ClaudeCodeNavigatorOption configures ClaudeCodeVisionNavigator.
type ClaudeCodeNavigatorOption func(*ClaudeCodeVisionNavigator)

// WithClaudeCodeActionRecordCallback sets the callback for recording AI navigation actions.
func WithClaudeCodeActionRecordCallback(callback ActionRecordCallback) ClaudeCodeNavigatorOption {
	return func(n *ClaudeCodeVisionNavigator) {
		n.onActionRecord = callback
	}
}

// WithClaudeCodeHub sets the WebSocket hub for broadcasting events.
func WithClaudeCodeHub(hub wsHub.HubInterface) ClaudeCodeNavigatorOption {
	return func(n *ClaudeCodeVisionNavigator) {
		n.wsHub = hub
	}
}

// WithClaudeCodeCommandRunner sets a custom command runner for testing.
func WithClaudeCodeCommandRunner(runner CommandRunner) ClaudeCodeNavigatorOption {
	return func(n *ClaudeCodeVisionNavigator) {
		n.cmdRunner = runner
	}
}

// NewClaudeCodeVisionNavigator creates a new Claude Code-based navigator.
func NewClaudeCodeVisionNavigator(log *logrus.Logger, opts ...ClaudeCodeNavigatorOption) *ClaudeCodeVisionNavigator {
	n := &ClaudeCodeVisionNavigator{
		log:               log,
		activeNavigations: make(map[string]*claudeCodeSession),
		cmdRunner:         &defaultCommandRunner{},
	}

	for _, opt := range opts {
		opt(n)
	}

	return n
}

// SetActionRecordCallback sets the callback for recording AI navigation actions.
func (n *ClaudeCodeVisionNavigator) SetActionRecordCallback(callback ActionRecordCallback) {
	n.onActionRecord = callback
}

// Type returns the navigator type.
func (n *ClaudeCodeVisionNavigator) Type() NavigatorType {
	return NavigatorClaudeCode
}

// Description returns a human-readable description.
func (n *ClaudeCodeVisionNavigator) Description() string {
	return "AI navigation using Claude Code CLI with Chrome"
}

// IsAvailable checks if Claude CLI is available and meets version requirements.
func (n *ClaudeCodeVisionNavigator) IsAvailable(ctx context.Context) bool {
	claudePath, err := n.cmdRunner.LookPath(resourceClaudeCommand)
	if err != nil {
		return false
	}

	version, err := n.getClaudeVersion(ctx, claudePath)
	if err != nil {
		return false
	}

	return !version.LessThan(minClaudeVersion)
}

// UnavailableReason returns why the navigator is unavailable.
func (n *ClaudeCodeVisionNavigator) UnavailableReason(ctx context.Context) string {
	claudePath, err := n.cmdRunner.LookPath(resourceClaudeCommand)
	if err != nil {
		return "claude CLI not found in PATH"
	}

	version, err := n.getClaudeVersion(ctx, claudePath)
	if err != nil {
		return fmt.Sprintf("failed to get claude version: %v", err)
	}

	if version.LessThan(minClaudeVersion) {
		return fmt.Sprintf("claude CLI version %s too old, requires %s+", version, minClaudeVersion)
	}

	return ""
}

// getClaudeVersion runs `claude --version` and parses the version string.
func (n *ClaudeCodeVisionNavigator) getClaudeVersion(ctx context.Context, claudePath string) (*semver.Version, error) {
	cmd := n.cmdRunner.CommandContext(ctx, claudePath, "run", "--", "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run claude --version: %w", err)
	}

	return parseClaudeVersion(strings.TrimSpace(string(output)))
}

// parseClaudeVersion extracts version from strings like "claude-code 1.0.18" or "1.0.18".
func parseClaudeVersion(versionStr string) (*semver.Version, error) {
	// Try to extract version number from the string
	// Expected formats: "claude-code 1.0.18", "1.0.18", "v1.0.18"
	versionStr = strings.TrimSpace(versionStr)

	// Remove common prefixes
	versionStr = strings.TrimPrefix(versionStr, "claude-code ")
	versionStr = strings.TrimPrefix(versionStr, "claude ")
	versionStr = strings.TrimPrefix(versionStr, "v")

	// Extract just the version part (handles "1.0.18 (build ...)" cases)
	parts := strings.Fields(versionStr)
	if len(parts) > 0 {
		versionStr = parts[0]
	}

	return semver.NewVersion(versionStr)
}

// CreditPolicy returns the credit policy for this navigator.
// Claude Code runs locally, so no credits are charged.
func (n *ClaudeCodeVisionNavigator) CreditPolicy() CreditPolicy {
	return CreditPolicy{
		RequiresCredits:  false,
		OperationType:    credits.OpAIVisionNavigate,
		PerStepCharging:  false,
		CreditsPerStep:   0,
		BypassConditions: []BypassCondition{BypassLocalExecution},
	}
}

// ClientSourcePolicy returns the client source policy (CLI only).
func (n *ClaudeCodeVisionNavigator) ClientSourcePolicy() ClientSourcePolicy {
	return CLIOnlyPolicy()
}

// Navigate starts an AI navigation session using Claude Code CLI.
func (n *ClaudeCodeVisionNavigator) Navigate(ctx context.Context, req NavigationRequest) (NavigationHandle, error) {
	// Verify availability
	if !n.IsAvailable(ctx) {
		reason := n.UnavailableReason(ctx)
		return nil, fmt.Errorf("%w: %s", ErrNavigatorNotAvailable, reason)
	}

	// Generate navigation ID
	navigationID := "nav_cc_" + uuid.New().String()[:12]

	// Resolve max steps
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 20
	}
	if maxSteps > 100 {
		maxSteps = 100
	}

	// Create session
	session := &NavigationSession{
		NavigationID:         navigationID,
		SessionID:            req.SessionID,
		UserID:               req.UserID,
		Model:                req.Model,
		StartedAt:            time.Now(),
		Status:               StatusNavigating,
		CredentialProvenance: CredentialProvenanceNone, // Claude Code uses local execution
		NavigatorType:        NavigatorClaudeCode,
	}

	ccSession := &claudeCodeSession{
		NavigationSession: session,
		doneChan:          make(chan struct{}),
	}

	// Track the session
	n.mu.Lock()
	n.activeNavigations[navigationID] = ccSession
	n.mu.Unlock()

	// Find claude CLI path
	claudePath, err := n.cmdRunner.LookPath(resourceClaudeCommand)
	if err != nil {
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("claude CLI not found: %w", err)
	}

	// Create cancellable context for the subprocess
	cmdCtx, cancel := context.WithCancel(context.Background())
	ccSession.cancel = cancel

	// Build command arguments
	// --print: non-interactive mode for subprocess
	// --output-format stream-json: JSON lines output
	// --verbose: required when using stream-json with --print
	// --max-turns: limit the number of agent turns
	// --allowedTools: restrict to Chrome MCP tools only
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--max-turns", fmt.Sprintf("%d", maxSteps*2), // Claude uses turns, not steps
		"--allowedTools", "mcp__claude-in-chrome__*",
	}

	cmdArgs := append([]string{"run", "--"}, args...)
	cmd := n.cmdRunner.CommandContext(cmdCtx, claudePath, cmdArgs...)
	ccSession.cmd = cmd

	// Set up stdin pipe to pass the prompt
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	// Set up stdout pipe for parsing stream-json output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	// Set up stderr pipe to capture error output
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}
	ccSession.stderrPipe = stderrPipe

	// Start the subprocess
	if err := cmd.Start(); err != nil {
		cancel()
		n.removeNavigation(navigationID)
		return nil, fmt.Errorf("start claude process: %w", err)
	}

	// Build and send the prompt
	prompt := n.buildPrompt(req, maxSteps)
	go func() {
		defer stdinPipe.Close()
		_, _ = io.WriteString(stdinPipe, prompt)
	}()

	// Start goroutine to capture stderr
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			ccSession.mu.Lock()
			ccSession.stderrBuf.WriteString(scanner.Text())
			ccSession.stderrBuf.WriteString("\n")
			ccSession.mu.Unlock()
		}
	}()

	// Start goroutine to parse stdout and report actions
	go n.parseOutput(ccSession, stdoutPipe)

	n.log.WithFields(logrus.Fields{
		"navigation_id": navigationID,
		"session_id":    req.SessionID,
		"max_steps":     maxSteps,
		"navigator":     "claude_code",
	}).Info("vision_navigation: started claude code navigation")

	return &claudeCodeNavigationHandle{
		navigator: n,
		session:   ccSession,
	}, nil
}

// buildPrompt creates the system prompt for Claude CLI.
func (n *ClaudeCodeVisionNavigator) buildPrompt(req NavigationRequest, maxSteps int) string {
	return fmt.Sprintf(`You are an AI browser automation agent. Navigate the web browser to accomplish this goal:

GOAL: %s

CONSTRAINTS:
- Maximum steps: %d
- You have access to Chrome browser via MCP tools
- Use mcp__claude-in-chrome__* tools for all browser interactions

INSTRUCTIONS:
1. First, get browser context with tabs_context_mcp to see available tabs
2. Create a new tab if needed with tabs_create_mcp
3. Navigate to the required page using the navigate tool
4. Interact with elements as needed to accomplish the goal
5. Use read_page or find tools to locate elements
6. Take screenshots when useful to verify progress
7. Stop when the goal is achieved

Begin now.`, req.Prompt, maxSteps)
}

// parseOutput reads the stream-json output from Claude CLI and processes events.
func (n *ClaudeCodeVisionNavigator) parseOutput(session *claudeCodeSession, stdout io.Reader) {
	defer func() {
		// Wait for the process to exit
		if session.cmd != nil {
			err := session.cmd.Wait()
			n.finalizeSession(session, err)
		}
		close(session.doneChan)
	}()

	// Use a large buffer for screenshots (base64 encoded images can be large)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // 4MB max

	stepNumber := 0
	lastReasoning := ""

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event claudeStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			n.log.WithError(err).WithField("line", line).Debug("failed to parse stream event")
			continue
		}

		switch event.Type {
		case "assistant":
			// Capture reasoning from assistant messages
			lastReasoning = n.extractReasoning(event.Content)

		case "tool_use":
			// Track MCP Chrome tool calls
			if strings.HasPrefix(event.Name, "mcp__claude-in-chrome__") {
				stepNumber++
				n.reportActionToRecording(session, &event, stepNumber, lastReasoning)
				n.broadcastStep(session, &event, stepNumber, lastReasoning)
				lastReasoning = "" // Clear after use
			}

		case "tool_result":
			// Could extract screenshot from tool_result if action was computer:screenshot
			// For now, we don't need to process this

		case "error":
			n.log.WithField("error", event.Error).Error("claude cli error")
			session.mu.Lock()
			session.Status = StatusFailed
			session.mu.Unlock()
			return

		case "result":
			// Navigation completed
			session.mu.Lock()
			if session.Status == StatusNavigating {
				session.Status = StatusCompleted
			}
			session.mu.Unlock()
			n.broadcastComplete(session)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		n.log.WithError(err).Error("error reading claude output")
	}
}

// extractReasoning extracts text content from assistant message content.
func (n *ClaudeCodeVisionNavigator) extractReasoning(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}

	// Content can be a string or array of content blocks
	var textContent string
	if err := json.Unmarshal(content, &textContent); err == nil {
		return textContent
	}

	// Try parsing as array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &blocks); err == nil {
		var texts []string
		for _, block := range blocks {
			if block.Type == "text" && block.Text != "" {
				texts = append(texts, block.Text)
			}
		}
		return strings.Join(texts, " ")
	}

	return ""
}

// reportActionToRecording creates a RecordedNavigationAction and invokes the callback.
func (n *ClaudeCodeVisionNavigator) reportActionToRecording(
	session *claudeCodeSession,
	event *claudeStreamEvent,
	stepNumber int,
	reasoning string,
) {
	if n.onActionRecord == nil {
		return
	}

	actionType, url, selector := n.mapMCPToolToAction(event)

	action := &RecordedNavigationAction{
		ActionType: actionType,
		URL:        url,
		PageTitle:  "", // Not available from stream output
		Selector:   selector,
		Reasoning:  reasoning,
		StepNumber: stepNumber,
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Source:     "ai",
	}

	n.onActionRecord(session.SessionID, action)
}

// mapMCPToolToAction extracts action type, URL, and selector from an MCP tool event.
func (n *ClaudeCodeVisionNavigator) mapMCPToolToAction(event *claudeStreamEvent) (actionType, url, selector string) {
	toolName := strings.TrimPrefix(event.Name, "mcp__claude-in-chrome__")

	// Parse input parameters
	var input map[string]interface{}
	if len(event.Input) > 0 {
		_ = json.Unmarshal(event.Input, &input)
	}

	switch toolName {
	case "navigate":
		actionType = "navigate"
		if u, ok := input["url"].(string); ok {
			url = u
		}

	case "computer":
		// Extract action type from input
		if action, ok := input["action"].(string); ok {
			actionType = n.normalizeComputerAction(action)
		}
		// Extract coordinates as selector if available
		if coord, ok := input["coordinate"].([]interface{}); ok && len(coord) == 2 {
			selector = fmt.Sprintf("[%v, %v]", coord[0], coord[1])
		}
		if ref, ok := input["ref"].(string); ok {
			selector = ref
		}

	case "form_input":
		actionType = "type"
		if ref, ok := input["ref"].(string); ok {
			selector = ref
		}

	case "find":
		actionType = "find"
		if query, ok := input["query"].(string); ok {
			selector = query
		}

	case "read_page":
		actionType = "read"
		if refID, ok := input["ref_id"].(string); ok {
			selector = refID
		}

	case "javascript_tool":
		actionType = "evaluate"

	case "tabs_context_mcp", "tabs_create_mcp":
		actionType = "tabs"

	case "get_page_text":
		actionType = "read"

	default:
		actionType = toolName
	}

	return
}

// normalizeComputerAction maps Claude computer action names to standardized types.
func (n *ClaudeCodeVisionNavigator) normalizeComputerAction(action string) string {
	switch action {
	case "left_click", "right_click", "double_click", "triple_click":
		return "click"
	case "type":
		return "type"
	case "screenshot":
		return "screenshot"
	case "scroll":
		return "scroll"
	case "key":
		return "keypress"
	case "left_click_drag":
		return "drag"
	case "hover":
		return "hover"
	case "wait":
		return "wait"
	case "zoom":
		return "zoom"
	case "scroll_to":
		return "scroll"
	default:
		return action
	}
}

// broadcastStep sends a WebSocket event for a navigation step.
func (n *ClaudeCodeVisionNavigator) broadcastStep(
	session *claudeCodeSession,
	event *claudeStreamEvent,
	stepNumber int,
	reasoning string,
) {
	if n.wsHub == nil {
		return
	}

	actionType, _, _ := n.mapMCPToolToAction(event)

	// Parse input for the action details
	var input map[string]interface{}
	if len(event.Input) > 0 {
		_ = json.Unmarshal(event.Input, &input)
	}

	wsEvent := map[string]interface{}{
		"type":         "ai_navigation_step",
		"navigationId": session.NavigationID,
		"sessionId":    session.SessionID,
		"stepNumber":   stepNumber,
		"action": map[string]interface{}{
			"type":  actionType,
			"input": input,
		},
		"reasoning":    reasoning,
		"goalAchieved": false,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	n.wsHub.BroadcastEnvelope(wsEvent)

	// Update session step count
	session.mu.Lock()
	session.StepCount = stepNumber
	session.mu.Unlock()
}

// broadcastComplete sends a WebSocket event when navigation completes.
func (n *ClaudeCodeVisionNavigator) broadcastComplete(session *claudeCodeSession) {
	if n.wsHub == nil {
		return
	}

	session.mu.Lock()
	status := session.Status
	stepCount := session.StepCount
	session.mu.Unlock()

	wsEvent := map[string]interface{}{
		"type":            "ai_navigation_complete",
		"navigationId":    session.NavigationID,
		"sessionId":       session.SessionID,
		"status":          status,
		"totalSteps":      stepCount,
		"totalDurationMs": time.Since(session.StartedAt).Milliseconds(),
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	n.wsHub.BroadcastEnvelope(wsEvent)
}

// finalizeSession updates the session status based on process exit.
func (n *ClaudeCodeVisionNavigator) finalizeSession(session *claudeCodeSession, err error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	// If already completed/failed/aborted, don't change status
	if session.Status != StatusNavigating {
		return
	}

	if err != nil {
		// Check if it was killed by signal
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == -1 {
				// Killed by signal (likely our abort)
				session.Status = StatusAborted
				return
			}
		}
		session.Status = StatusFailed
		n.log.WithError(err).WithField("navigation_id", session.NavigationID).Error("claude process exited with error")
	} else {
		session.Status = StatusCompleted
	}
}

// removeNavigation removes a navigation session from tracking.
func (n *ClaudeCodeVisionNavigator) removeNavigation(navigationID string) {
	n.mu.Lock()
	delete(n.activeNavigations, navigationID)
	n.mu.Unlock()
}

// GetSession returns a navigation session by ID.
func (n *ClaudeCodeVisionNavigator) GetSession(navigationID string) (*NavigationSession, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	session, exists := n.activeNavigations[navigationID]
	if !exists {
		return nil, false
	}
	return session.NavigationSession, true
}

// AbortNavigation sends a signal to stop the Claude CLI process.
func (n *ClaudeCodeVisionNavigator) AbortNavigation(ctx context.Context, navigationID string) error {
	n.mu.RLock()
	session, exists := n.activeNavigations[navigationID]
	n.mu.RUnlock()

	if !exists {
		return fmt.Errorf("navigation not found: %s", navigationID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.Status != StatusNavigating {
		return nil // Already finished
	}

	// Cancel context first
	if session.cancel != nil {
		session.cancel()
	}

	// Send SIGTERM
	if session.cmd != nil && session.cmd.Process != nil {
		if err := session.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			n.log.WithError(err).Debug("failed to send SIGTERM")
		}
	}

	// Wait up to 5 seconds for graceful exit, then SIGKILL
	select {
	case <-session.doneChan:
		// Process exited
	case <-time.After(5 * time.Second):
		if session.cmd != nil && session.cmd.Process != nil {
			_ = session.cmd.Process.Kill()
		}
	case <-ctx.Done():
		// Context cancelled, force kill
		if session.cmd != nil && session.cmd.Process != nil {
			_ = session.cmd.Process.Kill()
		}
	}

	session.Status = StatusAborted

	n.log.WithField("navigation_id", navigationID).Info("vision_navigation: claude code navigation aborted")
	return nil
}

// ResumeNavigation is not supported for Claude Code navigator.
// Claude Code doesn't pause for human input in the same way as playwright-driver.
func (n *ClaudeCodeVisionNavigator) ResumeNavigation(ctx context.Context, navigationID string) error {
	return errors.New("resume not supported for Claude Code navigator - start a new navigation instead")
}

// claudeCodeNavigationHandle implements NavigationHandle for Claude Code navigations.
type claudeCodeNavigationHandle struct {
	navigator *ClaudeCodeVisionNavigator
	session   *claudeCodeSession
}

// ID returns the navigation ID.
func (h *claudeCodeNavigationHandle) ID() string {
	return h.session.NavigationID
}

// SessionID returns the browser session ID.
func (h *claudeCodeNavigationHandle) SessionID() string {
	return h.session.SessionID
}

// Status returns the current navigation status.
func (h *claudeCodeNavigationHandle) Status() NavigationStatus {
	h.navigator.mu.RLock()
	session, exists := h.navigator.activeNavigations[h.session.NavigationID]
	h.navigator.mu.RUnlock()

	if !exists {
		return StatusCompleted // Session cleaned up
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	return session.Status
}

// Wait blocks until the navigation completes or the context is cancelled.
func (h *claudeCodeNavigationHandle) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.session.doneChan:
		return nil
	}
}

// Abort stops the navigation.
func (h *claudeCodeNavigationHandle) Abort(ctx context.Context) error {
	return h.navigator.AbortNavigation(ctx, h.session.NavigationID)
}

// Resume is not supported for Claude Code navigator.
func (h *claudeCodeNavigationHandle) Resume(ctx context.Context) error {
	return h.navigator.ResumeNavigation(ctx, h.session.NavigationID)
}
