package finiteinstallation

import (
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
)

func TestDevelopmentDescriptorIsAcceptanceOnlyAndImmutable(t *testing.T) {
	cfg := bundleInstallationFixture()
	cfg.Enabled = false
	cfg.Builds["acceptance"] = effortauthority.Digest("acceptance-build")
	cfg.ServerPins["acceptance"] = effortauthority.Digest("acceptance-pin")
	b := Bundle{Version: 1, Role: "acceptance", Installation: cfg, Endpoints: map[string]string{"am": "https://127.0.0.1:18443", "pm": "https://127.0.0.1:18444", "issuer": "https://127.0.0.1:18445", "acceptance": "https://127.0.0.1:18446"}, ProfileMappings: map[string]string{"bounded": "stored-profile"}, CommissionTarget: CommissionTarget{WorkShape: "bounded-authority", Kind: "scenario", Name: "fixture"}, ServerIdentity: "fixture/acceptance-server", DevelopmentSetup: json.RawMessage(`{"Version":1}`)}
	raw, _ := json.Marshal(b)
	plan, e := parseBundle(raw, "acceptance", cfg.Builds["acceptance"])
	if e != nil {
		t.Fatal(e)
	}
	first, _ := plan.Snapshot()
	first.DevelopmentSetup[0] = '['
	second, _ := plan.Snapshot()
	if string(second.DevelopmentSetup) != string(b.DevelopmentSetup) {
		t.Fatal("setup aliases bundle")
	}
	b.Role = "pm"
	b.ClientIdentity = "fixture/pm-client"
	b.ClientPin = effortauthority.Digest("client")
	raw, _ = json.Marshal(b)
	if plan, e = parseBundle(raw, "pm", cfg.Builds["pm"]); e == nil || plan != nil {
		t.Fatal("run role accepted Development setup")
	}
	b.Role = "acceptance"
	b.DevelopmentSetup = json.RawMessage(`null`)
	raw, _ = json.Marshal(b)
	if plan, e = parseBundle(raw, "acceptance", cfg.Builds["acceptance"]); e == nil || plan != nil {
		t.Fatal("null setup accepted")
	}
}
