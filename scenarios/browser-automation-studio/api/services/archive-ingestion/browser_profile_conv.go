package archiveingestion

import (
	"google.golang.org/protobuf/proto"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

// BrowserProfileFromProto converts a proto BrowserProfile to the domain type.
// Returns nil if the proto is nil.
func BrowserProfileFromProto(p *basbase.BrowserProfile) *BrowserProfile {
	if p == nil {
		return nil
	}

	bp := &BrowserProfile{
		Preset: p.GetPreset(),
	}

	if p.Fingerprint != nil {
		bp.Fingerprint = fingerprintFromProto(p.Fingerprint)
	}
	if p.Behavior != nil {
		bp.Behavior = behaviorFromProto(p.Behavior)
	}
	if p.AntiDetection != nil {
		bp.AntiDetection = antiDetectionFromProto(p.AntiDetection)
	}

	return bp
}

func fingerprintFromProto(p *basbase.FingerprintSettings) *FingerprintSettings {
	if p == nil {
		return nil
	}
	return &FingerprintSettings{
		ViewportWidth:       int(p.GetViewportWidth()),
		ViewportHeight:      int(p.GetViewportHeight()),
		DeviceScaleFactor:   p.GetDeviceScaleFactor(),
		HardwareConcurrency: int(p.GetHardwareConcurrency()),
		DeviceMemory:        int(p.GetDeviceMemory()),
		UserAgent:           p.GetUserAgent(),
		UserAgentPreset:     p.GetUserAgentPreset(),
		Locale:              p.GetLocale(),
		TimezoneID:          p.GetTimezoneId(),
		GeolocationEnabled:  p.GetGeolocationEnabled(),
		Latitude:            p.GetLatitude(),
		Longitude:           p.GetLongitude(),
		Accuracy:            p.GetAccuracy(),
		ColorScheme:         p.GetColorScheme(),
	}
}

func behaviorFromProto(p *basbase.BehaviorSettings) *BehaviorSettings {
	if p == nil {
		return nil
	}
	return &BehaviorSettings{
		TypingDelayMin:      int(p.GetTypingDelayMin()),
		TypingDelayMax:      int(p.GetTypingDelayMax()),
		MouseMovementStyle:  p.GetMouseMovementStyle(),
		MouseJitterAmount:   p.GetMouseJitterAmount(),
		ClickDelayMin:       int(p.GetClickDelayMin()),
		ClickDelayMax:       int(p.GetClickDelayMax()),
		ScrollStyle:         p.GetScrollStyle(),
		ScrollSpeedMin:      int(p.GetScrollSpeedMin()),
		ScrollSpeedMax:      int(p.GetScrollSpeedMax()),
		MicroPauseEnabled:   p.GetMicroPauseEnabled(),
		MicroPauseMinMs:     int(p.GetMicroPauseMinMs()),
		MicroPauseMaxMs:     int(p.GetMicroPauseMaxMs()),
		MicroPauseFrequency: p.GetMicroPauseFrequency(),
	}
}

func antiDetectionFromProto(p *basbase.AntiDetectionSettings) *AntiDetectionSettings {
	if p == nil {
		return nil
	}
	return &AntiDetectionSettings{
		DisableAutomationControlled: p.GetDisableAutomationControlled(),
		DisableWebRTC:               p.GetDisableWebrtc(),
		PatchNavigatorWebdriver:     p.GetPatchNavigatorWebdriver(),
		PatchNavigatorPlugins:       p.GetPatchNavigatorPlugins(),
		PatchNavigatorLanguages:     p.GetPatchNavigatorLanguages(),
		PatchWebGL:                  p.GetPatchWebgl(),
		PatchCanvas:                 p.GetPatchCanvas(),
		HeadlessDetectionBypass:     p.GetHeadlessDetectionBypass(),
	}
}

// BrowserProfileToProto converts a domain BrowserProfile to the proto type.
// Returns nil if the domain type is nil.
func BrowserProfileToProto(bp *BrowserProfile) *basbase.BrowserProfile {
	if bp == nil {
		return nil
	}

	p := &basbase.BrowserProfile{}

	if bp.Preset != "" {
		p.Preset = proto.String(bp.Preset)
	}
	if bp.Fingerprint != nil {
		p.Fingerprint = fingerprintToProto(bp.Fingerprint)
	}
	if bp.Behavior != nil {
		p.Behavior = behaviorToProto(bp.Behavior)
	}
	if bp.AntiDetection != nil {
		p.AntiDetection = antiDetectionToProto(bp.AntiDetection)
	}

	return p
}

func fingerprintToProto(fp *FingerprintSettings) *basbase.FingerprintSettings {
	if fp == nil {
		return nil
	}
	p := &basbase.FingerprintSettings{}

	if fp.ViewportWidth != 0 {
		p.ViewportWidth = proto.Int32(int32(fp.ViewportWidth))
	}
	if fp.ViewportHeight != 0 {
		p.ViewportHeight = proto.Int32(int32(fp.ViewportHeight))
	}
	if fp.DeviceScaleFactor != 0 {
		p.DeviceScaleFactor = proto.Float64(fp.DeviceScaleFactor)
	}
	if fp.HardwareConcurrency != 0 {
		p.HardwareConcurrency = proto.Int32(int32(fp.HardwareConcurrency))
	}
	if fp.DeviceMemory != 0 {
		p.DeviceMemory = proto.Int32(int32(fp.DeviceMemory))
	}
	if fp.UserAgent != "" {
		p.UserAgent = proto.String(fp.UserAgent)
	}
	if fp.UserAgentPreset != "" {
		p.UserAgentPreset = proto.String(fp.UserAgentPreset)
	}
	if fp.Locale != "" {
		p.Locale = proto.String(fp.Locale)
	}
	if fp.TimezoneID != "" {
		p.TimezoneId = proto.String(fp.TimezoneID)
	}
	if fp.GeolocationEnabled {
		p.GeolocationEnabled = proto.Bool(fp.GeolocationEnabled)
	}
	if fp.Latitude != 0 {
		p.Latitude = proto.Float64(fp.Latitude)
	}
	if fp.Longitude != 0 {
		p.Longitude = proto.Float64(fp.Longitude)
	}
	if fp.Accuracy != 0 {
		p.Accuracy = proto.Float64(fp.Accuracy)
	}
	if fp.ColorScheme != "" {
		p.ColorScheme = proto.String(fp.ColorScheme)
	}

	return p
}

func behaviorToProto(bh *BehaviorSettings) *basbase.BehaviorSettings {
	if bh == nil {
		return nil
	}
	p := &basbase.BehaviorSettings{}

	if bh.TypingDelayMin != 0 {
		p.TypingDelayMin = proto.Int32(int32(bh.TypingDelayMin))
	}
	if bh.TypingDelayMax != 0 {
		p.TypingDelayMax = proto.Int32(int32(bh.TypingDelayMax))
	}
	if bh.MouseMovementStyle != "" {
		p.MouseMovementStyle = proto.String(bh.MouseMovementStyle)
	}
	if bh.MouseJitterAmount != 0 {
		p.MouseJitterAmount = proto.Float64(bh.MouseJitterAmount)
	}
	if bh.ClickDelayMin != 0 {
		p.ClickDelayMin = proto.Int32(int32(bh.ClickDelayMin))
	}
	if bh.ClickDelayMax != 0 {
		p.ClickDelayMax = proto.Int32(int32(bh.ClickDelayMax))
	}
	if bh.ScrollStyle != "" {
		p.ScrollStyle = proto.String(bh.ScrollStyle)
	}
	if bh.ScrollSpeedMin != 0 {
		p.ScrollSpeedMin = proto.Int32(int32(bh.ScrollSpeedMin))
	}
	if bh.ScrollSpeedMax != 0 {
		p.ScrollSpeedMax = proto.Int32(int32(bh.ScrollSpeedMax))
	}
	if bh.MicroPauseEnabled {
		p.MicroPauseEnabled = proto.Bool(bh.MicroPauseEnabled)
	}
	if bh.MicroPauseMinMs != 0 {
		p.MicroPauseMinMs = proto.Int32(int32(bh.MicroPauseMinMs))
	}
	if bh.MicroPauseMaxMs != 0 {
		p.MicroPauseMaxMs = proto.Int32(int32(bh.MicroPauseMaxMs))
	}
	if bh.MicroPauseFrequency != 0 {
		p.MicroPauseFrequency = proto.Float64(bh.MicroPauseFrequency)
	}

	return p
}

func antiDetectionToProto(ad *AntiDetectionSettings) *basbase.AntiDetectionSettings {
	if ad == nil {
		return nil
	}
	p := &basbase.AntiDetectionSettings{}

	if ad.DisableAutomationControlled {
		p.DisableAutomationControlled = proto.Bool(ad.DisableAutomationControlled)
	}
	if ad.DisableWebRTC {
		p.DisableWebrtc = proto.Bool(ad.DisableWebRTC)
	}
	if ad.PatchNavigatorWebdriver {
		p.PatchNavigatorWebdriver = proto.Bool(ad.PatchNavigatorWebdriver)
	}
	if ad.PatchNavigatorPlugins {
		p.PatchNavigatorPlugins = proto.Bool(ad.PatchNavigatorPlugins)
	}
	if ad.PatchNavigatorLanguages {
		p.PatchNavigatorLanguages = proto.Bool(ad.PatchNavigatorLanguages)
	}
	if ad.PatchWebGL {
		p.PatchWebgl = proto.Bool(ad.PatchWebGL)
	}
	if ad.PatchCanvas {
		p.PatchCanvas = proto.Bool(ad.PatchCanvas)
	}
	if ad.HeadlessDetectionBypass {
		p.HeadlessDetectionBypass = proto.Bool(ad.HeadlessDetectionBypass)
	}

	return p
}

// MergeBrowserProfiles merges base and override profiles, with override values taking precedence.
// If both are nil, returns nil. If only one is nil, returns the other.
// For non-nil nested structs, the override values replace base values when set (non-zero).
func MergeBrowserProfiles(base, override *BrowserProfile) *BrowserProfile {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	// Start with a copy of base
	result := &BrowserProfile{
		Preset: base.Preset,
	}

	// Override preset if set
	if override.Preset != "" {
		result.Preset = override.Preset
	}

	// Merge fingerprint settings
	result.Fingerprint = mergeFingerprintSettings(base.Fingerprint, override.Fingerprint)

	// Merge behavior settings
	result.Behavior = mergeBehaviorSettings(base.Behavior, override.Behavior)

	// Merge anti-detection settings
	result.AntiDetection = mergeAntiDetectionSettings(base.AntiDetection, override.AntiDetection)

	return result
}

func mergeFingerprintSettings(base, override *FingerprintSettings) *FingerprintSettings {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := *base // Copy base

	if override.ViewportWidth != 0 {
		result.ViewportWidth = override.ViewportWidth
	}
	if override.ViewportHeight != 0 {
		result.ViewportHeight = override.ViewportHeight
	}
	if override.DeviceScaleFactor != 0 {
		result.DeviceScaleFactor = override.DeviceScaleFactor
	}
	if override.HardwareConcurrency != 0 {
		result.HardwareConcurrency = override.HardwareConcurrency
	}
	if override.DeviceMemory != 0 {
		result.DeviceMemory = override.DeviceMemory
	}
	if override.UserAgent != "" {
		result.UserAgent = override.UserAgent
	}
	if override.UserAgentPreset != "" {
		result.UserAgentPreset = override.UserAgentPreset
	}
	if override.Locale != "" {
		result.Locale = override.Locale
	}
	if override.TimezoneID != "" {
		result.TimezoneID = override.TimezoneID
	}
	if override.GeolocationEnabled {
		result.GeolocationEnabled = override.GeolocationEnabled
	}
	if override.Latitude != 0 {
		result.Latitude = override.Latitude
	}
	if override.Longitude != 0 {
		result.Longitude = override.Longitude
	}
	if override.Accuracy != 0 {
		result.Accuracy = override.Accuracy
	}
	if override.ColorScheme != "" {
		result.ColorScheme = override.ColorScheme
	}

	return &result
}

func mergeBehaviorSettings(base, override *BehaviorSettings) *BehaviorSettings {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := *base // Copy base

	if override.TypingDelayMin != 0 {
		result.TypingDelayMin = override.TypingDelayMin
	}
	if override.TypingDelayMax != 0 {
		result.TypingDelayMax = override.TypingDelayMax
	}
	if override.MouseMovementStyle != "" {
		result.MouseMovementStyle = override.MouseMovementStyle
	}
	if override.MouseJitterAmount != 0 {
		result.MouseJitterAmount = override.MouseJitterAmount
	}
	if override.ClickDelayMin != 0 {
		result.ClickDelayMin = override.ClickDelayMin
	}
	if override.ClickDelayMax != 0 {
		result.ClickDelayMax = override.ClickDelayMax
	}
	if override.ScrollStyle != "" {
		result.ScrollStyle = override.ScrollStyle
	}
	if override.ScrollSpeedMin != 0 {
		result.ScrollSpeedMin = override.ScrollSpeedMin
	}
	if override.ScrollSpeedMax != 0 {
		result.ScrollSpeedMax = override.ScrollSpeedMax
	}
	if override.MicroPauseEnabled {
		result.MicroPauseEnabled = override.MicroPauseEnabled
	}
	if override.MicroPauseMinMs != 0 {
		result.MicroPauseMinMs = override.MicroPauseMinMs
	}
	if override.MicroPauseMaxMs != 0 {
		result.MicroPauseMaxMs = override.MicroPauseMaxMs
	}
	if override.MicroPauseFrequency != 0 {
		result.MicroPauseFrequency = override.MicroPauseFrequency
	}

	return &result
}

func mergeAntiDetectionSettings(base, override *AntiDetectionSettings) *AntiDetectionSettings {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := *base // Copy base

	// For booleans, override if true (can't distinguish "not set" from "false" with plain bools)
	if override.DisableAutomationControlled {
		result.DisableAutomationControlled = override.DisableAutomationControlled
	}
	if override.DisableWebRTC {
		result.DisableWebRTC = override.DisableWebRTC
	}
	if override.PatchNavigatorWebdriver {
		result.PatchNavigatorWebdriver = override.PatchNavigatorWebdriver
	}
	if override.PatchNavigatorPlugins {
		result.PatchNavigatorPlugins = override.PatchNavigatorPlugins
	}
	if override.PatchNavigatorLanguages {
		result.PatchNavigatorLanguages = override.PatchNavigatorLanguages
	}
	if override.PatchWebGL {
		result.PatchWebGL = override.PatchWebGL
	}
	if override.PatchCanvas {
		result.PatchCanvas = override.PatchCanvas
	}
	if override.HeadlessDetectionBypass {
		result.HeadlessDetectionBypass = override.HeadlessDetectionBypass
	}

	return &result
}
