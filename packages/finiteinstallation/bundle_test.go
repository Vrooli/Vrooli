package finiteinstallation

import (
	"bytes"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
	"time"
)

func bundleInstallationFixture() effortauthority.Installation {
	now := time.Now().UTC().Truncate(time.Second)
	p := effortauthority.Policy{ID: "installed-policy", Owner: "owner", Client: "client", ClientKey: bytes.Repeat([]byte{1}, 32), Epoch: 1, Repository: "/approved/repository", Effort: "effort", Revision: "revision", ContentDigest: effortauthority.Digest("content"), Team: "team", TeamDigest: effortauthority.Digest("team"), BindingDigest: effortauthority.Digest("binding"), Members: []string{"leader"}, Profiles: map[string]string{"bounded": effortauthority.Digest("profile")}, Scopes: []string{"agent-manager:write"}, Effects: []string{"run.create"}, NotBefore: now, Deadline: now.Add(8 * time.Hour), MaxStarts: 8, MaxConcurrent: 1, MaxTurns: 32, MaxToolCalls: 100, MaxRunSeconds: 3600, TotalTurns: 256, TotalToolCalls: 800, TotalRunSeconds: 28800, SurviveLogout: true}
	h := effortauthority.Digest("installed-build")
	return effortauthority.Installation{Enabled: true, Policy: p, Origin: "https://owner.example", Builds: map[string]string{"issuer": h, "pm": h, "am": h}, ServerPins: map[string]string{"issuer": h, "pm": h, "am": h}}
}

func TestProtectedBundleParserExactRoleBuildAndImmutableDescriptor(t *testing.T) {
	cfg := bundleInstallationFixture()
	cfg.Enabled = false
	cfg.Builds["acceptance"] = effortauthority.Digest("acceptance-build")
	cfg.ServerPins["acceptance"] = effortauthority.Digest("acceptance-pin")
	b := Bundle{Version: 1, Role: "pm", Installation: cfg, Endpoints: map[string]string{"am": "https://127.0.0.1:18443", "pm": "https://127.0.0.1:18444", "issuer": "https://127.0.0.1:18445", "acceptance": "https://127.0.0.1:18446"}, ProfileMappings: map[string]string{"bounded": "stored-profile"}, ServerIdentity: "fixture/pm-server", ClientIdentity: "fixture/pm-client", ClientPin: effortauthority.Digest("client")}
	raw, _ := json.Marshal(b)
	plan, e := parseBundle(raw, "pm", cfg.Builds["pm"])
	if e != nil {
		t.Fatal(e)
	}
	first, e := plan.Snapshot()
	if e != nil {
		t.Fatal(e)
	}
	first.Endpoints["pm"] = "https://127.0.0.1:19999"
	first.Installation.Policy.Members[0] = "foreign"
	second, e := plan.Snapshot()
	if e != nil || second.Endpoints["pm"] != b.Endpoints["pm"] || second.Installation.Policy.Members[0] != "leader" {
		t.Fatal("snapshot aliases captured plan")
	}
	for _, change := range []func(*Bundle){func(b *Bundle) { b.Role = "am" }, func(b *Bundle) { b.Installation.Builds["pm"] = effortauthority.Digest("wrong-build") }, func(b *Bundle) { b.Endpoints["pm"] = "http://127.0.0.1:18444" }, func(b *Bundle) { b.ServerIdentity = "" }, func(b *Bundle) { b.Manifest.Enabled = true }} {
		var altered Bundle
		_ = json.Unmarshal(raw, &altered)
		change(&altered)
		bad, _ := json.Marshal(altered)
		if p, e := parseBundle(bad, "pm", cfg.Builds["pm"]); e == nil || p != nil {
			t.Fatal("tampered descriptor accepted")
		}
	}
	if p, e := parseBundle(nil, "pm", cfg.Builds["pm"]); e == nil || p != nil {
		t.Fatal("absent bundle accepted")
	}
}

func TestProtectedBundleRejectsAmbiguousDuplicateFields(t *testing.T) {
	for _, raw := range []string{`{"role":"pm","role":"am"}`, `{"outer":{"pin":"","pin":"other"}}`, `{} {}`, `[1,2`} {
		if exactJSON([]byte(raw)) == nil {
			t.Fatal("ambiguous protected JSON accepted")
		}
	}
	if e := exactJSON([]byte(`{"outer":{"pin":"value"},"items":[1,2]}`)); e != nil {
		t.Fatal(e)
	}
}

func TestProtectedBundleOriginalWindowRefusesBeforeAcquisition(t *testing.T) {
	cfg := bundleInstallationFixture()
	cfg.Enabled = false
	cfg.Builds["acceptance"] = effortauthority.Digest("acceptance-build")
	cfg.ServerPins["acceptance"] = effortauthority.Digest("acceptance-pin")
	b := Bundle{Version: 1, Role: "pm", Installation: cfg, Endpoints: map[string]string{"am": "https://127.0.0.1:18443", "pm": "https://127.0.0.1:18444", "issuer": "https://127.0.0.1:18445", "acceptance": "https://127.0.0.1:18446"}, ProfileMappings: map[string]string{"bounded": "stored-profile"}, ServerIdentity: "fixture/pm-server", ClientIdentity: "fixture/pm-client", ClientPin: effortauthority.Digest("client")}
	raw, _ := json.Marshal(b)
	if plan, e := parseBundleAt(raw, "pm", cfg.Builds["pm"], cfg.Policy.NotBefore.Add(-time.Second)); e != nil || plan == nil {
		t.Fatal("disabled commissioning cannot prepare future window")
	}
	live := b
	live.Installation.Enabled = true
	liveRaw, _ := json.Marshal(live)
	if plan, e := parseBundleAt(liveRaw, "pm", cfg.Builds["pm"], cfg.Policy.NotBefore.Add(-time.Second)); e == nil || plan != nil {
		t.Fatal("future live startup accepted")
	}
	for _, now := range []time.Time{cfg.Policy.Deadline, cfg.Policy.Deadline.Add(time.Second)} {
		if plan, e := parseBundleAt(raw, "pm", cfg.Builds["pm"], now); e == nil || plan != nil {
			t.Fatal("outside original owner window accepted")
		}
	}
}
