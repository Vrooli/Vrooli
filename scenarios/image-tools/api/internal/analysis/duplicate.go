package analysis

import (
	"fmt"
	"image"
	"math"

	internalops "image-tools/internal/ops"

	"github.com/disintegration/imaging"
)

// DuplicateDetect computes perceptual fingerprints of an image so callers can
// find near-duplicates by comparing hashes (Hamming distance) across a set:
//   - pHash: a 64-bit DCT perceptual hash (robust to re-encode / scale / minor
//     edits) — the primary dedup key.
//   - aHash: a 64-bit average hash (cheaper, more exposure-sensitive).
//
// Pure-Go (no model, no GPU) — the always-runnable analysis floor.
func DuplicateDetect(src []byte) (DuplicateResult, error) {
	img, _, err := internalops.Decode(src)
	if err != nil {
		return DuplicateResult{}, fmt.Errorf("analysis: decode: %w", err)
	}
	return DuplicateResult{
		PhashHex: fmt.Sprintf("%016x", perceptualHash(img)),
		AhashHex: fmt.Sprintf("%016x", averageHash(img)),
		HashBits: 64,
	}, nil
}

// averageHash reduces the image to 8x8 grayscale and sets each bit where the
// pixel exceeds the mean — the classic aHash.
func averageHash(img image.Image) uint64 {
	gray := imaging.Grayscale(imaging.Resize(img, 8, 8, imaging.Lanczos))
	vals := make([]float64, 64)
	var sum float64
	i := 0
	b := gray.Bounds()
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			r, _, _, _ := gray.At(b.Min.X+x, b.Min.Y+y).RGBA()
			vals[i] = float64(r >> 8)
			sum += vals[i]
			i++
		}
	}
	mean := sum / 64
	var hash uint64
	for bit := 0; bit < 64; bit++ {
		if vals[bit] > mean {
			hash |= 1 << uint(bit)
		}
	}
	return hash
}

// perceptualHash computes a 64-bit DCT pHash: 32x32 grayscale → 2D DCT → the
// top-left 8x8 low-frequency block (minus the DC term) compared to its median.
func perceptualHash(img image.Image) uint64 {
	const n = 32
	const lo = 8
	gray := imaging.Grayscale(imaging.Resize(img, n, n, imaging.Lanczos))
	vals := make([][]float64, n)
	b := gray.Bounds()
	for y := 0; y < n; y++ {
		vals[y] = make([]float64, n)
		for x := 0; x < n; x++ {
			r, _, _, _ := gray.At(b.Min.X+x, b.Min.Y+y).RGBA()
			vals[y][x] = float64(r >> 8)
		}
	}
	dct := dct2D(vals, n)
	coeffs := make([]float64, 0, lo*lo-1)
	for y := 0; y < lo; y++ {
		for x := 0; x < lo; x++ {
			if x == 0 && y == 0 {
				continue
			}
			coeffs = append(coeffs, dct[y][x])
		}
	}
	med := medianOf(coeffs)
	var hash uint64
	bit := 0
	for y := 0; y < lo; y++ {
		for x := 0; x < lo; x++ {
			if x == 0 && y == 0 {
				continue
			}
			if dct[y][x] > med {
				hash |= 1 << uint(bit)
			}
			bit++
		}
	}
	return hash
}

// dct2D returns the separable 2D type-II DCT of an n×n matrix.
func dct2D(m [][]float64, n int) [][]float64 {
	cos := make([][]float64, n)
	for u := 0; u < n; u++ {
		cos[u] = make([]float64, n)
		for x := 0; x < n; x++ {
			cos[u][x] = math.Cos((2*float64(x) + 1) * float64(u) * math.Pi / (2 * float64(n)))
		}
	}
	rows := make([][]float64, n)
	for y := 0; y < n; y++ {
		rows[y] = make([]float64, n)
		for u := 0; u < n; u++ {
			var sum float64
			for x := 0; x < n; x++ {
				sum += m[y][x] * cos[u][x]
			}
			rows[y][u] = sum
		}
	}
	out := make([][]float64, n)
	for v := 0; v < n; v++ {
		out[v] = make([]float64, n)
		for u := 0; u < n; u++ {
			var sum float64
			for y := 0; y < n; y++ {
				sum += rows[y][u] * cos[v][y]
			}
			out[v][u] = sum
		}
	}
	return out
}

func medianOf(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j-1] > cp[j]; j-- {
			cp[j-1], cp[j] = cp[j], cp[j-1]
		}
	}
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}
