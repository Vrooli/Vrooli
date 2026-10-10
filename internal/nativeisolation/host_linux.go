//go:build linux

package nativeisolation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

func trustedPath(path string, native uint32) error {
	for p := path; p != "/"; p = filepath.Dir(p) {
		info, e := os.Lstat(p)
		if e != nil || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 {
			return ErrRefused
		}
		s, ok := info.Sys().(*syscall.Stat_t)
		if !ok || s.Uid == native {
			return ErrRefused
		}
	}
	return nil
}
func trustedTree(root string, native uint32) error {
	if trustedPath(root, native) != nil {
		return ErrRefused
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return ErrRefused
		}
		info, e := d.Info()
		if e != nil {
			return ErrRefused
		}
		s, ok := info.Sys().(*syscall.Stat_t)
		if !ok || s.Uid == native || info.Mode().Perm()&0022 != 0 || (!info.Mode().IsRegular() && !info.IsDir() && info.Mode()&os.ModeSymlink == 0) {
			return ErrRefused
		}
		return nil
	})
}
func checkWorkspace(root string, protected []string) error {
	real, e := filepath.EvalSymlinks(root)
	if e != nil || real != root {
		return ErrRefused
	}
	forbidden := map[[2]uint64]bool{}
	for _, p := range protected {
		info, e := os.Lstat(p)
		if e != nil {
			return ErrRefused
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			forbidden[[2]uint64{uint64(st.Dev), st.Ino}] = true
		}
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return ErrRefused
		}
		info, e := d.Info()
		if e != nil {
			return ErrRefused
		}
		if !info.Mode().IsRegular() && !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return ErrRefused
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok && forbidden[[2]uint64{uint64(st.Dev), st.Ino}] {
			return ErrRefused
		}
		return nil
	})
}
func nativeLeader(pid int, uid uint32) bool {
	raw, e := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if e != nil {
		return false
	}
	if validatePosture(raw, uid) != nil {
		return false
	}
	pgid, e := syscall.Getpgid(pid)
	return e == nil && pgid == pid
}

// Validate all UID columns, all five capability sets and NNP; missing,
// duplicated or mismatched facts fail closed for both leader and descendants.
func validatePosture(raw []byte, uid uint32) error {
	required := map[string]string{"Uid": "", "CapInh": "", "CapPrm": "", "CapEff": "", "CapBnd": "", "CapAmb": "", "NoNewPrivs": ""}
	seen := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if _, needed := required[k]; !needed {
			continue
		}
		if seen[k] {
			return ErrRefused
		}
		seen[k] = true
		values := strings.Fields(v)
		if k == "Uid" {
			if len(values) != 4 {
				return ErrRefused
			}
			for _, s := range values {
				n, e := strconv.ParseUint(s, 10, 32)
				if e != nil || uint32(n) != uid {
					return ErrRefused
				}
			}
		} else {
			if len(values) != 1 {
				return ErrRefused
			}
			n, e := strconv.ParseUint(values[0], 16, 64)
			if e != nil {
				return ErrRefused
			}
			if (k == "NoNewPrivs" && values[0] != "1") || (k != "NoNewPrivs" && n != 0) {
				return ErrRefused
			}
		}
	}
	for k := range required {
		if !seen[k] {
			return ErrRefused
		}
	}
	return nil
}
func treeDigest(root string) (string, error) {
	rows := []string{}
	e := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		row := fmt.Sprintf("%s:%o", rel, info.Mode())
		if info.Mode().IsRegular() {
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			h := sha256.Sum256(b)
			row += ":" + hex.EncodeToString(h[:])
		} else if info.Mode()&os.ModeSymlink != 0 {
			target, e := os.Readlink(path)
			if e != nil {
				return e
			}
			row += ":" + target
		}
		rows = append(rows, row)
		return nil
	})
	if e != nil {
		return "", e
	}
	sort.Strings(rows)
	h := sha256.Sum256([]byte(strings.Join(rows, "\n")))
	return hex.EncodeToString(h[:]), nil
}
