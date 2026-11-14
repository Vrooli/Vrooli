//go:build linux

package hostwireless

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const existingMOKPrivate = "/var/lib/shim-signed/mok/MOK.priv"
const existingMOKPublic = "/var/lib/shim-signed/mok/MOK.der"

// SignPreparedBundle uses only the installed exact-ABI sign-file and the
// existing protected module-only shim MOK. It requires an already elevated
// caller and an independently reviewed preparation receipt digest. It never
// creates a key or enrolls trust, and never changes host modules.
func SignPreparedBundle(ctx context.Context, profile, generation, certPath, reviewedReceiptSHA256 string) (_ SignedBundleReceipt, resultErr error) {
	if os.Geteuid() != 0 {
		return SignedBundleReceipt{}, errors.New("signing requires an explicitly elevated root caller; publication of already signed copies is unprivileged")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	inputs, err := inspectSignedBundleInputs(ctx, profile, generation)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	if err = requireReviewedSigningReceipt(inputs, reviewedReceiptSHA256); err != nil {
		return SignedBundleReceipt{}, err
	}
	cert, err := readPublicCertificate(certPath)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	fixed, err := readPublicCertificate(existingMOKPublic)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	if !bytes.Equal(cert.Raw, fixed.Raw) {
		return SignedBundleReceipt{}, errors.New("signing public certificate must equal the existing shim MOK certificate")
	}
	moduleOnly := false
	for _, oid := range fixed.UnknownExtKeyUsage {
		if oid.Equal(moduleOnlyOID) {
			moduleOnly = true
		}
	}
	if !moduleOnly {
		return SignedBundleReceipt{}, errors.New("existing signing certificate lacks module-only EKU")
	}
	key, err := openProtectedSignerInput(existingMOKPrivate, false)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	defer key.Close()
	signerPath := filepath.Join("/usr/src", "linux-headers-"+inputs.expected.KernelRelease, "scripts", "sign-file")
	signer, err := openProtectedSignerInput(signerPath, true)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	defer signer.Close()
	// Reserve root-private signing storage independently of the public output.
	signingBytes := int64(256 << 10)
	unsignedDir := filepath.Join(filepath.Dir(filepath.Dir(inputs.expected.SourceDirectory)), "host-wireless", profile, generation)
	for _, m := range inputs.files {
		i, e := os.Stat(filepath.Join(unsignedDir, m.Path))
		if e != nil {
			return SignedBundleReceipt{}, e
		}
		signingBytes += i.Size() + maxSignatureBytes + 8192
	}
	release, err := reserveSignedBundle(ctx, "/var/tmp", signingBytes)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	cleanupVerified := true
	defer func() {
		if cleanupVerified {
			resultErr = errors.Join(resultErr, release())
		}
	}()
	// A root-owned, non-user-writable temporary workspace keeps the privileged
	// signer away from mutable operator paths. Only authenticated public outputs
	// leave it. No password environment or inherited OpenSSL configuration.
	work, err := os.MkdirTemp("/var/tmp", "vrooli-wireless-sign-")
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	defer func() { e := os.RemoveAll(work); cleanupVerified = e == nil; resultErr = errors.Join(resultErr, e) }()
	unsigned := filepath.Join(filepath.Dir(filepath.Dir(inputs.expected.SourceDirectory)), "host-wireless", profile, generation)
	source, err := os.OpenRoot(unsigned)
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	defer source.Close()
	signedDir, err := stageAndSignPrepared(ctx, inputs, source, work, fixed.Raw, func(ctx context.Context, module, public string) error {
		return runInstalledModuleSigner(ctx, signer, key, module, public)
	})
	if err != nil {
		return SignedBundleReceipt{}, err
	}
	return publishSignedBundle(ctx, inputs, generation, signedDir, filepath.Join(work, "certificate.der"), signedBundleMOKMethod, reserveSignedBundle)
}

func openProtectedSignerInput(path string, executable bool) (*os.File, error) {
	// O_PATH pins the inode without reading private key bytes into this process.
	// sign-file alone reopens its inherited descriptor when actually invoked.
	fd, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{Flags: unix.O_PATH | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
	if err != nil {
		return nil, fmt.Errorf("open protected signing input: %w", err)
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 || (!executable && info.Mode().Perm()&0077 != 0) || (executable && info.Mode().Perm()&0111 == 0) {
		f.Close()
		return nil, errors.New("signing input is not an exact protected root-owned regular file")
	}
	// A root-owned file below writable ancestors could be renamed before opening
	// but the descriptor above pins its admitted inode. Require each ancestor's
	// ownership and modes as well to reject attacker-controlled installations.
	for parent := filepath.Dir(path); parent != "/"; parent = filepath.Dir(parent) {
		i, e := os.Lstat(parent)
		if e != nil || !i.IsDir() {
			f.Close()
			return nil, errors.New("protected signing ancestor unavailable")
		}
		s, ok := i.Sys().(*syscall.Stat_t)
		if !ok || s.Uid != 0 || i.Mode().Perm()&0022 != 0 {
			f.Close()
			return nil, errors.New("protected signing ancestor is mutable")
		}
	}
	return f, nil
}

type preparedModuleSigner func(context.Context, string, string) error

// stageAndSignPrepared has no host-key knowledge. Tests inject an ephemeral
// key signer here; production exposes no arbitrary key/executable path.
func stageAndSignPrepared(ctx context.Context, inputs signedBundleInputs, source *os.Root, work string, certDER []byte, sign preparedModuleSigner) (string, error) {
	signedDir := filepath.Join(work, "modules")
	if err := os.Mkdir(signedDir, 0700); err != nil {
		return "", err
	}
	public := filepath.Join(work, "certificate.der")
	if err := os.WriteFile(public, certDER, 0400); err != nil {
		return "", err
	}
	for _, m := range inputs.files {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		data, err := readSmallRegular(bundleFiles{source}, m.Path, maxArtifactBytes)
		if err != nil {
			return "", err
		}
		if digest(data) != m.SHA256 {
			return "", errors.New("unsigned source changed before signing")
		}
		dest := filepath.Join(signedDir, m.Path)
		if err = os.WriteFile(dest, data, 0600); err != nil {
			return "", err
		}
		if err = sign(ctx, dest, public); err != nil {
			return "", err
		}
	}
	return signedDir, ctx.Err()
}

func runInstalledModuleSigner(ctx context.Context, signer, key *os.File, module, public string) error {
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", "sha256", "/proc/self/fd/4", public, module)
	cmd.ExtraFiles = []*os.File{signer, key}
	cmd.Env = []string{"LC_ALL=C", "OPENSSL_CONF=/dev/null"}
	cmd.Stdin = nil
	cmd.WaitDelay = time.Second
	// Drop diagnostic output: sign-file must never surface key material or a
	// password prompt. The generic error identifies only the bounded operation.
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("installed exact-ABI sign-file failed")
	}
	return nil
}
