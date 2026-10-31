package storageactivity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// FailureDetail preserves the failing process and read stage without returning
// raw filesystem paths, environment values, or arbitrary error strings.
func FailureDetail(err error) string {
	if errors.Is(err, context.Canceled) {
		return "process observation canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "process observation timed out"
	}
	var pathError *os.PathError
	if errors.As(err, &pathError) {
		parts := splitProcPath(pathError.Path)
		if len(parts) >= 2 {
			if _, parseErr := strconv.ParseUint(parts[0], 10, 32); parseErr == nil {
				stage := parts[1]
				switch stage {
				case "fd", "cwd", "exe", "environ", "maps":
					cause := "read failed"
					if errors.Is(pathError, os.ErrPermission) {
						cause = "permission denied"
					}
					return fmt.Sprintf("process %s %s: %s", parts[0], stage, cause)
				}
			}
		}
	}
	return "process observation incomplete"
}

func splitProcPath(path string) []string {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return nil
	}
	var parts []string
	for path != string(filepath.Separator) {
		parts = append([]string{filepath.Base(path)}, parts...)
		path = filepath.Dir(path)
	}
	if len(parts) == 0 || parts[0] != "proc" {
		return nil
	}
	return parts[1:]
}
