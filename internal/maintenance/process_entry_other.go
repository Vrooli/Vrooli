//go:build !linux && !darwin

package maintenance

// readProcessEntry has no implementation on this platform; report not-found
// so the orphan kill guard fails safe (refuses to signal a PID it cannot
// re-validate).
func readProcessEntry(pid int) (processTableEntry, bool) {
	return processTableEntry{}, false
}
