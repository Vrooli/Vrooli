// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Barrel re-exports for the voice input module.

export type { TranscriptionProvider, VoiceBackend, VoiceState, VoiceInputState, StartRecordingOpts } from "./types";
export { WHISPER_FAILED_SENTINEL, AUDIO_BITRATE, STREAM_CHUNK_INTERVAL_MS, computeFinalTimeout } from "./types";
export { createAudioFilterChain } from "./audioUtils";
export { playRecordingStartCue, playRecordingStopCue } from "./audioCues";
export type { VadState, VadRefs } from "./vad";
export { VAD_CALIBRATION_MS, VAD_FALLBACK_SILENCE_TIMEOUT_MS, VAD_FALLBACK_SEGMENT_SILENCE_MS, VAD_NO_SPEECH_TIMEOUT_MS, VAD_MIN_SILENCE_THRESHOLD, VAD_MIN_SPEECH_THRESHOLD, VAD_SLIDING_WINDOW_SIZE, VAD_NOISE_FLOOR_DECAY_RATE, createVadRefs, computeSlidingNoiseFloor, vadTick } from "./vad";
export { WhisperProvider } from "./WhisperProvider";
export { PcmVoiceStreamProvider } from "./PcmVoiceStreamProvider";
