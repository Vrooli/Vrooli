package tuning

import "time"

// Control-plane timing levers are named for the work they bound. The values
// are intentionally unchanged from the former value-named defaults.
const (
	// HostFactsRetryInterval bounds the retry delay between host-facts probes.
	HostFactsRetryInterval = 10 * time.Millisecond
	// FastPersistenceRetryInterval bounds short persistence transaction retries.
	FastPersistenceRetryInterval = 25 * time.Millisecond
	// BackgroundLaunchPollInterval bounds polling while a background launch settles.
	BackgroundLaunchPollInterval = 50 * time.Millisecond
	// LifecyclePollInterval bounds lifecycle state polling.
	LifecyclePollInterval = 100 * time.Millisecond
	// MaintenanceSettleDelay gives maintenance observations time to settle.
	MaintenanceSettleDelay = 150 * time.Millisecond
	// SetupFilesystemSettleDelay gives setup filesystem changes time to settle.
	SetupFilesystemSettleDelay = 200 * time.Millisecond
	// FastHealthPollInterval bounds fast health and process polling.
	FastHealthPollInterval = 250 * time.Millisecond
	// HealthProbeInterval bounds ordinary health probe polling.
	HealthProbeInterval = 500 * time.Millisecond
	// HostPresentationProbeTimeout bounds host presentation discovery.
	HostPresentationProbeTimeout = 750 * time.Millisecond
	// ShortOperationTimeout bounds a short control-plane operation.
	ShortOperationTimeout = 1 * time.Second
	// ShortOperationDeadline bounds short waits and unlock operations.
	ShortOperationDeadline = 2 * time.Second
	// HealthCheckTimeout bounds a health-check request.
	HealthCheckTimeout = 3 * time.Second
	// ServiceHealthTimeout bounds service health and reachability checks.
	ServiceHealthTimeout = 5 * time.Second
	// IntegrityCollectionTimeout bounds integrity inventory collection.
	IntegrityCollectionTimeout = 8 * time.Second
	// ControlPlaneClientTimeout bounds ordinary control-plane HTTP calls.
	ControlPlaneClientTimeout = 10 * time.Second
	// CredentialServiceTimeout bounds credential-provider operations.
	CredentialServiceTimeout = 15 * time.Second
	// ReloadFallbackGracePeriod bounds credential reload fallback handling.
	ReloadFallbackGracePeriod = 20 * time.Second
	// StandardOperationTimeout bounds ordinary lifecycle and resource operations.
	StandardOperationTimeout = 30 * time.Second
	// SupervisorHealthInterval controls supervisor health observation cadence.
	SupervisorHealthInterval = 45 * time.Second
	// ReachabilityTimeout bounds a service reachability probe.
	ReachabilityTimeout = 60 * time.Second
	// CredentialRepairTimeout bounds credential repair attempts.
	CredentialRepairTimeout = 90 * time.Second
	// ExtendedOperationTimeout bounds extended control-plane operations.
	ExtendedOperationTimeout = 2 * time.Minute
	// ProviderBudget bounds a provider's work budget.
	ProviderBudget = 3 * time.Minute
	// LongOperationTimeout bounds long-running lifecycle operations.
	LongOperationTimeout = 5 * time.Minute
	// LongOperationBudget bounds long-running provider and build work.
	LongOperationBudget = 10 * time.Minute
	// CopyRetentionWindow bounds credential-copy retention behavior.
	CopyRetentionWindow = 15 * time.Minute
	// RepairDeadline bounds a repair request deadline.
	RepairDeadline = 30 * time.Minute
	// DailyRetentionWindow bounds daily retention and cache validity.
	DailyRetentionWindow = 24 * time.Hour
	// TerminalClaimRetention bounds terminal runtime claim retention.
	TerminalClaimRetention = 14 * 24 * time.Hour
	// ArtifactRetentionWindow bounds managed artifact attestation retention.
	ArtifactRetentionWindow = 30 * 24 * time.Hour
	// AttestationValidityWindow bounds the maximum attestation lifetime.
	AttestationValidityWindow = 31 * 24 * time.Hour
)
