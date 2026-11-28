//go:build linux

// Package storageactivity owns the control-plane observation of temporary
// paths held by a user's processes. It reads metadata only and never deletes.
package storageactivity

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Snapshot struct {
	UID        uint32    `json:"uid"`
	ObservedAt time.Time `json:"observed_at"`
	Complete   bool      `json:"complete"`
	Paths      []string  `json:"paths"`
	Roots      []string  `json:"roots"`
	Error      string    `json:"error,omitempty"`
}

func Observe(ctx context.Context, uid uint32) (Snapshot, error) {
	if uid == 0 {
		return Snapshot{}, fmt.Errorf("a non-root target uid is required")
	}
	account, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return Snapshot{}, err
	}
	if !filepath.IsAbs(account.HomeDir) {
		return Snapshot{}, fmt.Errorf("target user has no absolute home")
	}
	return observe(ctx, "/proc", uid, []string{"/tmp", filepath.Join(account.HomeDir, ".vrooli", "tmp", "go-work")})
}

func observe(ctx context.Context, procRoot string, uid uint32, roots []string) (Snapshot, error) {
	out := Snapshot{UID: uid, ObservedAt: time.Now().UTC(), Paths: []string{}, Roots: append([]string(nil), roots...)}
	paths := map[string]bool{}
	ownedProcess := false
	referenceBytes := 0
	add := func(path string) {
		path = filepath.Clean(strings.TrimSuffix(path, " (deleted)"))
		for _, root := range roots {
			if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
				// A foreign/root process may hold this user's file. Include that
				// reference, without publishing another user's private temporary paths.
				if !ownedProcess {
					info, err := os.Lstat(path)
					if err != nil {
						continue
					}
					stat, ok := info.Sys().(*syscall.Stat_t)
					if !ok || stat.Uid != uid {
						continue
					}
				}
				if !paths[path] {
					referenceBytes += len(path)
				}
				paths[path] = true
			}
		}
	}
	fail := func(err error) (Snapshot, error) { out.Error = err.Error(); return out, err }
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	processes, err := os.ReadDir(procRoot)
	if err != nil {
		return fail(err)
	}
	if len(processes) > 65536 {
		return fail(fmt.Errorf("process observation limit exceeded"))
	}
	for _, process := range processes {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		if _, err := strconv.ParseUint(process.Name(), 10, 32); err != nil || !process.IsDir() {
			continue
		}
		root := filepath.Join(procRoot, process.Name())
		info, err := os.Stat(root)
		if processExited(err) {
			continue
		}
		if err != nil {
			return fail(err)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fail(fmt.Errorf("process ownership unavailable"))
		}
		ownedProcess = stat.Uid == uid
		fds, err := os.ReadDir(filepath.Join(root, "fd"))
		if processExited(err) {
			continue
		}
		if err != nil {
			return fail(fmt.Errorf("process %s descriptors: %w", process.Name(), err))
		}
		if len(fds) > 65536 {
			return fail(fmt.Errorf("process descriptor limit exceeded"))
		}
		for _, fd := range fds {
			if err := ctx.Err(); err != nil {
				return fail(err)
			}
			path := filepath.Join(root, "fd", fd.Name())
			// Directory streams include storage-manager's paused metadata cursors.
			// CWD below is the active directory reference, rather than a read stream.
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				continue
			}
			target, err := os.Readlink(path)
			if processExited(err) {
				continue
			}
			if err != nil {
				return fail(err)
			}
			add(target)
		}
		for _, name := range []string{"cwd", "exe"} {
			target, err := os.Readlink(filepath.Join(root, name))
			if processExited(err) {
				continue
			}
			if err != nil {
				return fail(fmt.Errorf("process %s %s: %w", process.Name(), name, err))
			}
			add(target)
		}
		// A build can need its cache again without holding a file open right now.
		// Record only storage variables, only below the temporary namespaces.
		environment, envErr := os.Open(filepath.Join(root, "environ"))
		if envErr != nil && !processExited(envErr) {
			return fail(fmt.Errorf("process %s environment: %w", process.Name(), envErr))
		}
		if envErr == nil {
			data, readErr := io.ReadAll(io.LimitReader(environment, (1<<20)+1))
			_ = environment.Close()
			if readErr != nil && !processExited(readErr) {
				return fail(readErr)
			}
			if len(data) > 1<<20 {
				return fail(fmt.Errorf("process environment limit exceeded"))
			}
			for _, variable := range strings.Split(string(data), "\x00") {
				key, value, ok := strings.Cut(variable, "=")
				if !ok {
					continue
				}
				switch key {
				case "HOME", "VROOLI_HOME", "GOCACHE", "GOMODCACHE", "GOTMPDIR", "TMPDIR", "TMP", "TEMP", "SCENARIO_DATA_DIR", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CONFIG_HOME":
					clean := filepath.Clean(value)
					generic := false
					for _, root := range roots {
						generic = generic || clean == root
					}
					if filepath.IsAbs(clean) && !generic {
						add(clean)
					}
				}
			}
		}
		f, err := os.Open(filepath.Join(root, "maps"))
		if processExited(err) {
			continue
		}
		if err != nil {
			return fail(fmt.Errorf("process %s mappings: %w", process.Name(), err))
		}
		scan := bufio.NewScanner(io.LimitReader(f, 16<<20))
		scan.Buffer(make([]byte, 4096), 1<<20)
		var consumed int
		for scan.Scan() {
			if err := ctx.Err(); err != nil {
				_ = f.Close()
				return fail(err)
			}
			line := scan.Text()
			consumed += len(line) + 1
			if mapped := mappedPath(line); mapped != "" {
				add(mapped)
			}
		}
		scanErr := scan.Err()
		_ = f.Close()
		if scanErr != nil && !processExited(scanErr) {
			return fail(scanErr)
		}
		if consumed >= 16<<20 {
			return fail(fmt.Errorf("process mappings limit exceeded"))
		}
		if len(paths) > 65536 || referenceBytes > 4<<20 {
			return fail(fmt.Errorf("temporary reference limit exceeded"))
		}
	}
	for path := range paths {
		out.Paths = append(out.Paths, path)
	}
	sort.Strings(out.Paths)
	out.Complete = true
	return out, nil
}

// Procfs may return ESRCH after exit (or when no userspace memory remains),
// rather than ENOENT. Neither is a permission refusal by a live process.
func processExited(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}

// The first five fields are kernel metadata; the remaining path may contain
// repeated spaces, so tokenizing the path would lose a live reference.
func mappedPath(line string) string {
	for i := 0; i < 5; i++ {
		line = strings.TrimLeft(line, " \t")
		end := strings.IndexAny(line, " \t")
		if end < 0 {
			return ""
		}
		line = line[end:]
	}
	return strings.TrimLeft(line, " \t")
}
