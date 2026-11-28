//go:build !linux

package hostwireless

import (
	"context"
	"errors"
)

func verifyKernelCMS(context.Context, []byte, []byte, []byte) error {
	return errors.New("kernel signature verification requires Linux")
}
