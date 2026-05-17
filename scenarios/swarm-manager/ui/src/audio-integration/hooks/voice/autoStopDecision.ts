// DOC: docs/internal/SEAMS.md#auto-stop-decision
//
// Pure decision boundary for the one-shot auto-stop trigger. Lifted out of
// useVoiceCore so the precedence between server- and client-side VAD is
// reviewable, testable, and shared across the three audio-integration copies.
//
// Precedence (strict, no averaging):
//   1. Fresh server VAD tick (receivedAt within SERVER_VAD_STALE_MS) that has
//      reached its configured silenceTimeoutMs → stop with source "server".
//   2. Else client VAD reported "stop" → stop with source "client-fallback".
//   3. Else continue recording.
//
// MUST stay byte-identical across audio-tools/ui, web-console/ui, and
// swarm-manager/ui (duplicate-before-extract). When this shape stabilises,
// extract once into a shared package.
//
// See plan: audio-tools-stt-accuracy-auto-stop-ssot.md §7 Phase 2.

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

  const tickIsFresh =
    serverVad.receivedAt > 0 &&
    nowPerf - serverVad.receivedAt <= staleTickMs;

  // When a fresh server tick is available with a usable timeout, the server
  // is the sole source of truth — it overrides any client "stop" so a
  // false-positive RMS dip doesn't cut a still-active utterance.
  if (tickIsFresh && serverVad.silenceTimeoutMs > 0) {
    if (serverVad.silenceElapsedMs >= serverVad.silenceTimeoutMs) {
      return { kind: "stop", source: "server" };
    }
    return { kind: "continue" };
  }

  if (clientVadResult === "stop") {
    return { kind: "stop", source: "client-fallback" };
  }

  return { kind: "continue" };
}
