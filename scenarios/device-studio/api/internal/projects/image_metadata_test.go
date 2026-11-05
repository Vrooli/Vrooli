package projects

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInspectReportsRecognizedUnsupportedColorMetadata(t *testing.T) {
	data := insertPNGChunk(grayPNG(t), "gAMA", []byte{0, 1, 0, 0})
	_, err := inspectSourceImage(data, DefaultSourceMediaLimits())
	var invalid ErrInvalidSourceMedia
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "unsupported color metadata", invalid.Reason)
}

func TestInspectRejectsValidNonIdentityEXIFOrientation(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	var encoded bytes.Buffer
	require.NoError(t, jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 80}))

	tiff := make([]byte, 8+2+12+4)
	copy(tiff[:2], "II")
	binary.LittleEndian.PutUint16(tiff[2:4], 42)
	binary.LittleEndian.PutUint32(tiff[4:8], 8)
	binary.LittleEndian.PutUint16(tiff[8:10], 1)
	entry := tiff[10:22]
	binary.LittleEndian.PutUint16(entry[:2], 0x0112)
	binary.LittleEndian.PutUint16(entry[2:4], 3)
	binary.LittleEndian.PutUint32(entry[4:8], 1)
	binary.LittleEndian.PutUint16(entry[8:10], 6)
	exif := append([]byte("Exif\x00\x00"), tiff...)
	segment := []byte{0xff, 0xe1, byte((len(exif) + 2) >> 8), byte(len(exif) + 2)}
	segment = append(segment, exif...)
	jpegWithOrientation := append([]byte{}, encoded.Bytes()[:2]...)
	jpegWithOrientation = append(jpegWithOrientation, segment...)
	jpegWithOrientation = append(jpegWithOrientation, encoded.Bytes()[2:]...)

	_, err := inspectSourceImage(jpegWithOrientation, DefaultSourceMediaLimits())
	var invalid ErrInvalidSourceMedia
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "non-identity EXIF orientation is not supported", invalid.Reason)
}

func insertPNGChunk(data []byte, name string, payload []byte) []byte {
	chunk := make([]byte, 4)
	binary.BigEndian.PutUint32(chunk, uint32(len(payload)))
	chunk = append(chunk, name...)
	chunk = append(chunk, payload...)
	crc := crc32.ChecksumIEEE(chunk[4:])
	sum := make([]byte, 4)
	binary.BigEndian.PutUint32(sum, crc)
	chunk = append(chunk, sum...)
	out := append([]byte{}, data[:33]...)
	out = append(out, chunk...)
	out = append(out, data[33:]...)
	return out
}
