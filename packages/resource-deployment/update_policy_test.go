package resourcedeployment

import (
	"math"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestStatelessUpdateMetricsRequireBoundedLocalCurrentGauges(t *testing.T) {
	for _, path := range []string{"https://external.invalid/metrics", "//external.invalid/metrics", "/../metrics", "/metrics?", "/metrics#", "/%2Fmetrics", "/${PORT}", "/metrics?x=1"} {
		m := statelessUpdateFixture()
		m.UpdatePolicy.ReadinessMetrics = []ServiceUpdateMetric{{Path: path, Name: "active_connections", Minimum: 1}}
		if err := m.ValidateUpdatePolicy(); err == nil {
			t.Fatalf("escaped metrics path accepted: %q", path)
		}
	}
	for _, minimum := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		m := statelessUpdateFixture()
		m.UpdatePolicy.ReadinessMetrics = []ServiceUpdateMetric{{Path: "/metrics", Name: "active_connections", Minimum: minimum}}
		if err := m.ValidateUpdatePolicy(); err == nil {
			t.Fatal("nonpositive or nonfinite threshold accepted")
		}
	}
	m := statelessUpdateFixture()
	m.UpdatePolicy.ReadinessMetrics = []ServiceUpdateMetric{{Path: "/metrics", Name: "active_connections", Minimum: 1}}
	if err := m.ValidateUpdatePolicy(); err != nil {
		t.Fatal(err)
	}
	m.UpdatePolicy.ReadinessMetrics = append(m.UpdatePolicy.ReadinessMetrics, m.UpdatePolicy.ReadinessMetrics[0])
	if err := m.ValidateUpdatePolicy(); err == nil {
		t.Fatal("duplicate gauge declaration accepted")
	}
}

func TestServiceConfigRequiresOneSafeSource(t *testing.T) {
	for _, config := range []ServiceConfig{
		{Path: "settings.json"},
		{Path: "settings.json", Content: `{"ok":true}`, ContentFile: "config/settings.json"},
		{Path: "settings.json", ContentFile: "../outside.json"},
		{Path: "settings.json", ContentFile: `config\outside.json`},
	} {
		if err := config.Validate(); err == nil {
			t.Errorf("unsafe or ambiguous service config accepted: %+v", config)
		}
	}
	for _, config := range []ServiceConfig{
		{Path: "settings.json", Content: `{"ok":true}`},
		{Path: "settings.json", ContentFile: "config/settings.json"},
	} {
		if err := config.Validate(); err != nil {
			t.Errorf("valid service config source rejected: %+v: %v", config, err)
		}
	}
}

func statelessUpdateFixture() ManagedService {
	return ManagedService{UpdatePolicy: &ServiceUpdatePolicy{Strategy: "rolling-stateless", CandidatePort: "http"}, ProviderPolicy: ProviderPolicy{AllowedModes: []ProviderMode{ProviderManagedPrivate}}, Artifact: ServiceArtifact{Path: "service", Version: "1.0.0", SHA256: strings.Repeat("a", 64)}, Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/service", SHA256: strings.Repeat("a", 64), Archive: "none", Layout: "file"}}}}
}

func TestStatelessUpdatePolicyRequiresRecoverableLaunchContract(t *testing.T) {
	if err := statelessUpdateFixture().ValidateUpdatePolicy(); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*ManagedService){
		"unknown strategy":       func(m *ManagedService) { m.UpdatePolicy.Strategy = "in-place" },
		"unnamed port":           func(m *ManagedService) { m.UpdatePolicy.CandidatePort = "" },
		"bootstrap":              func(m *ManagedService) { m.Bootstrap = &ServiceBootstrap{} },
		"data artifact":          func(m *ManagedService) { m.DataArtifacts = []ManagedServiceDataArtifact{{}} },
		"data group":             func(m *ManagedService) { m.MinimumDataGroups = 1 },
		"environment file":       func(m *ManagedService) { m.EnvironmentFile = "model.env" },
		"account switching":      func(m *ManagedService) { m.RunAsUser = "database" },
		"legacy process limits":  func(m *ManagedService) { m.ProcessLimits = &ProcessLimits{MemoryMaxPercent: 50} },
		"directory artifact":     func(m *ManagedService) { m.Artifact.Layout = "dir"; m.Artifact.EntryPath = "bin/service" },
		"owner home override":    func(m *ManagedService) { m.Environment = map[string]string{"HOME": "/elsewhere"} },
		"owner path override":    func(m *ManagedService) { m.Environment = map[string]string{"PATH": "/elsewhere"} },
		"host tool":              func(m *ManagedService) { m.Artifact.Verification = "HOST-TOOL" },
		"no acquisition":         func(m *ManagedService) { m.Acquisition = nil },
		"OCI source":             func(m *ManagedService) { m.Acquisition.Kind = "oci-image" },
		"adopted target":         func(m *ManagedService) { m.Acquisition.Targets[0].Executable = "external" },
		"target source override": func(m *ManagedService) { m.Acquisition.Targets[0].Kind = "none" },
		"target environment":     func(m *ManagedService) { m.Acquisition.Targets[0].RuntimeEnv = map[string]string{"PORT": "9"} },
		"no provider":            func(m *ManagedService) { m.ProviderPolicy.AllowedModes = nil },
		"shared provider":        func(m *ManagedService) { m.ProviderPolicy.AllowedModes = []ProviderMode{ProviderManagedShared} },
		"data argument":          func(m *ManagedService) { m.Arguments = []string{"--data=${RESOURCE_DATA_DIR}"} },
		"bare data argument":     func(m *ManagedService) { m.Arguments = []string{"--data=$RESOURCE_DATA_DIR"} },
		"shared state override":  func(m *ManagedService) { m.Environment = map[string]string{"RESOURCE_STATE_DIR": "/tmp/shared-state"} },
		"shared config override": func(m *ManagedService) {
			m.Environment = map[string]string{"RESOURCE_CONFIG_DIR": "/tmp/shared-config"}
		},
		"inherited secret argument": func(m *ManagedService) { m.Arguments = []string{"--auth=$PRIVATE_CALLER_TOKEN"} },
		"platform data argument": func(m *ManagedService) {
			m.ArgumentsByPlatform = map[string][]string{"linux": {"${RESOURCE_DATA_DIR}"}}
		},
		"data environment": func(m *ManagedService) { m.Environment = map[string]string{"DATA": "${RESOURCE_DATA_DIR}"} },
		"data config": func(m *ManagedService) {
			m.Config = &ServiceConfig{Path: "service.cfg", Content: "data=${RESOURCE_DATA_DIR}"}
		},
		"file-backed config": func(m *ManagedService) {
			m.Config = &ServiceConfig{Path: "service.cfg", ContentFile: "config/service.cfg"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := statelessUpdateFixture()
			change(&m)
			if err := m.ValidateUpdatePolicy(); err == nil {
				t.Fatal("unsafe rolling-update contract accepted")
			}
		})
	}
}

func TestExclusiveStopStartAllowsFrozenFileBackedConfig(t *testing.T) {
	m := statelessUpdateFixture()
	m.UpdatePolicy.Strategy = UpdateStrategyExclusiveStopStart
	m.Config = &ServiceConfig{Path: "OpenRGB/OpenRGB.json", ContentFile: "config/detectors.json"}
	if err := m.ValidateUpdatePolicy(); err != nil {
		t.Fatalf("exclusive stop/start rejected file-backed configuration: %v", err)
	}
}

func TestExclusiveStopStartRetainsManagedArtifactAndPortRequirements(t *testing.T) {
	for name, change := range map[string]func(*ManagedService){
		"host executable":     func(m *ManagedService) { m.Artifact.Verification = "host-tool" },
		"missing acquisition": func(m *ManagedService) { m.Acquisition = nil },
		"durable data":        func(m *ManagedService) { m.Arguments = append(m.Arguments, "${RESOURCE_DATA_DIR}") },
	} {
		t.Run(name, func(t *testing.T) {
			m := statelessUpdateFixture()
			m.UpdatePolicy.Strategy = UpdateStrategyExclusiveStopStart
			change(&m)
			if err := m.ValidateUpdatePolicy(); err == nil {
				t.Fatal("exclusive policy accepted unsafe launch contract")
			}
		})
	}
}

func TestOrdinaryServicesDoNotAcquireUpdateEligibility(t *testing.T) {
	m := ManagedService{Bootstrap: &ServiceBootstrap{}, DataArtifacts: []ManagedServiceDataArtifact{{}}}
	if err := m.ValidateUpdatePolicy(); err != nil {
		t.Fatalf("ordinary lifecycle policy changed: %v", err)
	}
}

func TestStatelessUpdateTemplatesUseDeclaredInputsAndOwnerPaths(t *testing.T) {
	m := statelessUpdateFixture()
	m.Environment = map[string]string{"MODE": "stable", "SERVICE_CONFIG": "${RESOURCE_CONFIG_DIR}/service.conf"}
	m.Arguments = []string{"--mode=$MODE", "--config=${SERVICE_CONFIG}", "--metrics=127.0.0.1:${RESOURCE_PORT_HTTP}", "--artifact=$RESOURCE_ARTIFACT_DIR"}
	if err := m.ValidateUpdatePolicy(); err != nil {
		t.Fatalf("declared source inputs were mistaken for caller environment: %v", err)
	}
}
