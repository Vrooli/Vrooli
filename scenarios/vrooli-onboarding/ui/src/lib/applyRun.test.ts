import { describe, expect, it, vi } from "vitest";
import { isApplyRunSettled, pollApplyRun } from "./applyRun";
import type { V2ApplyResponse } from "../types";

const run = (status: string): V2ApplyResponse => ({ run_id: "apply-1", status, items: [] });

// A clock the test advances itself, so the reconnect window is exercised
// without the test actually waiting five minutes.
function controlledClock() {
  let current = 0;
  return {
    now: () => current,
    wait: async (ms: number) => {
      current += ms;
    },
    advance: (ms: number) => {
      current += ms;
    },
  };
}

describe("isApplyRunSettled", () => {
  it("treats only in-flight statuses as unsettled", () => {
    expect(isApplyRunSettled("pending")).toBe(false);
    expect(isApplyRunSettled("applying")).toBe(false);
    for (const status of ["applied", "partially_applied", "configuration_incomplete", "interrupted", "failed"]) {
      expect(isApplyRunSettled(status)).toBe(true);
    }
  });
});

describe("pollApplyRun", () => {
  it("follows a run to a settled state", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn()
      .mockResolvedValueOnce(run("applying"))
      .mockResolvedValueOnce(run("applied"));
    const seen: string[] = [];

    const final = await pollApplyRun(run("pending"), {
      fetchStatus,
      onUpdate: (current) => seen.push(current.status),
      wait: clock.wait,
      now: clock.now,
    });

    expect(final.status).toBe("applied");
    expect(seen).toEqual(["pending", "applying", "applied"]);
  });

  // The whole point of the server-owned run: the API going away mid-apply is an
  // expected event on this path, because applying restarts scenarios.
  it("keeps waiting when the API disappears and recovers", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn()
      .mockRejectedValueOnce(new Error("Failed to fetch"))
      .mockRejectedValueOnce(new Error("Failed to fetch"))
      .mockResolvedValueOnce(run("applied"));
    const connectionEvents: boolean[] = [];

    const final = await pollApplyRun(run("applying"), {
      fetchStatus,
      onUpdate: () => {},
      onConnectionChange: (connected) => connectionEvents.push(connected),
      wait: clock.wait,
      now: clock.now,
    });

    expect(final.status).toBe("applied");
    expect(connectionEvents).toEqual([false, true]);
  });

  it("reports the outage only once, not on every failed poll", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn()
      .mockRejectedValueOnce(new Error("down"))
      .mockRejectedValueOnce(new Error("down"))
      .mockRejectedValueOnce(new Error("down"))
      .mockResolvedValueOnce(run("applied"));
    const connectionEvents: boolean[] = [];

    await pollApplyRun(run("applying"), {
      fetchStatus,
      onUpdate: () => {},
      onConnectionChange: (connected) => connectionEvents.push(connected),
      wait: clock.wait,
      now: clock.now,
    });

    expect(connectionEvents).toEqual([false, true]);
  });

  it("gives up only after the reconnect window", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn().mockRejectedValue(new Error("still down"));

    await expect(pollApplyRun(run("applying"), {
      fetchStatus,
      onUpdate: () => {},
      wait: async (ms) => {
        // Simulate a long outage: each poll costs far more than the interval.
        clock.advance(ms + 60_000);
      },
      now: clock.now,
      reconnectWindowMs: 120_000,
    })).rejects.toThrow("still down");
  });

  it("never polls a run that arrived already settled", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn();

    const final = await pollApplyRun(run("already_satisfied"), {
      fetchStatus,
      onUpdate: () => {},
      wait: clock.wait,
      now: clock.now,
    });

    expect(final.status).toBe("already_satisfied");
    expect(fetchStatus).not.toHaveBeenCalled();
  });
});
