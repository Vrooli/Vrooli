// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Dynamic Time Warping (DTW) with Sakoe-Chiba band constraint.
// Compares two MFCC frame sequences and returns a normalized distance.
//
// Performance: <10ms for two 100-frame sequences (~10K cells).

import {
  DTW_BAND_RATIO,
  FEATURE_START_COEFF,
  SCORE_MIDPOINT_Z,
  SCORE_SLOPE,
  UNCALIBRATED_MIDPOINT_DISTANCE,
  UNCALIBRATED_SLOPE,
  type EngineCalibration,
} from "./types";

/**
 * Euclidean distance between two equal-length vectors, over coefficients
 * [startCoeff, length). startCoeff defaults to FEATURE_START_COEFF so the c0
 * log-energy coefficient is excluded from the comparison (it tracks loudness,
 * not phonetic content); pass 0 to include all coefficients.
 */
function euclidean(a: number[], b: number[], startCoeff: number = FEATURE_START_COEFF): number {
  let sum = 0;
  for (let i = startCoeff; i < a.length; i++) {
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
 * Uses the textbook *symmetric* step pattern (Sakoe-Chiba P=0): a diagonal move
 * costs 2× the local distance while horizontal/vertical moves cost 1×, so the
 * total step weight along any monotonic path from (0,0) to (n-1,m-1) equals
 * (n + m). That is exactly the normalization divisor below — making the divisor
 * consistent with the step pattern (the previous unweighted pattern divided by
 * (n+m) anyway, which length-biased the result).
 *
 * @param bandRatio - Band width as fraction of the longer sequence (default: DTW_BAND_RATIO).
 * @param startCoeff - First MFCC coefficient included in the per-frame Euclidean
 *   (default FEATURE_START_COEFF, i.e. c0 excluded).
 * @returns Normalized DTW distance (total weighted cost / (n+m)). Lower = more similar.
 */
export function dtwDistance(
  seq1: number[][],
  seq2: number[][],
  bandRatio: number = DTW_BAND_RATIO,
  startCoeff: number = FEATURE_START_COEFF,
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

  // Initialize (0,0). The origin is reached by a diagonal step, so it carries
  // the diagonal weight of 2× under the symmetric pattern.
  prev[0] = 2 * euclidean(seq1Zero, seq2Zero, startCoeff);

  // Fill first row within band — only horizontal moves are available here, each
  // weighted 1×.
  for (let j = 1; j < m && j <= bandWidth; j++) {
    const seq2j = seq2[j];
    if (!seq2j) continue;
    prev[j] = (prev[j - 1] ?? INF) + euclidean(seq1Zero, seq2j, startCoeff);
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
      const cost = euclidean(seq1i, seq2j, startCoeff);
      // Symmetric step weights: vertical/horizontal add 1×cost, diagonal adds
      // 2×cost (it advances both indices, so it "earns" a double cost but skips
      // a cell). prev[j] is the vertical predecessor (i-1, j); curr[j-1] is the
      // horizontal predecessor (i, j-1); prev[j-1] is the true diagonal (i-1, j-1).
      let best = (prev[j] ?? INF) + cost; // vertical (i-1, j)
      if (j > 0) {
        best = Math.min(best, (curr[j - 1] ?? INF) + cost); // horizontal (i, j-1)
        best = Math.min(best, (prev[j - 1] ?? INF) + 2 * cost); // diagonal (i-1, j-1)
      }
      curr[j] = best;
    }

    [prev, curr] = [curr, prev];
  }

  // Normalize by total step weight (n + m), which the symmetric pattern above
  // makes exact regardless of how many diagonal vs. straight moves the path took.
  const rawDistance = prev[m - 1] ?? INF;
  return rawDistance / (n + m);
}

/**
 * Map a normalized DTW distance to a calibrated 0–1 similarity score, relative
 * to how consistent the user's own enrollment samples are with each other.
 *
 *   z = (distance - μ) / σ
 *   score = 1 / (1 + exp(SCORE_SLOPE * (z - SCORE_MIDPOINT_Z)))
 *
 * distance == μ (as close as the user's takes are to each other) → ≈0.99;
 * distance == μ + SCORE_MIDPOINT_Z·σ → 0.5; further out → low. Monotonically
 * decreasing in distance.
 */
export function calibratedScore(distance: number, calibration: EngineCalibration): number {
  if (!isFinite(distance)) return 0;
  const z = (distance - calibration.mu) / calibration.sigma;
  const score = 1 / (1 + Math.exp(SCORE_SLOPE * (z - SCORE_MIDPOINT_Z)));
  return Math.max(0, Math.min(1, score));
}

/**
 * Fallback score mapping when no enrollment calibration is available (<2 samples
 * or a pre-calibration comparison). Maps the raw normalized distance directly
 * through a logistic seeded by UNCALIBRATED_MIDPOINT_DISTANCE / UNCALIBRATED_SLOPE.
 */
export function uncalibratedScore(distance: number): number {
  if (!isFinite(distance)) return 0;
  if (distance === 0) return 1;
  const score = 1 / (1 + Math.exp(UNCALIBRATED_SLOPE * (distance - UNCALIBRATED_MIDPOINT_DISTANCE)));
  return Math.max(0, Math.min(1, score));
}

/** Legacy score helper retained for existing enrollment-test callers. */
export function distanceToScore(distance: number, scale: number = 1): number {
  if (!Number.isFinite(distance)) return 0;
  return 1 / (1 + distance * scale);
}
