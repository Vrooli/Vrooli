package main

import (
	"net/http"
	"testing"
)

func TestCmdInitiativesGraphShow_ReadsGraphJSON(t *testing.T) {
	var path string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{
			"initiative":"init",
			"generated_at":"2026-04-23T00:00:00Z",
			"nodes":[
				{"id":"execute/a","kind":"execute","title":"A","status":"backlog","priority":1,"effort":"M"},
				{"id":"execute/b","kind":"execute","title":"B","status":"ready","priority":2,"effort":"S","archived":true}
			],
			"edges":[{"from":"execute/a","to":"execute/b","kind":"depends_on"}]
		}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesGraphShow([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if path != "/api/v1/initiatives/init/files/graph.json" {
		t.Errorf("path: %s", path)
	}
}

func TestCmdInitiativesGraphShow_EmptyGraphFallsBackToRaw(t *testing.T) {
	// When the projection hasn't run or the response is malformed, the
	// command must surface something useful rather than crash.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"initiative":"init","nodes":[],"edges":[]}`))
	})
	app, _ := newFeedbackTestApp(t, handler)
	if err := app.cmdInitiativesGraphShow([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestCmdInitiativesGraphShow_RequiresName(t *testing.T) {
	app, _ := newFeedbackTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := app.cmdInitiativesGraphShow(nil); err == nil {
		t.Error("expected usage error")
	}
}
