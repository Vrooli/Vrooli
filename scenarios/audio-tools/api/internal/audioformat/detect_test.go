package audioformat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectDeclaredWins(t *testing.T) {
	// A declared codec is returned even when the bytes look like something
	// else — declare-first contract.
	got, err := Detect(CodecPCMS16LE, []byte("OggS....."))
	require.NoError(t, err)
	require.Equal(t, CodecPCMS16LE, got)
}

func TestDetectSniff(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		want Codec
	}{
		{"webm", []byte{0x1A, 0x45, 0xDF, 0xA3, 0x00}, CodecWebM},
		{"ogg", []byte("OggS\x00\x02"), CodecOGG},
		{"wav", append([]byte("RIFF\x00\x00\x00\x00"), []byte("WAVEfmt ")...), CodecWAV},
		{"flac", []byte("fLaC\x00\x00"), CodecFLAC},
		{"mp3-id3", []byte("ID3\x04\x00"), CodecMP3},
		{"mp3-sync", []byte{0xFF, 0xFB, 0x90, 0x00}, CodecMP3},
		{"aac-ftyp", append([]byte{0x00, 0x00, 0x00, 0x18}, []byte("ftypM4A ")...), CodecAAC},
		{"aac-adts", []byte{0xFF, 0xF1, 0x50, 0x80}, CodecAAC},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Detect(CodecUnknown, tc.head)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDetectUndeclaredUnknownErrors(t *testing.T) {
	_, err := Detect(CodecUnknown, []byte("not audio at all"))
	require.ErrorIs(t, err, ErrUnknownFormat)

	// Raw PCM is never sniffable — only recognized when declared.
	_, err = Detect(CodecUnknown, []byte{0x00, 0x01, 0x02, 0x03})
	require.ErrorIs(t, err, ErrUnknownFormat)
}
