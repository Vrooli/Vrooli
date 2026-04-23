package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCmdInitiativesReviewList_HitsCorrectPath(t *testing.T) {
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"rounds":[{"number":1,"status":"pending","summary":"hello"}]}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesReviewList([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/review" {
		t.Errorf("path: %s", path)
	}
}

func TestCmdInitiativesReviewList_EmptyOK(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"rounds":[]}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesReviewList([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestCmdInitiativesReviewGet_PrintsKnownFields(t *testing.T) {
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"number":2,"slug":"r2","kind":"initiative","name":"init","status":"delivered","extra":{"foo":1}}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesReviewGet([]string{"--name", "init", "--round", "2"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/review/2" {
		t.Errorf("path: %s", path)
	}
}

func TestCmdInitiativesReviewGet_RequiresRound(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := app.cmdInitiativesReviewGet([]string{"--name", "init"}); err == nil {
		t.Error("expected error for missing round")
	}
}

func TestCmdInitiativesReviewTrigger_PostsStart(t *testing.T) {
	var (
		method string
		path   string
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"started":true,"round":3,"run_id":"r1"}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesReviewTrigger([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if method != http.MethodPost || path != "/api/v1/initiatives/init/review/trigger" {
		t.Errorf("method=%s path=%s", method, path)
	}
}

func TestCmdInitiativesReviewDecide_SendsVerdict(t *testing.T) {
	cases := []struct {
		flag    string
		verdict string
	}{
		{"--accept", "accept"},
		{"--fail", "fail"},
		{"--followup", "followup"},
	}
	for _, tc := range cases {
		t.Run(tc.verdict, func(t *testing.T) {
			var got map[string]any
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/initiatives/init/review/decide" {
					t.Fatalf("path: %s", r.URL.Path)
				}
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &got)
				_, _ = w.Write([]byte(`{"initiative":"init","verdict":"` + tc.verdict + `","status":"completed","decided_at":"2026-04-23T00:00:00Z"}`))
			})
			app, _ := newFeedbackTestApp(t, handler)
			args := []string{"--name", "init", tc.flag, "--rationale", "ship", "--decided-by", "matt"}
			if err := app.cmdInitiativesReviewDecide(args); err != nil {
				t.Fatalf("run: %v", err)
			}
			if got["verdict"] != tc.verdict {
				t.Errorf("verdict: %v", got["verdict"])
			}
			if got["rationale"] != "ship" || got["decided_by"] != "matt" {
				t.Errorf("body: %#v", got)
			}
		})
	}
}

func TestCmdInitiativesReviewDecide_ExclusiveFlags(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cases := [][]string{
		{"--name", "init"},
		{"--name", "init", "--accept", "--fail"},
		{"--name", "init", "--accept", "--followup"},
		{"--name", "init", "--fail", "--followup"},
		{"--name", "init", "--accept", "--fail", "--followup"},
	}
	for _, args := range cases {
		err := app.cmdInitiativesReviewDecide(args)
		if err == nil || !strings.Contains(err.Error(), "exactly one of") {
			t.Errorf("args=%v: expected mutual-exclusion error, got %v", args, err)
		}
	}
}

func TestCmdInitiativesReviewDecisions_Prints(t *testing.T) {
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"decisions":[{"verdict":"accept","status":"completed","rationale":"ok","decided_by":"matt","decided_at":"2026-04-23T00:00:00Z","prior_status":"review_pending","round":1}]}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesReviewDecisions([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/review/decisions" {
		t.Errorf("path: %s", path)
	}
}
