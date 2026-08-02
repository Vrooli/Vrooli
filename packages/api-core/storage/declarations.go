package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Platform is the portable platform vocabulary used by manifests.
type Platform string

const (
	PlatformLinux   Platform = "linux"
	PlatformMacOS   Platform = "macos"
	PlatformWindows Platform = "windows"
)

// NotApplicable reports a declared entry that intentionally has no location
// on the requested platform. Callers can distinguish this from resolution
// failure without treating an absent optional entry as an error.
type NotApplicable struct {
	Entry    string
	Platform Platform
}

func (e *NotApplicable) Error() string {
	return fmt.Sprintf("storage entry %q is not applicable on %s", e.Entry, e.Platform)
}

// PortablePath is either one portable path or platform-specific overrides.
// A null platform value means the entry is intentionally absent there.
type PortablePath struct {
	Value string
	ByOS  map[Platform]*string
}

// MarshalJSON preserves the compact manifest shape when a normalized
// declaration is returned by inventory or placement APIs.
func (p PortablePath) MarshalJSON() ([]byte, error) {
	if p.ByOS == nil {
		return json.Marshal(p.Value)
	}
	raw := make(map[string]*string, len(p.ByOS))
	for platform, value := range p.ByOS {
		raw[string(platform)] = value
	}
	return json.Marshal(raw)
}

func (p *PortablePath) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		p.Value = value
		p.ByOS = nil
		return nil
	}
	var raw map[string]*string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("storage path must be a string or platform map: %w", err)
	}
	p.ByOS = make(map[Platform]*string, len(raw))
	for key, item := range raw {
		platform := normalizePlatform(key)
		if platform == "" {
			return fmt.Errorf("unsupported storage path platform %q", key)
		}
		if item != nil && strings.TrimSpace(*item) == "" {
			return fmt.Errorf("storage path for %s is empty", platform)
		}
		p.ByOS[platform] = item
	}
	return nil
}

// ResolvePortablePath resolves a declaration path using platform conventions.
// The seams make the exact same manifest testable for all supported platforms.
func ResolvePortablePath(entryName string, path PortablePath, requested Platform, seams PlatformSeams) (string, error) {
	platform := normalizePlatform(string(requested))
	if platform == "" {
		return "", fmt.Errorf("unsupported storage platform %q", requested)
	}
	value := path.Value
	if path.ByOS != nil {
		selected, ok := path.ByOS[platform]
		if !ok || selected == nil {
			return "", &NotApplicable{Entry: entryName, Platform: platform}
		}
		value = *selected
	}
	return resolvePortableTokens(value, seams)
}

// ResolveOwnerStoragePath resolves one declaration using the owner's native
// manifest location for relative paths and portable tokens for host paths.
// Scenario declarations are relative to the scenario directory. Other owner
// declarations are relative to the directory that contains their manifest.
// This keeps inventory, census, placement, and adoption consumers on one path
// contract instead of each consumer inventing its own base directory.
func ResolveOwnerStoragePath(repoRoot string, owner OwnerManifest, entry StorageEntry, requested Platform, seams PlatformSeams) (string, error) {
	value := strings.TrimSpace(entry.Path.Value)
	if entry.Path.ByOS != nil || filepath.IsAbs(value) || containsPortableToken(value) {
		return ResolvePortablePath(entry.Name, entry.Path, requested, seams)
	}
	if value == "" {
		return "", fmt.Errorf("storage path is empty")
	}
	base := filepath.Dir(owner.ManifestPath)
	if owner.Kind == OwnerScenario {
		base = filepath.Dir(base)
	}
	if !filepath.IsAbs(base) && strings.TrimSpace(repoRoot) != "" {
		base = filepath.Join(repoRoot, base)
	}
	return filepath.Clean(filepath.Join(base, filepath.FromSlash(value))), nil
}

func containsPortableToken(value string) bool {
	return strings.Contains(value, "$USER_HOME") ||
		strings.Contains(value, "$USER_CONFIG_DIR") ||
		strings.Contains(value, "$USER_CACHE_DIR") ||
		strings.Contains(value, "$USER_STATE_DIR") ||
		strings.Contains(value, "$TEMP_DIR") ||
		strings.Contains(value, "$HOME") ||
		strings.Contains(value, "%USERPROFILE%") ||
		value == "~" || strings.HasPrefix(value, "~/")
}

// PlatformSeams are injectable OS directory lookups used by the resolver.
type PlatformSeams struct {
	UserHomeDir   func() (string, error)
	UserConfigDir func() (string, error)
	UserCacheDir  func() (string, error)
	UserStateDir  func() (string, error)
	TempDir       func() string
}

func resolvePortableTokens(value string, seams PlatformSeams) (string, error) {
	if strings.Contains(value, "$XDG_") || strings.Contains(value, "${XDG_") {
		return "", fmt.Errorf("XDG variables are not portable storage tokens; use $USER_CONFIG_DIR, $USER_CACHE_DIR, or $USER_STATE_DIR")
	}
	home := seams.UserHomeDir
	if home == nil {
		home = os.UserHomeDir
	}
	config := seams.UserConfigDir
	if config == nil {
		config = os.UserConfigDir
	}
	cache := seams.UserCacheDir
	if cache == nil {
		cache = os.UserCacheDir
	}
	state := seams.UserStateDir
	temp := seams.TempDir
	if temp == nil {
		temp = os.TempDir
	}
	values := map[string]string{"$TEMP_DIR": temp()}
	var err error
	if values["$TEMP_DIR"] == "" {
		return "", fmt.Errorf("resolve $TEMP_DIR: empty result")
	}
	values["$USER_HOME"], err = home()
	if err != nil {
		return "", fmt.Errorf("resolve $USER_HOME: %w", err)
	}
	values["$USER_CONFIG_DIR"], err = config()
	if err != nil {
		return "", fmt.Errorf("resolve $USER_CONFIG_DIR: %w", err)
	}
	values["$USER_CACHE_DIR"], err = cache()
	if err != nil {
		return "", fmt.Errorf("resolve $USER_CACHE_DIR: %w", err)
	}
	if state != nil {
		values["$USER_STATE_DIR"], err = state()
		if err != nil {
			return "", fmt.Errorf("resolve $USER_STATE_DIR: %w", err)
		}
	} else {
		values["$USER_STATE_DIR"] = filepath.Join(values["$USER_HOME"], ".local", "state")
	}
	value = strings.ReplaceAll(value, "%USERPROFILE%", "$USER_HOME")
	if value == "~" || strings.HasPrefix(value, "~/") {
		value = "$USER_HOME" + strings.TrimPrefix(value, "~")
	}
	value = strings.ReplaceAll(value, "$HOME", "$USER_HOME")
	for token, replacement := range values {
		value = strings.ReplaceAll(value, token, replacement)
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("storage path %q resolves to non-absolute path %q", value, value)
	}
	return filepath.Clean(value), nil
}

func normalizePlatform(value string) Platform {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "linux":
		return PlatformLinux
	case "darwin", "macos":
		return PlatformMacOS
	case "windows", "win32":
		return PlatformWindows
	default:
		return ""
	}
}
