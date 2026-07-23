//go:build windows

package securestore

// Default deliberately remains unavailable until the Windows Credential
// Manager adapter has native-host evidence. cmdkey cannot safely read generic
// credential values, so using it here would be security theater.
func Default() Store {
	return Unavailable("Windows Credential Manager adapter has not received native-host evidence")
}
