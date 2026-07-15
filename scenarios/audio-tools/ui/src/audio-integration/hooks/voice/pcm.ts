// Compatibility import surface for existing Audio Tools tests and callers.
// Canonical implementation lives in packages/audio-capture-browser.
export {
  concatInt16,
  downsample,
  encodeWavFromPcm16,
  floatTo16BitPCM,
  frameToCanonicalPcm16,
  pcm16ToWavBuffer,
  TARGET_SAMPLE_RATE,
} from "@vrooli/audio-capture-browser";
