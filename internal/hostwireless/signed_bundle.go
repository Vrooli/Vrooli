package hostwireless

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/packages/capacity"
)

// SignedBundleReceipt describes an immutable, content-addressed publication.
// PublicCertificate and ModulesDirectory are retained canonical paths. Trust
// and activation are independent evidence, never inferred from a signature.
type SignedBundleReceipt struct {
	SchemaVersion int `json:"schema_version"`
	SignedCopyEvidence
	SigningMethod     string `json:"signing_method"`
	Directory         string `json:"directory,omitempty"`
	ModulesDirectory  string `json:"modules_directory,omitempty"`
	PublicCertificate string `json:"public_certificate,omitempty"`
	ReceiptSHA256     string `json:"receipt_sha256,omitempty"`
}

type signedBundleInputs struct {
	expected   Receipt
	files      []FileIdentity
	receiptSHA string
}

func inspectSignedBundleInputs(ctx context.Context, profile, generation string) (signedBundleInputs, error) {
	spec, ok := preparationProfiles[profile]
	if !ok || !bundleID.MatchString(generation) {
		return signedBundleInputs{}, errors.New("known profile and exact unsigned bundle generation required")
	}
	expected, err := Prepare(ctx, profile, false)
	if err != nil {
		return signedBundleInputs{}, err
	}
	if expected.State != "eligible" {
		return signedBundleInputs{}, fmt.Errorf("preparation is %s: %s", expected.State, expected.Reason)
	}
	files, sha, err := verifyPreparedBundle(ctx, expected, spec, generation)
	return signedBundleInputs{expected, files, sha}, err
}

// PublishSignedBundle authenticates existing signed modules and retains a closed
// public bundle under the invoking operator's canonical artifact root. It never
// reads a private key and can run unprivileged.
func PublishSignedBundle(ctx context.Context, profile, generation, signedDir, certPath string) (SignedBundleReceipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	inputs, err := inspectSignedBundleInputs(ctx, profile, generation)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	return publishSignedBundle(ctx, inputs, generation, signedDir, certPath, signedBundleSuppliedMethod, reserveSignedBundle)
}

type signedBundleReservation func(context.Context, string, int64) (func() error, error)

func reserveSignedBundle(ctx context.Context, root string, size int64) (func() error, error) {
	broker, err := capacity.NewBroker(ctx, "")
	if err != nil {
		return nil, err
	}
	lease, err := broker.ReserveDisk(ctx, "host-wireless-signed", root, size, 0)
	if err != nil {
		broker.Close()
		return nil, err
	}
	return func() error {
		defer broker.Close()
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return lease.Release(cleanup)
	}, nil
}

func signedBundleParent(expected Receipt) string {
	return filepath.Join(filepath.Dir(filepath.Dir(expected.SourceDirectory)), "host-wireless-signed", expected.Profile)
}

func publishSignedBundle(ctx context.Context, inputs signedBundleInputs, generation, signedDir, certPath, method string, reserve signedBundleReservation) (_ SignedBundleReceipt, resultErr error) {
	empty := SignedBundleReceipt{}
	cert, err := readPublicCertificate(certPath)
	if err != nil {
		return empty, err
	}
	info, err := os.Lstat(signedDir)
	if err != nil || !info.IsDir() {
		return empty, errors.New("signed copies require a regular directory")
	}
	source, err := os.OpenRoot(signedDir)
	if err != nil {
		return empty, err
	}
	defer source.Close()
	// Verify before allocating, then verify the private snapshot again. The
	// verifier's bounded no-follow reads prevent substituted links from escaping.
	modules, err := verifySignedFiles(ctx, bundleFiles{source}, inputs.files, cert)
	if err != nil {
		return empty, err
	}
	parent := signedBundleParent(inputs.expected)
	parentRoot, err := openSignedBundleParent(parent)
	if err != nil {
		return empty, err
	}
	defer parentRoot.Close()
	// Reuse the artifact owner lock, including invoking-user ownership handling.
	unlock, err := artifactlock.AcquireContext(ctx, "wireless-signed-"+inputs.expected.Profile)
	if err != nil {
		return empty, err
	}
	defer unlock()
	receipt := SignedBundleReceipt{SchemaVersion: 1, SigningMethod: method, SignedCopyEvidence: SignedCopyEvidence{
		Profile: inputs.expected.Profile, KernelRelease: inputs.expected.KernelRelease, Bundle: generation, BundleReceiptSHA256: inputs.receiptSHA,
		CertificateSHA256: digest(cert.Raw), Modules: modules, SignaturesVerified: true,
	}}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return empty, err
	}
	sha := digest(data)
	if _, err = parentRoot.Lstat(sha); err == nil {
		if err = syncSignedBundleDir(parentRoot, "."); err != nil {
			return empty, err
		}
		return verifyPublishedSignedBundle(ctx, inputs, generation, sha)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return empty, err
	}
	size := int64(256 << 10)
	for _, f := range inputs.files {
		i, e := source.Stat(f.Path)
		if e != nil {
			return empty, e
		}
		size += i.Size() + 8192
	}
	release, err := reserve(ctx, filepath.Dir(filepath.Dir(parent)), size)
	if err != nil {
		return empty, fmt.Errorf("reserve signed bundle disk: %w", err)
	}
	cleanupVerified := true
	defer func() {
		if cleanupVerified {
			resultErr = errors.Join(resultErr, release())
		}
	}()
	token := make([]byte, 16)
	if _, err = rand.Read(token); err != nil {
		return empty, err
	}
	work := fmt.Sprintf(".publish-%x", token)
	if err = parentRoot.Mkdir(work, 0700); err != nil {
		return empty, err
	}
	defer func() {
		if work != "" {
			cleanupErr := parentRoot.RemoveAll(work)
			cleanupVerified = cleanupErr == nil
			resultErr = errors.Join(resultErr, cleanupErr)
		}
	}()
	root, err := parentRoot.OpenRoot(work)
	if err != nil {
		return empty, err
	}
	defer root.Close()

	if err = root.Mkdir("modules", 0700); err != nil {
		return empty, err
	}
	for _, m := range modules {
		data, e := readSmallRegular(bundleFiles{source}, m.Name, maxSignedModuleBytes)
		if e != nil {
			return empty, e
		}
		if digest(data) != m.SignedSHA256 {
			return empty, errors.New("signed source changed during publication")
		}
		if e = writeSignedBundleFile(root, filepath.Join("modules", m.Name), data); e != nil {
			return empty, e
		}
	}
	if err = writeSignedBundleFile(root, "certificate.der", cert.Raw); err != nil {
		return empty, err
	}
	moduleRoot, err := root.OpenRoot("modules")
	if err != nil {
		return empty, err
	}
	_, err = verifySignedFiles(ctx, bundleFiles{moduleRoot}, inputs.files, cert)
	moduleRoot.Close()
	if err != nil {
		return empty, err
	}

	if err = writeSignedBundleFile(root, "receipt.json", data); err != nil {
		return empty, err
	}
	for _, name := range []string{"modules", "."} {
		if err = syncSignedBundleDir(root, name); err != nil {
			return empty, err
		}
	}
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	if _, err = parentRoot.Lstat(sha); err == nil {
		// Never replace a publication, even when it has been tampered with.
		if err = syncSignedBundleDir(parentRoot, "."); err != nil {
			return empty, err
		}
		return verifyPublishedSignedBundle(ctx, inputs, generation, sha)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return empty, err
	}
	// Hand the complete inode closure back to the invoking operator before the
	// rename. No key or root-private signing input is ever in this closure.
	if err = chownSignedBundleClosure(root, filepath.Join(parent, work)); err != nil {
		return empty, err
	}
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	if err = parentRoot.Rename(work, sha); err != nil {
		return empty, err
	}
	work = ""

	if err = syncSignedBundleDir(parentRoot, "."); err != nil {
		return empty, err
	}
	return verifyPublishedSignedBundle(ctx, inputs, generation, sha)
}

func writeSignedBundleFile(root *os.Root, name string, data []byte) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	return errors.Join(werr, f.Sync(), f.Close())
}
func syncSignedBundleDir(root *os.Root, name string) error {
	f, err := root.Open(name)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}
func locateSignedBundle(r SignedBundleReceipt, dir, sha string) SignedBundleReceipt {
	r.Directory = dir
	r.ModulesDirectory = filepath.Join(dir, "modules")
	r.PublicCertificate = filepath.Join(dir, "certificate.der")
	r.ReceiptSHA256 = sha
	return r
}

// VerifyPublishedSignedBundle reauthenticates retained bytes, receipt identity,
// public certificate and the current exact canonical unsigned preparation.
func VerifyPublishedSignedBundle(ctx context.Context, profile, generation, signedGeneration string) (SignedBundleReceipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	inputs, err := inspectSignedBundleInputs(ctx, profile, generation)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	return verifyPublishedSignedBundle(ctx, inputs, generation, signedGeneration)
}
func verifyPublishedSignedBundle(ctx context.Context, inputs signedBundleInputs, generation, id string) (SignedBundleReceipt, error) {
	empty := SignedBundleReceipt{}
	if !validSHA(id) {
		return empty, errors.New("signed generation must be its exact SHA-256 receipt identity")
	}
	dir := filepath.Join(signedBundleParent(inputs.expected), id)
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return empty, errors.New("signed generation unavailable")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return empty, err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return empty, err
	}
	if len(entries) != 3 {
		return empty, errors.New("signed publication closure changed")
	}
	for _, name := range []string{"receipt.json", "certificate.der"} {
		i, e := root.Lstat(name)
		if e != nil || !i.Mode().IsRegular() {
			return empty, errors.New("signed publication metadata must be regular")
		}
	}
	info, err = root.Lstat("modules")
	if err != nil || !info.IsDir() {
		return empty, errors.New("signed publication module directory changed")
	}
	data, err := readSmallRegular(bundleFiles{root}, "receipt.json", 64<<10)
	if err != nil {
		return empty, err
	}
	if digest(data) != id {
		return empty, errors.New("signed publication receipt digest mismatch")
	}
	var r SignedBundleReceipt
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&r); err != nil {
		return empty, err
	}
	canonical, err := json.MarshalIndent(r, "", "  ")
	if err != nil || !bytes.Equal(canonical, data) {
		return empty, errors.New("signed receipt is not canonical immutable JSON")
	}
	if r.Directory != "" || r.ModulesDirectory != "" || r.PublicCertificate != "" || r.ReceiptSHA256 != "" || r.SchemaVersion != 1 || r.Profile != inputs.expected.Profile || r.Bundle != generation || r.KernelRelease != inputs.expected.KernelRelease || r.BundleReceiptSHA256 != inputs.receiptSHA || !r.SignaturesVerified || r.KernelTrustVerified || r.ActivationEligible || (r.SigningMethod != signedBundleSuppliedMethod && r.SigningMethod != signedBundleMOKMethod) {
		return empty, errors.New("signed receipt does not match canonical preparation or policy")
	}
	cert, err := readPublicCertificate(filepath.Join(dir, "certificate.der"))
	if err != nil {
		return empty, err
	}
	if digest(cert.Raw) != r.CertificateSHA256 {
		return empty, errors.New("retained public certificate changed")
	}
	modules, err := root.OpenRoot("modules")
	if err != nil {
		return empty, err
	}
	defer modules.Close()
	proof, err := verifySignedFiles(ctx, bundleFiles{modules}, inputs.files, cert)
	if err != nil {
		return empty, err
	}
	got, _ := json.Marshal(proof)
	want, _ := json.Marshal(r.Modules)
	if !bytes.Equal(got, want) {
		return empty, errors.New("signed module manifest mismatch")
	}
	return locateSignedBundle(r, dir, id), nil
}

func chownSignedBundleClosure(root *os.Root, canonical string) error {
	return fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return errors.New("foreign publication inode")
		}
		f, err := openBundleFile(root, name)
		if err != nil {
			return err
		}
		return errors.Join(chownSignedBundleFile(f, filepath.Join(canonical, name)), f.Sync(), f.Close())
	})
}

// requireReviewedSigningReceipt binds privileged signing to the operator's
// independent reviewed preparation identity, never a mutable pathname alone.
func requireReviewedSigningReceipt(inputs signedBundleInputs, reviewed string) error {
	if !validSHA(reviewed) || inputs.receiptSHA != reviewed {
		return errors.New("privileged signing requires the exact independently reviewed preparation receipt SHA-256")
	}
	return nil
}
