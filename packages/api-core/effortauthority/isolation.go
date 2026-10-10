package effortauthority

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type ProtectedProcess struct {
	PID              int
	ExecutableSHA256 string
}

// NativeUIDIsolation qualifies a proposed distinct-UID host topology from real
// proc/filesystem evidence on that host. Setup selects the production launcher's
// disposable probe and exact built issuer/PM/AM service processes; callers never
// submit these fields through HTTP. A boolean or mode0600 is not evidence.
// The probe must use the same UID/filesystem view as the selected native launch
// contract. Container-only or sameUID isolation needs its own separately reviewed
// qualifier and is deliberately unsupported by this distinct-UID mechanism.
// This checks current host DAC/process privilege evidence. Runtime ownership
// must separately verify the selected native launcher/build, probe identity,
// namespace/filesystem view and non-bypassable UID/capability/NNP inheritance
// for every native descendant; this guard does not infer them from a label.
// Without that exact deployment authority evidence, installation remains held.
type NativeUIDIsolation struct {
	NativeUID      uint32
	NativeProbePID int
	Services       map[string]ProtectedProcess
	// Every ledger, custody handle file and PM/AM peer credential path must be
	// included by reviewed setup, with private parent directories. No file content
	// is read here; only executable bytes and filesystem ownership/modes are read.
	ProtectedPaths []string
}

func kernelUID(ctx context.Context, pid int) (uint32, error) {
	if ctx.Err() != nil || pid <= 0 {
		return 0, ErrRefused
	}
	prefix := filepath.Join("/proc", strconv.Itoa(pid))
	status, e := os.ReadFile(filepath.Join(prefix, "status"))
	if e != nil {
		return 0, ErrRefused
	}
	var uid uint32
	found := false
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			values := strings.Fields(line)
			if len(values) != 5 {
				return 0, ErrRefused
			}
			for i, v := range values[1:] {
				n, e := strconv.ParseUint(v, 10, 32)
				if e != nil {
					return 0, ErrRefused
				}
				if i == 0 {
					uid = uint32(n)
				} else if uint32(n) != uid {
					return 0, ErrRefused
				}
			}
			found = true
			break
		}
	}
	if !found {
		return 0, ErrRefused
	}
	return uid, nil
}
func processUID(ctx context.Context, p ProtectedProcess) (uint32, error) {
	if !validDigest(p.ExecutableSHA256) {
		return 0, ErrRefused
	}
	uid, e := kernelUID(ctx, p.PID)
	if e != nil {
		return 0, e
	}
	prefix := filepath.Join("/proc", strconv.Itoa(p.PID))
	executable, e := os.ReadFile(filepath.Join(prefix, "exe"))
	if e != nil {
		return 0, ErrRefused
	}
	digest := sha256.Sum256(executable)
	if hex.EncodeToString(digest[:]) != p.ExecutableSHA256 {
		return 0, ErrRefused
	}
	return uid, nil
}

// nativePrivilegePosture requires all capability sets dropped and no-new-privs
// on the exact disposable launch probe. Missing/duplicate fields fail closed.
// UID separation without this kernel posture cannot prove custody protection.
func nativePrivilegePosture(status []byte) error {
	required := map[string]bool{"CapInh": false, "CapPrm": false, "CapEff": false, "CapBnd": false, "CapAmb": false, "NoNewPrivs": false}
	for _, line := range strings.Split(string(status), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ":")
		seen, needed := required[name]
		if !needed {
			continue
		}
		if seen || len(fields) != 2 {
			return ErrRefused
		}
		required[name] = true
		if name == "NoNewPrivs" {
			if fields[1] != "1" {
				return ErrRefused
			}
		} else {
			value, e := strconv.ParseUint(fields[1], 16, 64)
			if e != nil || value != 0 {
				return ErrRefused
			}
		}
	}
	for _, seen := range required {
		if !seen {
			return ErrRefused
		}
	}
	return nil
}

func (q *NativeUIDIsolation) Qualify(ctx context.Context) error {
	if q == nil || q.NativeUID == 0 || len(q.Services) != 3 || len(q.ProtectedPaths) < 2 {
		return ErrRefused
	}
	native, e := kernelUID(ctx, q.NativeProbePID)
	if e != nil || native != q.NativeUID {
		return ErrRefused
	}
	posture, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(q.NativeProbePID), "status"))
	if err != nil || nativePrivilegePosture(posture) != nil {
		return ErrRefused
	}
	serviceUIDs := map[uint32]bool{}
	for _, role := range []string{"issuer", "prompt-manager", "agent-manager"} {
		p, exists := q.Services[role]
		if !exists {
			return ErrRefused
		}
		uid, e := processUID(ctx, p)
		if e != nil || uid == native {
			return ErrRefused
		}
		serviceUIDs[uid] = true
	}
	for _, path := range q.ProtectedPaths {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return ErrRefused
		}
		original := true
		for current := path; current != "/"; current = filepath.Dir(current) {
			info, e := os.Lstat(current)
			if e != nil || info.Mode()&os.ModeSymlink != 0 {
				return ErrRefused
			}
			stat, ok := info.Sys().(*syscall.Stat_t)
			if !ok || stat.Uid == native {
				return ErrRefused
			}
			if original {
				if !serviceUIDs[stat.Uid] || info.Mode().Perm()&0077 != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
					return ErrRefused
				}
			} else if !info.IsDir() || info.Mode().Perm()&0022 != 0 {
				return ErrRefused
			}
			original = false
		}
	}
	return nil
}
