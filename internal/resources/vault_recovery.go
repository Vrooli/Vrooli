package resources

import (
	"path/filepath"

	"github.com/vrooli/vrooli/internal/tuning"

	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	vaultbootstrap "github.com/vrooli/vrooli/packages/vaultbootstrap-go"
)

// VaultUnsealKeyEntry names one live instance's unseal key by its durable
// credential address, so a recovery inventory can capture it.
type VaultUnsealKeyEntry struct {
	InstanceID string `json:"instance_id"`
	LogicalID  string `json:"logical_id"`
	Field      string `json:"field"`
}

// VaultUnsealKeyEntries lists the unseal keys this host holds.
//
// This is the second inventory source recovery needs. Declared credentials are
// inventoried from manifests, which is the right rule for anything an author
// wrote down — but a managed instance's ID is generated at runtime, so no
// manifest can name it. Without this, `recovery export --all` walks only
// declarations and silently omits the one piece of material whose loss is
// unrecoverable: lose an unseal key and the instance stays sealed forever.
//
// It returns addresses, never values. The caller resolves through the
// credential authority, which is the only thing permitted to read a secret.
func VaultUnsealKeyEntries(broker *Broker) []VaultUnsealKeyEntry {
	if broker == nil {
		return nil
	}
	instances := broker.InstancesForResource("vault")
	entries := make([]VaultUnsealKeyEntry, 0, len(instances))
	for _, instance := range instances {
		identity, err := vaultbootstrap.UnsealKeyIdentity(instance.ID)
		if err != nil {
			// An instance whose ID cannot form an identity has nothing stored
			// under one either, so there is nothing to capture and nothing to
			// warn about.
			continue
		}
		entries = append(entries, VaultUnsealKeyEntry{
			InstanceID: instance.ID,
			LogicalID:  string(identity),
			Field:      vaultbootstrap.UnsealKeyField,
		})
	}
	return entries
}

// LiveVaultUnsealKeyEntries reads the host's persisted broker state and reports
// the unseal keys it holds.
//
// It is read-only and best-effort: a host with no broker state has no managed
// Vault, which is a normal answer and not a fault. Returning nil there keeps a
// credential inventory usable on a machine that has never run one.
func LiveVaultUnsealKeyEntries() []VaultUnsealKeyEntry {
	resolver, err := resourceStorageResolver()
	if err != nil {
		return nil
	}
	paths, err := runtimestorage.EnsureAllDirs(resolver, runtimestorage.Options{ResourceID: userResourceHostID}, tuning.PermPrivateDir)
	if err != nil {
		return nil
	}
	broker, err := NewPersistentBroker(nil, FileBrokerStore{Path: filepath.Join(paths.StateDir, "broker.json")})
	if err != nil {
		return nil
	}
	return VaultUnsealKeyEntries(broker)
}
