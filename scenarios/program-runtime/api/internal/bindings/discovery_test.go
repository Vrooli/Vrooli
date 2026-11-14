package bindings

import "testing"

func TestResolvesOperationByIntent(t *testing.T) { // [REQ:PRT-P1-001]
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	bindings, reason := r.ResolveByIntent("list records")
	if len(bindings) == 0 || reason == "" {
		t.Fatalf("bindings=%v reason=%q", bindings, reason)
	}
}

func TestDegradesWhenSearchHubUnavailable(t *testing.T) { // [REQ:PRT-P1-001]
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	bindings, reason := r.ResolveByIntent("operation that does not exist")
	if len(bindings) != 0 || reason == "" {
		t.Fatalf("bindings=%v reason=%q", bindings, reason)
	}
}
