//go:build !windows

package privsep

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	osuser "os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

// principal is the account a step runs as on the owner's behalf.
type principal struct {
	uid, gid uint32
	groups   []uint32
	name     string
	home     string
}

// checkoutPrincipal returns the account that owns workDir when this helper
// runs as root and that account is not root. nil means the step runs as the
// helper itself (the helper is not root, or the checkout is root's own).
func checkoutPrincipal(workDir string, clientUID int) (*principal, error) {
	if os.Geteuid() != 0 {
		return nil, nil
	}
	uid := clientUID
	if info, err := os.Stat(workDir); err == nil {
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			uid = int(st.Uid)
		}
	}
	return lookupPrincipal(uid)
}

// clientPrincipal returns the runner account the helper serves, or nil when
// the helper is not root or the runner is unknown.
func clientPrincipal(clientUID int) (*principal, error) {
	if os.Geteuid() != 0 {
		return nil, nil
	}
	return lookupPrincipal(clientUID)
}

func lookupPrincipal(uid int) (*principal, error) {
	if uid <= 0 {
		return nil, nil
	}
	u, err := osuser.LookupId(strconv.Itoa(uid))
	if err != nil {
		return nil, fmt.Errorf("resolve checkout owner uid %d: %w", uid, err)
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("resolve checkout owner gid %q: %w", u.Gid, err)
	}
	p := &principal{uid: uint32(uid), gid: uint32(gid), name: u.Username, home: u.HomeDir}
	if ids, err := u.GroupIds(); err == nil {
		for _, id := range ids {
			if value, err := strconv.ParseUint(id, 10, 32); err == nil {
				p.groups = append(p.groups, uint32(value))
			}
		}
	}
	return p, nil
}

// apply makes cmd run as the principal with the principal's environment.
func (p *principal) apply(cmd *exec.Cmd) {
	if p == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: p.uid, Gid: p.gid, Groups: p.groups}}
	base := cmd.Env
	if base == nil {
		base = os.Environ()
	}
	cmd.Env = ownerEnv(p.home, p.name, base)
}

// chownTree gives every entry under root to the principal. It follows no
// symlinks (Lchown), so it cannot be steered outside root.
func chownTree(root string, p *principal) error {
	if p == nil {
		return nil
	}
	return filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		return os.Lchown(path, int(p.uid), int(p.gid))
	})
}
