//go:build !windows

package hostinventory

import "os"

func effectiveUID() int { return os.Geteuid() }
