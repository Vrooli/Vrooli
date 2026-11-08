package isolation

import "golang.org/x/sys/unix"

func publishSQLiteDirectory(stage, final string) error {
	return unix.Renameat2(unix.AT_FDCWD, stage, unix.AT_FDCWD, final, unix.RENAME_NOREPLACE)
}
