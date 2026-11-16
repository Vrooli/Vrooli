import { describe, expect, it, vi } from "vitest";
import { createStageBatcher, type StageBatchRunner } from "./stage-batcher";

describe("createStageBatcher", () => {
  it("combines clicks received while the first stage write is active", () => {
    const run = vi.fn<StageBatchRunner<{ success: boolean }>>();
    const pending = vi.fn<(paths: string[]) => void>();
    let scheduledFlush: (() => void) | undefined;
    const batcher = createStageBatcher(run, pending, {
      schedule: callback => { scheduledFlush = callback; return callback; },
      cancelSchedule: () => undefined,
    });

    batcher.enqueue(["one.ts"], { onSuccess: vi.fn() });
    batcher.enqueue(["two.ts"]);
    batcher.enqueue(["three.ts", "two.ts"]);

    expect(run).toHaveBeenCalledTimes(1);
    expect(run.mock.calls[0]?.[0]).toEqual(["one.ts"]);
    expect(pending).toHaveBeenLastCalledWith(["two.ts", "three.ts"]);

    run.mock.calls[0]?.[1].onSettled();
    expect(run).toHaveBeenCalledTimes(1);
    scheduledFlush?.();
    expect(run).toHaveBeenCalledTimes(2);
    expect(run.mock.calls[1]?.[0]).toEqual(["two.ts", "three.ts"]);
  });

  it("keeps callbacks attached to their own request when later clicks are queued", () => {
    const onFirstSuccess = vi.fn();
    const onSecondSuccess = vi.fn();
    const run = vi.fn<StageBatchRunner<{ success: boolean }>>();
    let scheduledFlush: (() => void) | undefined;
    const batcher = createStageBatcher(run, vi.fn(), {
      schedule: callback => { scheduledFlush = callback; return callback; },
      cancelSchedule: () => undefined,
    });

    batcher.enqueue(["one.ts"], { onSuccess: onFirstSuccess });
    batcher.enqueue(["two.ts"], { onSuccess: onSecondSuccess });
    run.mock.calls[0]?.[1].onSuccess({ success: true });

    expect(onFirstSuccess).toHaveBeenCalledWith({ success: true });
    expect(onSecondSuccess).not.toHaveBeenCalled();

    run.mock.calls[0]?.[1].onSettled();
    scheduledFlush?.();
    run.mock.calls[1]?.[1].onSuccess({ success: true });
    expect(onSecondSuccess).toHaveBeenCalledWith({ success: true });
  });

  it("runs settlement callbacks even when a stage request has no success response", () => {
    const onSettled = vi.fn();
    const run = vi.fn<StageBatchRunner<{ success: boolean }>>();
    const batcher = createStageBatcher(run, vi.fn());
    batcher.enqueue(["one.ts"], { onSettled });
    expect(onSettled).not.toHaveBeenCalled();
    run.mock.calls[0]?.[1].onSettled();
    expect(onSettled).toHaveBeenCalledOnce();
  });

  it("delivers request failures to their callback and still settles", () => {
    const onError = vi.fn();
    const onSettled = vi.fn();
    const run = vi.fn<StageBatchRunner<{ success: boolean }>>();
    const batcher = createStageBatcher(run, vi.fn());
    const failure = new Error("staging request failed");

    batcher.enqueue(["one.ts"], { onSettled, onError });
    run.mock.calls[0]?.[1].onError(failure);
    expect(onError).toHaveBeenCalledWith(failure);
    expect(onSettled).not.toHaveBeenCalled();
    run.mock.calls[0]?.[1].onSettled();
    expect(onSettled).toHaveBeenCalledOnce();
  });

  it("releases a queued dispatch as retryable when its repository batcher is disposed", () => {
    const onError = vi.fn();
    const onSettled = vi.fn();
    const run = vi.fn<StageBatchRunner<{ success: boolean }>>();
    const batcher = createStageBatcher(run, vi.fn(), { scopeKey: "repo-a" });

    batcher.enqueue(["one.ts"], { onSettled, onError });
    batcher.dispose();

    expect(onError).toHaveBeenCalledOnce();
    expect(onError.mock.calls[0]?.[0]).toBeInstanceOf(Error);
    expect(onSettled).toHaveBeenCalledOnce();
    run.mock.calls[0]?.[1].onSettled();
    expect(onSettled).toHaveBeenCalledOnce();
  });

  it("fails and settles follow-up callbacks that never started before disposal", () => {
    const firstError = vi.fn();
    const firstSettled = vi.fn();
    const secondError = vi.fn();
    const secondSettled = vi.fn();
    const run = vi.fn<StageBatchRunner<{ success: boolean }>>();
    const batcher = createStageBatcher(run, vi.fn());

    batcher.enqueue(["one.ts"], { onSettled: firstSettled, onError: firstError });
    batcher.enqueue(["two.ts"], { onSettled: secondSettled, onError: secondError });
    batcher.dispose();

    expect(run).toHaveBeenCalledTimes(1);
    expect(firstError).toHaveBeenCalledOnce();
    expect(firstSettled).toHaveBeenCalledOnce();
    expect(secondError).toHaveBeenCalledOnce();
    expect(secondSettled).toHaveBeenCalledOnce();
  });

  it("settles follow-up callbacks once when disposed after the active batch schedules them", () => {
    const firstSettled = vi.fn();
    const secondError = vi.fn();
    const secondSettled = vi.fn();
    const run = vi.fn<StageBatchRunner<{ success: boolean }>>();
    let scheduledFlush: (() => void) | undefined;
    const batcher = createStageBatcher(run, vi.fn(), {
      schedule: callback => { scheduledFlush = callback; return callback; },
      cancelSchedule: () => undefined,
    });

    batcher.enqueue(["one.ts"], { onSettled: firstSettled });
    batcher.enqueue(["two.ts"], { onError: secondError, onSettled: secondSettled });
    run.mock.calls[0]?.[1].onSettled();
    expect(scheduledFlush).toBeTypeOf("function");
    batcher.dispose();

    expect(firstSettled).toHaveBeenCalledOnce();
    expect(secondError).toHaveBeenCalledOnce();
    expect(secondSettled).toHaveBeenCalledOnce();
    scheduledFlush?.();
    expect(run).toHaveBeenCalledOnce();
    expect(secondSettled).toHaveBeenCalledOnce();
  });
});
