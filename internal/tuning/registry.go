package tuning

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

const callerProvidedValue = "caller-provided"

// Lever describes one operator-visible control: a duration that bounds how
// long, or a count that bounds how many.
type Lever struct {
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Environment     string `json:"environment_variable"`
	CompiledDefault string `json:"compiled_default"`
	ResolvedValue   string `json:"resolved_value"`
	Source          string `json:"source"`
	Description     string `json:"description"`
}

const (
	leverKindDuration = "duration"
	leverKindCount    = "count"
)

type durationDefinition struct {
	Name                  string
	CompiledDefault       time.Duration
	CallerProvidedDefault bool
}

var durationDefinitions = []durationDefinition{
	{Name: "HostFactsRetryInterval", CompiledDefault: defaultHostFactsRetryInterval, CallerProvidedDefault: false},
	{Name: "FastPersistenceRetryInterval", CompiledDefault: defaultFastPersistenceRetryInterval, CallerProvidedDefault: false},
	{Name: "BackgroundLaunchPollInterval", CompiledDefault: defaultBackgroundLaunchPollInterval, CallerProvidedDefault: false},
	{Name: "LifecyclePollInterval", CompiledDefault: defaultLifecyclePollInterval, CallerProvidedDefault: false},
	{Name: "MaintenanceSettleDelay", CompiledDefault: defaultMaintenanceSettleDelay, CallerProvidedDefault: false},
	{Name: "SetupFilesystemSettleDelay", CompiledDefault: defaultSetupFilesystemSettleDelay, CallerProvidedDefault: false},
	{Name: "FastHealthPollInterval", CompiledDefault: defaultFastHealthPollInterval, CallerProvidedDefault: false},
	{Name: "HealthProbeInterval", CompiledDefault: defaultHealthProbeInterval, CallerProvidedDefault: false},
	{Name: "HostPresentationProbeTimeout", CompiledDefault: defaultHostPresentationProbeTimeout, CallerProvidedDefault: false},
	{Name: "ScenarioRequirementsSnapshotTimeout", CompiledDefault: defaultScenarioRequirementsSnapshotTimeout, CallerProvidedDefault: false},
	{Name: "LifecycleHealthPollInterval", CompiledDefault: defaultLifecycleHealthPollInterval, CallerProvidedDefault: false},
	{Name: "SetupProgressPollInterval", CompiledDefault: defaultSetupProgressPollInterval, CallerProvidedDefault: false},
	{Name: "CredentialStoreProbeTimeout", CompiledDefault: defaultCredentialStoreProbeTimeout, CallerProvidedDefault: false},
	{Name: "LifecycleTransitionTimeout", CompiledDefault: defaultLifecycleTransitionTimeout, CallerProvidedDefault: false},
	{Name: "LifecyclePollMaxInterval", CompiledDefault: defaultLifecyclePollMaxInterval, CallerProvidedDefault: false},
	{Name: "EphemeralPortProbeTimeout", CompiledDefault: defaultEphemeralPortProbeTimeout, CallerProvidedDefault: false},
	{Name: "SetupHTTPProbeTimeout", CompiledDefault: defaultSetupHTTPProbeTimeout, CallerProvidedDefault: false},
	{Name: "PrivilegeBrokerUnlockTimeout", CompiledDefault: defaultPrivilegeBrokerUnlockTimeout, CallerProvidedDefault: false},
	{Name: "HostPresentationCommandTimeout", CompiledDefault: defaultHostPresentationCommandTimeout, CallerProvidedDefault: false},
	{Name: "ManagedServiceForceStopTimeout", CompiledDefault: defaultManagedServiceForceStopTimeout, CallerProvidedDefault: false},
	{Name: "HealthCheckTimeout", CompiledDefault: defaultHealthCheckTimeout, CallerProvidedDefault: false},
	{Name: "ListenerEnrichmentTimeout", CompiledDefault: defaultListenerEnrichmentTimeout, CallerProvidedDefault: false},
	{Name: "CredentialUnitProbeTimeout", CompiledDefault: defaultCredentialUnitProbeTimeout, CallerProvidedDefault: false},
	{Name: "ServiceHealthTimeout", CompiledDefault: defaultServiceHealthTimeout, CallerProvidedDefault: false},
	{Name: "FreshnessCheckBudget", CompiledDefault: defaultFreshnessCheckBudget, CallerProvidedDefault: false},
	{Name: "RemoteDesktopProbeTimeout", CompiledDefault: defaultRemoteDesktopProbeTimeout, CallerProvidedDefault: false},
	{Name: "IntegrityCollectionTimeout", CompiledDefault: defaultIntegrityCollectionTimeout, CallerProvidedDefault: false},
	{Name: "ControlPlaneClientTimeout", CompiledDefault: defaultControlPlaneClientTimeout, CallerProvidedDefault: false},
	{Name: "CredentialServiceTimeout", CompiledDefault: defaultCredentialServiceTimeout, CallerProvidedDefault: false},
	{Name: "DependencyBestEffortStartTimeout", CompiledDefault: defaultDependencyBestEffortStartTimeout, CallerProvidedDefault: false},
	{Name: "SecretToolTimeout", CompiledDefault: defaultSecretToolTimeout, CallerProvidedDefault: false},
	{Name: "ReloadFallbackGracePeriod", CompiledDefault: defaultReloadFallbackGracePeriod, CallerProvidedDefault: false},
	{Name: "LifecycleOperationTimeout", CompiledDefault: defaultLifecycleOperationTimeout, CallerProvidedDefault: false},
	{Name: "HostRequirementCommandTimeout", CompiledDefault: defaultHostRequirementCommandTimeout, CallerProvidedDefault: false},
	{Name: "ScenarioHeartbeatTTL", CompiledDefault: defaultScenarioHeartbeatTTL, CallerProvidedDefault: false},
	{Name: "CapacityHeartbeatTTL", CompiledDefault: defaultCapacityHeartbeatTTL, CallerProvidedDefault: false},
	{Name: "SetupOperationTimeout", CompiledDefault: defaultSetupOperationTimeout, CallerProvidedDefault: false},
	{Name: "SetupProgressObservationInterval", CompiledDefault: defaultSetupProgressObservationInterval, CallerProvidedDefault: false},
	{Name: "ResourceControlTimeout", CompiledDefault: defaultResourceControlTimeout, CallerProvidedDefault: false},
	{Name: "PlatformSupportRequestTimeout", CompiledDefault: defaultPlatformSupportRequestTimeout, CallerProvidedDefault: false},
	{Name: "CapacityDegradeDebounce", CompiledDefault: defaultCapacityDegradeDebounce, CallerProvidedDefault: false},
	{Name: "CapabilityRequestTimeout", CompiledDefault: defaultCapabilityRequestTimeout, CallerProvidedDefault: false},
	{Name: "EagerScenarioWaitWindow", CompiledDefault: defaultEagerScenarioWaitWindow, CallerProvidedDefault: false},
	{Name: "StructureProviderBudget", CompiledDefault: defaultStructureProviderBudget, CallerProvidedDefault: false},
	{Name: "HostInventoryTTL", CompiledDefault: defaultHostInventoryTTL, CallerProvidedDefault: false},
	{Name: "SupervisorHealthInterval", CompiledDefault: defaultSupervisorHealthInterval, CallerProvidedDefault: false},
	{Name: "ReachabilityTimeout", CompiledDefault: defaultReachabilityTimeout, CallerProvidedDefault: false},
	{Name: "CredentialRepairTimeout", CompiledDefault: defaultCredentialRepairTimeout, CallerProvidedDefault: false},
	{Name: "TidinessProviderBudget", CompiledDefault: defaultTidinessProviderBudget, CallerProvidedDefault: false},
	{Name: "LifecycleExtendedOperationTimeout", CompiledDefault: defaultLifecycleExtendedOperationTimeout, CallerProvidedDefault: false},
	{Name: "SupervisorRecoveryQuietPeriod", CompiledDefault: defaultSupervisorRecoveryQuietPeriod, CallerProvidedDefault: false},
	{Name: "CredentialReloadFallbackDelay", CompiledDefault: defaultCredentialReloadFallbackDelay, CallerProvidedDefault: false},
	{Name: "HostGPUInventoryTTL", CompiledDefault: defaultHostGPUInventoryTTL, CallerProvidedDefault: false},
	{Name: "DockerRuntimeOperationTimeout", CompiledDefault: defaultDockerRuntimeOperationTimeout, CallerProvidedDefault: false},
	{Name: "ResourceControlExtendedTimeout", CompiledDefault: defaultResourceControlExtendedTimeout, CallerProvidedDefault: false},
	{Name: "SetupExtendedOperationTimeout", CompiledDefault: defaultSetupExtendedOperationTimeout, CallerProvidedDefault: false},
	{Name: "PrivilegeBrokerOperationTimeout", CompiledDefault: defaultPrivilegeBrokerOperationTimeout, CallerProvidedDefault: false},
	{Name: "StructureProviderExtendedBudget", CompiledDefault: defaultStructureProviderExtendedBudget, CallerProvidedDefault: false},
	{Name: "ProviderBudget", CompiledDefault: defaultProviderBudget, CallerProvidedDefault: false},
	{Name: "SupervisorRecoveryCooldown", CompiledDefault: defaultSupervisorRecoveryCooldown, CallerProvidedDefault: false},
	{Name: "EmergencyWatchdogInterval", CompiledDefault: defaultEmergencyWatchdogInterval, CallerProvidedDefault: false},
	{Name: "VaultBootstrapLease", CompiledDefault: defaultVaultBootstrapLease, CallerProvidedDefault: false},
	{Name: "ScenarioReservedClaimTTL", CompiledDefault: defaultScenarioReservedClaimTTL, CallerProvidedDefault: false},
	{Name: "SetupProgressStaleThreshold", CompiledDefault: defaultSetupProgressStaleThreshold, CallerProvidedDefault: false},
	{Name: "HostPlatformInventoryTTL", CompiledDefault: defaultHostPlatformInventoryTTL, CallerProvidedDefault: false},
	{Name: "HostWorkloadInventoryTTL", CompiledDefault: defaultHostWorkloadInventoryTTL, CallerProvidedDefault: false},
	{Name: "CapacityObservedPeakHalflife", CompiledDefault: defaultCapacityObservedPeakHalflife, CallerProvidedDefault: false},
	{Name: "ScenarioWaitTimeout", CompiledDefault: defaultScenarioWaitTimeout, CallerProvidedDefault: false},
	{Name: "CredentialEscrowRetention", CompiledDefault: defaultCredentialEscrowRetention, CallerProvidedDefault: false},
	{Name: "ResourceCommandTimeout", CompiledDefault: defaultResourceCommandTimeout, CallerProvidedDefault: false},
	{Name: "CompanionCrashWindow", CompiledDefault: defaultCompanionCrashWindow, CallerProvidedDefault: false},
	{Name: "CompanionHeartbeatInterval", CompiledDefault: defaultCompanionHeartbeatInterval, CallerProvidedDefault: false},
	{Name: "CompanionCapacitySyncInterval", CompiledDefault: defaultCompanionCapacitySyncInterval, CallerProvidedDefault: false},
	{Name: "ResourceHTTPTimeout", CompiledDefault: defaultResourceHTTPTimeout, CallerProvidedDefault: false},
	{Name: "ResourceShortHTTPTimeout", CompiledDefault: defaultResourceShortHTTPTimeout, CallerProvidedDefault: false},
	{Name: "ResourceMediumHTTPTimeout", CompiledDefault: defaultResourceMediumHTTPTimeout, CallerProvidedDefault: false},
	{Name: "ResourceLongHTTPTimeout", CompiledDefault: defaultResourceLongHTTPTimeout, CallerProvidedDefault: false},
	{Name: "ActivityDebounce", CompiledDefault: defaultActivityDebounce, CallerProvidedDefault: false},
	{Name: "ActivityReadHeaderTimeout", CompiledDefault: defaultActivityReadHeaderTimeout, CallerProvidedDefault: false},
	{Name: "ProgressDisplayResolution", CompiledDefault: defaultProgressDisplayResolution, CallerProvidedDefault: false},
	{Name: "AgentInstallDownloadTimeout", CompiledDefault: defaultAgentInstallDownloadTimeout, CallerProvidedDefault: false},
	{Name: "CopyRetentionWindow", CompiledDefault: defaultCopyRetentionWindow, CallerProvidedDefault: false},
	{Name: "RepairDeadline", CompiledDefault: defaultRepairDeadline, CallerProvidedDefault: false},
	{Name: "DailyRetentionWindow", CompiledDefault: defaultDailyRetentionWindow, CallerProvidedDefault: false},
	{Name: "TerminalClaimRetention", CompiledDefault: defaultTerminalClaimRetention, CallerProvidedDefault: false},
	{Name: "ArtifactRetentionWindow", CompiledDefault: defaultArtifactRetentionWindow, CallerProvidedDefault: false},
	{Name: "AttestationValidityWindow", CompiledDefault: defaultAttestationValidityWindow, CallerProvidedDefault: false},
	{Name: "ResourceHealthCheckTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "ResourceOperationTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "ManagedServiceConfiguredTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "TidinessProviderCallTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "CredentialReloadOperationTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "ProcessHealthCheckTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "DockerRuntimeEnvironmentTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "HygieneProviderExecutionBudget", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "PrivilegeBrokerRequestTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "StructureProviderCallTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "CredentialStoreCommandTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
	{Name: "ScenarioActionTimeout", CompiledDefault: 0, CallerProvidedDefault: true},
}

// Levers returns the complete control surface in stable name order. Calling
// it resolves each lever through Duration or Count, preserving the resolvers'
// once-per-process semantics.
func Levers() []Lever {
	levers := make([]Lever, 0, len(durationDefinitions)+len(countDefinitions))
	for _, definition := range countDefinitions {
		resolved := Count(definition.Name, definition.CompiledDefault)
		source := "default"
		if countHasOverride(definition.Name) {
			source = "environment"
		}
		levers = append(levers, Lever{
			Name:            definition.Name,
			Kind:            leverKindCount,
			Environment:     EnvironmentVariable(definition.Name),
			CompiledDefault: fmt.Sprintf("%d %s", definition.CompiledDefault, definition.Unit),
			ResolvedValue:   fmt.Sprintf("%d %s", resolved, definition.Unit),
			Source:          source,
			Description:     leverDescription(definition.Name),
		})
	}
	for _, definition := range durationDefinitions {
		resolved := Duration(definition.Name, definition.CompiledDefault)
		compiledDefault := definition.CompiledDefault.String()
		resolvedValue := resolved.String()
		if definition.CallerProvidedDefault && !durationHasOverride(definition.Name) {
			compiledDefault = callerProvidedValue
			resolvedValue = callerProvidedValue
		}
		source := "default"
		if durationHasOverride(definition.Name) {
			source = "environment"
		}
		levers = append(levers, Lever{
			Name:            definition.Name,
			Kind:            leverKindDuration,
			Environment:     EnvironmentVariable(definition.Name),
			CompiledDefault: compiledDefault,
			ResolvedValue:   resolvedValue,
			Source:          source,
			Description:     leverDescription(definition.Name),
		})
	}
	sort.Slice(levers, func(i, j int) bool { return levers[i].Name < levers[j].Name })
	return levers
}

// EnvironmentVariable returns the operator override name for a lever.
func EnvironmentVariable(name string) string {
	return environmentPrefix + upperSnake(name)
}

func leverDescription(name string) string {
	var words strings.Builder
	runes := []rune(name)
	for index, current := range runes {
		if index > 0 && unicode.IsUpper(current) &&
			(unicode.IsLower(runes[index-1]) || unicode.IsDigit(runes[index-1]) ||
				(index+1 < len(runes) && unicode.IsUpper(runes[index-1]) && unicode.IsLower(runes[index+1]))) {
			words.WriteByte(' ')
		}
		words.WriteRune(current)
	}
	return "Controls " + strings.ToLower(words.String()) + "."
}

// RenderDocumentation renders the generated reference block consumed by the
// environment-management guide. It intentionally omits resolved values so
// operator-local environment settings cannot perturb committed documentation.
func RenderDocumentation() string {
	levers := Levers()
	var output strings.Builder
	output.WriteString("| Lever | Kind | Environment variable | Compiled default | Description |\n")
	output.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, lever := range levers {
		_, _ = fmt.Fprintf(&output, "| `%s` | %s | `%s` | `%s` | %s |\n", lever.Name, lever.Kind, lever.Environment, lever.CompiledDefault, lever.Description)
	}
	return output.String()
}
