package operatingmode

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestRoundActionWithoutModeRejectsItemLevelInitiative(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"item-init": {
				Name:  "item-init",
				Title: "Item Init",
				Mode:  string(ModeItemLevel),
			},
		}},
		backlogMutator: &fakeBacklogMutator{},
	})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/initiatives/item-init/operating-mode/rounds/1/complete-items", bytes.NewBufferString(`{
		"run_id": "run-1",
		"item_refs": ["execute/do-thing"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "item-level mode") {
		t.Fatalf("body = %q, want item-level mode error", rec.Body.String())
	}
}

func TestCatalogEndpointReturnsRegisteredModes(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operating-modes", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	for _, mode := range []string{"item-level", "holistic-loop", "phased-plan-drain"} {
		if !strings.Contains(rec.Body.String(), mode) {
			t.Fatalf("catalog response missing %q: %s", mode, rec.Body.String())
		}
	}
	if !strings.Contains(rec.Body.String(), `"supports_phases":true`) {
		t.Fatalf("catalog response missing phase support metadata: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"description"`) {
		t.Fatalf("catalog response missing description field: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"usage_count"`) {
		t.Fatalf("catalog response missing usage_count field: %s", rec.Body.String())
	}
	// init-a is the default fakeInitiatives item bound to holistic-loop, so
	// the holistic-loop entry should report a non-zero usage count.
	if !strings.Contains(rec.Body.String(), `"usage_count":1`) {
		t.Fatalf("expected at least one mode to report usage_count=1: %s", rec.Body.String())
	}

	// Decision-metadata fields are part of the catalog wire contract from day
	// one (no omitempty on the three lists). Decode and assert per mode.
	var catalog struct {
		Modes []ModeCatalogEntry `json:"modes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(catalog.Modes) != len(Modes()) {
		t.Fatalf("catalog modes len = %d, want %d", len(catalog.Modes), len(Modes()))
	}
	for _, entry := range catalog.Modes {
		if len(entry.BestFor) == 0 {
			t.Errorf("mode %q wire response missing best_for entries", entry.Mode)
		}
		if len(entry.NotFor) == 0 {
			t.Errorf("mode %q wire response missing not_for entries", entry.Mode)
		}
		if len(entry.Tradeoffs) == 0 {
			t.Errorf("mode %q wire response missing tradeoffs entries", entry.Mode)
		}
	}
	// JSON keys must use snake_case to match the rest of the catalog wire.
	for _, key := range []string{`"best_for"`, `"not_for"`, `"tradeoffs"`} {
		if !strings.Contains(rec.Body.String(), key) {
			t.Fatalf("catalog response missing JSON key %s: %s", key, rec.Body.String())
		}
	}
	// when_in_doubt_pick_instead is omitempty; item-level intentionally omits
	// it (it is the safe default), so the key should appear at least once for
	// the other modes.
	if !strings.Contains(rec.Body.String(), `"when_in_doubt_pick_instead"`) {
		t.Fatalf("catalog response missing when_in_doubt_pick_instead key: %s", rec.Body.String())
	}
}

func TestGetModeReturnsLinkedInitiatives(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"loop-init": {
				Name:  "loop-init",
				Title: "Loop Initiative",
				Mode:  string(ModeHolisticLoop),
			},
			"drain-init": {
				Name:  "drain-init",
				Title: "Drain Initiative",
				Mode:  string(ModePhasedPlanDrain),
			},
		}},
	})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operating-modes/holistic-loop", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "loop-init") {
		t.Fatalf("response missing linked initiative loop-init: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "drain-init") {
		t.Fatalf("response should not include initiatives from other modes: %s", rec.Body.String())
	}

	// The catalog detail response surfaces the full per-phase shape and the
	// phase graph block. Spot-check via map[string]any so the assertions read
	// like the wire contract rather than re-encoding the wire types.
	var detail map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	entry, ok := detail["entry"].(map[string]any)
	if !ok {
		t.Fatalf("entry missing or wrong type: %#v", detail["entry"])
	}
	graph, ok := entry["phase_graph"].(map[string]any)
	if !ok {
		t.Fatalf("entry.phase_graph missing or wrong type: %#v", entry["phase_graph"])
	}
	if graph["start_phase"] != "investigate" {
		t.Fatalf("phase_graph.start_phase = %v, want investigate", graph["start_phase"])
	}
	phases, ok := entry["phases"].([]any)
	if !ok || len(phases) == 0 {
		t.Fatalf("entry.phases missing: %#v", entry["phases"])
	}
	first, ok := phases[0].(map[string]any)
	if !ok {
		t.Fatalf("first phase wrong type: %#v", phases[0])
	}
	if first["title"] == nil || first["title"] == "" {
		t.Fatalf("first phase missing title: %#v", first)
	}
	contract, ok := first["output_contract"].(map[string]any)
	if !ok {
		t.Fatalf("first phase output_contract missing: %#v", first)
	}
	if contract["requires_structured_result"] != true {
		t.Fatalf("requires_structured_result = %v, want true", contract["requires_structured_result"])
	}
	transitions, ok := graph["transitions"].([]any)
	if !ok || len(transitions) == 0 {
		t.Fatalf("phase_graph.transitions missing: %#v", graph["transitions"])
	}
	foundReplan := false
	for _, raw := range transitions {
		edge, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if edge["from"] == "execute" && edge["to"] == "investigate" && edge["label"] == "on replan_needed = true" {
			foundReplan = true
			break
		}
	}
	if !foundReplan {
		t.Fatalf("expected transition execute->investigate (replan) in: %#v", transitions)
	}
}

func TestGetModeRejectsUnknown(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operating-modes/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestSimulateModeEndpointReturnsTrace(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/operating-modes/phased-plan-drain/simulate", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var got SimulationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Mode != string(ModePhasedPlanDrain) {
		t.Fatalf("mode = %q, want phased-plan-drain", got.Mode)
	}
	if len(got.Trace) == 0 {
		t.Fatalf("trace missing: %+v", got)
	}
	if got.Trace[0].Phase != "prepare_plan" || got.Trace[0].Transition == nil || got.Trace[0].Transition.To != "execute_next" {
		t.Fatalf("first trace step = %+v, want prepare_plan -> execute_next", got.Trace[0])
	}
	if !strings.Contains(rec.Body.String(), `"condition_kind"`) {
		t.Fatalf("response missing transition guard metadata: %s", rec.Body.String())
	}
	if len(got.Presets) == 0 || got.ActivePreset != "happy-path" {
		t.Fatalf("presets=%+v active=%q, want happy-path default", got.Presets, got.ActivePreset)
	}

	// The preset query param selects a branch-covering scenario.
	presetReq := httptest.NewRequest(http.MethodPost, "/api/v1/operating-modes/phased-plan-drain/simulate?preset=blocked", nil)
	presetRec := httptest.NewRecorder()
	router.ServeHTTP(presetRec, presetReq)
	if presetRec.Code != http.StatusOK {
		t.Fatalf("preset status = %d, want 200; body=%s", presetRec.Code, presetRec.Body.String())
	}
	var blocked SimulationResponse
	if err := json.Unmarshal(presetRec.Body.Bytes(), &blocked); err != nil {
		t.Fatalf("decode preset response: %v", err)
	}
	if blocked.ActivePreset != "blocked" {
		t.Fatalf("active preset = %q, want blocked", blocked.ActivePreset)
	}
	last := blocked.Trace[len(blocked.Trace)-1]
	if last.Phase != "classify_progress" || !last.Terminal {
		t.Fatalf("blocked terminal step = %q terminal=%v, want classify_progress terminal", last.Phase, last.Terminal)
	}
}

func TestRenderSimulationPromptEndpoint(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/operating-modes/holistic-loop/simulate/render?preset=happy-path&step_index=0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var got RenderPromptResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Degraded {
		t.Fatalf("response degraded unexpectedly: %s", got.DegradedReason)
	}
	if got.Phase == "" || got.SkillID == "" || got.Prompt == "" {
		t.Fatalf("render response missing phase/skill/prompt: %+v", got)
	}
	if !strings.Contains(got.Prompt, "Unify the audio-session lifecycle") {
		t.Fatalf("prompt missing substituted title: %s", got.Prompt)
	}

	// Body-supplied preset + step index keep the endpoint curl-friendly.
	body := bytes.NewBufferString(`{"preset":"happy-path","step_index":1}`)
	bodyReq := httptest.NewRequest(http.MethodPost, "/api/v1/operating-modes/holistic-loop/simulate/render", body)
	bodyReq.Header.Set("Content-Type", "application/json")
	bodyRec := httptest.NewRecorder()
	router.ServeHTTP(bodyRec, bodyReq)
	if bodyRec.Code != http.StatusOK {
		t.Fatalf("body request status = %d, want 200; body=%s", bodyRec.Code, bodyRec.Body.String())
	}
	var second RenderPromptResponse
	if err := json.Unmarshal(bodyRec.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode body response: %v", err)
	}
	if second.StepIndex != 1 {
		t.Fatalf("step index = %d, want 1", second.StepIndex)
	}

	// An out-of-range step is a client error, not a rendered prompt.
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/operating-modes/holistic-loop/simulate/render?step_index=999", nil)
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("out-of-range status = %d, want 400; body=%s", badRec.Code, badRec.Body.String())
	}
}

func TestRenderSimulationPromptEndpointDegradesWithoutPromptClient(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})
	svc.prompts = nil
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/operating-modes/holistic-loop/simulate/render?step_index=0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("degraded status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var got RenderPromptResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.Degraded || got.Prompt != "" || len(got.Variables) == 0 {
		t.Fatalf("degraded response = %+v, want degraded with variables and no prompt", got)
	}
}

func TestPatchModeAppliesOverlay(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	body := bytes.NewBufferString(`{"label":"Renamed Loop","description":"Tighter wording."}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/operating-modes/holistic-loop", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Renamed Loop") {
		t.Fatalf("response missing updated label: %s", rec.Body.String())
	}

	// A subsequent GET should reflect the persisted overlay.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/operating-modes/holistic-loop", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if !strings.Contains(getRec.Body.String(), "Renamed Loop") {
		t.Fatalf("GET after PATCH did not return updated label: %s", getRec.Body.String())
	}
}

func TestPatchModeRejectsBlankLabel(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	body := bytes.NewBufferString(`{"label":""}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/operating-modes/holistic-loop", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestPatchModeRejectsUnknownMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	body := bytes.NewBufferString(`{"label":"x"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/operating-modes/does-not-exist", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

func TestRoundActionWithoutModeUsesCurrentNonDefaultInitiativeMode(t *testing.T) {
	root := t.TempDir()
	mutator := &fakeBacklogMutator{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{backlogMutator: mutator})
	_, err := svc.store.CreateRound(RoundEnvelope{
		Mode:           string(ModeHolisticLoop),
		InitiativeName: "init-a",
		ScopeID:        "init-a",
		Phase:          "execute",
		Status:         RoundStatusCompleted,
		RunID:          "run-1",
		Items:          []RoundItem{{Ref: "execute/do-thing"}},
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/initiatives/init-a/operating-mode/rounds/1/complete-items", bytes.NewBufferString(`{
		"run_id": "run-1",
		"item_refs": ["execute/do-thing"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(mutator.completed) != 1 || mutator.completed[0] != "execute/do-thing@initiative.operating_mode.complete_items" {
		t.Fatalf("completed = %#v, want operating-mode completion", mutator.completed)
	}
}

func TestServiceRoundActionsRequireNonDefaultMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{backlogMutator: &fakeBacklogMutator{}})

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
