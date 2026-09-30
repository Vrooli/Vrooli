// Package artifactcodecs registers archive formats at the application
// composition boundary. It deliberately imports neither runtime nor
// resources, so all consumers can opt into binaryfetch codecs without an
// import cycle.
package artifactcodecs

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
