package ports

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

/*
Rule: Ports Configuration
Description: Ensure service.json defines reusable API/UI ports with approved ranges
Reason: Consistent port allocation prevents collisions between scenarios and keeps every listener below the Linux ephemeral floor (32768), which otherwise causes intermittent "port already in use" races on restart.
Category: config
Severity: high
Standard: configuration-v1
Targets: service_json

<test-case id="missing-ports-block" should-fail="true" path=".vrooli/service.json">
  <description>service.json without a ports section</description>
  <input language="json"><![CDATA[
{
  "service": {
    "name": "sample"
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ports</expected-message>
</test-case>

<test-case id="incorrect-api-env-var" should-fail="true" path=".vrooli/service.json">
  <description>API port defined with wrong env var</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "PORT",
      "range": "15000-19999"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>API_PORT</expected-message>
</test-case>

<test-case id="incorrect-api-range" should-fail="true" path=".vrooli/service.json">
  <description>API port range outside the allowed window</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "10000-12000"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>15000-19999</expected-message>
</test-case>

<test-case id="ui-env-var-check" should-fail="true" path=".vrooli/service.json">
  <description>UI port defined without the expected env var</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "PORT",
      "range": "20000-24999"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>UI_PORT</expected-message>
</test-case>

<test-case id="valid-config" should-fail="false" path=".vrooli/service.json">
  <description>Valid ports configuration with canonical api and ui ranges</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "description": "Primary API",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "20000-24999",
      "description": "Optional UI"
    }
  }
}
  ]]></input>
</test-case>

<test-case id="invalid-json" should-fail="true" path=".vrooli/service.json">
  <description>service.json contains malformed JSON</description>
  <input language="json">{</input>
  <expected-violations>1</expected-violations>
  <expected-message>valid JSON</expected-message>
</test-case>

<test-case id="missing-api-entry" should-fail="false" path=".vrooli/service.json">
  <description>ports block can define only the listener ports a scenario needs</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "ui": {
      "env_var": "UI_PORT",
      "range": "20000-24999"
    }
  }
}
  ]]></input>
</test-case>

<test-case id="api-range-missing" should-fail="true" path=".vrooli/service.json">
  <description>API port missing range definition</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ports.api.range</expected-message>
</test-case>

<test-case id="ui-range-mismatch" should-fail="true" path=".vrooli/service.json">
  <description>UI range provided but outside approved window</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "32000-32500"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ports.ui.range</expected-message>
</test-case>

<test-case id="ui-port-fixed-valid" should-fail="false" path=".vrooli/service.json">
  <description>UI port uses fixed assignment with port field in the canonical UI band</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "port": 21234
    }
  }
}
  ]]></input>
</test-case>

<test-case id="ui-port-fixed-ephemeral" should-fail="true" path=".vrooli/service.json">
  <description>UI port pinned inside the Linux ephemeral range</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "port": 36234
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ephemeral range</expected-message>
</test-case>

<test-case id="ui-port-fixed-reserved" should-fail="true" path=".vrooli/service.json">
  <description>UI port uses fixed assignment in reserved range</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "port": 3500
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>reserved range</expected-message>
</test-case>

<test-case id="api-range-invalid-format" should-fail="true" path=".vrooli/service.json">
  <description>API range has invalid format</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-notanumber"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>invalid format</expected-message>
</test-case>

<test-case id="api-range-inverted" should-fail="true" path=".vrooli/service.json">
  <description>API range has start greater than end</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "19999-15000"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>start must be less than end</expected-message>
</test-case>

<test-case id="api-range-overlaps-reserved" should-fail="true" path=".vrooli/service.json">
  <description>API range overlaps with reserved ranges</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "3000-4000"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>reserved range</expected-message>
</test-case>

<test-case id="ui-range-overlaps-ephemeral" should-fail="true" path=".vrooli/service.json">
  <description>UI range overlaps with the Linux ephemeral window (32768-60999)</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "35000-39999"
    }
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ephemeral range</expected-message>
</test-case>

<test-case id="ports-api-not-object" should-fail="true" path=".vrooli/service.json">
  <description>api entry is not an object</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": 8080
  }
}
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>ports.api must be</expected-message>
</test-case>

<test-case id="ignored-non-service-json" should-fail="false" path="config.json">
  <description>Rule should ignore files that are not service.json</description>
  <input language="json"><![CDATA[
{
  "ports": {
    "api": {
      "env_var": "PORT",
      "range": "10000-12000"
    }
  }
}
  ]]></input>
</test-case>
*/

// Canonical scenario port bands. Kept in sync with
// github.com/vrooli/vrooli/internal/portspec (duplicated here because the
// scenario-auditor is a separate Go module and internal/ packages are not
// importable across modules; per project convention we duplicate before
// extracting).
const (
	canonicalAPIRange = "15000-19999"
	canonicalUIRange  = "20000-24999"
	canonicalWSRange  = "25000-29999"
	canonicalMax      = 32767
)

// staticEphemeralRange is the Linux default ephemeral window
// (/proc/sys/net/ipv4/ip_local_port_range). We use it as a static reference
// inside the auditor rather than probing the running OS so audit results are
// reproducible across CI hosts and developer machines.
var staticEphemeralRange = [2]int{32768, 60999}

// CheckServicePortConfiguration validates that service.json declares listener
// ports with explicit env vars and safe fixed/ranged allocations. Port names
// are scenario-defined; only canonical names/env vars receive canonical-band
// enforcement.
func CheckServicePortConfiguration(content []byte, filePath string, scenario ...string) []Violation {
	if !shouldCheckPortsServiceJSON(filePath) {
		return nil
	}

	source := string(content)
	if strings.TrimSpace(source) == "" {
		return []Violation{newPortsViolation(filePath, 1, "service.json is empty; expected a ports configuration block")}
	}

	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		msg := fmt.Sprintf("service.json must be valid JSON to validate ports: %v", err)
		return []Violation{newPortsViolation(filePath, 1, msg)}
	}

	portsRaw, ok := payload["ports"]
	if !ok {
		line := findPortsJSONLine(source, "\"ports\"")
		return []Violation{newPortsViolation(filePath, line, "service.json must define a top-level \"ports\" object")}
	}

	portsMap, ok := portsRaw.(map[string]any)
	if !ok {
		line := findPortsJSONLine(source, "\"ports\"")
		return []Violation{newPortsViolation(filePath, line, "service.json ports must be an object of named port configurations")}
	}

	var violations []Violation
	scenarioRoot := resolveScenarioRootForPortUsage(filePath)

	for name, raw := range portsMap {
		entry, ok := raw.(map[string]any)
		if !ok {
			line := findPortsJSONLine(source, strconv.Quote(name))
			violations = append(violations, newPortsViolation(filePath, line, fmt.Sprintf("ports.%s must be an object with env_var and either range or port fields", name)))
			continue
		}
		validateDeclaredPort(name, entry, source, filePath, &violations)
		validateDeclaredPortEnvUsage(name, entry, source, filePath, scenarioRoot, firstScenarioName(scenario), &violations)
	}

	return dedupePortViolations(violations)
}

func validateDeclaredPort(name string, entry map[string]any, source, filePath string, violations *[]Violation) {
	envVarLine := findPortsJSONLine(source, strconv.Quote(name), "\"env_var\"")
	envVar, ok := entry["env_var"].(string)
	if !ok || strings.TrimSpace(envVar) == "" {
		*violations = append(*violations, newPortsViolation(filePath, envVarLine, fmt.Sprintf("ports.%s.env_var must define the environment variable lifecycle reserves for this listener", name)))
		return
	}

	canonicalRange, canonicalEnv, canonical := canonicalPortContract(name, envVar)
	if canonical && envVar != canonicalEnv {
		*violations = append(*violations, newPortsViolation(filePath, envVarLine, fmt.Sprintf("ports.%s.env_var must be %q for the canonical %s port", name, canonicalEnv, name)))
	}

	_, hasRange := entry["range"]
	_, hasPort := entry["port"]
	if hasRange == hasPort {
		line := findPortsJSONLine(source, strconv.Quote(name))
		*violations = append(*violations, newPortsViolation(filePath, line, fmt.Sprintf("ports.%s.range or ports.%s.port must define exactly one allocation source", name, name)))
		return
	}

	if hasRange {
		validateDeclaredPortRange(name, entry, source, filePath, canonicalRange, canonical, violations)
		return
	}
	validateDeclaredFixedPort(name, entry, source, filePath, canonicalRange, canonical, violations)
}

func validateDeclaredPortRange(name string, entry map[string]any, source, filePath, canonicalRange string, canonical bool, violations *[]Violation) {
	rangeLine := findPortsJSONLine(source, strconv.Quote(name), "\"range\"")
	rangeVal := entry["range"]
	rangeStr, ok := rangeVal.(string)
	if !ok || strings.TrimSpace(rangeStr) == "" {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.%s.range must be a non-empty port range", name)))
		return
	}

	start, end, err := parsePortRange(rangeStr)
	if err != nil {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.%s.range has %s", name, err.Error())))
		return
	}
	if rangeOverlapsEphemeral(start, end) {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.%s.range %s overlaps the Linux ephemeral range %d-%d", name, rangeStr, staticEphemeralRange[0], staticEphemeralRange[1])))
		return
	}
	if end > canonicalMax {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.%s.range %s exceeds canonical maximum %d", name, rangeStr, canonicalMax)))
		return
	}
	if overlaps, reservedName := checkReservedRangeOverlap(start, end); overlaps {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.%s.range overlaps with reserved range: %s", name, reservedName)))
		return
	}
	if canonical && rangeStr != canonicalRange {
		*violations = append(*violations, newPortsViolation(filePath, rangeLine, fmt.Sprintf("ports.%s.range must be %q for the canonical %s port", name, canonicalRange, name)))
	}
}

func validateDeclaredFixedPort(name string, entry map[string]any, source, filePath, canonicalRange string, canonical bool, violations *[]Violation) {
	portLine := findPortsJSONLine(source, strconv.Quote(name), "\"port\"")
	port, ok := parseDeclaredPortNumber(entry["port"])
	if !ok {
		*violations = append(*violations, newPortsViolation(filePath, portLine, fmt.Sprintf("ports.%s.port must be a valid port number", name)))
		return
	}
	if port < 1 || port > 65535 {
		*violations = append(*violations, newPortsViolation(filePath, portLine, fmt.Sprintf("ports.%s.port must be between 1 and 65535", name)))
		return
	}
	if inReserved, reservedName := checkFixedPortInReserved(port); inReserved {
		*violations = append(*violations, newPortsViolation(filePath, portLine, fmt.Sprintf("ports.%s.port %d is in reserved range: %s", name, port, reservedName)))
		return
	}
	if fixedPortInEphemeral(port) {
		*violations = append(*violations, newPortsViolation(filePath, portLine, fmt.Sprintf("ports.%s.port %d sits inside the Linux ephemeral range %d-%d; move it below %d", name, port, staticEphemeralRange[0], staticEphemeralRange[1], canonicalMax+1)))
		return
	}
	if port > canonicalMax {
		*violations = append(*violations, newPortsViolation(filePath, portLine, fmt.Sprintf("ports.%s.port %d exceeds canonical maximum %d", name, port, canonicalMax)))
		return
	}
	if canonical {
		start, end, err := parsePortRange(canonicalRange)
		if err == nil && (port < start || port > end) {
			*violations = append(*violations, newPortsViolation(filePath, portLine, fmt.Sprintf("ports.%s.port %d should be in range %s", name, port, canonicalRange)))
		}
	}
}

func validateDeclaredPortEnvUsage(name string, entry map[string]any, source, filePath, scenarioRoot, scenarioName string, violations *[]Violation) {
	envVar, ok := entry["env_var"].(string)
	envVar = strings.TrimSpace(envVar)
	if !ok || envVar == "" || scenarioRoot == "" {
		return
	}
	evidence := collectPortEnvUsageEvidence(scenarioRoot, filePath, envVar)
	runtimeEvidence := lookupRuntimePortEvidence(scenarioName, name, envVar)
	if evidence.ListenerReferences > 0 {
		return
	}

	line := findPortsJSONLine(source, strconv.Quote(name), "\"env_var\"")
	if evidence.RuntimeReferences > 0 {
		*violations = append(*violations, newPortUsageAmbiguousViolation(filePath, line, withRuntimePortEvidence(fmt.Sprintf("ports.%s.env_var %q is referenced by runtime source but not near recognized listener startup code; this declared listener port should be verified with runtime evidence", name, envVar), runtimeEvidence)))
		return
	}
	if evidence.IgnoredReferences > 0 {
		*violations = append(*violations, newPortUsageViolation(filePath, line, withRuntimePortEvidence(fmt.Sprintf("ports.%s.env_var %q is only referenced in tests, docs, generated output, or scenario metadata; declared ports should correspond to runtime listener code", name, envVar), runtimeEvidence)))
		return
	}
	*violations = append(*violations, newPortUsageViolation(filePath, line, withRuntimePortEvidence(fmt.Sprintf("ports.%s.env_var %q is not referenced by scenario runtime source; this declared listener port may be stale manifest data", name, envVar), runtimeEvidence)))
}

type portEnvUsageEvidence struct {
	ListenerReferences int
	RuntimeReferences  int
	IgnoredReferences  int
}

func collectPortEnvUsageEvidence(scenarioRoot, servicePath, envVar string) portEnvUsageEvidence {
	var evidence portEnvUsageEvidence
	if strings.TrimSpace(scenarioRoot) == "" || strings.TrimSpace(envVar) == "" {
		return evidence
	}

	_ = filepath.WalkDir(scenarioRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if shouldSkipPortUsageDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if filepath.Clean(path) == filepath.Clean(servicePath) || !shouldScanPortUsageFile(path) {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 1024*1024 {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(content), envVar) {
			return nil
		}
		rel, err := filepath.Rel(scenarioRoot, path)
		if err != nil {
			rel = path
		}
		if isRuntimePortUsageFile(filepath.ToSlash(rel)) {
			if containsLikelyListenerUsage(string(content), envVar) {
				evidence.ListenerReferences++
			} else {
				evidence.RuntimeReferences++
			}
		} else {
			evidence.IgnoredReferences++
		}
		return nil
	})
	return evidence
}

func resolveScenarioRootForPortUsage(filePath string) string {
	clean := filepath.Clean(filePath)
	if !filepath.IsAbs(clean) {
		return ""
	}
	if filepath.Base(clean) != "service.json" || filepath.Base(filepath.Dir(clean)) != ".vrooli" {
		return ""
	}
	root := filepath.Dir(filepath.Dir(clean))
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		return root
	}
	return ""
}

func shouldSkipPortUsageDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".vrooli", "node_modules", "dist", "build", "coverage", ".next", ".nuxt", "vendor", "tmp", "logs", ".cache":
		return true
	default:
		return false
	}
}

func shouldScanPortUsageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".sh", ".bash", ".env", ".json", ".yaml", ".yml", ".toml", ".md":
		return true
	default:
		return false
	}
}

func isRuntimePortUsageFile(relativePath string) bool {
	lower := strings.ToLower(relativePath)
	if strings.HasPrefix(lower, "test/") || strings.HasPrefix(lower, "tests/") ||
		strings.HasPrefix(lower, "docs/") || strings.Contains(lower, "/testdata/") ||
		strings.Contains(lower, "/__tests__/") || strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") || strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".md") {
		return false
	}
	return strings.HasPrefix(lower, "api/") || strings.HasPrefix(lower, "cli/") ||
		strings.HasPrefix(lower, "ui/") || strings.HasPrefix(lower, "lib/") ||
		strings.HasPrefix(lower, "scripts/") || lower == "makefile"
}

func containsLikelyListenerUsage(content, envVar string) bool {
	envIndex := strings.Index(content, envVar)
	for envIndex >= 0 {
		if listenerTokenNear(content, envIndex) {
			return true
		}
		nextStart := envIndex + len(envVar)
		next := strings.Index(content[nextStart:], envVar)
		if next < 0 {
			break
		}
		envIndex = nextStart + next
	}
	return false
}

func listenerTokenNear(content string, envIndex int) bool {
	const window = 1600
	start := envIndex - window
	if start < 0 {
		start = 0
	}
	end := envIndex + window
	if end > len(content) {
		end = len(content)
	}
	snippet := content[start:end]
	for _, token := range listenerStartupTokens {
		if strings.Contains(snippet, token) {
			return true
		}
	}
	return false
}

var listenerStartupTokens = []string{
	"ListenAndServe",
	"ListenAndServeTLS",
	"net.Listen",
	"http.Server",
	".Listen(",
	".ListenTLS(",
	".Run(",
	".listen(",
	" listen(",
	".createServer(",
	"http.createServer",
	"https.createServer",
	"server.listen",
	"app.listen",
	"fastify.listen",
	"vite.listen",
}

const runtimePortEvidencePathEnv = "SCENARIO_AUDITOR_RUNTIME_PORT_EVIDENCE_PATH"

type runtimePortEvidenceArtifact struct {
	RegistryClaims []runtimePortEvidenceClaim `json:"registry_claims"`
}

type runtimePortEvidenceClaim struct {
	Scenario                  string `json:"scenario"`
	PortName                  string `json:"port_name"`
	EnvVar                    string `json:"env_var"`
	ListenerStatus            string `json:"listener_status"`
	ConsecutiveListenerMisses int    `json:"consecutive_listener_misses"`
	RecommendationCode        string `json:"recommendation_code"`
	RecommendationConfidence  string `json:"recommendation_confidence"`
	RecommendationRationale   string `json:"recommendation_rationale"`
}

func lookupRuntimePortEvidence(scenarioName, portName, envVar string) *runtimePortEvidenceClaim {
	path := strings.TrimSpace(os.Getenv(runtimePortEvidencePathEnv))
	if path == "" || strings.TrimSpace(envVar) == "" {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var artifact runtimePortEvidenceArtifact
	if err := json.Unmarshal(content, &artifact); err != nil {
		return nil
	}
	for _, claim := range artifact.RegistryClaims {
		if scenarioName != "" && claim.Scenario != "" && claim.Scenario != scenarioName {
			continue
		}
		if claim.EnvVar != envVar {
			continue
		}
		if portName != "" && claim.PortName != "" && claim.PortName != portName {
			continue
		}
		matched := claim
		return &matched
	}
	return nil
}

func withRuntimePortEvidence(message string, evidence *runtimePortEvidenceClaim) string {
	if evidence == nil {
		return message
	}
	if evidence.RecommendationCode == "" && evidence.ListenerStatus == "" {
		return message
	}
	var details []string
	if evidence.RecommendationCode != "" {
		details = append(details, "runtime recommendation "+evidence.RecommendationCode)
	}
	if evidence.RecommendationConfidence != "" {
		details = append(details, "confidence "+evidence.RecommendationConfidence)
	}
	if evidence.ListenerStatus != "" {
		details = append(details, "listener status "+evidence.ListenerStatus)
	}
	if evidence.ConsecutiveListenerMisses > 0 {
		details = append(details, fmt.Sprintf("%d consecutive listener misses", evidence.ConsecutiveListenerMisses))
	}
	if len(details) == 0 {
		return message
	}
	return message + "; historical runtime evidence reports " + strings.Join(details, ", ")
}

func firstScenarioName(scenarios []string) string {
	if len(scenarios) == 0 {
		return ""
	}
	return strings.TrimSpace(scenarios[0])
}

func parseDeclaredPortNumber(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), float64(int(v)) == v
	case int:
		return v, true
	case string:
		if strings.Contains(v, "${") {
			return 0, false
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func canonicalPortContract(name, envVar string) (canonicalRange string, canonicalEnv string, canonical bool) {
	switch {
	case name == "api" || envVar == "API_PORT":
		return canonicalAPIRange, "API_PORT", true
	case name == "ui" || envVar == "UI_PORT":
		return canonicalUIRange, "UI_PORT", true
	case name == "websocket" || name == "ws" || envVar == "WS_PORT":
		return canonicalWSRange, "WS_PORT", true
	default:
		return "", "", false
	}
}

func newPortsViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_service_ports",
		Severity:       "high",
		Title:          "Ports configuration issue",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Define each listener port with an explicit env_var and either a fixed port or range in .vrooli/service.json. Keep canonical API/UI/WS ports in their documented bands. See docs/reference/port-allocation.md.",
		Standard:       "configuration-v1",
	}
}

func newPortUsageViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_service_port_usage",
		Severity:       "medium",
		Title:          "Declared port env var is not used by runtime source",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Remove stale listener port declarations or update runtime source to read the declared env var when binding a listener. Treat this as evidence, not proof, when runtime history disagrees.",
		Standard:       "configuration-v1",
	}
}

func newPortUsageAmbiguousViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_service_port_usage",
		Severity:       "low",
		Title:          "Declared port env var has ambiguous listener usage",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Confirm that runtime source binds a listener using this env var, or correlate this static signal with runtime listener evidence before removing the declaration.",
		Standard:       "configuration-v1",
	}
}

func dedupePortViolations(list []Violation) []Violation {
	if len(list) == 0 {
		return list
	}
	seen := make(map[string]bool)
	var deduped []Violation
	for _, v := range list {
		key := fmt.Sprintf("%s|%s|%d", v.Description, v.FilePath, v.LineNumber)
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, v)
	}
	return deduped
}

func shouldCheckPortsServiceJSON(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	if base == "service.json" {
		return true
	}
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".json") {
		return true
	}
	return false
}

func findPortsJSONLine(content string, tokens ...string) int {
	if len(tokens) == 0 {
		return 1
	}
	lines := strings.Split(content, "\n")
	for idx, line := range lines {
		for _, token := range tokens {
			if strings.Contains(line, token) {
				return idx + 1
			}
		}
	}
	return 1
}

// Reserved port ranges that scenarios must NOT use (from port_registry.sh)
var reservedRanges = map[string][2]int{
	"vrooli_core": {3000, 4100},
	"databases":   {5432, 5499},
	"cache":       {6379, 6399},
	"debug":       {9200, 9299},
	"system":      {1, 1023},
}

// parsePortRange parses a range string like "15000-19999" and returns start, end, error
func parsePortRange(rangeStr string) (int, int, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format: expected 'start-end'")
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid format: start port must be a number")
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid format: end port must be a number")
	}

	if start < 1 || start > 65535 || end < 1 || end > 65535 {
		return 0, 0, fmt.Errorf("ports must be between 1 and 65535")
	}

	if start >= end {
		return 0, 0, fmt.Errorf("start must be less than end")
	}

	return start, end, nil
}

func checkReservedRangeOverlap(start, end int) (bool, string) {
	for name, reserved := range reservedRanges {
		if start <= reserved[1] && end >= reserved[0] {
			return true, fmt.Sprintf("%s (%d-%d)", name, reserved[0], reserved[1])
		}
	}
	return false, ""
}

func checkFixedPortInReserved(port int) (bool, string) {
	for name, reserved := range reservedRanges {
		if port >= reserved[0] && port <= reserved[1] {
			return true, fmt.Sprintf("%s (%d-%d)", name, reserved[0], reserved[1])
		}
	}
	return false, ""
}

func rangeOverlapsEphemeral(start, end int) bool {
	return start <= staticEphemeralRange[1] && end >= staticEphemeralRange[0]
}

func fixedPortInEphemeral(port int) bool {
	return port >= staticEphemeralRange[0] && port <= staticEphemeralRange[1]
}
