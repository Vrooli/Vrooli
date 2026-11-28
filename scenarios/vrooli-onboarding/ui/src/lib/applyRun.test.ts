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

  it("propagates update callback errors instead of treating them as an API outage", async () => {
    const clock = controlledClock();
    const fetchStatus = vi.fn().mockResolvedValue(run(ApplyRunState.APPLYING));
    const callbackError = new Error("render failed");
    const connectionEvents: boolean[] = [];

    await expect(pollApplyRun(run(ApplyRunState.PENDING), {
      fetchStatus,
      onUpdate: (current) => {
        if (current.status === ApplyRunState.APPLYING) throw callbackError;
      },
      onConnectionChange: (connected) => connectionEvents.push(connected),
      wait: clock.wait,
      now: clock.now,
    })).rejects.toBe(callbackError);

    expect(fetchStatus).toHaveBeenCalledTimes(1);
    expect(connectionEvents).toEqual([]);
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

  it("stops polling after the selected target changes", async () => {
    const controller = new AbortController();
    const fetchStatus = vi.fn().mockResolvedValue(run(ApplyRunState.APPLYING));
    const wait = vi.fn(async () => controller.abort());

    const final = await pollApplyRun(run(ApplyRunState.PENDING), {
      fetchStatus,
      onUpdate: () => {},
      wait,
      now: () => 0,
      signal: controller.signal,
    });

    expect(final.status).toBe(ApplyRunState.PENDING);
    expect(wait).toHaveBeenCalledTimes(1);
    expect(fetchStatus).not.toHaveBeenCalled();
  });

  it("does not publish a status response that arrives after abort", async () => {
    const controller = new AbortController();
    let resolveStatus!: (value: GetApplyRunResponse) => void;
    const statusResponse = new Promise<GetApplyRunResponse>((resolve) => { resolveStatus = resolve; });
    let markFetchStarted!: () => void;
    const fetchStarted = new Promise<void>((resolve) => { markFetchStarted = resolve; });
    const fetchStatus = vi.fn(() => {
      markFetchStarted();
      return statusResponse;
    });
    const onUpdate = vi.fn();
    const polling = pollApplyRun(run(ApplyRunState.APPLYING), {
      fetchStatus,
      onUpdate,
      wait: async () => {},
      now: () => 0,
      signal: controller.signal,
    });

    await fetchStarted;
    controller.abort();
    resolveStatus(run(ApplyRunState.APPLIED));
    const final = await polling;

    expect(final.status).toBe(ApplyRunState.APPLYING);
    expect(onUpdate).toHaveBeenCalledTimes(1);
    expect(onUpdate.mock.calls[0]?.[0].status).toBe(ApplyRunState.APPLYING);
  });
});
