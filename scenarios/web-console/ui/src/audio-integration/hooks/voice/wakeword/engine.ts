// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Concrete WakeWordEngine implementation wiring MFCC extraction + DTW matching.
// This is the single point of change for swapping to a neural engine later.

import type { AudioFeatures, EngineCalibration, MatchResult, WakeWordEngine } from "./types";
import { SIGMA_FLOOR } from "./types";
import { extractMfcc } from "./mfcc";
import { calibratedScore, dtwDistance, uncalibratedScore } from "./dtw";
import { trimSilence } from "./trim";

/** Variance floor for CMVN — avoids divide-by-zero on a silent/constant coeff. */
const CMVN_STD_FLOOR = 1e-6;

/**
 * Apply Cepstral Mean and Variance Normalization (CMVN) to MFCC features.
 * Subtracts the per-coefficient mean AND divides by the per-coefficient standard
 * deviation across frames. The mean term removes channel/loudness offsets; the
 * variance term puts all 13 coefficients on a comparable scale so the DTW
 * Euclidean is no longer dominated by a few high-variance coefficients (notably
 * c0, the log-energy). This replaces the old mean-only CMS.
 */
export function normalizeFeatures(mfccs: number[][]): number[][] {
  const first = mfccs[0];
  if (!first) return mfccs;
  const numCoeffs = first.length;
  const numFrames = mfccs.length;

  // Per-coefficient mean.
  const means = new Array<number>(numCoeffs).fill(0);
  for (const frame of mfccs) {
    for (let c = 0; c < numCoeffs; c++) {
      means[c] = (means[c] ?? 0) + (frame[c] ?? 0);
    }
  }
  for (let c = 0; c < numCoeffs; c++) {
    means[c] = (means[c] ?? 0) / numFrames;
  }

  // Per-coefficient standard deviation (population), floored.
  const stds = new Array<number>(numCoeffs).fill(0);
  for (const frame of mfccs) {
    for (let c = 0; c < numCoeffs; c++) {
      const d = (frame[c] ?? 0) - (means[c] ?? 0);
      stds[c] = (stds[c] ?? 0) + d * d;
    }
  }
  for (let c = 0; c < numCoeffs; c++) {
    stds[c] = Math.max(Math.sqrt((stds[c] ?? 0) / numFrames), CMVN_STD_FLOOR);
  }

  return mfccs.map(frame => frame.map((v, c) => (v - (means[c] ?? 0)) / (stds[c] ?? 1)));
}

/**
 * MFCC + DTW wake word engine.
 *
 * Extracts 13-coefficient MFCCs from (silence-trimmed) audio and compares frame
 * sequences using Dynamic Time Warping with a Sakoe-Chiba band constraint.
 * CMVN normalization is applied at comparison time; c0 (log-energy) is excluded
 * from the distance; scores are mapped relative to the enrollment set's own
 * consistency via an EngineCalibration.
 */
export class MfccDtwEngine implements WakeWordEngine {
  extractFeatures(audio: Float32Array, sampleRate: number): AudioFeatures {
    // Trim endpoint silence uniformly here so every consumer (enrollment, test,
    // load, passive) gets identically-bounded features from one seam.
    const trimmed = trimSilence(audio, sampleRate);
    const mfccs = extractMfcc(trimmed, sampleRate);
    return {
      kind: "mfcc-v1",
      data: mfccs,
      sampleRate,
      durationSec: trimmed.length / sampleRate,
    };
  }

  compare(candidate: AudioFeatures, template: AudioFeatures, threshold: number): MatchResult {
    const candidateData = candidate.data as number[][];
    const templateData = template.data as number[][];

    if (candidateData.length === 0 || templateData.length === 0) {
      return { score: 0, isMatch: false };
    }

    const distance = dtwDistance(normalizeFeatures(candidateData), normalizeFeatures(templateData));
    // compare() has no enrollment-set context, so it uses the uncalibrated map.
    // Calibrated scoring is the job of compareBest (the live/test path).
    const score = uncalibratedScore(distance);
    return { score, isMatch: score >= threshold };
  }

  compareBest(
    candidate: AudioFeatures,
    templates: AudioFeatures[],
    threshold: number,
    calibration?: EngineCalibration | null,
  ): MatchResult {
    if (templates.length === 0) return { score: 0, isMatch: false };
    const candidateData = candidate.data as number[][];
    if (candidateData.length === 0) return { score: 0, isMatch: false };

    const normCandidate = normalizeFeatures(candidateData);

    // The best match is the template with the SMALLEST distance to the
    // candidate. Both score maps are monotonically decreasing in distance, so
    // picking min-distance is equivalent to picking max-score — and lets us map
    // exactly once, after the loop.
    let bestDistance = Infinity;
    for (const template of templates) {
      const templateData = template.data as number[][];
      if (templateData.length === 0) continue;
      const distance = dtwDistance(normCandidate, normalizeFeatures(templateData));
      if (distance < bestDistance) bestDistance = distance;
    }
    if (!isFinite(bestDistance)) return { score: 0, isMatch: false };

    const score = calibration
      ? calibratedScore(bestDistance, calibration)
      : uncalibratedScore(bestDistance);
    return { score, isMatch: score >= threshold };
  }

  calibrate(samples: AudioFeatures[]): EngineCalibration | null {
    if (samples.length < 2) return null;

    const normalized = samples
      .map(s => s.data as number[][])
      .filter(d => d.length > 0)
      .map(normalizeFeatures);
    if (normalized.length < 2) return null;

    // Intra-set pairwise DTW distances: how consistent are the user's own takes
    // with each other? This defines the (μ, σ) the live score is measured against.
    const distances: number[] = [];
    for (let i = 0; i < normalized.length; i++) {
      const a = normalized[i];
      if (!a) continue;
      for (let j = i + 1; j < normalized.length; j++) {
        const b = normalized[j];
        if (!b) continue;
        const d = dtwDistance(a, b);
        if (isFinite(d)) distances.push(d);
      }
    }
    if (distances.length === 0) return null;

    const mu = distances.reduce((acc, d) => acc + d, 0) / distances.length;
    const variance = distances.reduce((acc, d) => acc + (d - mu) * (d - mu), 0) / distances.length;
    const sigma = Math.max(Math.sqrt(variance), SIGMA_FLOOR);

    return { kind: samples[0]?.kind ?? "mfcc-v1", mu, sigma };
  }
}

/**
 * Factory function — the single point of change for swapping engine implementations.
 * All consumers should call this instead of constructing MfccDtwEngine directly.
 */
export function createWakeWordEngine(): WakeWordEngine {
  return new MfccDtwEngine();
}
