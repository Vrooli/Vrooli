package composition

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// ValidatedWAV bounds RIFF structure before publication, then checks the complete
// declared byte count while streaming. It accepts uncompressed PCM/float WAV;
// supported media is never inferred solely from HTTP Content-Type or 12 bytes.
func ValidatedWAV(input io.Reader) (io.Reader, error) {
	prefix := make([]byte, 12)
	if _, err := io.ReadFull(input, prefix); err != nil {
		return nil, fmt.Errorf("invalid WAV header: %w", err)
	}
	if string(prefix[:4]) != "RIFF" || string(prefix[8:]) != "WAVE" {
		return nil, fmt.Errorf("resource returned invalid WAV header")
	}
	total := int64(binary.LittleEndian.Uint32(prefix[4:])) + 8
	if total < 44 || total > 128<<20 {
		return nil, fmt.Errorf("unsupported WAV byte count")
	}
	blockAlign := 0
	for len(prefix) < 64*1024 {
		var header [8]byte
		if _, err := io.ReadFull(input, header[:]); err != nil {
			return nil, fmt.Errorf("truncated WAV chunks: %w", err)
		}
		prefix = append(prefix, header[:]...)
		size := int(binary.LittleEndian.Uint32(header[4:]))
		if int64(len(prefix))+int64(size) > total {
			return nil, fmt.Errorf("WAV chunk exceeds RIFF size")
		}
		if string(header[:4]) == "data" {
			if blockAlign == 0 || size == 0 || size%blockAlign != 0 {
				return nil, fmt.Errorf("WAV data lacks a valid format or complete frames")
			}
			return &exactMediaReader{r: io.MultiReader(bytes.NewReader(prefix), input), remaining: total}, nil
		}
		padded := size + (size % 2)
		if padded > 64*1024-len(prefix) {
			return nil, fmt.Errorf("WAV header exceeds 64 KiB")
		}
		data := make([]byte, padded)
		if _, err := io.ReadFull(input, data); err != nil {
			return nil, err
		}
		prefix = append(prefix, data...)
		if string(header[:4]) == "fmt " {
			if size < 16 {
				return nil, fmt.Errorf("WAV fmt chunk too short")
			}
			format := binary.LittleEndian.Uint16(data)
			channels := int(binary.LittleEndian.Uint16(data[2:]))
			rate := binary.LittleEndian.Uint32(data[4:])
			bits := int(binary.LittleEndian.Uint16(data[14:]))
			blockAlign = int(binary.LittleEndian.Uint16(data[12:]))
			if (format != 1 && format != 3) || channels < 1 || channels > 8 || rate < 8000 || rate > 192000 || bits < 8 || bits > 64 || bits%8 != 0 || blockAlign != channels*(bits/8) {
				return nil, fmt.Errorf("unsupported WAV PCM format")
			}
		}
	}
	return nil, fmt.Errorf("WAV data chunk unavailable")
}

type exactMediaReader struct {
	r         io.Reader
	remaining int64
}

func (r *exactMediaReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		var extra [1]byte
		n, err := r.r.Read(extra[:])
		if n > 0 {
			return 0, fmt.Errorf("WAV exceeds declared byte count")
		}
		return 0, err
	}
	p = p[:min(int64(len(p)), r.remaining)]
	n, err := r.r.Read(p)
	r.remaining -= int64(n)
	if err == io.EOF && r.remaining > 0 {
		return n, io.ErrUnexpectedEOF
	}
	return n, err
}
