import { describe, expect, it, vi } from "vitest";
import { isApplyRunSettled, pollApplyRun } from "./applyRun";
import { create } from "@bufbuild/protobuf";
import { ApplyRunState, GetApplyRunResponseSchema, type GetApplyRunResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";

const run = (status: ApplyRunState): GetApplyRunResponse => create(GetApplyRunResponseSchema, {
  runId: "apply-1",
  status,
  legacyStatus: "",
  selectionDigest: "",
  error: "",
  steps: [],
  blockers: [],
  degraded: [],
  degradedDigest: "",
  runnerPid: 0n,
}) as unknown as GetApplyRunResponse;

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
    expect(isApplyRunSettled(ApplyRunState.PENDING)).toBe(false);
    expect(isApplyRunSettled(ApplyRunState.APPLYING)).toBe(false);
    for (const status of [ApplyRunState.APPLIED, ApplyRunState.ALREADY_SATISFIED, ApplyRunState.PARTIALLY_APPLIED, ApplyRunState.CONFIGURATION_INCOMPLETE, ApplyRunState.FAILED]) {
      expect(isApplyRunSettled(status)).toBe(true);
    }
  });
});

describe("pollApplyRun", () => {
  it("follows a run to a settled state", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn()
      .mockResolvedValueOnce(run(ApplyRunState.APPLYING))
      .mockResolvedValueOnce(run(ApplyRunState.APPLIED));
    const seen: string[] = [];

    const final = await pollApplyRun(run(ApplyRunState.PENDING), {
      fetchStatus,
      onUpdate: (current) => seen.push(ApplyRunState[current.status].toLowerCase()),
      wait: clock.wait,
      now: clock.now,
    });

    expect(final.status).toBe(ApplyRunState.APPLIED);
    expect(seen).toEqual(["pending", "applying", "applied"]);
  });

  // The whole point of the server-owned run: the API going away mid-apply is an
  // expected event on this path, because applying restarts scenarios.
  it("keeps waiting when the API disappears and recovers", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn()
      .mockRejectedValueOnce(new Error("Failed to fetch"))
      .mockRejectedValueOnce(new Error("Failed to fetch"))
      .mockResolvedValueOnce(run(ApplyRunState.APPLIED));
    const connectionEvents: boolean[] = [];

    const final = await pollApplyRun(run(ApplyRunState.APPLYING), {
      fetchStatus,
      onUpdate: () => {},
      onConnectionChange: (connected) => connectionEvents.push(connected),
      wait: clock.wait,
      now: clock.now,
    });

    expect(final.status).toBe(ApplyRunState.APPLIED);
    expect(connectionEvents).toEqual([false, true]);
  });

  it("reports the outage only once, not on every failed poll", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn()
      .mockRejectedValueOnce(new Error("down"))
      .mockRejectedValueOnce(new Error("down"))
      .mockRejectedValueOnce(new Error("down"))
      .mockResolvedValueOnce(run(ApplyRunState.APPLIED));
    const connectionEvents: boolean[] = [];

    await pollApplyRun(run(ApplyRunState.APPLYING), {
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

    await expect(pollApplyRun(run(ApplyRunState.APPLYING), {
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

    const final = await pollApplyRun(run(ApplyRunState.ALREADY_SATISFIED), {
      fetchStatus,
      onUpdate: () => {},
      wait: clock.wait,
      now: clock.now,
    });

    expect(final.status).toBe(ApplyRunState.ALREADY_SATISFIED);
    expect(fetchStatus).not.toHaveBeenCalled();
  });
});
