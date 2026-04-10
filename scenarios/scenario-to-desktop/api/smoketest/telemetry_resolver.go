package smoketest

import (
	"bufio"
	"encoding/json"
	"path/filepath"
	"strings"
)

// DefaultTelemetryPathResolver implements TelemetryPathResolver.
type DefaultTelemetryPathResolver struct {
	config    Config
	envReader EnvironmentReader
	fs        FileSystem
}

// NewTelemetryPathResolver creates a new telemetry path resolver.
func NewTelemetryPathResolver(config Config, envReader EnvironmentReader, fs FileSystem) *DefaultTelemetryPathResolver {
	return &DefaultTelemetryPathResolver{
		config:    config,
		envReader: envReader,
		fs:        fs,
	}
}

// ExtractFromOutput attempts to extract the telemetry path from smoke test output.
func (r *DefaultTelemetryPathResolver) ExtractFromOutput(output string) string {
	marker := r.config.TelemetryPathMarker
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, marker) {
			idx := strings.Index(line, marker)
			if idx >= 0 {
				return strings.TrimSpace(line[idx+len(marker):])
			}
		}
	}
	return ""
}

// ResolveFromArtifact attempts to resolve the telemetry path based on platform and artifact.
func (r *DefaultTelemetryPathResolver) ResolveFromArtifact(platform, artifactPath, scenarioName string) string {
	appName := r.resolveAppNameFromArtifact(artifactPath, scenarioName)
	if appName == "" {
		return ""
	}
	return r.resolveTelemetryPath(platform, appName)
}

// ReadTelemetryEvents reads telemetry events from the given path.
func (r *DefaultTelemetryPathResolver) ReadTelemetryEvents(path string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = r.config.MaxTelemetryEvents
	}

	file, err := r.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	events := make([]map[string]interface{}, 0, limit)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		if len(events) >= limit {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return events, err
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return events, err
	}
	return events, nil
}

// resolveAppNameFromArtifact extracts the app name from package.json or falls back to scenarioName.
func (r *DefaultTelemetryPathResolver) resolveAppNameFromArtifact(artifactPath, fallback string) string {
	dir := filepath.Dir(artifactPath)
	pkgPath := filepath.Clean(filepath.Join(dir, "..", "package.json"))

	raw, err := r.fs.ReadFile(pkgPath)
	if err != nil {
		return fallback
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fallback
	}

	if name, ok := payload["name"].(string); ok && name != "" {
		return name
	}
	return fallback
}

// resolveTelemetryPath builds the platform-specific telemetry file path.
func (r *DefaultTelemetryPathResolver) resolveTelemetryPath(platform, appName string) string {
	if appName == "" {
		return ""
	}

	switch platform {
	case "win":
		base := r.envReader.GetEnv("APPDATA")
		if base == "" {
			return ""
		}
		return filepath.Join(base, appName, r.config.TelemetryFileName)

	case "mac":
		home, err := r.envReader.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, "Library", "Application Support", appName, r.config.TelemetryFileName)

	default: // linux
		config := r.envReader.GetEnv("XDG_CONFIG_HOME")
		if config == "" {
			home, err := r.envReader.UserHomeDir()
			if err != nil {
				return ""
			}
			config = filepath.Join(home, ".config")
		}
		return filepath.Join(config, appName, r.config.TelemetryFileName)
	}
}
