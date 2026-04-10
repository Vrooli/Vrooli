// Package sessionprofile provides session profile management for browser automation.
package sessionprofile

import (
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// Preset names for browser profiles.
const (
	PresetStealth  = "stealth"
	PresetBalanced = "balanced"
	PresetFast     = "fast"
	PresetNone     = "none"
)

// GetPresetBrowserProfile returns the default settings for a given preset name.
// Returns nil for "none" preset or unknown preset names.
func GetPresetBrowserProfile(preset string) *persistence.BrowserProfile {
	switch preset {
	case PresetStealth:
		return &persistence.BrowserProfile{
			Preset: PresetStealth,
			Fingerprint: &persistence.FingerprintSettings{
				DeviceScaleFactor:   1,
				HardwareConcurrency: 4,
				DeviceMemory:        8,
			},
			Behavior: &persistence.BehaviorSettings{
				TypingDelayMin:        50,
				TypingDelayMax:        150,
				TypingStartDelayMin:   100,
				TypingStartDelayMax:   300,
				TypingPasteThreshold:  200, // Paste text longer than 200 chars
				TypingVarianceEnabled: true,
				MouseMovementStyle:    "natural",
				MouseJitterAmount:     2,
				ClickDelayMin:         100,
				ClickDelayMax:         300,
				ScrollStyle:           "smooth",
				MicroPauseEnabled:     true,
				MicroPauseMinMs:       200,
				MicroPauseMaxMs:       800,
				MicroPauseFrequency:   0.15,
			},
			AntiDetection: &persistence.AntiDetectionSettings{
				DisableAutomationControlled: true,
				DisableWebRTC:               true,
				PatchNavigatorWebdriver:     true,
				PatchNavigatorPlugins:       true,
				PatchNavigatorLanguages:     true,
				PatchWebGL:                  true,
				PatchCanvas:                 true,
				PatchAudioContext:           true,
				HeadlessDetectionBypass:     true,
				PatchFonts:                  true,
				PatchScreenProperties:       true,
				PatchBatteryAPI:             true,
				PatchConnectionAPI:          true,
				AdBlockingMode:              "ads_and_tracking",
			},
		}
	case PresetBalanced:
		return &persistence.BrowserProfile{
			Preset: PresetBalanced,
			Behavior: &persistence.BehaviorSettings{
				TypingDelayMin:        30,
				TypingDelayMax:        80,
				TypingStartDelayMin:   50,
				TypingStartDelayMax:   150,
				TypingPasteThreshold:  150, // Paste text longer than 150 chars
				TypingVarianceEnabled: true,
				MouseMovementStyle:    "bezier",
				ClickDelayMin:         50,
				ClickDelayMax:         150,
				MicroPauseEnabled:     true,
				MicroPauseMinMs:       100,
				MicroPauseMaxMs:       400,
				MicroPauseFrequency:   0.08,
			},
			AntiDetection: &persistence.AntiDetectionSettings{
				DisableAutomationControlled: true,
				PatchNavigatorWebdriver:     true,
				PatchAudioContext:           true,
				PatchFonts:                  true,
				PatchScreenProperties:       true,
				AdBlockingMode:              "ads_and_tracking",
			},
		}
	case PresetFast:
		return &persistence.BrowserProfile{
			Preset: PresetFast,
			Behavior: &persistence.BehaviorSettings{
				TypingDelayMin:        10,
				TypingDelayMax:        30,
				TypingStartDelayMin:   10,
				TypingStartDelayMax:   30,
				TypingPasteThreshold:  100, // Paste text longer than 100 chars
				TypingVarianceEnabled: true,
				MouseMovementStyle:    "linear",
				ClickDelayMin:         20,
				ClickDelayMax:         50,
				MicroPauseEnabled:     false,
			},
			AntiDetection: &persistence.AntiDetectionSettings{
				DisableAutomationControlled: true,
				PatchNavigatorWebdriver:     true,
				AdBlockingMode:              "ads_and_tracking",
			},
		}
	case PresetNone:
		return &persistence.BrowserProfile{
			Preset: PresetNone,
			AntiDetection: &persistence.AntiDetectionSettings{
				AdBlockingMode: "none",
			},
		}
	default:
		return nil
	}
}
