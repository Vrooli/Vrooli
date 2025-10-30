package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"app-monitor-api/repository"
)

// =============================================================================
// Repository Helper Methods
// =============================================================================

// requireRepo returns an error if the repository is not available.
// Use this for operations that must have database access.
func (s *AppService) requireRepo() error {
	if s.repo == nil {
		return ErrDatabaseUnavailable
	}
	return nil
}

// hasRepo returns true if the repository is available.
// Use this for operations that can work without database access.
func (s *AppService) hasRepo() bool {
	return s != nil && s.repo != nil
}

// =============================================================================
// Filesystem Utilities
// =============================================================================

// findRepoRoot locates the repository root directory
func findRepoRoot() (string, error) {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root, nil
	}
	if root := os.Getenv("APP_ROOT"); root != "" {
		return root, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if dir == "" || dir == string(filepath.Separator) {
			break
		}
		if _, err := os.Stat(filepath.Join(dir, ".vrooli")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return wd, nil
}

// =============================================================================
// Time and Duration Utilities
// =============================================================================

func humanizeDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := make([]string, 0, 3)
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if hours == 0 && minutes == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	if len(parts) == 0 {
		parts = append(parts, "0s")
	}

	return strings.Join(parts, " ")
}

func formatMillisTimestamp(value int64) string {
	if value <= 0 {
		return ""
	}
	t := time.Unix(0, value*int64(time.Millisecond))
	return t.UTC().Format("15:04:05.000")
}

// =============================================================================
// String Utilities
// =============================================================================

// normalizeIdentifier converts a string to lowercase and trims whitespace for comparison.
// Returns empty string if the input is empty or contains only whitespace.
// This is functionally equivalent to the TypeScript normalizeIdentifier which returns null
// for empty inputs - both empty string (Go) and null (TS) are falsy and must be checked
// before use (e.g., `if value != ""` or `if (!value)`).
func normalizeIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(trimmed)
}

// normalizeLower trims whitespace and converts to lowercase in a single operation.
// Use this for consistent string normalization patterns instead of nested trim+lower calls.
func normalizeLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// isEmptyOrWhitespace returns true if the string is empty or contains only whitespace.
// This provides a cleaner alternative to `strings.TrimSpace(s) == ""` checks.
func isEmptyOrWhitespace(s string) bool {
	return strings.TrimSpace(s) == ""
}

// trimmedStringPtr is a convenience helper that dereferences a string pointer and trims whitespace.
func trimmedStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func trimmedString(value interface{}) string {
	return strings.TrimSpace(anyString(value))
}

func anyString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func truncateString(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit < 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func sanitizeCommandIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	invalid := []string{
		"--", "~", ".", "/",
		"*", "?", "[", "]", "{", "}",
		"$", "&", "|", ";", "<", ">",
		"(", ")", "!", "#", "%", "^",
		"@", "`", "\"", "'", "\\",
	}

	safe := trimmed
	for _, char := range invalid {
		safe = strings.ReplaceAll(safe, char, "")
	}

	fields := strings.Fields(safe)
	if len(fields) == 0 {
		return safe
	}
	return strings.Join(fields, "_")
}

func parseOrEchoTimestamp(value string) string {
	if value == "" {
		return value
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format(time.RFC3339)
	}
	return value
}

// =============================================================================
// Slice Utilities
// =============================================================================

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	var deduped []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		deduped = append(deduped, trimmed)
	}
	return deduped
}

func filterNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return filtered
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sanitizeCaptureClasses(classes []string, maxEntries, maxLength int) []string {
	if len(classes) == 0 {
		return nil
	}
	result := make([]string, 0, min(len(classes), maxEntries))
	for i, class := range classes {
		if i >= maxEntries {
			break
		}
		trimmed := strings.TrimSpace(class)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > maxLength {
			trimmed = trimmed[:maxLength]
		}
		result = append(result, trimmed)
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// Number Utilities
// =============================================================================

func valueOrDefault(value *int, fallback int) int {
	if value != nil {
		return *value
	}
	return fallback
}

func anyNumber(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
	}
	return 0
}

func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// =============================================================================
// Port Resolution Utilities
// =============================================================================

// parsePortFromAny is the canonical port parser that handles all value types.
// It returns an error if the value cannot be parsed as a valid port number.
func parsePortFromAny(value interface{}) (int, error) {
	if value == nil {
		return 0, errors.New("nil port value")
	}

	switch v := value.(type) {
	case int:
		if v > 0 && v <= 65535 {
			return v, nil
		}
		return 0, fmt.Errorf("port %d out of valid range (1-65535)", v)
	case int32:
		if v > 0 && v <= 65535 {
			return int(v), nil
		}
		return 0, fmt.Errorf("port %d out of valid range (1-65535)", v)
	case int64:
		if v > 0 && v <= 65535 {
			return int(v), nil
		}
		return 0, fmt.Errorf("port %d out of valid range (1-65535)", v)
	case float64:
		port := int(v)
		if port > 0 && port <= 65535 && float64(port) == v {
			return port, nil
		}
		return 0, fmt.Errorf("port %v not a valid integer in range (1-65535)", v)
	case float32:
		port := int(v)
		if port > 0 && port <= 65535 && float32(port) == v {
			return port, nil
		}
		return 0, fmt.Errorf("port %v not a valid integer in range (1-65535)", v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			if i > 0 && i <= 65535 {
				return int(i), nil
			}
			return 0, fmt.Errorf("port %d out of valid range (1-65535)", i)
		}
		if f, err := v.Float64(); err == nil {
			port := int(f)
			if port > 0 && port <= 65535 && float64(port) == f {
				return port, nil
			}
		}
		return 0, fmt.Errorf("port value %q cannot be parsed as integer", v.String())
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, errors.New("empty port string")
		}
		port, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("port string %q is not a valid number: %w", trimmed, err)
		}
		if port <= 0 || port > 65535 {
			return 0, fmt.Errorf("port %d out of valid range (1-65535)", port)
		}
		return port, nil
	default:
		return 0, fmt.Errorf("unsupported port value type: %T", value)
	}
}

func convertPortsToMap(entries []scenarioPort) map[string]interface{} {
	if len(entries) == 0 {
		return nil
	}

	ports := make(map[string]interface{}, len(entries))
	for _, entry := range entries {
		key := strings.TrimSpace(entry.Key)
		if key == "" || entry.Port == nil {
			continue
		}

		ports[key] = normalizePortValue(entry.Port)
	}

	if len(ports) == 0 {
		return nil
	}

	return ports
}

func normalizePortValue(value interface{}) interface{} {
	// Try to parse as a valid port number
	if port, err := parsePortFromAny(value); err == nil {
		return port
	}

	// If parsing fails, preserve the original value for better error reporting
	switch v := value.(type) {
	case json.Number:
		// For json.Number, convert to string if we can't parse it
		return v.String()
	case string:
		// For strings, trim whitespace even if invalid
		return strings.TrimSpace(v)
	default:
		// For other types, return as-is
		return v
	}
}

func derivePrimaryPort(ports map[string]interface{}) (string, interface{}) {
	if len(ports) == 0 {
		return "", nil
	}

	if value, ok := ports["UI_PORT"]; ok {
		return "UI_PORT", value
	}
	if value, ok := ports["API_PORT"]; ok {
		return "API_PORT", value
	}

	keys := make([]string, 0, len(ports))
	for key := range ports {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	first := keys[0]
	return first, ports[first]
}

func applyPortConfig(app *repository.App, ports map[string]interface{}) {
	if len(ports) == 0 {
		return
	}

	app.PortMappings = make(map[string]interface{}, len(ports))
	for key, value := range ports {
		app.PortMappings[key] = value
	}

	if app.Config == nil {
		app.Config = make(map[string]interface{})
	}
	app.Config["ports"] = app.PortMappings

	primaryLabel, primaryValue := derivePrimaryPort(app.PortMappings)
	if primaryLabel != "" {
		app.Config["primary_port_label"] = primaryLabel
		app.Config["primary_port"] = fmt.Sprintf("%v", primaryValue)
	}

	if apiPort, ok := app.PortMappings["API_PORT"]; ok {
		app.Config["api_port"] = apiPort
	}
}

func resolvePort(portMappings map[string]interface{}, preferredKeys []string) int {
	if len(portMappings) == 0 {
		return 0
	}

	for _, key := range preferredKeys {
		for label, value := range portMappings {
			if strings.EqualFold(label, key) {
				if port, err := parsePortFromAny(value); err == nil {
					return port
				}
			}
		}
	}

	for _, value := range portMappings {
		if port, err := parsePortFromAny(value); err == nil {
			return port
		}
	}

	return 0
}

func resolveAppPort(app *repository.App, preferredKeys []string) int {
	if app == nil {
		return 0
	}

	if port := resolvePort(app.PortMappings, preferredKeys); port > 0 {
		return port
	}

	if configPorts := extractInterfaceMap(app.Config["ports"]); len(configPorts) > 0 {
		if port := resolvePort(configPorts, preferredKeys); port > 0 {
			return port
		}
	}

	for _, key := range preferredKeys {
		if value, ok := app.Config[key]; ok {
			if port, err := parsePortFromAny(value); err == nil {
				return port
			}
		}
	}

	if port := resolvePort(app.Environment, preferredKeys); port > 0 {
		return port
	}

	return 0
}

func extractInterfaceMap(value interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed
	case map[string]int:
		converted := make(map[string]interface{}, len(typed))
		for k, v := range typed {
			converted[k] = v
		}
		return converted
	case map[string]float64:
		converted := make(map[string]interface{}, len(typed))
		for k, v := range typed {
			converted[k] = v
		}
		return converted
	case map[string]string:
		converted := make(map[string]interface{}, len(typed))
		for k, v := range typed {
			converted[k] = v
		}
		return converted
	default:
		return nil
	}
}

// =============================================================================
// App Data Utilities
// =============================================================================

func anyStringFromMap(data map[string]interface{}, keys ...string) string {
	current := data
	for _, key := range keys {
		if current == nil {
			return ""
		}
		value, ok := current[key]
		if !ok {
			return ""
		}
		if nested, ok := value.(map[string]interface{}); ok {
			current = nested
			continue
		}
		return trimmedString(value)
	}
	return ""
}

// =============================================================================
// URL Building Utilities
// =============================================================================

// buildLocalAPIURL constructs a localhost HTTP URL for internal service communication.
// This centralizes URL construction for easier testing and future scheme changes.
func buildLocalAPIURL(port int, path string) string {
	if port <= 0 {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("http://localhost:%d%s", port, path)
}

// =============================================================================
// CLI Command Execution Utilities
// =============================================================================

// executeVrooliCommand wraps vrooli CLI execution with consistent timeouts and error handling.
// This provides a single point for all vrooli command invocations, making it easier to:
// - Add logging/metrics for CLI calls
// - Mock commands in tests
// - Ensure consistent error messages
// - Change timeout strategies globally
func executeVrooliCommand(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("vrooli %s failed: %w (output: %s)",
			strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}
