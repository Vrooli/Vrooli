//go:build !windows

package platform

import (
	"context"
	"os"
)

func prepareOwnedImageChild(*os.File, []string, []string, string) (*OwnedImageChild, error) {
	return nil, ErrUnsupported
}

func verifyOwnedImageChild(context.Context, ServiceRef, OwnedImageChildIdentity, *os.File) error {
	return ErrUnsupported
}
