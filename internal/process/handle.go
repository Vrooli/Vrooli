package process

// Handle identifies one process incarnation, independent of later PID reuse.
// Callers must open it before inspecting identity and check Alive after that
// inspection, before signaling. Methods must not be called concurrently.
type Handle interface {
	Alive() (bool, error)
	Signal(force bool) error
	Close() error
}

// OpenHandle fails closed where stable process handles are unavailable. It
// never falls back to signaling a numeric PID.
func OpenHandle(pid int) (Handle, error) { return openHandle(pid) }
