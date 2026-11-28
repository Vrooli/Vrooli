//go:build !unix

package files

import "os"

func changeTime(os.FileInfo) int64 { return 0 }
