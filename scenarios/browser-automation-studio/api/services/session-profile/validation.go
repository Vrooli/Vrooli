// Package sessionprofile provides session profile management for browser automation.
package sessionprofile

import (
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// ValidateBrowserProfile checks that browser profile settings are within valid ranges.
// Exported for use by the execution service during workflow execution.
func ValidateBrowserProfile(bp *persistence.BrowserProfile) error {
	if bp == nil {
		return nil
	}

	// Validate preset
	validPresets := map[string]bool{"": true, "stealth": true, "balanced": true, "fast": true, "none": true}
	if !validPresets[bp.Preset] {
		return fmt.Errorf("invalid preset: %s (must be stealth, balanced, fast, or none)", bp.Preset)
	}

	// Validate fingerprint settings
	if fp := bp.Fingerprint; fp != nil {
		if fp.ViewportWidth < 0 || fp.ViewportWidth > 7680 {
			return fmt.Errorf("viewport_width must be between 0 and 7680")
		}
		if fp.ViewportHeight < 0 || fp.ViewportHeight > 4320 {
			return fmt.Errorf("viewport_height must be between 0 and 4320")
		}
		if fp.DeviceScaleFactor < 0 || fp.DeviceScaleFactor > 5 {
			return fmt.Errorf("device_scale_factor must be between 0 and 5")
		}
		if fp.HardwareConcurrency < 0 || fp.HardwareConcurrency > 128 {
			return fmt.Errorf("hardware_concurrency must be between 0 and 128")
		}
		if fp.DeviceMemory < 0 || fp.DeviceMemory > 512 {
			return fmt.Errorf("device_memory must be between 0 and 512")
		}
		if fp.Latitude < -90 || fp.Latitude > 90 {
			return fmt.Errorf("latitude must be between -90 and 90")
		}
		if fp.Longitude < -180 || fp.Longitude > 180 {
			return fmt.Errorf("longitude must be between -180 and 180")
		}
		validColorSchemes := map[string]bool{"": true, "light": true, "dark": true, "no-preference": true}
		if !validColorSchemes[fp.ColorScheme] {
			return fmt.Errorf("color_scheme must be light, dark, or no-preference")
		}
	}

	// Validate behavior settings
	if bh := bp.Behavior; bh != nil {
		// Typing delay validation
		if bh.TypingDelayMin < 0 || bh.TypingDelayMin > 5000 {
			return fmt.Errorf("typing_delay_min must be between 0 and 5000")
		}
		if bh.TypingDelayMax < 0 || bh.TypingDelayMax > 5000 {
			return fmt.Errorf("typing_delay_max must be between 0 and 5000")
		}
		if bh.TypingDelayMin > bh.TypingDelayMax && bh.TypingDelayMax > 0 {
			return fmt.Errorf("typing_delay_min cannot exceed typing_delay_max")
		}

		// Typing start delay validation
		if bh.TypingStartDelayMin < 0 || bh.TypingStartDelayMin > 5000 {
			return fmt.Errorf("typing_start_delay_min must be between 0 and 5000")
		}
		if bh.TypingStartDelayMax < 0 || bh.TypingStartDelayMax > 5000 {
			return fmt.Errorf("typing_start_delay_max must be between 0 and 5000")
		}
		if bh.TypingStartDelayMin > bh.TypingStartDelayMax && bh.TypingStartDelayMax > 0 {
			return fmt.Errorf("typing_start_delay_min cannot exceed typing_start_delay_max")
		}

		// Typing paste threshold validation (-1 = always paste, 0 = always type, >0 = paste if text > threshold)
		if bh.TypingPasteThreshold < -1 || bh.TypingPasteThreshold > 10000 {
			return fmt.Errorf("typing_paste_threshold must be between -1 and 10000")
		}

		validMouseStyles := map[string]bool{"": true, "linear": true, "bezier": true, "natural": true}
		if !validMouseStyles[bh.MouseMovementStyle] {
			return fmt.Errorf("mouse_movement_style must be linear, bezier, or natural")
		}
		if bh.MouseJitterAmount < 0 || bh.MouseJitterAmount > 100 {
			return fmt.Errorf("mouse_jitter_amount must be between 0 and 100")
		}
		if bh.ClickDelayMin < 0 || bh.ClickDelayMin > 5000 {
			return fmt.Errorf("click_delay_min must be between 0 and 5000")
		}
		if bh.ClickDelayMax < 0 || bh.ClickDelayMax > 5000 {
			return fmt.Errorf("click_delay_max must be between 0 and 5000")
		}
		if bh.ClickDelayMin > bh.ClickDelayMax && bh.ClickDelayMax > 0 {
			return fmt.Errorf("click_delay_min cannot exceed click_delay_max")
		}
		validScrollStyles := map[string]bool{"": true, "smooth": true, "stepped": true}
		if !validScrollStyles[bh.ScrollStyle] {
			return fmt.Errorf("scroll_style must be smooth or stepped")
		}
		if bh.MicroPauseFrequency < 0 || bh.MicroPauseFrequency > 1 {
			return fmt.Errorf("micro_pause_frequency must be between 0 and 1")
		}
		if bh.MicroPauseMinMs < 0 || bh.MicroPauseMinMs > 10000 {
			return fmt.Errorf("micro_pause_min_ms must be between 0 and 10000")
		}
		if bh.MicroPauseMaxMs < 0 || bh.MicroPauseMaxMs > 10000 {
			return fmt.Errorf("micro_pause_max_ms must be between 0 and 10000")
		}
	}

	// Validate anti-detection settings
	if ad := bp.AntiDetection; ad != nil {
		validAdBlockingModes := map[string]bool{"": true, "none": true, "ads_only": true, "ads_and_tracking": true}
		if !validAdBlockingModes[ad.AdBlockingMode] {
			return fmt.Errorf("ad_blocking_mode must be none, ads_only, or ads_and_tracking")
		}
	}

	// Validate proxy settings
	if proxy := bp.Proxy; proxy != nil {
		if proxy.Enabled && proxy.Server == "" {
			return fmt.Errorf("proxy server is required when proxy is enabled")
		}
		if proxy.Server != "" {
			if !strings.HasPrefix(proxy.Server, "http://") &&
				!strings.HasPrefix(proxy.Server, "https://") &&
				!strings.HasPrefix(proxy.Server, "socks5://") {
				return fmt.Errorf("proxy server must start with http://, https://, or socks5://")
			}
		}
		// If password is set, username should also be set
		if proxy.Password != "" && proxy.Username == "" {
			return fmt.Errorf("proxy username is required when password is set")
		}
	}

	// Validate extra headers
	if err := validateExtraHeaders(bp.ExtraHeaders); err != nil {
		return err
	}

	return nil
}

// ValidateHistorySettings validates history settings.
func ValidateHistorySettings(settings *persistence.HistorySettings) error {
	if settings == nil {
		return nil
	}
	if settings.MaxEntries < 0 || settings.MaxEntries > 10000 {
		return fmt.Errorf("max_entries must be between 0 and 10000")
	}
	if settings.RetentionDays < 0 || settings.RetentionDays > 3650 {
		return fmt.Errorf("retention_days must be between 0 and 3650")
	}
	return nil
}

// blockedHeaders contains HTTP headers that cannot be set via extra_headers
// because they may break routing or conflict with other browser features.
var blockedHeaders = map[string]bool{
	"host":           true, // Can break routing
	"content-length": true, // Managed by browser
	"cookie":         true, // Use storage_state instead
}

// validateExtraHeaders checks that no blocked headers are included.
func validateExtraHeaders(headers map[string]string) error {
	for k := range headers {
		if blockedHeaders[strings.ToLower(k)] {
			return fmt.Errorf("header %q cannot be set via extra_headers", k)
		}
	}
	return nil
}
