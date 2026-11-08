package resources

import (
	"reflect"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/statecontract"
)

const (
	managedUpdateCredentialSchemaVersion   = 1
	managedUpdateExecFenceSchemaVersion    = 1
	managedUpdateInputCleanupSchemaVersion = 1
)

// PersistentState declares the resource control plane's durable records:
// the managed-update ledger, journal, fences, credential snapshots and input
// cleanup receipts; managed-service state, install facts and compose
// provenance; broker, model-cache, liveness and qualification state; and the
// operator archive lists.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name: "resources",
		Roots: []reflect.Type{
			reflect.TypeFor[managedUpdateLedger](),
			reflect.TypeFor[ManagedServiceUpdate](),
			reflect.TypeFor[ManagedUpdateEvent](),
			reflect.TypeFor[ManagedUpdateCredentialSnapshot](),
			reflect.TypeFor[ManagedUpdateLegacyObservation](),
			reflect.TypeFor[managedUpdateExecFence](),
			reflect.TypeFor[managedUpdateInputCleanup](),
			reflect.TypeFor[ManagedServiceState](),
			reflect.TypeFor[ManagedServiceArtifact](),
			reflect.TypeFor[ResourceManifest](),
			reflect.TypeFor[InstallFacts](),
			reflect.TypeFor[managedComposeManifest](),
			reflect.TypeFor[binaryfetch.ComposeStep](),
			reflect.TypeFor[ComposeCandidateReceipt](),
			reflect.TypeFor[BrokerState](),
			reflect.TypeFor[ModelCacheUsage](),
			reflect.TypeFor[livenessState](),
			reflect.TypeFor[resourceQualificationObservation](),
			reflect.TypeFor[DeprecatedResourceList](),
			reflect.TypeFor[BlueprintArchivedResourceList](),
		},
		Facts: map[string]any{
			"managedUpdateSchemaVersion":             managedUpdateSchemaVersion,
			"managedUpdateCredentialSchemaVersion":   managedUpdateCredentialSchemaVersion,
			"managedUpdateExecFenceSchemaVersion":    managedUpdateExecFenceSchemaVersion,
			"managedUpdateInputCleanupSchemaVersion": managedUpdateInputCleanupSchemaVersion,
			"managedComposeManifestSchemaVersion":    managedComposeManifestSchemaVersion,
			"credential_snapshot_mac_sample":         credentialSnapshotMACSample(),
		},
	}
}

// credentialSnapshotMACSample runs the production snapshot digest
// derivation (item selection, ordering, framing) on fixed in-memory values.
func credentialSnapshotMACSample() string {
	first, second := strings.Repeat("a", 64), strings.Repeat("b", 64)
	values := map[string][]byte{first: []byte("value"), second: []byte("v")}
	snapshot := ManagedUpdateCredentialSnapshot{SchemaVersion: managedUpdateCredentialSchemaVersion, Resource: "sample", Items: []ManagedUpdateCredentialItem{{LogicalID: "a", Field: "f", FileKey: first}, {LogicalID: "b", Field: "f", FileKey: second}}}
	digest, err := credentialSnapshotDigest(make([]byte, 32), snapshot, func(key string) ([]byte, error) { return values[key], nil })
	if err != nil {
		return "error: " + err.Error()
	}
	return digest
}
