package capacity

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// capacitySchemaDigest pins the compiled shape of format 4: the SQL DDL plus
// the JSON payload types stored inside its TEXT columns. The digest is derived
// from the types and DDL, so it moves on its own whenever a stored column or
// payload field is added, removed, renamed or retyped. This constant makes that
// move a review event instead of a silent compatibility change: a deliberate
// shape change must bump SchemaVersion, add the downlevel fixture that proves
// the new version is readable by the version it claims to support, and re-pin
// this digest. Update it for no other reason.
const capacitySchemaDigest = "6c6d66b7d1642f085c004ad8efb22c47b650219cbaf952b397373bde0980311e"

// capacityStatePath is the ledger's durable file, relative to the runtime home.
// It must match DefaultDBPath's `state/capacity.db` resolution.
const capacityStatePath = "state/capacity.db"

// StateFormat describes the capacity ledger as a versioned persistent store.
//
// The ledger is greenfield-by-policy: ensureSchema refuses a stamped database
// older than SchemaVersion and asks for an operator-run conversion, so this
// executable decodes exactly the format it writes and migrates nothing in
// place. That is why Reads is one version and MigratesFrom is zero.
func StateFormat() (stateformat.Store, error) {
	shapes, err := stateformat.DigestTypes(
		reflect.TypeFor[CapacityClaim](),
		reflect.TypeFor[ResidencyTransitionRecord](),
		reflect.TypeFor[PlacementQualification](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "capacity",
		Write: stateformat.Version(SchemaVersion),
		Reads: []stateformat.Version{stateformat.Version(SchemaVersion)},
		Schemas: map[stateformat.Version]string{
			stateformat.Version(SchemaVersion): stateformat.ComposeDigest(stateformat.DigestDDL(schemaSQL), shapes),
		},
		Vocabulary: storedVocabulary(),
		StatePaths: []string{capacityStatePath},
	}, nil
}

// storedVocabulary is every value this package can write into a stored column
// or JSON field. It is built from the constants themselves, so renaming one
// changes nothing while changing its value moves the vocabulary.
//
// The rule is directional: a serving image must understand every value a
// candidate can write, because it will read them after a rollback. Dropping a
// value is therefore safe and adding one is a transition.
func storedVocabulary() []string {
	values := []string{
		// Claim identity columns.
		OwnerKindResource, OwnerKindScenario, OwnerKindOp, OwnerUnknown,
		ResourceKindVRAM, ResourceKindRAM, ResourceKindCPU,
		StatusReserved, StatusGranted, StatusDegraded, StatusReleased, StatusExpired, StatusPreempted,
		ActivityActive, ActivityIdle,
		// Measured-footprint provenance.
		FootprintSourceManifestDefault, FootprintSourceMeasured,
		// Residency degrade profiles.
		ResidencyBackendCPU, ResidencyBackendCUDA, ResidencyBackendMetal,
		ResidencyBackendROCm, ResidencyBackendVulkan, ResidencyBackendUnloaded,
		// Disk job classes, delta bases and attribution owners.
		DiskJobClassMetadata, diskJobActualDeltaBasis,
		ownerOllama, ownerReranker, ownerWhisper,
		// Policy values and their priority classes.
		EnforceOff, EnforceOn, EnforceAdvisory,
		AccelReprobeOff, AccelReprobeReport, AccelReprobeRestart,
		policyInteractive, policyService, policyBatch,
		sweepCursorKey,
		// Postures, verdicts, findings and placement evidence written into
		// records or reported back through stored policy.
		CapacityPostureBalanced, CapacityPostureMinimal, CapacityPostureResponsive, CapacityPostureThroughput,
		VerdictGrant, VerdictDegrade, VerdictQueue, VerdictDeny,
		FindingClaimed, FindingUnclaimed, FindingOverClaim, FindingDeclaredUnclaimed,
		FitVerdictFits, FitVerdictOverSubscribed,
		PlacementEvidenceQualified, PlacementEvidenceFailed, PlacementEvidenceMissing,
		PlacementEvidenceMismatched, PlacementEvidenceStale,
		ActionWarn, ActionPreempt, ActionRequestDegrade, ActionRequestUpshift, ActionRequestResidencyUnload,
	}
	// Policy keys are the stored key vocabulary of capacity_policy.
	values = append(values, PolicyKeys...)
	return stateformat.CanonicalVocabulary(values...)
}

// notStoredConstants are string constants that never reach stored state: SQL
// text and environment-variable names. TestVocabularyCoversEveryConstant keeps
// this list closed, so a new constant is either vocabulary or an explicit,
// reviewed exception.
var notStoredConstants = map[string]bool{
	"schemaSQL":                   true,
	"claimSelectSQL":              true,
	"addPlacementGenerationSQL":   true,
	"placementGenerationIndexSQL": true,
	"addReservedPeakSQL":          true,
	"addExceededPeakSQL":          true,
	"EnvEnforce":                  true,
	// This file's own declarations: the pinned digest and the store's durable
	// path are compatibility metadata, not values written into the ledger.
	"capacitySchemaDigest": true,
	"capacityStatePath":    true,
}
