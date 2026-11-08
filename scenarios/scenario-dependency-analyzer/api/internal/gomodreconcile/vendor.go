package gomodreconcile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// A module with a committed vendor/ tree builds from that tree, and Go
// refuses it ("inconsistent vendoring") when vendor/modules.txt disagrees
// with go.mod: every requirement and every replace directive is recorded
// there. Vendored copies of locally replaced in-repo modules also go stale
// when those modules change. Convergence therefore includes the vendor tree:
// Converge re-vendors the converged go.mod into a staging directory and
// Write swaps it in with go.mod and go.sum, so a vendored module can never be
// left half converged.

// vendorStage is a re-vendored tree staged beside the module, plus what the
// current tree looked like when it was staged.
type vendorStage struct {
	dir              string
	modulesTxtBefore string
	modulesTxtAfter  string
	preserved        []string
}

func hasVendorTree(moduleDir string) bool {
	info, err := os.Stat(filepath.Join(moduleDir, "vendor", "modules.txt"))
	return err == nil && !info.IsDir()
}

// stageVendor runs `go mod vendor` for the converged go.mod into a staging
// directory inside the module (dot-prefixed, so the go command ignores it and
// the final swap is a same-filesystem rename). Top-level vendor/ entries that
// the go command does not own (neither the current nor the new modules.txt
// lists a package under them, such as packages/proto's vendored googleapis
// and protovalidate sources) are carried over unchanged. It returns nil when
// the staged tree is identical to the current one.
func stageVendor(ctx context.Context, moduleDir, convergedModFile string) (*vendorStage, error) {
	current := filepath.Join(moduleDir, "vendor")
	modulesBefore, err := os.ReadFile(filepath.Join(current, "modules.txt"))
	if err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(moduleDir, ".vendor-reconcile-")
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(staging)
		}
	}()
	staged := filepath.Join(staging, "vendor")
	if err := runGo(ctx, moduleDir, "mod", "vendor", "-modfile="+convergedModFile, "-o", staged); err != nil {
		return nil, fmt.Errorf("go mod vendor: %w", err)
	}
	modulesAfter, err := os.ReadFile(filepath.Join(staged, "modules.txt"))
	if err != nil {
		return nil, err
	}
	owned := vendorOwnedRoots(modulesBefore)
	for root := range vendorOwnedRoots(modulesAfter) {
		owned[root] = struct{}{}
	}
	entries, err := os.ReadDir(current)
	if err != nil {
		return nil, err
	}
	var preserved []string
	for _, entry := range entries {
		name := entry.Name()
		if _, ok := owned[name]; ok || name == "modules.txt" {
			continue
		}
		if err := copyTree(filepath.Join(current, name), filepath.Join(staged, name)); err != nil {
			return nil, fmt.Errorf("preserve vendor/%s: %w", name, err)
		}
		preserved = append(preserved, name)
	}
	same, err := sameTree(current, staged)
	if err != nil {
		return nil, err
	}
	if same {
		return nil, nil
	}
	keep = true
	sort.Strings(preserved)
	return &vendorStage{dir: staging, modulesTxtBefore: string(modulesBefore), modulesTxtAfter: string(modulesAfter), preserved: preserved}, nil
}

// vendorOwnedRoots returns the first path elements of every package and module
// that modules.txt records; the go command owns those vendor/ subtrees.
func vendorOwnedRoots(modulesTxt []byte) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range strings.Split(string(modulesTxt), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "##") {
			continue
		}
		path := strings.Fields(strings.TrimPrefix(line, "# "))
		if len(path) == 0 {
			continue
		}
		out[strings.SplitN(path[0], "/", 2)[0]] = struct{}{}
	}
	return out
}

// swapIn replaces the module's vendor/ with the staged tree. The old tree is
// moved aside first and restored if the swap fails.
func (v *vendorStage) swapIn(moduleDir string) error {
	current := filepath.Join(moduleDir, "vendor")
	modulesNow, err := os.ReadFile(filepath.Join(current, "modules.txt"))
	if err != nil {
		return err
	}
	if string(modulesNow) != v.modulesTxtBefore {
		return ErrConcurrentEdit
	}
	aside := filepath.Join(v.dir, "previous-vendor")
	if err := os.Rename(current, aside); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(v.dir, "vendor"), current); err != nil {
		if restoreErr := os.Rename(aside, current); restoreErr != nil {
			return errors.Join(err, restoreErr)
		}
		return err
	}
	// The staging directory now holds only the previous tree.
	_ = os.RemoveAll(v.dir)
	v.dir = ""
	return nil
}

func (v *vendorStage) close() {
	if v != nil && v.dir != "" {
		_ = os.RemoveAll(v.dir)
	}
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// sameTree reports whether two directory trees hold the same files with the
// same contents.
func sameTree(a, b string) (bool, error) {
	files := func(root string) (map[string]string, error) {
		out := map[string]string{}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(rel)] = path
			return nil
		})
		return out, err
	}
	left, err := files(a)
	if err != nil {
		return false, err
	}
	right, err := files(b)
	if err != nil {
		return false, err
	}
	if len(left) != len(right) {
		return false, nil
	}
	for rel, leftPath := range left {
		rightPath, ok := right[rel]
		if !ok {
			return false, nil
		}
		l, err := os.ReadFile(leftPath)
		if err != nil {
			return false, err
		}
		r, err := os.ReadFile(rightPath)
		if err != nil {
			return false, err
		}
		if !bytes.Equal(l, r) {
			return false, nil
		}
	}
	return true, nil
}
