package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	harnesslifecycle "github.com/vrooli/vrooli/packages/harness-lifecycle"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	capsH "web-console/handlers/capabilities"
)

// The time budget for one agent update action.
//
// A governed npm install takes up to three minutes in the Dependency Analyzer
// alone, and the verb bounds itself at six so a repair can still restore what it
// moved. harnessActionWindow outlives that, so the verb is never killed partway.
// This exceeds httpWriteTimeout, so the handler extends its own write deadline;
// and because a tunnel may drop a slow request anyway, the action runs detached
// from the request and a repeated request rejoins it.
const (
	harnessActionWindow             = 7 * time.Minute
	harnessActionConfirmationWindow = 40 * time.Second
	harnessActionBudget             = harnessActionWindow + harnessActionConfirmationWindow + 20*time.Second
	harnessActionResponseDeadline   = harnessActionBudget + 30*time.Second
	harnessActionConfirmInterval    = 5 * time.Second
)

// agentUpdateActionArgs maps a browser action to one exact command. Nothing from
// the request reaches argv except the catalogued resource name.
func agentUpdateActionArgs(action, resource string) (command string, args []string, ok bool) {
	switch action {
	case "install", "doctor", "preview_cleanup", "uninstall":
		return "resource update", []string{resource, "--action", strings.ReplaceAll(action, "_", "-"), "--json"}, true
	case "update":
		return "resource update", []string{resource, "--json"}, true
	case "use_managed_copy":
		return "resource update", []string{resource, "--action", "use-managed-copy", "--json"}, true
	case "link_managed_copy":
		return "resource update", []string{resource, "--action", "link-managed-copy", "--json"}, true
	case "rollback":
		return "resource update", []string{resource, "--action", "rollback", "--json"}, true
	case agentActionRefreshCatalog:
		return "resource update", []string{resource, "--action", "refresh-catalog", "--json"}, true
	}
	return "", nil, false
}

func agentUpdateActionKey(targetID, agentID string) string { return targetID + "\x00" + agentID }

type agentUpdateActionFlight struct {
	action string
	done   chan struct{}
	result capsH.AgentUpdateActionResult
	err    error
}

// agentUpdateActions serializes actions per harness per target. The control
// plane also locks across processes; this answer is just faster and kinder.
type agentUpdateActions struct {
	mu     sync.Mutex
	active map[string]*agentUpdateActionFlight
}

func (a *agentUpdateActions) running(key string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.active[key] != nil
}

// join returns the flight for key, starting run when none is active. A repeat of
// the running action shares its outcome; a different one is refused.
func (a *agentUpdateActions) join(key, action string, run func() (capsH.AgentUpdateActionResult, error)) (*agentUpdateActionFlight, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if flight := a.active[key]; flight != nil {
		if flight.action != action {
			return nil, fmt.Errorf("%w: %s is still running on this agent", capsH.ErrFailedPrecondition, flight.action)
		}
		return flight, nil
	}
	if a.active == nil {
		a.active = make(map[string]*agentUpdateActionFlight)
	}
	flight := &agentUpdateActionFlight{action: action, done: make(chan struct{})}
	a.active[key] = flight
	go func() {
		flight.result, flight.err = run()
		a.mu.Lock()
		delete(a.active, key)
		a.mu.Unlock()
		close(flight.done)
	}()
	return flight, nil
}

// guardAgentUpdateAction admits only a verified independent human on a
// same-origin request: the rule usage modes apply. A human credential forwarded
// by an agent inside a terminal carries agent provenance and is refused, so an
// agent can read this state and cannot change a machine through the console.
func guardAgentUpdateAction(w http.ResponseWriter, r *http.Request) bool {
	if !webConsoleSameOrigin(w, r) {
		return false
	}
	principal, err := authn.RequireIndependentHuman(r.Context())
	if err != nil || !principal.IsHuman() {
		http.Error(w, "agent updates can only be changed by the signed-in operator", http.StatusForbidden)
		return false
	}
	// The default bounds every response; this one is allowed a real install.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(harnessActionResponseDeadline)); err != nil {
		log.Printf("agent update action: cannot extend write deadline: %v", err)
	}
	return true
}

func (s *Server) runAgentUpdateAction(ctx context.Context, targetID, agentID, action, cleanupPlanID string) (capsH.AgentUpdateActionResult, error) {
	targetID, agentID, action = strings.TrimSpace(targetID), strings.TrimSpace(agentID), strings.TrimSpace(action)
	principal, err := authn.RequireIndependentHuman(ctx)
	if err != nil || !principal.IsHuman() {
		return capsH.AgentUpdateActionResult{}, fmt.Errorf("%w: agent updates can only be changed by the signed-in operator", capsH.ErrPermissionDenied)
	}
	resource, ok := agentResource(agentID)
	if !ok || targetID == "" {
		return capsH.AgentUpdateActionResult{}, fmt.Errorf("unknown target or agent")
	}
	if action == agentActionRetry {
		s.agentUpdates.invalidate(targetID, agentID)
		update, err := s.getAgentUpdate(ctx, targetID, agentID)
		if err != nil {
			return capsH.AgentUpdateActionResult{}, err
		}
		return capsH.AgentUpdateActionResult{Success: update.State != agentStateCheckFailed, Message: update.Reason, Update: agentUpdateResult(update)}, nil
	}
	command, args, ok := agentUpdateActionArgs(action, resource)
	if action == "uninstall" {
		args = append(args, "--cleanup-plan", cleanupPlanID)
	}
	if !ok {
		return capsH.AgentUpdateActionResult{}, fmt.Errorf("unknown action %q", action)
	}
	// Model lists are only read locally, so a remote refresh could not be seen.
	if action == agentActionRefreshCatalog && targetID != "local" {
		return capsH.AgentUpdateActionResult{}, fmt.Errorf("%w: a model list refreshes on this machine only", capsH.ErrFailedPrecondition)
	}
	target, err := s.agentUpdateTarget(targetID)
	if err != nil {
		return capsH.AgentUpdateActionResult{}, err
	}
	if !target.Available {
		return capsH.AgentUpdateActionResult{}, fmt.Errorf("%w: the selected machine is unavailable", capsH.ErrUnavailable)
	}
	actor := principal.OwnerSubject()
	// Detached: closing a tab or losing a tunnel must not kill an install midway.
	detached := context.WithoutCancel(ctx)
	flight, err := s.agentUpdateActions.join(agentUpdateActionKey(targetID, agentID), action+"\x00"+cleanupPlanID, func() (capsH.AgentUpdateActionResult, error) {
		budget, cancel := context.WithTimeout(detached, harnessActionBudget)
		defer cancel()
		return s.executeAgentUpdateAction(budget, target, targetID, agentID, action, actor, cleanupPlanID, command, args)
	})
	if err != nil {
		return capsH.AgentUpdateActionResult{}, err
	}
	select {
	case <-flight.done:
		return flight.result, flight.err
	case <-ctx.Done():
		return capsH.AgentUpdateActionResult{}, ctx.Err()
	}
}

func (s *Server) executeAgentUpdateAction(ctx context.Context, target targetConnection, targetID, agentID, action, actor, cleanupPlanID, command string, args []string) (capsH.AgentUpdateActionResult, error) {
	window, cancel := context.WithTimeout(ctx, harnessActionWindow)
	defer cancel()
	// The verb records this as the actor. Only this request's verified human reaches here.
	if command == "resource update" && actor != "" {
		args = append(args[:len(args):len(args)], "--on-behalf-of", actor)
	}
	outcome, err := s.invokeAgentUpdateCommand(window, target, targetID, command, args, agentID, action, actor, cleanupPlanID)
	// The install may have changed whatever the error says.
	s.agentUpdates.invalidate(targetID, agentID)
	s.auditAgentUpdateAction(actor, targetID, agentID, action, outcome, err)
	if err != nil {
		return capsH.AgentUpdateActionResult{}, err
	}
	update, observeErr := s.observeAgentUpdateAfterAction(ctx, targetID, agentID)
	if observeErr != nil {
		return capsH.AgentUpdateActionResult{}, observeErr
	}
	success := outcome.Success
	if action == "install" && success && update.State == agentStateMissing {
		success = false
		outcome.Message = "The installer finished, but this machine does not report the agent yet."
	}
	return capsH.AgentUpdateActionResult{Success: success, Message: outcome.Message, Update: agentUpdateResult(update), OperationID: outcome.OperationID, CleanupPlan: agentCleanupPlan(outcome.CleanupPlan)}, nil
}

// agentUpdateCommandOutcome is what one verb run established.
type agentUpdateCommandOutcome struct {
	CleanupPlan *harnesslifecycle.CleanupPlan
	Success     bool
	Message     string
	OperationID string
}

func (s *Server) invokeAgentUpdateCommand(ctx context.Context, target targetConnection, targetID, command string, args []string, agentID, action, actor, cleanupPlanID string) (agentUpdateCommandOutcome, error) {
	if targetID == "local" {
		if s.agentLifecycle.manage == nil && os.Getenv("VROOLI_DESKTOP_MODE") != "true" {
			data, runErr := runControlPlaneHarness(ctx, command, args)
			return parseAgentUpdateCommand(data, runErr == nil, errorText(runErr)), nil
		}
		resource, ok := agentResource(agentID)
		if !ok {
			return agentUpdateCommandOutcome{}, fmt.Errorf("unknown agent %q", agentID)
		}
		manage := s.agentLifecycle.manage
		if manage == nil {
			manage = harnesslifecycle.ManageDesktopHarness
		}
		result, err := manage(ctx, localHarnessRoot(), resource, action, actor, cleanupPlanID)
		return agentUpdateCommandOutcome{Success: result.Success, Message: result.Message, CleanupPlan: result.CleanupPlan}, err
	}
	response, err := s.relayAgentUpdate(ctx, target, command, args, harnessActionWindow)
	if err != nil {
		return agentUpdateCommandOutcome{}, fmt.Errorf("Bridge could not run the action on %s: %w", target.Label, err)
	}
	completed := response.Outcome == relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED && response.ExitCode == 0
	outcome := parseAgentUpdateCommand(response.Data, completed, response.Reason)
	outcome.OperationID = response.GetCorrelationId()
	return outcome, nil
}

func runControlPlaneHarness(ctx context.Context, command string, args []string) ([]byte, error) {
	binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{})
	if err != nil {
		return nil, err
	}
	var out, stderr boundedOutput
	result := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: binary, Args: append(strings.Fields(command), args...), Timeout: harnessActionWindow, Stdout: &out, Stderr: &stderr})
	return out.Bytes(), result.Error()
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// parseAgentUpdateCommand prefers the verb's own verdict. A failed action exits
// non-zero and still prints it, so the exit status alone would lose the reason.
func parseAgentUpdateCommand(data []byte, completed bool, fallback string) agentUpdateCommandOutcome {
	var envelope harnessUpdateEnvelope
	if json.NewDecoder(bytes.NewReader(data)).Decode(&envelope) == nil && envelope.Update.Action != "" {
		message := envelope.Update.Message
		if message == "" {
			message = fallback
		}
		return agentUpdateCommandOutcome{Success: completed && envelope.Update.Success, Message: message, CleanupPlan: envelope.Update.CleanupPlan}
	}
	if !completed {
		message := strings.TrimSpace(fallback)
		if message == "" {
			message = "The action did not complete."
		}
		return agentUpdateCommandOutcome{Message: message}
	}
	return agentUpdateCommandOutcome{Success: true, Message: "The action completed."}
}

// observeAgentUpdateAfterAction confirms by asking the target again. A local
// probe is immediate. A Bridge node re-probes on its heartbeat, so a version that
// just changed can briefly disagree with the node's last report.
func (s *Server) observeAgentUpdateAfterAction(ctx context.Context, targetID, agentID string) (agentUpdate, error) {
	deadline := time.Now().Add(harnessActionConfirmationWindow)
	for {
		s.agentUpdates.invalidate(targetID, agentID)
		update, err := s.getAgentUpdate(ctx, targetID, agentID)
		if err != nil || targetID == "local" || update.State != agentStateCheckFailed || time.Now().After(deadline) {
			return update, err
		}
		select {
		case <-ctx.Done():
			return update, nil
		case <-time.After(harnessActionConfirmInterval):
		}
	}
}

// auditAgentUpdateAction records who changed which machine's agent. A remote
// action also has Bridge's durable record, named by the operation ID.
func (s *Server) auditAgentUpdateAction(actor, targetID, agentID, action string, outcome agentUpdateCommandOutcome, err error) {
	success := err == nil && outcome.Success
	log.Printf("agent update action: actor=%q target=%q agent=%q action=%q success=%t operation=%q", actor, targetID, agentID, action, success, outcome.OperationID)
	if s.events == nil {
		return
	}
	s.events.Emit("agent_update_action", "", map[string]string{"actor": actor, "target_id": targetID, "agent_id": agentID, "action": action, "success": fmt.Sprint(success), "operation_id": outcome.OperationID})
}

func agentCleanupPlan(plan *harnesslifecycle.CleanupPlan) *capsH.HarnessCleanupPlan {
	if plan == nil {
		return nil
	}
	return &capsH.HarnessCleanupPlan{ID: plan.ID, Remove: plan.Remove, Preserved: plan.Preserved}
}
