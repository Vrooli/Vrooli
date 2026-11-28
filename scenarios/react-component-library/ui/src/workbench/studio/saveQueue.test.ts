import { describe, expect, it, vi } from "vitest";

import { createSaveQueue } from "./saveQueue";

const tick = () => new Promise((resolve) => setTimeout(resolve, 0));

describe("save queue", () => {
  it("runs one save at a time per design", async () => {
    const order: string[] = [];
    let release: (() => void) | undefined;
    const persist = vi.fn(async (id: string) => {
      order.push(`start:${id}`);
      await new Promise<void>((resolve) => {
        release = resolve;
      });
      order.push(`end:${id}`);
    });
    const queue = createSaveQueue(persist, () => {});
    queue.run("a");
    queue.run("a");
    expect(persist).toHaveBeenCalledTimes(1);
    expect(queue.isSaving("a")).toBe(true);
    release?.();
    await tick();
    // the second request coalesced into exactly one follow-up
    expect(persist).toHaveBeenCalledTimes(2);
    release?.();
    await tick();
    expect(persist).toHaveBeenCalledTimes(2);
    expect(order).toEqual(["start:a", "end:a", "start:a", "end:a"]);
  });

  it("lets different designs save in parallel", () => {
    const persist = vi.fn(() => new Promise<void>(() => {}));
    const queue = createSaveQueue(persist, () => {});
    queue.run("a");
    queue.run("b");
    expect(persist).toHaveBeenCalledTimes(2);
    expect(queue.isSaving("a")).toBe(true);
    expect(queue.isSaving("b")).toBe(true);
  });

  it("stops instead of looping when the save fails", async () => {
    const persist = vi.fn(() => Promise.reject(new Error("offline")));
    const states: boolean[] = [];
    const queue = createSaveQueue(persist, (active) => states.push(active));
    queue.run("a");
    queue.run("a");
    await tick();
    await tick();
    expect(persist).toHaveBeenCalledTimes(1);
    expect(queue.isSaving("a")).toBe(false);
    expect(states.at(-1)).toBe(false);
  });

  it("does not start queued work after dispose", async () => {
    let release: (() => void) | undefined;
    const persist = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          release = resolve;
        }),
    );
    const queue = createSaveQueue(persist, () => {});
    queue.run("a");
    queue.run("a");
    queue.dispose();
    release?.();
    await tick();
    expect(persist).toHaveBeenCalledTimes(1);
  });
});
