//go:build !darwin && !windows

package platform

import "context"

func nativeSystemSupervisorPID(context.Context) (int, error) { return 0, ErrUnsupported }

func nativeSystemSupervisorStateRoot(context.Context) (string, error) { return "", ErrUnsupported }
