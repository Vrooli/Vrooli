// DOC: docs/internal/VOICE-LATENCY.md#page-lifecycle-mic-cleanup-always-on-for-all-mic-owners
// DOC: docs/internal/SEAMS.md#voice-capture-lifecycle-controller-seam
//
// Microphone lifecycle policy — PURE decision helpers.
// =====================================================
//
// Two reviewable, side-effect-free decisions extracted so the iOS/PWA
// privacy policy and the registry-vs-UI honesty check are unit-testable
// without rendering React, mocking getUserMedia, or simulating page lifecycle.
// Mirrors voice/passiveArmDecision.ts and voice/autoStopDecision.ts.
//
//   1. `decideMicLifecycle` — given a page-lifecycle event and whether we are
//      running as an installed standalone/PWA, decide which leases the registry
//      backstop releases, whether the active recording is stopped for UI
//      honesty.
//   2. `selectStaleLeases` — given a metadata-only lease snapshot and the
//      current workflow state, decide which live leases are orphaned (a live
//      mic stream the UI is not honestly representing) and must self-heal.
//
// The impure `isStandaloneDisplayMode()` detector lives here too (it reads
// navigator/matchMedia) but is kept trivial so the decision it feeds stays the
// reviewable unit.

import { isActiveRecordingOwner, type MicOwner } from "./micOwnership";

/** Voice states the core can be in. Loose copy to avoid importing the full union. */
export type VoiceStateLite =
  | "idle"
  | "preparing"
  | "recording"
  | "listening"
  | "transcribing";

export type MicLifecycleEvent = "hidden" | "visible" | "pagehide" | "freeze";

/** Which leases the registry backstop releases for a lifecycle event. */
export type MicReleaseScope = "all" | "non-active" | "none";

export interface MicLifecycleDecision {
  /** Leases the registry backstop should release immediately (privacy). */
  release: MicReleaseScope;
  /**
   * Whether the capture controller should also stop an active user recording so
   * the workflow state returns to an honest stopped/idle/transcribing value.
   * The backstop releasing tracks alone does not update React state.
   */
  stopActiveRecording: boolean;
}

/**
 * Decide the lifecycle reaction for a page event.
 *
 * - `hidden`: standalone/PWA releases **all** leases (iOS keeps the OS mic
 *   indicator on otherwise, the failure mode this plan closes); a normal
 *   desktop tab releases only non-active leases and lets the controller stop
 *   the active recording so UI state stays consistent. Either way the active
 *   recording is stopped for honesty/privacy.
 * - `pagehide` / `freeze`: the page is going away — release **all** leases
 *   everywhere; best-effort stop of any active recording.
 * - `visible`: release nothing. Visibility alone is not mic intent, so it does
 *   not re-arm passive wake-word or low-latency prewarm.
 */
export function decideMicLifecycle(input: {
  event: MicLifecycleEvent;
  standalonePwa: boolean;
}): MicLifecycleDecision {
  switch (input.event) {
    case "hidden":
      return {
        release: input.standalonePwa ? "all" : "non-active",
        stopActiveRecording: true,
      };
    case "pagehide":
    case "freeze":
      return { release: "all", stopActiveRecording: true };
    case "visible":
      return { release: "none", stopActiveRecording: false };
  }
}

export interface StaleLeaseInput {
  /** Metadata-only lease snapshot (never the raw stream). */
  leases: Array<{ id: string; owner: MicOwner }>;
  /** Current workflow state machine value. */
  voiceState: VoiceStateLite;
  /** Whether low-latency voice (prewarm) is enabled. */
  lowLatencyVoice: boolean;
  /** Whether a passive wake-word listener is currently installed (ref truth). */
  passiveListenerActive: boolean;
}

/**
 * Select live leases the workflow should not be holding — the registry has a
 * live mic stream while the UI is not honestly representing it. These are the
 * "UI idle/off but the OS mic indicator is on" violations.
 *
 *   - Active-recording owners (voice-stream / whisper / web-speech) are stale
 *     whenever the workflow is `idle` (a recording/listening turn would keep
 *     the owner expected; `transcribing` legitimately holds the lease for the
 *     ~120 ms final-chunk settle window, so it is not flagged).
 *   - `low-latency-prewarm` is expected only while low-latency is enabled.
 *   - `passive-wake-word` is expected only while a passive listener is
 *     installed; a passive lease with no listener is a leak.
 *   - Settings-capture owners (enrollment / test / permission probe) are owned
 *     by the settings UI, transient, and never flagged here.
 *
 * The keys (`voiceState === "idle"`, listener-ref truth) are chosen so the
 * normal capture/handoff transitions never false-positive: during start the
 * state is `preparing`, during wake-word handoff the listener ref is already
 * set before the lease is acquired.
 */
export function selectStaleLeases(input: StaleLeaseInput): Array<{ id: string; owner: MicOwner }> {
  const { leases, voiceState, lowLatencyVoice, passiveListenerActive } = input;
  return leases.filter(({ owner }) => {
    if (isActiveRecordingOwner(owner)) return voiceState === "idle";
    if (owner === "low-latency-prewarm") return !lowLatencyVoice;
    if (owner === "passive-wake-word") return !passiveListenerActive;
    return false;
  });
}

/**
 * Whether the app is running as an installed standalone / PWA (iOS home-screen
 * or any display-mode: standalone install). Impure (reads navigator/matchMedia)
 * but trivial; the decision it feeds (`decideMicLifecycle`) is the pure unit.
 */
export function isStandaloneDisplayMode(): boolean {
  if (typeof window === "undefined") return false;
  // iOS Safari home-screen apps expose the non-standard navigator.standalone.
  const iosStandalone = (navigator as Navigator & { standalone?: boolean }).standalone === true;
  let displayModeStandalone = false;
  try {
    displayModeStandalone =
      typeof window.matchMedia === "function" &&
      window.matchMedia("(display-mode: standalone)").matches;
  } catch {
    displayModeStandalone = false;
  }
  return iosStandalone || displayModeStandalone;
}
