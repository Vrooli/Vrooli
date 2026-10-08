import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  armIdleSuspend,
  closeSharedAudioContext,
  ensureRunningSharedAudioContext,
  keepAudioContextAwake,
  suspendSharedAudioContext,
} from "./sharedAudioContext";

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
    expect(FakeAudioContext.instances[0]?.close).toHaveBeenCalled();
    // Returned context is the freshly-built one, not the wedged original.
    expect(first).toBe(FakeAudioContext.instances[FakeAudioContext.instances.length - 1]);
  });
});

describe("idle / background audio-session release", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    FakeAudioContext.instances = [];
    (globalThis as unknown as { AudioContext: unknown }).AudioContext =
      vi.fn(() => new FakeAudioContext("suspended"));
  });

  afterEach(() => {
    vi.useRealTimers();
    closeSharedAudioContext();
    vi.restoreAllMocks();
  });

  it("suspendSharedAudioContext suspends a running context (releases the iOS session)", async () => {
    const ctx = (await ensureRunningSharedAudioContext()) as unknown as FakeAudioContext & {
      suspend: ReturnType<typeof vi.fn>;
    };
    expect(ctx.state).toBe("running");
    ctx.suspend = vi.fn(async () => { ctx.state = "suspended"; });
    suspendSharedAudioContext();
    expect(ctx.suspend).toHaveBeenCalledTimes(1);
  });

  it("armIdleSuspend suspends after the delay, and keepAudioContextAwake cancels it", async () => {
    const ctx = (await ensureRunningSharedAudioContext()) as unknown as FakeAudioContext & {
      suspend: ReturnType<typeof vi.fn>;
    };
    ctx.suspend = vi.fn(async () => { ctx.state = "suspended"; });

    // Armed, then cancelled before the timer fires → no suspend.
    armIdleSuspend(1500);
    keepAudioContextAwake();
    vi.advanceTimersByTime(2000);
    expect(ctx.suspend).not.toHaveBeenCalled();

    // Armed and left alone → suspends after the delay.
    ctx.state = "running";
    armIdleSuspend(1500);
    vi.advanceTimersByTime(1500);
    expect(ctx.suspend).toHaveBeenCalledTimes(1);
  });
});
