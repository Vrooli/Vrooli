package generation

import (
	"bytes"
	"sort"
	"strings"
	"text/template"

	"scenario-to-desktop-api/signing/types"
)

// EntitlementKey represents a macOS entitlement key.
type EntitlementKey string

// Standard macOS entitlements for Electron apps.
const (
	// Security entitlements
	EntitlementAllowJIT                        EntitlementKey = "com.apple.security.cs.allow-jit"
	EntitlementAllowUnsignedExecutableMemory   EntitlementKey = "com.apple.security.cs.allow-unsigned-executable-memory"
	EntitlementDisableLibraryValidation        EntitlementKey = "com.apple.security.cs.disable-library-validation"
	EntitlementDisableExecutablePageProtection EntitlementKey = "com.apple.security.cs.disable-executable-page-protection"
	EntitlementAllowDYLDEnvironmentVariables   EntitlementKey = "com.apple.security.cs.allow-dyld-environment-variables"
	EntitlementDebugger                        EntitlementKey = "com.apple.security.cs.debugger"

	// App functionality entitlements
	EntitlementAppleEvents                 EntitlementKey = "com.apple.security.automation.apple-events"
	EntitlementNetworkClient               EntitlementKey = "com.apple.security.network.client"
	EntitlementNetworkServer               EntitlementKey = "com.apple.security.network.server"
	EntitlementDeviceAudio                 EntitlementKey = "com.apple.security.device.audio-input"
	EntitlementDeviceCamera                EntitlementKey = "com.apple.security.device.camera"
	EntitlementDeviceBluetooth             EntitlementKey = "com.apple.security.device.bluetooth"
	EntitlementDeviceUSB                   EntitlementKey = "com.apple.security.device.usb"
	EntitlementPersonalInformationLocation EntitlementKey = "com.apple.security.personal-information.location"
	EntitlementFilesUserSelected           EntitlementKey = "com.apple.security.files.user-selected.read-write"
	EntitlementFilesDownloads              EntitlementKey = "com.apple.security.files.downloads.read-write"

	// Hardened runtime specific
	EntitlementInheritSecurityScope EntitlementKey = "com.apple.security.inherit"
)

// DefaultElectronEntitlements are the entitlements typically needed for Electron apps.
// These enable JIT compilation and unsigned memory access required by V8/Chromium.
var DefaultElectronEntitlements = []EntitlementKey{
	EntitlementAllowJIT,
	EntitlementAllowUnsignedExecutableMemory,
	EntitlementDisableLibraryValidation,
	EntitlementAppleEvents,
}

// entitlementsPlistTemplate is the XML template for entitlements.plist.
const entitlementsPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
{{- range $key := .Keys }}
    <key>{{ $key }}</key>
    <true/>
{{- end }}
</dict>
</plist>
`

// capabilityEntitlements maps a capability name to the entitlement keys it enables.
var capabilityEntitlements = map[string][]EntitlementKey{
	"network":        {EntitlementNetworkClient},
	"network-client": {EntitlementNetworkClient},
	"network-server": {EntitlementNetworkServer},
	"audio":          {EntitlementDeviceAudio},
	"microphone":     {EntitlementDeviceAudio},
	"camera":         {EntitlementDeviceCamera},
	"bluetooth":      {EntitlementDeviceBluetooth},
	"usb":            {EntitlementDeviceUSB},
	"location":       {EntitlementPersonalInformationLocation},
	"files":          {EntitlementFilesUserSelected, EntitlementFilesDownloads},
	"filesystem":     {EntitlementFilesUserSelected, EntitlementFilesDownloads},
	"debugger":       {EntitlementDebugger},
	"inherit":        {EntitlementInheritSecurityScope},
}

// generateEntitlementsPlist generates macOS entitlements.plist content.
func generateEntitlementsPlist(config *types.MacOSSigningConfig, capabilities []string) ([]byte, error) {
	entitlements := collectEntitlements(config, capabilities)

	keys := make([]string, 0, len(entitlements))
	for k := range entitlements {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	return renderEntitlementsPlist(keys)
}

// collectEntitlements builds the full set of entitlements from defaults, capabilities, and config.
func collectEntitlements(config *types.MacOSSigningConfig, capabilities []string) map[EntitlementKey]bool {
	entitlements := make(map[EntitlementKey]bool, len(DefaultElectronEntitlements))
	for _, e := range DefaultElectronEntitlements {
		entitlements[e] = true
	}

	for _, cap := range capabilities {
		for _, key := range capabilityEntitlements[strings.ToLower(cap)] {
			entitlements[key] = true
		}
	}

	if config.HardenedRuntime {
		entitlements[EntitlementAllowJIT] = true
		entitlements[EntitlementAllowUnsignedExecutableMemory] = true
	}

	return entitlements
}

// renderEntitlementsPlist renders the sorted keys into a plist XML document.
func renderEntitlementsPlist(keys []string) ([]byte, error) {
	tmpl, err := template.New("entitlements").Parse(entitlementsPlistTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{"Keys": keys}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ParseCapabilities converts a comma-separated capability string to a slice.
func ParseCapabilities(caps string) []string {
	if caps == "" {
		return nil
	}

	parts := strings.Split(caps, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
