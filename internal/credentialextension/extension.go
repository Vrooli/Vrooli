// Package credentialextension manages the browser native-messaging registration
// for the Secrets Manager extension. It handles public registration metadata
// only; credentials and owner tokens never enter this package.
package credentialextension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const HostName = "com.vrooli.secrets-manager"

var extensionIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

// Options identifies a registration operation. OS and HomeDir are injectable
// so the control-plane command can be tested without touching a real browser
// profile. Empty OS and HomeDir use the current host.
type Options struct {
	HostPath    string
	ExtensionID string
	Browser     string
	OS          string
	HomeDir     string
}

// Target describes one browser's public registration location.
type Target struct {
	Browser            string `json:"browser"`
	ManifestPath       string `json:"manifest_path"`
	RegistryKey        string `json:"registry_key,omitempty"`
	ManifestInstalled  bool   `json:"manifest_installed"`
	RegistryConfigured bool   `json:"registry_configured"`
}

// Result is safe to print in command output. It contains no credential data.
type Result struct {
	Operation   string   `json:"operation"`
	HostPath    string   `json:"host_path"`
	ExtensionID string   `json:"extension_id"`
	Changed     bool     `json:"changed"`
	Targets     []Target `json:"targets"`
}

type nativeManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`
}

// Install registers the host for the selected browser profiles.
func Install(opts Options) (Result, error) {
	normalized, targets, err := prepare(opts, true)
	if err != nil {
		return Result{}, err
	}
	result := Result{Operation: "install", HostPath: normalized.HostPath, ExtensionID: normalized.ExtensionID, Targets: targets}
	for i := range result.Targets {
		manifest, err := manifestFor(normalized, result.Targets[i].Browser)
		if err != nil {
			return Result{}, err
		}
		changed, err := writeManifest(result.Targets[i].ManifestPath, manifest)
		if err != nil {
			return Result{}, err
		}
		result.Changed = result.Changed || changed
		if normalized.OS == "windows" && runtime.GOOS == "windows" {
			changed, err = registerWindows(result.Targets[i].RegistryKey, result.Targets[i].ManifestPath)
			if err != nil {
				return Result{}, err
			}
			result.Changed = result.Changed || changed
		}
		result.Targets[i].ManifestInstalled = true
		result.Targets[i].RegistryConfigured = normalized.OS != "windows" || runtime.GOOS == "windows"
	}
	return result, nil
}

// Uninstall removes the selected browser registrations and leaves the host
// binary and all Secrets Manager data untouched.
func Uninstall(opts Options) (Result, error) {
	normalized, targets, err := prepare(opts, false)
	if err != nil {
		return Result{}, err
	}
	result := Result{Operation: "uninstall", HostPath: normalized.HostPath, ExtensionID: normalized.ExtensionID, Targets: targets}
	for i := range result.Targets {
		changed, err := removeManifest(result.Targets[i].ManifestPath)
		if err != nil {
			return Result{}, err
		}
		result.Changed = result.Changed || changed
		if normalized.OS == "windows" && runtime.GOOS == "windows" {
			changed, err = unregisterWindows(result.Targets[i].RegistryKey)
			if err != nil {
				return Result{}, err
			}
			result.Changed = result.Changed || changed
		}
	}
	return statusResult(normalized, result)
}

// Status reports registration state without requiring the host binary to be
// present. This lets an operator diagnose a stale browser profile.
func Status(opts Options) (Result, error) {
	normalized, targets, err := prepare(opts, false)
	if err != nil {
		return Result{}, err
	}
	result := Result{Operation: "status", HostPath: normalized.HostPath, ExtensionID: normalized.ExtensionID, Targets: targets}
	return statusResult(normalized, result)
}

func statusResult(opts Options, result Result) (Result, error) {
	for i := range result.Targets {
		installed, err := manifestMatches(result.Targets[i].ManifestPath, opts, result.Targets[i].Browser)
		if err != nil {
			return Result{}, err
		}
		result.Targets[i].ManifestInstalled = installed
		if opts.OS == "windows" && runtime.GOOS == "windows" {
			result.Targets[i].RegistryConfigured = registryConfigured(result.Targets[i].RegistryKey)
		} else {
			result.Targets[i].RegistryConfigured = opts.OS != "windows"
		}
	}
	return result, nil
}

func prepare(opts Options, requireHost bool) (Options, []Target, error) {
	opts.OS = strings.ToLower(strings.TrimSpace(opts.OS))
	if opts.OS == "" {
		opts.OS = runtime.GOOS
	}
	if opts.OS != "linux" && opts.OS != "darwin" && opts.OS != "windows" {
		return Options{}, nil, fmt.Errorf("unsupported operating system %q", opts.OS)
	}
	opts.Browser = strings.ToLower(strings.TrimSpace(opts.Browser))
	if opts.Browser == "" {
		opts.Browser = "chrome"
	}
	if opts.Browser != "chrome" && opts.Browser != "chromium" && opts.Browser != "firefox" && opts.Browser != "all" {
		return Options{}, nil, fmt.Errorf("unsupported browser %q", opts.Browser)
	}
	opts.ExtensionID = strings.TrimSpace(opts.ExtensionID)
	if !extensionIDPattern.MatchString(opts.ExtensionID) {
		return Options{}, nil, errors.New("extension-id must contain 1-128 letters, numbers, dots, underscores, or hyphens")
	}
	opts.HostPath = strings.TrimSpace(opts.HostPath)
	if opts.HostPath == "" || !filepath.IsAbs(opts.HostPath) {
		return Options{}, nil, errors.New("host-path must be an absolute path")
	}
	if requireHost {
		info, err := os.Stat(opts.HostPath)
		if err != nil {
			return Options{}, nil, fmt.Errorf("host-path: %w", err)
		}
		if !info.Mode().IsRegular() {
			return Options{}, nil, errors.New("host-path must be a regular file")
		}
		if opts.OS != "windows" && info.Mode()&0111 == 0 {
			return Options{}, nil, errors.New("host-path must be executable")
		}
	}
	if opts.HomeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Options{}, nil, fmt.Errorf("resolve home directory: %w", err)
		}
		opts.HomeDir = home
	}
	if !filepath.IsAbs(opts.HomeDir) {
		return Options{}, nil, errors.New("home directory must be an absolute path")
	}

	browsers := []string{opts.Browser}
	if opts.Browser == "all" {
		browsers = []string{"chrome", "chromium", "firefox"}
	}
	targets := make([]Target, 0, len(browsers))
	for _, browser := range browsers {
		manifestPath, registryKey := registrationLocation(opts.OS, opts.HomeDir, browser)
		targets = append(targets, Target{Browser: browser, ManifestPath: manifestPath, RegistryKey: registryKey})
	}
	return opts, targets, nil
}

func registrationLocation(goos, home, browser string) (string, string) {
	var dir string
	switch goos {
	case "linux":
		switch browser {
		case "chrome":
			dir = filepath.Join(home, ".config", "google-chrome", "NativeMessagingHosts")
		case "chromium":
			dir = filepath.Join(home, ".config", "chromium", "NativeMessagingHosts")
		case "firefox":
			dir = filepath.Join(home, ".mozilla", "native-messaging-hosts")
		}
	case "darwin":
		switch browser {
		case "chrome":
			dir = filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
		case "chromium":
			dir = filepath.Join(home, "Library", "Application Support", "Chromium", "NativeMessagingHosts")
		case "firefox":
			dir = filepath.Join(home, "Library", "Application Support", "Mozilla", "NativeMessagingHosts")
		}
	case "windows":
		dir = filepath.Join(home, ".vrooli", "native-messaging-hosts")
	}
	registryKey := ""
	if goos == "windows" {
		base := `HKCU\Software\`
		vendor := map[string]string{"chrome": "Google\\Chrome", "chromium": "Chromium", "firefox": "Mozilla"}[browser]
		registryKey = base + vendor + `\NativeMessagingHosts\` + HostName
	}
	return filepath.Join(dir, HostName+".json"), registryKey
}

func manifestFor(opts Options, browser string) (nativeManifest, error) {
	manifest := nativeManifest{Name: HostName, Description: "Vrooli Secrets Manager native host", Path: opts.HostPath, Type: "stdio"}
	if browser == "firefox" {
		manifest.AllowedExtensions = []string{opts.ExtensionID}
	} else {
		manifest.AllowedOrigins = []string{"chrome-extension://" + opts.ExtensionID + "/"}
	}
	return manifest, nil
}

func writeManifest(path string, manifest nativeManifest) (bool, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	if existing, err := os.ReadFile(path); err == nil && string(existing) == string(data) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vrooli-native-host-*.tmp")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return false, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return false, err
	}
	return true, nil
}

func removeManifest(path string) (bool, error) {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func manifestMatches(path string, opts Options, browser string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var actual nativeManifest
	if err := json.Unmarshal(data, &actual); err != nil {
		return false, nil
	}
	want, _ := manifestFor(opts, browser)
	if browser == "firefox" {
		return actual.Name == want.Name && actual.Path == want.Path && len(actual.AllowedExtensions) == 1 && actual.AllowedExtensions[0] == opts.ExtensionID, nil
	}
	return actual.Name == want.Name && actual.Path == want.Path && len(actual.AllowedOrigins) == 1 && actual.AllowedOrigins[0] == "chrome-extension://"+opts.ExtensionID+"/", nil
}

func registerWindows(key, manifestPath string) (bool, error) {
	cmd := exec.Command("reg.exe", "ADD", key, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f")
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("register native host in %s: %w: %s", key, err, strings.TrimSpace(string(output)))
	}
	return true, nil
}

func unregisterWindows(key string) (bool, error) {
	cmd := exec.Command("reg.exe", "DELETE", key, "/f")
	if output, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(strings.ToLower(string(output)), "unable to find") {
			return false, nil
		}
		return false, fmt.Errorf("remove native host registry key %s: %w: %s", key, err, strings.TrimSpace(string(output)))
	}
	return true, nil
}

func registryConfigured(key string) bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return exec.CommandContext(ctx, "reg.exe", "QUERY", key, "/ve").Run() == nil
}
