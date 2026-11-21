//go:build !linux

package hostwireless

import (
	"context"
	"errors"
)

func queryMOKCertificate(context.Context, []byte) (string, error) {
	return "", errors.New("MOK inspection requires Linux")
}
