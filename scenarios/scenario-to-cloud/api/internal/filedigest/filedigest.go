// Package filedigest owns streaming file digest calculation for Scenario-to-Cloud.
package filedigest

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// SHA256File returns the hexadecimal SHA-256 and number of bytes read.
func SHA256File(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}
