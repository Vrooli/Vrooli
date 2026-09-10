package main

import (
	"net/http"
	"path/filepath"
	"testing"

	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestV2SessionReturnsComputedFirstUnsatisfiedStep(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","session":{"step":1},"core":{"seed":["demo"],"trusted_base":["demo"]},"scenarios":{"demo":{"enabled":true}},"resources":{"demo":{"enabled":true}},"host_tools":{"git":{"opted_in":true}},"host_safeguards":{"safe":{"opted_in":true}}}`)
	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.session.SessionService/GetSession", `{"target":"local"}`)
	response := &sessionv1.GetSessionResponse{}
	if w.Code != http.StatusOK || protojson.Unmarshal(w.Body.Bytes(), response) != nil || response.GetFirstUnsatisfiedStep() != 8 {
		t.Fatalf("session = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2DraftRoundTripIsTargetScopedAndDoesNotApplyChoices(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"demo":{"enabled":true}}}`)
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
	t.Setenv("BUNDLE_ROOT", "")
	const revision = "2026-08-11T00:00:00Z"
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"`+revision+`","scenarios":{"demo":{"enabled":true}}}`)
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
