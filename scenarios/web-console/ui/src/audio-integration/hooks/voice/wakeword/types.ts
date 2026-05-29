// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Wake word detection abstraction layer. The WakeWordEngine interface is the
// seam that allows swapping MFCC+DTW for a neural embedding model later.
// All consumers program to this interface, not to the concrete implementation.

/**
 * Feature vector extracted from an audio segment. The `kind` discriminator
 * enables runtime validation when loading persisted templates — if a future
 * engine produces "embedding-v1" features, old MFCC templates can be detected
 * and the user prompted to re-record.
 */
export interface AudioFeatures {
  /** Discriminator for serialization/deserialization versioning. */
  kind: "mfcc-v1" | "embedding-v1";
  /**
   * For MFCC: 2D array of frames x coefficients (number[][]).
   * For embeddings: 1D vector (number[]).
   */
  data: number[][] | number[];
  /** Sample rate the features were extracted at. */
  sampleRate: number;
  /** Duration of the source audio in seconds. */
  durationSec: number;
}

/** Result of comparing two feature sets. */
export interface MatchResult {
  /** 0-1 similarity score (1 = perfect match). */
  score: number;
  /** Whether the score meets or exceeds the configured threshold. */
  isMatch: boolean;
}

/**
 * Derived score calibration for an enrollment set. Captures how consistent the
 * user's own enrollment recordings are with each other (mean μ and std σ of the
 * intra-set pairwise DTW distances), so a live utterance's distance can be
 * mapped to a meaningful 0–1 score relative to that natural variation.
 *
 * NEVER persisted — recomputed on load from the stored raw audio, exactly like
 * the MFCC features. See extractFromBytes.ts for the persistence contract.
 */
export interface EngineCalibration {
  /** Discriminator matching the features this calibration was derived from. */
  kind: AudioFeatures["kind"];
  /** Mean of the intra-enrollment-set pairwise DTW distances. */
  mu: number;
  /** Std of those distances, floored at {@link SIGMA_FLOOR}. */
  sigma: number;
}

/**
 * Strategy interface for wake word feature extraction and comparison.
 * MFCC+DTW is the initial implementation; neural embedding+cosine similarity
 * can be slotted in later by implementing this same interface.
 */
export interface WakeWordEngine {
  /** Extract features from a raw audio buffer (mono PCM Float32). */
  extractFeatures(audio: Float32Array, sampleRate: number): AudioFeatures;

  /**
   * Compare candidate features against a single stored template.
   * @param threshold - 0-1 similarity threshold for isMatch.
   */
  compare(candidate: AudioFeatures, template: AudioFeatures, threshold: number): MatchResult;

  /**
   * Compare candidate against multiple templates, returning the best match.
   * This handles natural speaker variation across enrollment recordings.
   * @param threshold - 0-1 similarity threshold for isMatch.
   * @param calibration - Optional enrollment-set calibration. When provided, the
   *   score is mapped relative to the user's own enrollment consistency; when
   *   absent the engine falls back to an uncalibrated logistic so it is still
   *   usable before calibration is available.
   */
  compareBest(
    candidate: AudioFeatures,
    templates: AudioFeatures[],
    threshold: number,
    calibration?: EngineCalibration | null,
  ): MatchResult;

  /**
   * Derive a score calibration from an enrollment set by measuring how
   * consistent the samples are with each other (mean μ and std σ of the
   * intra-set pairwise DTW distances). Returns null if fewer than 2 samples are
   * available (no pair to measure). Any future engine implementation must
   * provide this so calibrated scoring keeps working across an engine swap.
   */
  calibrate(samples: AudioFeatures[]): EngineCalibration | null;
}

/** Stored wake word configuration persisted on the backend. */
export interface WakeWordTemplate {
  /** Individual sample features (kept for multi-template matching). */
  samples: AudioFeatures[];
  /** User-chosen display label for this wake word (e.g., "Hey Vrooli"). */
  label: string;
  /** 0.1-0.95 similarity threshold. */
  threshold: number;
  /** ISO timestamp of last update. */
  updatedAt: string;
  /**
   * Derived score calibration from the enrollment set. In-memory only — NOT
   * persisted (recomputed on load from the stored raw audio). Optional so the
   * template is still valid before calibration runs (e.g. <2 samples).
   */
  calibration?: EngineCalibration | null;
}

// ---------------------------------------------------------------------------
// Tunable constants — grouped here for discoverability.
// ---------------------------------------------------------------------------

/** Number of MFCC coefficients per frame. */
export const NUM_MFCC_COEFFICIENTS = 13;
/**
 * First MFCC coefficient included in the DTW comparison Euclidean. c0 (index 0)
 * is the overall log-energy of the frame — it tracks loudness / mic distance /
 * gain, not phonetic content, and dominates the un-normalized distance. We keep
 * it in the extracted/persisted features (still 13-d) but exclude it at compare
 * time by starting the Euclidean at this index.
 */
export const FEATURE_START_COEFF = 1;
/** Frame length in milliseconds for MFCC windowing. */
export const FRAME_LENGTH_MS = 25;
/** Frame hop (stride) in milliseconds. */
export const FRAME_HOP_MS = 10;
/** Number of triangular mel filters in the filterbank. */
export const MEL_FILTER_COUNT = 26;
/** Lower frequency bound for the mel filterbank (Hz). */
export const MEL_LOW_HZ = 300;
/** Upper frequency bound for the mel filterbank (Hz). */
export const MEL_HIGH_HZ = 8000;
/** Sakoe-Chiba band width as a fraction of the longer sequence length. */
export const DTW_BAND_RATIO = 0.2;
/**
 * Default similarity threshold for wake word matching. Re-picked for the
 * calibrated score scale (see SCORE_* below): genuine same-speaker/same-phrase
 * utterances score ≈0.85–0.99 (distance near the enrollment mean), the 0.5
 * crossover sits at the enrollment mean + SCORE_MIDPOINT_Z·σ, and clearly
 * different words/noise sit many σ out (score ≲0.3). 0.7 accepts the user's own
 * natural variation while keeping clear separation from non-matches.
 */
export const DEFAULT_WAKE_WORD_THRESHOLD = 0.7;
/**
 * Floor for calibration σ. Prevents divide-by-zero and over-confident scoring
 * when enrollment samples are unusually consistent (σ→0 would make any tiny
 * distance deviation blow up the z-score).
 */
export const SIGMA_FLOOR = 0.05;
/**
 * Calibrated-score crossover, in σ above the enrollment mean: a live distance of
 * μ + SCORE_MIDPOINT_Z·σ maps to score 0.5. At distance == μ the score is ≈0.99.
 */
export const SCORE_MIDPOINT_Z = 3;
/** Calibrated-score logistic slope (per σ). Higher = sharper falloff. */
export const SCORE_SLOPE = 1.5;
/**
 * Fallback (uncalibrated) logistic, used only when no calibration is available
 * (<2 enrollment samples, or a pre-calibration comparison). Maps raw normalized
 * DTW distance → score directly. Rough seeds — calibration is the real path
 * since enrollment requires ≥ MIN_ENROLLMENT_SAMPLES samples.
 */
export const UNCALIBRATED_MIDPOINT_DISTANCE = 2.5;
/** Logistic slope for the uncalibrated fallback (per unit distance). */
export const UNCALIBRATED_SLOPE = 1.5;
/**
 * Microphone capture constraints shared by ALL wake-word audio paths: enrollment
 * (VoiceInputSection), the settings test (useWakeWordTest), and the passive
 * listener. Pinning identical constraints keeps the acoustic channel consistent
 * so enrollment-time and detection-time features are comparable. autoGainControl
 * was previously unspecified everywhere — fixing it on removes a loudness
 * variable the (now energy-normalized) score should not depend on, and the
 * enrollment/test paths previously requested none of these.
 */
export const WAKE_WORD_AUDIO_CONSTRAINTS: MediaTrackConstraints = {
  echoCancellation: true,
  noiseSuppression: true,
  autoGainControl: true,
};

/** Minimum number of enrollment samples required. */
export const MIN_ENROLLMENT_SAMPLES = 3;
/** Maximum number of enrollment samples. */
export const MAX_ENROLLMENT_SAMPLES = 5;
