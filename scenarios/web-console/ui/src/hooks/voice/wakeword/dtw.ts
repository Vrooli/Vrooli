// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Dynamic Time Warping (DTW) with Sakoe-Chiba band constraint.
// Compares two MFCC frame sequences and returns a normalized distance.
//
// Performance: <10ms for two 100-frame sequences (~10K cells).

import { DTW_BAND_RATIO } from "./types";

/**
 * Euclidean distance between two equal-length vectors.
 */
function euclidean(a: number[], b: number[]): number {
  let sum = 0;
  for (let i = 0; i < a.length; i++) {
    const d = (a[i] ?? 0) - (b[i] ?? 0);
    sum += d * d;
  }
  return Math.sqrt(sum);
}

/**
 * Compute DTW distance between two MFCC frame sequences.
 *
 * Uses the Sakoe-Chiba band constraint to limit the warping window,
 * bounding computation to O(N * W) where W = bandWidth.
 *
 * @param seq1 - First sequence of MFCC frames (number[][]).
 * @param seq2 - Second sequence of MFCC frames (number[][]).
 * @param bandRatio - Band width as fraction of the longer sequence (default: DTW_BAND_RATIO).
 * @returns Normalized DTW distance (total cost / path length). Lower = more similar.
 */
export function dtwDistance(
  seq1: number[][],
  seq2: number[][],
  bandRatio: number = DTW_BAND_RATIO,
): number {
  const n = seq1.length;
  const m = seq2.length;
  if (n === 0 || m === 0) return Infinity;

  // The band must be wide enough to account for both the ratio-based width
  // AND the length difference between sequences, otherwise the path cannot
  // reach (n-1, m-1) when sequences have different lengths.
  const lengthDiff = Math.abs(n - m);
  const bandWidth = Math.max(10, lengthDiff, Math.ceil(Math.max(n, m) * bandRatio));

  // Cost matrix — only allocate what the band requires.
  // Using two rows (current + previous) for O(m) space.
  const INF = Infinity;
  let prev = new Float64Array(m).fill(INF);
  let curr = new Float64Array(m).fill(INF);

  const seq1Zero = seq1[0];
  const seq2Zero = seq2[0];
  if (!seq1Zero || !seq2Zero) return Infinity;

  // Initialize (0,0)
  prev[0] = euclidean(seq1Zero, seq2Zero);

  // Fill first row within band
  for (let j = 1; j < m && j <= bandWidth; j++) {
    const seq2j = seq2[j];
    if (!seq2j) continue;
    prev[j] = (prev[j - 1] ?? INF) + euclidean(seq1Zero, seq2j);
  }

  // Fill remaining rows
  for (let i = 1; i < n; i++) {
    curr.fill(INF);
    const jMin = Math.max(0, i - bandWidth);
    const jMax = Math.min(m - 1, i + bandWidth);
    const seq1i = seq1[i];
    if (!seq1i) continue;

    for (let j = jMin; j <= jMax; j++) {
      const seq2j = seq2[j];
      if (!seq2j) continue;
      const cost = euclidean(seq1i, seq2j);
      let minPrev = prev[j] ?? INF; // match (diagonal)
      if (j > 0) {
        minPrev = Math.min(minPrev, curr[j - 1] ?? INF); // insertion
        minPrev = Math.min(minPrev, prev[j - 1] ?? INF);  // diagonal
      }
      curr[j] = cost + minPrev;
    }

    [prev, curr] = [curr, prev];
  }

  // Normalize by path length (approximated by n + m to avoid backtracking)
  const rawDistance = prev[m - 1] ?? INF;
  return rawDistance / (n + m);
}

/**
 * Convert a DTW distance to a 0-1 similarity score.
 * Uses a sigmoid-like mapping: score = 1 / (1 + distance * scale).
 * The scale factor controls sensitivity — higher scale makes the score
 * drop off faster with increasing distance.
 */
export function distanceToScore(distance: number, scale: number = 1): number {
  if (!isFinite(distance)) return 0;
  return 1 / (1 + distance * scale);
}
