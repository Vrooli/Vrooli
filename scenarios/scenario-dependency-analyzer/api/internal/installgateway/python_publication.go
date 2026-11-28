package installgateway

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type pythonLockPublication struct{ source, output string }

// publishPythonLocks stages rollback files before changing either the ranged
// input or its compiled lock. A publication error restores earlier members.
// This is not a crash-atomic multi-file transaction; source/lock verification
// still detects an interrupted process.
func publishPythonLocks(pending []pythonLockPublication, rename func(string, string) error) error {
	backups := make([]string, len(pending))
	defer func() {
		for _, backup := range backups {
			if backup != "" {
				_ = os.Remove(backup)
			}
		}
	}()
	for i, item := range pending {
		info, err := os.Lstat(item.output)
		if err != nil {
			return fmt.Errorf("inspect Python dependency output: %w", err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Python dependency output must be a regular file: %s", item.output)
		}
		data, err := os.ReadFile(item.output)
		if err != nil {
			return err
		}
		backup, err := os.CreateTemp(filepath.Dir(item.source), ".previous-python-*")
		if err != nil {
			return err
		}
		backups[i] = backup.Name()
		_, writeErr := backup.Write(data)
		closeErr := backup.Close()
		if err := errors.Join(writeErr, closeErr, os.Chmod(backups[i], info.Mode().Perm()), os.Chmod(item.source, info.Mode().Perm())); err != nil {
			return err
		}
	}
	for i, item := range pending {
		if err := rename(item.source, item.output); err != nil {
			result := fmt.Errorf("publish Python dependency output %s: %w", item.output, err)
			for j := i - 1; j >= 0; j-- {
				if err := os.Rename(backups[j], pending[j].output); err != nil {
					result = errors.Join(result, fmt.Errorf("restore Python dependency output %s: %w", pending[j].output, err))
				}
			}
			return result
		}
	}
	return nil
}
