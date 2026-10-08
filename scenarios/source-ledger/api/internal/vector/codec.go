// Package vector owns the compact embedding representation used by the
// journal and forest caches.
package vector

import (
	"encoding/binary"
	"fmt"
	"math"
)

func Encode(values []float64) []byte {
	if len(values) == 0 {
		return []byte{}
	}
	out := make([]byte, 4+4*len(values))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(values)))
	for i, value := range values {
		binary.LittleEndian.PutUint32(out[4+i*4:], math.Float32bits(float32(value)))
	}
	return out
}

func Decode(raw []byte) ([]float64, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) < 4 {
		return nil, fmt.Errorf("vector blob is truncated")
	}
	n := int(binary.LittleEndian.Uint32(raw[:4]))
	if len(raw) != 4+4*n {
		return nil, fmt.Errorf("vector blob length %d does not match dimension %d", len(raw), n)
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[4+i*4:])))
	}
	return out, nil
}
