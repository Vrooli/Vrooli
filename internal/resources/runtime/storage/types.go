package storage

type Class string

const (
	ClassConfig Class = "config"
	ClassData   Class = "data"
	ClassCache  Class = "cache"
	ClassLogs   Class = "logs"
	ClassState  Class = "state"
)

type Profile string

const (
	ProfileAuto    Profile = "auto"
	ProfileDesktop Profile = "desktop"
	ProfileMobile  Profile = "mobile"
	ProfileVPS     Profile = "vps"
)

type Paths struct {
	ConfigDir string
	DataDir   string
	CacheDir  string
	LogsDir   string
	StateDir  string
}

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

type Options struct {
	ResourceID   string
	RootOverride string
}

type ResolverConfig struct {
	AppID string

	Profile Profile

	RuntimeOS string
	EnvGet    func(key string) string

	UserHomeDir   func() (string, error)
	UserConfigDir func() (string, error)
	UserCacheDir  func() (string, error)
}

