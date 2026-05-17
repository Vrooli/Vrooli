// Tiny external store carrying the latest server-emitted VAD-state snapshot.
//
// The audio-tools API emits StreamVadState ticks at ~20 Hz during silence
// (~2 Hz during voiced) over both the browser WebSocket and the Connect bidi
// transport. The mic-button ring reads this snapshot and renders a smooth
// fill that interpolates between ticks, so the ring visibly completes at the
// exact moment the server cuts a segment. When no tick has arrived in >250 ms
// the host wrapper falls back to the client-side VAD's auto-stop progress.
//
// Modeled on useVoiceConfigStore (same scenario folder) so the audio-
// integration directory stays portable. No external state-library dep; uses
// React's built-in useSyncExternalStore.
//
// MUST stay byte-identical across audio-tools/ui and swarm-manager/ui —
// duplicate-before-extract rule. Web-console reuses the same file under its
// audio-integration/ folder; host-specific wiring belongs in MicButton.

import { useSyncExternalStore } from "react";

export interface ServerVadStateSnapshot {
  /** True when the server last classified the frame as voiced. */
  voiced: boolean;
  /** Server-side elapsed silence in ms at the moment of the tick. */
  silenceElapsedMs: number;
  /** Server-side StreamConfig.VADSilenceMs echoed on every tick. */
  silenceTimeoutMs: number;
  /** performance.now() stamp on the client at arrival. 0 means "no tick yet". */
  receivedAt: number;
  /** Per-stream monotonic counter from the server. 0 means "no tick yet". */
  tickSeq: number;
}

const INITIAL_STATE: ServerVadStateSnapshot = {
  voiced: false,
  silenceElapsedMs: 0,
  silenceTimeoutMs: 0,
  receivedAt: 0,
  tickSeq: 0,
};

let state: ServerVadStateSnapshot = INITIAL_STATE;

const subscribers = new Set<() => void>();

function emit(): void {
  for (const fn of subscribers) fn();
}

function subscribe(fn: () => void): () => void {
  subscribers.add(fn);
  return () => {
    subscribers.delete(fn);
  };
}

function getSnapshot(): ServerVadStateSnapshot {
  return state;
}

/**
 * Apply a server tick. Stamps `receivedAt` with `performance.now()` so the
 * consumer can interpolate between ticks without a clock-sync concern.
 *
 * Drops out-of-order frames (proto bidi may interleave on some clients) and
 * applies a small monotonicity guard so a single late tick can't make the
 * ring jump backward by more than ~50 ms.
 */
export function setServerVadState(
  next: Pick<ServerVadStateSnapshot, "voiced" | "silenceElapsedMs" | "silenceTimeoutMs" | "tickSeq">,
): void {
  // tickSeq monotonicity — drop strictly older frames.
  if (state.tickSeq > 0 && next.tickSeq > 0 && next.tickSeq < state.tickSeq) {
    return;
  }
  // Voiced→voiced or silence→silence: prevent backward jumps >50 ms.
  if (
    state.receivedAt > 0 &&
    state.voiced === next.voiced &&
    !next.voiced &&
    next.silenceElapsedMs < state.silenceElapsedMs - 50
  ) {
    return;
  }
  state = {
    voiced: next.voiced,
    silenceElapsedMs: Math.max(0, next.silenceElapsedMs),
    silenceTimeoutMs: Math.max(0, next.silenceTimeoutMs),
    receivedAt:
      typeof performance !== "undefined" && typeof performance.now === "function"
        ? performance.now()
        : Date.now(),
    tickSeq: next.tickSeq,
  };
  emit();
}

/** Test-only: reset between specs. */
export function _resetServerVadStateForTesting(): void {
  state = INITIAL_STATE;
  emit();
}

/**
 * Hook returning the slice of server-VAD state the caller selects.
 * Zustand-shaped signature for parity with sibling stores.
 */
export function useServerVadStateStore<T>(selector: (s: ServerVadStateSnapshot) => T): T {
  return useSyncExternalStore(
    subscribe,
    () => selector(getSnapshot()),
    () => selector(getSnapshot()),
  );
}

/** Direct accessor for non-React callers (e.g. provider onVadState bridge). */
useServerVadStateStore.getState = (): ServerVadStateSnapshot & {
  set: typeof setServerVadState;
} => ({
  ...state,
  set: setServerVadState,
});
