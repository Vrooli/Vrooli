import { describe, expect, it } from "vitest";

import { withPathLock } from "../src/lock.js";

describe("withPathLock", () => {
  it("serializes concurrent calls for the same key", async () => {
    const order: string[] = [];
    let release1: () => void;
    const blocker = new Promise<void>((resolve) => {
      release1 = resolve;
    });

    const p1 = withPathLock("/k", async () => {
      order.push("1:start");
      await blocker;
      order.push("1:end");
    });

    // Yield a tick so p1 actually enters its work() body.
    await new Promise((r) => setImmediate(r));

    const p2 = withPathLock("/k", async () => {
      order.push("2:start");
      order.push("2:end");
    });

    // Yield again — p2 must be queued, not started.
    await new Promise((r) => setImmediate(r));
    expect(order).toEqual(["1:start"]);

    release1!();
    await Promise.all([p1, p2]);
    expect(order).toEqual(["1:start", "1:end", "2:start", "2:end"]);
  });

  it("does not serialize different keys", async () => {
    let release1: () => void;
    const blocker = new Promise<void>((resolve) => {
      release1 = resolve;
    });
    const order: string[] = [];

    const p1 = withPathLock("/a", async () => {
      order.push("a:start");
      await blocker;
      order.push("a:end");
    });
    await new Promise((r) => setImmediate(r));

    const p2 = withPathLock("/b", async () => {
      order.push("b:start");
      order.push("b:end");
    });
    await p2;

    // /b finished while /a still blocked.
    expect(order).toEqual(["a:start", "b:start", "b:end"]);
    release1!();
    await p1;
  });

  it("propagates the work() return value", async () => {
    const v = await withPathLock("/x", async () => 42);
    expect(v).toBe(42);
  });

  it("propagates thrown errors but releases the lock", async () => {
    await expect(
      withPathLock("/k2", async () => {
        throw new Error("boom");
      }),
    ).rejects.toThrow("boom");
    // Next call must proceed.
    const v = await withPathLock("/k2", async () => "ok");
    expect(v).toBe("ok");
  });
});
