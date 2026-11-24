package main

import (
	"net/http"
	"path/filepath"
	"testing"

	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestV2SessionReturnsComputedFirstUnsatisfiedStep(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","session":{"step":1},"core":{"seed":["demo"],"trusted_base":["demo"]},"scenarios":{"demo":{"enabled":true}},"resources":{"demo":{"enabled":true}},"host_tools":{"git":{"opted_in":true}},"host_safeguards":{"safe":{"opted_in":true}}}`)
	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.session.SessionService/GetSession", `{"target":"local"}`)
	response := &sessionv1.GetSessionResponse{}
	if w.Code != http.StatusOK || protojson.Unmarshal(w.Body.Bytes(), response) != nil || response.GetFirstUnsatisfiedStep() != 8 {
		t.Fatalf("session = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2DraftRoundTripIsTargetScopedAndDoesNotApplyChoices(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"demo":{"enabled":true}}}`)
	save := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.session.SessionService/SaveDraft", `{"target":"local","actor":"operator-a","baseRevision":"2026-08-11T00:00:00Z","stepId":"scenarios","choices":{"scenario.demo":"disabled"}}`)
	if save.Code != http.StatusOK || !containsJSONString(save.Body.Bytes(), "scenario.demo") || containsJSONString(save.Body.Bytes(), "operator-a") {
		t.Fatalf("save draft = %d: %s", save.Code, save.Body.String())
	}
	otherActor := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.session.SessionService/GetDraft", `{"target":"local","actor":"operator-b"}`)
	if otherActor.Code != http.StatusOK || !containsJSONString(otherActor.Body.Bytes(), "scenario.demo") || containsJSONString(otherActor.Body.Bytes(), "operator-b") {
		t.Fatalf("draft ownership was request-controlled = %d: %s", otherActor.Code, otherActor.Body.String())
	}
	state := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/GetOperatorState", `{"target":"local"}`)
	if state.Code != http.StatusOK || !containsJSONString(state.Body.Bytes(), `"enabled":true`) {
		t.Fatalf("draft changed effective state = %d: %s", state.Code, state.Body.String())
	}
	discard := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.session.SessionService/DiscardDraft", `{"target":"local","actor":"operator-a"}`)
	if discard.Code != http.StatusOK {
		t.Fatalf("discard draft = %d: %s", discard.Code, discard.Body.String())
	}
}

func TestV2OperatorStateRejectsStaleConcurrentPatch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	const revision = "2026-08-11T00:00:00Z"
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"`+revision+`","scenarios":{"demo":{"enabled":true}}}`)
	first := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState", `{"target":"local","state":{"scenarios":{"demo":{"enabled":false}}},"updateMask":"scenarios","expectedRevision":"`+revision+`"}`)
	if first.Code != http.StatusOK {
		t.Fatalf("first patch = %d: %s", first.Code, first.Body.String())
	}
	stale := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState", `{"target":"local","state":{"scenarios":{"other":{"enabled":true}}},"updateMask":"scenarios","expectedRevision":"`+revision+`"}`)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale patch = %d: %s", stale.Code, stale.Body.String())
	}
	state := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/GetOperatorState", `{"target":"local"}`)
	if state.Code != http.StatusOK || !containsJSONString(state.Body.Bytes(), `"enabled":false`) || containsJSONString(state.Body.Bytes(), `"other"`) {
		t.Fatalf("stale patch changed accepted state = %d: %s", state.Code, state.Body.String())
	}
}

func TestV2ScenarioChoiceReloadsAndRebasesAgainstCurrentRevision(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	const initialRevision = "2026-11-21T23:53:08.240626093Z"
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.3.0","updated_at":"`+initialRevision+`","scenarios":{"hello-python":{"enabled":true}}}`)

	get := func() *operatorstatev1.GetOperatorStateResponse {
		t.Helper()
		response := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/GetOperatorState", `{"target":"local"}`)
		var state operatorstatev1.GetOperatorStateResponse
		if response.Code != http.StatusOK || protojson.Unmarshal(response.Body.Bytes(), &state) != nil {
			t.Fatalf("operator state read = %d: %s", response.Code, response.Body.String())
		}
		return &state
	}
	initial := get()
	if !initial.GetState().GetScenarios()["hello-python"].GetEnabled() || initial.GetState().GetUpdatedAt() != initialRevision {
		t.Fatalf("initial reloaded state = %+v", initial.GetState())
	}

	// A new server instance models the API process restart while reusing the
	// same durable operator-state file.
	restarted := get()
	if !restarted.GetState().GetScenarios()["hello-python"].GetEnabled() || restarted.GetState().GetUpdatedAt() != initialRevision {
		t.Fatalf("state after API restart = %+v", restarted.GetState())
	}

	// Another owner write makes the browser's original revision stale. The
	// strict CAS must reject it; an explicit reload/rebase then succeeds.
	concurrent := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState", `{"target":"local","state":{"session":{"step":3,"stepId":"resources"}},"updateMask":"session","expectedRevision":"`+initialRevision+`"}`)
	if concurrent.Code != http.StatusOK {
		t.Fatalf("concurrent owner patch = %d: %s", concurrent.Code, concurrent.Body.String())
	}
	stale := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState", `{"target":"local","state":{"scenarios":{"hello-python":{"enabled":false}}},"updateMask":"scenarios","expectedRevision":"`+initialRevision+`"}`)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale scenario patch = %d: %s; want conflict", stale.Code, stale.Body.String())
	}

	latest := get()
	latestRevision := latest.GetState().GetUpdatedAt()
	if latestRevision == "" || latestRevision == initialRevision {
		t.Fatalf("latest revision = %q, initial=%q", latestRevision, initialRevision)
	}
	rebased := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState", `{"target":"local","state":{"scenarios":{"hello-python":{"enabled":false}}},"updateMask":"scenarios","expectedRevision":"`+latestRevision+`"}`)
	if rebased.Code != http.StatusOK {
		t.Fatalf("rebased scenario patch = %d: %s", rebased.Code, rebased.Body.String())
	}
	final := get().GetState()
	if final.GetScenarios()["hello-python"].GetEnabled() || final.GetSession().GetStep() != 3 {
		t.Fatalf("rebased state lost the intended scenario/session values: %s", mustJSON(t, final))
	}
}
