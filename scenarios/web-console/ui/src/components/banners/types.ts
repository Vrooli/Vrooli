/**
 * Banner model — re-exported from the library.
 *
 * The vocabulary, arbitration, damping, and presentation all live in
 * `react-component-library:Banner` now. What stays here is the part that is
 * genuinely this app's: the priority ladder below, which encodes web-console's
 * own judgement about what outranks what.
 *
 * Importing through this module rather than reaching for the package directly
 * keeps the version decision in one file — a later upgrade is edited here, not
 * across eleven descriptor call sites.
 */
export type {
  BannerAction,
  BannerDescriptor,
  BannerTone,
  MaybeBanner,
} from "@vrooli/react-component-library/Banner";

/**
 * Priority ladder. Bands, highest first:
 *
 *   90+  blocking and actionable — the app cannot do its job
 *   50…70 recoverable with data at risk — something of yours is retained
 *   20…45 transient progress — work is in flight and can be abandoned
 *   0…19  informational — nothing is wrong, nothing is at risk
 *
 * Keep new entries inside a band rather than inventing values between them;
 * the band is the design decision, the number is just an ordering.
 */
export const BANNER_PRIORITY = {
  connectionLost: 90,
  crashRecovery: 65,
  voiceRejection: 60,
  createError: 55,
  summarizeError: 50,
  voiceError: 45,
  voiceStaleMic: 42,
  sessionRecovery: 35,
  voiceFallback: 30,
  voiceTranscribing: 25,
  ttsSpeaking: 22,
  enableAudio: 20,
  // Informational: an optional side-feature is down. It sat at 70
  // — above `crashRecovery` — which said a degraded speech backend
  // outranked "sessions of yours survived a crash". Nothing of the
  // reader's is at risk here, and it is now raised only once they
  // have asked for audio, so it belongs at the top of the
  // informational band rather than in the at-risk one.
  audioUnavailable: 19,
  trackingDegraded: 10,
} as const;

export type BannerPriority = (typeof BANNER_PRIORITY)[keyof typeof BANNER_PRIORITY];
