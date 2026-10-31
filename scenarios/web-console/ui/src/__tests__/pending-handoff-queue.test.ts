import { describe, expect, it, vi } from "vitest";

import { PendingHandoffQueue } from "../hooks/pendingHandoffQueue";

function createStorage() {
  const values = new Map<string, string>();
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => { values.set(key, value); },
    removeItem: (key: string) => { values.delete(key); },
  };
}

describe("PendingHandoffQueue", () => {
  it("delivers two queued payloads once each and in FIFO order", () => {
    const queue = new PendingHandoffQueue(createStorage());
    queue.enqueue("session-a", "first context");
    queue.enqueue("session-a", "second context");
    const deliver = vi.fn((text: string) => ({ status: "sent" as const, offset: text.length }));

    expect(queue.flush("session-a", deliver)).toBe(2);
    expect(deliver.mock.calls.map(([text]) => text)).toEqual(["first context", "second context"]);
    expect(queue.getPending("session-a")).toEqual([]);
    expect(queue.flush("session-a", deliver)).toBe(0);
    expect(deliver).toHaveBeenCalledTimes(2);
  });

  it("keeps a rejected payload and later payloads for a future retry", () => {
    const queue = new PendingHandoffQueue(createStorage());
    queue.enqueue("session-a", "first context");
    queue.enqueue("session-a", "second context");
    const deliver = vi.fn()
      .mockReturnValueOnce({ status: "queued", reason: "not-ready" } as const)
      .mockReturnValueOnce({ status: "rejected", reason: "disposed" } as const);

    expect(queue.flush("session-a", deliver)).toBe(1);
    expect(queue.getPending("session-a")).toEqual(["second context"]);

    const retry = vi.fn(() => ({ status: "sent", offset: 0 } as const));
    expect(queue.flush("session-a", retry)).toBe(1);
    expect(retry).toHaveBeenCalledTimes(1);
    expect(retry).toHaveBeenCalledWith("second context");
    expect(queue.getPending("session-a")).toEqual([]);
  });

  it("drops pending payloads when their session is deleted", () => {
    const storage = createStorage();
    const queue = new PendingHandoffQueue(storage);
    queue.enqueue("session-a", "stale context");
    const afterReload = new PendingHandoffQueue(storage);

    afterReload.removeSession("session-a");

    const afterRemovalReload = new PendingHandoffQueue(storage);
    expect(afterRemovalReload.getPending("session-a")).toEqual([]);
    expect(afterRemovalReload.flush("session-a", vi.fn(() => ({ status: "sent", offset: 0 } as const)))).toBe(0);
  });

  it("restores pending payloads after a reload and clears them after acceptance", () => {
    const storage = createStorage();
    const beforeReload = new PendingHandoffQueue(storage);
    beforeReload.enqueue("session-a", "first context");
    beforeReload.enqueue("session-a", "second context");

    const afterReload = new PendingHandoffQueue(storage);
    expect(afterReload.getPending("session-a")).toEqual(["first context", "second context"]);
    const deliver = vi.fn((text: string) => ({ status: "sent" as const, offset: text.length }));
    expect(afterReload.flush("session-a", deliver)).toBe(2);
    expect(afterReload.flush("session-a", deliver)).toBe(0);
    expect(deliver).toHaveBeenCalledTimes(2);

    const afterSecondReload = new PendingHandoffQueue(storage);
    expect(afterSecondReload.getPending("session-a")).toEqual([]);
  });
});
