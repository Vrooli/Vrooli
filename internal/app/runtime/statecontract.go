package runtimeapp

import (
	"reflect"

	sharedscenariocli "github.com/vrooli/api-core/scenariocli"
	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/usagemode"

	platform "github.com/vrooli/platform-go"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"

	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/artifactlease"
	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/hostwireless"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/peerrecord"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	"github.com/vrooli/vrooli/internal/recovery"
	"github.com/vrooli/vrooli/internal/resources"
	resourcemanifest "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	watchdog "github.com/vrooli/vrooli/internal/safeguards/emergency-watchdog"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/securestore"
	"github.com/vrooli/vrooli/internal/setpoint"
	"github.com/vrooli/vrooli/internal/setup"
	"github.com/vrooli/vrooli/internal/setup/assets"
	"github.com/vrooli/vrooli/internal/statecontract"
	triposrsource "github.com/vrooli/vrooli/internal/tools/triposr-source"
)

// NativePersistentStateContract is the root and watchdog executables'
// persistent-state contract digest, or "" when the executable is unqualified
// for automatic rollback. It resolves no user paths and opens no stores.
func NativePersistentStateContract() string {
	return statecontract.Contract(nativePersistenceOwners())
}

// NativePersistentStateManifest is the per-owner contract behind the digest.
func NativePersistentStateManifest() (statecontract.Manifest, error) {
	return statecontract.Compute(nativePersistenceOwners())
}

// nativePersistenceOwners is the inventory of every owner whose state the
// native root control plane writes. A new persistence owner on that path
// must add its declaration here.
func nativePersistenceOwners() []statecontract.Owner {
	return append(rootCLIStateOwners(), []statecontract.Owner{
		{Name: "root-rebaseline-progress", WholeSource: true, Roots: []reflect.Type{reflect.TypeFor[rebaselineprogress.Progress]()}, Facts: map[string]any{"Version": 1}},
		{Name: "scenario-cli-install", WholeSource: true, Roots: sharedscenariocli.PersistentTypes()},
		setup.PersistentState(),
		projectstate.PersistentState(),
		peerrecord.PersistentState(),
		assets.PersistentState(),
		triposrsource.PersistentState(),
		scenarioruntime.PersistentState(),
		capacity.PersistentState(),
		process.PersistentState(),
		hostsession.PersistentState(),
		hostwireless.PersistentState(),
		agentsession.PersistentState(),
		setpoint.PersistentState(),
		resources.PersistentState(),
		resourcedeployment.PersistentState(),
		cliinstall.PersistentState(),
		coreactivation.PersistentState(),
		recovery.PersistentState(),
		runtimesupervisor.PersistentState(),
		lightingruntime.PersistentState(),
		watchdog.PersistentState(),
		nativeOperationIdentities(),
		operatorstate.PersistentState(),
		operatorcapability.PersistentState(),
		agentCatalogs(),
		acquisition(),
		artifactlease.PersistentState(),
		resourcemanifest.PersistentState(),
		scenario.PersistentState(),
		resourceStorage(),
		securestore.PersistentState(),
		hostinventory.PersistentState(),
		{Name: "agent-catalog-assets", WholeSource: true},
		{Name: "agent-catalogs-asset-policy", WholeSource: true},
		{Name: "template-validation-markers", WholeSource: true},
		{Name: "break-glass", WholeSource: true},
		{Name: "kopia-registry", WholeSource: true},
		{Name: "harness-lifecycle", WholeSource: true},
	}...)
}

// resourceStorage declares the resource storage root layout (the
// per-platform resolution of where resource config, data, cache, logs and
// state live). A changed layout would send a restored prior image to other
// paths; the package is small and rarely edited, so it is whole-source.
func resourceStorage() statecontract.Owner {
	return statecontract.Owner{Name: "resource-storage", WholeSource: true}
}

// acquisition declares binaryfetch, a separate module whose acquisition
// declarations resources stores (their shapes are reached from the resources
// roots) and whose artifact identity digests the prior image recomputes to
// verify what it launches. It is whole-source: its manifest vocabulary is
// validated with inline literals, so a precise owner would first need that
// vocabulary named.
func acquisition() statecontract.Owner {
	return statecontract.Owner{Name: "acquisition", WholeSource: true}
}

// agentCatalogs declares the coding-agent catalogs, context-window records
// and usage-mode file the shared cli-core module caches on disk. cli-core is a
// separate module, so its declaration lives here.
func agentCatalogs() statecontract.Owner {
	return statecontract.Owner{
		Name: "agent-catalogs",
		Roots: []reflect.Type{
			reflect.TypeFor[agentcatalog.CodingRoleCatalog](),
			reflect.TypeFor[agentcatalog.LiveModelCatalog](),
			reflect.TypeFor[agentcatalog.ModelResolution](),
			reflect.TypeFor[harness.ModelWindowRecord](),
			reflect.TypeFor[usagemode.File](),
			reflect.TypeFor[usagemode.LaunchMarker](),
		},
		Facts: map[string]any{
			"CodingRolePolicySchemaVersion":  agentcatalog.CodingRolePolicySchemaVersion,
			"ModelDiscoverySchemaVersion":    agentcatalog.ModelDiscoverySchemaVersion,
			"SchemaVersion":                  usagemode.SchemaVersion,
			"ModelWindowRecordSchemaVersion": harness.ModelWindowRecordSchemaVersion,
		},
	}
}

// nativeOperationIdentities covers the native operation and owned-service
// identities stored inside activation records, managed-update fences and
// service refs (intent shapes, spec digests, unit names, slices, kinds).
// Records written by one release are validated by recomputing them in
// another, so these platform sources participate whole (WholeSources).
func nativeOperationIdentities() statecontract.Owner {
	return statecontract.Owner{
		Name:        "native-operation-identities",
		WholeSource: true,
		Roots:       platform.NativeOwnedPersistentTypes(),
		Facts:       platform.NativeOwnedPersistentFacts(),
	}
}

// rootCLIStateOwners covers root-only stores that survive activation and are
// read by later commands or by recovery. Whole-source captures private types
// and codecs until their owners can expose a more precise format contract.
func rootCLIStateOwners() []statecontract.Owner {
	names := []string{"prepared-root-bundle", "companion-state", "tool-installation", "baseline-floor", "lifecycle-cache", "go-cache", "credential-recovery", "wordpress-state", "release-authority", "host-watchdog-state", "cloud-target", "guard-modes", "retained-runtime", "recovery-points"}
	owners := make([]statecontract.Owner, 0, len(names))
	for _, name := range names {
		owners = append(owners, statecontract.Owner{Name: name, WholeSource: true})
	}
	return owners
}
