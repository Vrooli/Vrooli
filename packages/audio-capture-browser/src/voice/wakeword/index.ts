// Barrel exports for the wake word detection module.

export type { AudioFeatures, EngineCalibration, MatchResult, WakeWordEngine, WakeWordTemplate } from "./types";
export {
  NUM_MFCC_COEFFICIENTS,
  FEATURE_START_COEFF,
  FRAME_LENGTH_MS,
  FRAME_HOP_MS,
  MEL_FILTER_COUNT,
  MEL_LOW_HZ,
  MEL_HIGH_HZ,
  DTW_BAND_RATIO,
  DEFAULT_WAKE_WORD_THRESHOLD,
  WAKE_WORD_AUDIO_CONSTRAINTS,
  SIGMA_FLOOR,
  SCORE_MIDPOINT_Z,
  SCORE_SLOPE,
  UNCALIBRATED_MIDPOINT_DISTANCE,
  UNCALIBRATED_SLOPE,
  MIN_ENROLLMENT_SAMPLES,
  MAX_ENROLLMENT_SAMPLES,
} from "./types";

export { extractMfcc } from "./mfcc";
export { bytesToFeatures, MFCC_SAMPLE_RATE } from "./extractFromBytes";
export { dtwDistance, calibratedScore, uncalibratedScore, distanceToScore } from "./dtw";
export { trimSilence } from "./trim";
export { MfccDtwEngine, createWakeWordEngine, normalizeFeatures, applyCms } from "./engine";
export { nextPow2, fft, hzToMel, melToHz, buildMelFilterbank, dctII } from "./mfcc";
export { PassiveListener } from "./passiveListener";
export type { PassiveListenerOpts, PassiveListenerSeams } from "./passiveListener";
export { useWakeWordTest } from "./useWakeWordTest";
export type { TestAttempt, TestStatus, WakeWordTestState, UseWakeWordTestOpts, UseWakeWordTestReturn } from "./useWakeWordTest";
