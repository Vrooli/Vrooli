package blobstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
)

// newTestStore returns a Store rooted at a fresh temp dir. The
// RootOverride ensures the test never touches the real ClassData root.
func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()

	root := t.TempDir()
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli-test",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		t.Fatalf("storage.NewResolver: %v", err)
	}

	// Wire the resolver to our test root by injecting RootOverride
	// via a wrapper Store: the public NewResolver doesn't take
	// RootOverride, but Resolver.Resolve does — so we override at
	// path-resolution time by configuring the env var the resolver
	// reads.
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(root, "runtime"))

	store, err := New(resolver)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return store, root
}

func newSandboxID(t *testing.T) string {
	t.Helper()
	return uuid.New().String()
}

func TestPutAndGetRoundtrip(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)

	cases := []struct {
		name    string
		content []byte
	}{
		{"empty", []byte{}},
		{"one_byte", []byte{0x00}},
		{"ascii", []byte("the quick brown fox jumps over the lazy dog")},
		{"binary", bytes.Repeat([]byte{0xAA, 0x55, 0xFF, 0x00}, 1024)},
		{"large_compressible", bytes.Repeat([]byte("AAAAAAAAAA"), 100_000)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := store.Put(ctx, sandboxID, tc.content)
			if err != nil {
				t.Fatalf("Put: %v", err)
			}
			expSum := sha256.Sum256(tc.content)
			if res.SHA256Hex != hex.EncodeToString(expSum[:]) {
				t.Errorf("hash = %q, want %q", res.SHA256Hex, hex.EncodeToString(expSum[:]))
			}
			if res.SizeUncompressed != int64(len(tc.content)) {
				t.Errorf("uncompressed size = %d, want %d", res.SizeUncompressed, len(tc.content))
			}
			if res.SizeOnDisk <= 0 && len(tc.content) > 0 {
				t.Errorf("on-disk size <= 0 for non-empty content")
			}

			got, err := store.Get(ctx, sandboxID, res.SHA256Hex)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if !bytes.Equal(got, tc.content) {
				t.Errorf("Get returned %d bytes; want %d (head mismatch=%v)",
					len(got), len(tc.content), !bytes.Equal(got[:min(len(got), 16)], tc.content[:min(len(tc.content), 16)]))
			}
		})
	}
}

func TestPutIsContentAddressedAndIdempotent(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)
	content := []byte("idempotency content")

	r1, err := store.Put(ctx, sandboxID, content)
	if err != nil {
		t.Fatalf("Put #1: %v", err)
	}
	r2, err := store.Put(ctx, sandboxID, content)
	if err != nil {
		t.Fatalf("Put #2: %v", err)
	}
	if r1.SHA256Hex != r2.SHA256Hex {
		t.Errorf("idempotent Put produced different hashes: %s vs %s", r1.SHA256Hex, r2.SHA256Hex)
	}
	if r1.SizeOnDisk != r2.SizeOnDisk {
		t.Errorf("idempotent Put produced different sizes: %d vs %d", r1.SizeOnDisk, r2.SizeOnDisk)
	}

	// Sanity: the content survives the rewrite.
	got, err := store.Get(ctx, sandboxID, r1.SHA256Hex)
	if err != nil || !bytes.Equal(got, content) {
		t.Errorf("Get after idempotent Put: got=%q err=%v", got, err)
	}
}

func TestPutRejectsInvalidSandboxID(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	bad := []string{
		"",
		"not-a-uuid",
		"../etc/passwd",
		"00000000-0000-0000-0000-00000000000z", // bad hex char
		"00000000_0000_0000_0000_000000000000", // wrong separator
	}
	for _, id := range bad {
		t.Run(id, func(t *testing.T) {
			if _, err := store.Put(ctx, id, []byte("x")); !errors.Is(err, ErrInvalidSandboxID) {
				t.Errorf("Put(%q) error = %v, want ErrInvalidSandboxID", id, err)
			}
			if _, err := store.Get(ctx, id, "00"+padHex(62)); !errors.Is(err, ErrInvalidSandboxID) {
				t.Errorf("Get(%q) error = %v, want ErrInvalidSandboxID", id, err)
			}
			if err := store.DeleteSandbox(ctx, id); !errors.Is(err, ErrInvalidSandboxID) {
				t.Errorf("DeleteSandbox(%q) error = %v, want ErrInvalidSandboxID", id, err)
			}
		})
	}
}

func TestGetRejectsInvalidHash(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)

	bad := []string{
		"",
		"deadbeef", // too short
		"DEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF", // uppercase
		"../passwd",
		"zz" + padHex(62), // non-hex char
	}
	for _, h := range bad {
		t.Run(h, func(t *testing.T) {
			if _, err := store.Get(ctx, sandboxID, h); !errors.Is(err, ErrInvalidHash) {
				t.Errorf("Get hash %q error = %v, want ErrInvalidHash", h, err)
			}
			if _, _, err := store.Stat(ctx, sandboxID, h); !errors.Is(err, ErrInvalidHash) {
				t.Errorf("Stat hash %q error = %v, want ErrInvalidHash", h, err)
			}
		})
	}
}

func TestGetMissingBlobReturnsErrNotFound(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)

	missing := padHex(64)
	_, err := store.Get(ctx, sandboxID, missing)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get on missing blob: err = %v, want ErrNotFound", err)
	}
	size, exists, err := store.Stat(ctx, sandboxID, missing)
	if err != nil || exists || size != 0 {
		t.Errorf("Stat on missing blob = (%d, %v, %v); want (0, false, nil)", size, exists, err)
	}
}

func TestStatReportsSizeForExistingBlob(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)
	content := []byte("stat me")

	res, err := store.Put(ctx, sandboxID, content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	size, exists, err := store.Stat(ctx, sandboxID, res.SHA256Hex)
	if err != nil || !exists {
		t.Fatalf("Stat: size=%d exists=%v err=%v", size, exists, err)
	}
	if size != res.SizeOnDisk {
		t.Errorf("Stat size = %d; want %d", size, res.SizeOnDisk)
	}
}

func TestDeleteSandboxRemovesAllBlobs(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)

	hashes := []string{}
	for _, c := range [][]byte{[]byte("a"), []byte("bb"), []byte("ccc")} {
		r, err := store.Put(ctx, sandboxID, c)
		if err != nil {
			t.Fatalf("Put: %v", err)
		}
		hashes = append(hashes, r.SHA256Hex)
	}

	if err := store.DeleteSandbox(ctx, sandboxID); err != nil {
		t.Fatalf("DeleteSandbox: %v", err)
	}

	for _, h := range hashes {
		if _, err := store.Get(ctx, sandboxID, h); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%s) after DeleteSandbox: err = %v, want ErrNotFound", h, err)
		}
	}

	// Idempotent: deleting again is a no-op.
	if err := store.DeleteSandbox(ctx, sandboxID); err != nil {
		t.Errorf("DeleteSandbox (second call): %v", err)
	}
}

func TestDeleteSandboxScopedToSandbox(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	a := newSandboxID(t)
	b := newSandboxID(t)
	rA, _ := store.Put(ctx, a, []byte("a-content"))
	rB, _ := store.Put(ctx, b, []byte("b-content"))

	if err := store.DeleteSandbox(ctx, a); err != nil {
		t.Fatalf("DeleteSandbox(a): %v", err)
	}

	if _, err := store.Get(ctx, a, rA.SHA256Hex); !errors.Is(err, ErrNotFound) {
		t.Errorf("a's blob still present: err = %v", err)
	}
	got, err := store.Get(ctx, b, rB.SHA256Hex)
	if err != nil || string(got) != "b-content" {
		t.Errorf("b's blob wiped by DeleteSandbox(a): got=%q err=%v", got, err)
	}
}

func TestPutConcurrentSameContentSucceeds(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)
	content := []byte("concurrent content")

	const N = 16
	var wg sync.WaitGroup
	hashes := make([]string, N)
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := store.Put(ctx, sandboxID, content)
			hashes[i] = r.SHA256Hex
			errs[i] = err
		}(i)
	}
	wg.Wait()
	for i := 0; i < N; i++ {
		if errs[i] != nil {
			t.Errorf("Put #%d: %v", i, errs[i])
		}
		if i > 0 && hashes[i] != hashes[0] {
			t.Errorf("Put #%d hash %q != #0 hash %q", i, hashes[i], hashes[0])
		}
	}
	got, err := store.Get(ctx, sandboxID, hashes[0])
	if err != nil || !bytes.Equal(got, content) {
		t.Errorf("Get after concurrent Put: got=%q err=%v", got, err)
	}
}

func TestPutConcurrentDifferentContent(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)

	const N = 16
	var wg sync.WaitGroup
	hashes := make([]string, N)
	contents := make([][]byte, N)
	for i := 0; i < N; i++ {
		contents[i] = []byte("payload-" + string(rune('A'+i)))
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := store.Put(ctx, sandboxID, contents[i])
			if err != nil {
				t.Errorf("Put #%d: %v", i, err)
				return
			}
			hashes[i] = r.SHA256Hex
		}(i)
	}
	wg.Wait()
	for i := 0; i < N; i++ {
		got, err := store.Get(ctx, sandboxID, hashes[i])
		if err != nil || !bytes.Equal(got, contents[i]) {
			t.Errorf("blob #%d roundtrip: got=%q err=%v want=%q", i, got, err, contents[i])
		}
	}
}

func TestGetDetectsCorruptedBlob(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)

	content := []byte("trust but verify")
	r, err := store.Put(ctx, sandboxID, content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Overwrite the on-disk blob with a gzipped payload whose
	// uncompressed bytes hash to something different. The reader
	// must detect the mismatch.
	tampered := []byte("not the same content at all")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(tampered); err != nil {
		t.Fatalf("gzip: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	path, err := store.blobPath(sandboxID, r.SHA256Hex)
	if err != nil {
		t.Fatalf("blobPath: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("overwrite blob: %v", err)
	}

	if _, err := store.Get(ctx, sandboxID, r.SHA256Hex); err == nil {
		t.Errorf("Get on corrupted blob returned nil error; expected hash-mismatch failure")
	}
}

func TestGetDetectsTruncatedGzip(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	sandboxID := newSandboxID(t)

	r, err := store.Put(ctx, sandboxID, []byte("the gzip header is fine; body truncated"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	path, _ := store.blobPath(sandboxID, r.SHA256Hex)
	full, err := os.ReadFile(path) // #nosec G304 -- test path
	if err != nil {
		t.Fatalf("readfile: %v", err)
	}
	// Truncate to half.
	if err := os.WriteFile(path, full[:len(full)/2], 0o644); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := store.Get(ctx, sandboxID, r.SHA256Hex); err == nil {
		t.Errorf("Get on truncated blob returned nil error; expected gzip read failure")
	}
}

func TestNewRejectsNilResolver(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Errorf("New(nil) returned nil error; want validation error")
	}
}

// padHex returns a string of n lowercase hex zeros.
func padHex(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = '0'
	}
	return string(out)
}

// Sanity check that gzip.Reader returns io.EOF cleanly on an empty
// blob — just to document expected behavior of the underlying lib.
func TestEmptyGzipRoundtrip(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if err := gw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	gr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("reader: %v", err)
	}
	out, err := io.ReadAll(gr)
	if err != nil || len(out) != 0 {
		t.Fatalf("empty gzip readall: out=%v err=%v", out, err)
	}
}
