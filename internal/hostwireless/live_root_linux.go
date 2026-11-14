//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// wirelessRootAdmission is produced only after the host operation's measured
// prerequisites pass. It cannot be supplied through CLI JSON or an artifact.
type wirelessRootAdmission struct {
	device              uint64
	inode               uint64
	uid                 uint32
	issued              time.Time
	paths               map[string]string
	independentRecovery bool
}

func (a *wirelessRootAdmission) permits(st unix.Stat_t) bool {
	return a != nil && a.device == uint64(st.Dev) && a.inode == st.Ino && a.uid == uint32(os.Geteuid()) && !a.issued.After(time.Now()) && time.Since(a.issued) < time.Minute
}

func (t *durableWireless) physical(name string) string {
	best := ""
	for prefix := range t.paths {
		if (name == prefix || strings.HasPrefix(name, prefix+"/")) && len(prefix) > len(best) {
			best = prefix
		}
	}
	if best != "" {
		return t.paths[best] + strings.TrimPrefix(name, best)
	}
	return name
}

// Resolve only the distribution's fixed usr-merge and exact ABI header link.
// No arbitrary symlink destination becomes an effect path.
func admitWirelessRootPaths(root *os.Root, uid uint32) (map[string]string, error) {
	paths := map[string]string{}
	info, err := root.Lstat("lib")
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := root.Readlink("lib")
		if err != nil || (link != "usr/lib" && link != "/usr/lib") {
			return nil, errors.New("unsupported /lib indirection")
		}
		paths["lib"] = "usr/lib"
	} else if !info.IsDir() {
		return nil, errors.New("module parent is not a directory")
	}
	build := abiTree + "/build"
	info, err = root.Lstat(build)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := root.Readlink(build)
		expected := "/usr/src/linux-headers-" + nobleABI
		if err != nil || link != expected {
			return nil, errors.New("unsupported exact-ABI header indirection")
		}
		paths[build] = strings.TrimPrefix(expected, "/")
	}
	fd, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	probe := &durableWireless{root: root, fd: fd, paths: paths}
	for _, name := range []string{abiTree, "lib/firmware", abiTree + "/build/Module.symvers"} {
		physical := probe.physical(name)
		parts := strings.Split(physical, "/")
		for i := range parts {
			prefix := strings.Join(parts[:i+1], "/")
			f, err := probe.open(prefix, unix.O_RDONLY|unix.O_NONBLOCK, 0)
			if err != nil {
				return nil, err
			}
			var st unix.Stat_t
			err = unix.Fstat(int(f.Fd()), &st)
			f.Close()
			if err != nil || st.Uid != uid || st.Mode&022 != 0 {
				return nil, fmt.Errorf("untrusted module/firmware/header ancestor: %s", prefix)
			}
		}
	}
	return paths, nil
}

// Snapshot only the reviewed module ABI and explicit protected paths. Walking
// the entire live root would observe unrelated workloads and exceed its scope.
func (t *durableWireless) snapshotTarget(ctx context.Context, indexed *indexedActivation) (map[string]privateTreeEntry, error) {
	if t.paths == nil && t.admission == nil {
		return snapshotPrivateTree(ctx, t.root)
	}
	files := map[string]privateTreeEntry{}
	measure := func(name string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		f, err := t.open(name, unix.O_RDONLY|unix.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		info, err := f.Stat()
		f.Close()
		if err != nil {
			return err
		}
		entry := privateTreeEntry{Mode: info.Mode()}
		if info.Mode().IsRegular() {
			identity, err := t.identity(name)
			if err != nil {
				return err
			}
			entry.SHA256 = identity.SHA256
		} else if !info.IsDir() {
			return errors.New("unsupported stock entry type")
		}
		files[name] = entry
		return nil
	}
	for name := range indexed.stockTree {
		if err := measure(name); err != nil {
			return nil, err
		}
	}
	var walk func(string) error
	walk = func(directory string) error {
		if len(files) > 12100 {
			return errors.New("module snapshot exceeds entry bound")
		}
		f, err := t.open(directory, unix.O_RDONLY|unix.O_DIRECTORY, 0)
		if err != nil {
			return err
		}
		entries, err := f.ReadDir(-1)
		f.Close()
		if err != nil {
			return err
		}
		for _, entry := range entries {
			name := path.Join(directory, entry.Name())
			if name == abiTree+"/build" || name == abiTree+"/vdso" {
				continue
			}
			if !entry.IsDir() && !entry.Type().IsRegular() {
				return fmt.Errorf("unsupported module entry: %s", name)
			}
			if err := measure(name); err != nil {
				return err
			}
			if entry.IsDir() {
				if err := walk(name); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(abiTree); err != nil {
		return nil, err
	}
	return files, nil
}
