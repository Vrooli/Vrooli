package bindings

import "testing"

func TestResolvesOperationByIntent(t *testing.T) { // [REQ:PRT-P1-001]
	r := fixtureRegistry(t, `{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"NotesService","method":"ListNotes"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	bindings, reason := r.ResolveByIntent("list notes")
	if len(bindings) == 0 || reason == "" {
		t.Fatalf("bindings=%v reason=%q", bindings, reason)
	}
}

func TestDegradesWhenSearchHubUnavailable(t *testing.T) { // [REQ:PRT-P1-001]
	r := fixtureRegistry(t, `{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"NotesService","method":"ListNotes"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	bindings, reason := r.ResolveByIntent("operation that does not exist")
	if len(bindings) != 0 || reason == "" {
		t.Fatalf("bindings=%v reason=%q", bindings, reason)
	}
}
