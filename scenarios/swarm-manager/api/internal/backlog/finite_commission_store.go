// finite_commission_store.go serializes finite disposition with existing effort writes.
package backlog

import (
	"encoding/json"
	"errors"
	platform "github.com/vrooli/platform-go"
	"os"
	"reflect"
	"swarm-manager/internal/identity"
)

// The same per-effort lock protects every existing Save and the owner-only
// finite transform. A stale completion/amend save cannot resurrect revocation.
func (s *FileEffortControlStore) withFiniteLock(id string, fn func() error) error {
	p, e := s.path(id)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(s.rootDir, 0700); e != nil {
		return e
	}
	// The installed root/ancestors must be protected by host qualification.
	// Refuse symlinks and file-identity changes before/after owner locking.
	if info, err := os.Lstat(p + ".lock"); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return identity.ErrRevisionConflict
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	f, e := os.OpenFile(p+".lock", os.O_CREATE|os.O_RDWR, 0600)
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
func (s *FileEffortControlStore) Save(control identity.EffortControl) error {
	return s.withFiniteLock(control.EffortID, func() error {
		current, e := s.Load(control.EffortID)
		if e != nil && !errors.Is(e, ErrNotFound) {
			return e
		}
		if e == nil && (control.Revision < current.Revision || control.Revision > current.Revision+1 || (control.Revision == current.Revision && control.AuthorityDigest() != current.AuthorityDigest())) {
			return identity.ErrRevisionConflict
		}
		if !reflect.DeepEqual(control.FiniteCommission, current.FiniteCommission) || !reflect.DeepEqual(control.Development, current.Development) {
			return identity.ErrRevisionConflict
		}
		return s.saveUnlocked(control)
	})
}
func (s *FileEffortControlStore) updateFinite(id string, expectedDigest string, transform func(identity.EffortControl) (identity.EffortControl, error)) error {
	return s.withFiniteLock(id, func() error {
		current, e := s.Load(id)
		if e != nil {
			return e
		}
		if current.AuthorityDigest() != expectedDigest {
			return identity.ErrRevisionConflict
		}
		next, e := transform(current)
		if e != nil {
			return e
		}
		// The finite owner may change only its nested disposition.
		a, b := current, next
		a.FiniteCommission = nil
		b.FiniteCommission = nil
		ar, _ := json.Marshal(a)
		br, _ := json.Marshal(b)
		if string(ar) != string(br) {
			return identity.ErrRevisionConflict
		}
		return s.saveUnlocked(next)
	})
}
