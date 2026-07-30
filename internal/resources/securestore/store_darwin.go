//go:build darwin

package securestore

// Default fails closed until the Keychain adapter uses the native Security
// framework. The security command accepts a password through -w, which puts a
// credential value in a process argument and therefore cannot implement the
// Vrooli credential authority safely.
func Default() Store {
	return Unavailable("macOS Keychain native adapter is not available; the security command value path is prohibited")
}
