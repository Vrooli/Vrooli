// Package percepthash owns the perceptual-fingerprint primitives shared by the
// analysis duplicate detector (internal/analysis) and the diff comparison
// metrics (internal/diff).
//
// It exists because both domains independently implemented the same 64-bit DCT
// pHash — the same 32x32 grayscale → DCT → median-threshold algorithm, plus the
// same separable DCT and median helpers. Two copies of a numeric primitive drift
// silently: a fix or tuning change in one would leave the duplicate detector and
// the comparison surface disagreeing about what "similar" means. This package is
// the single home; both domains delegate.
//
// Everything here is pure Go and allocation-bounded (fixed 32x32 matrices), so it
// runs on the always-available CPU floor with no model, GPU, or download.
package percepthash

import (
	"image"
	"math"
	"math/bits"

	"github.com/disintegration/imaging"
)

// hashSize is the grayscale reduction before the DCT.
const hashSize = 32

// lowFreq is the side of the low-frequency block kept from the DCT.
const lowFreq = 8

// AHash is the classic average hash: reduce to 8x8 grayscale and set each bit
// where the pixel exceeds the mean. Cheaper and more exposure-sensitive than
// PHash; used as a secondary dedup key.
func AHash(img image.Image) uint64 {
	gray := imaging.Grayscale(imaging.Resize(img, 8, 8, imaging.Lanczos))
	b := gray.Bounds()
	var sum float64
	pix := make([]float64, 64)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			r, _, _, _ := gray.At(b.Min.X+x, b.Min.Y+y).RGBA()
			v := float64(r >> 8)
			pix[y*8+x] = v
			sum += v
		}
	}
	mean := sum / 64
	var hash uint64
	for i, v := range pix {
		if v > mean {
			hash |= 1 << uint(i)
		}
	}
	return hash
}

// PHash computes a 64-bit DCT perceptual hash: 32x32 grayscale → 2D DCT → the
// top-left 8x8 low-frequency block (minus the DC term) compared to its median.
// Robust to re-encoding, scaling, and minor edits.
func PHash(img image.Image) uint64 {
	gray := imaging.Grayscale(imaging.Resize(img, hashSize, hashSize, imaging.Lanczos))
	b := gray.Bounds()
	vals := make([][]float64, hashSize)
	for y := 0; y < hashSize; y++ {
		vals[y] = make([]float64, hashSize)
		for x := 0; x < hashSize; x++ {
			r, _, _, _ := gray.At(b.Min.X+x, b.Min.Y+y).RGBA()
			vals[y][x] = float64(r >> 8)
		}
	}

	dct := DCT2D(vals, hashSize)

	coeffs := make([]float64, 0, lowFreq*lowFreq-1)
	for y := 0; y < lowFreq; y++ {
		for x := 0; x < lowFreq; x++ {
			if x == 0 && y == 0 {
				continue // DC term carries mean brightness, not structure
			}
			coeffs = append(coeffs, dct[y][x])
		}
	}
	med := Median(coeffs)

	var hash uint64
	bit := 0
	for y := 0; y < lowFreq; y++ {
		for x := 0; x < lowFreq; x++ {
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

// DCT2D returns the separable 2D type-II DCT of an n×n matrix (rows then cols).
// n is small (32) so a direct transform is cheaper than a library dependency.
func DCT2D(m [][]float64, n int) [][]float64 {
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

// Median returns the median of xs without mutating the caller's slice. An empty
// slice yields 0. Insertion sort is fine for the 63-element pHash block.
func Median(xs []float64) float64 {
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

// Hamming counts the differing bits between two 64-bit hashes.
func Hamming(a, b uint64) int { return Popcount(a ^ b) }

// Popcount counts set bits.
func Popcount(x uint64) int { return bits.OnesCount64(x) }
