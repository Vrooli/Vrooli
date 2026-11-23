//go:build !unix

package modelstray

// isCrossDevice cannot be detected portably here; the rename error is returned
// as is, and nothing is moved.
func isCrossDevice(error) bool { return false }
