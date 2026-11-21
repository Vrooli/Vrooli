package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	harnesslifecycle "github.com/vrooli/vrooli/packages/harness-lifecycle"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
	capsH "web-console/handlers/capabilities"
)

const (
	agentUpdateTTL         = 5 * time.Minute
	agentUpdateFailureTTL  = 30 * time.Second
	agentObservationMaxAge = 2 * time.Minute
	// A check runs the agent's --version, reads the release source and asks the
	// Dependency Analyzer about one exact version.
	agentUpdateTimeout    = 30 * time.Second
	grokAuthStatusTimeout = 3 * time.Second

	agentUpdateMaxOutput    = 512 * 1024
	grokAuthStatusMaxOutput = 4 * 1024
)

// Update states. `vrooli resource update` owns their meaning; these are the
// values capabilities.proto promises the browser.
const (
	agentStateMissing         = "missing"
	agentStateFound           = "found"
	agentStateShadowed        = "shadowed"
	agentStateUnlinked        = "unlinked"
	agentStateNeedsSetup      = "needs_setup"
	agentStateUpdateReady     = "update_ready"
	agentStateUpToDate        = "up_to_date"
	agentStateCheckFailed     = "check_failed"
	agentActionRetry          = "retry"
	agentActionRefreshCatalog = "refresh_catalog"
)

// agentUpdate is an installed-tool observation from the target's control plane.
// The executable path stays on the server; the browser gets the evidence verdict.
type agentUpdate struct {
	TargetID          string               `json:"target_id"`
	AgentID           string               `json:"agent_id"`
	Resource          string               `json:"resource"`
	Installed         string               `json:"installed,omitempty"`
	Pinned            string               `json:"pinned,omitempty"`
	Latest            string               `json:"latest,omitempty"`
	Status            upstreamcheck.Status `json:"status"`
	CheckedAt         time.Time            `json:"checked_at,omitempty"`
	SourceKind        string               `json:"source_kind,omitempty"`
	SourceID          string               `json:"source_id,omitempty"`
	ProbeVerified     bool                 `json:"probe_verified"`
	Note              string               `json:"note,omitempty"`
	State             string               `json:"state"`
	PrimaryAction     string               `json:"primary_action,omitempty"`
	OperatorCommand   string               `json:"operator_command,omitempty"`
	Reason            string               `json:"reason,omitempty"`
	Checks            []harnessUpdateCheck `json:"checks,omitempty"`
	InstallKind       string               `json:"install_kind,omitempty"`
	InstallState      string               `json:"install_state,omitempty"`
	SignInState       string               `json:"sign_in_state,omitempty"`
	UpdateState       string               `json:"update_state,omitempty"`
	RollbackVersion   string               `json:"rollback_version,omitempty"`
	NewerThanReviewed bool                 `json:"newer_than_reviewed"`
}

// harnessUpdateCheck and harnessUpdateReport decode the `vrooli resource update`
// JSON contract. Web Console invokes the verb and never repeats its detection.
type harnessUpdateCheck struct {
	ID      string `json:"id"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type harnessUpdateReport struct {
	Name              string               `json:"name"`
	State             string               `json:"state"`
	PrimaryAction     string               `json:"primary_action"`
	OperatorCommand   string               `json:"operator_command"`
	Reason            string               `json:"reason"`
	InstallKind       string               `json:"install_kind"`
	InstallState      string               `json:"install_state"`
	SignInState       string               `json:"sign_in_state"`
	UpdateState       string               `json:"update_state"`
	Installed         string               `json:"installed"`
	ManagedVersion    string               `json:"managed_version"`
	ManagedPath       string               `json:"managed_path"`
	Pinned            string               `json:"pinned"`
	Latest            string               `json:"latest"`
	CommandPath       string               `json:"command_path"`
	OS                string               `json:"os"`
	Arch              string               `json:"arch"`
	CheckedAt         time.Time            `json:"checked_at"`
	RollbackVersion   string               `json:"rollback_version"`
	NewerThanReviewed bool                 `json:"newer_than_reviewed"`
	Checks            []harnessUpdateCheck `json:"checks"`
	Source            struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	} `json:"source"`
}

type harnessUpdateEnvelope struct {
	Success bool `json:"success"`
	Update  struct {
		CleanupPlan   *harnesslifecycle.CleanupPlan `json:"cleanup_plan"`
		Action        string                        `json:"action"`
		Success       bool                          `json:"success"`
		Message       string                        `json:"message"`
		PreservedPath string                        `json:"preserved_path"`
		Report        harnessUpdateReport           `json:"report"`
	} `json:"harness_update"`
}

// agentUpdateCommandError retains the subprocess classification needed to
// distinguish the owner's expected check-failed exit from interruption or
// failure to launch the command. Error text remains the underlying CLI error.
type agentUpdateCommandError struct {
	class    cliinvoke.Class
	exitCode int
	cause    error
}

func (e *agentUpdateCommandError) Error() string { return e.cause.Error() }
func (e *agentUpdateCommandError) Unwrap() error { return e.cause }

func expectedAgentCheckFailure(err error) bool {
	var commandErr *agentUpdateCommandError
	return errors.As(err, &commandErr) && commandErr.class == cliinvoke.Lifecycle && commandErr.exitCode == 1
}

// decodeAgentUpdateCheck accepts only the check verb's structured result. A
// check_failed report is a useful owner verdict even though the command exits
// nonzero; all other command failures remain unavailable. In particular, an
// interrupted check must not turn buffered output into a fresh observation.
func decodeAgentUpdateCheck(data []byte, resource string, commandErr, contextErr error) (harnessUpdateReport, bool) {
	if len(data) == 0 || len(data) > agentUpdateMaxOutput || contextErr != nil {
		return harnessUpdateReport{}, false
	}
	var shape struct {
		Success *bool `json:"success"`
		Update  *struct {
			Action  *string         `json:"action"`
			Success *bool           `json:"success"`
			Report  json.RawMessage `json:"report"`
		} `json:"harness_update"`
	}
	if err := json.Unmarshal(data, &shape); err != nil || shape.Success == nil || shape.Update == nil || shape.Update.Action == nil || shape.Update.Success == nil || len(shape.Update.Report) == 0 {
		return harnessUpdateReport{}, false
	}
	var envelope harnessUpdateEnvelope
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&envelope); err != nil {
		return harnessUpdateReport{}, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return harnessUpdateReport{}, false
	}

	if envelope.Update.Action != "check" || envelope.Update.Report.Name != resource || envelope.Update.Report.State == "" || envelope.Success != envelope.Update.Success {
		return harnessUpdateReport{}, false
	}
	if commandErr != nil {
		if !expectedAgentCheckFailure(commandErr) || envelope.Success || envelope.Update.Report.State != agentStateCheckFailed {
			return harnessUpdateReport{}, false
		}
	} else if !envelope.Success {
		return harnessUpdateReport{}, false
	}
	return envelope.Update.Report, true
}

type agentUpdateEntry struct {
	value   agentUpdate
	expires time.Time
}
type agentUpdateFlight struct {
	done  chan struct{}
	value agentUpdate
	err   error
}
type agentUpdateCache struct {
	mu         sync.Mutex
	items      map[string]agentUpdateEntry
	pending    map[string]*agentUpdateFlight
	generation uint64
}

// read coalesces simultaneous requests for one target/agent and bounds cache
// lifetime. The cache key includes the machine ID; no local report can become
// a remote result. A caller canceling while waiting does not cancel the owner.
func (c *agentUpdateCache) read(ctx context.Context, key string, load func(context.Context) (agentUpdate, error)) (agentUpdate, error) {
	c.mu.Lock()
	if entry, ok := c.items[key]; ok && time.Now().Before(entry.expires) {
		c.mu.Unlock()
		return entry.value, nil
	}
	if flight := c.pending[key]; flight != nil {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return agentUpdate{}, ctx.Err()
		case <-flight.done:
			return flight.value, flight.err
		}
	}
	flight := &agentUpdateFlight{done: make(chan struct{})}
	if c.pending == nil {
		c.pending = make(map[string]*agentUpdateFlight)
	}
	c.pending[key] = flight
	generation := c.generation
	c.mu.Unlock()
	value, err := load(ctx)
	c.mu.Lock()
	flight.value, flight.err = value, err
	if c.pending[key] == flight {
		delete(c.pending, key)
	}
	// A check that started before an action finished describes the old install.
	if err == nil && generation == c.generation {
		if c.items == nil {
			c.items = make(map[string]agentUpdateEntry)
		}
		for k, entry := range c.items {
			if time.Now().After(entry.expires) {
				delete(c.items, k)
			}
		}
		// A long-running console must not retain an unlimited history of
		// devices and changing installed versions.
		if len(c.items) >= 128 {
			c.items = make(map[string]agentUpdateEntry)
		}
		ttl := agentUpdateTTL
		if value.State == agentStateCheckFailed {
			ttl = agentUpdateFailureTTL
		}
		c.items[key] = agentUpdateEntry{value: value, expires: time.Now().Add(ttl)}
	}
	close(flight.done)
	c.mu.Unlock()
	return value, err
}

// invalidate forgets one target and agent. Any check already running for them
// finishes for its callers but is not stored, and a later read starts afresh.
func (c *agentUpdateCache) invalidate(targetID, agentID string) {
	prefix := targetID + "\x00" + agentID + "\x00"
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	for key := range c.items {
		if strings.HasPrefix(key, prefix) {
			delete(c.items, key)
		}
	}
	for key := range c.pending {
		if strings.HasPrefix(key, prefix) {
			delete(c.pending, key)
		}
	}
}

func agentResource(agentID string) (string, bool) {
	for _, def := range capabilityprobe.AITools {
		if def.ID == agentID {
			return cliutil.CodingAgentForAlias(def.Command)
		}
	}
	return "", false
}

func failedAgentUpdate(targetID, agentID, resource, reason string) agentUpdate {
	return agentUpdate{TargetID: targetID, AgentID: agentID, Resource: resource, Status: upstreamcheck.StatusUnknown, State: agentStateCheckFailed, PrimaryAction: agentActionRetry, Reason: reason, Note: reason}
}

// agentUpdateTarget resolves one selected machine. The current catalog must
// still contain it even if a previous observation is cached.
func (s *Server) agentUpdateTarget(targetID string) (targetConnection, error) {
	if targetID == "local" {
		return targetConnection{Target: targetmodel.Target{ID: "local", Available: true, OS: runtime.GOOS, Architecture: runtime.GOARCH}}, nil
	}
	target, ok := s.targetByID(targetID)
	if !ok {
		return targetConnection{}, fmt.Errorf("%w: %s", capsH.ErrTargetNotFound, targetID)
	}
	return target, nil
}

func (s *Server) getAgentUpdate(ctx context.Context, targetID, agentID string) (agentUpdate, error) {
	targetID, agentID = strings.TrimSpace(targetID), strings.TrimSpace(agentID)
	resource, ok := agentResource(agentID)
	if !ok || targetID == "" {
		return agentUpdate{}, fmt.Errorf("unknown target or agent")
	}
	target, err := s.agentUpdateTarget(targetID)
	if err != nil {
		return agentUpdate{}, err
	}
	if !target.Available {
		return failedAgentUpdate(targetID, agentID, resource, "The selected machine is unavailable"), nil
	}
	// The full local catalog probes every agent. This endpoint only needs the
	// selected agent, so do not trigger all five potentially slow probes for
	// each card or let those probes delay the bounded update lookup.
	var observation capabilityprobe.Observation
	if targetID == "local" {
		for _, definition := range capabilityprobe.AITools {
			if definition.ID == agentID {
				results := capabilityprobe.Probe(ctx, []capabilityprobe.Definition{definition})
				if len(results) == 1 {
					observation = results[0]
				}
				break
			}
		}
	} else {
		for _, item := range target.Capabilities {
			if item.ID == agentID {
				observation = item
				break
			}
		}
	}
	// Probe timestamp changes at every heartbeat. Key on the facts that affect
	// comparison, not the timestamp, to avoid a fresh network request each time.
	key := strings.Join([]string{targetID, agentID, target.OS, target.Architecture, string(observation.State), observation.Path, observation.Version}, "\x00")
	result, err := s.agentUpdates.read(ctx, key, func(ctx context.Context) (agentUpdate, error) {
		ctx, cancel := context.WithTimeout(ctx, agentUpdateTimeout)
		defer cancel()
		var data []byte
		var err error
		if targetID == "local" {
			data, err = s.localAgentUpdate(ctx, resource)
		} else {
			data, err = s.remoteAgentUpdate(ctx, target, resource)
		}
		rep, valid := decodeAgentUpdateCheck(data, resource, err, ctx.Err())
		if !valid {
			return failedAgentUpdate(targetID, agentID, resource, "Update check unavailable"), nil
		}
		base := failedAgentUpdate(targetID, agentID, resource, "")
		return projectAgentUpdate(base, rep, observation, target.OS, target.Architecture, time.Now().UTC()), nil
	})
	if err == nil && result.ProbeVerified && (result.CheckedAt.IsZero() || result.CheckedAt.After(time.Now()) || time.Since(result.CheckedAt) > agentUpdateTTL) {
		reason := "The update observation is out of date"
		result.Status, result.ProbeVerified, result.State, result.PrimaryAction, result.Reason, result.Note = upstreamcheck.StatusUnknown, false, agentStateCheckFailed, agentActionRetry, reason, reason
	}
	if agentID == "grok" {
		if targetID == "local" {
			result.SignInState = localGrokSignInState(ctx)
		} else {
			result.SignInState = remoteGrokSignInState(result.SignInState)
		}
	}
	return result, err
}

// The current remote `resource update-check` owner reports sign-in as unknown
// and has no auth verifier. Do not promote a field in its general update report
// to an auth claim until the remote target has an owner-side probe.
func remoteGrokSignInState(_ string) string { return "unknown" }

type grokAuthStatusResponse struct {
	Success     bool   `json:"success"`
	SignInState string `json:"sign_in_state"`
	Reason      string `json:"reason,omitempty"`
}

func normalizedSignInState(state string) string {
	switch strings.TrimSpace(state) {
	case "signed_in", "signed_out":
		return strings.TrimSpace(state)
	default:
		return "unknown"
	}
}

func decodeGrokSignInState(data []byte) string {
	if len(data) == 0 || len(data) > grokAuthStatusMaxOutput {
		return "unknown"
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var response grokAuthStatusResponse
	if err := decoder.Decode(&response); err != nil || !response.Success {
		return "unknown"
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return "unknown"
	}
	return normalizedSignInState(response.SignInState)
}

func localGrokSignInState(ctx context.Context) string {
	return readGrokSignInState(ctx, runLocalGrokAuthStatus)
}

func readGrokSignInState(ctx context.Context, run func(context.Context) ([]byte, error)) string {
	ctx, cancel := context.WithTimeout(ctx, grokAuthStatusTimeout)
	defer cancel()
	data, err := run(ctx)
	if err != nil {
		return "unknown"
	}
	return decodeGrokSignInState(data)
}

func runLocalGrokAuthStatus(ctx context.Context) ([]byte, error) {
	var output boundedOutput
	cmd := exec.CommandContext(ctx, "resource-grok", "auth", "status", "--json")
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > agentUpdateMaxOutput-b.Len() {
		return 0, errors.New("agent update output exceeded the limit")
	}
	return b.Buffer.Write(p)
}

type localHarnessLifecycle struct {
	inspect func(context.Context, string, string) (harnesslifecycle.HarnessReport, error)
	manage  func(context.Context, string, string, string, string, string) (harnesslifecycle.HarnessActionResult, error)
}

func (s *Server) localAgentUpdate(ctx context.Context, resource string) ([]byte, error) {
	inspect := s.agentLifecycle.inspect
	if inspect == nil && os.Getenv("VROOLI_DESKTOP_MODE") != "true" {
		return runControlPlaneHarness(ctx, "resource update-check", []string{resource, "--json"})
	}
	if inspect == nil {
		inspect = harnesslifecycle.InspectDesktopHarness
	}
	report, err := inspect(ctx, localHarnessRoot(), resource)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Success bool                                 `json:"success"`
		Update  harnesslifecycle.HarnessActionResult `json:"harness_update"`
	}{Success: true, Update: harnesslifecycle.HarnessActionResult{Action: "check", Success: true, Report: report}})
}

func localHarnessRoot() string {
	if os.Getenv("VROOLI_DESKTOP_MODE") == "true" {
		return filepath.Join(os.Getenv("BUNDLE_ROOT"), "catalog")
	}
	return cliutil.ResolveRepoRoot()
}

func (s *Server) readAgentUpdate(ctx context.Context, targetID, agentID string) (capsH.AgentUpdateResult, error) {
	update, err := s.getAgentUpdate(ctx, targetID, agentID)
	if err != nil {
		return capsH.AgentUpdateResult{}, err
	}
	result := agentUpdateResult(update)
	// While an action runs the observation may be either side of it, and a
	// second action could only wait or be refused.
	if s.agentUpdateActions.running(agentUpdateActionKey(update.TargetID, update.AgentID)) {
		result.PrimaryAction = ""
		result.Note = "An action on this agent is still running."
	}
	return result, nil
}

func agentUpdateResult(update agentUpdate) capsH.AgentUpdateResult {
	checked := ""
	if !update.CheckedAt.IsZero() {
		checked = update.CheckedAt.UTC().Format(time.RFC3339)
	}
	checks := make([]capsH.AgentUpdateCheck, 0, len(update.Checks))
	for _, check := range update.Checks {
		checks = append(checks, capsH.AgentUpdateCheck{ID: check.ID, Passed: check.Passed, Message: check.Message})
	}
	return capsH.AgentUpdateResult{
		TargetID: update.TargetID, AgentID: update.AgentID, Resource: update.Resource,
		Installed: update.Installed, Pinned: update.Pinned, Latest: update.Latest, Status: string(update.Status),
		CheckedAt: checked, SourceKind: update.SourceKind, SourceID: update.SourceID, ProbeVerified: update.ProbeVerified, Note: update.Note,
		State: update.State, PrimaryAction: update.PrimaryAction, OperatorCommand: update.OperatorCommand, Reason: update.Reason, Checks: checks,
		InstallKind: update.InstallKind, InstallState: update.InstallState, SignInState: update.SignInState, UpdateState: update.UpdateState,
		RollbackVersion: update.RollbackVersion, NewerThanReviewed: update.NewerThanReviewed,
	}
}

// probeConfirms reports whether a fresh capability probe saw the same install.
// The probe uses the API's PATH, which can order the managed link ahead of a
// copy that shells started by sessions run first, so either of the report's two
// copies is the same install; a different path or version is not.
func probeConfirms(rep harnessUpdateReport, observation capabilityprobe.Observation, now time.Time) bool {
	if observation.ProbedAt.IsZero() || observation.ProbedAt.After(now) || now.Sub(observation.ProbedAt) > agentObservationMaxAge || observation.State != capabilityprobe.Ready || observation.Path == "" {
		return false
	}
	path := filepath.Clean(observation.Path)
	version := upstreamcheck.ExtractSemver(observation.Version)
	if version == "" {
		return false
	}
	if rep.CommandPath != "" && path == filepath.Clean(rep.CommandPath) && version == rep.Installed {
		return true
	}
	// A shim on the API's PATH runs the install the report names.
	if version == rep.Installed && isOperatorShim(path) {
		return true
	}
	return rep.ManagedPath != "" && path == filepath.Clean(rep.ManagedPath) && version == rep.ManagedVersion
}

func isOperatorShim(path string) bool {
	return strings.Contains(filepath.ToSlash(filepath.Dir(path)), "/.vrooli/shims")
}

func projectAgentUpdate(out agentUpdate, rep harnessUpdateReport, observation capabilityprobe.Observation, goos, arch string, now time.Time) agentUpdate {
	out.Installed, out.Pinned, out.Latest = rep.Installed, rep.Pinned, rep.Latest
	out.CheckedAt, out.SourceKind, out.SourceID = rep.CheckedAt, rep.Source.Kind, rep.Source.ID
	out.State, out.PrimaryAction, out.OperatorCommand, out.Reason, out.Note = rep.State, rep.PrimaryAction, rep.OperatorCommand, rep.Reason, ""
	out.Checks, out.InstallKind, out.RollbackVersion, out.NewerThanReviewed = rep.Checks, rep.InstallKind, rep.RollbackVersion, rep.NewerThanReviewed
	out.InstallState, out.SignInState, out.UpdateState = rep.InstallState, rep.SignInState, rep.UpdateState
	out.Status = upstreamcheck.Compare(rep.Installed, rep.Latest)
	refuse := func(reason string) agentUpdate {
		out.Status, out.ProbeVerified, out.State, out.PrimaryAction, out.OperatorCommand, out.Reason, out.Note = upstreamcheck.StatusUnknown, false, agentStateCheckFailed, agentActionRetry, "", reason, reason
		return out
	}
	// A report for another platform or moment does not describe this install.
	if rep.CheckedAt.IsZero() || rep.CheckedAt.After(now) || now.Sub(rep.CheckedAt) > agentUpdateTTL || rep.OS == "" || rep.Arch == "" || rep.OS != goos || rep.Arch != arch {
		return refuse("Cannot verify the installed executable on this machine")
	}
	out.ProbeVerified = probeConfirms(rep, observation, now)
	switch rep.State {
	case agentStateMissing, agentStateFound, agentStateShadowed, agentStateUnlinked, agentStateNeedsSetup, agentStateCheckFailed:
		// These come from the file evidence on the target and offer a repair, not
		// a version claim, so a lagging heartbeat probe cannot make them false.
		if rep.State == agentStateCheckFailed {
			// The owner verdict is diagnostic evidence only. A matching installed
			// version probe cannot turn a failed upstream/policy check into proof
			// that an update is ready or that the latest version is known.
			out.Status, out.ProbeVerified, out.PrimaryAction = upstreamcheck.StatusUnknown, false, agentActionRetry
			if out.Reason == "" {
				out.Reason = "The update check failed"
			}
			out.Note = out.Reason
		}
		return out
	case agentStateUpdateReady:
		if out.Status != upstreamcheck.StatusBehind {
			return refuse("Upstream version is unavailable or ambiguous")
		}
	case agentStateUpToDate:
		if out.Status != upstreamcheck.StatusUpToDate && out.Status != upstreamcheck.StatusAhead {
			return refuse("Upstream version is unavailable or ambiguous")
		}
	default:
		return refuse("Update check returned an unknown state")
	}
	if !out.ProbeVerified {
		return refuse("Cannot verify the installed executable on this machine")
	}
	return out
}

func (s *Server) remoteAgentUpdate(ctx context.Context, target targetConnection, resource string) ([]byte, error) {
	response, err := s.relayAgentUpdate(ctx, target, "resource update-check", []string{resource, "--json"}, agentUpdateTimeout)
	if err != nil {
		return nil, err
	}
	if response.Outcome != relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED {
		return nil, errors.New("target check was not completed")
	}
	if len(response.Data) > agentUpdateMaxOutput {
		return nil, errors.New("target check response exceeded the output limit")
	}
	if response.ExitCode != 0 {
		return response.Data, &agentUpdateCommandError{class: cliinvoke.Lifecycle, exitCode: int(response.ExitCode), cause: errors.New("target check returned an owner failure")}
	}
	return response.Data, nil
}

// relayAgentUpdate runs one typed command on a Bridge node. The relay performs
// owner authentication, node-scope admission by the verb's effect, and audit.
func (s *Server) relayAgentUpdate(ctx context.Context, target targetConnection, command string, args []string, window time.Duration) (*relayv1.RelayCallResponse, error) {
	if target.NodeID == "" {
		return nil, errors.New("target has no Bridge node")
	}
	base := strings.TrimRight(strings.TrimSpace(target.BaseURL), "/")
	if base == "" {
		var err error
		base, err = resolveBridgeScenarioURL(ctx)
		if err != nil {
			return nil, err
		}
	}
	client := relayconnect.NewRelayServiceClient(authenticatedHTTPClientWithTimeout(freshOwnerAuthorization(ctx, target.OwnerToken), target.ReauthToken, window+10*time.Second), base)
	response, err := client.Call(ctx, connect.NewRequest(&relayv1.RelayCallRequest{
		NodeId: target.NodeID, Scenario: "vrooli", Command: command, Args: args, TimeoutSeconds: int64(window.Seconds()), MaxResponseBytes: agentUpdateMaxOutput,
	}))
	if err != nil {
		return nil, err
	}
	if response == nil || response.Msg == nil {
		return nil, errors.New("Bridge returned no result")
	}
	return response.Msg, nil
}
