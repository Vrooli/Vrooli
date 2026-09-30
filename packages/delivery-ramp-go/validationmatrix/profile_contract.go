package validationmatrix

import domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"

// ProfileContract describes the target-owned prerequisites for an environment
// profile. A profile is never considered runnable merely because it is listed
// in the UI; the target must explicitly advertise every prerequisite.
type ProfileContract struct {
	Profile              domainv1.ValidationEnvironmentProfile
	Label                string
	Category             string
	Executable           bool
	RequiredCapabilities []domainv1.ValidationTargetCapability
}

var profileContracts = []ProfileContract{
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL, Label: "Normal", Category: "baseline", Executable: true},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_OFFLINE, Label: "Offline", Category: "network", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_OFFLINE_NETWORK}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_SLOW_NETWORK, Label: "Slow network", Category: "network", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NETWORK_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_MISSING_CREDENTIAL, Label: "Missing credential", Category: "credential", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_CREDENTIAL_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_PROVIDER_FAILURE, Label: "Provider failure", Category: "provider", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_PROVIDER_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UPDATE_INTERRUPTED, Label: "Interrupted update", Category: "update", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATER, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATE_FEED}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_CRASH_RECOVERY, Label: "Crash recovery", Category: "recovery", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_CRASH_RECOVERY}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_HIGH_LATENCY, Label: "High latency", Category: "network", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NETWORK_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_PACKET_LOSS, Label: "Packet loss", Category: "network", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NETWORK_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_PROVIDER_UNAVAILABLE, Label: "Provider unavailable", Category: "provider", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_PROVIDER_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_RECONNECT, Label: "Reconnect", Category: "network", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NETWORK_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_EXPIRED_CREDENTIAL, Label: "Expired credential", Category: "credential", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_CREDENTIAL_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UNAVAILABLE_CREDENTIAL, Label: "Unavailable credential", Category: "credential", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_CREDENTIAL_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_WRONG_SCOPE_CREDENTIAL, Label: "Wrong-scope credential", Category: "credential", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_CREDENTIAL_CONTROL}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UPDATE_DISCOVERY, Label: "Update discovery", Category: "update", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATER, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATE_FEED}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UPDATE_DOWNLOAD, Label: "Update download", Category: "update", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATER, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATE_FEED}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UPDATE_VERIFICATION, Label: "Update verification", Category: "update", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATER, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATE_FEED}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UPDATE_ROLLBACK, Label: "Update rollback", Category: "update", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATER, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATE_FEED}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UPDATE_RESTART, Label: "Update restart", Category: "update", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATER, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATE_FEED}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UPDATE_FAILURE, Label: "Update failure", Category: "update", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATER, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_UPDATE_FEED}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_BUNDLED_PRIVATE, Label: "Bundled private communication", Category: "communication", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_COMMUNICATION_PEER}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_TIER_ONE, Label: "Tier 1 communication", Category: "communication", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_COMMUNICATION_PEER}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_SHARED_PROVIDER, Label: "Shared-provider communication", Category: "communication", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_COMMUNICATION_PEER}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_FALLBACK, Label: "Fallback communication", Category: "communication", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_COMMUNICATION_PEER}},
	{Profile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_TIER_TWO_PEER, Label: "Tier 2 peer communication", Category: "communication", RequiredCapabilities: []domainv1.ValidationTargetCapability{domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_COMMUNICATION_PEER}},
}

// ProfileContracts returns a copy so callers cannot mutate the shared catalog.
func ProfileContracts() []ProfileContract {
	contracts := make([]ProfileContract, len(profileContracts))
	for i, contract := range profileContracts {
		contracts[i] = contract
		contracts[i].RequiredCapabilities = append([]domainv1.ValidationTargetCapability(nil), contract.RequiredCapabilities...)
	}
	return contracts
}

func profileContract(profile domainv1.ValidationEnvironmentProfile) (ProfileContract, bool) {
	for _, contract := range profileContracts {
		if contract.Profile == profile {
			return contract, true
		}
	}
	return ProfileContract{}, false
}
