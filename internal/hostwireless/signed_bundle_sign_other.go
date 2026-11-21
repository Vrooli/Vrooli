//go:build !linux

package hostwireless

import (
	"context"
	"errors"
)

func SignPreparedBundle(context.Context, string, string, string, string) (SignedBundleReceipt, error) {
	return SignedBundleReceipt{}, errors.New("prepared module signing requires the qualified Linux profile")
}
