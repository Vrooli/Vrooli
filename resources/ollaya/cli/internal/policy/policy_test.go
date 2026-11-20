package policy

import (
	"path/filepath"
	"runtime"
	"testing"
)

func shippedPolicy(t *testing.T) Policy {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	p, err := Load(filepath.Join(filepath.Dir(file), "..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("shipped model-policy.json must load: %v", err)
	}
	return p
}

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestRightsContextPathUsesSharedConfiguredOwnerPath(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "operator-context.json")
	if got := RightsContextPath(env(map[string]string{"VROOLI_MODEL_RIGHTS_CONTEXT_PATH": custom})); got != custom {
		t.Fatalf("configured rights context path=%q, want %q", got, custom)
	}
	root := t.TempDir()
	want := filepath.Join(root, "scenarios", "ai-gateway", "config", "model-rights-context.json")
	if got := RightsContextPath(env(map[string]string{"VROOLI_SOURCE_ROOT": root})); got != want {
		t.Fatalf("source-root rights context path=%q, want %q", got, want)
	}
}

func TestShippedPolicyDeclaresLocalAndCredentialGatedHostedRoles(t *testing.T) {
	p := shippedPolicy(t)
	want := map[string]string{
		"decide.local": LocalityLocal, "decide.local.accurate": LocalityLocal,
		"decide.jev.edenai": LocalityRemote, "decide.jev.typesafe": LocalityRemote, "decide.clef.cloudflare": LocalityRemote,
	}
	for role, locality := range want {
		resolution, err := p.Resolve(role, env(nil))
		if err != nil {
			t.Fatalf("resolve %s: %v", role, err)
		}
		if resolution.Locality != locality {
			t.Fatalf("%s locality = %s, want %s", role, resolution.Locality, locality)
		}
		if (locality == LocalityRemote) == resolution.Configured() {
			t.Fatalf("%s configured=%t with no credentials; hosted roles must need one and local roles none", role, resolution.Configured())
		}
	}
	if got := p.DefaultPulls(); len(got) != 3 || got[0] != "laya" {
		t.Fatalf("default pulls = %v, want the laya router first then its two targets", got)
	}
}

func TestEveryOllayaArtifactHasExplicitRightsDisposition(t *testing.T) {
	p := shippedPolicy(t)
	if len(p.Models) != 4 {
		t.Fatalf("shipped Ollaya artifact count=%d, want 4", len(p.Models))
	}
	for id, model := range p.Models {
		if model.Rights == nil {
			t.Errorf("model %s lacks a rights disposition", id)
			continue
		}
		if model.Rights.ArtifactID != "ollaya/"+id {
			t.Errorf("model %s rights artifact identity=%q", id, model.Rights.ArtifactID)
		}
		if model.Rights.Classification != "unknown" || model.Rights.UnverifiedReason == "" {
			t.Errorf("model %s must remain explicitly unknown until its exact license scope is verified: %+v", id, model.Rights)
		}
		if len(model.Rights.ReviewTriggers) == 0 {
			t.Errorf("model %s has no rights review trigger", id)
		}
	}
}

func TestResolveBindsCredentialsWithoutSerializingThem(t *testing.T) {
	p := shippedPolicy(t)
	resolution, err := p.Resolve("decide.clef.cloudflare", env(map[string]string{
		"CLOUDFLARE_WORKERS_AI_API_TOKEN": "tok", "CLOUDFLARE_WORKERS_AI_ACCOUNT_ID": "acct/1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Configured() || resolution.Token != "tok" {
		t.Fatalf("resolution = %+v, want configured with bearer token", resolution)
	}
	if resolution.Endpoint != "https://api.cloudflare.com/client/v4/accounts/acct%2F1/ai/run/@cf/cloudflare/clef-flash" {
		t.Fatalf("endpoint = %s, want the escaped account substituted", resolution.Endpoint)
	}
	partial, _ := p.Resolve("decide.clef.cloudflare", env(map[string]string{"CLOUDFLARE_WORKERS_AI_API_TOKEN": "tok"}))
	if partial.Configured() || len(partial.Missing) != 1 || partial.Missing[0] != "CLOUDFLARE_WORKERS_AI_ACCOUNT_ID" {
		t.Fatalf("missing = %v, want only the account id", partial.Missing)
	}
}

func TestValidateRejectsUnsafePolicies(t *testing.T) {
	base := func() Policy {
		return Policy{
			SchemaVersion: "x",
			Upstreams: map[string]Upstream{
				"local":  {Locality: LocalityLocal, Protocol: ProtocolOllaya},
				"hosted": {Locality: LocalityRemote, Protocol: ProtocolSystemOne, Endpoint: "https://example.test/v1/systemone", Credentials: []Credential{{Env: "K", Use: "bearer"}}},
			},
			Roles:  map[string]Role{"decide.local": {Upstream: "local", Model: "m"}, "decide.hosted": {Upstream: "hosted", Model: "jev"}},
			Models: map[string]Model{"m": {Digest: "c305a9276531a47000bf93559d2c94f1ed6cbb67055c9151084682dad7655e9d"}},
		}
	}
	if err := base().Validate(); err != nil {
		t.Fatalf("base policy must validate: %v", err)
	}
	cases := map[string]func(*Policy){
		"plain http hosted endpoint": func(p *Policy) {
			u := p.Upstreams["hosted"]
			u.Endpoint = "http://example.test"
			p.Upstreams["hosted"] = u
		},
		"hosted without bearer": func(p *Policy) {
			u := p.Upstreams["hosted"]
			u.Credentials = nil
			p.Upstreams["hosted"] = u
		},
		"local role on unpinned model": func(p *Policy) { p.Roles["decide.local"] = Role{Upstream: "local", Model: "other"} },
		"bad digest":                   func(p *Policy) { p.Models["m"] = Model{Digest: "latest"} },
		"unknown upstream":             func(p *Policy) { p.Roles["x"] = Role{Upstream: "nowhere", Model: "m"} },
		"credentials on local": func(p *Policy) {
			u := p.Upstreams["local"]
			u.Credentials = []Credential{{Env: "K", Use: "bearer"}}
			p.Upstreams["local"] = u
		},
		"gpu placement without a footprint": func(p *Policy) {
			p.Models["m"] = Model{Digest: "c305a9276531a47000bf93559d2c94f1ed6cbb67055c9151084682dad7655e9d", Placement: &ModelPlacement{Device: PlacementDeviceGPU}}
		},
		"unknown placement device": func(p *Policy) {
			p.Models["m"] = Model{Digest: "c305a9276531a47000bf93559d2c94f1ed6cbb67055c9151084682dad7655e9d", Placement: &ModelPlacement{Device: "tpu"}}
		},
	}
	for name, mutate := range cases {
		p := base()
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Fatalf("%s: Validate() = nil, want rejection", name)
		}
	}
}

func TestShippedPolicyMarksWinnowGPUOnlyAndAllowsLayaOnCPU(t *testing.T) {
	p := shippedPolicy(t)
	winnow, ok := p.Models["winnow:e4b"]
	if !ok || !winnow.RequiresGPU() {
		t.Fatalf("winnow:e4b = %+v, want a GPU-required placement", winnow)
	}
	if winnow.Placement.VRAMBytes <= winnow.DiskBytes {
		t.Fatalf("winnow vram_bytes %d must exceed its on-disk size %d", winnow.Placement.VRAMBytes, winnow.DiskBytes)
	}
	for _, name := range []string{"laya", "laya:en", "laya:multilingual"} {
		if p.Models[name].RequiresGPU() {
			t.Fatalf("%s must remain CPU-capable", name)
		}
	}
}
