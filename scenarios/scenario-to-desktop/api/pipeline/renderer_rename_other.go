//go:build !linux

package pipeline

import "fmt"

func renameDirectoryNoReplace(_, _ string) error {
	return fmt.Errorf("optional renderer pack staging is supported on Linux only")
}
