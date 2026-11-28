package runtimeapp

// These classifications bind each persistence writer to the specific owner
// or explain why the written bytes are not a cross-activation state format.
func init() {
	reviewed := map[string]map[string]string{
		"internal/setup": {
			"build_environment.go:freezeGoBuildEnvironment:os.WriteFile#1":                "copies Go environment configuration into the private build cohort staging directory, removed after the build",
			"bootstrap.go:ensureBootstrapPackageManager:os.CreateTemp#1":                  "temporary Homebrew installer script, removed after use",
			"develop.go:startProjectAPI:os.OpenFile#1":                                    "truncates setup API diagnostic log",
			"onboarding.go:launchOnboardingAsOperator:os.OpenFile#1":                      "opens os.DevNull for child I/O",
			"onboarding.go:markComplete:config.WriteOwnedFile#1":                          "writes bootstrap marker; onboarding.go is pinned whole",
			"onboarding.go:saveOnboardingPreferences:os.WriteFile#1":                      "writes onboarding preferences and preserves surrounding configuration; onboarding.go is pinned whole",
			"progress.go:(*progressCoordinator).writeState:config.WriteOwnedFileAtomic#1": "writes activeSetupState root; progress.go is pinned whole",
			"result.go:writeSetupResult:config.WriteOwnedFileAtomic#1":                    "writes SetupResult root; result.go is pinned whole",
		},
		"packages/api-core/scenariocli":          {"resolver.go:writeInstallMetadata:os.WriteFile#1": "writes installMetadata, a scenario-cli-install root"},
		"packages/cli-core/agentcatalog":         {"refresh.go:writeCatalogAtomically:os.CreateTemp#1": "writes CodingRoleCatalog, an agent-catalogs root"},
		"packages/cli-core/agentcatalog/harness": {"contextwindow.go:RecordModelWindow:os.CreateTemp#1": "writes ModelWindowRecord, an agent-catalogs root"},
		"internal/cliinstall": {
			"companions.go:prepareBundle:config.WriteOwnedFileDurable#1":                                  "publishes an empty per-bundle lease before image writes; lease metadata, not an interpreted state record",
			"companions.go:PublishPreparedBundleSourceLocked:config.WriteOwnedFileDurable#1":              "writes source-root pointer while rebaseline progress remains incomplete; prepared-root-bundle covers interpretation",
			"ordinary_bundle_transaction.go:finishOrdinaryBundle:config.WriteOwnedFileDurable#1":          "writes source-root pointer before clearing ordinary bundle intent; prepared-root-bundle covers interpretation",
			"companions.go:PublishPreparedBundleSource:config.WriteOwnedFileDurable#1":                    "writes canonical source-root pointer as an absolute path plus newline; prepared-root-bundle covers writer and reader, scenario-cli-install covers path",
			"ordinary_bundle_transaction.go:beginOrdinaryBundleWithSource:config.WriteOwnedFileDurable#1": "retains verified executable and manifest bytes for recovery; prepared-root-bundle covers transition order",
			"companions.go:prepareBundle:os.WriteFile#1":                                                  "writes verified executable image into retained prepared bundle; prepared-root-bundle source owns interpretation",
			"companions.go:prepareBundle:os.WriteFile#2":                                                  "writes SHA256SUMS manifest consumed on recovery; prepared-root-bundle source owns format and verification",
			"rebaseline_bundle.go:retainRebaselineBundle:config.WriteOwnedFileDurable#1":                  "copies verified SHA256SUMS into the retained recovery bundle; prepared-root-bundle covers its path, bytes and verification",
			"manager.go:atomicInstallVerifiedSource:config.WriteOwnedFileDurable#1":                       "publishes verified companion executable bytes, not a control-plane state envelope",
			"installation.go:(*Manager).writeInstallMetadata:config.WriteOwnedFile#1":                     "writes InstallMetadata, an installation-inventory root",
			"root_candidate.go:copyRootCLIImage:config.WriteOwnedFileDurableStream#1":                     "copies verified executable bytes, not an encoded state envelope",
			"uninstall_plan.go:(*uninstallService).Plan:config.WriteOwnedFile#1":                          "writes UninstallPlan, an installation-inventory root",
			"uninstall_plan.go:(*uninstallService).writeUninstallPlan:config.WriteOwnedFile#1":            "writes UninstallPlan progress, an installation-inventory root",
			"uninstall_service.go:writeInstallRecordLocked:config.WriteOwnedFileDurable#1":                "writes InstallRecord, an installation-inventory root",
		},
		"internal/buildinfo": {
			"prepared_bundle_guard.go:WritePreparedBundleIntent:config.WriteOwnedFileDurable#1": "writes PreparedBundleIntent through the prepared-root-bundle typed codec owner",
		},
		"internal/projectstate": {
			"setup_markers.go:MarkConfigurationComplete:config.WriteOwnedFile#1": "writes ConfigurationCompletion, a project-setup root",
			"setup_markers.go:SaveMigrationLedger:config.WriteOwnedFileAtomic#1": "writes MigrationLedger, a project-setup root",
		},
		"internal/operatorcapability": {"operatorinput_queue.go:Replace:os.WriteFile#1": "writes Pending, an operator-capability root"},
		"internal/operatorstate":      {"operatorstate.go:(*Service).writeStateFile:storage.WriteFileAtomic#1": "writes operator-state document through storage routing"},
		"internal/runtimesupervisor":  {"service.go:openSupervisorLog:os.OpenFile#1": "opens append-only diagnostic log, not recovery state"},
		"packages/cli-core/usagemode": {
			"learn.go:RecordLaunch:os.OpenFile#1":    "appends LaunchMarker JSONL, an agent-catalogs root",
			"learn.go:pruneMarkers:os.WriteFile#1":   "rewrites LaunchMarker JSONL, an agent-catalogs root",
			"lock_unix.go:lockPath:os.OpenFile#1":    "opens empty advisory lock sidecar, not an envelope",
			"usagemode.go:WriteFile:os.CreateTemp#1": "writes usage mode File, an agent-catalogs root",
		},
		"internal/hostsession": {"hostsession.go:persistentFallbackSession:config.WriteOwnedFile#1": "writes fallback session ID; host-session owner declares its filename"},
		"packages/resource-deployment": {
			"activation.go:atomicWrite:os.CreateTemp#1":             "copies canonical release manifest or signature supplied by release, without changing format",
			"activation.go:copyVerifiedFile:os.CreateTemp#1":        "copies checksum-verified release artifact bytes",
			"artifact_cache.go:StageCachedArtifacts:os.WriteFile#1": "writes artifactCacheManifest, a resource-artifact-cache root",
			"artifact_cache.go:copyCacheEntry:os.OpenFile#1":        "copies cached artifact bytes to staging",
		},
		"internal/process": {
			"logbounds.go:trimLog:os.OpenFile#1":                           "opens diagnostic log for truncation",
			"logbounds.go:trimLog:os.OpenFile#2":                           "opens bounded diagnostic backup",
			"process.go:WriteScenarioRecord:config.WriteOwnedFile#1":       "writes plaintext PID sidecar; process-records pins its name and encoder",
			"process.go:WriteScenarioRecord:config.WriteOwnedFileAtomic#1": "writes process Record at canonical path; process-records root",
		},
		"internal/artifactlease": {"artifactlease.go:Save:storage.WriteFileAtomic#1": "writes Lease, an artifact-leases root"},
		"internal/setpoint":      {"sustain.go:(FileState).Observe:os.WriteFile#1": "writes fileObservation, a sustain-observations root"},
	}
	for pkg, sites := range reviewed {
		if auditedWriterSites[pkg] == nil {
			auditedWriterSites[pkg] = map[string]string{}
		}
		for site, reason := range sites {
			auditedWriterSites[pkg][site] = reason
		}
	}
}
