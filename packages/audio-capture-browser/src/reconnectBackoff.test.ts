import { describe, expect, it } from "vitest";
import { MAX_CONSECUTIVE_RECONNECTS, RECONNECT_BACKOFF_MS, ReconnectBackoff } from "./reconnectBackoff";

describe("ReconnectBackoff", () => {
  it("returns the bounded delays in order and then exhausts", () => {
    const backoff = new ReconnectBackoff();
    expect(Array.from({ length: MAX_CONSECUTIVE_RECONNECTS + 1 }, () => backoff.nextDelay()))
      .toEqual([...RECONNECT_BACKOFF_MS, null]);
    expect(backoff.attemptCount).toBe(MAX_CONSECUTIVE_RECONNECTS);
  });

  it("starts a fresh consecutive-retry budget after reset", () => {
    const backoff = new ReconnectBackoff();
    backoff.nextDelay();
    backoff.nextDelay();
    backoff.reset();
    expect(backoff.attemptCount).toBe(0);
    expect(backoff.nextDelay()).toBe(RECONNECT_BACKOFF_MS[0]);
  });
});
