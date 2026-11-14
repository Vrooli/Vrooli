//go:build !linux && !darwin && !windows

package platform

import "os"

func processArguments(int) ([]string, error)      { return nil, ErrUnsupported }
func openProcessExecutable(int) (*os.File, error) { return nil, ErrUnsupported }
