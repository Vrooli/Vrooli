//go:build !linux && !darwin

package supervision

import (
	"context"
	"runtime"
)

func readNativeProcessTableContext(context.Context) (map[int]ProcessInfo, error) {
	return readNativeProcessTable()
}

func readNativeProcessTable() (map[int]ProcessInfo, error) {
	return nil, &UnsupportedProcessEvidenceError{Platform: runtime.GOOS}
}
