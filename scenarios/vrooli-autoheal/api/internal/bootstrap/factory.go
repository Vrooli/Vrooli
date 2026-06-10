// Package bootstrap provides check creation factory for dependency injection
// [REQ:HEALTH-REGISTRY-001] [REQ:TEST-SEAM-001]
package bootstrap

import (
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	hostchecks "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/host"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/infra"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/system"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/vrooli"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

// CheckFactory defines the interface for creating health checks.
// This abstraction enables testing check registration without creating
// real checks that may depend on system resources.
// [REQ:TEST-SEAM-001]
type CheckFactory interface {
	// CreateInfrastructureChecks creates all infrastructure checks
	CreateInfrastructureChecks(caps *platform.Capabilities) []checks.Check

	// CreateSystemChecks creates all system checks
	CreateSystemChecks() []checks.Check

	// CreateVrooliChecks creates all Vrooli-specific checks
	// Accepts platform capabilities for checks that need them (e.g., watchdog check)
	CreateVrooliChecks(caps *platform.Capabilities) []checks.Check
}

// DefaultCheckFactory is the production implementation of CheckFactory.
// It creates real checks using the check packages.
type DefaultCheckFactory struct {
	networkTarget          string
	dnsDomain              string
	cloudflaredExternalURL string
	criticalScenarios      []string
	nonCriticalScenarios   []string
	resources              []string
}

// NewDefaultCheckFactory creates a factory with standard configuration.
// Uses the default monitoring config from userconfig package.
func NewDefaultCheckFactory() *DefaultCheckFactory {
	monitoring := userconfig.DefaultMonitoring()
	return NewCheckFactoryFromMonitoring(monitoring)
}

// NewCheckFactoryFromMonitoring creates a factory from a MonitoringConfig.
// This allows the factory to be configured dynamically from user settings.
func NewCheckFactoryFromMonitoring(monitoring userconfig.MonitoringConfig) *DefaultCheckFactory {
	return &DefaultCheckFactory{
		networkTarget:          DefaultNetworkTarget,
		dnsDomain:              DefaultDNSDomain,
		cloudflaredExternalURL: "",
		criticalScenarios:      monitoring.GetCriticalScenarios(),
		nonCriticalScenarios:   monitoring.GetNonCriticalScenarios(),
		resources:              monitoring.Resources,
	}
}

// NewCheckFactoryFromConfigManager creates a factory that reads from a config manager.
// This is the preferred method for production use as it respects user configuration.
func NewCheckFactoryFromConfigManager(mgr *userconfig.Manager) *DefaultCheckFactory {
	monitoring := mgr.GetMonitoring()
	factory := NewCheckFactoryFromMonitoring(monitoring)

	cloudflaredCfg := mgr.GetCheck("infra-cloudflared")
	if cloudflaredCfg.Settings.TunnelTestURL != "" {
		factory.cloudflaredExternalURL = cloudflaredCfg.Settings.TunnelTestURL
	}

	return factory
}

// DefaultCheckFactoryOption configures the DefaultCheckFactory
type DefaultCheckFactoryOption func(*DefaultCheckFactory)

// WithNetworkTarget sets the network check target
func WithNetworkTarget(target string) DefaultCheckFactoryOption {
	return func(f *DefaultCheckFactory) {
		f.networkTarget = target
	}
}

// WithDNSDomain sets the DNS check domain
func WithDNSDomain(domain string) DefaultCheckFactoryOption {
	return func(f *DefaultCheckFactory) {
		f.dnsDomain = domain
	}
}

// WithCriticalScenarios sets the critical scenario names
func WithCriticalScenarios(scenarios []string) DefaultCheckFactoryOption {
	return func(f *DefaultCheckFactory) {
		f.criticalScenarios = scenarios
	}
}

// WithNonCriticalScenarios sets the non-critical scenario names
func WithNonCriticalScenarios(scenarios []string) DefaultCheckFactoryOption {
	return func(f *DefaultCheckFactory) {
		f.nonCriticalScenarios = scenarios
	}
}

// WithResources sets the resource names to monitor
func WithResources(resources []string) DefaultCheckFactoryOption {
	return func(f *DefaultCheckFactory) {
		f.resources = resources
	}
}

// NewDefaultCheckFactoryWithOptions creates a factory with custom configuration.
func NewDefaultCheckFactoryWithOptions(opts ...DefaultCheckFactoryOption) *DefaultCheckFactory {
	f := NewDefaultCheckFactory()
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// CreateInfrastructureChecks creates all infrastructure checks
func (f *DefaultCheckFactory) CreateInfrastructureChecks(caps *platform.Capabilities) []checks.Check {
	cloudflaredCheck := infra.NewCloudflaredCheck(caps, infra.WithExternalURL(f.cloudflaredExternalURL))
	return []checks.Check{
		infra.NewNetworkCheck(f.networkTarget),
		infra.NewDNSCheck(f.dnsDomain, caps),
		infra.NewDockerCheck(caps),
		cloudflaredCheck,
		infra.NewRDPCheck(caps),
		infra.NewNTPCheck(caps),
		infra.NewResolvedCheck(caps),
		infra.NewCertificateCheck(),
		infra.NewDisplayManagerCheck(caps),
	}
}

// CreateSystemChecks creates all system checks
func (f *DefaultCheckFactory) CreateSystemChecks() []checks.Check {
	systemChecks := []checks.Check{
		system.NewDiskCheck(),
		system.NewInodeCheck(),
		system.NewSwapCheck(),
		system.NewMemoryCheck(), // RAM usage monitoring
		system.NewZombieCheck(),
		system.NewPortCheck(),
		system.NewClaudeCacheCheck(),
		system.NewGPUCheck(),            // GPU health for AI/ML workloads
		system.NewLoadCheck(),           // System load average monitoring
		system.NewPstoreEvidenceCheck(), // Kernel crash dumps in /sys/fs/pstore
		system.NewBootHistoryCheck(),    // Unclean shutdown detection
		system.NewMCERecentCheck(),      // Recent hardware errors via rasdaemon
		system.NewPMRuntimeHogCheck(),   // Kernel pm_runtime CPU hogs
	}
	collector := hostinventory.NewCachedCollector(hostinventory.NewCollector(hostinventory.CollectorOptions{}), 30*time.Second)
	systemChecks = append(systemChecks, hostchecks.NewChecks(collector)...)
	return systemChecks
}

// CreateVrooliChecks creates all Vrooli-specific checks
func (f *DefaultCheckFactory) CreateVrooliChecks(caps *platform.Capabilities) []checks.Check {
	vrooliChecks := []checks.Check{
		vrooli.NewAPICheck(),
		vrooli.NewWatchdogCheck(caps), // OS watchdog boot recovery check
		vrooli.NewRuntimeSupervisorCheck(),
		vrooli.NewStaleLockCheck(), // Stale port lock detection
		vrooli.NewOrphanCheck(),    // Orphan process detection
	}

	// Resource checks
	for _, name := range f.resources {
		vrooliChecks = append(vrooliChecks, vrooli.NewResourceCheck(name))
	}

	// Critical scenario checks
	for _, name := range f.criticalScenarios {
		vrooliChecks = append(vrooliChecks, vrooli.NewScenarioCheck(name, true))
	}

	// Non-critical scenario checks
	for _, name := range f.nonCriticalScenarios {
		vrooliChecks = append(vrooliChecks, vrooli.NewScenarioCheck(name, false))
	}

	return vrooliChecks
}

// defaultFactory is the global default factory instance.
// Stored as interface type to allow mock injection in tests.
var defaultFactory CheckFactory = NewDefaultCheckFactory()

// GetDefaultFactory returns the global default factory.
// This can be replaced in tests using SetDefaultFactory.
func GetDefaultFactory() CheckFactory {
	return defaultFactory
}

// SetDefaultFactory sets the global default factory.
// This is primarily intended for testing - accepts any CheckFactory implementation.
// Returns the previous factory for restoration.
func SetDefaultFactory(f CheckFactory) CheckFactory {
	prev := defaultFactory
	defaultFactory = f
	return prev
}
