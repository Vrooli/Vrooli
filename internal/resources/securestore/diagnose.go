package securestore

import (
	"errors"
	"runtime"
)

// Diagnosis is the operator-facing account of this host's credential backend.
// It carries no credential value and no key name, so it is safe to print in
// full anywhere.
type Diagnosis struct {
	Platform string `json:"platform"`
	Adapter  string `json:"adapter"`
	// Backend is the store family holding the values — a native adapter, or the
	// encrypted file store on a host that has none. An operator who does not
	// know which store their values are in cannot reason about anything else.
	Backend string `json:"backend"`
	// KeyWrap names the key-encryption provider currently holding the store
	// open, and KeyStore what actually protects that wrap. They are empty for a
	// native backend, which has no wraps. The pair is reported rather than
	// summarized because the wraps are not equally strong: a TPM resists
	// possession of the disk and a host key on the same disk does not.
	KeyWrap  string `json:"key_wrap,omitempty"`
	KeyStore string `json:"key_store,omitempty"`
	// Available is true when a read reached the backend.
	Available bool `json:"available"`
	// Condition is "available", "unavailable", or "absent".
	Condition string `json:"condition"`
	// Explanation names the host condition in operator terms — a uid/session
	// mismatch, a headless host with no Secret Service, a missing adapter.
	// Empty when the backend is healthy.
	Explanation string `json:"explanation,omitempty"`
	// Fix is the concrete action that makes the backend work, when one exists.
	Fix string `json:"fix,omitempty"`
	// SessionRepair records a host condition the credential path corrected on
	// its own. It is reported even when the backend is healthy, because the
	// login that produced the condition usually degrades other session-scoped
	// tools too, and those cannot correct themselves.
	SessionRepair string `json:"session_repair,omitempty"`
	// Unlocked reports whether the encrypted store currently holds an open data
	// key. It is meaningless for a native backend and for a store opened by the
	// host-bound wrap, which never needs an unlock.
	Unlocked bool `json:"unlocked,omitempty"`
}

// Diagnose reports what this host's credential backend is and why it is not
// working, if it is not. It never fails: a diagnosis that could error would be
// useless in exactly the situation it exists for.
func Diagnose() Diagnosis {
	store := Default()
	diagnosis := Diagnosis{
		Platform:      runtime.GOOS,
		Condition:     "available",
		Available:     true,
		SessionRepair: sessionRepairNote(),
	}
	err := Probe(store)
	// Every backend field is read after the probe, never before. The probe is
	// what decides the chain and what opens a wrap, so reading them first would
	// describe the host as it looked before anything had looked at it.
	diagnosis.Adapter = AdapterName(store)
	diagnosis.Backend = backendName(store)
	diagnosis.KeyWrap, diagnosis.KeyStore = activeWrap(store)
	diagnosis.Unlocked = diagnosis.KeyWrap != ""
	if err == nil {
		return diagnosis
	}
	diagnosis.Available = false
	if errors.Is(err, ErrAbsent) {
		diagnosis.Condition = "absent"
		diagnosis.Explanation = err.Error()
		diagnosis.Fix = absentBackendFixFor(store)
		return diagnosis
	}
	diagnosis.Condition = "unavailable"
	diagnosis.Explanation = err.Error()
	if session := sessionDiagnosis(); session != "" {
		diagnosis.Fix = session
	}
	return diagnosis
}

// backendName reports the store family without the wrap detail that
// AdapterName appends, so a caller can group hosts by backend.
func backendName(store Store) string {
	switch typed := store.(type) {
	case singleLineStore:
		return backendName(typed.inner)
	case *chainStore:
		return backendName(typed.active())
	case *encryptedStore:
		return adapterEncryptedFile
	default:
		return AdapterName(store)
	}
}

// absentBackendFixFor names the action that reaches a working state. On a host
// where the encrypted store is the active backend, installing a desktop keyring
// is not that action — initializing the store is, and telling a headless server
// operator to install gnome-keyring is exactly the dead end this work removes.
func absentBackendFixFor(store Store) string {
	encrypted, ok := encryptedBackend(store)
	if !ok || encrypted.initialized() {
		return absentBackendFix()
	}
	// Only promise the unattended path on a host that can actually take it.
	// Telling an operator a TPM opens the store with no further action, on a
	// host where the TPM cannot be opened, is how someone plans an unattended
	// reboot that will never unlock.
	if blocked := hostBoundFix(); blocked != "" {
		return "run `vrooli credentials store init` and supply a passphrase on stdin. " +
			"The unattended host-bound wrap is NOT usable on this host as it stands: " + blocked
	}
	return "run `vrooli credentials store init` to create an encrypted credential store on this host; " +
		"a host with a TPM opens it with no further action, otherwise supply a passphrase on stdin"
}
