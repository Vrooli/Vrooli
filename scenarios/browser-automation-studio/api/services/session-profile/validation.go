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

	if err := validatePreset(bp.Preset); err != nil {
		return err
	}
	if err := validateFingerprintSettings(bp.Fingerprint); err != nil {
		return err
	}
	if err := validateBehaviorSettings(bp.Behavior); err != nil {
		return err
	}
	if err := validateAntiDetectionSettings(bp.AntiDetection); err != nil {
		return err
	}
	if err := validateProxySettings(bp.Proxy); err != nil {
		return err
	}
	return validateExtraHeaders(bp.ExtraHeaders)
}

func validatePreset(preset string) error {
	validPresets := map[string]bool{"": true, "stealth": true, "balanced": true, "fast": true, "none": true}
	if !validPresets[preset] {
		return fmt.Errorf("invalid preset: %s (must be stealth, balanced, fast, or none)", preset)
	}
	return nil
}

func validateFingerprintSettings(fp *persistence.FingerprintSettings) error {
	if fp == nil {
		return nil
	}
	if err := validateViewportSettings(fp); err != nil {
		return err
	}
	if err := validateDeviceSettings(fp); err != nil {
		return err
	}
	if err := validateGeolocationSettings(fp); err != nil {
		return err
	}
	return validateColorScheme(fp.ColorScheme)
}

func validateViewportSettings(fp *persistence.FingerprintSettings) error {
	if fp.ViewportWidth < 0 || fp.ViewportWidth > 7680 {
		return fmt.Errorf("viewport_width must be between 0 and 7680")
	}
	if fp.ViewportHeight < 0 || fp.ViewportHeight > 4320 {
		return fmt.Errorf("viewport_height must be between 0 and 4320")
	}
	return nil
}

func validateDeviceSettings(fp *persistence.FingerprintSettings) error {
	if fp.DeviceScaleFactor < 0 || fp.DeviceScaleFactor > 5 {
		return fmt.Errorf("device_scale_factor must be between 0 and 5")
	}
	if fp.HardwareConcurrency < 0 || fp.HardwareConcurrency > 128 {
		return fmt.Errorf("hardware_concurrency must be between 0 and 128")
	}
	if fp.DeviceMemory < 0 || fp.DeviceMemory > 512 {
		return fmt.Errorf("device_memory must be between 0 and 512")
	}
	return nil
}

func validateGeolocationSettings(fp *persistence.FingerprintSettings) error {
	if fp.Latitude < -90 || fp.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if fp.Longitude < -180 || fp.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

func validateColorScheme(colorScheme string) error {
	validColorSchemes := map[string]bool{"": true, "light": true, "dark": true, "no-preference": true}
	if !validColorSchemes[colorScheme] {
		return fmt.Errorf("color_scheme must be light, dark, or no-preference")
	}
	return nil
}

func validateBehaviorSettings(bh *persistence.BehaviorSettings) error {
	if bh == nil {
		return nil
	}
	if err := validateTypingBehavior(bh); err != nil {
		return err
	}
	if err := validateMouseBehavior(bh); err != nil {
		return err
	}
	return validateScrollBehavior(bh)
}

func validateTypingBehavior(bh *persistence.BehaviorSettings) error {
	if err := validateTypingDelays(bh.TypingDelayMin, bh.TypingDelayMax, "typing_delay"); err != nil {
		return err
	}
	if err := validateTypingDelays(bh.TypingStartDelayMin, bh.TypingStartDelayMax, "typing_start_delay"); err != nil {
		return err
	}
	if bh.TypingPasteThreshold < -1 || bh.TypingPasteThreshold > 10000 {
		return fmt.Errorf("typing_paste_threshold must be between -1 and 10000")
	}
	return nil
}

func validateTypingDelays(min, max int, field string) error {
	if min < 0 || min > 5000 {
		return fmt.Errorf("%s_min must be between 0 and 5000", field)
	}
	if max < 0 || max > 5000 {
		return fmt.Errorf("%s_max must be between 0 and 5000", field)
	}
	if min > max && max > 0 {
		return fmt.Errorf("%s_min cannot exceed %s_max", field, field)
	}
	return nil
}

func validateMouseBehavior(bh *persistence.BehaviorSettings) error {
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
	return nil
}

func validateScrollBehavior(bh *persistence.BehaviorSettings) error {
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
	return nil
}

func validateAntiDetectionSettings(ad *persistence.AntiDetectionSettings) error {
	if ad != nil {
		validAdBlockingModes := map[string]bool{"": true, "none": true, "ads_only": true, "ads_and_tracking": true}
		if !validAdBlockingModes[ad.AdBlockingMode] {
			return fmt.Errorf("ad_blocking_mode must be none, ads_only, or ads_and_tracking")
		}
	}
	return nil
}

func validateProxySettings(proxy *persistence.ProxySettings) error {
	if proxy != nil {
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
