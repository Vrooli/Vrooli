package resources

import (
	"path/filepath"

	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

// KopiaRepositoryEntry names one repository passphrase in the credential
// authority. It carries no passphrase value.
type KopiaRepositoryEntry struct {
	Repository string `json:"repository"`
	LogicalID  string `json:"logical_id"`
	Field      string `json:"field"`
}

// KopiaRepositoryEntries reads the non-secret registry and returns one
// recoverable credential address per registered repository.
func KopiaRepositoryEntries(registryPath string) ([]KopiaRepositoryEntry, error) {
	entries, err := kopiaregistry.New(registryPath).Load()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	out := make([]KopiaRepositoryEntry, 0, len(entries))
	for _, entry := range entries {
		identity, identityErr := kopiaregistry.PassphraseIdentity(entry.Name)
		if identityErr != nil {
			return nil, identityErr
		}
		out = append(out, KopiaRepositoryEntry{
			Repository: entry.Name,
			LogicalID:  string(identity),
			Field:      kopiaregistry.PassphraseField,
		})
	}
	return out, nil
}

// LiveKopiaRepositoryEntries reads the host's registered repository inventory.
// A missing or unreadable storage root is treated as an empty live inventory;
// doctor remains useful on a host that has never run resource-kopia.
func LiveKopiaRepositoryEntries() []KopiaRepositoryEntry {
	paths, err := resourceStoragePaths("kopia")
	if err != nil {
		return nil
	}
	entries, err := KopiaRepositoryEntries(filepath.Join(paths.StateDir, "registry.json"))
	if err != nil {
		return nil
	}
	return entries
}
