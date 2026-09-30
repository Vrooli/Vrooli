package process

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Scenario processes write their stdout and stderr straight into files under
// the runtime logs root, and the lifecycle only moved the previous file aside
// when a process restarted. A process that runs for weeks therefore grew its
// log without limit: on minimouse (a Mac mini with an 8 GB disk budget for
// Vrooli) the ui-health API log reached 75 MB and the logs root 294 MB after a
// month. BoundLogs is the size bound, run periodically by the runtime
// supervisor.

// DefaultLogFileLimit is the allocated size above which a log file is trimmed.
const DefaultLogFileLimit int64 = 16 << 20

// DefaultLogTailKeep is how much of the end of a trimmed file is preserved in
// its ".1" sibling, so the most recent output survives a trim.
const DefaultLogTailKeep int64 = 2 << 20

// maxLogBoundFiles caps one sweep so a pathological logs tree cannot turn the
// supervisor tick into an unbounded walk.
const maxLogBoundFiles = 4096

// LogBoundResult reports one sweep.
type LogBoundResult struct {
	Scanned        int
	Trimmed        []string
	ReclaimedBytes int64
	Errors         []string
}

// LogsRoot resolves <home>/.vrooli/logs, the tree BoundLogs keeps bounded.
func LogsRoot(home string) (string, error) {
	return repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyLogs)
}

// BoundLogs trims every *.log and *.log.bak file under root whose allocated
// size exceeds limit. The last keep bytes are copied to "<file>.1" (replacing
// an older one) and the file is truncated in place, then a one-line marker is
// appended saying where the earlier output went.
//
// Truncating in place, rather than renaming, is what makes this safe for a
// file a live process still holds open: the lifecycle opens step logs with
// O_APPEND, so the writer continues at the new end of file. A writer that
// opened without O_APPEND (processes started before that change) keeps its old
// offset; the file becomes sparse, which still frees the disk blocks. Size is
// therefore measured in allocated blocks, so a sparse file is not trimmed
// again on every sweep.
func BoundLogs(root string, limit, keep int64) (LogBoundResult, error) {
	var result LogBoundResult
	if limit <= 0 {
		limit = DefaultLogFileLimit
	}
	if keep < 0 || keep >= limit {
		keep = DefaultLogTailKeep
	}
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result, nil
		}
		return result, err
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path != root {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, walkErr))
				return nil
			}
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 || entry.IsDir() || !isBoundedLogName(entry.Name()) {
			return nil
		}
		if result.Scanned >= maxLogBoundFiles {
			return filepath.SkipAll
		}
		result.Scanned++
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		allocated := allocatedBytes(info)
		if allocated <= limit {
			return nil
		}
		if err := trimLog(path, info.Size(), keep); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		result.Trimmed = append(result.Trimmed, path)
		result.ReclaimedBytes += allocated - keep
		return nil
	})
	return result, err
}

func isBoundedLogName(name string) bool {
	return strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".log.bak")
}

func trimLog(path string, size, keep int64) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	if keep > 0 {
		start := size - keep
		if start < 0 {
			start = 0
		}
		tailPath := path + ".1"
		tmp := tailPath + ".tmp"
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, io.NewSectionReader(file, start, size-start))
		closeErr := out.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(tmp)
			return errors.Join(copyErr, closeErr)
		}
		if err := os.Rename(tmp, tailPath); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	if err := file.Truncate(0); err != nil {
		return err
	}
	marker := fmt.Sprintf("[vrooli] log exceeded its size bound; the last %d bytes before this point are in %s.1\n", keep, filepath.Base(path))
	if keep == 0 {
		marker = "[vrooli] log exceeded its size bound; earlier output was discarded\n"
	}
	// Seek to the (new) end so the marker lands after anything an O_APPEND
	// writer added between the truncate and this write.
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	_, err = file.WriteString(marker)
	return err
}
