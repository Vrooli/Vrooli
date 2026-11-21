package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

// ollayaModelRoot resolves the resource-owned storage location through the
// runtime storage authority and requires the daemon's configured path to agree.
// An arbitrary OLLAYA_MODELS override must never redirect writes outside the
// location capacity admission measures.
func ollayaModelRoot(getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve invoking user's home for Ollaya storage: %w", err)
	}
	_, _, sudoed := hostreqkit.InvokingUserIDs()
	cfg := runtimestorage.ResolverConfig{
		AppID: "vrooli", RuntimeOS: runtime.GOOS, EnvGet: getenv,
		UserHomeDir: func() (string, error) {
			if strings.TrimSpace(home) == "" {
				return "", errors.New("invoking user's home directory is unavailable")
			}
			return home, nil
		},
	}
	if sudoed && runtime.GOOS == "linux" {
		cfg.EnvGet = func(key string) string {
			switch key {
			case "XDG_DATA_HOME":
				return filepath.Join(home, ".local", "share")
			case "XDG_STATE_HOME":
				return filepath.Join(home, ".local", "state")
			default:
				return getenv(key)
			}
		}
		cfg.UserConfigDir = func() (string, error) { return filepath.Join(home, ".config"), nil }
		cfg.UserCacheDir = func() (string, error) { return filepath.Join(home, ".cache"), nil }
	}
	resolver, err := runtimestorage.NewResolver(cfg)
	if err != nil {
		return "", fmt.Errorf("resolve managed Ollaya storage: %w", err)
	}
	paths, err := resolver.Resolve(runtimestorage.Options{ResourceID: "ollaya"})
	if err != nil {
		return "", fmt.Errorf("resolve managed Ollaya resource paths: %w", err)
	}
	root := filepath.Join(paths.DataDir, "models")
	if !filepath.IsAbs(root) {
		return "", errors.New("managed Ollaya model-store path is not absolute; refusing pull")
	}
	if override := strings.TrimSpace(getenv("OLLAYA_MODELS")); override != "" {
		override, err = filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve OLLAYA_MODELS override: %w", err)
		}
		if filepath.Clean(override) != filepath.Clean(root) {
			return "", errors.New("OLLAYA_MODELS does not match the managed Ollaya resource model-store path; refusing an unverified write location")
		}
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve managed Ollaya model-store symlinks: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("managed Ollaya model store is unavailable or not a directory: %s", root)
	}
	return root, nil
}
