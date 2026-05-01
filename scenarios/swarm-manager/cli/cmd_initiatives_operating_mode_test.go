package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCmdInitiativesModeList_ReadsCatalog(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/operating-modes" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"modes":[{"mode":"item-level","label":"Item Level","scope_kind":"backlog_item","run_strategy":"existing_item_flow","workspace_tab_id":"info","default":true,"switchable":true,"supports_phases":false}]}`))
	}))

	if err := app.cmdInitiativesModeList([]string{}); err != nil {
		t.Fatalf("cmdInitiativesModeList returned error: %v", err)
	}
}

func TestCmdInitiativesModeWorkspace_ReadsWorkspace(t *testing.T) {
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{
			"initiative_name":"init",
			"mode":"holistic-loop",
			"definition":{
				"label":"Holistic Loop",
				"scope_kind":"initiative",
				"run_strategy":"operator_gated_loop",
				"phases":[{"phase":"investigate","profile_key":"swarm-manager/deep-work","writes_repo":false}]
			},
			"artifacts":[{"path":"modes/holistic-loop/findings.md","required":true}],
			"rounds":[{"round":1,"mode":"holistic-loop","phase":"investigate","status":"completed","run_id":"run-1","agent_profile_key":"swarm-manager/deep-work"}]
		}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesModeWorkspace([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/operating-mode/workspace" {
		t.Errorf("path: %s", path)
	}
}

func TestCmdInitiativesModeSwitch_PostsCancellationConfirmation(t *testing.T) {
	var path string
	var payload map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"initiative_name":"init","from_mode":"item-level","to_mode":"holistic-loop","canceled_item_executions":[{"item_ref":"execute/a"}]}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesModeSwitch([]string{"--name", "init", "--mode", "holistic-loop", "--cancel-active-item-executions"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/operating-mode/switch" {
		t.Errorf("path: %s", path)
	}
	if payload["mode"] != "holistic-loop" {
		t.Errorf("mode payload: %v", payload["mode"])
	}
	if payload["cancel_active_item_executions"] != true {
		t.Errorf("cancel payload: %v", payload["cancel_active_item_executions"])
	}
}

func TestCmdInitiativesModeStart_PostsPhaseStart(t *testing.T) {
	var path string
	var payload map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"round":2,"mode":"holistic-loop","phase":"execute","status":"agent_running","run_id":"run-2","agent_profile_key":"swarm-manager/deep-work"}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesModeStart([]string{"--name", "init", "--phase", "execute", "--note", "go"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/operating-mode/phases/execute/start" {
		t.Errorf("path: %s", path)
	}
	if payload["note"] != "go" {
		t.Errorf("note payload: %v", payload["note"])
	}
}

func TestCmdInitiativesModeRefresh_PostsRoundRefresh(t *testing.T) {
	var rawQuery string
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		rawQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"round":2,"mode":"phased-plan-drain","phase":"execute_next","status":"completed","agent_profile_key":"swarm-manager/deep-work"}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesModeRefresh([]string{"--name", "init", "--mode", "phased-plan-drain", "--round", "2"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/operating-mode/rounds/2/refresh" {
		t.Errorf("path: %s", path)
	}
	if rawQuery != "mode=phased-plan-drain" {
		t.Errorf("query: %s", rawQuery)
	}
}

func TestCmdInitiativesModeCompleteItems_PostsRunValidatedRefs(t *testing.T) {
	var path string
	var payload map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"initiative_name":"init","mode":"holistic-loop","phase":"execute","round":3,"run_id":"run-3","completed_items":[{"item_ref":"execute/a","from_status":"ready","to_status":"completed"}]}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesModeComplete([]string{"--name", "init", "--mode", "holistic-loop", "--round", "3", "--run-id", "run-3", "--items", "execute/a"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/operating-mode/rounds/3/complete-items" {
		t.Errorf("path: %s", path)
	}
	if payload["run_id"] != "run-3" {
		t.Errorf("run payload: %v", payload["run_id"])
	}
	items, ok := payload["item_refs"].([]any)
	if !ok || len(items) != 1 || items[0] != "execute/a" {
		t.Errorf("items payload: %#v", payload["item_refs"])
	}
}

func TestCmdInitiativesModeApplyBacklogSync_PostsSelectedMutationIDs(t *testing.T) {
	var path string
	var rawQuery string
	var payload map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		rawQuery = r.URL.RawQuery
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"initiative_name":"init","mode":"phased-plan-drain","phase":"classify_progress","round":4,"run_id":"run-4","proposal_result":{"applied":1,"failed":0,"skipped":1,"created":1,"outcomes":[{"mutation_id":"m1","op":"add_item","target":"fix/follow-up","applied":true}]}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesModeApplyBacklogSync([]string{"--name", "init", "--mode", "phased-plan-drain", "--round", "4", "--run-id", "run-4", "--mutations", "m1,m3"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/operating-mode/rounds/4/apply-backlog-sync" {
		t.Errorf("path: %s", path)
	}
	if rawQuery != "mode=phased-plan-drain" {
		t.Errorf("query: %s", rawQuery)
	}
	if payload["run_id"] != "run-4" {
		t.Errorf("run payload: %v", payload["run_id"])
	}
	mutations, ok := payload["accepted_mutation_ids"].([]any)
	if !ok || len(mutations) != 2 || mutations[0] != "m1" || mutations[1] != "m3" {
		t.Errorf("mutations payload: %#v", payload["accepted_mutation_ids"])
	}
}
