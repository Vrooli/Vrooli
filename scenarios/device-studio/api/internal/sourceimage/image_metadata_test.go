package sourceimage

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInspectRejectsColorEncodingTheStillProfileCannotPreserve(t *testing.T) {
	for _, chunk := range []string{"gAMA", "cHRM", "cICP"} {
		metadata, err := Inspect(pngMetadataChunks(chunk, []byte{1, 2, 3, 4}), "png")
		require.ErrorIs(t, err, ErrUnsupportedImageMetadata)
		require.Empty(t, metadata)
	}
	metadata, err := Inspect(pngMetadataChunks("sRGB", []byte{0}), "png")
	require.NoError(t, err)
	require.Equal(t, Metadata{Orientation: 1, ProfileStatus: "srgb"}, metadata)
}

func TestJPEGMetadataScannerFindsProfileAfterEntropyScan(t *testing.T) {
	payload := append([]byte("ICC_PROFILE\x00"), 1, 1)
	data := []byte{0xff, 0xd8, 0xff, 0xda, 0, 2, 0x11, 0xff, 0xe2, 0, byte(len(payload) + 2)}
	data = append(data, payload...)
	data = append(data, 0xff, 0xd9)
	metadata, err := Inspect(data, "jpeg")
	require.NoError(t, err)
	require.True(t, metadata.HasProfile, "APP2 after a scan must not be silently ignored")
}

func TestParseTIFFOrientationRejectsDuplicateTagsEvenWhenEqual(t *testing.T) {
	tiff := make([]byte, 8+2+24)
	copy(tiff[:2], "II")
	binary.LittleEndian.PutUint16(tiff[2:4], 42)
	binary.LittleEndian.PutUint32(tiff[4:8], 8)
	binary.LittleEndian.PutUint16(tiff[8:10], 2)
	for i := 0; i < 2; i++ {
		entry := tiff[10+i*12 : 22+i*12]
		binary.LittleEndian.PutUint16(entry[:2], 0x0112)
		binary.LittleEndian.PutUint16(entry[2:4], 3)
		binary.LittleEndian.PutUint32(entry[4:8], 1)
		binary.LittleEndian.PutUint16(entry[8:10], 1)
	}
	_, _, err := parseTIFFOrientation(tiff)
	require.ErrorContains(t, err, "duplicate EXIF orientation")
}

func pngMetadataChunks(chunk string, payload []byte) []byte {
	data := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	appendChunk := func(name string, b []byte) {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(len(b)))
		data = append(data, buf...)
		data = append(data, name...)
		data = append(data, b...)
		data = append(data, 0, 0, 0, 0)
	}
	appendChunk(chunk, payload)
	appendChunk("IEND", nil)
	return data
}
