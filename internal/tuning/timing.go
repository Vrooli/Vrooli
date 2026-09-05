package tuning

import "time"

// Control-plane timing levers are named for the work they bound. The values
// are intentionally unchanged from the former value-named defaults.
const (
	// HostFactsRetryInterval bounds the retry delay between host-facts probes.
	defaultHostFactsRetryInterval = 10 * time.Millisecond
	// FastPersistenceRetryInterval bounds short persistence transaction retries.
	defaultFastPersistenceRetryInterval = 25 * time.Millisecond
	// BackgroundLaunchPollInterval bounds polling while a background launch settles.
	defaultBackgroundLaunchPollInterval = 50 * time.Millisecond
	// LifecyclePollInterval bounds lifecycle state polling.
	defaultLifecyclePollInterval = 100 * time.Millisecond
	// MaintenanceSettleDelay gives maintenance observations time to settle.
	defaultMaintenanceSettleDelay = 150 * time.Millisecond
	// SetupFilesystemSettleDelay gives setup filesystem changes time to settle.
	defaultSetupFilesystemSettleDelay = 200 * time.Millisecond
	// FastHealthPollInterval bounds fast health and process polling.
	defaultFastHealthPollInterval = 250 * time.Millisecond
	// HealthProbeInterval bounds ordinary health probe polling.
	defaultHealthProbeInterval = 500 * time.Millisecond
	// HostPresentationProbeTimeout bounds host presentation discovery.
	defaultHostPresentationProbeTimeout   = 750 * time.Millisecond
	defaultLifecycleHealthPollInterval    = 1 * time.Second
	defaultSetupProgressPollInterval      = 1 * time.Second
	defaultCredentialStoreProbeTimeout    = 1 * time.Second
	defaultLifecycleTransitionTimeout     = 2 * time.Second
	defaultLifecyclePollMaxInterval       = 2 * time.Second
	defaultEphemeralPortProbeTimeout      = 2 * time.Second
	defaultSetupHTTPProbeTimeout          = 2 * time.Second
	defaultPrivilegeBrokerUnlockTimeout   = 2 * time.Second
	defaultHostPresentationCommandTimeout = 2 * time.Second
	defaultManagedServiceForceStopTimeout = 2 * time.Second
	// HealthCheckTimeout bounds a health-check request.
	defaultHealthCheckTimeout         = 3 * time.Second
	defaultListenerEnrichmentTimeout  = 3 * time.Second
	defaultCredentialUnitProbeTimeout = 3 * time.Second
	// ServiceHealthTimeout bounds service health and reachability checks.
	defaultServiceHealthTimeout                = 5 * time.Second
	defaultFreshnessCheckBudget                = 5 * time.Second
	defaultRemoteDesktopProbeTimeout           = 5 * time.Second
	defaultScenarioRequirementsSnapshotTimeout = 5 * time.Second
	// IntegrityCollectionTimeout bounds integrity inventory collection.
	defaultIntegrityCollectionTimeout = 8 * time.Second
	// ControlPlaneClientTimeout bounds ordinary control-plane HTTP calls.
	defaultControlPlaneClientTimeout = 10 * time.Second
	// CredentialServiceTimeout bounds credential-provider operations.
	defaultCredentialServiceTimeout         = 15 * time.Second
	defaultDependencyBestEffortStartTimeout = 15 * time.Second
	defaultSecretToolTimeout                = 15 * time.Second
	// ReloadFallbackGracePeriod bounds credential reload fallback handling.
	defaultReloadFallbackGracePeriod        = 20 * time.Second
	defaultLifecycleOperationTimeout        = 30 * time.Second
	defaultHostRequirementCommandTimeout    = 30 * time.Second
	defaultScenarioHeartbeatTTL             = 30 * time.Second
	defaultCapacityHeartbeatTTL             = 30 * time.Second
	defaultSetupOperationTimeout            = 30 * time.Second
	defaultSetupProgressObservationInterval = 30 * time.Second
	defaultResourceControlTimeout           = 30 * time.Second
	defaultPlatformSupportRequestTimeout    = 30 * time.Second
	defaultCapacityDegradeDebounce          = 30 * time.Second
	defaultCapabilityRequestTimeout         = 30 * time.Second
	defaultEagerScenarioWaitWindow          = 30 * time.Second
	defaultStructureProviderBudget          = 30 * time.Second
	defaultHostInventoryTTL                 = 30 * time.Second
	// SupervisorHealthInterval controls supervisor health observation cadence.
	defaultSupervisorHealthInterval = 45 * time.Second
	// ReachabilityTimeout bounds a service reachability probe.
	defaultReachabilityTimeout = 60 * time.Second
	// CredentialRepairTimeout bounds credential repair attempts.
	defaultCredentialRepairTimeout           = 90 * time.Second
	defaultTidinessProviderBudget            = 2 * time.Minute
	defaultLifecycleExtendedOperationTimeout = 2 * time.Minute
	defaultSupervisorRecoveryQuietPeriod     = 2 * time.Minute
	defaultCredentialReloadFallbackDelay     = 2 * time.Minute
	defaultHostGPUInventoryTTL               = 2 * time.Minute
	defaultDockerRuntimeOperationTimeout     = 2 * time.Minute
	defaultResourceControlExtendedTimeout    = 2 * time.Minute
	defaultSetupExtendedOperationTimeout     = 2 * time.Minute
	defaultPrivilegeBrokerOperationTimeout   = 2 * time.Minute
	defaultStructureProviderExtendedBudget   = 2 * time.Minute
	// ProviderBudget bounds a provider's work budget.
	defaultProviderBudget                = 3 * time.Minute
	defaultSupervisorRecoveryCooldown    = 5 * time.Minute
	defaultEmergencyWatchdogInterval     = 5 * time.Minute
	defaultVaultBootstrapLease           = 5 * time.Minute
	defaultScenarioReservedClaimTTL      = 5 * time.Minute
	defaultSetupProgressStaleThreshold   = 5 * time.Minute
	defaultHostPlatformInventoryTTL      = 5 * time.Minute
	defaultHostWorkloadInventoryTTL      = 5 * time.Minute
	defaultCapacityObservedPeakHalflife  = 10 * time.Minute
	defaultScenarioWaitTimeout           = 10 * time.Minute
	defaultCredentialEscrowRetention     = 10 * time.Minute
	defaultResourceCommandTimeout        = 10 * time.Minute
	defaultCompanionCrashWindow          = 10 * time.Minute
	defaultCompanionHeartbeatInterval    = 60 * time.Second
	defaultCompanionCapacitySyncInterval = 15 * time.Second
	defaultResourceHTTPTimeout           = 60 * time.Second
	defaultResourceShortHTTPTimeout      = 10 * time.Second
	defaultResourceMediumHTTPTimeout     = 30 * time.Second
	defaultResourceLongHTTPTimeout       = 15 * time.Minute
	defaultActivityDebounce              = 5 * time.Second
	defaultActivityReadHeaderTimeout     = 15 * time.Second
	defaultProgressDisplayResolution     = 100 * time.Millisecond
	defaultAgentInstallDownloadTimeout   = 10 * time.Minute
	defaultCompanionLogMaxBytes          = int64(1 << 20)
	// CopyRetentionWindow bounds credential-copy retention behavior.
	defaultCopyRetentionWindow = 15 * time.Minute
	// RepairDeadline bounds a repair request deadline.
	defaultRepairDeadline = 30 * time.Minute
	// DailyRetentionWindow bounds daily retention and cache validity.
	defaultDailyRetentionWindow = 24 * time.Hour
	// TerminalClaimRetention bounds terminal runtime claim retention.
	defaultTerminalClaimRetention = 14 * 24 * time.Hour
	// ArtifactRetentionWindow bounds managed artifact attestation retention.
	defaultArtifactRetentionWindow = 30 * 24 * time.Hour
	// AttestationValidityWindow bounds the maximum attestation lifetime.
	defaultAttestationValidityWindow = 31 * 24 * time.Hour
)

// Timing levers are accessors so environment overrides are resolved once per
// process while their compiled defaults remain immutable.
// HostFactsRetryInterval controls host facts retry interval.
func HostFactsRetryInterval() time.Duration {
	return Duration("HostFactsRetryInterval", defaultHostFactsRetryInterval)
}

// FastPersistenceRetryInterval controls fast persistence retry interval.
func FastPersistenceRetryInterval() time.Duration {
	return Duration("FastPersistenceRetryInterval", defaultFastPersistenceRetryInterval)
}

// BackgroundLaunchPollInterval controls background launch poll interval.
func BackgroundLaunchPollInterval() time.Duration {
	return Duration("BackgroundLaunchPollInterval", defaultBackgroundLaunchPollInterval)
}

// LifecyclePollInterval controls lifecycle poll interval.
func LifecyclePollInterval() time.Duration {
	return Duration("LifecyclePollInterval", defaultLifecyclePollInterval)
}

// MaintenanceSettleDelay controls maintenance settle delay.
func MaintenanceSettleDelay() time.Duration {
	return Duration("MaintenanceSettleDelay", defaultMaintenanceSettleDelay)
}

// SetupFilesystemSettleDelay controls setup filesystem settle delay.
func SetupFilesystemSettleDelay() time.Duration {
	return Duration("SetupFilesystemSettleDelay", defaultSetupFilesystemSettleDelay)
}

// FastHealthPollInterval controls fast health poll interval.
func FastHealthPollInterval() time.Duration {
	return Duration("FastHealthPollInterval", defaultFastHealthPollInterval)
}

// HealthProbeInterval controls health probe interval.
func HealthProbeInterval() time.Duration {
	return Duration("HealthProbeInterval", defaultHealthProbeInterval)
}

// HostPresentationProbeTimeout controls host presentation probe timeout.
func HostPresentationProbeTimeout() time.Duration {
	return Duration("HostPresentationProbeTimeout", defaultHostPresentationProbeTimeout)
}

// ScenarioRequirementsSnapshotTimeout controls scenario requirements snapshot timeout.
func ScenarioRequirementsSnapshotTimeout() time.Duration {
	return Duration("ScenarioRequirementsSnapshotTimeout", defaultScenarioRequirementsSnapshotTimeout)
}

// LifecycleHealthPollInterval controls lifecycle health poll interval.
func LifecycleHealthPollInterval() time.Duration {
	return Duration("LifecycleHealthPollInterval", defaultLifecycleHealthPollInterval)
}

// SetupProgressPollInterval controls setup progress poll interval.
func SetupProgressPollInterval() time.Duration {
	return Duration("SetupProgressPollInterval", defaultSetupProgressPollInterval)
}

// CredentialStoreProbeTimeout controls credential store probe timeout.
func CredentialStoreProbeTimeout() time.Duration {
	return Duration("CredentialStoreProbeTimeout", defaultCredentialStoreProbeTimeout)
}

// LifecycleTransitionTimeout controls lifecycle transition timeout.
func LifecycleTransitionTimeout() time.Duration {
	return Duration("LifecycleTransitionTimeout", defaultLifecycleTransitionTimeout)
}

// LifecyclePollMaxInterval controls lifecycle poll max interval.
func LifecyclePollMaxInterval() time.Duration {
	return Duration("LifecyclePollMaxInterval", defaultLifecyclePollMaxInterval)
}

// EphemeralPortProbeTimeout controls ephemeral port probe timeout.
func EphemeralPortProbeTimeout() time.Duration {
	return Duration("EphemeralPortProbeTimeout", defaultEphemeralPortProbeTimeout)
}

// SetupHTTPProbeTimeout controls setup http probe timeout.
func SetupHTTPProbeTimeout() time.Duration {
	return Duration("SetupHTTPProbeTimeout", defaultSetupHTTPProbeTimeout)
}

// PrivilegeBrokerUnlockTimeout controls privilege broker unlock timeout.
func PrivilegeBrokerUnlockTimeout() time.Duration {
	return Duration("PrivilegeBrokerUnlockTimeout", defaultPrivilegeBrokerUnlockTimeout)
}

// HostPresentationCommandTimeout controls host presentation command timeout.
func HostPresentationCommandTimeout() time.Duration {
	return Duration("HostPresentationCommandTimeout", defaultHostPresentationCommandTimeout)
}

// ManagedServiceForceStopTimeout controls managed service force stop timeout.
func ManagedServiceForceStopTimeout() time.Duration {
	return Duration("ManagedServiceForceStopTimeout", defaultManagedServiceForceStopTimeout)
}

// HealthCheckTimeout controls health check timeout.
func HealthCheckTimeout() time.Duration {
	return Duration("HealthCheckTimeout", defaultHealthCheckTimeout)
}

// ListenerEnrichmentTimeout controls listener enrichment timeout.
func ListenerEnrichmentTimeout() time.Duration {
	return Duration("ListenerEnrichmentTimeout", defaultListenerEnrichmentTimeout)
}

// CredentialUnitProbeTimeout controls credential unit probe timeout.
func CredentialUnitProbeTimeout() time.Duration {
	return Duration("CredentialUnitProbeTimeout", defaultCredentialUnitProbeTimeout)
}

// ServiceHealthTimeout controls service health timeout.
func ServiceHealthTimeout() time.Duration {
	return Duration("ServiceHealthTimeout", defaultServiceHealthTimeout)
}

// FreshnessCheckBudget controls freshness check budget.
func FreshnessCheckBudget() time.Duration {
	return Duration("FreshnessCheckBudget", defaultFreshnessCheckBudget)
}

// RemoteDesktopProbeTimeout controls remote desktop probe timeout.
func RemoteDesktopProbeTimeout() time.Duration {
	return Duration("RemoteDesktopProbeTimeout", defaultRemoteDesktopProbeTimeout)
}

// IntegrityCollectionTimeout controls integrity collection timeout.
func IntegrityCollectionTimeout() time.Duration {
	return Duration("IntegrityCollectionTimeout", defaultIntegrityCollectionTimeout)
}

// ControlPlaneClientTimeout controls control plane client timeout.
func ControlPlaneClientTimeout() time.Duration {
	return Duration("ControlPlaneClientTimeout", defaultControlPlaneClientTimeout)
}

// CredentialServiceTimeout controls credential service timeout.
func CredentialServiceTimeout() time.Duration {
	return Duration("CredentialServiceTimeout", defaultCredentialServiceTimeout)
}

// DependencyBestEffortStartTimeout controls dependency best effort start timeout.
func DependencyBestEffortStartTimeout() time.Duration {
	return Duration("DependencyBestEffortStartTimeout", defaultDependencyBestEffortStartTimeout)
}

// SecretToolTimeout controls secret tool timeout.
func SecretToolTimeout() time.Duration {
	return Duration("SecretToolTimeout", defaultSecretToolTimeout)
}

// ReloadFallbackGracePeriod controls reload fallback grace period.
func ReloadFallbackGracePeriod() time.Duration {
	return Duration("ReloadFallbackGracePeriod", defaultReloadFallbackGracePeriod)
}

// LifecycleOperationTimeout controls lifecycle operation timeout.
func LifecycleOperationTimeout() time.Duration {
	return Duration("LifecycleOperationTimeout", defaultLifecycleOperationTimeout)
}

// HostRequirementCommandTimeout controls host requirement command timeout.
func HostRequirementCommandTimeout() time.Duration {
	return Duration("HostRequirementCommandTimeout", defaultHostRequirementCommandTimeout)
}

// ScenarioHeartbeatTTL controls scenario heartbeat ttl.
func ScenarioHeartbeatTTL() time.Duration {
	return Duration("ScenarioHeartbeatTTL", defaultScenarioHeartbeatTTL)
}

// CapacityHeartbeatTTL controls capacity heartbeat ttl.
func CapacityHeartbeatTTL() time.Duration {
	return Duration("CapacityHeartbeatTTL", defaultCapacityHeartbeatTTL)
}

// SetupOperationTimeout controls setup operation timeout.
func SetupOperationTimeout() time.Duration {
	return Duration("SetupOperationTimeout", defaultSetupOperationTimeout)
}

// SetupProgressObservationInterval controls setup progress observation interval.
func SetupProgressObservationInterval() time.Duration {
	return Duration("SetupProgressObservationInterval", defaultSetupProgressObservationInterval)
}

// ResourceControlTimeout controls resource control timeout.
func ResourceControlTimeout() time.Duration {
	return Duration("ResourceControlTimeout", defaultResourceControlTimeout)
}

// PlatformSupportRequestTimeout controls platform support request timeout.
func PlatformSupportRequestTimeout() time.Duration {
	return Duration("PlatformSupportRequestTimeout", defaultPlatformSupportRequestTimeout)
}

// CapacityDegradeDebounce controls capacity degrade debounce.
func CapacityDegradeDebounce() time.Duration {
	return Duration("CapacityDegradeDebounce", defaultCapacityDegradeDebounce)
}

// CapabilityRequestTimeout controls capability request timeout.
func CapabilityRequestTimeout() time.Duration {
	return Duration("CapabilityRequestTimeout", defaultCapabilityRequestTimeout)
}

// EagerScenarioWaitWindow controls eager scenario wait window.
func EagerScenarioWaitWindow() time.Duration {
	return Duration("EagerScenarioWaitWindow", defaultEagerScenarioWaitWindow)
}

// StructureProviderBudget controls structure provider budget.
func StructureProviderBudget() time.Duration {
	return Duration("StructureProviderBudget", defaultStructureProviderBudget)
}

// HostInventoryTTL controls host inventory ttl.
func HostInventoryTTL() time.Duration {
	return Duration("HostInventoryTTL", defaultHostInventoryTTL)
}

// SupervisorHealthInterval controls supervisor health interval.
func SupervisorHealthInterval() time.Duration {
	return Duration("SupervisorHealthInterval", defaultSupervisorHealthInterval)
}

// ReachabilityTimeout controls reachability timeout.
func ReachabilityTimeout() time.Duration {
	return Duration("ReachabilityTimeout", defaultReachabilityTimeout)
}

// CredentialRepairTimeout controls credential repair timeout.
func CredentialRepairTimeout() time.Duration {
	return Duration("CredentialRepairTimeout", defaultCredentialRepairTimeout)
}

// TidinessProviderBudget controls tidiness provider budget.
func TidinessProviderBudget() time.Duration {
	return Duration("TidinessProviderBudget", defaultTidinessProviderBudget)
}

// LifecycleExtendedOperationTimeout controls lifecycle extended operation timeout.
func LifecycleExtendedOperationTimeout() time.Duration {
	return Duration("LifecycleExtendedOperationTimeout", defaultLifecycleExtendedOperationTimeout)
}

// SupervisorRecoveryQuietPeriod controls supervisor recovery quiet period.
func SupervisorRecoveryQuietPeriod() time.Duration {
	return Duration("SupervisorRecoveryQuietPeriod", defaultSupervisorRecoveryQuietPeriod)
}

// CredentialReloadFallbackDelay controls credential reload fallback delay.
func CredentialReloadFallbackDelay() time.Duration {
	return Duration("CredentialReloadFallbackDelay", defaultCredentialReloadFallbackDelay)
}

// HostGPUInventoryTTL controls host gpu inventory ttl.
func HostGPUInventoryTTL() time.Duration {
	return Duration("HostGPUInventoryTTL", defaultHostGPUInventoryTTL)
}

// DockerRuntimeOperationTimeout controls docker runtime operation timeout.
func DockerRuntimeOperationTimeout() time.Duration {
	return Duration("DockerRuntimeOperationTimeout", defaultDockerRuntimeOperationTimeout)
}

// ResourceControlExtendedTimeout controls resource control extended timeout.
func ResourceControlExtendedTimeout() time.Duration {
	return Duration("ResourceControlExtendedTimeout", defaultResourceControlExtendedTimeout)
}

// SetupExtendedOperationTimeout controls setup extended operation timeout.
func SetupExtendedOperationTimeout() time.Duration {
	return Duration("SetupExtendedOperationTimeout", defaultSetupExtendedOperationTimeout)
}

// PrivilegeBrokerOperationTimeout controls privilege broker operation timeout.
func PrivilegeBrokerOperationTimeout() time.Duration {
	return Duration("PrivilegeBrokerOperationTimeout", defaultPrivilegeBrokerOperationTimeout)
}

// StructureProviderExtendedBudget controls structure provider extended budget.
func StructureProviderExtendedBudget() time.Duration {
	return Duration("StructureProviderExtendedBudget", defaultStructureProviderExtendedBudget)
}

// ProviderBudget controls provider budget.
func ProviderBudget() time.Duration { return Duration("ProviderBudget", defaultProviderBudget) }

// SupervisorRecoveryCooldown controls supervisor recovery cooldown.
func SupervisorRecoveryCooldown() time.Duration {
	return Duration("SupervisorRecoveryCooldown", defaultSupervisorRecoveryCooldown)
}

// EmergencyWatchdogInterval controls emergency watchdog interval.
func EmergencyWatchdogInterval() time.Duration {
	return Duration("EmergencyWatchdogInterval", defaultEmergencyWatchdogInterval)
}

// VaultBootstrapLease controls vault bootstrap lease.
func VaultBootstrapLease() time.Duration {
	return Duration("VaultBootstrapLease", defaultVaultBootstrapLease)
}

// ScenarioReservedClaimTTL controls scenario reserved claim ttl.
func ScenarioReservedClaimTTL() time.Duration {
	return Duration("ScenarioReservedClaimTTL", defaultScenarioReservedClaimTTL)
}

// SetupProgressStaleThreshold controls setup progress stale threshold.
func SetupProgressStaleThreshold() time.Duration {
	return Duration("SetupProgressStaleThreshold", defaultSetupProgressStaleThreshold)
}

// HostPlatformInventoryTTL controls host platform inventory ttl.
func HostPlatformInventoryTTL() time.Duration {
	return Duration("HostPlatformInventoryTTL", defaultHostPlatformInventoryTTL)
}

// HostWorkloadInventoryTTL controls host workload inventory ttl.
func HostWorkloadInventoryTTL() time.Duration {
	return Duration("HostWorkloadInventoryTTL", defaultHostWorkloadInventoryTTL)
}

// CapacityObservedPeakHalflife controls capacity observed peak halflife.
func CapacityObservedPeakHalflife() time.Duration {
	return Duration("CapacityObservedPeakHalflife", defaultCapacityObservedPeakHalflife)
}

// ScenarioWaitTimeout controls scenario wait timeout.
func ScenarioWaitTimeout() time.Duration {
	return Duration("ScenarioWaitTimeout", defaultScenarioWaitTimeout)
}

// CredentialEscrowRetention controls credential escrow retention.
func CredentialEscrowRetention() time.Duration {
	return Duration("CredentialEscrowRetention", defaultCredentialEscrowRetention)
}

// ResourceCommandTimeout controls resource command timeout.
func ResourceCommandTimeout() time.Duration {
	return Duration("ResourceCommandTimeout", defaultResourceCommandTimeout)
}

// CompanionCrashWindow controls companion crash window.
func CompanionCrashWindow() time.Duration {
	return Duration("CompanionCrashWindow", defaultCompanionCrashWindow)
}

// CompanionHeartbeatInterval controls companion heartbeat interval.
func CompanionHeartbeatInterval() time.Duration {
	return Duration("CompanionHeartbeatInterval", defaultCompanionHeartbeatInterval)
}

// CompanionCapacitySyncInterval controls companion capacity sync interval.
func CompanionCapacitySyncInterval() time.Duration {
	return Duration("CompanionCapacitySyncInterval", defaultCompanionCapacitySyncInterval)
}

// ResourceHTTPTimeout controls resource http timeout.
func ResourceHTTPTimeout() time.Duration {
	return Duration("ResourceHTTPTimeout", defaultResourceHTTPTimeout)
}

// ResourceShortHTTPTimeout controls resource short http timeout.
func ResourceShortHTTPTimeout() time.Duration {
	return Duration("ResourceShortHTTPTimeout", defaultResourceShortHTTPTimeout)
}

// ResourceMediumHTTPTimeout controls resource medium http timeout.
func ResourceMediumHTTPTimeout() time.Duration {
	return Duration("ResourceMediumHTTPTimeout", defaultResourceMediumHTTPTimeout)
}

// ResourceLongHTTPTimeout controls resource long http timeout.
func ResourceLongHTTPTimeout() time.Duration {
	return Duration("ResourceLongHTTPTimeout", defaultResourceLongHTTPTimeout)
}

// ActivityDebounce controls activity debounce.
func ActivityDebounce() time.Duration { return Duration("ActivityDebounce", defaultActivityDebounce) }

// ActivityReadHeaderTimeout controls activity read header timeout.
func ActivityReadHeaderTimeout() time.Duration {
	return Duration("ActivityReadHeaderTimeout", defaultActivityReadHeaderTimeout)
}

// ProgressDisplayResolution controls progress display resolution.
func ProgressDisplayResolution() time.Duration {
	return Duration("ProgressDisplayResolution", defaultProgressDisplayResolution)
}

func CompanionLogMaxBytes() int64 {
	return defaultCompanionLogMaxBytes
}

// AgentInstallDownloadTimeout controls agent install download timeout.
func AgentInstallDownloadTimeout() time.Duration {
	return Duration("AgentInstallDownloadTimeout", defaultAgentInstallDownloadTimeout)
}

// CopyRetentionWindow controls copy retention window.
func CopyRetentionWindow() time.Duration {
	return Duration("CopyRetentionWindow", defaultCopyRetentionWindow)
}

// RepairDeadline controls repair deadline.
func RepairDeadline() time.Duration { return Duration("RepairDeadline", defaultRepairDeadline) }

// DailyRetentionWindow controls daily retention window.
func DailyRetentionWindow() time.Duration {
	return Duration("DailyRetentionWindow", defaultDailyRetentionWindow)
}

// TerminalClaimRetention controls terminal claim retention.
func TerminalClaimRetention() time.Duration {
	return Duration("TerminalClaimRetention", defaultTerminalClaimRetention)
}

// ArtifactRetentionWindow controls artifact retention window.
func ArtifactRetentionWindow() time.Duration {
	return Duration("ArtifactRetentionWindow", defaultArtifactRetentionWindow)
}

// AttestationValidityWindow controls attestation validity window.
func AttestationValidityWindow() time.Duration {
	return Duration("AttestationValidityWindow", defaultAttestationValidityWindow)
}

// Runtime-backed bounds preserve a caller or manifest value as the fallback
// while still exposing a stable operator override for that specific operation.
// ResourceHealthCheckTimeout controls resource health check timeout.
func ResourceHealthCheckTimeout(fallback time.Duration) time.Duration {
	return Duration("ResourceHealthCheckTimeout", fallback)
}

// ResourceOperationTimeout controls resource operation timeout.
func ResourceOperationTimeout(fallback time.Duration) time.Duration {
	return Duration("ResourceOperationTimeout", fallback)
}

// ManagedServiceConfiguredTimeout controls managed service configured timeout.
func ManagedServiceConfiguredTimeout(fallback time.Duration) time.Duration {
	return Duration("ManagedServiceConfiguredTimeout", fallback)
}

// TidinessProviderCallTimeout controls tidiness provider call timeout.
func TidinessProviderCallTimeout(fallback time.Duration) time.Duration {
	return Duration("TidinessProviderCallTimeout", fallback)
}

// CredentialReloadOperationTimeout controls credential reload operation timeout.
func CredentialReloadOperationTimeout(fallback time.Duration) time.Duration {
	return Duration("CredentialReloadOperationTimeout", fallback)
}

// ProcessHealthCheckTimeout controls process health check timeout.
func ProcessHealthCheckTimeout(fallback time.Duration) time.Duration {
	return Duration("ProcessHealthCheckTimeout", fallback)
}

// DockerRuntimeEnvironmentTimeout controls docker runtime environment timeout.
func DockerRuntimeEnvironmentTimeout(fallback time.Duration) time.Duration {
	return Duration("DockerRuntimeEnvironmentTimeout", fallback)
}

// HygieneProviderExecutionBudget controls hygiene provider execution budget.
func HygieneProviderExecutionBudget(fallback time.Duration) time.Duration {
	return Duration("HygieneProviderExecutionBudget", fallback)
}

// PrivilegeBrokerRequestTimeout controls privilege broker request timeout.
func PrivilegeBrokerRequestTimeout(fallback time.Duration) time.Duration {
	return Duration("PrivilegeBrokerRequestTimeout", fallback)
}

// StructureProviderCallTimeout controls structure provider call timeout.
func StructureProviderCallTimeout(fallback time.Duration) time.Duration {
	return Duration("StructureProviderCallTimeout", fallback)
}

// CredentialStoreCommandTimeout controls credential store command timeout.
func CredentialStoreCommandTimeout(fallback time.Duration) time.Duration {
	return Duration("CredentialStoreCommandTimeout", fallback)
}

// ScenarioActionTimeout controls scenario action timeout.
func ScenarioActionTimeout(fallback time.Duration) time.Duration {
	return Duration("ScenarioActionTimeout", fallback)
}
