package platform

import (
	"fmt"
	"os"
)

// ReplaceProcessImage executes the already-open native executable. Replacing
// its pathname after verification cannot select different bytes. This initial
// backend accepts ELF binaries only; scripts require an interpreter contract.
// Unsupported hosts return ErrUnsupported without a pathname fallback.
func ReplaceProcessImage(file *os.File, argv, env []string) error {
	if file == nil || len(argv) == 0 {
		return fmt.Errorf("platform: executable image and argv are required")
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("platform: executable image must be regular")
	}
	return replaceProcessImage(file, argv, env)
}
