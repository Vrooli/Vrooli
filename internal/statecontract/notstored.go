package statecontract

// Every package-level untyped string constant of a PreciseSources package is
// classified here as stored or not stored. Stored values participate in the
// contract by name and value. An unclassified constant is treated as stored
// when the executable is built (fail closed) and fails that package's
// persistent-state test, so a new constant is classified during development.
//
// NotStoredConstants also lists tunables (retention counts, limits, labels)
// of ConstantSources packages, which are otherwise stamped wholesale.

// StoredConstants are untyped string constants whose values reach stored
// state: statuses, kinds, file and directory names, format versions.
var StoredConstants = map[string][]string{
	"internal/agentsession":                  {"KindAgentManagerRun", "KindDesktopApp", "KindScenarioInstance", "KindTerminal", "KindUnknown", "KindWebConsoleSession", "OriginProgrammatic", "OriginRemote", "OriginUI", "OriginUnspecified", "RunIDSourceIdentityToken", "RunIDSourceInheritedIdentityToken", "SourceInferred"},
	"internal/artifactlease":                 {"Schema", "Suffix"},
	"internal/cliinstall":                    {"installationGoModule", "managerGoModule"},
	"internal/hostsession":                   {"sessionFile"},
	"internal/operatorcapability":            {"ContractVersion", "DecisionDeclined", "DecisionPending", "EvidenceSchemaVersion"},
	"internal/operatorstate":                 {"AccelPreferenceAuto", "AccelPreferenceForceCPU", "AccelPreferencePreferGPU", "SchemaPath", "StateFile", "capacityPostureBalanced", "capacityPostureMinimal", "capacityPostureResponsive", "capacityPostureThroughput", "documentVersion", "hostWorkloadPostureVrooliOnly", "hostWorkloadPostureWholeHost", "profileSessionGuided", "profileSessionManual", "trustPostureHosted", "trustPosturePersonal", "trustPostureShared", "updateControlGuard", "updateControlObserve", "updateControlOwn"},
	"internal/process":                       {"RecordStatusExitedVerified", "RecordStatusRunning"},
	"internal/resources":                     {"ErrorCodeCommandUnavailable", "ErrorCodeOperationFailed", "StatusCodeCommandError", "StatusCodeInvalidStatusPayload", "StatusCodeOK", "StatusCodeTimeout", "StatusCodeUnavailable", "StatusCodeUnsupportedPlatform", "accelBridgeExternalCli", "accelBridgeManagedService", "accelBridgeNativeCli", "blueprintArchivedResourcesPath", "blueprintDirPath", "deprecatedResourcesPath", "driversCliCloudAPI", "installFactsFile", "managedComposeManifestSchemaVersion", "managedServiceAcquisitionDir", "managedServiceAcquisitionFile", "managedServiceHealthy", "managedServiceStateFile", "managedServiceUnhealthy", "managedUpdateCredentialDefaultField", "managedUpdateExecFenceClaimed", "managedUpdateExecFencePrepared", "managedUpdateExecFenceRevoked", "managedUpdateLedgerFile", "managedUpdateRolePrimary", "managedUpdateRoleRestored", "modelCacheEventInstalled", "modelCacheEventStarted", "modelCacheEventStopped", "qualificationObservationRelativePath", "resourceMetadataRootPath", "restoredResourcesDirPath", "userResourceHostID", "vaultBootstrapVault"},
	"internal/resources/manifest":            {"BackendCPU", "BackendCUDA", "BackendMetal", "BackendROCm", "BackendVulkan", "DegradeVerb", "DurableDataDir", "DurableDataFile", "DurableDataJSON", "DurableDataSQLite", "HealthCheckHTTP", "HealthCheckLiveness", "HealthCheckReadiness", "RequireNone", "RequirePreferred", "RequireRequired", "TunableBoolean", "TunableInteger", "TunableString", "VerifyContainerDevice", "VerifyNone", "VerifyProcessDevice", "manifestCloudAPI", "manifestExternalCli", "manifestManagedService", "manifestNativeCli", "manifestUnsupported"},
	"internal/runtimesupervisor":             {"LogFileName"},
	"internal/scenario":                      {"CLIArtifactLocationSibling", "CLIDistributionPrebuiltArtifact", "CLIInvokeInstalledCommand", "CLIKindGoModule", "DependencyFreshnessPolicyRebuildOnly", "DependencyFreshnessPolicyRestartWhenStale", "DependencyFreshnessPolicyReuseRunning", "DependencyStartupPolicyIgnore", "DependencyStartupPolicyMustStart", "DependencyStartupPolicyTryStart", "DependencySupervisionPrecedenceRequired", "DependencySupervisionPrecedenceStartupPolicy", "authenticationProfileCloudflareAccess", "authenticationProfileHybrid", "authenticationProfileLocalReadOnly", "authenticationProfileNone", "authenticationProfileScenarioAuthenticator", "scenarioStatusRunning", "scenarioStatusUnhealthy"},
	"internal/securestore":                   {"BackendNative", "adapterEncryptedFile", "aeadContext", "backendSelectionFile", "credentialCopyLaunchLabel", "credentialCopyTask", "credentialCopyTimer", "hostBoundTPM2Mode", "keyStoreDPAPI", "keyStoreHostKey", "keyStoreKeychain", "keyStorePassphrase", "keyStoreTPM2", "keyringEncryptedMagic", "keyringPlaintextPrefix", "providerHostBound", "providerNativeWrap", "providerPassphrase", "secureStoreFieldSecret", "secureStoreFieldValue", "systemdCredsName", "tpm2PCRPolicy", "unlockCacheDirName", "unlockCacheFileName", "unsafeValueBytes", "valueEncodingPrefix", "vrooliServicePrefix"},
	"internal/setpoint":                      {"CellCPUPressure", "CellCrashLoop", "CellForkRate", "CellMemoryPSI", "CellStrandedMemory", "fallbackDocumentSchemaVersion"},
	"packages/cli-core/agentcatalog":         {"CodingRolePolicySchemaVersion", "LiveCatalogCurated", "ModelDiscoverySchemaVersion", "ModelSourceClaudeSubscription", "TokenWeightPolicyVersion"},
	"packages/cli-core/agentcatalog/harness": {"WindowSourceClaudeStatusline", "WindowSourceCodexModelsCache"},
	"packages/cli-core/usagemode":            {"ArmChallenger", "ArmIncumbent", "ExceptionExplicitModel", "ExceptionFullPower", "KindLearn", "KindSave", "UntilOff", "UntilTime", "UntilWindowReset", "markerFile"},
	"packages/resource-deployment":           {"ActivationStateActivated", "ExternalManagementForbidden", "ServiceShutdownInterrupt", "ServiceShutdownTerminate", "UpdateStrategyRollingStateless", "activationCurrentEntry", "activationGenerations", "activationStagingDir", "artifactCacheManifestName", "artifactCacheSchemaVersion", "dataAcquisitionEager", "dataAcquisitionOnDemand", "deploymentLayoutDir", "deploymentLayoutFile", "deploymentTargetUnsupported", "deploymentVerificationChecksum", "deploymentVerificationHostTool", "releaseTrustSchemaVersion"},
}

// NotStoredConstants are constants that never reach stored state:
// environment variable names, command and transport verbs, messages, request
// routes, schema paths in the repository, reasons printed rather than
// recorded, and tunables of ConstantSources packages.
var NotStoredConstants = map[string][]string{
	"internal/scenario": {
		// environment variable name
		"envPortValidation",
	},
	"internal/resources/manifest": {
		// repository schema path
		"SchemaPath",
	},
	"internal/securestore": {
		// environment variable, process and socket names, and the OS
		// runtime root
		"BackendOverrideEnv", "keyringControlSocket", "keyringDaemonComm", "runtimeRoot",
		// diagnosis and report labels and remedies, printed rather than stored
		"WriteConditionNotChecked", "diagnoseAbsent", "diagnoseUnavailable", "keyringFormatEncrypted", "keyringFormatPlaintext", "keyringFormatUnknown", "nativeScheduleProvider", "secureStoreBackendLibsecret", "secureStoreConditionAvailable", "secureStoreStaleDaemonChecked", "staleKeyringDaemonRemedy",
		// the availability probe entry, written and removed within one check
		"probeKey", "probeService",
	},
	"internal/scenarioruntime": {
		// retention count, not a stored format
		"DemandAuditRetention",
	},
	"internal/resources": {
		// placement explanation text
		"accelBridgeNoWorkload",
		// broker and driver transport/command verbs
		"brokerHttp", "brokerTransportHttp", "brokerTransportLogs", "brokerTransportRestart", "brokerTransportStart", "brokerTransportStop",
		"driversCliHealthChecksFailed", "driversCliInstall", "driversCliLogs", "driversCliStatus", "driversCliUninstall", "driversDockerUninstall",
		// fleet contract validation vocabulary (checks repository manifests)
		"fleetContractDir", "fleetContractExternalCLIDriver", "fleetContractUnsupported",
		// environment names and commands passed to children, not stored
		"gpuOverrideEnvVar", "managedServiceOwnershipTokenEnv", "managedUpdateExecCommand", "modelCacheOwnerCommand",
		// retention reasons printed by the artifact retention command
		"ArtifactRetainActive", "ArtifactRetainInUse", "ArtifactRetainRemove", "ArtifactRetainRollback", "ArtifactRetainRunning", "ArtifactRetainUpdate",
		// generated schema artifacts in the repository
		"resourceConfigRef", "resourceDefinitionsRelPath", "resourcesSchemaRelPath",
	},
	"internal/agentsession": {
		// environment names read from processes and the web-console route
		"webConsoleGetProcedure", "envWebConsoleSession", "envWebConsoleState", "envTmux", "envIdentityToken", "envCodexHome", "envGrokHome",
		"envDesktopMode", "envDesktopData", "envAppDataDir", "envBundleRoot", "envSSHTTY", "envScreen", "envRuntimeInstance",
		// live observation states, reap decisions and usage measurement labels
		"OwnerAlive", "OwnerGone", "OwnerUnknown", "OwnerNotApplicable",
		"StateLive", "StateGhost", "StateAgentless", "StateUnrecorded", "StateUndetermined", "StateOrphaned", "StateIdle",
		"DecisionStop", "DecisionKeep", "OutcomeWouldStop", "OutcomeOwnerReturns", "OutcomeKept",
		"UsageMeasured", "UsagePartial", "UsageUnsupported", "UsageUnavailable",
	},
	"internal/setpoint": {
		// setpoint document location, environment names and compiled-fallback labels
		"RelativePath", "PathEnv", "RootEnv", "FallbackPath", "fallbackPressureSustainText", "fallbackDecisionRef",
	},
	"internal/runtimesupervisor": {
		// mode environment and status labels computed for display
		"ModeEnv", "ModeOff", "ModeOn", "ModeAuto", "StatusStale", "StatusDead", "cliBinaryName",
	},
	"internal/cliinstall": {
		// predicates printed in plans and receipts as explanations
		"scenarioDeletionPredicate", "uninstallPlanPredicate",
	},
	"internal/hostsession": {
		// kernel source path read, never written
		"linuxBootIDPath",
	},
	"packages/resource-deployment": {
		// accepted OS spellings in repository manifests
		"deploymentOSLinux", "deploymentOSMac", "deploymentOSMacOS", "deploymentOSDarwin", "deploymentOSWin", "deploymentOSWindows",
		// input to the derived-identity sample fact, which records its output
		"sampleArch",
	},
	"packages/cli-core/agentcatalog": {
		"OverrideFlag", "refreshProvenancePrefix", "refreshProvenanceEarlier",
	},
	"packages/cli-core/agentcatalog/harness": {
		"ModelWindowsEnv", "ContextLimitEnv", "ContextLimitArg", "HomeIsolationNone", "HomeIsolationEnv", "HomeLayoutAllowlist", "HomeLayoutMirror", "repoMarker",
	},
	"packages/cli-core/usagemode": {
		"EnvArm", "EnvFullPower", "ArmExceptionKey", "EnvMode",
	},
}
