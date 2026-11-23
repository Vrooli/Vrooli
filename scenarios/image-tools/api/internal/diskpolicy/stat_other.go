//go:build !unix

package diskpolicy

import "errors"

func observe(string) (int64, int64, error) {
	return 0, 0, errors.New("diskpolicy: disk size is not observable on this platform")
}
