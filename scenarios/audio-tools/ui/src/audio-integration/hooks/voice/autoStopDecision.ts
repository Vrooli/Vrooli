// DOC: docs/internal/SEAMS.md#auto-stop-decision
//
// Pure decision boundary for the one-shot auto-stop trigger. Lifted out of
// useVoiceCore so the precedence between server- and client-side VAD is
// reviewable, testable, and shared across the three audio-integration copies.
//
// Precedence (strict):
//   1. If a server tick has EVER arrived this session (receivedAt > 0), the
//      server is the sole authority — client VAD never fires stop:
//        a. Fresh tick (within SERVER_VAD_STALE_MS) that has reached its
//           configured silenceTimeoutMs → stop with source "server".
//        b. Otherwise → continue. A briefly stale tick (> staleTickMs) does
//           NOT hand authority back to the client; if it did, the client's
//           RMS VAD — which is more aggressive than the server's PCM-frame
//           detector — would fire false-positive stops mid-utterance.
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

/**
 * Pure function — no I/O, no timers, no module-level state. Caller is
 * responsible for sampling nowPerf and reading the server VAD snapshot.
 */
export function decideAutoStop(input: AutoStopInputs): AutoStopVerdict {
  const { serverVad, clientVadResult, nowPerf, staleTickMs } = input;

  // Once the server has ever spoken this session, it owns the stop decision.
  // Briefly stale ticks do NOT re-enable client fallback — that's the source
  // of the mid-utterance false positives we're trying to eliminate.
  if (serverVad.receivedAt > 0) {
    const tickIsFresh = nowPerf - serverVad.receivedAt <= staleTickMs;
    if (
      tickIsFresh &&
      serverVad.silenceTimeoutMs > 0 &&
      serverVad.silenceElapsedMs >= serverVad.silenceTimeoutMs
    ) {
      return { kind: "stop", source: "server" };
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
