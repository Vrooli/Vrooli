package securestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

// MigrationEntry identifies one credential without carrying its value. The
// caller owns the inventory (normally the manifest-backed credential
// inventory); securestore only moves values that it can read and verify.
type MigrationEntry struct {
	Service string `json:"service"`
	Key     string `json:"key"`
}

// MigrationReceipt is safe to persist or print: it names no credential value.
// A failed migration never commits the backend selection.
type MigrationReceipt struct {
	From      string   `json:"from"`
	To        string   `json:"to"`
	Attempted []string `json:"attempted"`
	Verified  []string `json:"verified"`
	Failed    []string `json:"failed,omitempty"`
	Committed bool     `json:"committed"`
}

var (
	nativeStoreForSelectionFn    = func() Store { return guardValues(nativeDefault()) }
	encryptedStoreForSelectionFn = func() Store { return guardValues(defaultEncryptedStore()) }
)

const (
	backendSelectionVersion = 1
	backendSelectionFile    = "credential-backend.json"
)

// BackendSelection is the setup-time authority decision for this installation.
// It contains no credential material. Once written, transient native-store
// outages do not change the authority and cannot split credentials across two
// backends.
type BackendSelection struct {
	Version    int       `json:"version"`
	Backend    string    `json:"backend"`
	SelectedAt time.Time `json:"selected_at"`
	Reason     string    `json:"reason,omitempty"`
}

var backendSelectionPath = func() (string, error) {
	return config.VrooliPath(repocontract.HomeKeyState, backendSelectionFile)
}

// SelectedBackend returns the persisted setup decision. A missing file means
// this is an older installation that has not passed through setup's backend
// decision yet; callers may use the legacy discovery behavior for that one
// transition state.
func SelectedBackend() (string, bool, error) {
	path, err := backendSelectionPath()
	if err != nil {
		return "", false, fmt.Errorf("resolve credential backend selection path: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read credential backend selection: %w", err)
	}
	var selection BackendSelection
	if err := json.Unmarshal(data, &selection); err != nil {
		return "", false, fmt.Errorf("decode credential backend selection: %w", err)
	}
	if selection.Version != backendSelectionVersion {
		return "", false, fmt.Errorf("credential backend selection version %d is unsupported", selection.Version)
	}
	if err := validateBackend(selection.Backend); err != nil {
		return "", false, fmt.Errorf("validate credential backend selection: %w", err)
	}
	return selection.Backend, true, nil
}

// SelectBackend records the single backend this installation uses. It is an
// explicit setup operation, not a fallback triggered by an individual read or
// write failure.
func SelectBackend(backend, reason string) error {
	if err := validateBackend(backend); err != nil {
		return err
	}
	path, err := backendSelectionPath()
	if err != nil {
		return err
	}
	selection := BackendSelection{
		Version:    backendSelectionVersion,
		Backend:    backend,
		SelectedAt: time.Now().UTC(),
		Reason:     strings.TrimSpace(reason),
	}
	data, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential backend selection: %w", err)
	}
	if err := config.WriteOwnedFile(path, append(data, '\n'), tuning.PermSecret); err != nil {
		return fmt.Errorf("write credential backend selection: %w", err)
	}
	return nil
}

// ReselectBackend re-diagnoses the native authority and, when it is a better
// writable destination than the persisted fallback, migrates values with a
// copy-verify-then-commit protocol. The selection file is the final write.
// Existing values in the destination are never overwritten with a different
// value, and values written during a failed migration are rolled back when
// they did not exist before the attempt.
//
//nolint:gocyclo // backend reselection handles migration ordering, capability, and per-entry failure outcomes.
func ReselectBackend(entries []MigrationEntry) (MigrationReceipt, error) {
	current, found, err := SelectedBackend()
	if err != nil {
		return MigrationReceipt{}, err
	}
	if !found {
		return MigrationReceipt{}, fmt.Errorf("credential backend has not been selected; run setup first")
	}
	native := diagnoseNativeForSetupFn()
	target := BackendEncryptedFile
	if native.Writable {
		target = BackendNative
	}
	receipt := MigrationReceipt{From: current, To: target}
	if current == target {
		receipt.Committed = true
		return receipt, nil
	}
	// A native authority can only be a migration source when it is readable.
	// Do not turn a transient keyring outage into a split store.
	if current == BackendNative && !native.Available {
		return receipt, fmt.Errorf("cannot reselect credential backend: native authority is unavailable: %s", native.Explanation)
	}
	source := storeForSelectedBackend(current)
	destination := storeForSelectedBackend(target)
	if err := Probe(source); err != nil {
		return receipt, fmt.Errorf("read source credential backend %q: %w", current, err)
	}
	if err := ProbeWritable(destination); err != nil {
		return receipt, fmt.Errorf("destination credential backend %q is not writable: %w", target, err)
	}
	created := []MigrationEntry{}
	for _, entry := range entries {
		service, key := strings.TrimSpace(entry.Service), strings.TrimSpace(entry.Key)
		if service == "" || key == "" {
			continue
		}
		value, getErr := source.Get(service, key)
		if errors.Is(getErr, ErrNotFound) {
			continue
		}
		name := service + "/" + key
		receipt.Attempted = append(receipt.Attempted, name)
		if getErr != nil {
			receipt.Failed = append(receipt.Failed, name)
			rollbackMigration(destination, created)
			return receipt, fmt.Errorf("read %s: %w", name, getErr)
		}
		before, beforeErr := destination.Get(service, key)
		if beforeErr == nil && before != value {
			receipt.Failed = append(receipt.Failed, name)
			rollbackMigration(destination, created)
			return receipt, fmt.Errorf("destination already contains a different value for %s", name)
		}
		if beforeErr != nil && !errors.Is(beforeErr, ErrNotFound) {
			receipt.Failed = append(receipt.Failed, name)
			rollbackMigration(destination, created)
			return receipt, fmt.Errorf("inspect destination %s: %w", name, beforeErr)
		}
		if beforeErr != nil {
			if err := destination.Put(service, key, value); err != nil {
				receipt.Failed = append(receipt.Failed, name)
				rollbackMigration(destination, created)
				return receipt, fmt.Errorf("write destination %s: %w", name, err)
			}
			created = append(created, MigrationEntry{Service: service, Key: key})
		}
		verified, verifyErr := destination.Get(service, key)
		if verifyErr != nil || verified != value {
			receipt.Failed = append(receipt.Failed, name)
			rollbackMigration(destination, created)
			if verifyErr != nil {
				return receipt, fmt.Errorf("verify destination %s: %w", name, verifyErr)
			}
			return receipt, fmt.Errorf("verify destination %s: readback did not match", name)
		}
		receipt.Verified = append(receipt.Verified, name)
	}
	if err := SelectBackend(target, "reselected after operator-identity diagnosis and verified migration"); err != nil {
		rollbackMigration(destination, created)
		return receipt, fmt.Errorf("commit credential backend selection: %w", err)
	}
	receipt.Committed = true
	return receipt, nil
}

// RetireEmptyBackend removes only an initialized encrypted fallback that is
// demonstrably empty and is no longer selected. Native stores are owned by the
// operating system and are never deleted by Vrooli.
func RetireEmptyBackend(backend string) error {
	if err := validateBackend(backend); err != nil {
		return err
	}
	selected, found, err := SelectedBackend()
	if err != nil {
		return err
	}
	if found && selected == backend {
		return fmt.Errorf("cannot retire the selected credential backend %q", backend)
	}
	if backend != BackendEncryptedFile {
		return fmt.Errorf("cannot retire native credential storage; it is owned by the operating system")
	}
	status, err := DescribeStore()
	if err != nil {
		return err
	}
	if status.Entries != 0 {
		return fmt.Errorf("cannot retire encrypted credential store: it still contains %d credential(s)", status.Entries)
	}
	path, err := credentialStorePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("retire empty encrypted credential store: %w", err)
	}
	return nil
}

func storeForSelectedBackend(backend string) Store {
	if backend == BackendNative {
		return nativeStoreForSelectionFn()
	}
	return encryptedStoreForSelectionFn()
}

func rollbackMigration(destination Store, entries []MigrationEntry) {
	for _, entry := range entries {
		_ = destination.Delete(entry.Service, entry.Key)
	}
}

func validateBackend(backend string) error {
	switch strings.TrimSpace(backend) {
	case BackendNative, BackendEncryptedFile:
		return nil
	default:
		return fmt.Errorf("unsupported credential backend %q; use %q or %q", backend, BackendNative, BackendEncryptedFile)
	}
}
