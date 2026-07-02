// DOC: docs/internal/SEAMS.md#auto-stop-decision
//
// Pure decision boundary for the one-shot auto-stop trigger. Lifted out of
// useVoiceCore so the precedence between server- and client-side VAD is
// reviewable, testable, and shared across the three audio-integration copies.
//
// Precedence (strict):
//   1. If a server tick has EVER arrived this session (receivedAt > 0), the
//      server is normally the authority — but only while its signal is fresh:
//        a. The server latched silence_timed_out (silenceTimedOut, sticky in
//           the store) → stop with source "server", REGARDLESS of freshness.
//           The server emits the threshold tick exactly once (the frame it
//           cuts the segment) then goes quiet, so a crossed threshold must
//           NOT be un-crossed by the tick going stale — that was the wedge.
//        b. Fresh tick (within SERVER_VAD_STALE_MS) that has reached its
//           configured silenceTimeoutMs → stop with source "server". (Backstop
//           for transports that drop the flag; same outcome.)
//        c. Fresh tick below timeout, OR stale tick without the latch but the
//           client has NOT reported stop → continue. The server is still in
//           charge; we're just waiting on the next tick or the latch.
//        d. STALE tick (>staleTickMs) AND latch is false AND clientVad="stop"
//           → stop with source "client-fallback". Belt-and-suspenders for the
//           dead zone: if the server's threshold tick was lost in transit (or
//           never fired because of a server-side bug), the client RMS VAD's
//           independent stop verdict breaks the wedge. The clientVad="stop"
//           gate prevents the old "stale tick lets aggressive client VAD fire
//           false positives mid-utterance" regression — both signals must
//           agree before we hand authority back to the client.
//   2. No server tick has ever arrived (server VAD not running for this
//      session — passthrough strategy, transport error, etc.):
//        a. Client VAD reported "stop" → stop with source "client-fallback".
//        b. Otherwise → continue.
//
// The "any tick" check is the key behaviour difference vs the prior version:
// before 2026-05-17 v2, transient WS jitter that stretched a tick gap past
// 250 ms would let the client fire, causing the user-visible "stops while
// I'm still talking" bug.
//
// MUST stay byte-identical across audio-tools/ui, web-console/ui, and
// swarm-manager/ui (duplicate-before-extract). When this shape stabilises,
// extract once into a shared package.

import type { ServerVadStateSnapshot } from "../useServerVadStateStore";
import type { VoiceActivitySnapshot } from "./types";
import type { VadAction } from "./vad";

export interface AutoStopInputs {
  serverVad: ServerVadStateSnapshot;
  /** Latest result from vadTick — null when the client VAD did not act. */
  clientVadResult: VadAction | null;
  /** performance.now() (or Date.now() fallback) captured by the caller. */
  nowPerf: number;
  /** Staleness threshold for a server tick (ms). Pass SERVER_VAD_STALE_MS. */
  staleTickMs: number;
}

export type AutoStopVerdict =
  | { kind: "continue" }
  | { kind: "stop"; source: "server" | "client-fallback" };

export interface AutoStopRingInputs {
  isRecording: boolean;
  serverVad: ServerVadStateSnapshot | null | undefined;
  voiceActivity: VoiceActivitySnapshot | null | undefined;
  nowPerf: number;
  staleTickMs: number;
  visualGraceMs: number;
}

export interface AutoStopRingState {
  visible: boolean;
  progress: number;
}

const HIDDEN_RING: AutoStopRingState = { visible: false, progress: 0 };

/**
 * Pure function — no I/O, no timers, no module-level state. Caller is
 * responsible for sampling nowPerf and reading the server VAD snapshot.
 */
export function decideAutoStop(input: AutoStopInputs): AutoStopVerdict {
  const { serverVad, clientVadResult, nowPerf, staleTickMs } = input;

  // Once the server has ever spoken this session, it owns the stop decision
  // while fresh. A stale tick + matching client-stop verdict is the only
  // escape hatch (see header §1d).
  if (serverVad.receivedAt > 0) {
    // Latched timeout: the server told us the silence threshold was reached.
    // This is sticky in the store, so it survives the freshness window — the
    // crossed threshold is terminal and must not be un-crossed by a stale
    // tick. Freshness only gates the BELOW-timeout case below.
    if (serverVad.silenceTimedOut) {
      return { kind: "stop", source: "server" };
    }
    const tickIsFresh = nowPerf - serverVad.receivedAt <= staleTickMs;
    if (
      tickIsFresh &&
      serverVad.silenceTimeoutMs > 0 &&
      serverVad.silenceElapsedMs >= serverVad.silenceTimeoutMs
    ) {
      return { kind: "stop", source: "server" };
    }
    // Belt-and-suspenders: if the server's threshold tick was lost (network
    // hiccup, server-side gating bug, etc.) and the tick has gone stale
    // without the latch firing, and the client RMS VAD has independently
    // decided "stop", honour it. Both signals must agree — this prevents the
    // client's more aggressive VAD from firing false positives mid-utterance
    // while a fresh server tick is keeping the session alive.
    if (!tickIsFresh && clientVadResult === "stop") {
      return { kind: "stop", source: "client-fallback" };
    }
    return { kind: "continue" };
  }

  // No server tick has ever arrived — server VAD isn't producing signal for
  // this session. Fall back to the client RMS VAD's verdict.
  if (clientVadResult === "stop") {
    return { kind: "stop", source: "client-fallback" };
  }

  return { kind: "continue" };
}

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value));
}

/**
 * Derive the mic-button auto-stop ring from the same server/client authority
 * rules as decideAutoStop(). The important shared rule is that a latched server
 * timeout remains terminal even after the final tick goes stale.
 */
export function decideAutoStopRing(input: AutoStopRingInputs): AutoStopRingState {
  if (!input.isRecording) return HIDDEN_RING;

  const { serverVad, voiceActivity, nowPerf, staleTickMs, visualGraceMs } = input;
  if (serverVad && serverVad.receivedAt > 0) {
    if (serverVad.silenceTimedOut) {
      return serverVad.silenceTimeoutMs > 0
        ? { visible: true, progress: 1 }
        : HIDDEN_RING;
    }

    const serverAge = nowPerf - serverVad.receivedAt;
    if (serverAge < staleTickMs && serverVad.silenceTimeoutMs > 0) {
      const interpolated = Math.min(
        serverVad.silenceElapsedMs + serverAge,
        serverVad.silenceTimeoutMs,
      );
      return {
        visible: !serverVad.voiced && interpolated >= visualGraceMs,
        progress: clamp01(interpolated / serverVad.silenceTimeoutMs),
      };
    }
  }

  const progress = clamp01(voiceActivity?.autoStopProgress ?? 0);
  const visible = voiceActivity?.phase === "silence"
    && voiceActivity.autoStopVisible
    && voiceActivity.silenceTimeoutMs > 0;
  return visible ? { visible: true, progress } : HIDDEN_RING;
}
