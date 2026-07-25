import type { AudioStrategy } from './capability';

/** Chromium keeps the muted AudioContext API behind this renderer feature. */
const AUDIO_CONTEXT_SINK_ID_FEATURES = [
  '--enable-blink-features=AudioContextSetSinkId',
  '--enable-features=AudioContextSetSinkId',
];

/** Launch-only audio policy; pooled browsers must never cross this boundary. */
export function getAudioLaunchArgs(strategy: AudioStrategy): string[] {
  return strategy === 'synthetic_sink' ? AUDIO_CONTEXT_SINK_ID_FEATURES : [];
}
