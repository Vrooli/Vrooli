//go:build !unix

package modelstorage

import "os"

func allocatedBytes(fi os.FileInfo) int64 { return fi.Size() }

func inodeKey(os.FileInfo) ([2]uint64, bool) { return [2]uint64{}, false }
