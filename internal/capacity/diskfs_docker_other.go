//go:build !linux

package capacity

import (
	"context"
	"fmt"
)

func resolveDockerDiskPath(context.Context, DiskJobPath) (string, string, int64, string, error) {
	return "", "", 0, "", fmt.Errorf("Docker-managed inaccessible disk paths require Linux host mount identity support")
}

func resolveDockerRootDir(context.Context, DiskJobPath) (string, string, int64, string, error) {
	return "", "", 0, "", fmt.Errorf("Docker-managed inaccessible disk paths require Linux host mount identity support")
}
