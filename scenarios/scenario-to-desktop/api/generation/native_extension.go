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
	ActivationShortcut string   `json:"activation_shortcut,omitempty"`
	Version            uint32   `json:"version"`
	Module             string   `json:"module"`
	Permissions        []string `json:"permissions"`
	Platforms          []string `json:"platforms"`
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
		return fmt.Errorf("unsupported native extension contract, module or framework")
	}
	required := map[string]bool{"window.presentation": true}
	if e.Version >= 2 {
		required["global-shortcut"] = true
	}
	if e.Version == 3 {
		required["desktop.context"] = true
	}
	if len(e.Permissions) != len(required) {
		return fmt.Errorf("native extension permissions do not match contract")
	}
	for _, permission := range e.Permissions {
		if !required[permission] {
			return fmt.Errorf("invalid native extension permission")
		}
		delete(required, permission)
	}
	if e.Version == 1 && e.ActivationShortcut != "" {
		return fmt.Errorf("shortcut requires native extension version 2")
	}
	if e.Version >= 2 {
		if !regexp.MustCompile(`^(CommandOrControl|Control|Alt|Super)(\+(Shift|Alt))?\+(Space|[A-Z0-9])$`).MatchString(e.ActivationShortcut) {
			return fmt.Errorf("version 2 requires a supported activation shortcut")
		}
		seen := map[string]bool{}
		for _, part := range strings.Split(e.ActivationShortcut, "+") {
			if seen[part] {
				return fmt.Errorf("duplicate shortcut modifier")
			}
			seen[part] = true
		}
	}
	platforms := map[string]bool{}
	for _, target := range e.Platforms {
		if (target != "linux" && target != "mac" && target != "win") || platforms[target] {
			return fmt.Errorf("invalid or duplicate native extension platform %q", target)
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
			return fmt.Errorf("native extension does not support build target %q", target)
		}
	}
	return nil
}
