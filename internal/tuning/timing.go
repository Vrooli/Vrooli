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
func HostFactsRetryInterval() time.Duration {
	return Duration("HostFactsRetryInterval", defaultHostFactsRetryInterval)
}

func FastPersistenceRetryInterval() time.Duration {
	return Duration("FastPersistenceRetryInterval", defaultFastPersistenceRetryInterval)
}

func BackgroundLaunchPollInterval() time.Duration {
	return Duration("BackgroundLaunchPollInterval", defaultBackgroundLaunchPollInterval)
}

func LifecyclePollInterval() time.Duration {
	return Duration("LifecyclePollInterval", defaultLifecyclePollInterval)
}

func MaintenanceSettleDelay() time.Duration {
	return Duration("MaintenanceSettleDelay", defaultMaintenanceSettleDelay)
}

func SetupFilesystemSettleDelay() time.Duration {
	return Duration("SetupFilesystemSettleDelay", defaultSetupFilesystemSettleDelay)
}

func FastHealthPollInterval() time.Duration {
	return Duration("FastHealthPollInterval", defaultFastHealthPollInterval)
}

func HealthProbeInterval() time.Duration {
	return Duration("HealthProbeInterval", defaultHealthProbeInterval)
}

func HostPresentationProbeTimeout() time.Duration {
	return Duration("HostPresentationProbeTimeout", defaultHostPresentationProbeTimeout)
}

func ScenarioRequirementsSnapshotTimeout() time.Duration {
	return Duration("ScenarioRequirementsSnapshotTimeout", defaultScenarioRequirementsSnapshotTimeout)
}

func LifecycleHealthPollInterval() time.Duration {
	return Duration("LifecycleHealthPollInterval", defaultLifecycleHealthPollInterval)
}

func SetupProgressPollInterval() time.Duration {
	return Duration("SetupProgressPollInterval", defaultSetupProgressPollInterval)
}

func CredentialStoreProbeTimeout() time.Duration {
	return Duration("CredentialStoreProbeTimeout", defaultCredentialStoreProbeTimeout)
}

func LifecycleTransitionTimeout() time.Duration {
	return Duration("LifecycleTransitionTimeout", defaultLifecycleTransitionTimeout)
}

func LifecyclePollMaxInterval() time.Duration {
	return Duration("LifecyclePollMaxInterval", defaultLifecyclePollMaxInterval)
}

func EphemeralPortProbeTimeout() time.Duration {
	return Duration("EphemeralPortProbeTimeout", defaultEphemeralPortProbeTimeout)
}

func SetupHTTPProbeTimeout() time.Duration {
	return Duration("SetupHTTPProbeTimeout", defaultSetupHTTPProbeTimeout)
}

func PrivilegeBrokerUnlockTimeout() time.Duration {
	return Duration("PrivilegeBrokerUnlockTimeout", defaultPrivilegeBrokerUnlockTimeout)
}

func HostPresentationCommandTimeout() time.Duration {
	return Duration("HostPresentationCommandTimeout", defaultHostPresentationCommandTimeout)
}

func ManagedServiceForceStopTimeout() time.Duration {
	return Duration("ManagedServiceForceStopTimeout", defaultManagedServiceForceStopTimeout)
}

func HealthCheckTimeout() time.Duration {
	return Duration("HealthCheckTimeout", defaultHealthCheckTimeout)
}

func ListenerEnrichmentTimeout() time.Duration {
	return Duration("ListenerEnrichmentTimeout", defaultListenerEnrichmentTimeout)
}

func CredentialUnitProbeTimeout() time.Duration {
	return Duration("CredentialUnitProbeTimeout", defaultCredentialUnitProbeTimeout)
}

func ServiceHealthTimeout() time.Duration {
	return Duration("ServiceHealthTimeout", defaultServiceHealthTimeout)
}

func FreshnessCheckBudget() time.Duration {
	return Duration("FreshnessCheckBudget", defaultFreshnessCheckBudget)
}

func RemoteDesktopProbeTimeout() time.Duration {
	return Duration("RemoteDesktopProbeTimeout", defaultRemoteDesktopProbeTimeout)
}

func IntegrityCollectionTimeout() time.Duration {
	return Duration("IntegrityCollectionTimeout", defaultIntegrityCollectionTimeout)
}

func ControlPlaneClientTimeout() time.Duration {
	return Duration("ControlPlaneClientTimeout", defaultControlPlaneClientTimeout)
}

func CredentialServiceTimeout() time.Duration {
	return Duration("CredentialServiceTimeout", defaultCredentialServiceTimeout)
}

func DependencyBestEffortStartTimeout() time.Duration {
	return Duration("DependencyBestEffortStartTimeout", defaultDependencyBestEffortStartTimeout)
}

func SecretToolTimeout() time.Duration {
	return Duration("SecretToolTimeout", defaultSecretToolTimeout)
}

func ReloadFallbackGracePeriod() time.Duration {
	return Duration("ReloadFallbackGracePeriod", defaultReloadFallbackGracePeriod)
}

func LifecycleOperationTimeout() time.Duration {
	return Duration("LifecycleOperationTimeout", defaultLifecycleOperationTimeout)
}

func HostRequirementCommandTimeout() time.Duration {
	return Duration("HostRequirementCommandTimeout", defaultHostRequirementCommandTimeout)
}

func ScenarioHeartbeatTTL() time.Duration {
	return Duration("ScenarioHeartbeatTTL", defaultScenarioHeartbeatTTL)
}

func CapacityHeartbeatTTL() time.Duration {
	return Duration("CapacityHeartbeatTTL", defaultCapacityHeartbeatTTL)
}

func SetupOperationTimeout() time.Duration {
	return Duration("SetupOperationTimeout", defaultSetupOperationTimeout)
}

func SetupProgressObservationInterval() time.Duration {
	return Duration("SetupProgressObservationInterval", defaultSetupProgressObservationInterval)
}

func ResourceControlTimeout() time.Duration {
	return Duration("ResourceControlTimeout", defaultResourceControlTimeout)
}

func PlatformSupportRequestTimeout() time.Duration {
	return Duration("PlatformSupportRequestTimeout", defaultPlatformSupportRequestTimeout)
}

func CapacityDegradeDebounce() time.Duration {
	return Duration("CapacityDegradeDebounce", defaultCapacityDegradeDebounce)
}

func CapabilityRequestTimeout() time.Duration {
	return Duration("CapabilityRequestTimeout", defaultCapabilityRequestTimeout)
}

func EagerScenarioWaitWindow() time.Duration {
	return Duration("EagerScenarioWaitWindow", defaultEagerScenarioWaitWindow)
}

func StructureProviderBudget() time.Duration {
	return Duration("StructureProviderBudget", defaultStructureProviderBudget)
}

func HostInventoryTTL() time.Duration {
	return Duration("HostInventoryTTL", defaultHostInventoryTTL)
}

func SupervisorHealthInterval() time.Duration {
	return Duration("SupervisorHealthInterval", defaultSupervisorHealthInterval)
}

func ReachabilityTimeout() time.Duration {
	return Duration("ReachabilityTimeout", defaultReachabilityTimeout)
}

func CredentialRepairTimeout() time.Duration {
	return Duration("CredentialRepairTimeout", defaultCredentialRepairTimeout)
}

func TidinessProviderBudget() time.Duration {
	return Duration("TidinessProviderBudget", defaultTidinessProviderBudget)
}

func LifecycleExtendedOperationTimeout() time.Duration {
	return Duration("LifecycleExtendedOperationTimeout", defaultLifecycleExtendedOperationTimeout)
}

func SupervisorRecoveryQuietPeriod() time.Duration {
	return Duration("SupervisorRecoveryQuietPeriod", defaultSupervisorRecoveryQuietPeriod)
}

func CredentialReloadFallbackDelay() time.Duration {
	return Duration("CredentialReloadFallbackDelay", defaultCredentialReloadFallbackDelay)
}

func HostGPUInventoryTTL() time.Duration {
	return Duration("HostGPUInventoryTTL", defaultHostGPUInventoryTTL)
}

func DockerRuntimeOperationTimeout() time.Duration {
	return Duration("DockerRuntimeOperationTimeout", defaultDockerRuntimeOperationTimeout)
}

func ResourceControlExtendedTimeout() time.Duration {
	return Duration("ResourceControlExtendedTimeout", defaultResourceControlExtendedTimeout)
}

func SetupExtendedOperationTimeout() time.Duration {
	return Duration("SetupExtendedOperationTimeout", defaultSetupExtendedOperationTimeout)
}

func PrivilegeBrokerOperationTimeout() time.Duration {
	return Duration("PrivilegeBrokerOperationTimeout", defaultPrivilegeBrokerOperationTimeout)
}

func StructureProviderExtendedBudget() time.Duration {
	return Duration("StructureProviderExtendedBudget", defaultStructureProviderExtendedBudget)
}
func ProviderBudget() time.Duration { return Duration("ProviderBudget", defaultProviderBudget) }
func SupervisorRecoveryCooldown() time.Duration {
	return Duration("SupervisorRecoveryCooldown", defaultSupervisorRecoveryCooldown)
}

func EmergencyWatchdogInterval() time.Duration {
	return Duration("EmergencyWatchdogInterval", defaultEmergencyWatchdogInterval)
}

func VaultBootstrapLease() time.Duration {
	return Duration("VaultBootstrapLease", defaultVaultBootstrapLease)
}

func ScenarioReservedClaimTTL() time.Duration {
	return Duration("ScenarioReservedClaimTTL", defaultScenarioReservedClaimTTL)
}

func SetupProgressStaleThreshold() time.Duration {
	return Duration("SetupProgressStaleThreshold", defaultSetupProgressStaleThreshold)
}

func HostPlatformInventoryTTL() time.Duration {
	return Duration("HostPlatformInventoryTTL", defaultHostPlatformInventoryTTL)
}

func HostWorkloadInventoryTTL() time.Duration {
	return Duration("HostWorkloadInventoryTTL", defaultHostWorkloadInventoryTTL)
}

func CapacityObservedPeakHalflife() time.Duration {
	return Duration("CapacityObservedPeakHalflife", defaultCapacityObservedPeakHalflife)
}

func ScenarioWaitTimeout() time.Duration {
	return Duration("ScenarioWaitTimeout", defaultScenarioWaitTimeout)
}

func CredentialEscrowRetention() time.Duration {
	return Duration("CredentialEscrowRetention", defaultCredentialEscrowRetention)
}

func ResourceCommandTimeout() time.Duration {
	return Duration("ResourceCommandTimeout", defaultResourceCommandTimeout)
}

func CompanionCrashWindow() time.Duration {
	return Duration("CompanionCrashWindow", defaultCompanionCrashWindow)
}

func CompanionHeartbeatInterval() time.Duration {
	return Duration("CompanionHeartbeatInterval", defaultCompanionHeartbeatInterval)
}

func CompanionCapacitySyncInterval() time.Duration {
	return Duration("CompanionCapacitySyncInterval", defaultCompanionCapacitySyncInterval)
}

func ResourceHTTPTimeout() time.Duration {
	return Duration("ResourceHTTPTimeout", defaultResourceHTTPTimeout)
}

func ResourceShortHTTPTimeout() time.Duration {
	return Duration("ResourceShortHTTPTimeout", defaultResourceShortHTTPTimeout)
}

func ResourceMediumHTTPTimeout() time.Duration {
	return Duration("ResourceMediumHTTPTimeout", defaultResourceMediumHTTPTimeout)
}

func ResourceLongHTTPTimeout() time.Duration {
	return Duration("ResourceLongHTTPTimeout", defaultResourceLongHTTPTimeout)
}
func ActivityDebounce() time.Duration { return Duration("ActivityDebounce", defaultActivityDebounce) }
func ActivityReadHeaderTimeout() time.Duration {
	return Duration("ActivityReadHeaderTimeout", defaultActivityReadHeaderTimeout)
}

func ProgressDisplayResolution() time.Duration {
	return Duration("ProgressDisplayResolution", defaultProgressDisplayResolution)
}

func CompanionLogMaxBytes() int64 {
	return defaultCompanionLogMaxBytes
}

func AgentInstallDownloadTimeout() time.Duration {
	return Duration("AgentInstallDownloadTimeout", defaultAgentInstallDownloadTimeout)
}

func CopyRetentionWindow() time.Duration {
	return Duration("CopyRetentionWindow", defaultCopyRetentionWindow)
}
func RepairDeadline() time.Duration { return Duration("RepairDeadline", defaultRepairDeadline) }
func DailyRetentionWindow() time.Duration {
	return Duration("DailyRetentionWindow", defaultDailyRetentionWindow)
}

func TerminalClaimRetention() time.Duration {
	return Duration("TerminalClaimRetention", defaultTerminalClaimRetention)
}

func ArtifactRetentionWindow() time.Duration {
	return Duration("ArtifactRetentionWindow", defaultArtifactRetentionWindow)
}

func AttestationValidityWindow() time.Duration {
	return Duration("AttestationValidityWindow", defaultAttestationValidityWindow)
}

// Runtime-backed bounds preserve a caller or manifest value as the fallback
// while still exposing a stable operator override for that specific operation.
func ResourceHealthCheckTimeout(fallback time.Duration) time.Duration {
	return Duration("ResourceHealthCheckTimeout", fallback)
}

func ResourceOperationTimeout(fallback time.Duration) time.Duration {
	return Duration("ResourceOperationTimeout", fallback)
}

func ManagedServiceConfiguredTimeout(fallback time.Duration) time.Duration {
	return Duration("ManagedServiceConfiguredTimeout", fallback)
}

func TidinessProviderCallTimeout(fallback time.Duration) time.Duration {
	return Duration("TidinessProviderCallTimeout", fallback)
}

func CredentialReloadOperationTimeout(fallback time.Duration) time.Duration {
	return Duration("CredentialReloadOperationTimeout", fallback)
}

func ProcessHealthCheckTimeout(fallback time.Duration) time.Duration {
	return Duration("ProcessHealthCheckTimeout", fallback)
}

func DockerRuntimeEnvironmentTimeout(fallback time.Duration) time.Duration {
	return Duration("DockerRuntimeEnvironmentTimeout", fallback)
}

func HygieneProviderExecutionBudget(fallback time.Duration) time.Duration {
	return Duration("HygieneProviderExecutionBudget", fallback)
}

func PrivilegeBrokerRequestTimeout(fallback time.Duration) time.Duration {
	return Duration("PrivilegeBrokerRequestTimeout", fallback)
}

func StructureProviderCallTimeout(fallback time.Duration) time.Duration {
	return Duration("StructureProviderCallTimeout", fallback)
}

func CredentialStoreCommandTimeout(fallback time.Duration) time.Duration {
	return Duration("CredentialStoreCommandTimeout", fallback)
}

func ScenarioActionTimeout(fallback time.Duration) time.Duration {
	return Duration("ScenarioActionTimeout", fallback)
}
