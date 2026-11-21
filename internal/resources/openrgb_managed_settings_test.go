package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRGBManagedSettingsKeepPreferencesWithoutRemoteControllers(t *testing.T) {
	current := []byte(`{"Client":{"clients":[{"ip":"192.0.2.2","port":6742}],"name":"desk"},"Server":{"all_controllers":true,"legacy_workaround":true},"UI":{"color":"amber"},"ProfileManager":{"service_startup_profile":{"enabled":true,"name":"unowned.orp"}}}`)
	frozen, err := mergeOpenRGBConfigContentWithSelection(current, `{"Detectors":{"detectors":{"local":false}}}`, []string{"local"})
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(frozen), &root); err != nil {
		t.Fatal(err)
	}
	if _, err := readOpenRGBDetectorState([]byte(frozen)); err != nil {
		t.Fatalf("managed freeze is not witnessable: %v", err)
	}
	var preferences struct {
		UI     struct{ Color string }
		Client struct{ Name string }
		Server struct {
			Legacy bool `json:"legacy_workaround"`
		}
	}
	if err := json.Unmarshal([]byte(frozen), &preferences); err != nil || preferences.UI.Color != "amber" || preferences.Client.Name != "desk" || !preferences.Server.Legacy {
		t.Fatalf("unrelated preferences lost: %+v %v", preferences, err)
	}
	for _, change := range []struct {
		section string
		value   string
	}{
		{openRGBClientSettings, `{"clients":[{"ip":"192.0.2.3","port":6742}]}`},
		{openRGBClientSettings, `{"clients":null}`},
		{openRGBServerSettings, `{"all_controllers":true}`},
		{openRGBServerSettings, `{}`},
		{openRGBProfilesSettings, `{"service_startup_profile":{"enabled":true,"name":"other.orp"}}`},
	} {
		t.Run(change.section+change.value, func(t *testing.T) {
			mutated := make(map[string]json.RawMessage, len(root))
			for key, value := range root {
				mutated[key] = value
			}
			mutated[change.section] = json.RawMessage(change.value)
			live, err := json.Marshal(mutated)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := readOpenRGBDetectorState(live); err == nil {
				t.Fatal("remote configuration passed pre-exec settings admission")
			}
			if err := verifyOpenRGBServingDetectors([]byte(frozen), live); err == nil {
				t.Fatal("remote configuration passed live serving publication")
			}
		})
	}
	for _, current := range []string{`{"Client":[]}`, `{"Server":null}`, `{"ProfileManager":[]}`} {
		if _, err := mergeOpenRGBConfigContentWithSelection([]byte(current), `{"Detectors":{"detectors":{"local":false}}}`, nil); err == nil {
			t.Fatalf("malformed network settings silently replaced: %s", current)
		}
	}
}

func TestOpenRGBRuntimeNormalizationRetainsWitnessedConfigForAdmissionAndStop(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	contract := x.update.Rollback
	var live map[string]json.RawMessage
	if err := json.Unmarshal(contract.ConfigBytes, &live); err != nil {
		t.Fatal(err)
	}
	var detectors map[string]json.RawMessage
	var states map[string]bool
	if err := json.Unmarshal(live["Detectors"], &detectors); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(detectors["detectors"], &states); err != nil {
		t.Fatal(err)
	}
	pruned := false
	for name, enabled := range states {
		if !enabled {
			delete(states, name)
			pruned = true
			break
		}
	}
	if !pruned {
		t.Fatal("fixture lacks removable disabled key")
	}
	detectors["detectors"], _ = json.Marshal(states)
	live["Detectors"], _ = json.Marshal(detectors)
	normalized, _ := json.MarshalIndent(live, "", "    ") // Runtime uses4 spaces, no trailing newline.
	if err := verifyOpenRGBNormalizedServingSettings(contract.ConfigBytes, normalized); err != nil {
		t.Fatalf("harmless normalization failed: %v", err)
	}
	u := x.update
	runtime := x.fence.Intent.Runtime
	u.Original.UpdateLaunch = &ManagedUpdateServingLaunch{OperationID: u.ID, Runtime: runtime, Contract: contract}
	root, err := runtime.roleRoot(u.ID, u.Original.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(root, "config")
	path := filepath.Join(configDir, contract.Manifest.ManagedService.Config.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, normalized, 0600); err != nil {
		t.Fatal(err)
	}
	retained, err := freezeManagedUpdateServingConfig(u.Original, configDir, contract)
	if err != nil || retained.ConfigSHA256 != contract.ConfigSHA256 || retained.Digest != contract.Digest || string(retained.ConfigBytes) != string(contract.ConfigBytes) {
		t.Fatalf("adoption changed consumed config contract: %v", err)
	}
	if err := verifyManagedUpdateOriginalConfig(u); err != nil {
		t.Fatalf("exclusive stop rejected normalized owned settings: %v", err)
	}
	live["UI"] = json.RawMessage(`{"unexpected_preference_change":true}`)
	changedPreference, _ := json.Marshal(live)
	if err := verifyOpenRGBNormalizedServingSettings(contract.ConfigBytes, changedPreference); err == nil {
		t.Fatal("normalization ignored unrelated config drift")
	}
	delete(live, "UI")
	live[openRGBClientSettings] = json.RawMessage(`{"clients":[{"ip":"192.0.2.4","port":6742}]}`)
	unsafe, _ := json.Marshal(live)
	if err := os.WriteFile(path, unsafe, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := freezeManagedUpdateServingConfig(u.Original, configDir, contract); err == nil {
		t.Fatal("adoption sanitized away unsafe network scope")
	}
	if err := verifyManagedUpdateOriginalConfig(u); err == nil {
		t.Fatal("exclusive stop hash normalized away unsafe network scope")
	}
}

func TestOpenRGBPinnedRuntimeNormalizationFixture(t *testing.T) {
	dir := os.Getenv("VROOLI_OPENRGB_NORMALIZATION_FIXTURE")
	if dir == "" {
		t.Skip("set explicit device-free pinned runtime normalization evidence directory")
	}
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	frozen, serving, exited := read("before.json"), read("serving.json"), read("after-exit.json")
	before, err := readOpenRGBDetectorState(frozen)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := readOpenRGBDetectorState(serving)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 1948 || len(actual) != 1931 {
		t.Fatalf("unexpected qualified runtime transformation: %d -> %d", len(before), len(actual))
	}
	for _, enabled := range actual {
		if enabled {
			t.Fatal("device-free fixture enabled a detector")
		}
	}
	if err := verifyOpenRGBNormalizedServingSettings(frozen, serving); err != nil {
		t.Fatalf("actual serving normalization rejected: %v", err)
	}
	if err := verifyOpenRGBNormalizedServingSettings(frozen, exited); err != nil {
		t.Fatalf("actual shutdown normalization rejected: %v", err)
	}
	if string(serving) != string(exited) {
		t.Fatal("settings changed at shutdown rather than only initialization")
	}
}

func TestOpenRGBAutomaticProfileValidationUsesUpstreamExactKeys(t *testing.T) {
	frozen, err := mergeOpenRGBConfigContentWithSelection(nil, `{"Detectors":{"detectors":{"local":false}}}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	var base map[string]json.RawMessage
	if err := json.Unmarshal([]byte(frozen), &base); err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{
		`{"enabled":true,"Enabled":false,"name":"unowned.orp","Name":""}`,
		`{"enabled":false,"name":"","unexpected_profile_option":true}`,
	} {
		var live map[string]json.RawMessage
		_ = json.Unmarshal([]byte(frozen), &live)
		var profiles map[string]json.RawMessage
		_ = json.Unmarshal(live[openRGBProfilesSettings], &profiles)
		profiles[openRGBProfileServiceStart] = json.RawMessage(event)
		live[openRGBProfilesSettings], _ = json.Marshal(profiles)
		data, _ := json.Marshal(live)
		if _, err := readOpenRGBDetectorState(data); err == nil {
			t.Fatalf("profile scope accepted noncanonical exact keys: %s", event)
		}
		if err := verifyOpenRGBNormalizedServingSettings([]byte(frozen), data); err == nil {
			t.Fatalf("normalization erased changed profile object: %s", event)
		}
	}
	var server map[string]json.RawMessage
	_ = json.Unmarshal(base[openRGBServerSettings], &server)
	delete(server, openRGBServerDefaultHost)
	base[openRGBServerSettings], _ = json.Marshal(server)
	data, _ := json.Marshal(base)
	if _, err := readOpenRGBDetectorState(data); err != nil {
		t.Fatalf("fixture changed safety scope: %v", err)
	}
	if err := verifyOpenRGBNormalizedServingSettings([]byte(frozen), data); err == nil {
		t.Fatal("normalization regenerated unrelated missing settings")
	}
}
