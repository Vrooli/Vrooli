//go:build !linux

package platform

import "os"

func replaceProcessImage(*os.File, []string, []string) error { return ErrUnsupported }
