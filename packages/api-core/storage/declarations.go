package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	platform := NormalizePlatform(string(requested))
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
	identity := SyntheticIdentity(platform)
	if platform == HostPlatform() {
		var err error
		identity, err = HostIdentity()
		if err != nil && seams.UserHomeDir == nil {
			return "", fmt.Errorf("resolve host identity: %w", err)
		}
	}
	return resolvePortableTokens(platform, value, mergeSeams(DefaultSeams(platform, identity), seams))
}

// ResolveOwnerStoragePath is the compatibility entry point for owner-aware
// resolution. The Resolver is the only implementation of the decision; this
// function exists so existing inventory and retention callers can migrate
// without carrying a second path authority.
func ResolveOwnerStoragePath(repoRoot string, owner OwnerManifest, entry StorageEntry, requested Platform, seams PlatformSeams) (string, error) {
	resolver, err := NewResolver(ResolverConfig{UserHomeDir: seams.UserHomeDir})
	if err != nil {
		return "", err
	}
	return resolver.ResolveOwner(repoRoot, owner, entry, requested, seams)
}

func containsPortableToken(value string) bool {
	return strings.Contains(value, "$USER_HOME") ||
		strings.Contains(value, "$USER_CONFIG_DIR") ||
		strings.Contains(value, "$USER_CACHE_DIR") ||
		strings.Contains(value, "$USER_STATE_DIR") ||
		strings.Contains(value, "$USER_DATA_DIR") ||
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

// UserIdentity is the identity whose home directory is used for portable
// storage resolution. Synthetic identities make foreign-platform verification
// deterministic without pretending that those directories exist on the host.
type UserIdentity struct {
	HomeDir string
}

// HostIdentity returns the real user's identity on the machine running Vrooli.
func HostIdentity() (UserIdentity, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return UserIdentity{}, err
	}
	if strings.TrimSpace(home) == "" {
		return UserIdentity{}, fmt.Errorf("home directory is empty")
	}
	return UserIdentity{HomeDir: home}, nil
}

// SyntheticIdentity returns the documented stable identity for a target
// platform that is not the host platform.
func SyntheticIdentity(platform Platform) UserIdentity {
	switch NormalizePlatform(string(platform)) {
	case PlatformWindows:
		return UserIdentity{HomeDir: `C:\Users\vrooli`}
	case PlatformMacOS:
		return UserIdentity{HomeDir: "/Users/vrooli"}
	default:
		return UserIdentity{HomeDir: "/home/vrooli"}
	}
}

// DefaultSeams builds platform-native directory conventions from identity.
// Callers may override any individual seam by passing it to a resolver.
func DefaultSeams(platform Platform, identity UserIdentity) PlatformSeams {
	platform = NormalizePlatform(string(platform))
	home := identity.HomeDir
	join := func(parts ...string) string { return joinPlatformPath(platform, parts...) }
	switch platform {
	case PlatformWindows:
		return PlatformSeams{
			UserHomeDir:   func() (string, error) { return home, nil },
			UserConfigDir: func() (string, error) { return join(home, `AppData\Roaming`), nil },
			UserCacheDir:  func() (string, error) { return join(home, `AppData\Local`), nil },
			UserStateDir:  func() (string, error) { return join(home, `AppData\Local`), nil },
			TempDir:       func() string { return `C:\Windows\Temp` },
		}
	case PlatformMacOS:
		return PlatformSeams{
			UserHomeDir:   func() (string, error) { return home, nil },
			UserConfigDir: func() (string, error) { return join(home, "Library", "Application Support"), nil },
			UserCacheDir:  func() (string, error) { return join(home, "Library", "Caches"), nil },
			UserStateDir:  func() (string, error) { return join(home, "Library", "Application Support"), nil },
			TempDir:       func() string { return "/tmp" },
		}
	default:
		return PlatformSeams{
			UserHomeDir:   func() (string, error) { return home, nil },
			UserConfigDir: func() (string, error) { return join(home, ".config"), nil },
			UserCacheDir:  func() (string, error) { return join(home, ".cache"), nil },
			UserStateDir:  func() (string, error) { return join(home, ".local", "state"), nil },
			TempDir:       func() string { return "/tmp" },
		}
	}
}

func joinPlatformPath(platform Platform, parts ...string) string {
	if NormalizePlatform(string(platform)) != PlatformWindows {
		return filepath.Join(parts...)
	}
	result := ""
	for _, part := range parts {
		part = strings.TrimRight(part, `/\`)
		if part == "" {
			continue
		}
		if result == "" {
			result = part
		} else {
			result += `\` + strings.TrimLeft(part, `/\`)
		}
	}
	return result
}

func mergeSeams(defaults, overrides PlatformSeams) PlatformSeams {
	if overrides.UserHomeDir != nil {
		defaults.UserHomeDir = overrides.UserHomeDir
	}
	if overrides.UserConfigDir != nil {
		defaults.UserConfigDir = overrides.UserConfigDir
	}
	if overrides.UserCacheDir != nil {
		defaults.UserCacheDir = overrides.UserCacheDir
	}
	if overrides.UserStateDir != nil {
		defaults.UserStateDir = overrides.UserStateDir
	}
	if overrides.TempDir != nil {
		defaults.TempDir = overrides.TempDir
	}
	return defaults
}

func resolvePortableTokens(platform Platform, value string, seams PlatformSeams) (string, error) {
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
	if platform == PlatformWindows {
		values["$USER_DATA_DIR"] = joinPlatformPath(platform, values["$USER_HOME"], `AppData\Local`)
	} else if platform == PlatformMacOS {
		values["$USER_DATA_DIR"] = filepath.Join(values["$USER_HOME"], "Library", "Application Support")
	} else {
		values["$USER_DATA_DIR"] = filepath.Join(values["$USER_HOME"], ".local", "share")
	}
	value = strings.ReplaceAll(value, "%USERPROFILE%", "$USER_HOME")
	if value == "~" || strings.HasPrefix(value, "~/") {
		value = "$USER_HOME" + strings.TrimPrefix(value, "~")
	}
	value = strings.ReplaceAll(value, "$HOME", "$USER_HOME")
	for token, replacement := range values {
		value = strings.ReplaceAll(value, token, replacement)
	}
	if !isAbsoluteFor(platform, value) {
		return "", fmt.Errorf("storage path %q resolves to non-absolute path %q", value, value)
	}
	return cleanForPlatform(platform, value), nil
}

// NormalizePlatform accepts the manifest and CLI aliases for supported OSes.
func NormalizePlatform(value string) Platform {
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

func normalizePlatform(value string) Platform { return NormalizePlatform(value) }

// HostPlatform returns the normalized platform of the current process.
func HostPlatform() Platform { return NormalizePlatform(runtime.GOOS) }

func platformIncluded(platforms []Platform, requested Platform) bool {
	for _, platform := range platforms {
		if platform == requested {
			return true
		}
	}
	return false
}

func isAbsoluteFor(platform Platform, value string) bool {
	platform = NormalizePlatform(string(platform))
	if platform == PlatformWindows {
		if strings.HasPrefix(value, `\\`) {
			return true
		}
		return len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\')
	}
	return strings.HasPrefix(value, "/")
}

func cleanForPlatform(platform Platform, value string) string {
	if NormalizePlatform(string(platform)) == PlatformWindows {
		if strings.HasPrefix(value, `\\`) {
			return strings.ReplaceAll(value, "/", `\`)
		}
		return strings.ReplaceAll(filepath.Clean(value), "/", `\`)
	}
	return filepath.Clean(value)
}
