//go:build darwin

package target

// macOS processes launched with Setpgid share the process group with the
// Electron command. killElectronCommand already terminates that group, so no
// Linux-specific descendant traversal is required on Darwin.
func killElectronDescendants(rootPID int) error {
	return nil
}
