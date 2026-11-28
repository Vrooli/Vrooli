package platform

import (
	"fmt"
	"os"
)

// ReplaceProcessImage hands off a verified native executable with exact argv
// and environment. Linux executes the open ELF descriptor. Darwin admits only
// a matching read-only Mach-O vnode under fully root-sealed ancestry; its
// privileged publishing owner must never mutate an admitted version in place.
// Callers own the verified loader/dependency closure. Scripts are refused;
// unsupported hosts return ErrUnsupported without an unowned launch fallback.
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
