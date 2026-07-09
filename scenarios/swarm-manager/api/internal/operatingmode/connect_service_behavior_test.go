package operatingmode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// newConnectClientForService mounts the Connect handler over a caller-built
// *Service and returns a generated client. Tests that need to seed the store or
// mutate the service (e.g. drop the prompt client to exercise the degraded path)
// use this instead of newConnectTestClient, which hides the service.
func newConnectClientForService(t *testing.T, svc *Service) apiconnect.OperatingModeServiceClient {
	t.Helper()
	path, handler := apiconnect.NewOperatingModeServiceHandler(NewConnectService(svc))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return apiconnect.NewOperatingModeServiceClient(server.Client(), server.URL)
}

// TestConnectCatalogCarriesDecisionMetadata pins the catalog wire contract's
// decision-metadata + usage fields (best_for/not_for/tradeoffs, usage_count) so
// the projection can't silently drop them — coverage inherited from the deleted
// REST catalog test.
func TestConnectCatalogCarriesDecisionMetadata(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})
	resp, err := client.Catalog(context.Background(), connect.NewRequest(&apipb.OperatingModeCatalogRequest{}))
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(resp.Msg.GetModes()) != len(Modes()) {
		t.Fatalf("catalog modes len = %d, want %d", len(resp.Msg.GetModes()), len(Modes()))
	}
	var sawUsage bool
	for _, entry := range resp.Msg.GetModes() {
		if len(entry.GetBestFor()) == 0 {
			t.Errorf("mode %q missing best_for", entry.GetMode())
		}
		if len(entry.GetNotFor()) == 0 {
			t.Errorf("mode %q missing not_for", entry.GetMode())
		}
		if len(entry.GetTradeoffs()) == 0 {
			t.Errorf("mode %q missing tradeoffs", entry.GetMode())
		}
		if entry.GetUsageCount() > 0 {
			sawUsage = true
		}
	}
	// init-a (a default fake initiative) is bound to holistic-loop, so at least
	// one mode reports a non-zero usage count.
	if !sawUsage {
		t.Errorf("expected at least one mode to report a non-zero usage_count")
	}
}

// TestConnectGetModeReturnsLinkedInitiatives proves GetMode filters linked
// initiatives to the requested mode and projects the full phase graph, including
// the generic replan guard edge.
func TestConnectGetModeReturnsLinkedInitiatives(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"loop-init":  {Name: "loop-init", Title: "Loop Initiative", Mode: string(ModeHolisticLoop)},
			"drain-init": {Name: "drain-init", Title: "Drain Initiative", Mode: string(ModePhasedPlanDrain)},
		}},
	})

	resp, err := client.GetMode(context.Background(), connect.NewRequest(&apipb.OperatingModeGetRequest{Mode: "holistic-loop"}))
	if err != nil {
		t.Fatalf("GetMode: %v", err)
	}
	names := map[string]bool{}
	for _, init := range resp.Msg.GetLinkedInitiatives() {
		names[init.GetName()] = true
	}
	if !names["loop-init"] {
		t.Fatalf("missing linked initiative loop-init: %+v", resp.Msg.GetLinkedInitiatives())
	}
	if names["drain-init"] {
		t.Fatalf("should not include initiatives bound to other modes: %+v", resp.Msg.GetLinkedInitiatives())
	}

	graph := resp.Msg.GetEntry().GetPhaseGraph()
	if graph.GetStartPhase() != "investigate" {
		t.Fatalf("start phase = %q, want investigate", graph.GetStartPhase())
	}
	if len(resp.Msg.GetEntry().GetPhases()) == 0 {
		t.Fatalf("entry missing projected phases")
	}
	for _, phase := range resp.Msg.GetEntry().GetPhases() {
		if phase.GetSkillId() == "" {
			t.Fatalf("phase %q missing resolved skill_id", phase.GetPhase())
		}
	}
	var foundReplan bool
	for _, edge := range graph.GetTransitions() {
		if edge.GetFrom() == "execute" && edge.GetTo() == "investigate" && strings.Contains(edge.GetLabel(), "replan_needed") {
			foundReplan = true
			break
		}
	}
	if !foundReplan {
		t.Fatalf("expected execute->investigate replan edge in %+v", graph.GetTransitions())
	}
}

// TestConnectSimulateBlockedPresetIsTerminal walks a branch-covering preset over
// Connect and asserts it ends at the guarded-stop terminal — coverage inherited
// from the deleted REST simulate test.
func TestConnectSimulateBlockedPresetIsTerminal(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})
	resp, err := client.SimulateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeSimulateRequest{
		Mode:   "phased-plan-drain",
		Preset: "blocked",
	}))
	if err != nil {
		t.Fatalf("SimulateMode: %v", err)
	}
	if resp.Msg.GetActivePreset() != "blocked" {
		t.Fatalf("active preset = %q, want blocked", resp.Msg.GetActivePreset())
	}
	trace := resp.Msg.GetTrace()
	if len(trace) == 0 {
		t.Fatalf("blocked trace should be non-empty")
	}
	last := trace[len(trace)-1]
	if last.GetPhase() != "classify_progress" || !last.GetTerminal() {
		t.Fatalf("blocked terminal step = %q terminal=%v, want classify_progress terminal", last.GetPhase(), last.GetTerminal())
	}
}

func TestConnectRenderSimulationPrompt(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{prompts: &fakePrompts{render: echoRender}})

	resp, err := client.RenderSimulationPrompt(context.Background(), connect.NewRequest(&apipb.OperatingModeRenderSimulationRequest{
		Mode:      "holistic-loop",
		Preset:    "happy-path",
		StepIndex: 0,
	}))
	if err != nil {
		t.Fatalf("RenderSimulationPrompt: %v", err)
	}
	if resp.Msg.GetDegraded() {
		t.Fatalf("degraded unexpectedly: %s", resp.Msg.GetDegradedReason())
	}
	if resp.Msg.GetPhase() == "" || resp.Msg.GetSkillId() == "" || resp.Msg.GetPrompt() == "" {
		t.Fatalf("render response missing phase/skill/prompt: %+v", resp.Msg)
	}
	if !strings.Contains(resp.Msg.GetPrompt(), "Unify the audio-session lifecycle") {
		t.Fatalf("prompt missing substituted title: %s", resp.Msg.GetPrompt())
	}

	// An out-of-range step is a client error, not a rendered prompt.
	_, err = client.RenderSimulationPrompt(context.Background(), connect.NewRequest(&apipb.OperatingModeRenderSimulationRequest{
		Mode:      "holistic-loop",
		StepIndex: 999,
	}))
	if err == nil {
		t.Fatalf("expected out-of-range step to error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("out-of-range code = %v, want InvalidArgument; err=%v", got, err)
	}
}

// TestConnectRenderSimulationPromptDegrades proves the render surface returns a
// typed degraded response (not a transport error) when the prompt-manager seam
// is unavailable.
func TestConnectRenderSimulationPromptDegrades(t *testing.T) {
	svc := newTestServiceWithOptions(t, t.TempDir(), serviceOptions{prompts: &fakePrompts{render: echoRender}})
	svc.prompts = nil
	client := newConnectClientForService(t, svc)

	resp, err := client.RenderSimulationPrompt(context.Background(), connect.NewRequest(&apipb.OperatingModeRenderSimulationRequest{
		Mode:      "holistic-loop",
		StepIndex: 0,
	}))
	if err != nil {
		t.Fatalf("RenderSimulationPrompt: %v", err)
	}
	if !resp.Msg.GetDegraded() || resp.Msg.GetPrompt() != "" || len(resp.Msg.GetVariables()) == 0 {
		t.Fatalf("degraded response = %+v, want degraded with variables and no prompt", resp.Msg)
	}
}

func TestConnectUpdateModeAppliesOverlay(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})
	ctx := context.Background()

	label := "Renamed Loop"
	desc := "Tighter wording."
	upd, err := client.UpdateMode(ctx, connect.NewRequest(&apipb.OperatingModeUpdateRequest{
		Mode: "holistic-loop", Label: &label, Description: &desc,
	}))
	if err != nil {
		t.Fatalf("UpdateMode: %v", err)
	}
	if upd.Msg.GetEntry().GetLabel() != "Renamed Loop" {
		t.Fatalf("updated label = %q, want Renamed Loop", upd.Msg.GetEntry().GetLabel())
	}

	// A subsequent GetMode reflects the persisted overlay.
	got, err := client.GetMode(ctx, connect.NewRequest(&apipb.OperatingModeGetRequest{Mode: "holistic-loop"}))
	if err != nil {
		t.Fatalf("GetMode: %v", err)
	}
	if got.Msg.GetEntry().GetLabel() != "Renamed Loop" {
		t.Fatalf("GetMode after update = %q, want Renamed Loop", got.Msg.GetEntry().GetLabel())
	}
}

func TestConnectUpdateModeRejectsBlankLabel(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})
	blank := ""
	_, err := client.UpdateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeUpdateRequest{
		Mode: "holistic-loop", Label: &blank,
	}))
	if err == nil {
		t.Fatalf("expected blank label to error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument; err=%v", got, err)
	}
}

func TestConnectUpdateModeRejectsUnknownMode(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})
	label := "x"
	_, err := client.UpdateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeUpdateRequest{
		Mode: "does-not-exist", Label: &label,
	}))
	if err == nil {
		t.Fatalf("expected unknown mode to error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound; err=%v", got, err)
	}
}

// TestConnectCompleteItemsRejectsItemLevelInitiative proves a round action
// against an item-level initiative (which has no phase rounds) is rejected as an
// invalid argument rather than silently acting.
func TestConnectCompleteItemsRejectsItemLevelInitiative(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"item-init": {Name: "item-init", Title: "Item Init", Mode: string(ModeItemLevel)},
		}},
		backlogMutator: &fakeBacklogMutator{},
	})
	_, err := client.CompleteItems(context.Background(), connect.NewRequest(&apipb.OperatingModeCompleteItemsRequest{
		InitiativeName: "item-init",
		Round:          1,
		RunId:          "run-1",
		ItemRefs:       []string{"execute/do-thing"},
	}))
	if err == nil {
		t.Fatalf("expected item-level round action to error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument; err=%v", got, err)
	}
	if !strings.Contains(err.Error(), "item-level mode") {
		t.Fatalf("error = %q, want item-level mode message", err.Error())
	}
}

// TestConnectCompleteItemsResolvesModeFromRound proves an omitted mode is
// resolved from the initiative's current non-default round.
func TestConnectCompleteItemsResolvesModeFromRound(t *testing.T) {
	mutator := &fakeBacklogMutator{}
	svc := newTestServiceWithOptions(t, t.TempDir(), serviceOptions{backlogMutator: mutator})
	if _, err := svc.store.CreateRound(RoundEnvelope{
		Mode:           string(ModeHolisticLoop),
		InitiativeName: "init-a",
		ScopeID:        "init-a",
		Phase:          "execute",
		Status:         RoundStatusCompleted,
		RunID:          "run-1",
		Items:          []RoundItem{{Ref: "execute/do-thing"}},
	}); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	client := newConnectClientForService(t, svc)

	_, err := client.CompleteItems(context.Background(), connect.NewRequest(&apipb.OperatingModeCompleteItemsRequest{
		InitiativeName: "init-a",
		Round:          1,
		RunId:          "run-1",
		ItemRefs:       []string{"execute/do-thing"},
	}))
	if err != nil {
		t.Fatalf("CompleteItems: %v", err)
	}
	if len(mutator.completed) != 1 || mutator.completed[0] != "execute/do-thing@initiative.operating_mode.complete_items" {
		t.Fatalf("completed = %#v, want operating-mode completion", mutator.completed)
	}
}

// TestServiceRoundActionsRequireNonDefaultMode exercises the service directly
// (no transport) to pin the mode-resolution guards the round actions share.
func TestServiceRoundActionsRequireNonDefaultMode(t *testing.T) {
	svc := newTestServiceWithOptions(t, t.TempDir(), serviceOptions{backlogMutator: &fakeBacklogMutator{}})

	_, err := svc.CompleteItems(context.Background(), CompleteItemsRequest{
		InitiativeName: "init-a",
		Round:          1,
		RunID:          "run-1",
		ItemRefs:       []string{"execute/do-thing"},
	})
	if err == nil || !strings.Contains(err.Error(), "mode is required") {
		t.Fatalf("blank mode error = %v, want required mode error", err)
	}

	_, err = svc.CancelRound(context.Background(), "init-a", ModeItemLevel, 1)
	if err == nil || !strings.Contains(err.Error(), "item-level mode") {
		t.Fatalf("item-level cancel error = %v, want item-level mode error", err)
	}
}
