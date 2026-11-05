//go:build !windows

package hostinventory

import "context"

func nativeWindowsSession(context.Context) ([]byte, error) { return nil, nil }
