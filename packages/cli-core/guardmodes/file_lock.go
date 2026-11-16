package guardmodes

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const staleLockAfter = 10 * time.Minute

func acquireFileLock(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			if stale, staleErr := staleLock(path); staleErr == nil && stale {
				if removeErr := os.Remove(path); removeErr == nil {
					return acquireFileLock(path)
				}
			}
			return nil, ErrStoreBusy
		}
		return nil, fmt.Errorf("acquire guard mode lock: %w", err)
	}
	if _, err := file.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("write guard mode lock: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return func() { _ = os.Remove(path) }, nil
}

func staleLock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	nanos, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return false, err
	}
	return time.Since(time.Unix(0, nanos)) > staleLockAfter, nil
}
