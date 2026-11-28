package generation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"
)

// NativeExtension selects reviewed code owned by the generator. It accepts no paths.
type NativeExtension struct {
	ActivationShortcut string           `json:"activation_shortcut,omitempty"`
	Version            uint32           `json:"version"`
	Module             string           `json:"module"`
	Permissions        []string         `json:"permissions"`
	Platforms          []string         `json:"platforms"`
	HelperProviders    []HelperProvider `json:"helper_providers,omitempty"`
}

// HelperProvider is a closed owner/capability reference. It selects a
// governed provider already known to the ramp; it cannot name a path,
// executable, import, or arbitrary IPC method.
type HelperProvider struct {
	Owner      string `json:"owner"`
	Capability string `json:"capability"`
}

// NativeExtensionCompatibilityError gives callers a stable reason for refusing
// an extension before generation starts. The message remains suitable for the
// existing build log, while Code/Field/Target support typed client handling.
type NativeExtensionCompatibilityError struct {
	Code    string
	Message string
	Field   string
	Target  string
}

func (e *NativeExtensionCompatibilityError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func nativeExtensionError(code, message string) *NativeExtensionCompatibilityError {
	return &NativeExtensionCompatibilityError{Code: code, Message: message}
}

func (e *NativeExtension) UnmarshalJSON(data []byte) error {
	type plain NativeExtension
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*e = NativeExtension(value)
	return nil
}

func (c *DesktopConfig) ValidateNativeExtension() error {
	e := c.NativeExtension
	if e == nil {
		return nil
	}
	if c.Framework != "electron" || (e.Version != 1 && e.Version != 2 && e.Version != 3) || e.Module != "presentation" {
		return nativeExtensionError("NATIVE_EXTENSION_UNSUPPORTED_CONTRACT", "unsupported native extension contract, module or framework")
	}
	required := map[string]bool{"window.presentation": true}
	if e.Version >= 2 {
		required["global-shortcut"] = true
	}
	if e.Version == 3 {
		required["desktop.context"] = true
	}
	if len(e.Permissions) != len(required) {
		err := nativeExtensionError("NATIVE_EXTENSION_PERMISSION_MISMATCH", "native extension permissions do not match contract")
		err.Field = "permissions"
		return err
	}
	for _, permission := range e.Permissions {
		if !required[permission] {
			err := nativeExtensionError("NATIVE_EXTENSION_PERMISSION_MISMATCH", "invalid native extension permission")
			err.Field = "permissions"
			return err
		}
		delete(required, permission)
	}
	providers := map[string]bool{}
	for _, provider := range e.HelperProviders {
		key := strings.TrimSpace(provider.Owner) + "/" + strings.TrimSpace(provider.Capability)
		if provider.Owner != "device-control" || provider.Capability != "desktop.session" || providers[key] {
			err := nativeExtensionError("NATIVE_EXTENSION_HELPER_PROVIDER_INVALID", "helper provider is not in the governed registry")
			err.Field = "helper_providers"
			return err
		}
		providers[key] = true
	}
	if e.Version == 1 && e.ActivationShortcut != "" {
		err := nativeExtensionError("NATIVE_EXTENSION_SHORTCUT_INVALID", "shortcut requires native extension version 2")
		err.Field = "activation_shortcut"
		return err
	}
	if e.Version >= 2 {
		if !regexp.MustCompile(`^(CommandOrControl|Control|Alt|Super)(\+(Shift|Alt))?\+(Space|[A-Z0-9])$`).MatchString(e.ActivationShortcut) {
			err := nativeExtensionError("NATIVE_EXTENSION_SHORTCUT_INVALID", "version 2 requires a supported activation shortcut")
			err.Field = "activation_shortcut"
			return err
		}
		seen := map[string]bool{}
		for _, part := range strings.Split(e.ActivationShortcut, "+") {
			if seen[part] {
				err := nativeExtensionError("NATIVE_EXTENSION_SHORTCUT_INVALID", "duplicate shortcut modifier")
				err.Field = "activation_shortcut"
				return err
			}
			seen[part] = true
		}
	}
	platforms := map[string]bool{}
	for _, target := range e.Platforms {
		if (target != "linux" && target != "mac" && target != "win") || platforms[target] {
			err := nativeExtensionError("NATIVE_EXTENSION_PLATFORM_INVALID", fmt.Sprintf("invalid or duplicate native extension platform %q", target))
			err.Field = "platforms"
			return err
		}
		platforms[target] = true
	}
	targets := c.Platforms
	if len(targets) == 0 {
		target := runtime.GOOS
		if target == "darwin" {
			target = "mac"
		}
		if target == "windows" {
			target = "win"
		}
		targets = []string{target}
	}
	for _, target := range targets {
		osName := target
		if parts := strings.Split(target, "-"); len(parts) == 2 && (parts[1] == "amd64" || parts[1] == "arm64") {
			osName = map[string]string{"linux": "linux", "darwin": "mac", "windows": "win"}[parts[0]]
		}
		if !platforms[osName] {
			err := nativeExtensionError("NATIVE_EXTENSION_TARGET_UNSUPPORTED", fmt.Sprintf("native extension does not support build target %q", target))
			err.Field = "platforms"
			err.Target = target
			return err
		}
	}
	return nil
}
