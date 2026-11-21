//go:build linux

package hostwireless

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"

	"golang.org/x/sys/unix"
)

func validJournalID(id string) bool {
	data, err := hex.DecodeString(id)
	return err == nil && len(data) == 16
}
func strictWirelessJSON(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("trailing wireless journal data")
	}
	return nil
}
func (t *durableWireless) statePath(name string) string {
	return path.Join(wirelessJournalDir, t.intent.ID, name)
}
func (t *durableWireless) check(point string) error {
	if t.fault != nil {
		return t.fault(point)
	}
	return nil
}

// Every pathname operation is descriptor-relative and rejects all symlinks,
// including in-root links. A future live adapter must resolve usr-merge before
// admission instead of weakening this policy.
func (t *durableWireless) open(name string, flags int, mode uint32) (*os.File, error) {
	if !fs.ValidPath(name) {
		return nil, errors.New("invalid wireless path")
	}
	how := &unix.OpenHow{Flags: uint64((flags &^ unix.O_TRUNC) | unix.O_CLOEXEC | unix.O_NOFOLLOW), Mode: uint64(mode), Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS}
	fd, err := unix.Openat2(int(t.fd.Fd()), t.physical(name), how)
	if err != nil {
		return nil, err
	}
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil {
		unix.Close(fd)
		return nil, err
	}
	if st.Mode&unix.S_IFMT == unix.S_IFREG && st.Nlink != 1 {
		unix.Close(fd)
		return nil, errors.New("wireless hard-linked file rejected")
	}
	if flags&unix.O_TRUNC != 0 {
		if err = unix.Ftruncate(fd, 0); err != nil {
			unix.Close(fd)
			return nil, err
		}
	}
	return os.NewFile(uintptr(fd), name), nil
}
func (t *durableWireless) parent(name string) (*os.File, string, error) {
	f, e := t.open(path.Dir(name), unix.O_RDONLY|unix.O_DIRECTORY, 0)
	return f, path.Base(name), e
}
func (t *durableWireless) syncDir(name string) error {
	if err := t.check("before-directory-sync:" + name); err != nil {
		return err
	}
	f, e := t.open(name, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if e != nil {
		return e
	}
	if err := errors.Join(f.Sync(), f.Close()); err != nil {
		return err
	}
	return t.check("after-directory-sync:" + name)
}
func (t *durableWireless) syncParents(name string) error {
	for dir := path.Dir(name); ; dir = path.Dir(dir) {
		if err := t.syncDir(dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if dir == "." {
			return nil
		}
	}
}
func (t *durableWireless) mkdir(name string) error {
	if t.readOnly {
		return errors.New("wireless preview cannot create directories")
	}
	f, base, err := t.parent(name)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = unix.Mkdirat(int(f.Fd()), base, 0700); err != nil {
		return err
	}
	if err = t.check("after-mkdir:" + name); err != nil {
		return err
	}
	return errors.Join(f.Sync(), t.syncDir(name))
}
func (t *durableWireless) makeStateDirs() error {
	for _, dir := range []string{"var", "var/lib", "var/lib/vrooli", wirelessJournalDir} {
		f, err := t.open(dir, unix.O_RDONLY|unix.O_DIRECTORY, 0)
		if err == nil {
			var st unix.Stat_t
			if statErr := unix.Fstat(int(f.Fd()), &st); statErr != nil || st.Uid != uint32(os.Geteuid()) || st.Mode&022 != 0 || (dir == wirelessJournalDir && st.Mode&07777 != 0700) {
				f.Close()
				return errors.New("wireless state directory is not private")
			}
			f.Close()
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err = t.mkdir(dir); err != nil {
			return err
		}
	}
	return errors.Join(t.syncDir(wirelessJournalDir), t.syncParents(wirelessJournalDir))
}

func (t *durableWireless) qualifyStateDirs() error {
	for _, name := range []string{"var", "var/lib", "var/lib/vrooli", wirelessJournalDir} {
		f, err := t.open(name, unix.O_RDONLY|unix.O_DIRECTORY, 0)
		if err != nil {
			return err
		}
		var st unix.Stat_t
		err = unix.Fstat(int(f.Fd()), &st)
		f.Close()
		if err != nil || st.Uid != uint32(os.Geteuid()) || st.Mode&022 != 0 || (name == wirelessJournalDir && st.Mode&07777 != 0700) {
			return errors.New("untrusted wireless journal ancestor")
		}
	}
	return nil
}

func (t *durableWireless) qualifyTransactionDir(id string) error {
	if !validJournalID(id) {
		return errors.New("invalid wireless transaction id")
	}
	f, err := t.open(path.Join(wirelessJournalDir, id), unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	var st unix.Stat_t
	if err = unix.Fstat(int(f.Fd()), &st); err != nil {
		return err
	}
	if st.Uid != uint32(os.Geteuid()) || st.Mode&07777 != 0700 {
		return errors.New("wireless transaction directory is not private")
	}
	return nil
}

func (t *durableWireless) readState(name string, limit int64) ([]byte, error) {
	f, err := t.open(name, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var st unix.Stat_t
	if err = unix.Fstat(int(f.Fd()), &st); err != nil {
		return nil, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&07777 != 0600 || st.Size > limit {
		return nil, errors.New("wireless journal file is not a private bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(data)) > limit {
		return nil, errors.New("wireless journal file exceeds bound")
	}
	return data, err
}
func (t *durableWireless) read(name string, limit int64) ([]byte, error) {
	f, e := t.open(name, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("wireless file outside bounds")
	}
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(b)) > limit {
		return nil, errors.New("wireless read exceeds bounds")
	}
	return b, e
}
func (t *durableWireless) identity(name string) (durableFileIdentity, error) {
	f, e := t.open(name, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if errors.Is(e, fs.ErrNotExist) {
		return durableFileIdentity{Absent: true}, nil
	}
	if e != nil {
		return durableFileIdentity{}, e
	}
	defer f.Close()
	var st unix.Stat_t
	if e = unix.Fstat(int(f.Fd()), &st); e != nil {
		return durableFileIdentity{}, e
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Size > maxPreservedBytes {
		return durableFileIdentity{}, errors.New("wireless target is not a bounded regular file")
	}
	b, e := io.ReadAll(io.LimitReader(f, maxPreservedBytes+1))
	if e != nil || len(b) > maxPreservedBytes {
		return durableFileIdentity{}, errors.New("wireless target exceeds bounds")
	}
	return durableFileIdentity{SHA256: digest(b), Mode: st.Mode & 07777, UID: st.Uid, GID: st.Gid}, nil
}

// atomic publishes bytes from the trusted journal filesystem. Both parent
// directories are fsynced. Temporary files are confined to the private journal;
// no unclassified live temporary pathname is deleted during recovery.
func (t *durableWireless) atomic(name string, data []byte, identity durableFileIdentity, expected ...durableFileIdentity) error {
	if t.readOnly {
		return errors.New("wireless preview cannot publish files")
	}
	if err := t.check("before-write:" + name); err != nil {
		return err
	}
	temp := path.Join(wirelessJournalDir, "pending")
	f, err := t.open(temp, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		var st unix.Stat_t
		err = unix.Fstat(int(f.Fd()), &st)
		if err == nil && (st.Uid != identity.UID || st.Gid != identity.GID) {
			err = unix.Fchown(int(f.Fd()), int(identity.UID), int(identity.GID))
		}
	}
	if err == nil {
		err = unix.Fchmod(int(f.Fd()), identity.Mode)
	}
	if err == nil {
		err = t.check("before-file-sync:" + name)
	}
	if err == nil {
		err = f.Sync()
	}
	if err == nil {
		err = t.check("after-file-sync:" + name)
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = t.check("before-rename:" + name); err != nil {
		return err
	}
	if len(expected) > 0 {
		got, err := t.identity(name)
		if err != nil || got != expected[0] {
			return errors.New("wireless target changed immediately before publication")
		}
	}
	from, fromBase, err := t.parent(temp)
	if err != nil {
		return err
	}
	defer from.Close()
	to, toBase, err := t.parent(name)
	if err != nil {
		return err
	}
	defer to.Close()
	if err = unix.Renameat(int(from.Fd()), fromBase, int(to.Fd()), toBase); err != nil {
		return err
	}
	if err = t.check("after-rename:" + name); err != nil {
		return err
	}
	if err = errors.Join(from.Sync(), to.Sync()); err != nil {
		return err
	}
	return t.check("after-sync:" + name)
}
func (t *durableWireless) ownIdentity(mode uint32) durableFileIdentity {
	return durableFileIdentity{Mode: mode, UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}
}
func (t *durableWireless) writeState(name string, data []byte) error {
	if t.readOnly {
		return errors.New("wireless preview cannot write journal state")
	}
	return t.atomic(name, data, t.ownIdentity(0600))
}
func (t *durableWireless) setPhase(phase string) error {
	if err := t.writeState(t.statePath("phase"), []byte(phase)); err != nil {
		return err
	}
	t.phase = phase
	return nil
}
