package main

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newFeedbackTestApp spins up an httptest server and configures an App to
// talk to it. Returns the server so tests can close it and inspect handler
// invocations, plus the App.
func newFeedbackTestApp(t *testing.T, handler http.Handler) (*App, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app, server
}

func TestCmdInitiativesFeedbackList_PrintsRounds(t *testing.T) {
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{
			"rounds": [{
				"number": 1,
				"slug": "ui-rewrite",
				"type": "feedback",
				"status": "awaiting_user",
				"submission": {"text": "something off"},
				"current_proposal_id": "p1",
				"proposals": [
					{"id":"p1","message_index":1,"proposal":{"form":"mutation_list","mutations":[{"id":"m1","op":"change_status","target":"execute/x"}]}}
				]
			}],
			"count": 1
		}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesFeedbackList([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/feedback" {
		t.Errorf("path: %s", path)
	}
}

func TestCmdInitiativesFeedbackList_RequiresName(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := app.cmdInitiativesFeedbackList(nil); err == nil {
		t.Error("expected usage error")
	}
}

func TestCmdInitiativesFeedbackGet_PrintsRound(t *testing.T) {
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{
			"number":1,"slug":"ui","type":"feedback","status":"awaiting_user",
			"submission":{"text":"hi"},
			"thread":[{"role":"user","content":"hi"}],
			"proposals":[{"id":"p1","message_index":1,"proposal":{"form":"mutation_list","mutations":[]}}],
			"current_proposal_id":"p1"
		}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesFeedbackGet([]string{"--name", "init", "--round", "1"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/feedback/1" {
		t.Errorf("path: %s", path)
	}
}

func TestCmdInitiativesFeedbackGet_RejectsInvalidRound(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	err := app.cmdInitiativesFeedbackGet([]string{"--name", "init", "--round", "0"})
	if err == nil || !strings.Contains(err.Error(), "--round") {
		t.Errorf("expected round validation error, got %v", err)
	}
}

func TestCmdInitiativesFeedbackSubmit_JSONBodyWhenNoFiles(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotCT     string
		gotBody   map[string]any
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"number":2,"slug":"s","type":"feedback","status":"agent_thinking","submission":{"text":"hi"}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	err := app.cmdInitiativesFeedbackSubmit([]string{
		"--name", "init",
		"--type", "feedback",
		"--text", "please look",
		"--override",
		"--decided-by", "matt",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/initiatives/init/feedback" {
		t.Errorf("method=%s path=%s", gotMethod, gotPath)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Errorf("content-type should be JSON, got %q", gotCT)
	}
	if gotBody["type"] != "feedback" || gotBody["text"] != "please look" || gotBody["override"] != true || gotBody["decided_by"] != "matt" {
		t.Errorf("unexpected body: %#v", gotBody)
	}
}

func TestCmdInitiativesFeedbackSubmit_MultipartWhenFilesProvided(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "shot.png")
	if err := os.WriteFile(filePath, []byte("pretend-png"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	var gotCT string
	var parsedFields map[string][]string
	var gotFiles []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(gotCT)
		if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
			t.Errorf("expected multipart, got %q", gotCT)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		parsedFields = map[string][]string{}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("multipart: %v", err)
			}
			name := part.FormName()
			if fn := part.FileName(); fn != "" {
				gotFiles = append(gotFiles, fn)
				_, _ = io.Copy(io.Discard, part)
				continue
			}
			buf, _ := io.ReadAll(part)
			parsedFields[name] = append(parsedFields[name], string(buf))
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"number":3,"slug":"s","type":"feedback","status":"agent_thinking","submission":{"text":"t"}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	err := app.cmdInitiativesFeedbackSubmit([]string{
		"--name", "init",
		"--type", "feedback",
		"--text", "with screenshot",
		"--file", filePath,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if parsedFields["text"][0] != "with screenshot" {
		t.Errorf("text field missing/wrong: %#v", parsedFields)
	}
	if parsedFields["type"][0] != "feedback" {
		t.Errorf("type field missing/wrong: %#v", parsedFields)
	}
	if len(gotFiles) != 1 || gotFiles[0] != "shot.png" {
		t.Errorf("unexpected files: %v", gotFiles)
	}
}

func TestCmdInitiativesFeedbackSubmit_RejectsUnknownType(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	err := app.cmdInitiativesFeedbackSubmit([]string{"--name", "init", "--type", "bogus", "--text", "x"})
	if err == nil || !strings.Contains(err.Error(), "--type") {
		t.Errorf("expected type validation error, got %v", err)
	}
}

func TestCmdInitiativesFeedbackSubmit_RequiresTextOrFile(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	err := app.cmdInitiativesFeedbackSubmit([]string{"--name", "init", "--type", "feedback"})
	if err == nil || !strings.Contains(err.Error(), "--text") {
		t.Errorf("expected text/file error, got %v", err)
	}
}

func TestCmdInitiativesFeedbackContinue_JSONBody(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		_, _ = w.Write([]byte(`{"number":4,"slug":"s","type":"feedback","status":"agent_thinking","submission":{"text":"t"}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	err := app.cmdInitiativesFeedbackContinue([]string{"--name", "init", "--round", "4", "--text", "please drop m1"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/api/v1/initiatives/init/feedback/4/continue" {
		t.Errorf("path: %s", gotPath)
	}
	if gotBody["text"] != "please drop m1" {
		t.Errorf("body: %#v", gotBody)
	}
}

func TestCmdInitiativesFeedbackDecide_AcceptAll(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		_, _ = w.Write([]byte(`{
			"round":{"number":5,"slug":"s","type":"feedback","status":"applied","submission":{"text":"t"}},
			"apply_result":{"applied":2,"failed":0,"skipped":0,"outcomes":[
				{"mutation_id":"m1","op":"change_status","target":"execute/x","applied":true},
				{"mutation_id":"m2","op":"add_edge","applied":true}
			]}
		}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	err := app.cmdInitiativesFeedbackDecide([]string{"--name", "init", "--round", "5", "--accept", "--rationale", "ship it"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/api/v1/initiatives/init/feedback/5/decide" {
		t.Errorf("path: %s", gotPath)
	}
	if gotBody["kind"] != "accept" || gotBody["rationale"] != "ship it" {
		t.Errorf("body: %#v", gotBody)
	}
	if _, has := gotBody["accepted_mutation_ids"]; has {
		t.Errorf("accept (all) should not send accepted_mutation_ids: %#v", gotBody)
	}
}

func TestCmdInitiativesFeedbackDecide_PartialAccept(t *testing.T) {
	var gotBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		_, _ = w.Write([]byte(`{"round":{"number":5,"slug":"s","type":"feedback","status":"applied","submission":{"text":"t"}},"apply_result":{"applied":1,"failed":0,"skipped":1,"outcomes":[]}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	err := app.cmdInitiativesFeedbackDecide([]string{"--name", "init", "--round", "5", "--accept", "--mutations", "m1,m3"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotBody["kind"] != "partial_accept" {
		t.Errorf("kind: %v", gotBody["kind"])
	}
	ids, _ := gotBody["accepted_mutation_ids"].([]any)
	if len(ids) != 2 || ids[0] != "m1" || ids[1] != "m3" {
		t.Errorf("ids: %#v", ids)
	}
}

func TestCmdInitiativesFeedbackDecide_DismissRoutesToDismissEndpoint(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			_ = json.Unmarshal(body, &gotBody)
		}
		_, _ = w.Write([]byte(`{"number":7,"slug":"s","type":"feedback","status":"dismissed","submission":{"text":"t"}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	err := app.cmdInitiativesFeedbackDecide([]string{"--name", "init", "--round", "7", "--dismiss", "--rationale", "not useful"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/api/v1/initiatives/init/feedback/7/dismiss" {
		t.Errorf("path: %s", gotPath)
	}
	if gotBody["rationale"] != "not useful" {
		t.Errorf("body: %#v", gotBody)
	}
	if _, has := gotBody["kind"]; has {
		t.Errorf("/dismiss endpoint should not require 'kind' field")
	}
}

func TestCmdInitiativesFeedbackCancel_HitsCancelEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			_ = json.Unmarshal(body, &gotBody)
		}
		_, _ = w.Write([]byte(`{"number":3,"slug":"s","type":"feedback","status":"dismissed","submission":{"text":"t"},"decision":{"kind":"dismiss","rationale":"agent stuck","decided_at":"2026-04-25T12:00:00Z"}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	err := app.cmdInitiativesFeedbackCancel([]string{
		"--name", "init",
		"--round", "3",
		"--rationale", "agent stuck",
		"--decided-by", "matt",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method: %s", gotMethod)
	}
	if gotPath != "/api/v1/initiatives/init/feedback/3/cancel" {
		t.Errorf("path: %s", gotPath)
	}
	if gotBody["rationale"] != "agent stuck" {
		t.Errorf("body rationale: %#v", gotBody)
	}
	if gotBody["decided_by"] != "matt" {
		t.Errorf("body decided_by: %#v", gotBody)
	}
}

func TestCmdInitiativesFeedbackCancel_RequiresName(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	err := app.cmdInitiativesFeedbackCancel([]string{"--round", "1"})
	if err == nil {
		t.Fatal("expected error: --name required")
	}
}

func TestCmdInitiativesFeedbackCancel_RequiresPositiveRound(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	err := app.cmdInitiativesFeedbackCancel([]string{"--name", "init", "--round", "0"})
	if err == nil {
		t.Fatal("expected error: --round must be positive")
	}
}

func TestCmdInitiativesFeedbackDecide_ExclusiveDecisionFlags(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cases := [][]string{
		{"--name", "init", "--round", "1"},
		{"--name", "init", "--round", "1", "--accept", "--reject"},
		{"--name", "init", "--round", "1", "--accept", "--dismiss"},
		{"--name", "init", "--round", "1", "--reject", "--dismiss"},
	}
	for _, args := range cases {
		err := app.cmdInitiativesFeedbackDecide(args)
		if err == nil || !strings.Contains(err.Error(), "exactly one of") {
			t.Errorf("args=%v: expected mutual-exclusion error, got %v", args, err)
		}
	}
}

func TestCmdInitiativesFeedbackDecide_MutationsOnlyWithAccept(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	err := app.cmdInitiativesFeedbackDecide([]string{"--name", "init", "--round", "1", "--reject", "--mutations", "m1"})
	if err == nil || !strings.Contains(err.Error(), "--mutations") {
		t.Errorf("expected --mutations-with-accept error, got %v", err)
	}
}

func TestCmdInitiativesFeedbackLock_PrintsLocked(t *testing.T) {
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"locked":true,"holder":{"run_id":"r1","purpose":"feedback","acquired_at":"2026-04-23T00:00:00Z"}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesFeedbackLock([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/feedback/lock" {
		t.Errorf("path: %s", path)
	}
}
