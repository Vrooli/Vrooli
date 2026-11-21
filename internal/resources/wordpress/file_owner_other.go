//go:build !linux

package wordpress

import "os"

func directoryOwnerID(_ os.FileInfo) (int, int, error) { return 0, 0, nil }
