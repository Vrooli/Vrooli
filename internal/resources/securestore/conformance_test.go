package securestore

import (
	"errors"
	"strings"
	"testing"
)

// One suite, every adapter. A platform-specific adapter that quietly answered a
// missing key with "backend broken" instead of ErrNotFound would recreate the
// exact defect this work removed, on a platform nobody is looking at.
//
// newStore returns a fresh handle to the platform backend. skipReason, when
// non-empty, explains why this host has no backend to test against; the
// platform test asserts the reason so a silent skip cannot masquerade as a
// pass.
// multiLinePEM is shaped like the value that corrupted a real operator's login
// keyring: a PKCS#8 block whose newlines are load-bearing to the format. The
// key material is not real.
const multiLinePEM = `-----BEGIN PRIVATE KEY-----
MIIG/QIBADANBgkqhkiG9w0BAQEFAASCBucwggbjAgEAAoIBgQCcX0/9ykvEELDI
3nSpbXHrdIHb82ZmFwbGVob2xkZXJub3RhcmVhbGtleXBsYWNlaG9sZGVybm90YQ
-----END PRIVATE KEY-----
`

func runConformance(t *testing.T, newStore func() Store) {
	t.Helper()

	const service = "vrooli.securestore.conformance"

	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "ascii value", key: "conformance-ascii", value: "plain-value"},
		{name: "unicode value", key: "conformance-unicode", value: "pässwörd-🔐-日本語"},
		{name: "4 KiB value", key: "conformance-large", value: strings.Repeat("k", 4096)},
		// The incident case. A PEM private key is multi-line by construction,
		// and a backend that writes one line per value wrote the newlines
		// verbatim, corrupting the whole keyring file rather than the one
		// entry. Every adapter must round-trip this, whether it does so
		// natively or through the encoding guard.
		{name: "multi-line PEM value", key: "conformance-pem", value: multiLinePEM},
		// Bare CR and CRLF are the same defect on a backend with different
		// line endings, and are easy to miss when only "\n" is guarded.
		{name: "CRLF value", key: "conformance-crlf", value: "first\r\nsecond\rthird"},
		// A caller storing text that looks like the encoding marker must get
		// its own bytes back, not a decode of them.
		{name: "value resembling the encoding marker", key: "conformance-marker", value: valueEncodingPrefix + "bm90LWVuY29kZWQ="},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newStore()
			t.Cleanup(func() { _ = store.Delete(service, testCase.key) })

			// get-before-put: the backend must say "no value", not "broken".
			if _, err := store.Get(service, testCase.key); !errors.Is(err, ErrNotFound) {
				t.Fatalf("Get before Put error = %v, want ErrNotFound", err)
			}

			if err := store.Put(service, testCase.key, testCase.value); err != nil {
				t.Fatalf("Put: %v", err)
			}
			got, err := store.Get(service, testCase.key)
			if err != nil {
				t.Fatalf("Get after Put: %v", err)
			}
			if got != testCase.value {
				t.Fatalf("Get returned %d bytes, want the %d bytes that were stored", len(got), len(testCase.value))
			}

			// overwrite: Put on an existing key replaces rather than duplicates.
			overwritten := testCase.value + "-overwritten"
			if err := store.Put(service, testCase.key, overwritten); err != nil {
				t.Fatalf("Put overwrite: %v", err)
			}
			got, err = store.Get(service, testCase.key)
			if err != nil {
				t.Fatalf("Get after overwrite: %v", err)
			}
			if got != overwritten {
				t.Fatalf("overwrite did not replace the stored value (%d bytes back, wanted %d)", len(got), len(overwritten))
			}

			if err := store.Delete(service, testCase.key); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			if _, err := store.Get(service, testCase.key); !errors.Is(err, ErrNotFound) {
				t.Fatalf("Get after Delete error = %v, want ErrNotFound", err)
			}

			// Delete is idempotent: removing what is already gone is success.
			if err := store.Delete(service, testCase.key); err != nil {
				t.Fatalf("Delete of an absent key = %v, want nil", err)
			}
		})
	}

	t.Run("probe reads a reachable backend", func(t *testing.T) {
		if err := Probe(newStore()); err != nil {
			t.Fatalf("Probe() = %v, want nil against a working backend", err)
		}
	})

	t.Run("write probe round-trips", func(t *testing.T) {
		if err := ProbeWritable(newStore()); err != nil {
			t.Fatalf("ProbeWritable() = %v, want nil against a working backend", err)
		}
	})
}

// conformanceSkipReason reports why this host cannot run the suite, or "" when
// it can. Asserting the reason is what keeps a skip from passing as a pass.
func conformanceSkipReason() string {
	diagnosis := Diagnose()
	if diagnosis.Available {
		return ""
	}
	return diagnosis.Condition + ": " + diagnosis.Explanation
}

// TestPlatformAdapterConformance runs the shared suite against whatever this
// host's Default() returns. It skips only with an explanation, and the
// explanation itself is asserted.
func TestPlatformAdapterConformance(t *testing.T) {
	if reason := conformanceSkipReason(); reason != "" {
		if strings.TrimSpace(reason) == "" || strings.HasPrefix(reason, ":") {
			t.Fatalf("adapter is unusable and gave no reason: %q", reason)
		}
		if !strings.HasPrefix(reason, "absent: ") && !strings.HasPrefix(reason, "unavailable: ") {
			t.Fatalf("skip reason %q does not name a known provider condition", reason)
		}
		t.Skipf("no usable credential backend on this runner — %s", reason)
	}
	runConformance(t, Default)
}
