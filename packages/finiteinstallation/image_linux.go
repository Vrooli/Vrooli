//go:build linux

package finiteinstallation

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/vrooli/api-core/effortauthority"
	"golang.org/x/sys/unix"
	"io"
	"os"
)

// The proc executable FD identifies the running image even when its deployment
// path was replaced or unlinked. No executable pathname is used as evidence.
func executingBuild() (string, error) {
	f, e := os.Open("/proc/self/exe")
	if e != nil {
		return "", effortauthority.ErrRefused
	}
	defer f.Close()
	return hashImageFD(f)
}
func hashImageFD(f *os.File) (string, error) {
	if f == nil {
		return "", effortauthority.ErrRefused
	}
	var before, after unix.Stat_t
	if unix.Fstat(int(f.Fd()), &before) != nil || before.Mode&unix.S_IFMT != unix.S_IFREG || before.Size <= 0 {
		return "", effortauthority.ErrRefused
	}
	h := sha256.New()
	n, e := io.Copy(h, io.NewSectionReader(f, 0, before.Size+1))
	if e != nil || n != before.Size || unix.Fstat(int(f.Fd()), &after) != nil || before.Dev != after.Dev || before.Ino != after.Ino || before.Size != after.Size || before.Mtim != after.Mtim || before.Ctim != after.Ctim {
		return "", effortauthority.ErrRefused
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
