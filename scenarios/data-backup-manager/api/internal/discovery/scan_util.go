package discovery

import (
	"context"
	"io/fs"
	"path/filepath"
)

// boundedDirSize sums regular-file sizes under root, bailing out after
// maxEntries files or on context cancellation. It reads only metadata, never
// file contents, and never fails the scan (unreadable entries are skipped).
//
// Shared by WellKnownScanner and ResourceDataScanner so a target with a huge
// tree (e.g. a multi-gigabyte coding-agent session dir) never stalls the RPC on
// a deep walk.
func boundedDirSize(ctx context.Context, root string, maxEntries int) int64 {
	var total int64
	count := 0
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil || count >= maxEntries {
			return filepath.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		count++
		if info, ierr := d.Info(); ierr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}
