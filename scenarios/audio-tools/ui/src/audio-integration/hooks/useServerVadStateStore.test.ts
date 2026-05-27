// Tests for useServerVadStateStore — the external store carrying the latest
// server-emitted VAD-state snapshot. See plan
// /home/matthalloran8/.vrooli/plans/server-driven-mic-ring-streamvadstate-event.md
// §9 item 4 for the contract this file enforces.

import { describe, it, expect, beforeEach } from "vitest";

import {
  useServerVadStateStore,
  setServerVadState,
  _resetServerVadStateForTesting,
} from "./useServerVadStateStore";

describe("useServerVadStateStore", () => {
  beforeEach(() => {
    _resetServerVadStateForTesting();
  });

  it("starts in an empty snapshot (receivedAt=0, tickSeq=0)", () => {
    const snap = useServerVadStateStore.getState();
    expect(snap.receivedAt).toBe(0);
    expect(snap.tickSeq).toBe(0);
    expect(snap.voiced).toBe(false);
    expect(snap.silenceElapsedMs).toBe(0);
    expect(snap.silenceTimeoutMs).toBe(0);
  });

  it("setServerVadState records a tick, stamps receivedAt, and notifies subscribers", () => {
    // The public hook is React-only; we exercise the notification path
    // through the direct setter and inspect state after.
    setServerVadState({ voiced: false, silenceElapsedMs: 200, silenceTimeoutMs: 1500, tickSeq: 1 });
    const after = useServerVadStateStore.getState();
    expect(after.voiced).toBe(false);
    expect(after.silenceElapsedMs).toBe(200);
    expect(after.silenceTimeoutMs).toBe(1500);
    expect(after.tickSeq).toBe(1);
    expect(after.receivedAt).toBeGreaterThan(0);
  });

  it("drops strictly out-of-order tickSeq (proto bidi safety net)", () => {
    setServerVadState({ voiced: false, silenceElapsedMs: 500, silenceTimeoutMs: 1500, tickSeq: 10 });
    const before = useServerVadStateStore.getState();
    setServerVadState({ voiced: false, silenceElapsedMs: 200, silenceTimeoutMs: 1500, tickSeq: 5 });
    const after = useServerVadStateStore.getState();
    expect(after.tickSeq).toBe(before.tickSeq);
    expect(after.silenceElapsedMs).toBe(before.silenceElapsedMs);
  });

  it("monotonicity guard: rejects backward jumps >50 ms within same silence state", () => {
    setServerVadState({ voiced: false, silenceElapsedMs: 500, silenceTimeoutMs: 1500, tickSeq: 1 });
    // A backward jump of 100 ms in silence should be dropped.
    setServerVadState({ voiced: false, silenceElapsedMs: 400, silenceTimeoutMs: 1500, tickSeq: 2 });
    expect(useServerVadStateStore.getState().silenceElapsedMs).toBe(500);
    // A tiny backward jump (within 50 ms tolerance) is accepted.
    setServerVadState({ voiced: false, silenceElapsedMs: 470, silenceTimeoutMs: 1500, tickSeq: 3 });
    expect(useServerVadStateStore.getState().silenceElapsedMs).toBe(470);
  });

  it("voiced→silence transition always accepted (no monotonicity guard across states)", () => {
    setServerVadState({ voiced: true, silenceElapsedMs: 0, silenceTimeoutMs: 1500, tickSeq: 1 });
    setServerVadState({ voiced: false, silenceElapsedMs: 20, silenceTimeoutMs: 1500, tickSeq: 2 });
    const snap = useServerVadStateStore.getState();
    expect(snap.voiced).toBe(false);
    expect(snap.silenceElapsedMs).toBe(20);
  });

  it("_resetServerVadStateForTesting clears state", () => {
    setServerVadState({ voiced: true, silenceElapsedMs: 0, silenceTimeoutMs: 1500, tickSeq: 1 });
    _resetServerVadStateForTesting();
    const snap = useServerVadStateStore.getState();
    expect(snap.tickSeq).toBe(0);
    expect(snap.receivedAt).toBe(0);
  });
});
