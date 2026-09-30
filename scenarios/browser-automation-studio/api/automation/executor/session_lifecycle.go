package executor

import (
	"os"
	"strings"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
)

// executionBoundary names the points at which an execution may isolate browser
// state. A workflow's steps are deliberately not an isolation boundary.
type executionBoundary uint8

const (
	executionStart executionBoundary = iota
	betweenSteps
	executionEnd
)

type navigationState struct {
	hasNavigated bool
	lastAttempt  *failedNavigateAttempt
}

type failedNavigateAttempt struct {
	nodeID  string
	url     string
	summary string
}

// lifecycleDecision is the complete policy for one execution boundary. The
// executor owns actual session I/O; this package owns the policy so a reuse
// mode cannot accidentally acquire different semantics in graph execution.
type lifecycleDecision struct {
	ResetNavigation bool
}

func decideLifecycle(mode engine.SessionReuseMode, boundary executionBoundary, hasSession bool) lifecycleDecision {
	// A missing session always invalidates navigation, regardless of reuse
	// policy. Otherwise only execution start establishes a new navigation
	// state. In particular, fresh and clean never reset between steps.
	switch boundary {
	case executionStart:
		return lifecycleDecision{ResetNavigation: true}
	case betweenSteps:
		return lifecycleDecision{ResetNavigation: !hasSession}
	case executionEnd:
		return lifecycleDecision{}
	default:
		return lifecycleDecision{ResetNavigation: mode == engine.ReuseModeFresh && !hasSession}
	}
}

func resolveReuseMode(req Request) engine.SessionReuseMode {
	if req.ReuseMode != "" {
		return req.ReuseMode
	}
	if v, ok := req.Plan.Metadata["sessionReuseMode"].(string); ok && strings.TrimSpace(v) != "" {
		return normalizeReuseMode(v)
	}
	if env := strings.TrimSpace(os.Getenv("BAS_SESSION_STRATEGY")); env != "" {
		return normalizeReuseMode(env)
	}
	return engine.ReuseModeReuse
}

func normalizeReuseMode(raw string) engine.SessionReuseMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "fresh":
		return engine.ReuseModeFresh
	case "clean":
		return engine.ReuseModeClean
	default:
		return engine.ReuseModeReuse
	}
}

// shouldResetNavigation makes navigation state follow the session boundary.
// Fresh and clean isolate executions, never individual workflow steps.
func markNavigation(state *navigationState) {
	if state != nil {
		state.hasNavigated = true
		state.lastAttempt = nil
	}
}

func resetNavigation(state *navigationState) {
	if state != nil {
		state.hasNavigated = false
		state.lastAttempt = nil
	}
}

func recordFailedNavigate(state *navigationState, instruction contracts.CompiledInstruction, summary string) {
	if state == nil {
		return
	}
	url := ""
	if action := instruction.Action; action != nil && action.GetNavigate() != nil {
		url = action.GetNavigate().GetUrl()
	}
	state.lastAttempt = &failedNavigateAttempt{nodeID: instruction.NodeID, url: url, summary: summary}
}

func lastFailedNavigate(state *navigationState) *failedNavigateAttempt {
	if state == nil {
		return nil
	}
	return state.lastAttempt
}
