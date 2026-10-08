//go:build !linux

package cleanup

import "io"

func writeChaosChunk(dst io.Writer, buf []byte, _ int64) (int, error) {
	return dst.Write(buf)
}
