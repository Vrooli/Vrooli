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

// generateEntitlementsPlist generates macOS entitlements.plist content.
func generateEntitlementsPlist(config *types.MacOSSigningConfig, capabilities []string) ([]byte, error) {
	// Start with default Electron entitlements
	entitlements := make(map[EntitlementKey]bool)
	for _, e := range DefaultElectronEntitlements {
		entitlements[e] = true
	}

	// Add entitlements based on requested capabilities
	for _, cap := range capabilities {
		switch strings.ToLower(cap) {
		case "network", "network-client":
			entitlements[EntitlementNetworkClient] = true
		case "network-server":
			entitlements[EntitlementNetworkServer] = true
		case "audio", "microphone":
			entitlements[EntitlementDeviceAudio] = true
		case "camera":
			entitlements[EntitlementDeviceCamera] = true
		case "bluetooth":
			entitlements[EntitlementDeviceBluetooth] = true
		case "usb":
			entitlements[EntitlementDeviceUSB] = true
		case "location":
			entitlements[EntitlementPersonalInformationLocation] = true
		case "files", "filesystem":
			entitlements[EntitlementFilesUserSelected] = true
			entitlements[EntitlementFilesDownloads] = true
		case "debugger":
			entitlements[EntitlementDebugger] = true
		case "inherit":
			entitlements[EntitlementInheritSecurityScope] = true
		}
	}

	// If hardened runtime is enabled, we need the default entitlements
	// (they're already added above, but this makes the intent clear)
	if config.HardenedRuntime {
		entitlements[EntitlementAllowJIT] = true
		entitlements[EntitlementAllowUnsignedExecutableMemory] = true
	}

	// Convert to sorted slice for deterministic output
	keys := make([]string, 0, len(entitlements))
	for k := range entitlements {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	// Generate the plist
	tmpl, err := template.New("entitlements").Parse(entitlementsPlistTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Keys": keys,
	})
	if err != nil {
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
