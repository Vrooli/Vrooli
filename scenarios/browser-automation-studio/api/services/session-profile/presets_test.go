package sessionprofile

import (
	"testing"
)

func TestGetPresetBrowserProfile_ValidPresets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		preset         string
		expectNil      bool
		expectedPreset string
	}{
		{"stealth", PresetStealth, false, PresetStealth},
		{"balanced", PresetBalanced, false, PresetBalanced},
		{"fast", PresetFast, false, PresetFast},
		{"none", PresetNone, false, PresetNone},
		{"unknown", "unknown", true, ""},
		{"empty", "", true, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := GetPresetBrowserProfile(tc.preset)

			if tc.expectNil {
				if result != nil {
					t.Errorf("expected nil for preset %q, got profile with preset %q", tc.preset, result.Preset)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected non-nil profile for preset %q", tc.preset)
			}

			if result.Preset != tc.expectedPreset {
				t.Errorf("expected preset %q, got %q", tc.expectedPreset, result.Preset)
			}
		})
	}
}

func TestGetPresetBrowserProfile_Stealth_HasExpectedSettings(t *testing.T) {
	t.Parallel()

	profile := GetPresetBrowserProfile(PresetStealth)
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	// Verify fingerprint settings exist
	if profile.Fingerprint == nil {
		t.Error("stealth preset should have fingerprint settings")
	} else {
		if profile.Fingerprint.DeviceScaleFactor != 1 {
			t.Errorf("expected device_scale_factor 1, got %v", profile.Fingerprint.DeviceScaleFactor)
		}
		if profile.Fingerprint.HardwareConcurrency != 4 {
			t.Errorf("expected hardware_concurrency 4, got %d", profile.Fingerprint.HardwareConcurrency)
		}
		if profile.Fingerprint.DeviceMemory != 8 {
			t.Errorf("expected device_memory 8, got %d", profile.Fingerprint.DeviceMemory)
		}
	}

	// Verify behavior settings
	if profile.Behavior == nil {
		t.Error("stealth preset should have behavior settings")
	} else {
		if profile.Behavior.MouseMovementStyle != "natural" {
			t.Errorf("expected mouse_movement_style 'natural', got %q", profile.Behavior.MouseMovementStyle)
		}
		if profile.Behavior.TypingDelayMin != 50 {
			t.Errorf("expected typing_delay_min 50, got %d", profile.Behavior.TypingDelayMin)
		}
		if profile.Behavior.TypingDelayMax != 150 {
			t.Errorf("expected typing_delay_max 150, got %d", profile.Behavior.TypingDelayMax)
		}
		if !profile.Behavior.MicroPauseEnabled {
			t.Error("expected micro_pause_enabled true for stealth preset")
		}
	}

	// Verify anti-detection settings
	if profile.AntiDetection == nil {
		t.Error("stealth preset should have anti-detection settings")
	} else {
		if !profile.AntiDetection.DisableAutomationControlled {
			t.Error("expected disable_automation_controlled true for stealth preset")
		}
		if !profile.AntiDetection.DisableWebRTC {
			t.Error("expected disable_webrtc true for stealth preset")
		}
		if !profile.AntiDetection.PatchNavigatorWebdriver {
			t.Error("expected patch_navigator_webdriver true for stealth preset")
		}
		if profile.AntiDetection.AdBlockingMode != "ads_and_tracking" {
			t.Errorf("expected ad_blocking_mode 'ads_and_tracking', got %q", profile.AntiDetection.AdBlockingMode)
		}
	}
}

func TestGetPresetBrowserProfile_Balanced_HasExpectedSettings(t *testing.T) {
	t.Parallel()

	profile := GetPresetBrowserProfile(PresetBalanced)
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	// Balanced should not have fingerprint settings
	if profile.Fingerprint != nil {
		t.Error("balanced preset should not have custom fingerprint settings")
	}

	// Verify behavior settings (should be less aggressive than stealth)
	if profile.Behavior == nil {
		t.Error("balanced preset should have behavior settings")
	} else {
		if profile.Behavior.MouseMovementStyle != "bezier" {
			t.Errorf("expected mouse_movement_style 'bezier', got %q", profile.Behavior.MouseMovementStyle)
		}
		// Balanced should have faster typing than stealth
		if profile.Behavior.TypingDelayMin != 30 {
			t.Errorf("expected typing_delay_min 30, got %d", profile.Behavior.TypingDelayMin)
		}
		if profile.Behavior.TypingDelayMax != 80 {
			t.Errorf("expected typing_delay_max 80, got %d", profile.Behavior.TypingDelayMax)
		}
	}

	// Verify anti-detection (should be more limited than stealth)
	if profile.AntiDetection == nil {
		t.Error("balanced preset should have anti-detection settings")
	} else {
		if !profile.AntiDetection.DisableAutomationControlled {
			t.Error("expected disable_automation_controlled true")
		}
		// Should not disable WebRTC (less aggressive than stealth)
		if profile.AntiDetection.DisableWebRTC {
			t.Error("balanced preset should not disable WebRTC")
		}
	}
}

func TestGetPresetBrowserProfile_Fast_HasExpectedSettings(t *testing.T) {
	t.Parallel()

	profile := GetPresetBrowserProfile(PresetFast)
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	// Fast should not have fingerprint settings
	if profile.Fingerprint != nil {
		t.Error("fast preset should not have custom fingerprint settings")
	}

	// Verify behavior settings (should be fastest)
	if profile.Behavior == nil {
		t.Error("fast preset should have behavior settings")
	} else {
		if profile.Behavior.MouseMovementStyle != "linear" {
			t.Errorf("expected mouse_movement_style 'linear', got %q", profile.Behavior.MouseMovementStyle)
		}
		// Fast should have minimal delays
		if profile.Behavior.TypingDelayMin != 10 {
			t.Errorf("expected typing_delay_min 10, got %d", profile.Behavior.TypingDelayMin)
		}
		if profile.Behavior.TypingDelayMax != 30 {
			t.Errorf("expected typing_delay_max 30, got %d", profile.Behavior.TypingDelayMax)
		}
		// Micro pauses should be disabled for speed
		if profile.Behavior.MicroPauseEnabled {
			t.Error("expected micro_pause_enabled false for fast preset")
		}
	}
}

func TestGetPresetBrowserProfile_None_HasMinimalSettings(t *testing.T) {
	t.Parallel()

	profile := GetPresetBrowserProfile(PresetNone)
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	// None should have no fingerprint
	if profile.Fingerprint != nil {
		t.Error("none preset should not have fingerprint settings")
	}

	// None should have no behavior modifications
	if profile.Behavior != nil {
		t.Error("none preset should not have behavior settings")
	}

	// None should have anti-detection with blocking disabled
	if profile.AntiDetection == nil {
		t.Error("none preset should have anti-detection settings")
	} else {
		if profile.AntiDetection.AdBlockingMode != "none" {
			t.Errorf("expected ad_blocking_mode 'none', got %q", profile.AntiDetection.AdBlockingMode)
		}
	}
}

func TestGetPresetBrowserProfile_AllPresetsAreValid(t *testing.T) {
	t.Parallel()

	presets := []string{PresetStealth, PresetBalanced, PresetFast, PresetNone}

	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			t.Parallel()

			profile := GetPresetBrowserProfile(preset)
			if profile == nil {
				t.Fatalf("preset %q returned nil", preset)
			}

			// Validate each preset passes validation
			if err := ValidateBrowserProfile(profile); err != nil {
				t.Errorf("preset %q failed validation: %v", preset, err)
			}
		})
	}
}
