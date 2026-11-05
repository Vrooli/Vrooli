import { describe, expect, it, vi } from "vitest";
import { createStageBatcher, type StageBatchRunner } from "./stage-batcher";

describe("createStageBatcher", () => {
  it("combines clicks received while the first stage write is active", () => {
    const run = vi.fn<StageBatchRunner<{ success: boolean }>>();
    const pending = vi.fn<(paths: string[]) => void>();
    let scheduledFlush: (() => void) | undefined;
    const batcher = createStageBatcher(
      run,
      pending,
      {
        schedule: (callback) => {
          scheduledFlush = callback;
          return callback;
        },
        cancelSchedule: () => undefined,
      },
    );

    batcher.enqueue(["one.ts"], vi.fn());
    batcher.enqueue(["two.ts"]);
    batcher.enqueue(["three.ts", "two.ts"]);

    expect(run).toHaveBeenCalledTimes(1);
    expect(run.mock.calls[0]?.[0]).toEqual(["one.ts"]);
    expect(pending).toHaveBeenLastCalledWith(["two.ts", "three.ts"]);

    const firstSettled = run.mock.calls[0]?.[2];
    expect(firstSettled).toBeDefined();
    firstSettled?.();
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
      schedule: (callback) => {
        scheduledFlush = callback;
        return callback;
      },
      cancelSchedule: () => undefined,
    });

    batcher.enqueue(["one.ts"], onFirstSuccess);
    batcher.enqueue(["two.ts"], onSecondSuccess);
    run.mock.calls[0]?.[1]({ success: true });

    expect(onFirstSuccess).toHaveBeenCalledWith({ success: true });
    expect(onSecondSuccess).not.toHaveBeenCalled();

    run.mock.calls[0]?.[2]();
    scheduledFlush?.();
    run.mock.calls[1]?.[1]({ success: true });
    expect(onSecondSuccess).toHaveBeenCalledWith({ success: true });
  });
});
