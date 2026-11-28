package hostwireless

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/vrooli/binaryfetch"
)

const abiTree = "lib/modules/" + nobleABI

var depmodIndexes = []string{
	"modules.alias", "modules.alias.bin", "modules.builtin.alias.bin", "modules.builtin.bin",
	"modules.dep", "modules.dep.bin", "modules.devname", "modules.softdep", "modules.symbols", "modules.symbols.bin",
}

type copiedIndexer func(context.Context, *CopiedActivation, ActivationPlan) error

type privateTreeEntry struct {
	SHA256 string      `json:"sha256,omitempty"`
	Mode   fs.FileMode `json:"mode"`
}

// indexedActivation publishes one complete private root through an atomic
// selector. Both generations are owned copies. The selected stock root
// remains available while a candidate is being generated and validated.
type indexedActivation struct {
	mu                       sync.Mutex
	dir                      string
	plan                     ActivationPlan
	stock, candidate         *CopiedActivation
	stockTree, candidateTree map[string]privateTreeEntry
	index                    copiedIndexer
	certSHA                  string
	modules                  []SignedModuleEvidence
}

func newIndexedActivation(ctx context.Context, baseline fs.FS, plan ActivationPlan, index copiedIndexer) (*indexedActivation, error) {
	return newIndexedActivationAt(ctx, baseline, plan, index, "")
}
func newIndexedActivationAt(ctx context.Context, baseline fs.FS, plan ActivationPlan, index copiedIndexer, parent string) (*indexedActivation, error) {
	if index == nil || !bundleID.MatchString(plan.Bundle) || len(plan.Modaliases) == 0 ||
		!validSHA(plan.KernelSymbolsSHA256) || preservedSHA(plan, "/"+abiTree+"/build/Module.symvers") != plan.KernelSymbolsSHA256 {
		return nil, errors.New("indexed rehearsal requires a reviewed bundle, observed aliases and an index owner")
	}
	dir, err := os.MkdirTemp(parent, "vrooli-wireless-indexed-")
	if err != nil {
		return nil, err
	}
	tx := &indexedActivation{dir: dir, plan: plan, index: index}
	tx.stock, err = newCopiedActivation(ctx, baseline, plan, dir)
	if err == nil {
		err = cloneABITree(ctx, baseline, tx.stock.root)
	}
	if err == nil {
		err = tx.stock.verifyProtected(ctx)
	}
	if err == nil {
		tx.stockTree, err = snapshotPrivateTree(ctx, tx.stock.root)
	}
	if err == nil {
		err = requireDepmodIndexes(tx.stockTree)
	}
	if err == nil {
		err = os.Symlink(filepath.Base(tx.stock.root.Name()), filepath.Join(dir, "current"))
	}
	if err != nil {
		return nil, errors.Join(err, tx.close())
	}
	return tx, nil
}

func preservedSHA(plan ActivationPlan, name string) string {
	for _, file := range plan.Preserved {
		if file.Path == name {
			return file.SHA256
		}
	}
	return ""
}

func (t *indexedActivation) stage(ctx context.Context, signed fs.FS, cert *x509.Certificate) (_ []SignedModuleEvidence, resultErr error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.verifySelected(ctx); err != nil {
		return nil, err
	}
	modules, err := verifySignedFiles(ctx, signed, t.plan.Candidates, cert)
	if err != nil {
		return nil, err
	}
	certSHA := digest(cert.Raw)
	if t.candidate != nil {
		if certSHA != t.certSHA || !slices.Equal(modules, t.modules) {
			return nil, errors.New("a different signed generation is already selected")
		}
		return modules, nil
	}

	candidate, err := newCopiedActivation(ctx, t.stock.root.FS(), t.plan, t.dir)
	if err != nil {
		return nil, err
	}
	defer func() {
		if t.candidate == nil {
			resultErr = errors.Join(resultErr, candidate.Close())
		}
	}()
	if err := cloneABITree(ctx, t.stock.root.FS(), candidate.root); err != nil {
		return nil, err
	}
	candidate.candidates = make([]FileIdentity, 0, len(modules))
	for _, module := range modules {
		candidate.candidates = append(candidate.candidates, FileIdentity{Path: module.Name, SHA256: module.SignedSHA256})
	}
	if err := candidate.stage(ctx, signed, maxSignedModuleBytes); err != nil {
		return nil, err
	}
	preIndex, err := snapshotPrivateTree(ctx, candidate.root)
	if err != nil {
		return nil, err
	}
	for _, name := range depmodIndexes {
		if err := candidate.root.Remove(path.Join(abiTree, name)); err != nil {
			return nil, err
		}
	}
	if err := t.index(ctx, candidate, t.plan); err != nil {
		return nil, fmt.Errorf("generate private module indexes: %w", err)
	}
	indexed, err := snapshotPrivateTree(ctx, candidate.root)
	if err != nil {
		return nil, err
	}
	if err := requireDepmodIndexes(indexed); err != nil {
		return nil, err
	}
	for _, name := range depmodIndexes {
		file := path.Join(abiTree, name)
		preIndex[file] = indexed[file]
	}
	if !maps.Equal(preIndex, indexed) {
		return nil, errors.New("index generation changed files outside the exact index set")
	}
	if err := compareDependencyProviders(t.stock.root.FS(), candidate.root.FS(), t.plan); err != nil {
		return nil, err
	}
	if err := t.verifySelected(ctx); err != nil {
		return nil, err
	}
	if err := t.selectRoot(ctx, candidate); err != nil {
		return nil, err
	}
	t.candidate, t.candidateTree, t.certSHA, t.modules = candidate, indexed, certSHA, slices.Clone(modules)
	return modules, nil
}

func (t *indexedActivation) rollback(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.verifySelected(ctx); err != nil || t.candidate == nil {
		return err
	}
	if err := t.selectRoot(ctx, t.stock); err != nil {
		return err
	}
	candidate := t.candidate
	t.candidate, t.candidateTree, t.certSHA, t.modules = nil, nil, "", nil
	return candidate.Close()
}

func (t *indexedActivation) verifySelected(ctx context.Context) error {
	if t.stock == nil {
		return os.ErrClosed
	}
	selected := t.stock
	if t.candidate != nil {
		selected = t.candidate
	}
	link, err := os.Readlink(filepath.Join(t.dir, "current"))
	if err != nil || link != filepath.Base(selected.root.Name()) {
		return errors.New("private module generation selector drifted")
	}
	stock, err := snapshotPrivateTree(ctx, t.stock.root)
	if err != nil || !maps.Equal(stock, t.stockTree) {
		return errors.Join(errors.New("private stock generation drifted"), err)
	}
	if t.candidate != nil {
		candidate, err := snapshotPrivateTree(ctx, t.candidate.root)
		if err != nil || !maps.Equal(candidate, t.candidateTree) {
			return errors.Join(errors.New("private signed generation drifted"), err)
		}
	}
	return ctx.Err()
}

func (t *indexedActivation) selectRoot(ctx context.Context, next *CopiedActivation) error {
	pending := filepath.Join(t.dir, ".next")
	if err := os.Symlink(filepath.Base(next.root.Name()), pending); err != nil {
		return err
	}
	defer os.Remove(pending)
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(pending, filepath.Join(t.dir, "current"))
}

func (t *indexedActivation) close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	var err error
	for _, root := range []*CopiedActivation{t.candidate, t.stock} {
		if root != nil {
			err = errors.Join(err, root.Close())
		}
	}
	t.stock, t.candidate = nil, nil
	return errors.Join(err, os.RemoveAll(t.dir))
}

func requireDepmodIndexes(tree map[string]privateTreeEntry) error {
	for _, name := range depmodIndexes {
		if !validSHA(tree[path.Join(abiTree, name)].SHA256) {
			return fmt.Errorf("missing private depmod index %s", name)
		}
	}
	return nil
}

func compareDependencyProviders(stock, candidate fs.FS, plan ActivationPlan) error {
	file := path.Join(abiTree, "modules.dep")
	oldData, err := readSmallRegular(stock, file, maxArtifactBytes)
	if err != nil {
		return err
	}
	newData, err := readSmallRegular(candidate, file, maxArtifactBytes)
	if err != nil {
		return err
	}
	replacements := map[string]string{}
	selected := map[string]bool{}
	for i, module := range plan.Candidates {
		original := strings.TrimPrefix(plan.Originals[i].Path, "/"+abiTree+"/")
		replacements[strings.TrimPrefix(plan.OverlayDirectory, abiTree+"/")+"/"+module.Path] = original
		selected[original] = true
	}
	old, err := dependencyProviders(oldData, nil, selected)
	if err != nil {
		return err
	}
	current, err := dependencyProviders(newData, replacements, selected)
	if err != nil {
		return err
	}
	if !maps.Equal(old, current) {
		return errors.New("private depmod changed stock provider paths or an unchanged module's dependency closure")
	}
	return nil
}

func dependencyProviders(data []byte, replacements map[string]string, selected map[string]bool) (map[string]string, error) {
	providers := map[string]string{}
	found := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		name, dependencies, ok := strings.Cut(line, ":")
		if !ok || !fs.ValidPath(name) || name == "." {
			return nil, errors.New("invalid private module dependency record")
		}
		if stock, ok := replacements[name]; ok {
			found[name] = true
			name = stock
		}
		if _, exists := providers[name]; exists {
			return nil, errors.New("duplicate private module dependency provider")
		}
		items := strings.Fields(dependencies)
		for i, dependency := range items {
			if !fs.ValidPath(dependency) || dependency == "." {
				return nil, errors.New("invalid private module dependency path")
			}
			if stock, ok := replacements[dependency]; ok {
				items[i] = stock
			}
		}
		slices.Sort(items)
		providers[name] = strings.Join(items, " ")
		if selected[name] {
			providers[name] = ""
		}
	}
	if len(found) != len(replacements) {
		return nil, errors.New("private dependency index did not select the exact signed overlay")
	}
	return providers, nil
}

func cloneABITree(ctx context.Context, baseline fs.FS, dest *os.Root) error {
	var entries int
	var remaining int64 = 512 << 20
	return fs.WalkDir(baseline, abiTree, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// kmod does not index the external Kbuild link or vDSO debug files.
		if name == abiTree+"/build" || name == abiTree+"/vdso" {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		entries++
		if entries > 12000 {
			return errors.New("private module tree exceeds entry bound")
		}
		if entry.IsDir() {
			return dest.MkdirAll(name, 0700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > min(remaining, maxPreservedBytes) {
			return fmt.Errorf("private module tree entry outside bounds: %s", name)
		}
		remaining -= info.Size()
		if _, err := dest.Lstat(name); err == nil {
			return nil // Already copied and pinned by the activation plan.
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return cloneRegular(ctx, baseline, dest, name, info.Size())
	})
}

func cloneRegular(ctx context.Context, source fs.FS, root *os.Root, name string, size int64) error {
	in, err := source.Open(name)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(out, io.LimitReader(checkedReader{ctx, in}, size+1))
	if n != size {
		copyErr = errors.Join(copyErr, fmt.Errorf("private module tree entry changed: %s", name))
	}
	return errors.Join(copyErr, out.Close())
}

func snapshotPrivateTree(ctx context.Context, root *os.Root) (map[string]privateTreeEntry, error) {
	files := map[string]privateTreeEntry{}
	var remaining int64 = 1 << 30
	err := fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(files) > 12100 {
			return errors.New("private root exceeds entry bound")
		}
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			files[name] = privateTreeEntry{Mode: info.Mode()}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > min(remaining, maxPreservedBytes) {
			return fmt.Errorf("private root entry outside bounds: %s", name)
		}
		remaining -= info.Size()
		hash, err := binaryfetch.FileDigestContext(ctx, filepath.Join(root.Name(), name))
		files[name] = privateTreeEntry{SHA256: hash, Mode: info.Mode()}
		return err
	})
	return files, err
}
