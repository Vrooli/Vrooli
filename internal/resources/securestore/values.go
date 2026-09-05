package securestore

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// A credential value is opaque bytes to this package, but it is not opaque to
// every backend that stores it. GNOME Keyring writes an unencrypted keyring in
// GKeyFile's textual format, where a value lives on exactly one line. Handed a
// PEM private key — which is ~26 lines by construction — it wrote the newlines
// verbatim and produced a keyring file that GKeyFile could no longer parse.
//
// The blast radius is what makes this worth a guard rather than a caller-side
// fix. GNOME Keyring does not skip the malformed entry; it rejects the entire
// file, taking every unrelated secret in it down at the next login. On the host
// that motivated this code, one Vrooli-written PEM cost the operator all 32
// stored credentials, including the RDP password that let them reach the
// machine at all.
//
// So the rule is a property of the seam, not of the caller: a value that leaves
// this package for a backend is single-line. Callers keep passing whatever
// bytes they hold and read the same bytes back.

// valueEncodingPrefix marks a stored value that was encoded on the way in. It
// is versioned because the encoding is now part of the on-disk contract with
// every keyring Vrooli has ever written to, and a future change has to be able
// to tell its own output apart from this one.
const valueEncodingPrefix = "vrooli-b64:v1:"

// unsafeValueBytes are the bytes that cost a backend its ability to parse its
// own store. Newline is the one that caused the incident; carriage return is
// the same defect on a backend with different line endings; NUL truncates any
// value that reaches a C string, which every native adapter here does.
//
// Tab is deliberately absent. GKeyFile escapes it losslessly, and widening the
// trigger would encode values that store perfectly well today, making every
// keyring less legible to the operator reading it in Seahorse for no defect
// prevented.
const unsafeValueBytes = "\n\r\x00"

// needsValueEncoding reports whether a value must be encoded before storage.
//
// The prefix clause is not redundant. Without it, a caller storing the literal
// text "vrooli-b64:v1:abc" would have it written raw and then decoded on the
// way out, returning bytes that were never stored. Encoding any value that
// could be mistaken for an encoded one makes the marker mean exactly one thing:
// a raw stored value provably does not carry it.
func needsValueEncoding(value string) bool {
	return strings.ContainsAny(value, unsafeValueBytes) ||
		strings.HasPrefix(value, valueEncodingPrefix)
}

// encodeValue returns the form of a value that is safe to hand to a backend.
//
// Values that are already single-line pass through untouched, which is what
// keeps this change compatible with every credential Vrooli stored before it:
// those values were written raw, they read back raw, and no migration has to
// run against an operator's keyring to make them legible again.
func encodeValue(value string) string {
	if !needsValueEncoding(value) {
		return value
	}
	return valueEncodingPrefix + base64.StdEncoding.EncodeToString([]byte(value))
}

// decodeValue returns the caller's original bytes for a stored value.
//
// An unmarked value is returned as-is rather than probed for base64-ness.
// Sniffing would misread any single-line secret that happens to be valid
// base64 — which a generated API token frequently is — and hand back decoded
// garbage that fails somewhere far from here.
func decodeValue(stored string) (string, error) {
	encoded, marked := strings.CutPrefix(stored, valueEncodingPrefix)
	if !marked {
		return stored, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// The marker is ours, so a payload that will not decode means the
		// stored value was truncated or edited underneath us. Reporting that as
		// a corrupt store beats returning a shorter secret that looks fine.
		return "", fmt.Errorf("%w: stored value carries the %s marker but its payload is not valid base64",
			ErrUnavailable, valueEncodingPrefix)
	}
	return string(decoded), nil
}

// singleLineStore holds the single-line guarantee for whatever backend it
// wraps. It sits at the composition root rather than inside each adapter so
// that the guarantee cannot be lost by an adapter added later — the failure it
// prevents is silent at write time and only surfaces at the next login, which
// is far too late to catch in review.
type singleLineStore struct{ inner Store }

// guardValues wraps a backend so no multi-line value reaches it.
func guardValues(inner Store) Store { return singleLineStore{inner: inner} }

func (store singleLineStore) Put(service, key, value string) error {
	return store.inner.Put(service, key, encodeValue(value))
}

func (store singleLineStore) Get(service, key string) (string, error) {
	stored, err := store.inner.Get(service, key)
	if err != nil {
		return "", err
	}
	return decodeValue(stored)
}

func (store singleLineStore) Delete(service, key string) error {
	return store.inner.Delete(service, key)
}

// AdapterName forwards the wrapped backend's label. The guard is not a backend
// and must not appear in diagnostics as one; an operator reading "libsecret"
// needs to keep reading "libsecret".
func (store singleLineStore) AdapterName() string { return AdapterName(store.inner) }
