//go:build !linux && !darwin && !windows

package runtimeapp

import (
	"os"

	platform "github.com/vrooli/platform-go"
)

func openNativeSourcePointer(string) (*os.File, error) { return nil, platform.ErrUnsupported }
