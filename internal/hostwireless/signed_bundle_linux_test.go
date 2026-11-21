//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
	"golang.org/x/sys/unix"
)

func signedBundleFixture(t *testing.T) (signedBundleInputs, string, string, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("SUDO_USER", "")
	spec := preparationProfiles[NobleBluetoothProfile]
	stage := filepath.Join(root, "mt7927-noble-source", "pinned")
	candidate := filepath.Join(root, "candidate")
	unsigned := filepath.Join(root, "host-wireless", NobleBluetoothProfile, "abcdef")
	signed := filepath.Join(root, "signed")
	for _, d := range []string{stage, candidate, filepath.Dir(unsigned), signed} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeFixture(t, filepath.Join(stage, mt7927source.Manifest), "reviewed source")
	expected := Receipt{SchemaVersion: 1, Profile: NobleBluetoothProfile, KernelRelease: nobleABI, SourceDirectory: stage,
		SourceSHA256: digest([]byte("reviewed source")), PatchSHA256: digest(spec.patch), BuildRecipeSHA256: digest(spec.recipe),
		Modaliases: []string{"fixture"}, CopiedTreeRollback: true, StockABI: ABIProof{StockExports: 14, Imports: 198}, CandidateABI: ABIProof{StockExports: 14, Imports: 199}}
	cert, key := testCertificate(t, root, "bundle")
	for _, name := range spec.modules {
		payload := []byte("reviewed " + name)
		writeFixture(t, filepath.Join(candidate, name+".ko"), string(payload))
		writeFixture(t, filepath.Join(signed, name+".ko"), string(appendModuleSignature(payload, signFixture(t, root, cert, key, payload))))
	}
	if _, err := publishCandidate(context.Background(), expected, candidate, unsigned); err != nil {
		t.Fatal(err)
	}
	files, sha, err := verifyPreparedBundle(context.Background(), expected, spec, "abcdef")
	if err != nil {
		t.Fatal(err)
	}
	return signedBundleInputs{expected, files, sha}, signed, cert, key
}

func fixtureSignedReservation(context.Context, string, int64) (func() error, error) {
	return func() error { return nil }, nil
}

func TestSignedBundlePublicationRetainsAuthenticatedClosure(t *testing.T) {
	inputs, signed, cert, _ := signedBundleFixture(t)
	ctx := context.Background()
	r, err := publishSignedBundle(ctx, inputs, "abcdef", signed, cert, "supplied-signed-copies", fixtureSignedReservation)
	if err != nil {
		t.Fatal(err)
	}
	if !validSHA(r.ReceiptSHA256) || r.Directory == "" || r.PublicCertificate == "" || r.ModulesDirectory == "" || !r.SignaturesVerified || r.KernelTrustVerified || r.ActivationEligible {
		t.Fatalf("invalid evidence %+v", r)
	}
	proof, err := verifyPublishedSignedBundle(ctx, inputs, "abcdef", r.ReceiptSHA256)
	if err != nil || proof.Directory != r.Directory {
		t.Fatalf("reopen %+v %v", proof, err)
	}
	retry, err := publishSignedBundle(ctx, inputs, "abcdef", signed, cert, "supplied-signed-copies", func(context.Context, string, int64) (func() error, error) {
		t.Error("retry allocated disk despite intact publication")
		return nil, errors.New("capacity unavailable")
	})
	if err != nil || retry.ReceiptSHA256 != r.ReceiptSHA256 {
		t.Fatalf("retry %+v %v", retry, err)
	}
	// Canonical preparation still validates, and public source may disappear.
	if _, _, err := verifyPreparedBundle(ctx, inputs.expected, preparationProfiles[inputs.expected.Profile], "abcdef"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(signed); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyPublishedSignedBundle(ctx, inputs, "abcdef", r.ReceiptSHA256); err != nil {
		t.Fatal(err)
	}
	for _, m := range r.Modules {
		file := filepath.Join(r.ModulesDirectory, m.Name)
		if err := os.Chmod(file, 0600); err != nil {
			t.Fatal(err)
		}
		writeFixture(t, file, "foreign bytes")
		if _, err := verifyPublishedSignedBundle(ctx, inputs, "abcdef", r.ReceiptSHA256); err == nil {
			t.Fatal("accepted changed signed publication")
		}
		break
	}
}

func TestSignedBundleRejectsWrongCertificateSourceRaceAndCapacity(t *testing.T) {
	inputs, signed, cert, _ := signedBundleFixture(t)
	ctx := context.Background()
	wrong, _ := testCertificate(t, t.TempDir(), "wrong")
	if _, err := publishSignedBundle(ctx, inputs, "abcdef", signed, wrong, "supplied-signed-copies", fixtureSignedReservation); err == nil {
		t.Fatal("accepted wrong signer")
	}
	denied := errors.New("fixture capacity denied")
	reserve := func(context.Context, string, int64) (func() error, error) { return nil, denied }
	if _, err := publishSignedBundle(ctx, inputs, "abcdef", signed, cert, "supplied-signed-copies", reserve); !errors.Is(err, denied) {
		t.Fatalf("capacity denial %v", err)
	}
	released := false
	reserve = func(context.Context, string, int64) (func() error, error) {
		writeFixture(t, filepath.Join(signed, inputs.files[0].Path), "foreign source")
		return func() error { released = true; return nil }, nil
	}
	if _, err := publishSignedBundle(ctx, inputs, "abcdef", signed, cert, "supplied-signed-copies", reserve); err == nil {
		t.Fatal("accepted source substitution")
	}
	if !released {
		t.Fatal("disk reservation leaked")
	}
	entries, err := os.ReadDir(signedBundleParent(inputs.expected))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("partial publication remained: %v", entries)
	}
}

func TestSignedBundleCancellationAndConcurrentRetry(t *testing.T) {
	inputs, signed, cert, _ := signedBundleFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	released := false
	reserve := func(context.Context, string, int64) (func() error, error) {
		cancel()
		return func() error { released = true; return nil }, nil
	}
	if _, err := publishSignedBundle(ctx, inputs, "abcdef", signed, cert, "supplied-signed-copies", reserve); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel result %v", err)
	}
	if !released {
		t.Fatal("cancel leaked reservation")
	}
	entries, err := os.ReadDir(signedBundleParent(inputs.expected))
	if err != nil || len(entries) != 0 {
		t.Fatalf("cancel left partial output %v %v", entries, err)
	}
	var wg sync.WaitGroup
	ids := make(chan string, 4)
	errs := make(chan error, 4)
	var held atomic.Int32
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reserve := func(context.Context, string, int64) (func() error, error) {
				if held.Add(1) != 1 {
					t.Error("concurrent capacity section despite artifact lock")
				}
				return func() error { held.Add(-1); return nil }, nil
			}
			r, e := publishSignedBundle(context.Background(), inputs, "abcdef", signed, cert, "supplied-signed-copies", reserve)
			ids <- r.ReceiptSHA256
			errs <- e
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first || !validSHA(id) {
			t.Errorf("publication identities diverged %s %s", first, id)
		}
	}
	entries, err = os.ReadDir(signedBundleParent(inputs.expected))
	if err != nil || len(entries) != 1 {
		t.Fatalf("retry output %v %v", entries, err)
	}
}

func TestSignedBundleRejectsExtraFilesCertificateAndReceiptDrift(t *testing.T) {
	inputs, signed, cert, _ := signedBundleFixture(t)
	r, err := publishSignedBundle(context.Background(), inputs, "abcdef", signed, cert, "supplied-signed-copies", fixtureSignedReservation)
	if err != nil {
		t.Fatal(err)
	}
	wrong, _ := testCertificate(t, t.TempDir(), "wrong")
	wrongBytes, err := os.ReadFile(wrong)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(r.PublicCertificate, 0600); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, r.PublicCertificate, string(wrongBytes))
	if _, err = verifyPublishedSignedBundle(context.Background(), inputs, "abcdef", r.ReceiptSHA256); err == nil {
		t.Fatal("accepted changed retained certificate")
	}
	// Retry must preserve, rather than overwrite, the damaged immutable artifact.
	if _, err = publishSignedBundle(context.Background(), inputs, "abcdef", signed, cert, "supplied-signed-copies", fixtureSignedReservation); err == nil {
		t.Fatal("retry overwrote damaged artifact")
	}
	originalCert, e := readPublicCertificate(cert)
	if e != nil {
		t.Fatal(e)
	}
	writeFixture(t, r.PublicCertificate, string(originalCert.Raw))
	extra := filepath.Join(r.Directory, "foreign.json")
	writeFixture(t, extra, "unexpected")
	if _, e = verifyPublishedSignedBundle(context.Background(), inputs, "abcdef", r.ReceiptSHA256); e == nil {
		t.Fatal("accepted extra closure entry")
	}
	if e = os.Remove(extra); e != nil {
		t.Fatal(e)
	}
	receiptPath := filepath.Join(r.Directory, "receipt.json")
	if e = os.Chmod(receiptPath, 0600); e != nil {
		t.Fatal(e)
	}
	writeFixture(t, receiptPath, "changed receipt")
	if _, e = verifyPublishedSignedBundle(context.Background(), inputs, "abcdef", r.ReceiptSHA256); e == nil {
		t.Fatal("accepted changed immutable receipt")
	}

	if _, err = verifyPublishedSignedBundle(context.Background(), inputs, "abcdef", "../escape"); err == nil {
		t.Fatal("accepted traversal")
	}
}

func TestSignedBundleParentRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "redirect")); err != nil {
		t.Fatal(err)
	}
	if _, err := openSignedBundleParent(filepath.Join(root, "redirect", "artifact")); err == nil {
		t.Fatal("publisher followed parent symlink")
	}
	if _, err := os.Stat(filepath.Join(outside, "artifact")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("publisher wrote outside canonical parent: %v", err)
	}
}

func TestSignedBundleEphemeralSigningUsesExactUnsignedPayload(t *testing.T) {
	inputs, _, cert, key := signedBundleFixture(t)
	unsigned := filepath.Join(filepath.Dir(filepath.Dir(inputs.expected.SourceDirectory)), "host-wireless", inputs.expected.Profile, "abcdef")
	source, err := os.OpenRoot(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	parsed, err := readPublicCertificate(cert)
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	signed, err := stageAndSignPrepared(context.Background(), inputs, source, work, parsed.Raw, func(ctx context.Context, module, public string) error {
		payload, e := os.ReadFile(module)
		if e != nil {
			return e
		}
		sig := signFixture(t, t.TempDir(), cert, key, payload)
		return os.WriteFile(module, appendModuleSignature(payload, sig), 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = publishSignedBundle(context.Background(), inputs, "abcdef", signed, filepath.Join(work, "certificate.der"), "installed-exact-abi-sign-file-existing-mok", fixtureSignedReservation); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(unsigned, inputs.files[0].Path), "foreign unsigned module")
	_, err = stageAndSignPrepared(context.Background(), inputs, source, t.TempDir(), parsed.Raw, func(context.Context, string, string) error { t.Fatal("signed changed unsigned bytes"); return nil })
	if err == nil || !strings.Contains(err.Error(), "source changed") {
		t.Fatalf("changed unsigned source result %v", err)
	}
}

// Opt-in native installed-tool fixture. It uses only a newly generated ephemeral
// test key; neither the host MOK key nor trust enrollment is accessed.
func TestSignedBundleInstalledSignFileEphemeralFixture(t *testing.T) {
	if os.Getenv("VROOLI_WIRELESS_SIGN_FILE_TEST") != "1" {
		t.Skip("explicit installed sign-file fixture not enabled")
	}
	inputs, _, cert, key := signedBundleFixture(t)
	signerPath := filepath.Join("/usr/src", "linux-headers-"+nobleABI, "scripts", "sign-file")
	signer, err := openProtectedSignerInput(signerPath, true)
	if err != nil {
		t.Fatal(err)
	}
	defer signer.Close()
	keyFD, err := unix.Open(key, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	ephemeral := os.NewFile(uintptr(keyFD), key)
	defer ephemeral.Close()
	parsed, err := readPublicCertificate(cert)
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(filepath.Dir(filepath.Dir(inputs.expected.SourceDirectory)), "host-wireless", inputs.expected.Profile, "abcdef")
	source, err := os.OpenRoot(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	work := t.TempDir()
	signed, err := stageAndSignPrepared(context.Background(), inputs, source, work, parsed.Raw, func(ctx context.Context, module, public string) error {
		return runInstalledModuleSigner(ctx, signer, ephemeral, module, public)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = publishSignedBundle(context.Background(), inputs, "abcdef", signed, filepath.Join(work, "certificate.der"), "installed-exact-abi-sign-file-existing-mok", fixtureSignedReservation); err != nil {
		t.Fatal(err)
	}
}

func TestSignedBundleSigningRequiresExplicitElevation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root boundary fixture applies to unprivileged caller")
	}
	if _, err := SignPreparedBundle(context.Background(), NobleBluetoothProfile, "abcdef", "/not-read", digest([]byte("reviewed"))); err == nil || !strings.Contains(err.Error(), "explicitly elevated") {
		t.Fatalf("signing elevation guard %v", err)
	}
}

func TestSignedBundlePrivilegedSigningPinsReviewedReceipt(t *testing.T) {
	inputs, _, _, _ := signedBundleFixture(t)
	for _, reviewed := range []string{"", inputs.receiptSHA[:63], digest([]byte("other receipt"))} {
		if err := requireReviewedSigningReceipt(inputs, reviewed); err == nil {
			t.Fatal("accepted missing or changed reviewed receipt pin")
		}
	}
	if err := requireReviewedSigningReceipt(inputs, inputs.receiptSHA); err != nil {
		t.Fatal(err)
	}
}
