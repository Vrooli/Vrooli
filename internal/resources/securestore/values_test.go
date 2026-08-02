package securestore

import (
	"errors"
	"strings"
	"testing"
)

// recordingStore captures what actually reached the backend. The guard's whole
// job is to control that boundary, so asserting on the caller's round-trip
// alone would pass even if the raw newlines went straight through.
type recordingStore struct {
	values map[string]string
	name   string
}

func newRecordingStore() *recordingStore {
	return &recordingStore{values: map[string]string{}, name: "recording"}
}

func (store *recordingStore) Put(service, key, value string) error {
	store.values[service+"/"+key] = value
	return nil
}

func (store *recordingStore) Get(service, key string) (string, error) {
	value, ok := store.values[service+"/"+key]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

func (store *recordingStore) Delete(service, key string) error {
	delete(store.values, service+"/"+key)
	return nil
}

func (store *recordingStore) AdapterName() string { return store.name }

// TestGuardNeverStoresUnsafeBytes is the regression test for the incident: a
// PEM handed to a backend that stores one line per value corrupted the entire
// keyring file. What matters is not that the caller gets their bytes back, but
// that the bytes reaching the backend are single-line.
func TestGuardNeverStoresUnsafeBytes(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "PEM private key", value: multiLinePEM},
		{name: "LF only", value: "one\ntwo"},
		{name: "CRLF", value: "one\r\ntwo"},
		{name: "bare CR", value: "one\rtwo"},
		{name: "embedded NUL", value: "one\x00two"},
		{name: "trailing newline", value: "value\n"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			inner := newRecordingStore()
			guarded := guardValues(inner)

			if err := guarded.Put("svc", "key", testCase.value); err != nil {
				t.Fatalf("Put: %v", err)
			}

			stored := inner.values["svc/key"]
			if strings.ContainsAny(stored, unsafeValueBytes) {
				t.Fatalf("guard handed the backend a value containing unsafe bytes (%d bytes)", len(stored))
			}
			if !strings.HasPrefix(stored, valueEncodingPrefix) {
				t.Fatalf("stored value is not marked as encoded")
			}

			got, err := guarded.Get("svc", "key")
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got != testCase.value {
				t.Fatalf("round-trip returned %d bytes, want the %d stored", len(got), len(testCase.value))
			}
		})
	}
}

// TestGuardLeavesSingleLineValuesUntouched is what keeps this change compatible
// with every credential already sitting in an operator's keyring. Those were
// written raw; encoding them on read would be a migration nobody asked for, and
// re-encoding them on write would make the keyring unreadable in Seahorse for
// no defect prevented.
func TestGuardLeavesSingleLineValuesUntouched(t *testing.T) {
	values := []string{
		"plain-value",
		"pässwörd-🔐-日本語",
		strings.Repeat("k", 4096),
		"has\ttab",
		"  leading and trailing  ",
		"",
	}

	for _, value := range values {
		inner := newRecordingStore()
		guarded := guardValues(inner)

		if err := guarded.Put("svc", "key", value); err != nil {
			t.Fatalf("Put(%q): %v", value, err)
		}
		if stored := inner.values["svc/key"]; stored != value {
			t.Fatalf("single-line value was rewritten on the way to the backend: got %d bytes, want %d", len(stored), len(value))
		}

		got, err := guarded.Get("svc", "key")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got != value {
			t.Fatalf("round-trip changed a single-line value")
		}
	}
}

// TestGuardReadsLegacyRawValues covers the values written before the guard
// existed. They carry no marker and must come back exactly as stored rather
// than being sniffed for base64-ness — a generated API token is frequently
// valid base64, and decoding one would hand back garbage far from here.
func TestGuardReadsLegacyRawValues(t *testing.T) {
	inner := newRecordingStore()
	// Written directly to the backend, as a pre-guard Vrooli would have.
	legacy := "dGhpcyBpcyB2YWxpZCBiYXNlNjQ="
	if err := inner.Put("svc", "legacy", legacy); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := guardValues(inner).Get("svc", "legacy")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != legacy {
		t.Fatalf("legacy raw value came back as %q, want %q", got, legacy)
	}
}

// TestGuardDisambiguatesTheMarker closes the hole where a caller stores text
// that looks like the guard's own output. Without encoding it, the value would
// be written raw and decoded on the way out, returning bytes never stored.
func TestGuardDisambiguatesTheMarker(t *testing.T) {
	inner := newRecordingStore()
	guarded := guardValues(inner)

	value := valueEncodingPrefix + "bm90LWVuY29kZWQ="
	if err := guarded.Put("svc", "key", value); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := guarded.Get("svc", "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != value {
		t.Fatalf("marker-shaped value came back as %q, want %q", got, value)
	}
}

// TestGuardReportsCorruptEncodedPayload prefers a loud error over a silently
// shorter secret when a marked value will not decode.
func TestGuardReportsCorruptEncodedPayload(t *testing.T) {
	inner := newRecordingStore()
	if err := inner.Put("svc", "key", valueEncodingPrefix+"not!valid!base64"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := guardValues(inner).Get("svc", "key")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get on a corrupt encoded payload = %v, want ErrUnavailable", err)
	}
}

// TestGuardPreservesBackendErrors keeps the three transport conditions distinct
// through the wrapper. Collapsing ErrNotFound into a generic failure is the
// defect the Store contract exists to prevent.
func TestGuardPreservesBackendErrors(t *testing.T) {
	guarded := guardValues(failingStore{err: ErrNotFound})
	if _, err := guarded.Get("svc", "key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get = %v, want ErrNotFound to survive the guard", err)
	}

	guarded = guardValues(failingStore{err: ErrAbsent})
	if _, err := guarded.Get("svc", "key"); !errors.Is(err, ErrAbsent) {
		t.Fatalf("Get = %v, want ErrAbsent to survive the guard", err)
	}
}

// TestGuardForwardsAdapterName keeps the guard invisible in diagnostics. An
// operator reading "libsecret" must keep reading "libsecret".
func TestGuardForwardsAdapterName(t *testing.T) {
	inner := newRecordingStore()
	inner.name = "libsecret"
	if got := AdapterName(guardValues(inner)); got != "libsecret" {
		t.Fatalf("AdapterName through the guard = %q, want %q", got, "libsecret")
	}
}
