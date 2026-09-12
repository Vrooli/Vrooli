import type { AudioStrategy } from './capability';

/**
 * Version insurance, not a requirement. Chromium gates the `sinkId` *attribute*
 * and `setSinkId()` separately from the `AudioContextOptions.sinkId` constructor
 * member. On the build measured here the attribute is absent
 * (`'sinkId' in AudioContext.prototype === false`) yet the constructor member is
 * still honoured, so the silent-sink patch works with or without these switches.
 * They are kept because builds that gate the constructor member behind the
 * feature would otherwise fail silently: an unknown dictionary member is ignored
 * by WebIDL rather than throwing, so there is nothing to catch.
 *
 * Do not add another `--enable-features` entry elsewhere in the launch args.
 * Chromium honours only the first occurrence of a repeated switch, so a second
 * one would silently drop this list.
 */
const AUDIO_CONTEXT_SINK_ID_FEATURES = [
  '--enable-blink-features=AudioContextSetSinkId',
  '--enable-features=AudioContextSetSinkId',
];

/** Launch-only audio policy; pooled browsers must never cross this boundary. */
export function getAudioLaunchArgs(strategy: AudioStrategy): string[] {
  return strategy === 'synthetic_sink' ? AUDIO_CONTEXT_SINK_ID_FEATURES : [];
}
