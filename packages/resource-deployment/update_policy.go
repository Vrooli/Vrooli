package resourcedeployment

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

const UpdateStrategyRollingStateless = "rolling-stateless"

var serviceUpdatePortName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// ServiceUpdatePolicy opts a reviewed stateless service into candidate-based
// replacement. It is eligibility, not automatic-update enrollment or authority.
// CandidatePort names the manifest port remapped for the parallel instance.
type ServiceUpdatePolicy struct {
	Strategy         string                `json:"strategy"`
	CandidatePort    string                `json:"candidate_port"`
	ReadinessMetrics []ServiceUpdateMetric `json:"readiness_metrics,omitempty"`
}

// ServiceUpdateMetric requires one current unlabelled Prometheus gauge from the
// exact role's loopback listener. Timestamped, duplicate and non-finite samples
// cannot qualify a launch. This supplements HTTP readiness, never replaces it.
type ServiceUpdateMetric struct {
	Path    string  `json:"path"`
	Name    string  `json:"name"`
	Minimum float64 `json:"minimum"`
}

var serviceUpdateMetricName = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

// ValidateUpdatePolicy rejects launch contracts whose rollback requires data or
// account migration. Resources without a policy retain their ordinary lifecycle.
// The resource owner also validates port/readiness and runtime process ownership.
func (m ManagedService) ValidateUpdatePolicy() error {
	p := m.UpdatePolicy
	if p == nil {
		return nil
	}
	if p.Strategy != UpdateStrategyRollingStateless {
		return fmt.Errorf("unsupported update strategy %q", p.Strategy)
	}
	if !serviceUpdatePortName.MatchString(p.CandidatePort) {
		return fmt.Errorf("candidate_port must name one declared service port")
	}
	if len(p.ReadinessMetrics) > 8 {
		return fmt.Errorf("update readiness metrics exceed the eight-gauge bound")
	}
	seen := map[string]bool{}
	for _, metric := range p.ReadinessMetrics {
		u, err := url.Parse(metric.Path)
		key := metric.Path + "\x00" + metric.Name
		if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || u.Path != metric.Path || u.ForceQuery || !strings.HasPrefix(metric.Path, "/") || strings.HasPrefix(metric.Path, "//") || path.Clean(metric.Path) != metric.Path || strings.ContainsAny(metric.Path, "$\\\x00\r\n") || !serviceUpdateMetricName.MatchString(metric.Name) || math.IsNaN(metric.Minimum) || math.IsInf(metric.Minimum, 0) || metric.Minimum <= 0 || seen[key] {
			return fmt.Errorf("update readiness requires distinct canonical local gauge paths, names and positive finite minimums")
		}
		seen[key] = true
	}
	if layout := strings.ToLower(strings.TrimSpace(m.Artifact.Layout)); layout != "" && layout != "file" {
		return fmt.Errorf("rolling-stateless updates initially require a single native executable file")
	}
	if strings.EqualFold(strings.TrimSpace(m.Artifact.Verification), "host-tool") || m.Acquisition == nil || !strings.EqualFold(strings.TrimSpace(m.Acquisition.Kind), "url") {
		return fmt.Errorf("rolling updates require resource-owned checksum-pinned URL acquisition")
	}
	if m.Bootstrap != nil || len(m.DataArtifacts) > 0 || m.MinimumDataGroups > 0 || m.EnvironmentFile != "" || m.RunAsUser != "" || m.ProcessLimits != nil {
		return fmt.Errorf("rolling-stateless updates do not support bootstrap, data artifacts, environment files, account switching or legacy process limits")
	}
	if len(m.ProviderPolicy.AllowedModes) != 1 || m.ProviderPolicy.AllowedModes[0] != ProviderManagedPrivate {
		return fmt.Errorf("rolling-stateless updates require only the managed-private provider")
	}
	for _, target := range m.Acquisition.Targets {
		if target.Unsupported != "" {
			continue
		}
		if (target.Kind != "" && !strings.EqualFold(strings.TrimSpace(target.Kind), "url")) || target.Executable != "" || len(target.RuntimeEnv) != 0 {
			return fmt.Errorf("rolling-stateless targets require static URL artifacts without adopted executables or runtime environment overrides")
		}
	}
	// A policy cannot make a declared durable-data path safe to share with a
	// parallel process. Logical credential files remain separately materialized.
	var templates []string
	known := map[string]bool{
		"HOME": true, "ROOT": true, "PATH": true,
		"RESOURCE_ROOT": true, "RESOURCE_CONFIG_DIR": true,
		"RESOURCE_CACHE_DIR": true, "RESOURCE_LOGS_DIR": true,
		"RESOURCE_STATE_DIR": true, "RESOURCE_ARTIFACT_DIR": true,
		"VROOLI_MANAGED_SERVICE_ARTIFACT":           true,
		ServicePortEnvironmentName(p.CandidatePort): true,
	}
	templates = append(templates, m.Arguments...)
	for _, args := range m.ArgumentsByPlatform {
		templates = append(templates, args...)
	}
	for key, v := range m.Environment {
		if key == "HOME" || key == "ROOT" || key == "PATH" || strings.HasPrefix(key, "RESOURCE_") || strings.HasPrefix(key, "VROOLI_") {
			return fmt.Errorf("rolling-stateless environment cannot override owner runtime variables")
		}
		known[key] = true
		templates = append(templates, v)
	}
	if m.Config != nil {
		templates = append(templates, m.Config.Content, m.Config.Path)
	}
	for _, v := range templates {
		data, unknown := false, false
		os.Expand(v, func(key string) string {
			data = data || key == "RESOURCE_DATA_DIR"
			unknown = unknown || !known[key]
			return ""
		})
		if data {
			return fmt.Errorf("rolling-stateless launch contract references durable resource data")
		}
		if unknown {
			return fmt.Errorf("rolling-stateless launch contract depends on undeclared caller environment")
		}
	}
	return nil
}

// ServicePortEnvironmentName is the canonical manifest port-to-launch template
// projection used by both validation and managed-service execution.
func ServicePortEnvironmentName(name string) string {
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	return "RESOURCE_PORT_" + strings.ToUpper(name)
}
