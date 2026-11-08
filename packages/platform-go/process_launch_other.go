//go:build !linux && !darwin

package platform

import "os"

func processArguments(int) ([]string, error)      { return nil, ErrUnsupported }
func openProcessExecutable(int) (*os.File, error) { return nil, ErrUnsupported }
