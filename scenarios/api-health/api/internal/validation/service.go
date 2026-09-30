package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/metrics"
)

// Service validates one target scenario tree at a time.
type Service struct {
	repoRoot     string
	healthProbe  HealthProbe
	probeTimeout time.Duration
}

type Deps struct {
	RepoRoot        string
	HealthProbe     HealthProbe
	ProbeTimeout    time.Duration
	PortURLResolver PortURLResolver
	HTTPClient      *http.Client
}

func New(d Deps) *Service {
	timeout := d.ProbeTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	probe := d.HealthProbe
	if probe == nil {
		probe = NewHTTPHealthProbe(HTTPHealthProbeDeps{
			Client:       d.HTTPClient,
			Resolver:     d.PortURLResolver,
			ProbeTimeout: timeout,
		})
	}
	return &Service{repoRoot: d.RepoRoot, healthProbe: probe, probeTimeout: timeout}
}

// ValidateScenario resolves the target, runs static API readiness checks, and
// adds live health evidence only when the caller requests execution.
func (s *Service) ValidateScenario(ctx context.Context, scenario, explicitPath string, includeExecution bool) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	explicitPath = strings.TrimSpace(explicitPath)
	if scenario == "" && explicitPath == "" {
		return Report{}, fmt.Errorf("scenario or path is required")
	}
	var stage stageRecorder
	if collector := metricsFrom(ctx); collector != nil {
		stage = collector.Stage("resolve-target")
	}
	target, findings := s.resolveTarget(scenario, explicitPath)
	if stage != nil {
		stage.Gauge("findings", float64(len(findings))).End()
	}
	if scenario == "" {
		scenario = target.Scenario
	}
	if scenario == "" {
		scenario = filepath.Base(target.RootPath)
	}
	if includeExecution && target.Resolution == ResolutionResolved && target.APIKind != APIKindAbsent {
		if stage := metricsStage(ctx, "probe-health"); stage != nil {
			probeFindings := s.validateLiveHealth(ctx, &target)
			stage.Gauge("findings", float64(len(probeFindings))).End()
			findings = append(findings, probeFindings...)
		} else {
			findings = append(findings, s.validateLiveHealth(ctx, &target)...)
		}
	}
	return finalize(scenario, target, findings), nil
}

type stageRecorder interface {
	Gauge(name string, value float64) *metrics.Stage
	End() *metrics.Stage
}

func metricsStage(ctx context.Context, name string) stageRecorder {
	if collector := metricsFrom(ctx); collector != nil {
		return collector.Stage(name)
	}
	return nil
}

func (s *Service) resolveTarget(scenario, explicitPath string) (Target, []Finding) {
	target, findings := s.resolveRoot(scenario, explicitPath)
	if len(findings) > 0 {
		return target, findings
	}
	target.Resolution = ResolutionResolved

	if findings := inspectServiceManifest(&target); len(findings) > 0 {
		return target, findings
	}
	if findings := inspectAPIDir(&target); len(findings) > 0 {
		return target, findings
	}

	target.APIKind = classifyAPISurface(target)
	if target.APIKind == APIKindAbsent {
		return target, []Finding{{
			Severity:    SeverityInfo,
			Code:        CodeAPISurfaceAbsent,
			Title:       "API surface absent",
			Location:    target.RootPath,
			Message:     "target scenario has no declared or discoverable API surface",
			Remediation: "declare an API surface only if this scenario is intended to expose one",
		}}
	}
	lifecycleFindings := validateLifecycle(&target)
	httpFindings := validateHTTPSemantics(&target)
	runtimeFindings := validateRuntimeHygiene(&target)
	findings = append(lifecycleFindings, httpFindings...)
	findings = append(findings, runtimeFindings...)
	return target, findings
}

func (s *Service) resolveRoot(scenario, explicitPath string) (Target, []Finding) {
	target := Target{Scenario: scenario}
	if explicitPath != "" {
		target.RootPath = explicitPath
		if target.Scenario == "" {
			target.Scenario = filepath.Base(filepath.Clean(explicitPath))
		}
	} else {
		target.RootPath = filepath.Join(s.repoRoot, "scenarios", scenario)
	}
	if abs, err := filepath.Abs(target.RootPath); err == nil {
		target.RootPath = abs
	}
	info, err := os.Stat(target.RootPath)
	if err != nil {
		target.Resolution = ResolutionMissing
		return unresolvedWithDiagnostic(target, "target scenario could not be resolved", err)
	}
	if !info.IsDir() {
		target.Resolution = ResolutionUnreadable
		return target, []Finding{targetUnresolved(target, "target path is not a scenario directory", nil)}
	}
	return target, nil
}

func inspectServiceManifest(target *Target) []Finding {
	servicePath := filepath.Join(target.RootPath, ".vrooli", "service.json")
	raw, err := os.ReadFile(servicePath)
	switch {
	case err == nil:
		target.ServiceManifestPath = servicePath
		target.ServiceManifestReadable = true
		target.Service = parseServiceManifest(raw)
		return nil
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		target.Resolution = ResolutionUnreadable
		findings := addUnresolvedDiagnostic(target, "target service manifest is unreadable", err)
		return findings
	}
}

func inspectAPIDir(target *Target) []Finding {
	apiDir := filepath.Join(target.RootPath, "api")
	info, err := os.Stat(apiDir)
	switch {
	case err == nil && info.IsDir():
		target.APIDir = apiDir
		target.HasAPIDir = true
		return nil
	case err == nil:
		return nil
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		target.Resolution = ResolutionUnreadable
		return addUnresolvedDiagnostic(target, "target API directory is unreadable", err)
	}
}

func unresolvedWithDiagnostic(target Target, message string, cause error) (Target, []Finding) {
	target.Diagnostics = append(target.Diagnostics, cause.Error())
	return target, []Finding{targetUnresolved(target, message, cause)}
}

func addUnresolvedDiagnostic(target *Target, message string, cause error) []Finding {
	target.Diagnostics = append(target.Diagnostics, cause.Error())
	return []Finding{targetUnresolved(*target, message, cause)}
}

func targetUnresolved(target Target, message string, cause error) Finding {
	if cause != nil {
		message = fmt.Sprintf("%s: %v", message, cause)
	}
	return Finding{
		Severity:    SeverityError,
		Code:        CodeTargetUnresolved,
		Title:       "Target unresolved",
		Location:    target.RootPath,
		Message:     message,
		Remediation: "pass a readable scenario name or explicit scenario path",
	}
}

func classifyAPISurface(target Target) APIKind {
	if target.HasAPIDir {
		return APIKindGo
	}
	if target.Service.PortsAPI || target.Service.HealthAPIPath != "" {
		return APIKindDeclared
	}
	return APIKindAbsent
}

func parseServiceManifest(raw []byte) ServiceManifest {
	var doc struct {
		Ports     map[string]json.RawMessage `json:"ports"`
		Lifecycle struct {
			Health struct {
				Endpoints map[string]string `json:"endpoints"`
				Checks    []struct {
					Name   string `json:"name"`
					Type   string `json:"type"`
					Target string `json:"target"`
				} `json:"checks"`
			} `json:"health"`
		} `json:"lifecycle"`
	}
	var out ServiceManifest
	if err := json.Unmarshal(raw, &doc); err != nil {
		out.ParseError = err.Error()
		return out
	}
	_, out.PortsAPI = doc.Ports["api"]
	out.HealthAPIPath = strings.TrimSpace(doc.Lifecycle.Health.Endpoints["api"])
	for _, check := range doc.Lifecycle.Health.Checks {
		if strings.EqualFold(strings.TrimSpace(check.Type), "http") &&
			strings.Contains(check.Target, "${API_PORT}") {
			out.HealthAPICheck = true
			out.HealthAPICheckURL = strings.TrimSpace(check.Target)
			break
		}
	}
	return out
}
