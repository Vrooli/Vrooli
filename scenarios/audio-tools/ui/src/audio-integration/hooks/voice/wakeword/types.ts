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
   */
  compareBest(candidate: AudioFeatures, templates: AudioFeatures[], threshold: number): MatchResult;
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
}

// ---------------------------------------------------------------------------
// Tunable constants — grouped here for discoverability.
// ---------------------------------------------------------------------------

/** Number of MFCC coefficients per frame. */
export const NUM_MFCC_COEFFICIENTS = 13;
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
/** Default similarity threshold for wake word matching. */
export const DEFAULT_WAKE_WORD_THRESHOLD = 0.65;
/** Minimum number of enrollment samples required. */
export const MIN_ENROLLMENT_SAMPLES = 3;
/** Maximum number of enrollment samples. */
export const MAX_ENROLLMENT_SAMPLES = 5;
