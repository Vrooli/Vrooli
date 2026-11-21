package resources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	openRGBHIDSafeMode          = "hid_safe_mode"
	openRGBClientSettings       = "Client"
	openRGBSavedClients         = "clients"
	openRGBServerSettings       = "Server"
	openRGBExportAllControllers = "all_controllers"
	openRGBServerDefaultHost    = "default_host"
	openRGBServerDefaultPort    = "default_port"
	openRGBServerLegacy         = "legacy_workaround"
	openRGBServerLoopback       = "127.0.0.1"
	openRGBProfilesSettings     = "ProfileManager"
	openRGBProfileOpen          = "open_profile"
	openRGBProfileExit          = "exit_profile"
	openRGBProfileResume        = "resume_profile"
	openRGBProfileSuspend       = "suspend_profile"
	openRGBProfileServiceStart  = "service_startup_profile"
	openRGBProfileServiceStop   = "service_shutdown_profile"
	openRGBDisabledProfileJSON  = `{"enabled":false,"name":""}`
)

// OpenRGB1.1 ResourceManager initializes saved clients even in server mode.
// Managed headless operation owns selected local hardware only; preferences
// cannot introduce a remote client or export its controllers through our SDK.
func freezeOpenRGBLocalSettings(root map[string]json.RawMessage) error {
	var detectorSettings map[string]json.RawMessage
	if err := json.Unmarshal(root["Detectors"], &detectorSettings); err != nil || detectorSettings == nil {
		return errors.New("OpenRGB detector settings must be an object")
	}
	detectorSettings[openRGBHIDSafeMode] = json.RawMessage("true")
	encodedDetectors, err := json.Marshal(detectorSettings)
	if err != nil {
		return err
	}
	root["Detectors"] = encodedDetectors
	for _, rule := range []struct {
		section, key string
		value        json.RawMessage
	}{
		{openRGBClientSettings, openRGBSavedClients, json.RawMessage("[]")},
		{openRGBServerSettings, openRGBExportAllControllers, json.RawMessage("false")},
	} {
		settings := map[string]json.RawMessage{}
		if existing, ok := root[rule.section]; ok {
			if err := json.Unmarshal(existing, &settings); err != nil || settings == nil {
				return fmt.Errorf("OpenRGB %s settings must be a JSON object", rule.section)
			}
		}
		settings[rule.key] = rule.value
		encoded, err := json.Marshal(settings)
		if err != nil {
			return err
		}
		root[rule.section] = encoded
	}
	// These are the exact1.1 constructor defaults. Freeze them before exec so
	// harmless runtime default insertion cannot redefine the witnessed hash.
	var server map[string]json.RawMessage
	_ = json.Unmarshal(root[openRGBServerSettings], &server)
	host, _ := json.Marshal(openRGBServerLoopback)
	for key, value := range map[string]json.RawMessage{openRGBServerDefaultHost: host, openRGBServerDefaultPort: json.RawMessage("6742"), openRGBServerLegacy: json.RawMessage("false")} {
		if _, exists := server[key]; !exists {
			server[key] = value
		}
	}
	encoded, err := json.Marshal(server)
	if err != nil {
		return err
	}
	root[openRGBServerSettings] = encoded
	profiles := map[string]json.RawMessage{}
	if existing, ok := root[openRGBProfilesSettings]; ok {
		if err := json.Unmarshal(existing, &profiles); err != nil || profiles == nil {
			return errors.New("OpenRGB profile manager settings must be a JSON object")
		}
	}
	for _, key := range openRGBAutomaticProfileKeys() {
		profiles[key] = json.RawMessage(openRGBDisabledProfileJSON)
	}
	encoded, err = json.Marshal(profiles)
	if err != nil {
		return err
	}
	root[openRGBProfilesSettings] = encoded
	return nil
}

func openRGBAutomaticProfileKeys() []string {
	return []string{openRGBProfileOpen, openRGBProfileExit, openRGBProfileResume, openRGBProfileSuspend, openRGBProfileServiceStart, openRGBProfileServiceStop}
}

func verifyOpenRGBLocalSettings(root map[string]json.RawMessage) error {
	var detectorSettings map[string]json.RawMessage
	var safeMode *bool
	if json.Unmarshal(root["Detectors"], &detectorSettings) != nil || json.Unmarshal(detectorSettings[openRGBHIDSafeMode], &safeMode) != nil || safeMode == nil || !*safeMode {
		return errors.New("OpenRGB must use finite HID safe detection without autonomous hotplug probes")
	}
	var client, server map[string]json.RawMessage
	var clients []json.RawMessage
	var exportAll *bool
	if json.Unmarshal(root[openRGBClientSettings], &client) != nil || json.Unmarshal(client[openRGBSavedClients], &clients) != nil || clients == nil || len(clients) != 0 {
		return errors.New("OpenRGB managed settings must explicitly disable saved remote clients")
	}
	if json.Unmarshal(root[openRGBServerSettings], &server) != nil || json.Unmarshal(server[openRGBExportAllControllers], &exportAll) != nil || exportAll == nil || *exportAll {
		return errors.New("OpenRGB managed SDK must export selected local controllers only")
	}
	var profiles map[string]json.RawMessage
	if json.Unmarshal(root[openRGBProfilesSettings], &profiles) != nil || profiles == nil {
		return errors.New("OpenRGB automatic profiles must be explicitly disabled")
	}
	for _, key := range openRGBAutomaticProfileKeys() {
		var profile map[string]json.RawMessage
		var enabled *bool
		var name *string
		if json.Unmarshal(profiles[key], &profile) != nil || len(profile) != 2 || json.Unmarshal(profile["enabled"], &enabled) != nil || enabled == nil || *enabled || json.Unmarshal(profile["name"], &name) != nil || name == nil || *name != "" {
			return errors.New("OpenRGB automatic profiles cannot override managed desired output")
		}
	}
	return nil
}

// Normalize object ordering before freezing bytes. RawMessage's nested key
// order would otherwise differ from the official runtime's sorted JSON dump.
// UseNumber preserves large integer preferences instead of rounding to float64.
func marshalOpenRGBSettings(root map[string]json.RawMessage) ([]byte, error) {
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var settings any
	if err := decoder.Decode(&settings); err != nil {
		return nil, err
	}
	return json.MarshalIndent(settings, "", "  ")
}
