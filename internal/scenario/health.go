package scenario

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	healthParameterA = 2
)

//nolint:gocyclo // health checks preserve HTTP, process, port, and retry failure semantics.
func PerformHealthCheck(check HealthCheck, ports map[string]int) error {
	switch strings.TrimSpace(check.Type) {
	case "", "http":
		target, err := ExpandHealthTarget(check.Target, ports)
		if err != nil {
			return err
		}
		if _, err := url.Parse(target); err != nil {
			return fmt.Errorf("invalid URL %q: %w", target, err)
		}

		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = tuning.ServiceHealthTimeout()
		}

		client := &http.Client{Timeout: timeout}
		resp, err := client.Get(target)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return nil
	case "connect_rpc":

		target, err := ExpandHealthTarget(check.Target, ports)
		if err != nil {
			return err
		}
		if _, err := url.Parse(target); err != nil {
			return fmt.Errorf("invalid URL %q: %w", target, err)
		}

		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = tuning.ServiceHealthTimeout()
		}

		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader([]byte("{}")))
		if err != nil {
			return fmt.Errorf("build connect_rpc request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return nil
	case "postgres":
		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = tuning.HealthCheckTimeout()
		}

		address := "127.0.0.1:5432"
		if parsed, err := parsePostgresAddress(check.Target); err == nil && parsed != "" {
			address = parsed
		}

		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return err
		}
		_ = conn.Close()
		return nil
	default:
		return fmt.Errorf("unsupported health check type %q", check.Type)
	}
}

func parsePostgresAddress(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", nil
	}

	if strings.HasPrefix(target, "postgres://") || strings.HasPrefix(target, "postgresql://") {
		parsed, err := url.Parse(target)
		if err != nil {
			return "", err
		}
		host := parsed.Hostname()
		if host == "" {
			return "", nil
		}
		port := parsed.Port()
		if port == "" {
			port = "5432"
		}
		return net.JoinHostPort(host, port), nil
	}

	if strings.Contains(target, ":") {
		host, port, err := net.SplitHostPort(target)
		if err == nil && host != "" && port != "" {
			return net.JoinHostPort(host, port), nil
		}
		if err != nil {
			return "", err
		}
	}

	return "", nil
}

func scanScenarioNames(baseDir string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		servicePath := filepath.Join(baseDir, entry.Name(), filepath.FromSlash(defaultScenarioServiceRelPath))
		if _, err := os.Stat(servicePath); err == nil {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

// scanSandboxScenarioNames mirrors the bash sandbox discovery contract: the
// merged dir can represent the repo root, the scenarios directory, or one
// specific scenario depending on the active sandbox scope.
func scanSandboxScenarioNames(root string, env SandboxEnv) ([]string, error) {
	if !env.Enabled() {
		return nil, nil
	}
	if info, err := os.Stat(env.Merged); err != nil || !info.IsDir() {
		return nil, nil
	}
	scope := normalizeSandboxScope(env.Scope)
	scenarioDir := strings.Trim(strings.TrimSpace(filepath.ToSlash(contractPaths.ScenarioDirName(root))), "/")
	prefix := strings.Trim(strings.TrimSpace(filepath.ToSlash(contractPaths.ScenarioScopePrefix(root))), "/")
	if prefix == "" {
		prefix = scenarioDir
	}
	switch {
	case contractPaths.IsFullRepoScope(root, scope):
		return scanScenarioNames(filepath.Join(env.Merged, filepath.FromSlash(scenarioDir)))
	case scope == scenarioDir:
		return scanScenarioNames(env.Merged)
	case strings.HasPrefix(scope, prefix+"/"):
		name := strings.TrimPrefix(scope, prefix+"/")
		name = strings.SplitN(name, "/", healthParameterA)[0]
		resolved := ResolveMergedPath(root, name, env.Scope, env.Merged)
		if _, err := os.Stat(filepath.Join(resolved, filepath.FromSlash(defaultScenarioServiceRelPath))); err == nil {
			return []string{name}, nil
		}
	}

	return nil, nil
}

func normalizeSandboxScope(scope string) string {
	scope = strings.TrimSpace(filepath.ToSlash(scope))
	scope = strings.TrimSuffix(scope, "/")
	return scope
}

func scenarioBaseDir(root string) string {
	return contractPaths.ScenarioBaseDir(root)
}

func scenarioRootPath(root, name string) string {
	return contractPaths.ScenarioRootPath(root, name)
}

func scenarioServicePath(root, name, scenarioPath string) string {
	return contractPaths.ScenarioServicePath(root, name, scenarioPath)
}
