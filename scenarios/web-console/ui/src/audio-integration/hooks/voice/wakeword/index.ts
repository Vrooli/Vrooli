// Barrel exports for the wake word detection module.

export type { AudioFeatures, MatchResult, WakeWordEngine, WakeWordTemplate } from "./types";
export {
  NUM_MFCC_COEFFICIENTS,
  FRAME_LENGTH_MS,
  FRAME_HOP_MS,
  MEL_FILTER_COUNT,
  MEL_LOW_HZ,
  MEL_HIGH_HZ,
  DTW_BAND_RATIO,
  DEFAULT_WAKE_WORD_THRESHOLD,
  MIN_ENROLLMENT_SAMPLES,
  MAX_ENROLLMENT_SAMPLES,
} from "./types";

export { extractMfcc } from "./mfcc";
export { bytesToFeatures, MFCC_SAMPLE_RATE } from "./extractFromBytes";
export { dtwDistance, distanceToScore } from "./dtw";
export { MfccDtwEngine, createWakeWordEngine, applyCms } from "./engine";
export { PassiveListener } from "./passiveListener";
export type { PassiveListenerOpts } from "./passiveListener";
export { useWakeWordTest } from "./useWakeWordTest";
export type { TestAttempt, TestStatus, WakeWordTestState, UseWakeWordTestOpts, UseWakeWordTestReturn } from "./useWakeWordTest";
