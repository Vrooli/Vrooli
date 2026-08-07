//go:build windows

package tpmcredentialaccess

import "os"

func fileInfoGroupID(os.FileInfo) (uint32, bool) { return 0, false }
