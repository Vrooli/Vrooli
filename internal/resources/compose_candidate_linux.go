package resources

import "golang.org/x/sys/unix"

func publishComposeCandidate(stage, destination string) error {
	return unix.Renameat2(unix.AT_FDCWD, stage, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
}
func composeCandidatePlatformSupported() bool { return true }
