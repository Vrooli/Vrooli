package teams

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// fakeContext is a minimal stand-in for appctx.Context that records the last
// call and returns a canned response, scoped to a single subcommand under test.
type fakeContext struct {
	t *testing.T

	// gotMethod / gotPath / gotPayload capture what the command actually sent.
	gotMethod  string
	gotPath    string
	gotPayload []byte

	// response is encoded into the result of Get/Post/Put if non-nil.
	response interface{}

	// err is returned from the matching method if non-nil.
	err error
}

func (f *fakeContext) record(method, path string, payload interface{}) error {
	f.gotMethod = method
	f.gotPath = path
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		f.gotPayload = raw
	}
	return f.err
}

func (f *fakeContext) writeResult(result interface{}) error {
	if f.response == nil || result == nil {
		return nil
	}
	raw, err := json.Marshal(f.response)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

func (f *fakeContext) Get(path string, result interface{}) error {
	if err := f.record("GET", path, nil); err != nil {
		return err
	}
	return f.writeResult(result)
}

func (f *fakeContext) GetWithQuery(path string, query url.Values, result interface{}) error {
	if query != nil {
		path = path + "?" + query.Encode()
	}
	return f.Get(path, result)
}

func (f *fakeContext) Post(path string, payload, result interface{}) error {
	if err := f.record("POST", path, payload); err != nil {
		return err
	}
	return f.writeResult(result)
}

func (f *fakeContext) Put(path string, payload, result interface{}) error {
	if err := f.record("PUT", path, payload); err != nil {
		return err
	}
	return f.writeResult(result)
}

func (f *fakeContext) Delete(path string) error {
	return f.record("DELETE", path, nil)
}

func (f *fakeContext) assertMethodPath(t *testing.T, method, path string) {
	t.Helper()
	if f.gotMethod != method {
		t.Errorf("method = %q, want %q", f.gotMethod, method)
	}
	if f.gotPath != path {
		t.Errorf("path = %q, want %q", f.gotPath, path)
	}
}

// --- decision-update ---

func TestCmdDecisionUpdateRequiresAtLeastOneField(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdDecisionUpdate(fc, []string{"team-a", "dec-1"})
	if err == nil {
		t.Fatal("expected error for no field flags, got nil")
	}
	if !strings.Contains(err.Error(), "at least one field") {
		t.Errorf("error = %v, want 'at least one field'", err)
	}
}

func TestCmdDecisionUpdateSendsPatchWithProvidedFields(t *testing.T) {
	fc := &fakeContext{
		t:        t,
		response: DecisionEntry{ID: "dec-1", Status: "accepted", Selected: "A"},
	}
	err := cmdDecisionUpdate(fc, []string{
		"team-a", "dec-1",
		"--status=accepted",
		"--selected=A",
		"--notes=ship it",
	})
	if err != nil {
		t.Fatalf("cmdDecisionUpdate error = %v", err)
	}
	fc.assertMethodPath(t, "PUT", "/teams/team-a/decisions/dec-1")

	var sent map[string]interface{}
	if err := json.Unmarshal(fc.gotPayload, &sent); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if sent["status"] != "accepted" {
		t.Errorf("status = %v, want accepted", sent["status"])
	}
	if sent["selected"] != "A" {
		t.Errorf("selected = %v, want A", sent["selected"])
	}
	if sent["notes"] != "ship it" {
		t.Errorf("notes = %v, want 'ship it'", sent["notes"])
	}
	if _, present := sent["rationale"]; present {
		t.Error("rationale must not be sent when not provided")
	}
}

func TestCmdDecisionUpdateAllowsEmptyStringNotes(t *testing.T) {
	fc := &fakeContext{t: t, response: DecisionEntry{ID: "dec-1"}}
	err := cmdDecisionUpdate(fc, []string{"team-a", "dec-1", "--notes="})
	if err != nil {
		t.Fatalf("cmdDecisionUpdate error = %v", err)
	}
	var sent map[string]interface{}
	if err := json.Unmarshal(fc.gotPayload, &sent); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	v, present := sent["notes"]
	if !present {
		t.Fatal("notes must be present in payload (clear-to-empty intent)")
	}
	if v != "" {
		t.Errorf("notes = %v, want empty string", v)
	}
}

func TestCmdDecisionUpdateRejectsInvalidOptionsJSON(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdDecisionUpdate(fc, []string{"team-a", "dec-1", "--options=not-json"})
	if err == nil {
		t.Fatal("expected error for invalid options JSON")
	}
	if !strings.Contains(err.Error(), "invalid options JSON") {
		t.Errorf("error = %v, want 'invalid options JSON'", err)
	}
}

// --- decision-accept ---

func TestCmdDecisionAcceptRequiresSelected(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdDecisionAccept(fc, []string{"team-a", "dec-1"})
	if err == nil {
		t.Fatal("expected error for missing --selected")
	}
	if !strings.Contains(err.Error(), "--selected is required") {
		t.Errorf("error = %v, want '--selected is required'", err)
	}
}

func TestCmdDecisionAcceptRequiresFreeformWhenOther(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdDecisionAccept(fc, []string{"team-a", "dec-1", "--selected=__other__"})
	if err == nil {
		t.Fatal("expected error for __other__ without --freeform")
	}
	if !strings.Contains(err.Error(), "--freeform is required") {
		t.Errorf("error = %v, want '--freeform is required'", err)
	}
}

func TestCmdDecisionAcceptSendsAcceptedStatus(t *testing.T) {
	fc := &fakeContext{
		t:        t,
		response: DecisionEntry{ID: "dec-1", Status: "accepted", Selected: "B"},
	}
	err := cmdDecisionAccept(fc, []string{"team-a", "dec-1", "--selected=B", "--notes=ok"})
	if err != nil {
		t.Fatalf("cmdDecisionAccept error = %v", err)
	}
	fc.assertMethodPath(t, "PUT", "/teams/team-a/decisions/dec-1")

	var sent map[string]interface{}
	if err := json.Unmarshal(fc.gotPayload, &sent); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if sent["status"] != "accepted" {
		t.Errorf("status = %v, want accepted", sent["status"])
	}
	if sent["selected"] != "B" {
		t.Errorf("selected = %v, want B", sent["selected"])
	}
	if sent["notes"] != "ok" {
		t.Errorf("notes = %v, want ok", sent["notes"])
	}
}

func TestCmdDecisionAcceptOmitsUnsetFreeform(t *testing.T) {
	fc := &fakeContext{t: t, response: DecisionEntry{ID: "dec-1"}}
	err := cmdDecisionAccept(fc, []string{"team-a", "dec-1", "--selected=B"})
	if err != nil {
		t.Fatalf("cmdDecisionAccept error = %v", err)
	}
	var sent map[string]interface{}
	if err := json.Unmarshal(fc.gotPayload, &sent); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if _, present := sent["freeform"]; present {
		t.Error("freeform must be omitted when not provided")
	}
}

// --- decision-reject ---

func TestCmdDecisionRejectRequiresNotes(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdDecisionReject(fc, []string{"team-a", "dec-1"})
	if err == nil {
		t.Fatal("expected error for missing --notes")
	}
	if !strings.Contains(err.Error(), "--notes is required") {
		t.Errorf("error = %v, want '--notes is required'", err)
	}
}

func TestCmdDecisionRejectSendsRejectedStatus(t *testing.T) {
	fc := &fakeContext{t: t, response: DecisionEntry{ID: "dec-1", Status: "rejected"}}
	err := cmdDecisionReject(fc, []string{"team-a", "dec-1", "--notes=stale"})
	if err != nil {
		t.Fatalf("cmdDecisionReject error = %v", err)
	}
	fc.assertMethodPath(t, "PUT", "/teams/team-a/decisions/dec-1")

	var sent map[string]interface{}
	if err := json.Unmarshal(fc.gotPayload, &sent); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if sent["status"] != "rejected" {
		t.Errorf("status = %v, want rejected", sent["status"])
	}
	if sent["notes"] != "stale" {
		t.Errorf("notes = %v, want 'stale'", sent["notes"])
	}
}

// --- decision-delete ---

func TestCmdDecisionDeleteRequiresYesOrConfirm(t *testing.T) {
	fc := &fakeContext{t: t}
	// No --yes flag and no stdin available — should abort.
	err := cmdDecisionDelete(fc, []string{"team-a", "dec-1"})
	if err == nil {
		t.Fatal("expected abort error when --yes omitted and no confirmation")
	}
	if !strings.Contains(err.Error(), "aborted") {
		t.Errorf("error = %v, want 'aborted'", err)
	}
	if fc.gotMethod != "" {
		t.Error("delete must not call API when aborted")
	}
}

func TestCmdDecisionDeleteSkipsConfirmWithYes(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdDecisionDelete(fc, []string{"team-a", "dec-1", "--yes"})
	if err != nil {
		t.Fatalf("cmdDecisionDelete error = %v", err)
	}
	fc.assertMethodPath(t, "DELETE", "/teams/team-a/decisions/dec-1")
}

// --- decision-show ---

func TestCmdDecisionShowFindsByID(t *testing.T) {
	fc := &fakeContext{
		t: t,
		response: DecisionListResponse{
			TeamID: "team-a",
			Entries: []DecisionEntry{
				{ID: "dec-1", By: "agent", Decision: "first"},
				{ID: "dec-2", By: "agent", Decision: "second"},
			},
			Total: 2,
		},
	}
	err := cmdDecisionShow(fc, []string{"team-a", "dec-2"})
	if err != nil {
		t.Fatalf("cmdDecisionShow error = %v", err)
	}
	fc.assertMethodPath(t, "GET", "/teams/team-a/decisions?last=0")
}

func TestCmdDecisionShowReturnsErrorWhenMissing(t *testing.T) {
	fc := &fakeContext{
		t:        t,
		response: DecisionListResponse{TeamID: "team-a", Entries: []DecisionEntry{}},
	}
	err := cmdDecisionShow(fc, []string{"team-a", "missing-id"})
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want 'not found'", err)
	}
}

// --- usage / arity ---

func TestDecisionSubcommandsRequireBothPositionalArgs(t *testing.T) {
	commands := []struct {
		name string
		run  func(args []string) error
	}{
		{"decision-update", func(args []string) error { return cmdDecisionUpdate(&fakeContext{t: t}, args) }},
		{"decision-accept", func(args []string) error { return cmdDecisionAccept(&fakeContext{t: t}, args) }},
		{"decision-reject", func(args []string) error { return cmdDecisionReject(&fakeContext{t: t}, args) }},
		{"decision-delete", func(args []string) error { return cmdDecisionDelete(&fakeContext{t: t}, args) }},
		{"decision-show", func(args []string) error { return cmdDecisionShow(&fakeContext{t: t}, args) }},
	}
	for _, c := range commands {
		err := c.run([]string{"team-a"}) // missing the second positional
		if err == nil {
			t.Errorf("%s: expected usage error for missing decision-id", c.name)
			continue
		}
		if !strings.Contains(err.Error(), "usage:") {
			t.Errorf("%s: error = %v, want 'usage:'", c.name, err)
		}
	}
}
