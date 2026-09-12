package teams

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"prompt-manager/internal/teamconfig"
)

type fleetHealthContext struct {
	fakeContext
	teams     []Team
	configs   map[string][]HeartbeatConfig
	knowledge map[string]KnowledgeListResponse
	products  map[string]bool
}

func (f *fleetHealthContext) HasDurableProduct(teamID, runID string, cutoff time.Time) (bool, error) {
	return f.products[teamID+":"+runID], nil
}

func stringPtr(value string) *string { return &value }

func (f *fleetHealthContext) Get(path string, result interface{}) error {
	var value interface{}
	switch path {
	case "/teams":
		value = f.teams
	default:
		const prefix = "/teams/"
		const suffix = "/heartbeats"
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
			teamID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
			value = f.configs[teamID]
		} else if strings.HasPrefix(path, prefix) && strings.Contains(path, "/knowledge") {
			teamID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/knowledge?last=100")
			value = f.knowledge[teamID]
		} else {
			return f.fakeContext.Get(path, result)
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

func parseTeamFlagsForTest(t *testing.T, includeDefaults bool, args ...string) teamConfigFlagSet {
	t.Helper()

	fs := flag.NewFlagSet("teams-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flags := registerTeamConfigFlags(fs, includeDefaults)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("Parse(%v) error = %v", args, err)
	}

	return flags
}

func TestResolveCreateTeamConfigDefaultsToIndependentMultiProcess(t *testing.T) {
	flags := parseTeamFlagsForTest(t, true)

	runtime, coordination, execution, err := resolveCreateTeamConfig(flags)
	if err != nil {
		t.Fatalf("resolveCreateTeamConfig() error = %v", err)
	}

	if runtime.Mode != teamconfig.RuntimeModeMultiProcess {
		t.Fatalf("runtime.mode = %q, want %q", runtime.Mode, teamconfig.RuntimeModeMultiProcess)
	}
	if coordination.Pattern != teamconfig.CoordinationPatternIndependent {
		t.Fatalf("coordination.pattern = %q, want %q", coordination.Pattern, teamconfig.CoordinationPatternIndependent)
	}
	if coordination.Capabilities != teamconfig.DefaultIndependentCapabilities() {
		t.Fatalf(
			"coordination.capabilities = %+v, want %+v",
			coordination.Capabilities,
			teamconfig.DefaultIndependentCapabilities(),
		)
	}
	if execution.QueuePolicy != teamconfig.QueuePolicyBoundedParallel || execution.MaxConcurrentRuns != 2 {
		t.Fatalf("execution = %+v, want bounded-parallel/2", execution)
	}
}

func TestResolveCreateTeamConfigPromotesSingleProcessToLeaderLed(t *testing.T) {
	flags := parseTeamFlagsForTest(
		t,
		true,
		"--runtime-mode=single-process",
		"--lead-agent-id=director",
	)

	runtime, coordination, execution, err := resolveCreateTeamConfig(flags)
	if err != nil {
		t.Fatalf("resolveCreateTeamConfig() error = %v", err)
	}

	if runtime.Mode != teamconfig.RuntimeModeSingleProcess {
		t.Fatalf("runtime.mode = %q, want %q", runtime.Mode, teamconfig.RuntimeModeSingleProcess)
	}
	if coordination.Pattern != teamconfig.CoordinationPatternLeaderLed {
		t.Fatalf("coordination.pattern = %q, want %q", coordination.Pattern, teamconfig.CoordinationPatternLeaderLed)
	}
	if coordination.LeadAgentID != "director" {
		t.Fatalf("coordination.leadAgentId = %q, want %q", coordination.LeadAgentID, "director")
	}
	if coordination.MessagingMode != teamconfig.MessagingModeInSession {
		t.Fatalf("coordination.messagingMode = %q, want %q", coordination.MessagingMode, teamconfig.MessagingModeInSession)
	}
	if execution.QueuePolicy != teamconfig.QueuePolicySerialized || execution.MaxConcurrentRuns != 1 {
		t.Fatalf("execution = %+v, want serialized/1", execution)
	}
}

func TestResolveUpdatedTeamConfigPromotesRuntimeTransitions(t *testing.T) {
	flags := parseTeamFlagsForTest(
		t,
		false,
		"--runtime-mode=single-process",
		"--lead-agent-id=director",
	)

	current := TeamDetails{
		Team: Team{
			Runtime: Runtime{Mode: teamconfig.RuntimeModeMultiProcess},
			Coordination: Coordination{
				Pattern:       teamconfig.CoordinationPatternIndependent,
				ReportingMode: teamconfig.ReportingModeNone,
				MessagingMode: teamconfig.MessagingModeDisabled,
				Capabilities:  teamconfig.DefaultIndependentCapabilities(),
			},
			Execution: Execution{
				QueuePolicy:       teamconfig.QueuePolicyBoundedParallel,
				MaxConcurrentRuns: 3,
			},
		},
	}

	runtime, coordination, execution, err := resolveUpdatedTeamConfig(current, flags)
	if err != nil {
		t.Fatalf("resolveUpdatedTeamConfig() error = %v", err)
	}

	if runtime == nil || runtime.Mode != teamconfig.RuntimeModeSingleProcess {
		t.Fatalf("runtime = %+v, want single-process", runtime)
	}
	if coordination == nil || coordination.Pattern != teamconfig.CoordinationPatternLeaderLed {
		t.Fatalf("coordination = %+v, want leader-led", coordination)
	}
	if coordination.LeadAgentID != "director" {
		t.Fatalf("coordination.leadAgentId = %q, want %q", coordination.LeadAgentID, "director")
	}
	if execution == nil || execution.QueuePolicy != teamconfig.QueuePolicySerialized || execution.MaxConcurrentRuns != 1 {
		t.Fatalf("execution = %+v, want serialized/1", execution)
	}
}

func TestResolveUpdatedTeamConfigAppliesCapabilityOverrides(t *testing.T) {
	flags := parseTeamFlagsForTest(
		t,
		false,
		"--show-org-context=false",
		"--require-handoff=false",
	)

	current := TeamDetails{
		Team: Team{
			Runtime: Runtime{Mode: teamconfig.RuntimeModeMultiProcess},
			Coordination: Coordination{
				Pattern:       teamconfig.CoordinationPatternPeer,
				ReportingMode: teamconfig.ReportingModeOrgChart,
				MessagingMode: teamconfig.MessagingModeAsyncInbox,
				Capabilities:  teamconfig.DefaultPeerCapabilities(),
			},
			Execution: Execution{
				QueuePolicy:       teamconfig.QueuePolicyBoundedParallel,
				MaxConcurrentRuns: 2,
			},
		},
	}

	_, coordination, _, err := resolveUpdatedTeamConfig(current, flags)
	if err != nil {
		t.Fatalf("resolveUpdatedTeamConfig() error = %v", err)
	}

	if coordination == nil {
		t.Fatal("expected coordination update")
	}
	if coordination.Capabilities.ShowOrgContext {
		t.Fatal("expected showOrgContext override to be false")
	}
	if coordination.Capabilities.RequireHandoff {
		t.Fatal("expected requireHandoff override to be false")
	}
}

func captureTeamStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writePipe
	runErr := fn()
	_ = writePipe.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, readPipe); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	_ = readPipe.Close()
	return buf.String(), runErr
}

func TestCmdHeartbeatFleetHealthRetainsRecentCompletionWhileMemberIsRunning(t *testing.T) {
	now := time.Now().UTC()
	ctx := &fleetHealthContext{
		teams: []Team{{ID: "director-swarm", Enabled: true}},
		configs: map[string][]HeartbeatConfig{
			"director-swarm": {{
				TeamID:  "director-swarm",
				AgentID: "vision-walk-prep",
				Enabled: true,
				LastExecution: &HeartbeatExecResult{
					Status:    "running",
					StartedAt: now.Format(time.RFC3339),
				},
				LastSuccessfulExecution: &HeartbeatExecResult{
					Status:    "completed",
					StartedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
					EndedAt:   now.Add(-time.Hour).Format(time.RFC3339),
					RunID:     "run-success",
				},
			}},
		},
		knowledge: map[string]KnowledgeListResponse{"director-swarm": {Entries: []KnowledgeEntry{{At: now.Add(-time.Hour).Format(time.RFC3339), Attribution: AttributionInfo{RunID: stringPtr("run-success")}}}}},
	}

	out, err := captureTeamStdout(t, func() error {
		return cmdHeartbeatFleetHealth(ctx, []string{"--json"})
	})
	if err != nil {
		t.Fatalf("cmdHeartbeatFleetHealth: %v", err)
	}
	var summary HeartbeatFleetHealth
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatalf("decode fleet health: %v\n%s", err, out)
	}
	if summary.EnabledMembers != 1 || summary.SucceededMembers != 1 {
		t.Fatalf("fleet health = %+v, want the in-flight member's recent completion retained", summary)
	}
	if summary.SuccessPercent != 100 || summary.ThresholdPercent != 90 || !summary.MeetsThreshold {
		t.Fatalf("fleet threshold = %+v, want exact 100%% pass against 90%%", summary)
	}
}

func TestCmdHeartbeatFleetHealthBlocksCompletionWithoutDurableWrite(t *testing.T) {
	now := time.Now().UTC()
	ctx := &fleetHealthContext{
		teams:     []Team{{ID: "infra-health", Enabled: true}},
		configs:   map[string][]HeartbeatConfig{"infra-health": {{Enabled: true, LastExecution: &HeartbeatExecResult{Status: "completed", RunID: "run-no-write", StartedAt: now.Add(-time.Hour).Format(time.RFC3339), EndedAt: now.Add(-30 * time.Minute).Format(time.RFC3339)}}}},
		knowledge: map[string]KnowledgeListResponse{"infra-health": {}},
	}
	out, err := captureTeamStdout(t, func() error { return cmdHeartbeatFleetHealth(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatal(err)
	}
	var summary HeartbeatFleetHealth
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.SucceededMembers != 1 || summary.ProducedMembers != 0 || summary.BlockedMembers != 1 || summary.FailedMembers != 0 {
		t.Fatalf("summary = %+v, want completed-but-unwritten member blocked", summary)
	}
}

func TestCmdHeartbeatFleetHealthCountsSourceLedgerReceipt(t *testing.T) {
	now := time.Now().UTC()
	ctx := &fleetHealthContext{
		teams:     []Team{{ID: "infra-health", Enabled: true}},
		configs:   map[string][]HeartbeatConfig{"infra-health": {{Enabled: true, LastExecution: &HeartbeatExecResult{Status: "completed", RunID: "run-ledger", StartedAt: now.Add(-time.Hour).Format(time.RFC3339), EndedAt: now.Add(-30 * time.Minute).Format(time.RFC3339)}}}},
		knowledge: map[string]KnowledgeListResponse{"infra-health": {}},
		products:  map[string]bool{"infra-health:run-ledger": true},
	}
	out, err := captureTeamStdout(t, func() error { return cmdHeartbeatFleetHealth(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatal(err)
	}
	var summary HeartbeatFleetHealth
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.ProducedMembers != 1 || summary.BlockedMembers != 0 {
		t.Fatalf("summary = %+v, want Source Ledger receipt counted", summary)
	}
}

func TestCmdPromptPreviewCallsFullPreviewEndpoint(t *testing.T) {
	fc := &fakeContext{
		t:        t,
		response: PromptPreviewResponse{TeamID: "team-a", AgentID: "agent-a", Prompt: "# Heartbeat Task\n\nDo work"},
	}

	out, err := captureTeamStdout(t, func() error {
		return cmdPromptPreview(fc, []string{"team-a", "agent-a"})
	})
	if err != nil {
		t.Fatalf("cmdPromptPreview: %v", err)
	}
	fc.assertMethodPath(t, "POST", "/prompt-preview")
	var payload PromptPreviewRequest
	if err := json.Unmarshal(fc.gotPayload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.TeamID != "team-a" || payload.AgentID != "agent-a" {
		t.Fatalf("payload = %+v", payload)
	}
	if !strings.Contains(out, "# Heartbeat Task") {
		t.Fatalf("expected full prompt output, got:\n%s", out)
	}
}

func TestCmdPromptPreviewStructuredFormatsSections(t *testing.T) {
	fc := &fakeContext{
		t: t,
		response: StructuredPromptPreviewResponse{
			TeamID:  "team-a",
			AgentID: "agent-a",
			Sections: []PromptSection{{
				Kind:    "operating-policy-team",
				Label:   "Operating Policy (Team)",
				Content: "# Operating Policy (Team)\n\nTeam policy.",
			}},
		},
	}

	out, err := captureTeamStdout(t, func() error {
		return cmdPromptPreviewStructured(fc, []string{"team-a", "agent-a"})
	})
	if err != nil {
		t.Fatalf("cmdPromptPreviewStructured: %v", err)
	}
	fc.assertMethodPath(t, "POST", "/prompt-preview-structured")
	if !strings.Contains(out, "Kind: operating-policy-team") || !strings.Contains(out, "# Operating Policy (Team)") {
		t.Fatalf("unexpected structured output:\n%s", out)
	}
}

func TestCmdPromptMatrixUsesBackendOrder(t *testing.T) {
	fc := &fakeContext{
		t: t,
		response: TeamPromptMatrixResponse{
			TeamID: "team-a",
			Entries: []TeamPromptMatrixEntry{{
				AgentID:     "agent-a",
				DisplayName: "Agent A",
				Sections: []PromptSection{
					{Kind: "agent-file", Label: "SOUL.md", Content: "abc"},
					{Kind: "operating-policy-team", Label: "Operating Policy (Team)", Content: "abcdef"},
				},
			}},
		},
	}

	out, err := captureTeamStdout(t, func() error {
		return cmdPromptMatrix(fc, []string{"team-a"})
	})
	if err != nil {
		t.Fatalf("cmdPromptMatrix: %v", err)
	}
	fc.assertMethodPath(t, "GET", "/teams/team-a/prompt-matrix")
	header := "Member\tagent-file\toperating-policy-team"
	if !strings.Contains(out, header) {
		t.Fatalf("expected backend-order header %q, got:\n%s", header, out)
	}
}

func TestUsageDescribesMemberContextAsTaskless(t *testing.T) {
	usage := usageText()
	if !strings.Contains(usage, "Get standing member context without HEARTBEAT.md") {
		t.Fatalf("member-context help should describe taskless context:\n%s", usage)
	}
	if strings.Contains(usage, "Get full"+" member context prompt") {
		t.Fatalf("usage contains stale member-context wording:\n%s", usage)
	}
}
