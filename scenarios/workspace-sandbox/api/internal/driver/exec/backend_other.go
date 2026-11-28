//go:build !linux && !darwin

package exec

// platformContainmentBackend returns the containment backend for this OS.
// Linux ships bwrap (backend_linux.go) and macOS ships Seatbelt
// (backend_darwin.go); every other OS has no native containment backend, so
// this returns nil: ContainmentNone and ContainmentPreferred fall back to
// direct execution, while ContainmentRequired hard-errors.
func platformContainmentBackend() containmentBackend {
	return nil
}
