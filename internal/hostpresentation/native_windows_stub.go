//go:build !windows

package hostpresentation

import "context"

func nativeWindowsSession(context.Context) ([]byte, error) { return nil, nil }
