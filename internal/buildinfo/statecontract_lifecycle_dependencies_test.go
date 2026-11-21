package buildinfo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// lifecycleStateDependencyPackages are package boundaries whose APIs either
// define persistent records consumed by lifecycle, or determine the inputs and
// identities of lifecycle's durable freshness records. Calls across these
// boundaries need explicit review because hashing internal/lifecycle alone
// does not bind their implementations.
var lifecycleStateDependencyPackages = map[string]bool{
	"github.com/vrooli/vrooli/internal/buildinfo":       true,
	"github.com/vrooli/vrooli/internal/config":          true,
	"github.com/vrooli/cli-core/cliutil":                true,
	"github.com/vrooli/vrooli/internal/packagegov":      true,
	"github.com/vrooli/vrooli/internal/peerrecord":      true,
	"github.com/vrooli/vrooli/internal/process":         true,
	"github.com/vrooli/vrooli/internal/retainedruntime": true,
	"github.com/vrooli/vrooli/internal/scenario":        true,
	"github.com/vrooli/vrooli/internal/scenarioruntime": true,
	"github.com/vrooli/vrooli/packages/proto/protogen":  true,
}

// lifecycleProjectImports classifies every imported Vrooli package in the
// lifecycle source tree, including packages currently judged unrelated to
// persistence. A new repository-owned dependency must be reviewed before it
// can become an unrecorded part of a writer or reader's transitive closure.
var lifecycleProjectImports = map[string]string{
	"github.com/vrooli/api-core/coreset":                                                       "runtime integration; no persistence API used",
	"github.com/vrooli/api-core/discovery":                                                     "runtime integration; no persistence API used",
	"github.com/vrooli/api-core/lifecycle":                                                     "runtime integration; no persistence API used",
	"github.com/vrooli/api-core/runtimediagnostics":                                            "diagnostic paths; no persisted lifecycle record",
	"github.com/vrooli/api-core/scenariocli":                                                   "scenario CLI resolution; lifecycle-cache exclusion reviewed separately",
	"github.com/vrooli/cli-core/cliutil":                                                       "scenario-cli-install owner; freshness manifest writer and reader",
	"github.com/vrooli/envkit-go":                                                              "runtime environment handling; no persistence API used",
	"github.com/vrooli/platform-go":                                                            "process/runtime operations; no lifecycle state API used",
	"github.com/vrooli/repo-contract-go":                                                       "runtime-home/path identity; scenario-cli-install owner",
	"github.com/vrooli/vrooli/internal/baselinefloor":                                          "baseline comparison; baseline-floor owner",
	"github.com/vrooli/vrooli/internal/buildflags":                                             "runtime build configuration; no persistent lifecycle state API used",
	"github.com/vrooli/vrooli/internal/buildinfo":                                              "Go input classifier call is an open source-coverage obligation",
	"github.com/vrooli/vrooli/internal/capacity":                                               "runtime admission service; capacity owner",
	"github.com/vrooli/vrooli/internal/cliinstall":                                             "installation inventory; installation-inventory owner",
	"github.com/vrooli/vrooli/internal/config":                                                 "file/path helpers; classify individual calls before extracting state code",
	"github.com/vrooli/vrooli/internal/credentialauthority":                                    "credential authority; credential-recovery owner",
	"github.com/vrooli/vrooli/internal/fsx":                                                    "filesystem helpers; no persistent lifecycle codec API used",
	"github.com/vrooli/vrooli/internal/hostinventory":                                          "host observations; host-facts-cache owner",
	"github.com/vrooli/vrooli/internal/hostreqkit":                                             "host requirement evaluation; no persistent lifecycle state API used",
	"github.com/vrooli/vrooli/internal/hostreqspec":                                            "host requirement schema; precise owner",
	"github.com/vrooli/vrooli/internal/hostsession":                                            "host-session owner",
	"github.com/vrooli/vrooli/internal/lifecycle/retainedruntime":                              "retained-runtime whole-source owner",
	"github.com/vrooli/vrooli/internal/logx":                                                   "logging; no persistent lifecycle record API used",
	"github.com/vrooli/vrooli/internal/maintenance":                                            "maintenance control; no persistent lifecycle record API used",
	"github.com/vrooli/vrooli/internal/network":                                                "network probes; no persistent lifecycle record API used",
	"github.com/vrooli/vrooli/internal/nodejsruntime":                                          "runtime process planning; no persistent lifecycle record API used",
	"github.com/vrooli/vrooli/internal/packagegov":                                             "manifest/output/build semantics; source coverage remains open",
	"github.com/vrooli/vrooli/internal/peerrecord":                                             "peer-records precise owner plus lifecycle caller source",
	"github.com/vrooli/vrooli/internal/ports":                                                  "runtime port operations; stored representation owned by scenario-contract",
	"github.com/vrooli/vrooli/internal/process":                                                "process-records precise owner plus lifecycle caller source",
	"github.com/vrooli/vrooli/internal/repocontractmeta":                                       "repository metadata; no persistent lifecycle codec API used",
	"github.com/vrooli/vrooli/internal/resources":                                              "resource control integration; resources owner",
	"github.com/vrooli/vrooli/internal/resources/control":                                      "resource control integration; resources owner",
	"github.com/vrooli/vrooli/internal/resources/env":                                          "resource environment handling; resources owner",
	"github.com/vrooli/vrooli/internal/resources/manifest":                                     "resource-manifest precise owner",
	"github.com/vrooli/vrooli/internal/runtime":                                                "runtime process integration; tool-installation owner",
	"github.com/vrooli/vrooli/internal/runtimeprofile":                                         "runtime diagnostics; no lifecycle state API used",
	"github.com/vrooli/vrooli/internal/runtimesupervisor":                                      "supervisor-owner precise owner",
	"github.com/vrooli/vrooli/internal/scenario":                                               "scenario-contract precise owner",
	"github.com/vrooli/vrooli/internal/scenarioruntime":                                        "scenario-contract precise owner plus lifecycle caller source",
	"github.com/vrooli/vrooli/internal/shell":                                                  "subprocess execution; no persistent lifecycle record API used",
	"github.com/vrooli/vrooli/internal/tuning":                                                 "runtime tuning; no stored state encoding API used",
	"github.com/vrooli/vrooli/internal/values":                                                 "value parsing; no persistent lifecycle record API used",
	"github.com/vrooli/vrooli/packages/capacity":                                               "capacity owner",
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain":                   "generated runtime messages; no persistence codec API used",
	"github.com/vrooli/vrooli/packages/proto/gen/go/common/v1":                                 "generated runtime messages; no persistence codec API used",
	"github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config":                  "generated runtime messages; no persistence codec API used",
	"github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect": "generated runtime client; no persistent lifecycle state API used",
	"github.com/vrooli/vrooli/packages/proto/protogen":                                         "Proto artifact selection and identity; source coverage remains open",
}

// lifecycleStateDependencyCallSites is the reviewed set of calls from
// internal/lifecycle into the boundaries above. Each entry needs an owner or
// a reason it is not a persistent-state dependency. Exact function and
// ordinal identity makes a new call site fail closed.
var lifecycleStateDependencyCallSites = map[string]string{
	"closure_cache.go:isGoBuildSource:github.com/vrooli/vrooli/internal/buildinfo.IsGoBuildSource#1":                                                   "build-input classifier; source coverage is an open proof obligation",
	"phases.go:(*Runner).runWithLifecycleLog:github.com/vrooli/vrooli/internal/config.ChownToInvokingUser#1":                                           "run-log ownership helper; transient logs, not state records",
	"phases.go:(*Runner).runWithLifecycleLog:github.com/vrooli/vrooli/internal/config.EnsureOwnedDir#1":                                                "run-log directory helper; transient logs, not state records",
	"phases.go:(*Runner).runWithLifecycleLog:github.com/vrooli/vrooli/internal/config.OpenOwnedFile#1":                                                 "run-log writer; transient logs, not state records",
	"phases.go:(*Runner).startTrackedProcessContext:github.com/vrooli/vrooli/internal/config.EnsureOwnedDir#1":                                         "process sidecar directory helper; process-records owner",
	"phases.go:(*Runner).startTrackedProcessContext:github.com/vrooli/vrooli/internal/config.OpenOwnedFile#1":                                          "process sidecar writer; process-records owner",
	"phases.go:createComponentRuntimeDirectories:github.com/vrooli/vrooli/internal/config.EnsureOwnedDir#1":                                            "runtime directory helper; no serialized state",
	"run_log_retention.go:(*retainedRunLog).close:github.com/vrooli/vrooli/internal/config.ChownToInvokingUser#1":                                      "run-log ownership helper; transient logs, not state records",
	"run_log_retention.go:openRetainedRunLog:github.com/vrooli/vrooli/internal/config.EnsureOwnedDir#1":                                                "run-log directory helper; transient logs, not state records",
	"run_log_retention.go:openRetainedRunLog:github.com/vrooli/vrooli/internal/config.OpenOwnedFile#1":                                                 "run-log writer; transient logs, not state records",
	"scenario_locator.go:LocateScenarioCompletenessCLI:github.com/vrooli/vrooli/internal/config.HomeDir#1":                                             "scenario CLI path helper; excluded from root state contract",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStop:github.com/vrooli/vrooli/internal/scenarioruntime.ActivePortClaimStatuses#1":               "port-claim status vocabulary used when stop enumerates the instance's dynamically allocated ports; runtime-registry owns the rows",
	"component_exit.go:(*Runner).retireComponentGeneration:github.com/vrooli/vrooli/internal/process.LaunchRefID#1":                                    "process identity; process-records owner",
	"component_exit.go:(*Runner).retireComponentGeneration:github.com/vrooli/vrooli/internal/process.ReadScenarioRecords#1":                            "process record reader; process-records owner",
	"component_exit.go:(*Runner).retireComponentGeneration:github.com/vrooli/vrooli/internal/process.ReadScenarioRecords#2":                            "process record reader; process-records owner",
	"component_exit.go:(*Runner).retireComponentGeneration:github.com/vrooli/vrooli/internal/process.WriteScenarioRecord#1":                            "process tombstone writer; process-records owner",
	"component_exit.go:(*Runner).stopManagedComponents:github.com/vrooli/vrooli/internal/process.ReadScenarioRecords#1":                                "process record reader; process-records owner",
	"component_exit.go:stopComponentGenerations:github.com/vrooli/vrooli/internal/process.AwaitExit#1":                                                 "process retirement reader; process-records owner",
	"dependencies.go:(*Runner).dependencyHasOtherLiveConsumers:github.com/vrooli/vrooli/internal/scenarioruntime.ActiveInstanceStatuses#1":             "runtime registry status reader; scenario-contract owner",
	"dependencies.go:(*Runner).ensureDependency:github.com/vrooli/vrooli/internal/scenarioruntime.DependencyConsumerID#1":                              "runtime registry identity; scenario-contract owner",
	"dependency_demand.go:(*Runner).acquireDependencyDemandLease:github.com/vrooli/vrooli/internal/scenarioruntime.DependencyConsumerID#1":             "runtime registry identity; scenario-contract owner",
	"dependency_demand.go:(*Runner).acquireDependencyDemandLease:github.com/vrooli/vrooli/internal/scenarioruntime.DependencyLeaseID#1":                "runtime registry identity; scenario-contract owner",
	"dependency_demand.go:(*Runner).releaseDependencyDemandIfCandidate:github.com/vrooli/vrooli/internal/scenarioruntime.DependencyConsumerID#1":       "runtime registry identity; scenario-contract owner",
	"dependency_demand.go:(*Runner).releaseDependencyLeasesForConsumerExcept:github.com/vrooli/vrooli/internal/scenarioruntime.DependencyConsumerID#1": "runtime registry identity; scenario-contract owner",
	"dependency_demand.go:(*Runner).snapshotDependencyDemandLeases:github.com/vrooli/vrooli/internal/scenarioruntime.ActiveInstanceStatuses#1":         "runtime registry status reader; scenario-contract owner",
	"dependency_demand.go:(*Runner).snapshotDependencyDemandLeases:github.com/vrooli/vrooli/internal/scenarioruntime.DependencyConsumerID#1":           "runtime registry identity; scenario-contract owner",
	"deploy_report.go:(*Runner).DeployReportByName:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                               "runtime key reader; scenario-contract owner",
	"deploy_report.go:(*Runner).deployReportArtifact:github.com/vrooli/cli-core/cliutil.ComputeFreshnessManifest#1":                                    "freshness manifest writer; scenario-cli-install owner",
	"deploy_report.go:(*Runner).deployReportArtifact:github.com/vrooli/cli-core/cliutil.ReadFreshnessManifest#1":                                       "freshness manifest reader; scenario-cli-install owner",
	"freshness.go:(*Runner).FreshnessReportContext:github.com/vrooli/cli-core/cliutil.ReadFreshnessManifest#1":                                         "freshness manifest reader; scenario-cli-install owner",
	"freshness.go:(*Runner).evaluateArtifactFreshnessContext:github.com/vrooli/cli-core/cliutil.EvaluateFreshness#1":                                   "freshness manifest reader; scenario-cli-install owner",
	"freshness.go:(*Runner).evaluateArtifactFreshnessContext:github.com/vrooli/cli-core/cliutil.ReadFreshnessManifest#1":                               "freshness manifest reader; scenario-cli-install owner",
	"freshness.go:(*Runner).freshnessInputsContext:github.com/vrooli/cli-core/cliutil.ResolveFreshnessInputFiles#1":                                    "freshness input discovery; scenario-cli-install owner",
	"freshness.go:(*Runner).stampArtifactFreshness:github.com/vrooli/cli-core/cliutil.ComputeFreshnessManifest#1":                                      "freshness manifest writer; scenario-cli-install owner",
	"freshness.go:(*Runner).stampArtifactFreshness:github.com/vrooli/cli-core/cliutil.WriteFreshnessManifest#1":                                        "freshness manifest writer; scenario-cli-install owner",
	"freshness.go:binariesFreshnessArtifactsContext:github.com/vrooli/cli-core/cliutil.FreshnessManifestPath#1":                                        "freshness manifest path; scenario-cli-install owner",
	"freshness_builder.go:builderFreshnessContextWithRequiredClosure:github.com/vrooli/cli-core/cliutil.FreshnessManifestPath#1":                       "freshness manifest path; scenario-cli-install owner",
	"lifecycle.go:(*Runner).Restart:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                                              "runtime key reader; scenario-contract owner",
	"lifecycle.go:(*Runner).Start:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                                                "runtime key reader; scenario-contract owner",
	"lifecycle.go:(*Runner).Stop:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                                                 "runtime key reader; scenario-contract owner",
	"lifecycle.go:(*Runner).admitVariantBuild:github.com/vrooli/vrooli/internal/scenarioruntime.ActiveInstanceStatuses#1":                              "runtime registry status reader; scenario-contract owner",
	"lifecycle.go:(*Runner).cleanupScenarioRuntimeWithRegistryContext:github.com/vrooli/vrooli/internal/process.LiveRecords#1":                         "process record reader; process-records owner",
	"lifecycle.go:(*Runner).cleanupScenarioRuntimeWithRegistryContext:github.com/vrooli/vrooli/internal/process.LiveRecords#2":                         "process record reader; process-records owner",
	"lifecycle.go:(*Runner).cleanupScenarioRuntimeWithRegistryContext:github.com/vrooli/vrooli/internal/process.RemoveScenarioRecord#1":                "process record remover; process-records owner",
	"lifecycle.go:(*Runner).cleanupScenarioRuntimeWithRegistryContext:github.com/vrooli/vrooli/internal/process.ScenarioProcessDir#1":                  "process record path; process-records owner",
	"lifecycle.go:(*Runner).loadScenario:github.com/vrooli/vrooli/internal/scenario.Load#1":                                                            "scenario manifest reader; scenario-contract owner",
	"lifecycle.go:(*Runner).loadScenario:github.com/vrooli/vrooli/internal/scenario.ReadService#1":                                                     "scenario manifest reader; scenario-contract owner",
	"lifecycle.go:(*Runner).loadScenario:github.com/vrooli/vrooli/internal/scenario.SandboxEnvFromEnv#1":                                               "scenario environment input; scenario-contract owner",
	"lifecycle.go:(*Runner).runtimeDeps:github.com/vrooli/vrooli/internal/scenarioruntime.NewSQLiteStore#1":                                            "runtime registry store constructor; scenario-contract owner",
	"lifecycle.go:(*Runner).startScenario:github.com/vrooli/vrooli/internal/scenario.EvaluateHealth#1":                                                 "scenario manifest reader; scenario-contract owner",
	"lifecycle.go:(*Runner).startScenario:github.com/vrooli/vrooli/internal/scenarioruntime.DependencyConsumerID#1":                                    "runtime registry identity; scenario-contract owner",
	"lifecycle.go:(*Runner).startWithState:github.com/vrooli/vrooli/internal/scenario.ValidateManifestPorts#1":                                         "scenario manifest reader; scenario-contract owner",
	"lifecycle.go:scenarioTreeHasRequiredHostRequirements:github.com/vrooli/vrooli/internal/scenario.ReadService#1":                                    "scenario manifest reader; scenario-contract owner",
	"peer_env.go:removePeerRecord:github.com/vrooli/vrooli/internal/peerrecord.Remove#1":                                                               "peer record remover; peer-records owner",
	"peer_env.go:writePeerRecord:github.com/vrooli/vrooli/internal/peerrecord.Write#1":                                                                 "peer record writer; peer-records owner",
	"phases.go:(*Runner).activeInstances:github.com/vrooli/vrooli/internal/scenarioruntime.ActiveInstanceStatuses#1":                                   "runtime registry status reader; scenario-contract owner",
	"phases.go:(*Runner).checkComponentReadinessNamed:github.com/vrooli/vrooli/internal/process.LiveRecords#1":                                         "process record reader; process-records owner",
	"phases.go:(*Runner).checkComponentReadinessNamed:github.com/vrooli/vrooli/internal/process.ReadScenarioRecords#1":                                 "process record reader; process-records owner",
	"phases.go:(*Runner).executePhaseDetailed:github.com/vrooli/vrooli/internal/process.ScenarioLifecycleLogPath#1":                                    "process log path; process-records owner",
	"phases.go:(*Runner).placeServiceProcess:github.com/vrooli/vrooli/internal/scenarioruntime.ServiceLaunchScopeName#1":                               "runtime process identity; scenario-contract owner",
	"phases.go:(*Runner).runForegroundStep:github.com/vrooli/vrooli/internal/packagegov.GuardPnpmStore#1":                                              "package output guard; packagegov implementation not yet stamped",
	"phases.go:(*Runner).runForegroundStep:github.com/vrooli/vrooli/internal/packagegov.GuardPnpmStore#2":                                              "package output guard; packagegov implementation not yet stamped",
	"phases.go:(*Runner).runPhaseDetailed:github.com/vrooli/vrooli/internal/process.ScenarioLifecycleLogPath#1":                                        "process log path; process-records owner",
	"phases.go:(*Runner).runWithLifecycleLog:github.com/vrooli/vrooli/internal/process.ScenarioLifecycleLogPath#1":                                     "process log path; process-records owner",
	"phases.go:(*Runner).startTrackedProcessContext:github.com/vrooli/vrooli/internal/process.Incarnation#1":                                           "process identity; process-records owner",
	"phases.go:(*Runner).startTrackedProcessContext:github.com/vrooli/vrooli/internal/process.RemoveScenarioRecord#1":                                  "process record remover; process-records owner",
	"phases.go:(*Runner).startTrackedProcessContext:github.com/vrooli/vrooli/internal/process.ScenarioLogsDir#1":                                       "process record path; process-records owner",
	"phases.go:(*Runner).startTrackedProcessContext:github.com/vrooli/vrooli/internal/process.WriteScenarioRecord#1":                                   "process record writer; process-records owner",
	"phases.go:(*Runner).startTrackedProcessContext:github.com/vrooli/vrooli/internal/process.WriteScenarioRecord#2":                                   "process record writer; process-records owner",
	"phases.go:(*Runner).waitForComponentDependencies:github.com/vrooli/vrooli/internal/process.LiveRecords#1":                                         "process record reader; process-records owner",
	"phases.go:(*Runner).waitForComponentDependencies:github.com/vrooli/vrooli/internal/process.ReadScenarioRecords#1":                                 "process record reader; process-records owner",
	"retained_runtime.go:(*Runner).CaptureRuntimeRelease:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                         "runtime key reader; scenario-contract owner",
	"retained_runtime.go:(*Runner).SelectedRuntimeReleaseID:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                      "runtime key reader; scenario-contract owner",
	"retained_runtime.go:(*Runner).VerifyRuntimeRelease:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                          "runtime key reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStop:github.com/vrooli/vrooli/internal/scenarioruntime.StopCandidateInstanceStatuses#1":         "runtime registry status reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:github.com/vrooli/vrooli/internal/scenarioruntime.ActiveInstanceStatuses#1":                   "runtime registry status reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:github.com/vrooli/vrooli/internal/scenarioruntime.IsActivePortClaimStatus#1":                  "runtime registry status reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:github.com/vrooli/vrooli/internal/scenarioruntime.ListenerEvidenceFromClaims#1":               "runtime registry evidence reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:github.com/vrooli/vrooli/internal/scenarioruntime.ProcessEvidenceForInstance#1":               "runtime registry evidence reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:github.com/vrooli/vrooli/internal/scenarioruntime.ReconcileRuntime#1":                         "runtime registry reader/writer; scenario-contract owner",
	"runtime_registry.go:recordRuntimeProcessRef:github.com/vrooli/vrooli/internal/process.LaunchRefID#1":                                              "process identity; process-records owner",
	"runtime_registry.go:runtimePortURL:github.com/vrooli/vrooli/internal/scenarioruntime.LocalPortURL#1":                                              "runtime registry identity; scenario-contract owner",
	"service_protection.go:recoveryServiceContainment:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                            "runtime key reader; scenario-contract owner",
	"shared_packages.go:(*Runner).acquireProtoSetupLock:github.com/vrooli/vrooli/packages/proto/protogen.DefaultArtifactRoot#1":                        "Proto artifact path; protogen implementation not yet stamped",
	"shared_packages.go:ProvisionGeneratedPackages:github.com/vrooli/vrooli/internal/packagegov.LoadPackage#1":                                         "package manifest reader; packagegov implementation not yet stamped",
	"shared_packages.go:loadGovernedPackage:github.com/vrooli/vrooli/internal/packagegov.LoadPackage#1":                                                "package manifest reader; packagegov implementation not yet stamped",
	"shared_packages.go:provisionSelectedProtoArtifact:github.com/vrooli/vrooli/packages/proto/protogen.ArtifactErrorCode#1":                           "Proto artifact error vocabulary; protogen implementation not yet stamped",
	"shared_packages.go:provisionSelectedProtoArtifact:github.com/vrooli/vrooli/packages/proto/protogen.DefaultArtifactRoot#1":                         "Proto artifact path; protogen implementation not yet stamped",
	"shared_packages.go:provisionSelectedProtoArtifact:github.com/vrooli/vrooli/packages/proto/protogen.NewArtifactStore#1":                            "Proto selected artifact reader; protogen implementation not yet stamped",
	"shared_packages.go:provisionSharedPackageWithOptions:github.com/vrooli/vrooli/internal/packagegov.RunCommandsWithOptions#1":                       "package build runner; packagegov implementation not yet stamped",
	"shared_packages.go:sharedPackageArtifactSelectionDigest:github.com/vrooli/vrooli/packages/proto/protogen.DefaultArtifactRoot#1":                   "Proto artifact path; protogen implementation not yet stamped",
	"shared_refs.go:ScenarioAt:github.com/vrooli/vrooli/internal/scenario.ReadService#1":                                                               "scenario manifest reader; scenario-contract owner",
	"wait.go:(*Runner).WaitScenario:github.com/vrooli/vrooli/internal/scenarioruntime.ParseInstanceKey#1":                                              "runtime key reader; scenario-contract owner",
}

// lifecycleStateRepositoryCallSites inventories calls through the composed
// runtime repository interface. These methods are invoked through `store`, so
// an import-qualified call scan alone would miss them. Method names are read
// from the repository interfaces in internal/scenarioruntime, which means a
// new interface method used by lifecycle must be reviewed here.
var lifecycleStateRepositoryCallSites = map[string]string{
	"component_exit.go:(*Runner).retireComponentGeneration:scenarioruntime-repository.store.ListProcessRefs#1":                           "runtime registry process-ref reader; scenario-contract owner",
	"component_exit.go:(*Runner).retireComponentGeneration:scenarioruntime-repository.store.UpdateProcessRefStatus#1":                    "runtime registry process-ref writer; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStop:scenarioruntime-repository.store.ListPortClaims#1":                           "runtime registry port-claim reader; stop includes claims whose listener outlived its process record",
	"dependencies.go:(*Runner).dependencyHasOtherLiveConsumers:scenarioruntime-repository.store.ListInstances#1":                         "runtime registry instance reader; scenario-contract owner",
	"dependency_demand.go:(*Runner).acquireDependencyDemandLease:scenarioruntime-repository.store.AcquireDemandLease#1":                  "runtime demand-lease writer; scenario-contract owner",
	"dependency_demand.go:(*Runner).releaseDemandLease:scenarioruntime-repository.store.ReleaseDemandLease#1":                            "runtime demand-lease writer; scenario-contract owner",
	"dependency_demand.go:(*Runner).releaseDependencyLeasesForConsumerExcept:scenarioruntime-repository.store.ListDemandLeases#1":        "runtime demand-lease reader; scenario-contract owner",
	"dependency_demand.go:(*Runner).releaseDependencyLeasesForConsumerExcept:scenarioruntime-repository.store.ReleaseDemandLease#1":      "runtime demand-lease writer; scenario-contract owner",
	"dependency_demand.go:(*Runner).snapshotDependencyDemandLeases:scenarioruntime-repository.store.ListDemandLeases#1":                  "runtime demand-lease reader; scenario-contract owner",
	"dependency_demand.go:(*Runner).snapshotDependencyDemandLeases:scenarioruntime-repository.store.ListInstances#1":                     "runtime registry instance reader; scenario-contract owner",
	"lifecycle.go:(*Runner).admitVariantBuild:scenarioruntime-repository.store.ListInstances#1":                                          "runtime registry instance reader; scenario-contract owner",
	"lifecycle.go:(*Runner).latestStartOperationSnapshot:scenarioruntime-repository.store.GetLatestStartOperation#1":                     "runtime operation reader; scenario-contract owner",
	"lifecycle.go:(*Runner).latestStartOperationSnapshot:scenarioruntime-repository.store.GetStartOperationFenceRecoveryOwner#1":         "runtime fence recovery reader; scenario-contract owner",
	"lifecycle.go:(*Runner).startScenario:scenarioruntime-repository.store.RetainExplicitInstance#1":                                     "runtime demand-lease writer; scenario-contract owner",
	"operation_record.go:(*Runner).beginStartOperationRecord:scenarioruntime-repository.store.BeginStartOperation#1":                     "runtime operation writer; scenario-contract owner",
	"operation_record.go:(*Runner).beginStartOperationRecord:scenarioruntime-repository.store.MarkStartOperationAbandoned#1":             "runtime operation writer; scenario-contract owner",
	"operation_record.go:(*startOperationRecorder).settleWithFreshHandle:scenarioruntime-repository.store.MarkStartOperationAbandoned#1": "runtime operation writer; scenario-contract owner",
	"operation_record.go:(*startOperationRecorder).settleWithFreshHandle:scenarioruntime-repository.store.UpdateStartOperation#1":        "runtime operation writer; scenario-contract owner",
	"phases.go:(*Runner).activeInstances:scenarioruntime-repository.store.ListInstances#1":                                               "runtime registry instance reader; scenario-contract owner",
	"provider_fence.go:(*Runner).startOperationFenceOwner:scenarioruntime-repository.store.GetStartOperationFenceOwner#1":                "runtime fence recovery reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStart:scenarioruntime-repository.store.CreateLease#1":                             "runtime registry lease writer; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStart:scenarioruntime-repository.store.RecordEvent#1":                             "runtime registry event writer; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStart:scenarioruntime-repository.store.UpdateInstanceStatus#1":                    "runtime registry status writer; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStop:scenarioruntime-repository.store.ListInstances#1":                            "runtime registry instance reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStop:scenarioruntime-repository.store.ListInstances#2":                            "runtime registry instance reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStop:scenarioruntime-repository.store.ListProcessRefs#1":                          "runtime registry process-ref reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).beginRuntimeRegistryStop:scenarioruntime-repository.store.UpdateInstanceStatus#1":                     "runtime registry status writer; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:scenarioruntime-repository.store.GetHealthSnapshot#1":                           "runtime registry health reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:scenarioruntime-repository.store.ListInstances#1":                               "runtime registry instance reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:scenarioruntime-repository.store.ListPortClaims#1":                              "runtime registry port reader; scenario-contract owner",
	"runtime_registry.go:(*Runner).lookupRegistryRuntime:scenarioruntime-repository.store.ListProcessRefs#1":                             "runtime registry process-ref reader; scenario-contract owner",
	"runtime_registry.go:recordRuntimeProcessRef:scenarioruntime-repository.store.AddProcessRef#1":                                       "runtime registry process-ref writer; scenario-contract owner",
	"wait.go:(*Runner).readStartOperationView:scenarioruntime-repository.store.GetLatestStartOperation#1":                                "runtime operation reader; scenario-contract owner",
	"wait.go:(*Runner).readStartOperationView:scenarioruntime-repository.store.GetStartOperation#1":                                      "runtime operation reader; scenario-contract owner",
	"wait.go:(*Runner).readStartOperationView:scenarioruntime-repository.store.PhaseDurationEstimates#1":                                 "runtime operation reader; scenario-contract owner",
}

var lifecycleStateRepositoryInterfaces = []string{
	"LifecycleRepository",
	"DemandLeaseRepository",
	"PortClaimRepository",
	"CleanupRepository",
	"HealthRepository",
	"ProcessRefRepository",
	"EventRepository",
	"StartOperationRepository",
}

func lifecycleStateDependencyCalls(root string) ([]string, error) {
	dir := filepath.Join(root, "internal/lifecycle")
	repositoryMethods, err := lifecycleStateRepositoryMethods(root)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var sites []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil, err
		}
		aliases := map[string]string{}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if !lifecycleStateDependencyPackages[importPath] {
				continue
			}
			alias := filepath.Base(importPath)
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			aliases[alias] = importPath
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			function := fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) == 1 {
				receiver := fn.Recv.List[0].Type
				start, end := fset.Position(receiver.Pos()).Offset, fset.Position(receiver.End()).Offset
				function = "(" + string(src[start:end]) + ")." + function
			}
			counts := map[string]int{}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				base, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				if importPath, tracked := aliases[base.Name]; tracked {
					key := importPath + "." + selector.Sel.Name
					counts[key]++
					sites = append(sites, fmt.Sprintf("%s:%s:%s#%d", entry.Name(), function, key, counts[key]))
					return true
				}
				if !repositoryMethods[selector.Sel.Name] {
					return true
				}
				receiverStart := fset.Position(selector.X.Pos()).Offset
				receiverEnd := fset.Position(selector.X.End()).Offset
				key := "scenarioruntime-repository." + string(src[receiverStart:receiverEnd]) + "." + selector.Sel.Name
				counts[key]++
				sites = append(sites, fmt.Sprintf("%s:%s:%s#%d", entry.Name(), function, key, counts[key]))
				return true
			})
		}
	}
	sort.Strings(sites)
	for i := 1; i < len(sites); i++ {
		if sites[i] == sites[i-1] {
			return nil, fmt.Errorf("duplicate lifecycle state dependency call %s", sites[i])
		}
	}
	return sites, nil
}

func lifecycleStateProjectImports(root string) ([]string, error) {
	dir := filepath.Join(root, "internal/lifecycle")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	imports := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(importPath, "github.com/vrooli/") {
				imports[importPath] = true
			}
		}
	}
	result := make([]string, 0, len(imports))
	for importPath := range imports {
		result = append(result, importPath)
	}
	sort.Strings(result)
	return result, nil
}

func lifecycleStateRepositoryMethods(root string) (map[string]bool, error) {
	methods := map[string]bool{}
	for _, fileName := range []string{"types.go", "startop.go"} {
		path := filepath.Join(root, "internal/scenarioruntime", fileName)
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok.String() != "type" {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || !containsString(lifecycleStateRepositoryInterfaces, typeSpec.Name.Name) {
					continue
				}
				iface, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					return nil, fmt.Errorf("%s is not an interface", typeSpec.Name.Name)
				}
				for _, field := range iface.Methods.List {
					if field.Names == nil {
						// The selected lifecycle interfaces currently embed only local
						// value interfaces. Fail closed if that shape changes.
						return nil, fmt.Errorf("%s embeds an unexpanded interface", typeSpec.Name.Name)
					}
					for _, name := range field.Names {
						methods[name.Name] = true
					}
				}
			}
		}
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("lifecycle runtime repository method inventory is empty")
	}
	return methods, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestLifecycleStateDependencyCallsAreInventoried(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	sites, err := lifecycleStateDependencyCalls(root)
	if err != nil {
		t.Fatal(err)
	}
	actual := make(map[string]bool, len(sites))
	for _, site := range sites {
		actual[site] = true
		switch {
		case strings.Contains(site, ":scenarioruntime-repository."):
			if lifecycleStateRepositoryCallSites[site] == "" {
				t.Errorf("unreviewed lifecycle repository dependency call %s", site)
			}
		default:
			if lifecycleStateDependencyCallSites[site] == "" {
				t.Errorf("unreviewed lifecycle state dependency call %s", site)
			}
		}
	}
	for site := range lifecycleStateDependencyCallSites {
		if !actual[site] {
			t.Errorf("stale lifecycle state dependency inventory entry %s", site)
		}
	}
	for site := range lifecycleStateRepositoryCallSites {
		if !actual[site] {
			t.Errorf("stale lifecycle repository dependency inventory entry %s", site)
		}
	}
	imports, err := lifecycleStateProjectImports(root)
	if err != nil {
		t.Fatal(err)
	}
	actualImports := make(map[string]bool, len(imports))
	for _, importPath := range imports {
		actualImports[importPath] = true
		if lifecycleProjectImports[importPath] == "" {
			t.Errorf("unreviewed repository-owned lifecycle import %s", importPath)
		}
	}
	for importPath := range lifecycleProjectImports {
		if !actualImports[importPath] {
			t.Errorf("stale repository-owned lifecycle import inventory entry %s", importPath)
		}
	}
}
