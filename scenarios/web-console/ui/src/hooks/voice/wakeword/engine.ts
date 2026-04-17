// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Concrete WakeWordEngine implementation wiring MFCC extraction + DTW matching.
// This is the single point of change for swapping to a neural engine later.

import type { AudioFeatures, MatchResult, WakeWordEngine } from "./types";
import { extractMfcc } from "./mfcc";
import { distanceToScore, dtwDistance } from "./dtw";

/**
 * Apply Cepstral Mean Subtraction (CMS) to normalize MFCC features.
 * Subtracts the per-coefficient mean across all frames, removing channel
 * effects and normalizing magnitude for more consistent DTW distances.
 */
export function applyCms(mfccs: number[][]): number[][] {
  const first = mfccs[0];
  if (!first) return mfccs;
  const numCoeffs = first.length;
  const numFrames = mfccs.length;

  // Compute mean per coefficient
  const means = new Array<number>(numCoeffs).fill(0);
  for (const frame of mfccs) {
    for (let c = 0; c < numCoeffs; c++) {
      means[c] = (means[c] ?? 0) + (frame[c] ?? 0);
    }
  }
  for (let c = 0; c < numCoeffs; c++) {
    means[c] = (means[c] ?? 0) / numFrames;
  }

  // Subtract mean
  return mfccs.map(frame => frame.map((v, c) => v - (means[c] ?? 0)));
}

/**
 * MFCC + DTW wake word engine.
 *
 * Extracts 13-coefficient MFCCs from audio and compares frame sequences
 * using Dynamic Time Warping with a Sakoe-Chiba band constraint.
 * CMS normalization is applied at comparison time for consistent scoring.
 */
export class MfccDtwEngine implements WakeWordEngine {
  extractFeatures(audio: Float32Array, sampleRate: number): AudioFeatures {
    const mfccs = extractMfcc(audio, sampleRate);
    return {
      kind: "mfcc-v1",
      data: mfccs,
      sampleRate,
      durationSec: audio.length / sampleRate,
    };
  }

  compare(candidate: AudioFeatures, template: AudioFeatures, threshold: number): MatchResult {
    const candidateData = candidate.data as number[][];
    const templateData = template.data as number[][];

    if (candidateData.length === 0 || templateData.length === 0) {
      return { score: 0, isMatch: false };
    }

    // Apply CMS normalization before comparison for consistent distances
    const normCandidate = applyCms(candidateData);
    const normTemplate = applyCms(templateData);

    const distance = dtwDistance(normCandidate, normTemplate);
    const score = distanceToScore(distance);
    return { score, isMatch: score >= threshold };
  }

  compareBest(candidate: AudioFeatures, templates: AudioFeatures[], threshold: number): MatchResult {
    if (templates.length === 0) return { score: 0, isMatch: false };

    let bestScore = 0;
    for (const template of templates) {
      const result = this.compare(candidate, template, threshold);
      if (result.score > bestScore) {
        bestScore = result.score;
      }
    }

    return { score: bestScore, isMatch: bestScore >= threshold };
  }
}

/**
 * Factory function — the single point of change for swapping engine implementations.
 * All consumers should call this instead of constructing MfccDtwEngine directly.
 */
export function createWakeWordEngine(): WakeWordEngine {
  return new MfccDtwEngine();
}
