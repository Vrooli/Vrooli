import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { closeSharedAudioContext, ensureRunningSharedAudioContext } from "./sharedAudioContext";

// Minimal controllable AudioContext fake. `state` is mutable so a test can
// simulate a wedged context whose resume() does or does not recover it.
class FakeAudioContext {
  static instances: FakeAudioContext[] = [];
  state: "suspended" | "running" | "closed" | "interrupted";
  resume = vi.fn(async () => { if (this.resumeRecovers) this.state = "running"; });
  close = vi.fn(async () => { this.state = "closed"; });
  resumeRecovers = true;

  constructor(initial: FakeAudioContext["state"] = "suspended") {
    this.state = initial;
    FakeAudioContext.instances.push(this);
  }
}

describe("ensureRunningSharedAudioContext", () => {
  beforeEach(() => {
    FakeAudioContext.instances = [];
    // Each new context starts suspended; resume() recovers it by default.
    (globalThis as unknown as { AudioContext: unknown }).AudioContext =
      vi.fn(() => new FakeAudioContext("suspended"));
  });

  afterEach(() => {
    closeSharedAudioContext();
    vi.restoreAllMocks();
  });

  it("resumes a suspended context to running (no rebuild)", async () => {
    const ctx = await ensureRunningSharedAudioContext();
    expect(ctx.state).toBe("running");
    expect(FakeAudioContext.instances).toHaveLength(1);
    expect((ctx as unknown as FakeAudioContext).resume).toHaveBeenCalled();
  });

  it("rebuilds a wedged context whose resume() never recovers it", async () => {
    // First context: resume() does NOT bring it back — the stuck state that
    // previously survived until a full page reload.
    (globalThis as unknown as { AudioContext: unknown }).AudioContext = vi.fn(() => {
      const c = new FakeAudioContext("suspended");
      c.resumeRecovers = false;
      return c;
    });
    const first = await ensureRunningSharedAudioContext();
    expect(FakeAudioContext.instances.length).toBeGreaterThanOrEqual(2);
    // The wedged one was closed and discarded.
    expect(FakeAudioContext.instances[0]!.close).toHaveBeenCalled();
    // Returned context is the freshly-built one, not the wedged original.
    expect(first).toBe(FakeAudioContext.instances[FakeAudioContext.instances.length - 1]);
  });
});
