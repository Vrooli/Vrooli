//go:build !linux

package securestore

import (
	"context"
	"fmt"
	"io"
)

func UnlockLoginKeyring(context.Context, io.Reader) error {
	return fmt.Errorf("login keyring unlock is unsupported on this platform")
}
