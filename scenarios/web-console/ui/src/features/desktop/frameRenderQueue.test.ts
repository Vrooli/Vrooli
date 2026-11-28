import { describe, expect, it } from "vitest";
import { LatestFrameRenderQueue } from "./frameRenderQueue";

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

describe("LatestFrameRenderQueue", () => {
  it("keeps one decode active and commits only the newest waiting frame", async () => {
    const firstDecode = deferred<number>();
    const prepared: number[] = [];
    const committed: number[] = [];
    const discarded: number[] = [];
    let active = 0;
    let peakActive = 0;
    const queue = new LatestFrameRenderQueue<number, number>(
      async (frame) => {
        prepared.push(frame);
        active += 1;
        peakActive = Math.max(peakActive, active);
        try {
          if (frame === 1) return await firstDecode.promise;
          return frame;
        } finally {
          active -= 1;
        }
      },
      (_frame, output) => committed.push(output),
      (output) => discarded.push(output),
      (error) => { throw error; },
    );

    queue.enqueue(1);
    queue.enqueue(2);
    queue.enqueue(3);
    expect(prepared).toEqual([1]);
    firstDecode.resolve(1);
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(prepared).toEqual([1, 3]);
    expect(peakActive).toBe(1);
    expect(discarded).toEqual([1]);
    expect(committed).toEqual([3]);
  });
});
