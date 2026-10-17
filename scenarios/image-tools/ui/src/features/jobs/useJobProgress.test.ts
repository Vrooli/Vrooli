/**
 * useJobProgress tests — the WatchJob (SSE-style) live-progress subscription.
 *
 * Branches exercised:
 *   - disabled (`enabled === false`) → no subscription, stays null
 *   - empty jobId → no subscription, stays null
 *   - active subscription replays + streams events into state
 *   - a stream that throws is swallowed (the catch), state untouched
 *   - unmount / disable aborts the controller (teardown)
 *
 * `../../api/jobs` is mocked via the co-located builder so only the
 * network-touching client methods are substituted; the re-exported proto
 * types/enums stay intact.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";

import { asProgressStream, makeProgressEvent } from "./mocks/factories";

// vi.mock is hoisted above all imports, so the mock builder cannot be a
// top-level binding. Resolve it inside vi.hoisted (async + dynamic import,
// the sanctioned escape hatch) so the substituted client is shared with
// the test bodies below via the returned `mocks` object.
const { mocks } = await vi.hoisted(async () => {
  const { makeJobsMocks } = await import("./mocks/jobs");
  return { mocks: makeJobsMocks() };
});

vi.mock("../../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/jobs")>();
  return { ...actual, ...mocks };
});

import { useJobProgress } from "./useJobProgress";

describe("useJobProgress", () => {
  beforeEach(() => {
    mocks.jobsClient.watchJob.mockReset();
    // Default: empty stream that completes immediately.
    mocks.jobsClient.watchJob.mockImplementation(() => asProgressStream([]));
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("does not subscribe and returns null when disabled", () => {
    const { result } = renderHook(() => useJobProgress("job-1", false));

    expect(mocks.jobsClient.watchJob).not.toHaveBeenCalled();
    expect(result.current).toBeNull();
  });

  it("does not subscribe and returns null when the jobId is empty", () => {
    const { result } = renderHook(() => useJobProgress("", true));

    expect(mocks.jobsClient.watchJob).not.toHaveBeenCalled();
    expect(result.current).toBeNull();
  });

  it("subscribes and surfaces the latest streamed event", async () => {
    const event = makeProgressEvent({ jobId: "job-1", progress: 80 });
    mocks.jobsClient.watchJob.mockImplementation(() => asProgressStream([event]));

    const { result } = renderHook(() => useJobProgress("job-1", true));

    expect(mocks.jobsClient.watchJob).toHaveBeenCalledTimes(1);
    expect(mocks.jobsClient.watchJob).toHaveBeenCalledWith(
      { id: "job-1" },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );

    await waitFor(() => {
      expect(result.current?.progress).toBe(80);
    });
  });

  it("applies the most recent event when several stream in", async () => {
    mocks.jobsClient.watchJob.mockImplementation(() =>
      asProgressStream([
        makeProgressEvent({ progress: 10 }),
        makeProgressEvent({ progress: 55 }),
        makeProgressEvent({ progress: 100 }),
      ]),
    );

    const { result } = renderHook(() => useJobProgress("job-1", true));

    await waitFor(() => {
      expect(result.current?.progress).toBe(100);
    });
  });

  it("swallows a stream error and leaves state null", async () => {
    mocks.jobsClient.watchJob.mockImplementation(async function* () {
      await Promise.resolve();
      throw new Error("stream failed");
       
      yield makeProgressEvent();
    });

    const { result } = renderHook(() => useJobProgress("job-1", true));

    // Give the microtask queue a chance to run the rejected generator.
    await Promise.resolve();
    await Promise.resolve();

    expect(result.current).toBeNull();
  });

  it("aborts the WatchJob request on unmount", () => {
    let captured: AbortSignal | undefined;
    mocks.jobsClient.watchJob.mockImplementation(
      (_input: { id: string }, opts: { signal: AbortSignal }) => {
        captured = opts.signal;
        return asProgressStream([]);
      },
    );

    const { unmount } = renderHook(() => useJobProgress("job-1", true));
    expect(captured).toBeDefined();
    expect(captured?.aborted).toBe(false);

    unmount();
    expect(captured?.aborted).toBe(true);
  });

  it("aborts and resubscribes when the jobId changes", () => {
    const signals: AbortSignal[] = [];
    mocks.jobsClient.watchJob.mockImplementation(
      (_input: { id: string }, opts: { signal: AbortSignal }) => {
        signals.push(opts.signal);
        return asProgressStream([]);
      },
    );

    const { rerender } = renderHook(({ id }: { id: string }) => useJobProgress(id, true), {
      initialProps: { id: "job-1" },
    });
    rerender({ id: "job-2" });

    expect(mocks.jobsClient.watchJob).toHaveBeenCalledTimes(2);
    // The first subscription's controller is aborted by the cleanup.
    expect(signals[0]?.aborted).toBe(true);
    expect(signals[1]?.aborted).toBe(false);
  });
});
