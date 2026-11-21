package runtimehandlers

import (
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

// rootUpdateStore resolves the compiled state authority for the status view.
// Status reads are not part of the prepared-root recovery compatibility owner.
func rootUpdateStore(home string) (*coreactivation.Store, error) {
	if buildinfo.RuntimeHomeSnapshotBase64 == "" {
		return coreactivation.NewStore(home)
	}
	state, err := buildinfo.CompiledRuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, err
	}
	return coreactivation.NewStoreAt(home, filepath.Join(state, "core-activation"))
}

func readRootUpdateStatus(home string) (coreactivation.Enrollment, bool, coreactivation.Record, bool, error) {
	store, err := rootUpdateStore(home)
	if err != nil {
		return coreactivation.Enrollment{}, false, coreactivation.Record{}, false, err
	}
	enrollment, enrolled, err := store.Enrollment()
	if err != nil {
		return coreactivation.Enrollment{}, false, coreactivation.Record{}, false, err
	}
	record, exists, err := store.Current()
	if err != nil {
		return coreactivation.Enrollment{}, false, coreactivation.Record{}, false, err
	}
	return enrollment, enrolled, record, exists, nil
}
