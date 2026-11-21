//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package capacity

import "fmt"

func diskFilesystem(string) (string, string, int64, error) {
	return "", "", 0, fmt.Errorf("disk admission is not qualified for this platform")
}
