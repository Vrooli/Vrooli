package cliutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConfigFile manages loading and saving JSON config files and ensures the parent
// directory exists.
type ConfigFile struct {
	Path string
}

func NewConfigFile(path string) (*ConfigFile, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}
	return &ConfigFile{Path: path}, nil
}

func (c *ConfigFile) Load(target interface{}) error {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}

func (c *ConfigFile) Save(value interface{}) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(c.Path, payload, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

type APIBaseOptions struct {
	Override     string
	EnvVars      []string
	ConfigBase   string
	PortEnvVars  []string
	PortDetector func() string
	DefaultBase  string
}

// ValidateAPIBase resolves and validates an API base URL, returning a trimmed
// base or an error with guidance when missing or malformed.
func ValidateAPIBase(opts APIBaseOptions) (string, error) {
	base := DetermineAPIBase(opts)
	if base == "" {
		return "", fmt.Errorf("api base URL is empty; set --api-base, %s, config api_base, or a port env", strings.Join(append(opts.EnvVars, opts.PortEnvVars...), ", "))
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid api base URL %q", base)
	}
	return base, nil
}

// DetermineAPIBase resolves the API base URL from override flags, environment,
// config, port hints, and a default.
func DetermineAPIBase(opts APIBaseOptions) string {
	trim := func(val string) string {
		return strings.TrimRight(strings.TrimSpace(val), "/")
	}

	if base := trim(opts.Override); base != "" {
		return base
	}
	for _, env := range opts.EnvVars {
		if val := trim(os.Getenv(env)); val != "" {
			return val
		}
	}
	if base := trim(opts.ConfigBase); base != "" {
		return base
	}
	for _, env := range opts.PortEnvVars {
		if port := strings.TrimSpace(os.Getenv(env)); port != "" {
			return fmt.Sprintf("http://localhost:%s", port)
		}
	}
	if opts.PortDetector != nil {
		if port := strings.TrimSpace(opts.PortDetector()); port != "" {
			return fmt.Sprintf("http://localhost:%s", port)
		}
	}
	return trim(opts.DefaultBase)
}

// ResolveSourceRoot returns the first existing directory from the provided
// env vars and buildSourceRoot.
func ResolveSourceRoot(buildSourceRoot string, envVars ...string) string {
	var candidates []string
	for _, env := range envVars {
		value := strings.TrimSpace(os.Getenv(env))
		if value != "" {
			candidates = append(candidates, value)
		}
	}
	candidates = append(candidates, buildSourceRoot)

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || candidate == "unknown" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func PrintJSON(data []byte) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, data, "", "  "); err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(pretty.String())
}

func PrintJSONMap(m map[string]interface{}, indent int) {
	if len(m) == 0 {
		fmt.Printf("%s(none)\n", strings.Repeat(" ", indent))
		return
	}
	prefix := strings.Repeat(" ", indent)
	for key, value := range m {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", prefix, key)
			PrintJSONMap(v, indent+2)
		default:
			jsonVal, err := json.Marshal(v)
			if err != nil {
				fmt.Printf("%s%s: %v\n", prefix, key, v)
				continue
			}
			fmt.Printf("%s%s: %s\n", prefix, key, string(jsonVal))
		}
	}
}

// ResolveTimeout returns the first parseable duration from envVars or the fallback.
// Accepts standard duration strings (e.g. "45s", "2m") and plain integers as seconds.
func ResolveTimeout(envVars []string, fallback time.Duration) time.Duration {
	for _, env := range envVars {
		val := strings.TrimSpace(os.Getenv(env))
		if val == "" {
			continue
		}
		parsed, err := time.ParseDuration(val)
		if err != nil {
			if secs, convErr := time.ParseDuration(val + "s"); convErr == nil {
				return secs
			}
			continue
		}
		return parsed
	}
	return fallback
}
