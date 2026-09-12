//go:build !linux && !darwin

package supervision

import "runtime"

func readNativeProcessTable() (map[int]ProcessInfo, error) {
	return nil, &UnsupportedProcessEvidenceError{Platform: runtime.GOOS}
}
