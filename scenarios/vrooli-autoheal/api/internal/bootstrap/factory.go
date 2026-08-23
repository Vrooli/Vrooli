// Package bootstrap provides check creation factory for dependency injection
// [REQ:HEALTH-REGISTRY-001] [REQ:TEST-SEAM-001]
package bootstrap

import (
	"context"
	"log"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	hostchecks "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/host"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/infra"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/resourcegpu"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/system"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/vrooli"
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
	externalDNSServer      string
	cloudflaredExternalURL string
	criticalScenarios      []string
	nonCriticalScenarios   []string
	resources              []string

	// diskConfig carries the resolved system-disk configuration so the check
	// is built from the same values the configuration surface reports. A zero
	// value means "use the check's own defaults", which are read from
	// userconfig anyway.
	diskConfig userconfig.CheckConfig
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
		networkTarget:          configuredInfrastructureValue(NetworkTargetEnv),
		dnsDomain:              configuredInfrastructureValue(DNSDomainEnv),
		externalDNSServer:      configuredInfrastructureValue(ExternalDNSServerEnv),
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

	factory.diskConfig = mgr.GetCheck("system-disk")

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
		infra.NewDNSCheck(f.dnsDomain, caps, infra.WithExternalDNSServer(f.externalDNSServer)),
		infra.NewDockerCheck(caps),
		cloudflaredCheck,
		infra.NewRDPCheck(caps, infra.WithRDPDesiredStateProvider(func(ctx context.Context) (infra.RemoteDesktopIntent, error) {
			return infra.ResolveRemoteDesktopIntent(ctx, caps)
		})),
		infra.NewNTPCheck(caps),
		infra.NewResolvedCheck(caps),
		infra.NewCertificateCheck(),
		infra.NewDisplayManagerCheck(caps),
		infra.NewCredentialRecoveryCheck(),
	}
}

// diskCheckOptions translates the resolved system-disk configuration into
// check options. The interval matters most: the registry schedules from
// Check.IntervalSeconds(), so a configured interval that never reaches the
// check is a setting the operator can change with no effect.
func (f *DefaultCheckFactory) diskCheckOptions() []system.DiskCheckOption {
	cfg := f.diskConfig
	var opts []system.DiskCheckOption

	if cfg.IntervalSeconds > 0 {
		opts = append(opts, system.WithDiskInterval(cfg.IntervalSeconds))
	}
	if len(cfg.Thresholds.Partitions) > 0 {
		opts = append(opts, system.WithPartitions(cfg.Thresholds.Partitions))
	}
	if cfg.Thresholds.WarningPercent != nil && cfg.Thresholds.CriticalPercent != nil {
		opts = append(opts, system.WithDiskThresholds(
			int(*cfg.Thresholds.WarningPercent),
			int(*cfg.Thresholds.CriticalPercent),
		))
	}
	return opts
}

// CreateSystemChecks creates all system checks
func (f *DefaultCheckFactory) CreateSystemChecks() []checks.Check {
	systemChecks := []checks.Check{
		system.NewDiskCheck(f.diskCheckOptions()...),
		system.NewInodeCheck(),
		system.NewSwapCheck(),
		system.NewMemoryCheck(), // RAM usage monitoring
		system.NewZombieCheck(),
		system.NewPortCheck(),
		system.NewClaudeCacheCheck(),
		system.NewGPUCheck(),                // GPU health for AI/ML workloads
		system.NewLoadCheck(),               // System load average monitoring
		system.NewPstoreEvidenceCheck(),     // Kernel crash dumps in /sys/fs/pstore
		system.NewPanicEvidenceCheck(),      // Kernel panics captured by kdump
		system.NewStaleServiceBinaryCheck(), // Supervised services running replaced binaries
		system.NewBootHistoryCheck(),        // Unclean shutdown detection
		system.NewMCERecentCheck(),          // Recent hardware errors via rasdaemon
		system.NewPMRuntimeHogCheck(),       // Kernel pm_runtime CPU hogs
		system.NewHostPressureCheck(system.WithReclaimer(system.NewProductionHostPressureReclaimer())), // CPU, memory, swap, process and fork pressure
	}
	collector := hostinventory.NewCachedIntegrityCollector(hostinventory.NewIntegrityCollector(hostinventory.IntegrityCollectorOptions{}), 30*time.Second)
	systemChecks = append(systemChecks, hostchecks.NewChecks(collector)...)
	systemChecks = append(systemChecks, resourcegpu.New())
	return systemChecks
}

// CreateVrooliChecks creates all Vrooli-specific checks
func (f *DefaultCheckFactory) CreateVrooliChecks(caps *platform.Capabilities) []checks.Check {
	vrooliChecks := []checks.Check{
		vrooli.NewWatchdogCheck(caps), // OS watchdog boot recovery check
		vrooli.NewRuntimeSupervisorCheck(),
		vrooli.NewStaleLockCheck(), // Stale port lock detection
		vrooli.NewOrphanCheck(),    // Orphan process detection
	}

	// Resource checks. Names that are not resources in this repository are
	// dropped: a check for a resource that does not exist fails forever and
	// heals nothing, which is how browserless stayed in the monitored list long
	// after it was removed from the fleet.
	monitored, dropped := PruneMissingResources(f.resources)
	for _, name := range dropped {
		log.Printf("vrooli-autoheal: %q is in the monitored resource list but is not a resource in this repository; skipping its checks", name)
	}
	for _, name := range monitored {
		vrooliChecks = append(vrooliChecks, vrooli.NewResourceCheck(name))
	}

	// Accelerator placement checks, for the resources that declare a non-CPU
	// backend. This is the check that answers "is it on the device it asked
	// for", which no other check in the fleet asks.
	// The system-gpu check resolves the same set itself, so it stays
	// self-contained rather than depending on the order this factory builds
	// things in.
	for _, name := range acceleratedResources(monitored) {
		vrooliChecks = append(vrooliChecks, vrooli.NewModeDriftCheck(name))
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
