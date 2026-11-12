//go:build !linux && !windows && !darwin

package platform

import "os"

func replaceProcessImage(*os.File, []string, []string) error { return ErrUnsupported }
