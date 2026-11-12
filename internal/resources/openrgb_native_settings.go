package resources

import (
	"encoding/json"
	"errors"
)

// OpenRGB 1.1's normal HID path starts hotplug callbacks. Its safe mode uses
// one-shot enumeration instead, leaving reconnect and new-device admission to
// the Vrooli owner rather than probing a newly attached unenrolled controller.
func enforceNativeOpenRGBSettings(data []byte) ([]byte, error) {
	if err := validateManagedUpdateJSON(data); err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var detectors map[string]json.RawMessage
	if err := json.Unmarshal(root["Detectors"], &detectors); err != nil || detectors == nil {
		return nil, errors.Join(errors.New("native OpenRGB requires explicit detector settings"), err)
	}
	detectors["hid_safe_mode"] = json.RawMessage("true")
	encoded, err := json.Marshal(detectors)
	if err != nil {
		return nil, err
	}
	root["Detectors"] = encoded
	return json.Marshal(root)
}

func verifyNativeOpenRGBSettings(data []byte) error {
	var root struct {
		Detectors struct {
			HIDSafeMode *bool `json:"hid_safe_mode"`
		} `json:"Detectors"`
	}
	if err := json.Unmarshal(data, &root); err != nil || root.Detectors.HIDSafeMode == nil || !*root.Detectors.HIDSafeMode {
		return errors.Join(errors.New("native OpenRGB must use one-shot safe HID detection without upstream hotplug callbacks"), err)
	}
	return nil
}
