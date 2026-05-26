// Package storage resolves and manages runtime storage paths for Vrooli APIs.
package storage

// Class identifies a storage class with different durability/lifecycle expectations.
type Class string

const (
	// ClassConfig stores durable scenario configuration.
	ClassConfig Class = "config"
	// ClassData stores primary mutable application data.
	ClassData Class = "data"
	// ClassCache stores rebuildable cache artifacts.
	ClassCache Class = "cache"
	// ClassLogs stores operational logs and diagnostics.
	ClassLogs Class = "logs"
	// ClassState stores runtime state such as checkpoints and lockfiles.
	ClassState Class = "state"
)

// Profile specifies the deployment/runtime intent used for path defaults.
type Profile string

const (
	// ProfileAuto infers suitable defaults from the current OS.
	ProfileAuto Profile = "auto"
	// ProfileVPS uses server-style defaults (for system deployments).
	ProfileVPS Profile = "vps"
	// ProfileDesktop uses user-scoped desktop defaults.
	ProfileDesktop Profile = "desktop"
	// ProfileMobile is reserved for mobile adapters and currently falls back to desktop-like defaults.
	ProfileMobile Profile = "mobile"
)

// Paths contains resolved absolute directories for all storage classes.
type Paths struct {
	// ConfigDir is the absolute config directory.
	ConfigDir string
	// DataDir is the absolute primary data directory.
	DataDir string
	// CacheDir is the absolute cache directory.
	CacheDir string
	// LogsDir is the absolute logs directory.
	LogsDir string
	// StateDir is the absolute runtime state directory.
	StateDir string
}

// ForClass returns the resolved absolute directory for class.
func (p Paths) ForClass(class Class) (string, error) {
	switch class {
	case ClassConfig:
		return p.ConfigDir, nil
	case ClassData:
		return p.DataDir, nil
	case ClassCache:
		return p.CacheDir, nil
	case ClassLogs:
		return p.LogsDir, nil
	case ClassState:
		return p.StateDir, nil
	default:
		return "", &Error{Kind: ErrInvalidInput, Message: "unknown storage class", Details: string(class)}
	}
}

// Options configures one resolution request.
type Options struct {
	// ScenarioID is the scenario identifier used for path scoping.
	// Valid values are alnum plus '-', '_' and '.'.
	ScenarioID string
	// RootOverride forces all class roots under this absolute directory.
	// Useful in tests or controlled containerized environments.
	RootOverride string
}

// ResolverConfig configures a storage resolver and provides test seams.
type ResolverConfig struct {
	// AppID is included in generated paths. Defaults to "vrooli".
	AppID string
	// Profile controls class-root defaults. Defaults to ProfileAuto.
	Profile Profile

	// EnvGet reads environment variables. Defaults to os.Getenv.
	EnvGet func(key string) string

	// UserHomeDir resolves the operator home dir from which the runtime-home
	// default (~/.vrooli) is derived. Defaults to os.UserHomeDir. Composition
	// roots that may run under sudo should inject a sudo-aware resolver here.
	UserHomeDir func() (string, error)

	// RuntimeOS, UserConfigDir, and UserCacheDir are retained accepted seams for
	// API compatibility. They no longer influence the user-profile default, which
	// is now the OS-agnostic operator runtime home resolved via the repo-contract
	// runtime_home authority (see platform.go). Set UserHomeDir to redirect it.
	RuntimeOS     string
	UserConfigDir func() (string, error)
	UserCacheDir  func() (string, error)
}
