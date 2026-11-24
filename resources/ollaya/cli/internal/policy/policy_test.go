package policy

import (
	"path/filepath"
	"runtime"
	"strings"
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

// Every pinned artifact carries a rights disposition: either verified
// behavioral-restriction facts bound to fetched license evidence, or an
// explicit unknown with the reason it is unresolved. A model never ships
// without one.
func TestEveryOllayaArtifactHasExplicitRightsDisposition(t *testing.T) {
	p := shippedPolicy(t)
	for id, model := range p.Models {
		if model.Rights == nil {
			t.Errorf("model %s lacks a rights disposition", id)
			continue
		}
		rights := model.Rights
		if rights.ArtifactID != "ollaya/"+id || rights.SHA256 != model.Digest {
			t.Errorf("model %s rights are not bound to its artifact and manifest digest: %+v", id, rights)
		}
		if len(rights.ReviewTriggers) == 0 {
			t.Errorf("model %s has no rights review trigger", id)
		}
		switch rights.Classification {
		case "unknown":
			if rights.UnverifiedReason == "" {
				t.Errorf("model %s is unknown without saying why", id)
			}
		case "behavioral_restrictions":
			if len(rights.Obligations) == 0 || rights.Evidence.URL == "" || len(rights.Evidence.SHA256) != 64 || rights.Evidence.ObservedAt == "" {
				t.Errorf("model %s is classified without fetched license evidence and obligations: %+v", id, rights)
			}
		default:
			t.Errorf("model %s classification %q is not one the policy ships", id, rights.Classification)
		}
	}
}

// A non-commercial or revenue-capped license is not an option for a pinned
// model; the policy excludes them by name rather than classifying them.
func TestShippedPolicyPinsNoNonCommercialOrRevenueCappedModel(t *testing.T) {
	for id, model := range shippedPolicy(t).Models {
		license := strings.ToLower(model.License)
		for _, banned := range []string{"cc by-nc", "non-commercial", "lfm open"} {
			if strings.Contains(license, banned) {
				t.Errorf("model %s license %q is excluded from the policy", id, model.License)
			}
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

// servable is the smallest model entry the policy accepts.
func servable(digest string) Model {
	return Model{
		Digest:    digest,
		Placement: &ModelPlacement{Device: PlacementDeviceAny, VRAMBytes: 1 << 30, RAMBytes: 2 << 30},
		Limits:    &ModelLimits{ContextTokens: 512, MaxOptions: 255},
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
			Models: map[string]Model{"m": servable("c305a9276531a47000bf93559d2c94f1ed6cbb67055c9151084682dad7655e9d")},
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
		"no placement": func(p *Policy) {
			m := p.Models["m"]
			m.Placement = nil
			p.Models["m"] = m
		},
		"no limits": func(p *Policy) {
			m := p.Models["m"]
			m.Limits = nil
			p.Models["m"] = m
		},
		"one option is not a choice": func(p *Policy) {
			m := p.Models["m"]
			m.Limits = &ModelLimits{ContextTokens: 512, MaxOptions: 1}
			p.Models["m"] = m
		},
		"cpu model without a ram footprint": func(p *Policy) {
			m := p.Models["m"]
			m.Placement = &ModelPlacement{Device: PlacementDeviceCPU}
			p.Models["m"] = m
		},
		"cpu model claiming vram": func(p *Policy) {
			m := p.Models["m"]
			m.Placement = &ModelPlacement{Device: PlacementDeviceCPU, RAMBytes: 1, VRAMBytes: 1}
			p.Models["m"] = m
		},
		"keep alive that is not a duration": func(p *Policy) {
			m := p.Models["m"]
			m.Placement = &ModelPlacement{Device: PlacementDeviceAny, KeepAlive: "soon"}
			p.Models["m"] = m
		},
		"unknown upstream": func(p *Policy) { p.Roles["x"] = Role{Upstream: "nowhere", Model: "m"} },
		"credentials on local": func(p *Policy) {
			u := p.Upstreams["local"]
			u.Credentials = []Credential{{Env: "K", Use: "bearer"}}
			p.Upstreams["local"] = u
		},
		"gpu placement without a footprint": func(p *Policy) {
			m := p.Models["m"]
			m.Placement = &ModelPlacement{Device: PlacementDeviceGPU}
			p.Models["m"] = m
		},
		"unknown placement device": func(p *Policy) {
			m := p.Models["m"]
			m.Placement = &ModelPlacement{Device: "tpu"}
			p.Models["m"] = m
		}, "negative idle unload": func(p *Policy) { p.Residency.IdleUnloadSeconds = -1 },
		"idle unload shorter than a burst gap": func(p *Policy) { p.Residency.IdleUnloadSeconds = 30 },
		"resident role that is not declared":   func(p *Policy) { p.Residency.ResidentRoles = []string{"decide.nowhere"} },
		"resident hosted role":                 func(p *Policy) { p.Residency.ResidentRoles = []string{"decide.hosted"} },
		"resident role listed twice":           func(p *Policy) { p.Residency.ResidentRoles = []string{"decide.local", "decide.local"} },
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

// Contract C5: every servable model declares what it can read and where it
// runs, and a router promises only what all of its targets do.
func TestEveryServableModelDeclaresLimitsAndPlacement(t *testing.T) {
	p := shippedPolicy(t)
	for id, model := range p.Models {
		if model.IsRouter() {
			continue
		}
		if model.Limits == nil || model.Limits.ContextTokens <= 0 || model.Limits.MaxOptions < 2 {
			t.Errorf("model %s declares no usable limits: %+v", id, model.Limits)
		}
		if model.Placement == nil || (model.Placement.VRAMBytes <= 0 && model.Placement.RAMBytes <= 0 && model.Placement.Device != PlacementDeviceAny) {
			t.Errorf("model %s declares no usable placement: %+v", id, model.Placement)
		}
	}
	for name, role := range p.Roles {
		if resolution, err := p.Resolve(name, env(nil)); err == nil && resolution.Locality == LocalityLocal {
			if resolution.Limits == nil || resolution.Placement == nil {
				t.Errorf("local role %s (model %s) resolves without limits or placement", name, role.Model)
			}
		}
	}
}

func TestRouterLimitsAreTheSmallestOfItsTargets(t *testing.T) {
	p := shippedPolicy(t)
	router := p.EffectiveLimits("laya")
	en, multilingual := p.Models["laya:en"].Limits, p.Models["laya:multilingual"].Limits
	if router == nil || router.ContextTokens != minPositive(en.ContextTokens, multilingual.ContextTokens) {
		t.Fatalf("router limits = %+v, want the smaller target context (%d, %d)", router, en.ContextTokens, multilingual.ContextTokens)
	}
	placement := p.EffectivePlacement("laya")
	if placement == nil || placement.Device != PlacementDeviceAny || placement.VRAMBytes < p.Models["laya:en"].Placement.VRAMBytes {
		t.Fatalf("router placement = %+v, want any device and the most demanding target's footprint", placement)
	}
}

func TestGPUOnlyCandidatesDeclareAShortResidency(t *testing.T) {
	p := shippedPolicy(t)
	for id, model := range p.Models {
		if model.RequiresGPU() && model.Placement.VRAMBytes >= 4<<30 && id != "winnow:e4b" && model.Placement.KeepAlive == "" {
			t.Errorf("large GPU-only candidate %s must declare keep_alive so it hands its VRAM back", id)
		}
	}
}
