//go:build unix

package files

import (
	"os"
	"syscall"
)

// changeTime is the inode change time. Tools that restore mtime after a
// write (cp -p, rsync -t, archive extraction) cannot restore it, so a same
// size, same mtime edit is still seen.
func changeTime(info os.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Ctim.Sec*1e9 + st.Ctim.Nsec
	}
	return 0
}
