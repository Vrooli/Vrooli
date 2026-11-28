//go:build !windows && !darwin

package platform

import (
	"context"
	"io"
)

func prepareImmutableNativeDirectory(context.Context, string) (func() error, error) {
	return nil, ErrUnsupported
}
func writeImmutableNativeFile(context.Context, string, func(io.Writer) error) error {
	return ErrUnsupported
}

func grantImmutableNativeCodeReaders(context.Context, string, []string) error { return ErrUnsupported }
