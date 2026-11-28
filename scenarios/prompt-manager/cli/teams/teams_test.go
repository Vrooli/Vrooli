package teams

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"prompt-manager/internal/teamconfig"
	"prompt-manager/internal/teamcontract"
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

func TestCmdQueueClearJSONRetainsUnresolvedOwnerFields(t *testing.T) {
	ctx := &fakeContext{response: map[string]interface{}{
		"teamId": "delivery", "state": "active", "runningAgentIds": []string{"reviewer"},
		"queue": []string{}, "queuePolicy": "serialized", "maxConcurrentRuns": 1,
		"uncertainAgentIds": []string{"uncertain"}, "unreachableAgentIds": []string{"unreachable"},
		"pausedAgentIds": []string{"paused"}, "approvalPendingAgentIds": []string{"approval"},
		"needsReviewAgentIds": []string{"review"}, "watcherUnavailableAgentIds": []string{"watcher"},
		"preparationInterruptedAgentIds": []string{"interrupted"},
		"admissionDeferredAgentIds":      []string{"deferred"},
	}}
	out, err := captureTeamStdout(t, func() error {
		return cmdQueueClear(ctx, []string{"delivery", "reviewer", "--json"})
	})
	if err != nil {
		t.Fatalf("cmdQueueClear: %v", err)
	}
	ctx.assertMethodPath(t, "DELETE", "/teams/delivery/queue/running/reviewer")
	var got teamQueueStatus
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode queue status: %v\n%s", err, out)
	}
	if len(got.UncertainAgentIDs) != 1 || got.UncertainAgentIDs[0] != "uncertain" ||
		len(got.UnreachableAgentIDs) != 1 || got.UnreachableAgentIDs[0] != "unreachable" ||
		len(got.PausedAgentIDs) != 1 || got.PausedAgentIDs[0] != "paused" ||
		len(got.ApprovalPendingAgentIDs) != 1 || got.ApprovalPendingAgentIDs[0] != "approval" ||
		len(got.NeedsReviewAgentIDs) != 1 || got.NeedsReviewAgentIDs[0] != "review" ||
		len(got.WatcherUnavailableAgentIDs) != 1 || got.WatcherUnavailableAgentIDs[0] != "watcher" ||
		len(got.PreparationInterruptedAgentIDs) != 1 || got.PreparationInterruptedAgentIDs[0] != "interrupted" ||
		len(got.AdmissionDeferredAgentIDs) != 1 || got.AdmissionDeferredAgentIDs[0] != "deferred" {
		t.Fatalf("queue JSON dropped unresolved owner fields: %+v", got)
	}
}

func TestCmdQueueClearShowsAdmissionDeferredInHumanOutput(t *testing.T) {
	ctx := &fakeContext{response: map[string]interface{}{
		"teamId": "delivery", "state": "active", "runningAgentIds": []string{},
		"queue": []string{}, "admissionDeferredAgentIds": []string{"reviewer"},
	}}
	out, err := captureTeamStdout(t, func() error {
		return cmdQueueClear(ctx, []string{"delivery", "reviewer"})
	})
	if err != nil {
		t.Fatalf("cmdQueueClear: %v", err)
	}
	if !strings.Contains(out, "Admission deferred (awaiting scheduled capacity retry): [reviewer]") {
		t.Fatalf("queue-clear output hid deferred admission: %s", out)
	}
}

func TestCmdQueueClearShowsInterruptedPreparationInHumanOutput(t *testing.T) {
	ctx := &fakeContext{response: map[string]interface{}{
		"teamId": "delivery", "state": "active", "runningAgentIds": []string{},
		"queue": []string{}, "preparationInterruptedAgentIds": []string{"reviewer"},
	}}
	out, err := captureTeamStdout(t, func() error {
		return cmdQueueClear(ctx, []string{"delivery", "reviewer"})
	})
	if err != nil {
		t.Fatalf("cmdQueueClear: %v", err)
	}
	if !strings.Contains(out, "Preparation interrupted (awaiting fresh admission): [reviewer]") {
		t.Fatalf("queue-clear output hid interrupted preparation: %s", out)
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

func TestTeamMetadataCLIChangesAndClearsAreExplicit(t *testing.T) {
	ctx := &fakeContext{t: t, response: TeamDetails{Team: Team{ID: "aquila"}}}
	_, err := captureTeamStdout(t, func() error {
		return cmdCreate(ctx, []string{"Aquila", "--purpose=delivery", "--lifetime=finite", "--effort-refs=effort:aquila,effort:launch", "--home-path=.vrooli/work/delivery/aquila"})
	})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(ctx.gotPayload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["purpose"] != "delivery" || payload["homePath"] != ".vrooli/work/delivery/aquila" || payload["lifetime"] != "finite" || len(payload["effortRefs"].([]any)) != 2 {
		t.Fatalf("metadata create payload: %s", ctx.gotPayload)
	}
	_, err = captureTeamStdout(t, func() error {
		return cmdUpdate(ctx, []string{"aquila", "--purpose=", "--effort-refs=", "--home-path="})
	})
	if err != nil {
		t.Fatal(err)
	}
	payload = map[string]any{}
	if err := json.Unmarshal(ctx.gotPayload, &payload); err != nil {
		t.Fatal(err)
	}
	if value, ok := payload["purpose"]; !ok || value != "" {
		t.Fatalf("purpose clear missing: %s", ctx.gotPayload)
	}
	if value, ok := payload["effortRefs"]; !ok || len(value.([]any)) != 0 {
		t.Fatalf("reference clear missing: %s", ctx.gotPayload)
	}
	if value, ok := payload["homePath"]; !ok || value != "" {
		t.Fatalf("home path clear missing: %s", ctx.gotPayload)
	}
	if _, ok := payload["lifetime"]; ok {
		t.Fatal("omitted lifetime must not be changed")
	}
	if _, ok := payload["enabled"]; ok {
		t.Fatal("metadata must not change execution eligibility")
	}
	ctx.gotMethod = ""
	if err := cmdUpdate(ctx, []string{"aquila", "--lifetime=permanent"}); err == nil || ctx.gotMethod != "" {
		t.Fatal("invalid metadata should fail before mutation")
	}
}

func TestTeamUpdateOperatingContractFileSendsOnlyRequestedContract(t *testing.T) {
	contract := teamcontract.Minimal("effort-supervisor")
	member := contract.Members["effort-supervisor"]
	member.AllowedWrites = append(member.AllowedWrites, teamcontract.WriteRef{
		Base: teamcontract.BaseRepoRoot,
		Path: "scenarios/react-component-library/docs/internal/goal/FEEDBACK.md",
	})
	contract.Members["effort-supervisor"] = member
	raw, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(t.TempDir(), "operating-contract.json")
	if err := os.WriteFile(contractPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	ctx := &fakeContext{t: t, response: TeamDetails{Team: Team{ID: "effort-supervision"}}}
	_, err = captureTeamStdout(t, func() error {
		return cmdUpdate(ctx, []string{"effort-supervision", "--operating-contract-file=" + contractPath})
	})
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}
	ctx.assertMethodPath(t, "PUT", "/teams/effort-supervision")

	var sent UpdateTeamRequest
	if err := json.Unmarshal(ctx.gotPayload, &sent); err != nil {
		t.Fatalf("decode update payload: %v", err)
	}
	if !reflect.DeepEqual(sent.OperatingContract, contract) {
		t.Fatalf("sent operating contract differs from file: %+v", sent.OperatingContract)
	}
	if sent.Enabled != nil || sent.Archived != nil || sent.Runtime != nil || sent.Coordination != nil || sent.Execution != nil {
		t.Fatalf("contract-only update changed unrelated fields: %+v", sent)
	}
}

func TestTeamUpdateRejectsMalformedOperatingContractBeforeRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"schemaVersion":`},
		{name: "null", body: `null`},
		{name: "unknown field", body: `{"schemaVersion":1,"unexpected":true}`},
		{name: "multiple values", body: `{} {}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contractPath := filepath.Join(t.TempDir(), "operating-contract.json")
			if err := os.WriteFile(contractPath, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx := &fakeContext{t: t}
			err := cmdUpdate(ctx, []string{"effort-supervision", "--operating-contract-file=" + contractPath})
			if err == nil {
				t.Fatal("cmdUpdate unexpectedly accepted malformed contract JSON")
			}
			if ctx.gotMethod != "" {
				t.Fatalf("malformed contract must fail before request, got %s %s", ctx.gotMethod, ctx.gotPath)
			}
		})
	}
}

func TestTeamMetadataCLIListFiltersAndRetainsObjectiveRecords(t *testing.T) {
	ctx := &fakeContext{t: t, response: []Team{
		{ID: "marketing", Purpose: "domain-stewardship", Lifetime: "standing"},
		{ID: "supervisor", Purpose: "supervision", Lifetime: "standing", HomePath: ".vrooli/work/delivery/aquila", ObjectivesServed: []teamconfig.ObjectiveDeclaration{{ID: "T1", AcknowledgedRevision: "r3"}}},
		{ID: "aquila", Purpose: "delivery", Lifetime: "finite"},
		{ID: "legacy"},
	}}
	output, err := captureTeamStdout(t, func() error { return cmdList(ctx, []string{"--purpose=supervision", "--lifetime=standing", "--json"}) })
	if err != nil {
		t.Fatal(err)
	}
	var teams []Team
	if err := json.Unmarshal([]byte(output), &teams); err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].ID != "supervisor" || teams[0].HomePath != ".vrooli/work/delivery/aquila" || teams[0].ObjectivesServed[0].AcknowledgedRevision != "r3" {
		t.Fatalf("filter/read metadata lost: %s", output)
	}
	output, err = captureTeamStdout(t, func() error { return cmdList(ctx, []string{"--lifetime=unspecified", "--json"}) })
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(output), &teams); err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].ID != "legacy" {
		t.Fatalf("unspecified filter: %s", output)
	}
}

func TestTeamRolesSetUsesCompleteRoleFileAndRejectsInvalidInputBeforeRequest(t *testing.T) {
	tempDir := t.TempDir()
	rolesFile := filepath.Join(tempDir, "roles.json")
	if err := os.WriteFile(rolesFile, []byte(`[{"id":"lead","name":"Lead","description":"Owns delivery"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := &fakeContext{t: t, response: []Role{{ID: "lead", Name: "Lead", Description: "Owns delivery"}}}
	if err := route(ctx, []string{"roles-set", "team-1", "--file", rolesFile, "--json"}); err != nil {
		t.Fatalf("team roles-set route: %v", err)
	}
	ctx.assertMethodPath(t, "PUT", "/teams/team-1/roles")
	var payload struct {
		Roles []Role `json:"roles"`
	}
	if err := json.Unmarshal(ctx.gotPayload, &payload); err != nil {
		t.Fatalf("decode request payload: %v", err)
	}
	if !reflect.DeepEqual(payload.Roles, []Role{{ID: "lead", Name: "Lead", Description: "Owns delivery"}}) {
		t.Fatalf("roles payload = %+v", payload.Roles)
	}

	for _, tc := range []struct {
		name string
		args []string
		file string
	}{
		{name: "missing file flag", args: []string{"team-1"}},
		{name: "malformed file", args: []string{"team-1", "--file", rolesFile}, file: "{"},
		{name: "null is not an explicit role list", args: []string{"team-1", "--file", rolesFile}, file: "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.file != "" {
				if err := os.WriteFile(rolesFile, []byte(tc.file), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			refused := &fakeContext{t: t}
			if err := route(refused, append([]string{"roles-set"}, tc.args...)); err == nil {
				t.Fatal("invalid command input was accepted")
			}
			if refused.gotMethod != "" {
				t.Fatalf("invalid input issued %s %s", refused.gotMethod, refused.gotPath)
			}
		})
	}
}

func TestHeartbeatEnableCanReapplyDefaultScheduleOnExistingConfig(t *testing.T) {
	ctx := &fakeContext{getResponse: HeartbeatConfig{TeamID: "arbitrary-team", AgentID: "leader", Schedule: "*/5 * * * *"}}
	err := cmdHeartbeatEnable(ctx, []string{"arbitrary-team", "leader", "--schedule=0 */6 * * *"})
	if err != nil {
		t.Fatal(err)
	}
	var req UpdateHeartbeatRequest
	if err := json.Unmarshal(ctx.gotPayload, &req); err != nil {
		t.Fatal(err)
	}
	if req.Schedule == nil || *req.Schedule != "0 */6 * * *" {
		t.Fatalf("explicit default schedule was not persisted: %+v", req.Schedule)
	}
}

func TestHeartbeatJSONPreservesAdmissionReasonFromOwnerResponse(t *testing.T) {
	ctx := &fakeContext{getResponse: map[string]any{
		"teamId":  "effort-supervision",
		"agentId": "effort-supervisor",
		"schedulerState": map[string]any{
			"running": true, "registered": 1, "last_callback": "2026-10-20T22:00:00Z",
		},
		"schedulerRegistered": true,
		"lastExecution": map[string]any{
			"status":          "failed",
			"admissionReason": "grant_limit_reached",
		},
		"lastSuccessfulExecution": map[string]any{
			"status":          "completed",
			"admissionReason": "authority_state_unavailable",
		},
	}}
	output, err := captureTeamStdout(t, func() error {
		return cmdHeartbeat(ctx, []string{"effort-supervision", "effort-supervisor", "--json"})
	})
	if err != nil {
		t.Fatalf("cmdHeartbeat: %v", err)
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode heartbeat JSON: %v\n%s", err, output)
	}
	for field, want := range map[string]string{
		"lastExecution":           "grant_limit_reached",
		"lastSuccessfulExecution": "authority_state_unavailable",
	} {
		execution, ok := response[field].(map[string]any)
		if !ok {
			t.Fatalf("%s = %#v, want execution object", field, response[field])
		}
		if got := execution["admissionReason"]; got != want {
			t.Errorf("%s.admissionReason = %#v, want %q", field, got, want)
		}
	}
	state, ok := response["schedulerState"].(map[string]any)
	if !ok {
		t.Fatalf("schedulerState = %#v, want scheduler runtime metadata", response["schedulerState"])
	}
	if state["last_callback"] != "2026-10-20T22:00:00Z" || state["running"] != true || state["registered"] != float64(1) || response["schedulerRegistered"] != true {
		t.Fatalf("scheduler state was not preserved: %#v registered=%#v", state, response["schedulerRegistered"])
	}
	ctx.assertMethodPath(t, "GET", "/teams/effort-supervision/heartbeats/effort-supervisor")
}

func TestHeartbeatAttemptsUsesBoundedExactMemberFiltersAndPreservesJSON(t *testing.T) {
	ctx := &fakeContext{t: t, getResponse: map[string]any{
		"attempts": []map[string]any{{
			"id": "attempt-1", "teamId": "rcl-delivery", "agentId": "orchestrator",
			"status": "failed", "phase": "dispatch", "newEvidence": map[string]any{"source": "owner"},
		}},
		"total": 3, "totalExact": false, "hasMore": true, "scannedEvents": 12,
		"windowNote": "bounded event scan",
	}}
	output, err := captureTeamStdout(t, func() error {
		return route(ctx, []string{"heartbeat-attempts", "rcl-delivery", "orchestrator", "--limit", "7", "--offset", "4", "--json"})
	})
	if err != nil {
		t.Fatalf("heartbeat-attempts: %v", err)
	}
	ctx.assertMethodPath(t, "GET", "/heartbeat-attempts?agent_id=orchestrator&limit=7&offset=4&team_id=rcl-delivery")
	var response struct {
		Attempts      []map[string]any `json:"attempts"`
		Total         int              `json:"total"`
		TotalExact    bool             `json:"totalExact"`
		HasMore       bool             `json:"hasMore"`
		ScannedEvents int              `json:"scannedEvents"`
		WindowNote    string           `json:"windowNote"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON response: %v\n%s", err, output)
	}
	if response.Total != 3 || response.TotalExact || !response.HasMore || response.ScannedEvents != 12 || response.WindowNote != "bounded event scan" || len(response.Attempts) != 1 {
		t.Fatalf("response metadata = %+v", response)
	}
	if got := response.Attempts[0]["newEvidence"].(map[string]any)["source"]; got != "owner" {
		t.Fatalf("unrecognized server field was dropped: %#v", response.Attempts[0])
	}
}

func TestHeartbeatAttemptsRefusesInvalidBoundsAndPreservesOwnerErrors(t *testing.T) {
	for _, args := range [][]string{{"--limit", "0"}, {"--limit", "201"}, {"--offset", "-1"}} {
		t.Run(strings.Join(args, "="), func(t *testing.T) {
			ctx := &fakeContext{t: t}
			if err := route(ctx, append([]string{"heartbeat-attempts", "team", "member"}, args...)); err == nil {
				t.Fatalf("invalid bounds were accepted: %v", args)
			}
			if ctx.gotMethod != "" {
				t.Fatalf("invalid limit issued %s %s", ctx.gotMethod, ctx.gotPath)
			}
		})
	}

	denied := errors.New("verified caller identity unavailable")
	ctx := &fakeContext{t: t, err: denied}
	if err := route(ctx, []string{"heartbeat-attempts", "team", "member", "--limit=1", "--json"}); !errors.Is(err, denied) {
		t.Fatalf("owner error was hidden: %v", err)
	}
	ctx.assertMethodPath(t, "GET", "/heartbeat-attempts?agent_id=member&limit=1&offset=0&team_id=team")
}

func TestExecutionStatusUsesBoundedOwnerReadAndPreservesJSON(t *testing.T) {
	ctx := &fakeContext{t: t, getResponse: map[string]any{
		"teamId":          "nooch-delivery",
		"state":           "active",
		"runningAgentIds": []string{"nooch-delivery-orchestrator"},
		"owners": []map[string]any{{
			"agentId": "nooch-delivery-orchestrator", "phase": "retained", "state": "dispatch_uncertain",
			"dispatchStage": "task_intent", "dispatchFailurePhase": "task_resolution",
			"dispatchFailureCode": "owner_forbidden", "taskId": "task-123", "runId": "",
		}},
	}}
	output, err := captureTeamStdout(t, func() error {
		return route(ctx, []string{"execution-status", "nooch-delivery", "--json"})
	})
	if err != nil {
		t.Fatalf("execution-status: %v", err)
	}
	ctx.assertMethodPath(t, "GET", "/teams/nooch-delivery/execution-status")
	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON response: %v\n%s", err, output)
	}
	owners, ok := response["owners"].([]any)
	if !ok || len(owners) != 1 {
		t.Fatalf("owners = %#v, want one owner record", response["owners"])
	}
	owner := owners[0].(map[string]any)
	if owner["taskId"] != "task-123" || owner["dispatchStage"] != "task_intent" ||
		owner["dispatchFailurePhase"] != "task_resolution" || owner["dispatchFailureCode"] != "owner_forbidden" {
		t.Fatalf("owner metadata was not preserved: %#v", owner)
	}

	human, err := captureTeamStdout(t, func() error {
		return route(ctx, []string{"execution-status", "nooch-delivery"})
	})
	if err != nil {
		t.Fatalf("human execution-status: %v", err)
	}
	if !strings.Contains(human, "Owner nooch-delivery-orchestrator (retained) state=dispatch_uncertain dispatch=task_intent failure=task_resolution/owner_forbidden task=task-123") {
		t.Fatalf("human output did not explain retained owner metadata: %s", human)
	}
}

func TestExecutionStatusPreservesReadFailureAndRejectsEmptyTeam(t *testing.T) {
	denied := errors.New("verified caller identity unavailable")
	ctx := &fakeContext{t: t, err: denied}
	if err := route(ctx, []string{"execution-status", "team", "--json"}); !errors.Is(err, denied) {
		t.Fatalf("owner read error was hidden: %v", err)
	}
	ctx.assertMethodPath(t, "GET", "/teams/team/execution-status")

	ctx = &fakeContext{t: t}
	if err := route(ctx, []string{"execution-status", "  "}); err == nil {
		t.Fatal("empty team ID was accepted")
	}
	if ctx.gotMethod != "" {
		t.Fatalf("empty team ID issued %s %s", ctx.gotMethod, ctx.gotPath)
	}
}

func TestHeartbeatDisableCreatesExplicitOnDemandConfiguration(t *testing.T) {
	ctx := &fakeContext{response: HeartbeatConfig{TeamID: "delivery", AgentID: "reviewer", Enabled: false, ProfileKey: "owner/inspection"}}
	if err := cmdHeartbeatDisable(ctx, []string{"delivery", "reviewer", "--on-demand", "--profile=owner/inspection", "--json"}); err != nil {
		t.Fatal(err)
	}
	ctx.assertMethodPath(t, "POST", "/teams/delivery/heartbeats/reviewer")
	var request map[string]any
	if err := json.Unmarshal(ctx.gotPayload, &request); err != nil {
		t.Fatal(err)
	}
	if request["enabled"] != false || request["schedule"] != "" || request["profileKey"] != "owner/inspection" {
		t.Fatalf("reviewer got a timer or wrong profile: %s", ctx.gotPayload)
	}
	refusal := errors.New("permission denied")
	ctx = &fakeContext{err: refusal}
	if err := cmdHeartbeatDisable(ctx, []string{"delivery", "reviewer", "--on-demand", "--profile=owner/inspection"}); !errors.Is(err, refusal) {
		t.Fatalf("owner refusal lost: %v", err)
	}
	ctx.assertMethodPath(t, "POST", "/teams/delivery/heartbeats/reviewer")
}
func TestHeartbeatDisableOnDemandRequiresExplicitProfileAndOwnerReadback(t *testing.T) {
	ctx := &fakeContext{}
	if err := cmdHeartbeatDisable(ctx, []string{"delivery", "reviewer", "--on-demand"}); err == nil || ctx.gotMethod != "" {
		t.Fatal("missing profile reached owner")
	}
	ctx = &fakeContext{response: HeartbeatConfig{TeamID: "delivery", AgentID: "reviewer", Enabled: true, ProfileKey: "owner/inspection"}}
	if err := cmdHeartbeatDisable(ctx, []string{"delivery", "reviewer", "--on-demand", "--profile=owner/inspection"}); err == nil {
		t.Fatal("enabled owner response accepted as on-demand")
	}
}
