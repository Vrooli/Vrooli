//go:build !windows

package securestore

import "os"

// RestrictCredentialFile applies the owner-only protection required for
// encrypted-store copies on POSIX systems. Encryption remains the protection
// boundary; the mode prevents accidental disclosure through ordinary reads.
func RestrictCredentialFile(path string) error { return os.Chmod(path, sealedFilePerm) }

func restrictCredentialDirectory(path string) error { return os.Chmod(path, sealedDirPerm) }
