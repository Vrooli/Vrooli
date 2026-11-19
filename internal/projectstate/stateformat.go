package projectstate

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// projectSetupSchemaDigest pins the stored shape of format 1: the migration
// ledger, the completion marker and the phase strings they record. The digest
// is derived from the types, so a stored field change moves it. A deliberate
// change must bump migrationLedgerSchemaVersion and re-pin.
const projectSetupSchemaDigest = "2479957f55eed64a33ec2ed269145e0e334d30cd55b9c81a1c29f4659902616c"

// Project state is scoped by project key, so the files exist once per project
// and their directory names are not known when this binary is compiled. The
// glob families below mirror Locator's methods one-for-one.
const (
	projectSetupLedgerPattern   = "state/projects/*/setup/" + migrationLedgerName
	projectSetupCompletePattern = "state/projects/*/setup/.*-complete"
	projectSetupPopulatedGlob   = "state/projects/*/setup/.*-populated"
)

// StateFormat describes project-scoped setup state as a versioned store.
//
// The stored vocabulary is the set of migration statuses and the completion
// phase: a prior reader that does not know a status cannot interpret a ledger a
// candidate writes, so adding one is a transition.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[MigrationLedger](),
		reflect.TypeFor[ConfigurationCompletion](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "project-setup",
		Write: stateformat.Version(migrationLedgerSchemaVersion),
		Reads: []stateformat.Version{stateformat.Version(migrationLedgerSchemaVersion)},
		Schemas: map[stateformat.Version]string{
			stateformat.Version(migrationLedgerSchemaVersion): digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			configurationCompletePhase,
			string(MigrationPending), string(MigrationRunning), string(MigrationComplete), string(MigrationFailed), string(MigrationInterrupted),
		),
		StatePaths: []string{
			projectSetupLedgerPattern,
			projectSetupCompletePattern,
			projectSetupPopulatedGlob,
		},
	}, nil
}
