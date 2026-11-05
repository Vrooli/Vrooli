// Contract tests for the shared AudioContext lifecycle exported by the
// audio-capture-browser package. The scenario file is intentionally only a
// re-export; lifecycle behavior belongs to the shared package.

import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

import {
  armIdleSuspend,
  closeSharedAudioContext,
  ensureRunningSharedAudioContext,
  getSharedAudioContext,
  keepAudioContextAwake,
  suspendSharedAudioContext,
} from "./sharedAudioContext";

class FakeAudioContext {
  state: AudioContextState = "running";
  readonly sampleRate = 48000;
  readonly currentTime = 0;
  readonly destination = {};
  resume = vi.fn().mockImplementation(() => {
    this.state = "running";
    return Promise.resolve();
  });
  suspend = vi.fn().mockImplementation(() => {
    this.state = "suspended";
    return Promise.resolve();
  });
  close = vi.fn().mockImplementation(() => {
    this.state = "closed";
    return Promise.resolve();
  });
}

let fakeContextInstances: FakeAudioContext[] = [];

beforeEach(() => {
  fakeContextInstances = [];
  const OriginalFake = FakeAudioContext;
  vi.stubGlobal("AudioContext", class extends OriginalFake {
    constructor() {
      super();
      fakeContextInstances.push(this);
    }
  });
  closeSharedAudioContext();
});

afterEach(() => {
  closeSharedAudioContext();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("getSharedAudioContext", () => {
  it("lazily creates one singleton", () => {
    const first = getSharedAudioContext();
    expect(first).toBeInstanceOf(FakeAudioContext);
    expect(getSharedAudioContext()).toBe(first);
    expect(fakeContextInstances).toHaveLength(1);
  });

  it("rebuilds after a terminal context state", () => {
    const first = getSharedAudioContext() as unknown as FakeAudioContext;
    first.state = "closed";
    expect(getSharedAudioContext()).not.toBe(first);
    expect(fakeContextInstances).toHaveLength(2);
  });
});

describe("ensureRunningSharedAudioContext", () => {
  it("resumes a suspended context", async () => {
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    ctx.state = "suspended";

    expect(await ensureRunningSharedAudioContext()).toBe(ctx);
    expect(ctx.resume).toHaveBeenCalledOnce();
  });

  it("rebuilds a context when resume cannot recover it", async () => {
    const first = getSharedAudioContext() as unknown as FakeAudioContext;
    first.state = "suspended";
    first.resume.mockImplementation(() => Promise.resolve());
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});

    const recovered = await ensureRunningSharedAudioContext();
    expect(recovered).not.toBe(first);
    expect(first.close).toHaveBeenCalledOnce();
    expect(fakeContextInstances).toHaveLength(2);
    warn.mockRestore();
  });
});

describe("idle lifecycle", () => {
  it("suspends a running context after the requested delay", () => {
    vi.useFakeTimers();
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;

    armIdleSuspend(100);
    vi.advanceTimersByTime(99);
    expect(ctx.suspend).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(ctx.suspend).toHaveBeenCalledOnce();
  });

  it("cancels an idle suspend when audio is activated", () => {
    vi.useFakeTimers();
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;

    armIdleSuspend(100);
    keepAudioContextAwake();
    vi.advanceTimersByTime(100);
    expect(ctx.suspend).not.toHaveBeenCalled();
  });

  it("suspends immediately through the background backstop", () => {
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;

    suspendSharedAudioContext();
    expect(ctx.suspend).toHaveBeenCalledOnce();
  });

  it("does not arm an idle timer for a non-running context", () => {
    vi.useFakeTimers();
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    ctx.state = "suspended";

    armIdleSuspend(100);
    vi.advanceTimersByTime(100);
    expect(ctx.suspend).not.toHaveBeenCalled();
  });
});

describe("closeSharedAudioContext", () => {
  it("closes and resets the singleton", () => {
    const first = getSharedAudioContext() as unknown as FakeAudioContext;

    closeSharedAudioContext();

    expect(first.close).toHaveBeenCalledOnce();
    expect(getSharedAudioContext()).not.toBe(first);
  });

  it("is safe when no context exists", () => {
    expect(() => closeSharedAudioContext()).not.toThrow();
  });
});
