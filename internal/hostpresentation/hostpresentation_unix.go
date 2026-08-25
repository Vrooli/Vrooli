//go:build !windows

package hostpresentation

import "os"

func effectiveUID() int { return os.Geteuid() }
