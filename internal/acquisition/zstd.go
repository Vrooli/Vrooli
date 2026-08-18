// Package acquisition registers optional archive codecs for binaryfetch at
// the boundary that owns their dependency. Keeping this integration outside
// packages/binaryfetch preserves that package's standard-library-only core.
package acquisition

import (
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/vrooli/binaryfetch"
)

func init() {
	binaryfetch.RegisterArchiveDecompressor("tar.zst", func(reader io.Reader) (io.ReadCloser, error) {
		decoder, err := zstd.NewReader(reader)
		if err != nil {
			return nil, err
		}
		return decoder.IOReadCloser(), nil
	})
}
