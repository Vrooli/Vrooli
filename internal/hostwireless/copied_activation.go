package hostwireless

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
)

const maxPreservedBytes = 128 << 20

// CopiedActivation owns a private rehearsal root. It never adopts an existing root.
// It rehearses module directory publication; native index generation is exercised
// by prepare and is a separate gate for any future live transaction.
type CopiedActivation struct {
	mu             sync.Mutex
	root           *os.Root
	overlay        string
	protected      []FileIdentity
	candidates     []FileIdentity
	createdUpdates bool
}

// NewCopiedActivation copies and verifies the protected file identities before a
// rehearsal. Source files and the newly created private root have separate lifetimes.
func NewCopiedActivation(ctx context.Context, baseline fs.FS, plan ActivationPlan) (*CopiedActivation, error) {
	return newCopiedActivation(ctx, baseline, plan, "")
}

func newCopiedActivation(ctx context.Context, baseline fs.FS, plan ActivationPlan, parent string) (*CopiedActivation, error) {
	recoveryRelease := plan.RecoveryRelease
	spec, ok := preparationProfiles[plan.Profile]
	if !ok || plan.KernelRelease != nobleABI || plan.OverlayDirectory != overlayDirectory(plan.Profile) ||
		plan.ActivationEligible || !validSHA(plan.BundleReceiptSHA256) ||
		recoveryRelease == nobleABI || !nativeName.MatchString(recoveryRelease) ||
		!slices.Contains(plan.RecoveryCandidates, recoveryRelease) {
		return nil, errors.New("copied activation requires the reviewed ABI, an exact bundle receipt and a distinct recovery candidate")
	}
	if err := verifyPlanModules(plan.Originals, plan.Candidates, spec); err != nil {
		return nil, err
	}
	protected := append(slices.Clone(plan.Originals), plan.Preserved...)
	if len(protected) > 32 {
		return nil, errors.New("protected file count exceeds rehearsal bound")
	}
	seen := map[string]bool{}
	for i := range protected {
		file := &protected[i]
		file.Path = strings.TrimPrefix(file.Path, "/")
		if !safeProtectedPath(file.Path, plan.OverlayDirectory) || seen[file.Path] || file.Issue != "" || !validSHA(file.SHA256) {
			return nil, fmt.Errorf("invalid protected identity %s", file.Path)
		}
		seen[file.Path] = true
	}
	required := append(bootFiles(nobleABI), "/boot/vmlinuz-"+recoveryRelease, "/boot/initrd.img-"+recoveryRelease)
	if len(plan.WiredInterfaces) > 0 {
		required = append(required, "/proc/net/route")
	}
	for _, name := range required {
		if !seen[strings.TrimPrefix(name, "/")] {
			return nil, fmt.Errorf("missing preserved boot identity %s", name)
		}
	}
	if err := verifyPreservedProviders(plan, protected); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(parent, "vrooli-wireless-copy-")
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, errors.Join(err, os.RemoveAll(dir))
	}
	tx := &CopiedActivation{root: root, overlay: plan.OverlayDirectory, protected: protected, candidates: slices.Clone(plan.Candidates)}
	var remaining int64 = 512 << 20
	for _, file := range protected {
		n, err := copyVerified(ctx, baseline, root, file.Path, file, min(remaining, maxPreservedBytes))
		if err != nil {
			return nil, errors.Join(err, tx.Close())
		}
		remaining -= n
	}
	return tx, nil
}

func verifyPreservedProviders(plan ActivationPlan, protected []FileIdentity) error {
	modules := slices.Clone(plan.NVIDIALoadedModules)
	for _, iface := range plan.WiredInterfaces {
		module := plan.WiredDrivers[iface]
		if !nativeName.MatchString(module) {
			return fmt.Errorf("missing wired module identity for %s", iface)
		}
		modules = append(modules, module)
	}
	for _, module := range modules {
		name := strings.ReplaceAll(module, "_", "-") + ".ko"
		found := false
		for _, file := range protected {
			base := strings.ReplaceAll(path.Base(file.Path), "_", "-")
			if strings.HasPrefix(file.Path, "lib/modules/"+nobleABI+"/") &&
				(base == name || base == name+".zst" || base == name+".xz" || base == name+".gz") {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing preserved provider identity for %s", module)
		}
	}
	return nil
}

func verifyPlanModules(originals, candidates []FileIdentity, spec buildProfile) error {
	if len(candidates) != len(spec.modules) || len(originals) != len(spec.modules) {
		return errors.New("module plan does not contain the exact reviewed module set")
	}
	for i, module := range spec.modules {
		original, candidate := originals[i], candidates[i]
		name := module + ".ko"
		base := path.Base(original.Path)
		if candidate.Path != name || candidate.Issue != "" || !validSHA(candidate.SHA256) ||
			!strings.HasPrefix(strings.TrimPrefix(original.Path, "/"), "lib/modules/"+nobleABI+"/kernel/") ||
			(base != name && base != name+".zst" && base != name+".xz" && base != name+".gz") {
			return fmt.Errorf("module plan does not match stock and candidate identities for %s", module)
		}
	}
	return nil
}

func safeProtectedPath(name, overlay string) bool {
	return fs.ValidPath(name) && !strings.Contains(name, "\\") &&
		name != overlay && !strings.HasPrefix(name, overlay+"/") &&
		(strings.HasPrefix(name, "boot/") || name == "etc/default/grub" || name == "etc/depmod.d/ubuntu.conf" || name == "proc/net/route" ||
			strings.HasPrefix(name, "lib/modules/"))
}

// Stage publishes only the complete digest-verified overlay. Repeating Stage
// succeeds if the previously published bytes and the baseline still match.
func (t *CopiedActivation) Stage(ctx context.Context, candidate fs.FS) (resultErr error) {
	return t.stage(ctx, candidate, maxArtifactBytes)
}

func (t *CopiedActivation) stage(ctx context.Context, candidate fs.FS, maxFileBytes int64) (resultErr error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.root == nil {
		return os.ErrClosed
	}
	if err := t.verifyProtected(ctx); err != nil {
		return err
	}
	if present, err := t.verifyOverlay(ctx); present || err != nil {
		return err
	}
	const pending = ".pending"
	if err := t.root.RemoveAll(pending); err != nil {
		return err
	}
	if err := t.root.Mkdir(pending, 0700); err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, t.root.RemoveAll(pending)) }()
	var remaining int64 = 128 << 20
	for _, file := range t.candidates {
		n, err := copyVerified(ctx, candidate, t.root, path.Join(pending, file.Path), file, min(remaining, maxFileBytes))
		if err != nil {
			return err
		}
		remaining -= n
	}
	if err := t.verifyProtected(ctx); err != nil {
		return err
	}
	updates := path.Dir(t.overlay)
	if err := t.root.Mkdir(updates, 0700); err == nil {
		t.createdUpdates = true
	} else if errors.Is(err, fs.ErrExist) {
		info, statErr := t.root.Lstat(updates)
		if statErr != nil || !info.IsDir() {
			return errors.New("copied updates parent is not a regular directory")
		}
	} else {
		return err
	}
	if err := t.root.Rename(pending, t.overlay); err != nil {
		return errors.Join(err, t.removeCreatedUpdates())
	}
	return t.verifyProtected(ctx)
}

// Rollback removes only an overlay whose file set and bytes still match. A
// missing overlay is an idempotent abort and leaves the baseline untouched.
func (t *CopiedActivation) Rollback(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.root == nil {
		return os.ErrClosed
	}
	const retired = ".retired"
	present, err := t.verifyOverlay(ctx)
	if err != nil {
		return err
	}
	if err := t.root.RemoveAll(retired); err != nil {
		return err
	}
	if present {
		if err := t.root.Rename(t.overlay, retired); err != nil {
			return err
		}
		if err := t.root.RemoveAll(retired); err != nil {
			return err
		}
	}
	return errors.Join(t.removeCreatedUpdates(), t.verifyProtected(ctx))
}

func (t *CopiedActivation) removeCreatedUpdates() error {
	if !t.createdUpdates {
		return nil
	}
	if err := t.root.Remove(path.Dir(t.overlay)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	t.createdUpdates = false
	return nil
}

func (t *CopiedActivation) verifyProtected(ctx context.Context) error {
	for _, file := range t.protected {
		if err := verifyFile(ctx, t.root.FS(), file.Path, file.SHA256); err != nil {
			return fmt.Errorf("protected baseline changed: %w", err)
		}
	}
	return nil
}

func (t *CopiedActivation) verifyOverlay(ctx context.Context) (bool, error) {
	info, err := t.root.Lstat(t.overlay)
	if errors.Is(err, fs.ErrNotExist) {
		return false, ctx.Err()
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, errors.New("copied overlay is not a regular directory")
	}
	entries, err := fs.ReadDir(t.root.FS(), t.overlay)
	if err != nil {
		return false, err
	}
	if len(entries) != len(t.candidates) {
		return false, errors.New("copied overlay file set changed")
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return false, fmt.Errorf("unexpected copied overlay entry %s", entry.Name())
		}
	}
	for _, file := range t.candidates {
		if err := verifyFile(ctx, t.root.FS(), path.Join(t.overlay, file.Path), file.SHA256); err != nil {
			return false, err
		}
	}
	return true, ctx.Err()
}

// Close releases and deletes the private rehearsal root.
func (t *CopiedActivation) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.root == nil {
		return nil
	}
	root := t.root
	t.root = nil
	return errors.Join(root.Close(), os.RemoveAll(root.Name()))
}

func validSHA(value string) bool {
	data, err := hex.DecodeString(value)
	return err == nil && len(data) == sha256.Size && value == strings.ToLower(value)
}

func verifyFile(ctx context.Context, tree fs.FS, file, expected string) error {
	if !validSHA(expected) {
		return fmt.Errorf("invalid expected digest for %s", file)
	}
	_, err := digestCopy(ctx, io.Discard, tree, FileIdentity{Path: file, SHA256: expected}, maxPreservedBytes)
	return err
}

func copyVerified(ctx context.Context, source fs.FS, root *os.Root, dest string, file FileIdentity, maxBytes int64) (int64, error) {
	if err := root.MkdirAll(path.Dir(dest), 0700); err != nil {
		return 0, err
	}
	out, err := root.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return 0, err
	}
	size, err := digestCopy(ctx, out, source, file, maxBytes)
	return size, errors.Join(err, out.Close())
}

func digestCopy(ctx context.Context, dst io.Writer, source fs.FS, file FileIdentity, maxBytes int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	in, err := source.Open(file.Path)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxBytes {
		return 0, fmt.Errorf("file type or size is outside rehearsal bounds: %s", file.Path)
	}
	hash := sha256.New()
	reader := &io.LimitedReader{R: checkedReader{ctx, in}, N: maxBytes + 1}
	n, err := io.Copy(io.MultiWriter(hash, dst), reader)
	if err != nil {
		return n, err
	}
	if n != info.Size() || n > maxBytes || hex.EncodeToString(hash.Sum(nil)) != file.SHA256 {
		return n, fmt.Errorf("file digest mismatch: %s", file.Path)
	}
	return n, ctx.Err()
}

type checkedReader struct {
	ctx context.Context
	io.Reader
}

func (r checkedReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(data)
}
