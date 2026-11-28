package imageheader

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func pngHeader(width, height uint32) []byte {
	data := append([]byte{}, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}...)
	data = append(data, 0, 0, 0, 13) // IHDR length
	data = append(data, "IHDR"...)
	var dimensions [8]byte
	binary.BigEndian.PutUint32(dimensions[0:4], width)
	binary.BigEndian.PutUint32(dimensions[4:8], height)
	return append(data, dimensions[:]...)
}

func gifHeader(width, height uint16) []byte {
	data := []byte("GIF89a")
	var dimensions [4]byte
	binary.LittleEndian.PutUint16(dimensions[0:2], width)
	binary.LittleEndian.PutUint16(dimensions[2:4], height)
	return append(data, dimensions[:]...)
}

func TestDimensionsReadsHeaders(t *testing.T) {
	cases := map[string]struct {
		mime   string
		data   []byte
		width  int
		height int
	}{
		"png":    {mime: "image/png", data: pngHeader(640, 480), width: 640, height: 480},
		"gif":    {mime: "image/gif", data: gifHeader(320, 200), width: 320, height: 200},
		"params": {mime: "IMAGE/PNG; charset=binary", data: pngHeader(12, 34), width: 12, height: 34},
		"svg":    {mime: "image/svg+xml", data: []byte(`<svg width="10" height="10"></svg>`)},
		"unknown": {
			mime: "image/webp",
			data: []byte("RIFF\x00\x00\x00\x00WEBPnope"),
		},
		"truncated png": {mime: "image/png", data: []byte{0x89, 'P', 'N', 'G'}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			width, height := Dimensions(tc.mime, tc.data)
			require.Equal(t, tc.width, width)
			require.Equal(t, tc.height, height)
		})
	}
}

func TestNormalizeMimeDropsParametersAndCase(t *testing.T) {
	require.Equal(t, "image/png", NormalizeMime(" IMAGE/PNG; charset=binary "))
	require.Equal(t, "image/webp", NormalizeMime("image/webp"))
	require.Equal(t, "", NormalizeMime("  "))
}
