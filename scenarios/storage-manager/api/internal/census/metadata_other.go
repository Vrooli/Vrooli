//go:build !(aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris)

package census

import "os"

type fileIdentity struct {
	device uint64
	inode  uint64
	valid  bool
}

type fileMetadata struct {
	identity  fileIdentity
	device    uint64
	bytes     int64
	allocated int64
}

func inspectPathHost(path string) (fileMetadata, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return fileMetadata{}, err
	}
	return fileMetadata{bytes: info.Size(), allocated: info.Size()}, nil
}

func (hostFileSystem) inspectMetadata(path string) (fileMetadata, error) {
	return inspectPathHost(path)
}
