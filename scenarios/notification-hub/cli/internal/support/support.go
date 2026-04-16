package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type DefaultsConfig struct {
	ProfileID string `json:"profile_id,omitempty"`
}

type DefaultsStore struct {
	file *cliutil.ConfigFile
}

type Dependencies struct {
	Core            func() *cliapp.ScenarioApp
	Defaults        func() *DefaultsStore
	ProfileOverride func() string
	AppName         string
}

func NewDefaultsStore(core *cliapp.ScenarioApp) (*DefaultsStore, error) {
	if core == nil || core.ConfigFile == nil {
		return nil, fmt.Errorf("scenario config is not initialized")
	}
	cfgFile, err := cliutil.NewConfigFile(filepath.Join(filepath.Dir(core.ConfigFile.Path), "defaults.json"))
	if err != nil {
		return nil, err
	}
	return &DefaultsStore{file: cfgFile}, nil
}

func (s *DefaultsStore) Load() (DefaultsConfig, error) {
	if s == nil || s.file == nil {
		return DefaultsConfig{}, nil
	}
	cfg := DefaultsConfig{}
	if err := s.file.Load(&cfg); err != nil {
		return DefaultsConfig{}, err
	}
	return cfg, nil
}

func (s *DefaultsStore) Save(cfg DefaultsConfig) error {
	if s == nil || s.file == nil {
		return fmt.Errorf("defaults store is not initialized")
	}
	return s.file.Save(cfg)
}

func (d Dependencies) ScenarioApp() *cliapp.ScenarioApp {
	if d.Core == nil {
		return nil
	}
	return d.Core()
}

func (d Dependencies) DefaultsStore() *DefaultsStore {
	if d.Defaults == nil {
		return nil
	}
	return d.Defaults()
}

func (d Dependencies) DefaultConfig() DefaultsConfig {
	store := d.DefaultsStore()
	if store == nil {
		return DefaultsConfig{}
	}
	cfg, err := store.Load()
	if err != nil {
		return DefaultsConfig{}
	}
	return cfg
}

func (d Dependencies) ProfileID() string {
	if d.ProfileOverride != nil {
		if value := strings.TrimSpace(d.ProfileOverride()); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("NOTIFICATION_HUB_PROFILE_ID")); value != "" {
		return value
	}
	return strings.TrimSpace(d.DefaultConfig().ProfileID)
}

func (d Dependencies) ResolveProfileID(flagValue string) (string, error) {
	if value := strings.TrimSpace(flagValue); value != "" {
		return value, nil
	}
	if value := d.ProfileID(); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("profile ID is required; pass --profile-id or run 'notification-hub configure profile_id <id>'")
}

func (d Dependencies) ScopedGet(profileID, path string, query url.Values) ([]byte, error) {
	return d.ScenarioApp().Get("/profiles/"+strings.TrimSpace(profileID)+normalizeScopedPath(path), query)
}

func (d Dependencies) ScopedRequest(profileID, method, path string, query url.Values, body interface{}) ([]byte, error) {
	return d.ScenarioApp().Request(method, "/profiles/"+strings.TrimSpace(profileID)+normalizeScopedPath(path), query, body)
}

func normalizeScopedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func ExtractLegacyGlobals(args []string) (remaining []string, profileID string, apiKey string, err error) {
	if len(args) == 0 {
		return nil, "", "", nil
	}
	remaining = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--profile-id":
			if i+1 >= len(args) {
				return nil, "", "", fmt.Errorf("--profile-id requires a value")
			}
			i++
			profileID = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--profile-id="):
			profileID = strings.TrimSpace(strings.TrimPrefix(arg, "--profile-id="))
		case arg == "--api-key":
			if i+1 >= len(args) {
				return nil, "", "", fmt.Errorf("--api-key requires a value")
			}
			i++
			apiKey = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--api-key="):
			apiKey = strings.TrimSpace(strings.TrimPrefix(arg, "--api-key="))
		case arg == "--api-url":
			if i+1 >= len(args) {
				return nil, "", "", fmt.Errorf("--api-url requires a value")
			}
			remaining = append(remaining, "--api-base", args[i+1])
			i++
		case strings.HasPrefix(arg, "--api-url="):
			remaining = append(remaining, "--api-base="+strings.TrimSpace(strings.TrimPrefix(arg, "--api-url=")))
		case arg == "--format":
			if i+1 >= len(args) {
				return nil, "", "", fmt.Errorf("--format requires a value")
			}
			i++
		case strings.HasPrefix(arg, "--format="):
			continue
		default:
			remaining = append(remaining, arg)
		}
	}
	return remaining, profileID, apiKey, nil
}

func DefaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func DerefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func PrintJSONReport(stdout *os.File, value interface{}) error {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func BoolFlag(fs *flag.FlagSet, name string, target *bool, usage string) {
	fs.BoolVar(target, name, false, usage)
}
