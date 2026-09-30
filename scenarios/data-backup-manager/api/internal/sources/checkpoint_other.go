//go:build !linux

package sources

import (
	"fmt"
	"os"
)

func checkpointMetadata(string, os.FileInfo) error {
	return fmt.Errorf("workspace checkpoint posix-basic-v1 currently requires Linux; no portable fidelity claim is available")
}
