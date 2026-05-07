package pstoreramoops

import "os"

// defaultGetenv keeps the os import out of handler.go so the test seam
// (GetenvFn) is the only call path during unit tests.
func defaultGetenv(key string) string {
	return os.Getenv(key)
}
