// Package artifactcodecs registers archive formats at the application
// composition boundary. It deliberately imports neither runtime nor
// resources, so all consumers can opt into binaryfetch codecs without an
// import cycle.
package artifactcodecs

import (
	"io"

	"github.com/klauspost/compress/zstd"
	fastxz "github.com/mikelolasagasti/xz"
	"github.com/vrooli/binaryfetch"
)

const maxXZDictionaryBytes = 64 << 20

func init() {
	binaryfetch.RegisterArchiveDecompressor("tar.zst", func(reader io.Reader) (io.ReadCloser, error) {
		decoder, err := zstd.NewReader(reader)
		if err != nil {
			return nil, err
		}
		return decoder.IOReadCloser(), nil
	})
	binaryfetch.RegisterArchiveDecompressor("tar.xz", newXZDecoder)
}

func newXZDecoder(reader io.Reader) (io.ReadCloser, error) {
	return newXZDecoderWithLimit(reader, maxXZDictionaryBytes)
}

func newXZDecoderWithLimit(reader io.Reader, dictionaryLimit uint32) (io.ReadCloser, error) {
	decoder, err := fastxz.NewReader(reader, dictionaryLimit)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(decoder), nil
}
