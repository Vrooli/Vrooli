//go:build !unix

package models

import "errors"

func allocatedDirectoryBytes(string) (int64, error) {
	return 0, errors.New("allocated-block measurement is unsupported on this platform; refusing model export")
}
