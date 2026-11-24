package tuning

import "time"

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
