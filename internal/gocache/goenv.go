package gocache

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// goSettings is how one go command resolves the settings that decide which
// build cache it uses and whether it uses this protocol. It mirrors
// cmd/go/internal/cfg: a non-empty environment variable wins, then the user's
// go env file, then cmd/go's default.
type goSettings struct {
	// EnvFile is the go env file the command reads; empty for GOENV=off or
	// when no user configuration directory resolves.
	EnvFile string
	// Cache is the effective GOCACHE, or "off".
	Cache          string
	CacheFromEnv   bool
	Program        string
	ProgramFromEnv bool
}

// goDirs resolves the per-user directories cmd/go derives defaults from.
type goDirs struct {
	config func() (string, error)
	cache  func() (string, error)
}

// processDirs applies the Linux os.UserConfigDir/os.UserCacheDir rules to
// another process's environment.
func processDirs(lookup func(string) string) goDirs {
	xdg := func(key, fallback string) func() (string, error) {
		return func() (string, error) {
			if dir := lookup(key); dir != "" {
				if !filepath.IsAbs(dir) {
					return "", os.ErrInvalid
				}
				return dir, nil
			}
			home := lookup("HOME")
			if home == "" {
				return "", os.ErrNotExist
			}
			return filepath.Join(home, fallback), nil
		}
	}
	return goDirs{config: xdg("XDG_CONFIG_HOME", ".config"), cache: xdg("XDG_CACHE_HOME", ".cache")}
}

// ownDirs uses this process's platform-correct user directories.
func ownDirs() goDirs { return goDirs{config: os.UserConfigDir, cache: os.UserCacheDir} }

func resolveGoSettings(lookup func(string) string, dirs goDirs) goSettings {
	var settings goSettings
	switch file := lookup("GOENV"); {
	case file == "off":
	case file != "":
		settings.EnvFile = file
	default:
		if dir, err := dirs.config(); err == nil && dir != "" {
			settings.EnvFile = filepath.Join(dir, "go", "env")
		}
	}
	file := readGoEnvFile(settings.EnvFile)
	if value := lookup("GOCACHE"); value != "" {
		settings.Cache, settings.CacheFromEnv = value, true
	} else if value := file["GOCACHE"]; value != "" {
		settings.Cache = value
	} else if dir, err := dirs.cache(); err == nil && dir != "" {
		settings.Cache = filepath.Join(dir, "go-build")
	} else {
		settings.Cache = "off"
	}
	if value := lookup("GOCACHEPROG"); value != "" {
		settings.Program, settings.ProgramFromEnv = value, true
	} else {
		settings.Program = file["GOCACHEPROG"]
	}
	return settings
}

// readGoEnvFile parses a go env file exactly as cmd/go's readEnvFile does.
func readGoEnvFile(path string) map[string]string {
	values := map[string]string{}
	if path == "" {
		return values
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return values
	}
	for len(data) > 0 {
		line := data
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			line, data = data[:i], data[i+1:]
		} else {
			data = nil
		}
		i := bytes.IndexByte(line, '=')
		if i < 0 || line[0] < 'A' || 'Z' < line[0] {
			continue
		}
		values[string(line[:i])] = string(line[i+1:])
	}
	return values
}

// supportsCacheProg reports whether a toolchain's go command honours
// GOCACHEPROG without an experiment flag (Go 1.24 and later).
func supportsCacheProg(goVersion string) bool {
	version := strings.TrimPrefix(strings.TrimSpace(goVersion), "devel ")
	version, ok := strings.CutPrefix(version, "go1.")
	if !ok {
		return false
	}
	end := 0
	for end < len(version) && version[end] >= '0' && version[end] <= '9' {
		end++
	}
	minor, err := strconv.Atoi(version[:end])
	return err == nil && minor >= 24
}
