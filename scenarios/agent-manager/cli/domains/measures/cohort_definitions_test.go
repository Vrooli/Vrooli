package measures

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

type recordedCall struct {
	method, path string
	query        url.Values
	body         interface{}
}

func recorder(calls *[]recordedCall) requestFunc {
	return func(method, path string, query url.Values, body interface{}) ([]byte, error) {
		*calls = append(*calls, recordedCall{method, path, query, body})
		return []byte(`{}`), nil
	}
}

func TestCohortActionsMapToDurableCohortEndpoints(t *testing.T) {
	var calls []recordedCall
	req := recorder(&calls)
	steps := []struct {
		action, name, filter string
		limit                int
		method, path         string
	}{
		{"list", "", "", 0, "GET", "runs/cohorts"},
		{"define", "base line", `{"TagPrefix":"plan-x"}`, 0, "POST", "runs/cohorts"},
		{"show", "base line", "", 25, "GET", "runs/cohorts/base%20line"},
		{"delete", "base line", "", 0, "DELETE", "runs/cohorts/base%20line"},
	}
	for _, step := range steps {
		if _, err := runCohortAction(req, step.action, step.name, step.filter, "", step.limit); err != nil {
			t.Fatalf("%s: %v", step.action, err)
		}
	}
	for i, step := range steps {
		if calls[i].method != step.method || calls[i].path != step.path {
			t.Errorf("%s -> %s %s, want %s %s", step.action, calls[i].method, calls[i].path, step.method, step.path)
		}
	}
	if calls[2].query.Get("limit") != "25" {
		t.Errorf("show limit = %q, want 25", calls[2].query.Get("limit"))
	}
	encoded, _ := json.Marshal(calls[1].body)
	if !strings.Contains(string(encoded), `"name":"base line"`) || !strings.Contains(string(encoded), `"TagPrefix":"plan-x"`) {
		t.Errorf("define body = %s", encoded)
	}
}

func TestCohortActionValidatesInputsBeforeCallingTheAPI(t *testing.T) {
	var calls []recordedCall
	req := recorder(&calls)
	for name, run := range map[string]func() error{
		"define without filter": func() error { _, err := runCohortAction(req, "define", "n", "", "", 0); return err },
		"define non-object":     func() error { _, err := runCohortAction(req, "define", "n", `[1]`, "", 0); return err },
		"define invalid json":   func() error { _, err := runCohortAction(req, "define", "n", `{`, "", 0); return err },
		"show without name":     func() error { _, err := runCohortAction(req, "show", " ", "", "", 0); return err },
		"delete without name":   func() error { _, err := runCohortAction(req, "delete", "", "", "", 0); return err },
		"unknown action":        func() error { _, err := runCohortAction(req, "rename", "n", "", "", 0); return err },
	} {
		if run() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if len(calls) != 0 {
		t.Errorf("invalid input reached the API: %+v", calls)
	}
}

func TestCohortCommandIsRegisteredWithValidSchema(t *testing.T) {
	for _, command := range Register().Subcommands {
		if command.Name == "cohort" {
			if err := command.Args.Validate(); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("measures cohort is not registered")
}
