// effort_control_store_save.go holds the live delivery-board Save path and its
// per-effort lock. Relocated out of the deleted finite commission broker; the
// lock and optimistic revision guard are the delivery board's own concern.
package backlog

import (
	"errors"
	"os"

	platform "github.com/vrooli/platform-go"
	"swarm-manager/internal/identity"
)

// withEffortLock takes the same per-effort file lock every board Save uses, so
// concurrent completion/amend writes serialize and a stale save cannot resurrect
// an earlier revision. The installed root/ancestors must be protected by host
// qualification: refuse symlinks and file-identity changes before and after
// locking.
func (s *FileEffortControlStore) withEffortLock(id string, fn func() error) error {
	p, e := s.path(id)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(s.rootDir, 0o700); e != nil {
		return e
	}
	if info, err := os.Lstat(p + ".lock"); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return identity.ErrRevisionConflict
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	f, e := os.OpenFile(p+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if e != nil {
		return e
	}
	defer f.Close()
	checkLock := func() error {
		info, err := os.Lstat(p + ".lock")
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return identity.ErrRevisionConflict
		}
		opened, err := f.Stat()
		if err != nil || !os.SameFile(opened, info) {
			return identity.ErrRevisionConflict
		}
		return nil
	}
	if e = checkLock(); e != nil {
		return e
	}
	release, e := platform.LockFile(f, false)
	if e != nil {
		return e
	}
	defer release()
	if e = checkLock(); e != nil {
		return e
	}
	return fn()
}

// Save persists one revision under the per-effort lock with an optimistic
// revision guard: a write may only advance the revision by one (or rewrite the
// current revision when the authored authority digest is unchanged).
func (s *FileEffortControlStore) Save(control identity.EffortControl) error {
	return s.withEffortLock(control.EffortID, func() error {
		current, e := s.Load(control.EffortID)
		if e != nil && !errors.Is(e, ErrNotFound) {
			return e
		}
		if e == nil && (control.Revision < current.Revision || control.Revision > current.Revision+1 || (control.Revision == current.Revision && control.AuthorityDigest() != current.AuthorityDigest())) {
			return identity.ErrRevisionConflict
		}
		return s.saveUnlocked(control)
	})
}
