package renderjobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
)

// Only shared immutable inputs may avoid a redundant write. Attempt-specific
// intents, receipts and outputs keep their existing publication path.
func contentAddressedInputKey(key, digest string) bool {
	return strings.HasPrefix(key, workerBlobPrefix+digest+"/") ||
		strings.HasPrefix(key, inputBlobPrefix+digest+"/") ||
		key == closureBlobPrefix+digest+".json"
}

func (c *Coordinator) reuseVerifiedBlob(ctx context.Context, route routePin, key string, size int64, mime, digest string) (bool, error) {
	reader, storedMIME, err := c.blobs.Get(ctx, key)
	if err != nil {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		return false, c.requireRoute(ctx, route)
	}
	hash := sha256.New()
	n, readErr := io.Copy(hash, io.LimitReader(reader, size+1))
	closeErr := reader.Close()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return false, err
	}
	return readErr == nil && closeErr == nil && n == size && storedMIME == mime &&
		hex.EncodeToString(hash.Sum(nil)) == digest, nil
}
