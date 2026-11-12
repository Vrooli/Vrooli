//go:build darwin && !cgo

package platform

import "os"

func darwinWorkingDir(int) (string, error)                        { return "", ErrUnsupported }
func nativeProcessTCPListeners(int) ([]ProcessTCPListener, error) { return nil, ErrUnsupported }

func darwinOpenProcessExecutable(int) (*os.File, error) { return nil, ErrUnsupported }

func darwinObservationStamp(int) ([4]uint64, error) { return [4]uint64{}, ErrUnsupported }
