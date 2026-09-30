package vps

import (
	"encoding/json"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
)

func edgeContext(env, email string) CommandContext {
	return CommandContext{DeploymentID: "dep-1", ScenarioID: "app", Identity: Identity{OperationID: "op-1", Fence: 1}, Manifest: domain.CloudManifest{
		Target: domain.ManifestTarget{VPS: &domain.ManifestVPS{Workdir: "/root/Vrooli"}}, Scenario: domain.ManifestScenario{ID: "app"},
		Edge: domain.ManifestEdge{Domain: "app.example.test", ACMEEnvironment: env, Caddy: domain.ManifestCaddy{Enabled: true, Email: email}},
	}}
}

// [REQ:STC-P0-036] The edge spec the executor hands the target owner routes
// only the public UI listener of the deployment, carries the manifest's ACME
// options and never a credential; it travels as one argv-safe argument.
func TestEdgeRouteApplyBuildsOwnerScopedSpec(t *testing.T) {
	cc := edgeContext("staging", "ops@example.test")
	action := execplan.Action{ID: execplan.OpEdgeRouteApply, OwnerOperation: execplan.OpEdgeRouteApply, Inputs: map[string]string{"domain": "app.example.test", "upstream_port": "3000", "tls_email": "ops@example.test", "tls_enabled": "true"}}
	commands, err := ActionCommands(action, cc)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].Command.Verb != "cloud-target edge route-apply" || !commands[0].Command.Effectful {
		t.Fatalf("commands = %+v", commands)
	}
	args := commands[0].Command.Args
	var encoded string
	for i, a := range args {
		if a == "--spec" {
			encoded = args[i+1]
		}
	}
	if !strings.HasPrefix(encoded, JSONArgPrefix) {
		t.Fatalf("spec must travel as b64 JSON, got %q", encoded)
	}
	raw, err := DecodeJSONArg(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		DeploymentID    string `json:"deployment_id"`
		Domain          string `json:"domain"`
		Snippet         string `json:"snippet"`
		ACMEEnvironment string `json:"acme_environment"`
		Routes          []struct {
			Host         string `json:"host"`
			UpstreamPort int    `json:"upstream_port"`
			ListenerID   string `json:"listener_id"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.DeploymentID != "dep-1" || spec.Domain != "app.example.test" || spec.ACMEEnvironment != "staging" || len(spec.Routes) != 1 || spec.Routes[0].UpstreamPort != 3000 || spec.Routes[0].ListenerID != "app/ui" {
		t.Fatalf("spec = %+v", spec)
	}
	for _, want := range []string{"app.example.test {", "reverse_proxy 127.0.0.1:3000", "tls ops@example.test", "acme-staging"} {
		if !strings.Contains(spec.Snippet, want) {
			t.Fatalf("snippet lacks %q: %s", want, spec.Snippet)
		}
	}
	if strings.Contains(spec.Snippet, "canary") || strings.Contains(strings.Join(args, " "), "canary") {
		t.Fatal("no credential may ride in the edge spec")
	}
}

// [REQ:STC-P0-036] A missing domain or a non-port upstream is refused
// before any invocation is built.
func TestEdgeRouteApplyRefusesInvalidInputs(t *testing.T) {
	cc := edgeContext("", "")
	for _, inputs := range []map[string]string{{"domain": "", "upstream_port": "3000"}, {"domain": "app.example.test", "upstream_port": "x"}, {"domain": "app.example.test", "upstream_port": "0"}} {
		action := execplan.Action{ID: execplan.OpEdgeRouteApply, OwnerOperation: execplan.OpEdgeRouteApply, Inputs: inputs}
		if _, err := ActionCommands(action, cc); err == nil {
			t.Fatalf("inputs %v must be refused", inputs)
		}
	}
	spec, err := EdgeSpecFor(cc, map[string]string{"domain": "app.example.test", "upstream_port": "3000"})
	if err != nil || spec.ACMEEnvironment != "production" || spec.Digest == "" {
		t.Fatalf("default ACME environment must be production with a digest: %+v err=%v", spec, err)
	}
}
