//go:build linux

package hostwireless

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// Hold the distribution's normal package-maintenance locks across native
// snapshot/index publication and radio effects. Never run a package manager or
// create missing package locks; active package work is a measured refusal.
func lockWirelessPackageState(root *os.Root) (func(), error) {
	base, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer base.Close()
	probe := &durableWireless{root: root, fd: base}
	var files []*os.File
	release := func() {
		for _, file := range files {
			unlock := unix.Flock_t{Type: unix.F_UNLCK, Whence: 0}
			_ = unix.FcntlFlock(file.Fd(), unix.F_SETLK, &unlock)
			_ = file.Close()
		}
	}
	for _, name := range []string{"var/lib/dpkg/lock-frontend", "var/lib/dpkg/lock"} {
		file, err := probe.open(name, unix.O_RDWR, 0)
		if err != nil {
			release()
			return nil, err
		}
		var st unix.Stat_t
		if err = unix.Fstat(int(file.Fd()), &st); err != nil || st.Uid != uint32(os.Geteuid()) || st.Mode&022 != 0 || st.Mode&unix.S_IFMT != unix.S_IFREG {
			file.Close()
			release()
			return nil, errors.New("untrusted package-maintenance lock")
		}
		lock := unix.Flock_t{Type: unix.F_WRLCK, Whence: 0}
		if err = unix.FcntlFlock(file.Fd(), unix.F_SETLK, &lock); err != nil {
			file.Close()
			release()
			return nil, errors.New("kernel package maintenance is active; retry after it completes")
		}
		files = append(files, file)
	}
	return release, nil
}

func qualifyWirelessScratchParent(root *os.Root) error {
	info, err := root.Stat("var/tmp")
	if err != nil {
		return err
	}
	st, ok := info.Sys().(*unix.Stat_t)
	if !ok {
		// os.FileInfo uses syscall.Stat_t; use the descriptor for one portable
		// Linux metadata representation instead of trusting pathname shape.
		file, e := root.Open("var/tmp")
		if e != nil {
			return e
		}
		defer file.Close()
		var value unix.Stat_t
		if e = unix.Fstat(int(file.Fd()), &value); e != nil {
			return e
		}
		st = &value
	}
	if !info.IsDir() || st.Uid != uint32(os.Geteuid()) || (st.Mode&022 != 0 && st.Mode&unix.S_ISVTX == 0) {
		return errors.New("untrusted native wireless scratch parent")
	}
	return nil
}
