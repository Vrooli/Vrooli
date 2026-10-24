package manifestschema

import "testing"

func TestScenarioShellInvocationRejectsDeclaredShells(t *testing.T) {
	manifest := []byte(`{
  "lifecycle": {"setup": {"steps": [{"name": "bad", "exec": ["bash", "-c", "echo bad"]}]}},
  "components": {"api": {"run": {"argv": ["./scripts/start.sh"]}}}
}`)
	got := CheckScenarioShellInvocations(manifest, "scenarios/demo/.vrooli/service.json")
	if len(got) != 2 {
		t.Fatalf("expected two shell findings, got %d: %#v", len(got), got)
	}
}

func TestScenarioShellInvocationAllowsNativeArgv(t *testing.T) {
	manifest := []byte(`{
  "lifecycle": {"setup": {"steps": [{"name": "test", "exec": ["make", "test"]}]}},
  "components": {"api": {"run": {"argv": ["{{bin.api}}", "serve"]}}}
}`)
	if got := CheckScenarioShellInvocations(manifest, "scenarios/demo/.vrooli/service.json"); len(got) != 0 {
		t.Fatalf("expected native argv to pass, got %#v", got)
	}
}
