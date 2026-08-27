package securestore

import (
	"errors"
	"runtime"
)

const (
	diagnoseAbsent      = "absent"
	diagnoseUnavailable = "unavailable"
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
	// Condition is "available", diagnoseUnavailable, or diagnoseAbsent.
	Condition string `json:"condition"`
	// Explanation names the host condition in operator terms — a uid/session
	// mismatch, a headless host with no Secret Service, a missing adapter.
	// Empty when the backend is healthy.
	Explanation string `json:"explanation,omitempty"`
	// Fix is the concrete action that makes the backend work, when one exists.
	Fix string `json:"fix,omitempty"`
	// NativeStorageStrength describes protection outside the process boundary.
	NativeStorageStrength string `json:"native_storage_strength,omitempty"`
	NativeStorageCaveat   string `json:"native_storage_caveat,omitempty"`
	// NativeWrap reports the build/runtime availability of the unattended
	// platform wrap even when the encrypted store is not initialized yet. This
	// keeps a CGO-free macOS binary or unavailable Windows DPAPI from looking
	// healthy merely because no native wrap has been attempted.
	NativeWrap string `json:"native_wrap,omitempty"`
	// Writable is true only when the diagnostic throwaway write, readback, and
	// delete all succeeded. It is intentionally separate from Available: a
	// backend can answer reads while rejecting every provisioning write.
	Writable         bool   `json:"writable"`
	WriteCondition   string `json:"write_condition"`
	WriteExplanation string `json:"write_explanation,omitempty"`
	WriteFix         string `json:"write_fix,omitempty"`
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

// Remediable is an error that names the operator action which clears it.
//
// The remedy travels with the error rather than being reassembled by the
// diagnosis, because only the layer that detected a condition knows what
// resolves it. A stale keyring daemon needs a fresh login; a uid/session
// mismatch needs a corrected session. A caller that had to infer one from the
// other's error text would eventually print the wrong one — which is exactly
// what happened: `Fix` fell back to the already-applied session-repair note,
// so the field labelled "here is what to do" described a non-problem while the
// real remedy sat buried in the explanation.
type Remediable interface {
	Remediation() string
}

// remediableError pairs a condition with the action that clears it.
type remediableError struct {
	err    error
	remedy string
}

func (e remediableError) Error() string       { return e.err.Error() }
func (e remediableError) Unwrap() error       { return e.err }
func (e remediableError) Remediation() string { return e.remedy }

// withRemediation attaches an operator action to an error.
func withRemediation(err error, remedy string) error {
	if err == nil {
		return nil
	}
	return remediableError{err: err, remedy: remedy}
}

// remediationFor returns the action that clears an error, or "" when nothing
// in the chain claims to know one. Returning empty is deliberate: printing no
// fix is better than printing a guess.
func remediationFor(err error) string {
	var typed Remediable
	if errors.As(err, &typed) {
		return typed.Remediation()
	}
	return ""
}

// WriteConditionNotChecked is the write condition of a read-only diagnosis. It
// is distinct from diagnoseUnavailable on purpose: not knowing whether a store can be
// written is not the same as knowing it cannot.
const WriteConditionNotChecked = "not-checked"

// Diagnose reports what this host's credential backend is and why it is not
// working, if it is not. It never fails: a diagnosis that could error would be
// useless in exactly the situation it exists for.
//
// It is read-only. The write probe stores, reads back, and deletes a throwaway
// value in the operator's real credential store, which is defensible before
// writing durable recovery material and indefensible in a diagnostic: an
// operator runs `doctor` precisely when the store is already misbehaving, and a
// write is what makes GNOME Keyring raise the unlock prompt nobody is there to
// answer. It also costs the full Secret Service timeout on a host in that
// state, so the command that explains a hang was itself hanging.
//
// Use DiagnoseWritable when the question really is "can I provision right now".
func Diagnose() Diagnosis {
	return diagnoseStore(Default(), false)
}

// DiagnoseWritable is Diagnose plus proof that a value can actually be stored.
// It writes to the real backend, so it belongs to callers whose next step is a
// write — onboarding asking whether the operator can provision, a preflight
// before a recovery restore — and not to a routine health read.
func DiagnoseWritable() Diagnosis {
	return diagnoseStore(Default(), true)
}

// DiagnoseNativeWritable checks the platform adapter itself, without applying
// the persisted installation choice. Setup uses this to choose an authority
// once; normal credential operations must use DiagnoseWritable so they honor
// that choice and never drift between backends.
func DiagnoseNativeWritable() Diagnosis {
	return diagnoseStore(nativeDefault(), true)
}

func diagnoseStore(store Store, checkWrites bool) Diagnosis {
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
	diagnosis.NativeWrap = nativeWrapDiagnosis()
	if diagnosis.Backend == "libsecret" {
		diagnosis.NativeStorageStrength, diagnosis.NativeStorageCaveat = nativeStorageStrength()
	}
	diagnosis.KeyWrap, diagnosis.KeyStore = activeWrap(store)
	diagnosis.Unlocked = diagnosis.KeyWrap != ""
	if err == nil {
		diagnosis.WriteCondition = "checking"
	} else {
		diagnosis.Available = false
		if errors.Is(err, ErrAbsent) {
			diagnosis.Condition = diagnoseAbsent
			diagnosis.Explanation = err.Error()
			diagnosis.Fix = absentBackendFixFor(store)
		} else {
			diagnosis.Condition = diagnoseUnavailable
			diagnosis.Explanation = err.Error()
			// The condition that was actually detected names its own remedy
			// first. Only when nothing does do we fall back to the session
			// diagnosis, which is a guess about a different layer.
			//
			// SessionRepair is deliberately NOT a fallback here. It records a
			// correction Vrooli already made successfully, so presenting it as
			// the fix tells an operator to go and do something that is both
			// already done and unrelated to what is currently broken. It stays
			// visible in its own field.
			if remedy := remediationFor(err); remedy != "" {
				diagnosis.Fix = remedy
			} else if locked := lockedStoreFix(store); locked != "" {
				diagnosis.Fix = locked
			} else if session := sessionDiagnosis(); session != "" {
				diagnosis.Fix = session
			}
		}
	}
	if errors.Is(err, ErrAbsent) {
		diagnosis.WriteCondition = diagnoseAbsent
		diagnosis.WriteExplanation = "the backend is absent, so a write probe cannot run"
		diagnosis.WriteFix = diagnosis.Fix
		return diagnosis
	}
	if !checkWrites {
		diagnosis.WriteCondition = WriteConditionNotChecked
		diagnosis.WriteExplanation = "no write was attempted; this diagnosis did not touch the store"
		diagnosis.WriteFix = "run `vrooli credentials doctor --check-writes` to prove a credential can be stored"
		return diagnosis
	}
	writeErr := ProbeWritable(store)
	if writeErr == nil {
		diagnosis.Writable, diagnosis.WriteCondition = true, "available"
		return diagnosis
	}
	diagnosis.WriteCondition = conditionFor(writeErr)
	diagnosis.WriteExplanation = writeErr.Error()
	// The write failure names its own remedy when it has one; otherwise the
	// read-side fix is the best available. Appending an empty fix used to
	// produce a sentence ending in a semicolon and nothing.
	diagnosis.WriteFix = "no credential can be provisioned on this host now"
	if remedy := remediationFor(writeErr); remedy != "" {
		diagnosis.WriteFix += "; " + remedy
	} else if diagnosis.Fix != "" {
		diagnosis.WriteFix += "; " + diagnosis.Fix
	}
	return diagnosis
}

func conditionFor(err error) string {
	switch {
	case errors.Is(err, ErrAbsent):
		return diagnoseAbsent
	case errors.Is(err, ErrUnavailable):
		return diagnoseUnavailable
	default:
		return diagnoseUnavailable
	}
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

// lockedStoreFix names the action for the condition an encrypted store reports
// most often and that nothing else here explains: the file is present and
// intact, and no wrap on this host opened it. That is neither a broken host nor
// a missing backend — it is a store waiting for the passphrase it was sealed
// with. Leaving it without a fix is how an operator ends up reading "secure
// storage is unavailable" and concluding the backend failed.
//
// Where the host could hold an unattended wrap and does not, that is the more
// useful answer, because it addresses why the store was asking at all rather
// than only what to type this time.
func lockedStoreFix(store Store) string {
	if backendName(store) != adapterEncryptedFile {
		return ""
	}
	encrypted, ok := encryptedBackend(store)
	if !ok || !encrypted.initialized() {
		return ""
	}
	const unlock = "supply the store passphrase with `vrooli credentials store unlock`, which reads it from stdin"
	if blocked := hostBoundFix(); blocked != "" {
		return unlock + ". This host is asking at all because it has no unattended wrap: " + blocked
	}
	return unlock + ", or run `vrooli setup` to add an unattended wrap so this host opens the store after a reboot with no passphrase at all"
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
