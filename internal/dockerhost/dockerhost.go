package dockerhost

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const (
	DaemonConfigPath = "/etc/docker/daemon.json"

	invalidDefaultCgroupParent = "default-cgroup-parent"
)

type Health struct {
	ClientInstalled     bool
	InfoOK              bool
	PermissionDenied    bool
	DaemonUnavailable   bool
	ServiceActive       bool
	ServiceFailed       bool
	ConfigValid         bool
	Detail              string
	ValidationDetail    string
	ServiceActiveDetail string
	ServiceFailedDetail string
}

type ConfigOptions struct {
	ApplyWorkloadCgroupPolicy bool
}

type ConfigRepairResult struct {
	Changed            bool
	RemovedInvalidKeys []string
	PreservedKeys      []string
	ValidationDetail   string
}

func InspectHealth() Health {
	var health Health
	if _, err := hostreqkit.LookPathFn("docker"); err != nil {
		health.Detail = err.Error()
		return health
	}
	health.ClientInstalled = true

	out, err := hostreqkit.CombinedOutputFn("docker", "info")
	if err == nil {
		health.InfoOK = true
		health.ConfigValid = true
		health.Detail = strings.TrimSpace(string(out))
		return health
	}
	detail := strings.TrimSpace(string(out))
	if detail == "" {
		detail = err.Error()
	}
	health.Detail = detail
	health.PermissionDenied = IsPermissionDenied(detail)
	health.DaemonUnavailable = IsDaemonUnavailable(detail)
	health.ServiceActive, health.ServiceActiveDetail = systemctlStatus("is-active", "docker")
	health.ServiceFailed, health.ServiceFailedDetail = systemctlStatus("is-failed", "docker")
	health.ConfigValid, health.ValidationDetail = ValidateDaemonConfig(DaemonConfigPath)
	return health
}

func IsPermissionDenied(detail string) bool {
	value := strings.ToLower(detail)
	return strings.Contains(value, "permission denied") ||
		strings.Contains(value, "got permission denied") ||
		strings.Contains(value, "connect: permission denied")
}

func IsDaemonUnavailable(detail string) bool {
	value := strings.ToLower(detail)
	return strings.Contains(value, "cannot connect to the docker daemon") ||
		strings.Contains(value, "is the docker daemon running") ||
		strings.Contains(value, "docker daemon is not running") ||
		strings.Contains(value, "error during connect")
}

func DiagnosticLine(detail string) string {
	lines := strings.Split(strings.TrimSpace(detail), "\n")
	fallback := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if fallback == "" {
			fallback = trimmed
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "cannot connect") ||
			strings.Contains(lower, "permission denied") ||
			strings.Contains(lower, "error during connect") ||
			strings.Contains(lower, "is the docker daemon running") ||
			strings.Contains(lower, "directives don't match") {
			return trimmed
		}
	}
	return fallback
}

func ConfigHasWorkloadPolicy(path string) bool {
	cfg, err := readConfig(path)
	if err != nil {
		return false
	}
	parent, _ := cfg["cgroup-parent"].(string)
	if parent != "workload.slice" {
		return false
	}
	return stringSliceContains(readExecOpts(cfg), "native.cgroupdriver=systemd")
}

func SanitizeDaemonConfig(path string, opts ConfigOptions, ensureOpts hostreqkit.EnsureOptions) (ConfigRepairResult, error) {
	cfg, err := readConfig(path)
	if err != nil {
		return ConfigRepairResult{}, err
	}
	result := ConfigRepairResult{PreservedKeys: preservedKeys(cfg)}
	if _, ok := cfg[invalidDefaultCgroupParent]; ok {
		delete(cfg, invalidDefaultCgroupParent)
		result.Changed = true
		result.RemovedInvalidKeys = append(result.RemovedInvalidKeys, invalidDefaultCgroupParent)
	}
	if opts.ApplyWorkloadCgroupPolicy {
		if cfg["cgroup-parent"] != "workload.slice" {
			cfg["cgroup-parent"] = "workload.slice"
			result.Changed = true
		}
		execOpts := readExecOpts(cfg)
		if !stringSliceContains(execOpts, "native.cgroupdriver=systemd") {
			execOpts = append(execOpts, "native.cgroupdriver=systemd")
			cfg["exec-opts"] = execOpts
			result.Changed = true
		}
	}
	if !result.Changed {
		if _, err := hostreqkit.ReadFileFn(path); err != nil {
			result.ValidationDetail = "daemon config file not present; nothing to sanitize"
			return result, nil
		}
		valid, detail := ValidateDaemonConfig(path)
		if !valid {
			return result, fmt.Errorf("docker daemon config validation failed: %s", detail)
		}
		result.ValidationDetail = detail
		return result, nil
	}

	output, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return result, fmt.Errorf("marshal docker config: %w", err)
	}
	content := string(output) + "\n"
	if err := hostreqkit.EnsureManagedDir(filepath.Dir(path), ensureOpts.SudoMode, ensureOpts); err != nil {
		return result, fmt.Errorf("create docker config dir: %w", err)
	}
	if err := hostreqkit.InstallManagedContent(path, content, ensureOpts.SudoMode, ensureOpts); err != nil {
		return result, fmt.Errorf("install docker daemon.json: %w", err)
	}
	if !ensureOpts.DryRun {
		valid, detail := ValidateDaemonConfig(path)
		if !valid {
			return result, fmt.Errorf("docker daemon config validation failed after repair: %s", detail)
		}
		result.ValidationDetail = detail
	}
	return result, nil
}

func ValidateDaemonConfig(path string) (bool, string) {
	if _, err := hostreqkit.ReadFileFn(path); err != nil {
		return true, "daemon config file not present; skipped daemon config validation"
	}
	if _, err := hostreqkit.LookPathFn("dockerd"); err != nil {
		return true, "dockerd not found; skipped daemon config validation"
	}
	out, err := hostreqkit.CombinedOutputFn("dockerd", "--validate", "--config-file", path)
	detail := strings.TrimSpace(string(out))
	if detail == "" && err != nil {
		detail = err.Error()
	}
	if err != nil {
		return false, detail
	}
	return true, detail
}

func StartDockerService(opts hostreqkit.EnsureOptions) error {
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"daemon-reload"}, opts); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"reset-failed", "docker"}, opts); err != nil {
		return fmt.Errorf("systemctl reset-failed docker: %w", err)
	}
	if hostreqkit.CommandAvailable("systemctl") {
		if _, err := hostreqkit.CombinedOutputFn("systemctl", "is-enabled", "docker"); err != nil {
			if enableErr := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"enable", "docker"}, opts); enableErr != nil {
				return fmt.Errorf("systemctl enable docker: %w", enableErr)
			}
		}
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"start", "docker"}, opts); err != nil {
		return fmt.Errorf("systemctl start docker: %w", err)
	}
	return nil
}

func RestartDockerIfActive(opts hostreqkit.EnsureOptions) error {
	if _, err := hostreqkit.CombinedOutputFn("systemctl", "is-active", "docker"); err != nil {
		return nil
	}
	return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"restart", "docker"}, opts)
}

func systemctlStatus(args ...string) (bool, string) {
	if _, err := hostreqkit.LookPathFn("systemctl"); err != nil {
		return false, err.Error()
	}
	out, err := hostreqkit.CombinedOutputFn("systemctl", args...)
	detail := strings.TrimSpace(string(out))
	if detail == "" && err != nil {
		detail = err.Error()
	}
	return err == nil, detail
}

func readConfig(path string) (map[string]any, error) {
	cfg := make(map[string]any)
	data, err := hostreqkit.ReadFileFn(path)
	if err != nil {
		return cfg, nil
	}
	if strings.TrimSpace(string(data)) == "" {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse docker daemon config %s: %w", path, err)
	}
	return cfg, nil
}

func readExecOpts(cfg map[string]any) []string {
	result := []string{}
	switch existing := cfg["exec-opts"].(type) {
	case []any:
		for _, value := range existing {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, text)
			}
		}
	case []string:
		result = append(result, existing...)
	}
	return result
}

func preservedKeys(cfg map[string]any) []string {
	keys := make([]string, 0, len(cfg))
	for key := range cfg {
		if key == invalidDefaultCgroupParent {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
