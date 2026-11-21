//go:build darwin

package platform

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type darwinOwnedHost struct{}

func (darwinOwnedHost) Run(ctx context.Context, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, "/bin/launchctl", args...)
	c.Env = []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin", "LC_ALL=C"}
	return c.CombinedOutput()
}

// The ProcessOnly lane is a closed maintained entrypoint, not a generic exec
// service. The host CLI validates its capsule and applies the resource's one
// no-children/transport profile before handing off to the trusted vendor.
func (darwinOwnedHost) ValidateLaunch(ctx context.Context, i OperationIntent, s OperationSpec) error {
	return (darwinMaintainedLaunchGuard{approved: openInstalledNativeSupervisorImage, verifyPath: verifyImmutableNativePath, published: OpenPublishedNativeLauncher}).validate(ctx, i, s)
}

// Resolution is read-only and can be used by explicit elevated setup. It
// neither switches credentials nor authorizes execution under another account.
func resolveDarwinOwnedAccount(ctx context.Context, principal string) (darwinOwnedAccount, error) {
	if e := ctx.Err(); e != nil {
		return darwinOwnedAccount{}, e
	}
	if !validDarwinOwnedAccount(principal) {
		return darwinOwnedAccount{}, ErrOperationIdentity
	}
	uidText := strings.TrimPrefix(principal, "uid:")
	uid, e := strconv.ParseUint(uidText, 10, 32)
	if e != nil || uid == 0 {
		return darwinOwnedAccount{}, ErrOperationIdentity
	}
	u, e := user.LookupId(uidText)
	if e != nil {
		return darwinOwnedAccount{}, e
	}
	canonical, e := user.Lookup(u.Username)
	if e != nil {
		return darwinOwnedAccount{}, e
	}
	if u.Uid != uidText || canonical.Uid != uidText || canonical.Username != u.Username || canonical.Gid != u.Gid || canonical.HomeDir != u.HomeDir {
		return darwinOwnedAccount{}, ErrOperationIdentity
	}
	gid, e := strconv.ParseUint(u.Gid, 10, 32)
	if e != nil || gid == 0 {
		return darwinOwnedAccount{}, ErrOperationIdentity
	}
	return darwinOwnedAccount{Name: principal, UID: uint32(uid), GID: uint32(gid)}, nil
}
func (darwinOwnedHost) Account(ctx context.Context, principal string) (darwinOwnedAccount, error) {
	a, e := resolveDarwinOwnedAccount(ctx, principal)
	if e != nil {
		return a, e
	}
	if os.Getuid() == 0 || os.Getuid() != os.Geteuid() || uint32(os.Getuid()) != a.UID || os.Getgid() != os.Getegid() || uint32(os.Getgid()) != a.GID {
		return darwinOwnedAccount{}, ErrOperationIdentity
	}
	return a, nil
}
func (darwinOwnedHost) Boot(ctx context.Context) (string, error) {
	if e := ctx.Err(); e != nil {
		return "", e
	}
	s, e := unix.Sysctl("kern.bootsessionuuid")
	s = strings.ToLower(strings.TrimSpace(s))
	if e != nil {
		return "", e
	}
	if !serviceBootIDPattern.MatchString(s) {
		return "", ErrOperationIdentity
	}
	return s, nil
}
func (darwinOwnedHost) Snapshot(ctx context.Context, domain, label, expected string) (darwinJobSnapshot, bool, error) {
	if e := ctx.Err(); e != nil {
		return darwinJobSnapshot{}, false, e
	}
	if domain != "user/"+strconv.Itoa(os.Getuid()) || os.Getuid() == 0 || os.Getuid() != os.Geteuid() {
		return darwinJobSnapshot{}, false, ErrOperationIdentity
	}
	return darwinManagerSnapshot(false, label, expected)
}
func (darwinOwnedHost) Process(ctx context.Context, pid int, uid uint32) (darwinOwnedProcess, bool, error) {
	if e := ctx.Err(); e != nil {
		return darwinOwnedProcess{}, false, e
	}
	if pid <= 0 || uid == 0 {
		return darwinOwnedProcess{}, false, ErrOperationIdentity
	}
	info, e := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if e != nil {
		if e == unix.ESRCH || e == unix.ENOENT {
			return darwinOwnedProcess{}, false, nil
		}
		return darwinOwnedProcess{}, false, e
	}
	if info == nil || int(info.Proc.P_pid) != pid || info.Proc.P_stat == darwinProcessZombie {
		return darwinOwnedProcess{}, false, nil
	}
	stamp, e := darwinObservationStamp(pid)
	if e != nil {
		return darwinOwnedProcess{}, false, e
	}
	after, e := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if e != nil {
		return darwinOwnedProcess{}, false, e
	}
	if after == nil || info.Proc.P_starttime != after.Proc.P_starttime || info.Eproc.Ucred.Uid != uid || info.Eproc.Pcred.P_ruid != uid || info.Eproc.Pcred.P_svuid != uid || after.Eproc.Ucred.Uid != uid || after.Eproc.Pcred.P_ruid != uid || after.Eproc.Pcred.P_svuid != uid {
		return darwinOwnedProcess{}, false, ErrOperationIdentity
	}
	end, e := darwinObservationStamp(pid)
	if e != nil || stamp != end {
		return darwinOwnedProcess{}, false, ErrOperationIdentity
	}
	return darwinOwnedProcess{PID: pid, Seconds: info.Proc.P_starttime.Sec, Micros: int64(info.Proc.P_starttime.Usec), UniqueID: stamp[2], ExecVersion: uint32(stamp[3])}, true, nil
}
func darwinOwnedPrivatePath(path string, directory bool) error {
	st, e := os.Lstat(path)
	if e != nil {
		return e
	}
	s, ok := st.Sys().(*syscall.Stat_t)
	if !ok || s.Uid != uint32(os.Getuid()) || st.Mode().Perm()&0077 != 0 || st.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid) != 0 || directory != st.IsDir() || !directory && (!st.Mode().IsRegular() || s.Nlink != 1) {
		return fmt.Errorf("%w: unsafe operator ownership path", ErrOperationIdentity)
	}
	return darwinOperatorPathACL(path, st)
}
func nativeDarwinOwnedBackend() darwinOwnedBackend {
	u, lookupErr := user.LookupId(strconv.Itoa(os.Getuid()))
	root := ""
	if lookupErr == nil {
		root = filepath.Join(u.HomeDir, ".vrooli", "native-owned-launchd")
	}
	prepare := func(create bool) error {
		if lookupErr != nil {
			return lookupErr
		}
		if _, e := (darwinOwnedHost{}).Account(context.Background(), "uid:"+strconv.Itoa(os.Getuid())); e != nil {
			return e
		}
		// Anchor the complete path, including root-owned home ancestors. A safe
		// ledger inode under a foreign-writable parent is not a frozen-intent proof.
		for p := u.HomeDir; ; p = filepath.Dir(p) {
			st, e := os.Lstat(p)
			if e != nil {
				return e
			}
			if e = darwinOperatorPathACL(p, st); e != nil {
				return e
			}
			if p == filepath.Dir(p) {
				break
			}
		}
		for _, p := range []string{u.HomeDir, filepath.Join(u.HomeDir, ".vrooli"), root} {
			st, e := os.Lstat(p)
			if os.IsNotExist(e) && create && p != u.HomeDir {
				if e = os.Mkdir(p, 0700); e != nil {
					return e
				}
				st, e = os.Lstat(p)
			}
			if e != nil {
				return e
			}
			native, ok := st.Sys().(*syscall.Stat_t)
			if !ok || native.Uid != uint32(os.Getuid()) || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0022 != 0 {
				return ErrOperationIdentity
			}
			if p == root && st.Mode().Perm() != 0700 {
				return ErrOperationIdentity
			}
			if e = darwinOperatorPathACL(p, st); e != nil {
				return e
			}
		}
		return nil
	}
	return darwinOwnedBackend{native: darwinOwnedHost{}, store: darwinOwnedFileStore{root: root, plistDir: root, verify: darwinOwnedPrivatePath, prepare: func() error { return prepare(true) }, prepareRead: func() error { return prepare(false) }}}
}
func startOwnedOperation(ctx context.Context, i OperationIntent, s OperationSpec) (*OwnedOperation, error) {
	return nativeDarwinOwnedBackend().start(ctx, i, s)
}
func lookupOwnedOperation(ctx context.Context, i OperationIntent) (OperationRef, error) {
	return nativeDarwinOwnedBackend().lookup(ctx, i)
}
func inspectOwnedOperation(ctx context.Context, r OperationRef) (OperationObservation, error) {
	return nativeDarwinOwnedBackend().inspect(ctx, r)
}
func stopOwnedOperation(ctx context.Context, r OperationRef) (OperationStopReceipt, error) {
	return nativeDarwinOwnedBackend().stop(ctx, r)
}
func releaseOwnedOperation(ctx context.Context, r OperationRef) error {
	return nativeDarwinOwnedBackend().release(ctx, r)
}
func validateNativeOperationRef(r OperationRef) error { return validateDarwinOwnedRef(r) }
func resolveOwnedServiceAccount(ctx context.Context, name string) (OwnedServiceAccount, error) {
	a, e := resolveDarwinOwnedAccount(ctx, name)
	if e != nil {
		return OwnedServiceAccount{}, e
	}
	return OwnedServiceAccount{Name: a.Name, Principal: a.Name}, nil
}
func prepareOwnedServiceAccount(ctx context.Context, name string) (OwnedServiceAccount, error) {
	return resolveOwnedServiceAccount(ctx, name)
}
