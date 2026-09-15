package agents

import "testing"

// [REQ:SWBD-P1-007]
func TestParseGrantReadsExistingScopeAndProgramBindingNames(t *testing.T) {
	grant, err := ParseGrant([]byte(`{"grant":{"scopes":["read","read"],"program_bindings":["workflow.run"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(grant.Scopes) != 1 || grant.Scopes[0] != "read" {
		t.Fatalf("scopes = %#v", grant.Scopes)
	}
	if len(grant.ProgramBindings) != 1 || grant.ProgramBindings[0] != "workflow.run" {
		t.Fatalf("bindings = %#v", grant.ProgramBindings)
	}
}

// [REQ:SWBD-P1-007]
func TestParseGrantFailsClosedWhenMissing(t *testing.T) {
	if _, err := ParseGrant([]byte(`{"id":"agent"}`)); err == nil {
		t.Fatal("expected missing grant error")
	}
}
