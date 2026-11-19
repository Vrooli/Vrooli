package cloudtarget

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// cloudTargetSchemaDigest pins the stored shapes of a deployment's control
// records: the active-release pointer, the activation intent, the fence and the
// per-step operation receipts. A deliberate change must bump the store's write
// version and re-pin.
const cloudTargetSchemaDigest = "2aa360f9784fa5106f97846476a3928c5ccd44837b2448c6dceb9bcd588b0982"

// The deployment control records live under cloud/deployments/<id>/. Release
// trees beside them are deployed artefacts, not rollback state, so they are
// deliberately excluded: restoring them would duplicate the deployment's own
// install transaction.
const (
	cloudTargetControlPattern   = "cloud/deployments/*/*.json"
	cloudTargetOperationPattern = "cloud/deployments/*/operations/*/*.json"
)

// StateFormat describes a cloud deployment's control records as a versioned
// store.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[ActiveRelease](),
		reflect.TypeFor[ActivationIntent](),
		reflect.TypeFor[Receipt](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:    "cloud-target",
		Write:   1,
		Reads:   []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{1: digest},
		Vocabulary: stateformat.CanonicalVocabulary(
			CodeActionNotAllowed,
			CodeActivationFailed,
			CodeArchiveEntryLimit,
			CodeArchiveSymlinkEscape,
			CodeArchiveTimeBudget,
			CodeArchiveTooLarge,
			CodeArchiveTraversal,
			CodeArchiveUnreadable,
			CodeArchiveUnsupportedEntry,
			CodeBrokerUnavailable,
			CodeBundleSHA256Mismatch,
			CodeCredentialAuthorityFailed,
			CodeCredentialNotIngested,
			CodeCredentialPayloadInvalid,
			CodeCredentialStoreLocked,
			CodeCredentialVersionMismatch,
			CodeCredentialVersionRevoked,
			CodeCredentialVersionStale,
			CodeDataBindingConflict,
			CodeEdgeConfigInvalid,
			CodeEdgePrivateListener,
			CodeEdgeReloadFailed,
			CodeEdgeRollbackNotEligible,
			CodeEdgeSnippetOutOfScope,
			CodeEdgeSpecInvalid,
			CodeEdgeUnrelatedChanged,
			CodeFenceStale,
			CodeHostActionFailed,
			CodeInvalidArgument,
			CodeManifestInvalid,
			CodeNativeCLIMismatch,
			CodeReceiptInputMismatch,
			CodeReceiptNotFound,
			CodeReleaseDigestMismatch,
			CodeReleaseHasNoScenarios,
			CodeReleaseIncomplete,
			CodeReleaseNotStaged,
			CodeRollbackNotEligible,
			CodeStageFailed,
			CodeStoreIO,
			string(OutcomeFailed),
			string(OutcomeSucceeded),
			string(OutcomeUnchanged),
			PostgresCredentialIdentity,
			string(ReleaseStateComplete),
			string(ReleaseStateIncomplete),
			string(ReleaseStateMissing),
			string(ReleaseStateStaging),
			string(StrategyMaintenance),
			string(StrategySideBySide),
		),
		StatePaths: []string{cloudTargetControlPattern, cloudTargetOperationPattern},
	}, nil
}
