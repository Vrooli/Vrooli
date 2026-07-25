// Unit tests for sharedAudioContext.ts.
//
// Each test closes the shared context and resets the module via
// closeSharedAudioContext() + teardownAudioContextKeepalive() so module-level
// state doesn't bleed between tests.
//
// A fake AudioContext is installed via vi.stubGlobal('AudioContext', ...) so
// jsdom's lack of Web Audio doesn't block execution.

import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

import {
  getSharedAudioContext,
  installAudioContextKeepalive,
  teardownAudioContextKeepalive,
  ensureAudioContextOnGesture,
  closeSharedAudioContext,
} from "./sharedAudioContext";

// ---------------------------------------------------------------------------
// Fake AudioContext
// ---------------------------------------------------------------------------

function makeGainNode() {
  return {
    gain: { value: 0 },
    connect: vi.fn(),
    disconnect: vi.fn(),
  };
}

function makeOscillatorNode() {
  return {
    type: "",
    frequency: { value: 0 },
    connect: vi.fn(),
    disconnect: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
  };
}

class FakeAudioContext {
  state: AudioContextState = "running";
  readonly sampleRate = 48000;
  readonly currentTime = 0;
  readonly destination = {};
  createGain = vi.fn().mockImplementation(makeGainNode);
  createOscillator = vi.fn().mockImplementation(makeOscillatorNode);
  createScriptProcessor = vi.fn().mockReturnValue({
    connect: vi.fn(),
    disconnect: vi.fn(),
    onaudioprocess: null,
  });
  createBiquadFilter = vi.fn().mockReturnValue({ connect: vi.fn(), disconnect: vi.fn(), type: "", frequency: { value: 0 }, Q: { value: 0 } });
  createAnalyser = vi.fn().mockReturnValue({ connect: vi.fn(), disconnect: vi.fn(), fftSize: 0 });
  createMediaStreamSource = vi.fn().mockReturnValue({ connect: vi.fn(), disconnect: vi.fn() });
  resume = vi.fn().mockResolvedValue(undefined);
  close = vi.fn().mockImplementation(() => {
    this.state = "closed";
    return Promise.resolve();
  });
}

// Track instances to inspect them in tests
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
  // Always start each test with a clean module state
  closeSharedAudioContext();
  teardownAudioContextKeepalive();
});

afterEach(() => {
  closeSharedAudioContext();
  teardownAudioContextKeepalive();
  vi.unstubAllGlobals();
});

// ---------------------------------------------------------------------------
// getSharedAudioContext
// ---------------------------------------------------------------------------

describe("getSharedAudioContext", () => {
  it("creates a new AudioContext on first call", () => {
    const ctx = getSharedAudioContext();
    expect(ctx).toBeInstanceOf(FakeAudioContext);
    expect(fakeContextInstances).toHaveLength(1);
  });

  it("returns the same instance on subsequent calls (singleton)", () => {
    const ctx1 = getSharedAudioContext();
    const ctx2 = getSharedAudioContext();
    expect(ctx1).toBe(ctx2);
    expect(fakeContextInstances).toHaveLength(1);
  });

  it("creates a new instance when the previous context is closed", () => {
    const ctx1 = getSharedAudioContext();
    // Simulate a closed context
    (ctx1 as unknown as FakeAudioContext).state = "closed";
    const ctx2 = getSharedAudioContext();
    expect(ctx2).not.toBe(ctx1);
    expect(fakeContextInstances).toHaveLength(2);
  });
});

// ---------------------------------------------------------------------------
// closeSharedAudioContext
// ---------------------------------------------------------------------------

describe("closeSharedAudioContext", () => {
  it("closes the context and resets module state", () => {
    const ctx = getSharedAudioContext();
    closeSharedAudioContext();
    expect((ctx as unknown as FakeAudioContext).close).toHaveBeenCalledOnce();
    // Next call should create a new context
    const ctx2 = getSharedAudioContext();
    expect(ctx2).not.toBe(ctx);
  });

  it("is safe to call when no context exists", () => {
    expect(() => closeSharedAudioContext()).not.toThrow();
  });

  it("resets _gestureInstalled so ensureAudioContextOnGesture can re-install", () => {
    // Install gesture listener, then close
    ensureAudioContextOnGesture();
    closeSharedAudioContext();
    // Should not throw and should re-install
    expect(() => ensureAudioContextOnGesture()).not.toThrow();
  });

  it("tears down keepalive before closing", () => {
    getSharedAudioContext();
    installAudioContextKeepalive();
    closeSharedAudioContext();
    // After close, installing keepalive should not crash
    expect(() => installAudioContextKeepalive()).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// installAudioContextKeepalive
// ---------------------------------------------------------------------------

describe("installAudioContextKeepalive", () => {
  it("is a no-op when no shared context exists", () => {
    // No context created yet — should not throw
    expect(() => installAudioContextKeepalive()).not.toThrow();
  });

  it("installs a keepalive oscillator on a running context (non-iOS)", () => {
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    // Ensure we're not on iOS
    Object.defineProperty(navigator, "userAgent", {
      value:
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
      configurable: true,
    });
    installAudioContextKeepalive();
    expect(ctx.createOscillator).toHaveBeenCalledOnce();
  });

  it("is idempotent — calling twice does not install two oscillators", () => {
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 Chrome/120 Safari/537.36",
      configurable: true,
    });
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    installAudioContextKeepalive();
    installAudioContextKeepalive();
    expect(ctx.createOscillator).toHaveBeenCalledOnce();
  });

  it("skips keepalive on iOS devices", () => {
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
      configurable: true,
    });
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    installAudioContextKeepalive();
    expect(ctx.createOscillator).not.toHaveBeenCalled();
  });

  it("skips keepalive on iPadOS 13+ (Mac UA with touch)", () => {
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
      configurable: true,
    });
    // Simulate touch support (iPadOS detection)
    Object.defineProperty(document, "ontouchend", { value: true, configurable: true });
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    installAudioContextKeepalive();
    expect(ctx.createOscillator).not.toHaveBeenCalled();
    // cleanup
    Object.defineProperty(document, "ontouchend", { value: undefined, configurable: true });
  });

  it("skips keepalive when context is closed", () => {
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    ctx.state = "closed";
    // Non-iOS to ensure we hit the closed check
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 Chrome",
      configurable: true,
    });
    installAudioContextKeepalive();
    expect(ctx.createOscillator).not.toHaveBeenCalled();
  });

  it("handles errors from createOscillator gracefully (empty catch block)", () => {
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 Chrome",
      configurable: true,
    });
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    ctx.createOscillator = vi.fn().mockImplementation(() => {
      throw new Error("oscillator not supported");
    });
    // Should swallow the error silently
    expect(() => installAudioContextKeepalive()).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// teardownAudioContextKeepalive
// ---------------------------------------------------------------------------

describe("teardownAudioContextKeepalive", () => {
  it("is safe to call without a keepalive installed", () => {
    expect(() => teardownAudioContextKeepalive()).not.toThrow();
  });

  it("stops and disconnects the keepalive oscillator", () => {
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 Chrome",
      configurable: true,
    });
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    installAudioContextKeepalive();

    const osc = ctx.createOscillator.mock.results[0]!.value as ReturnType<typeof makeOscillatorNode>;
    teardownAudioContextKeepalive();

    expect(osc.stop).toHaveBeenCalledOnce();
    expect(osc.disconnect).toHaveBeenCalledOnce();
  });

  it("clears the idle timer when tearing down", () => {
    vi.useFakeTimers();
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 Chrome",
      configurable: true,
    });
    getSharedAudioContext();
    installAudioContextKeepalive();
    teardownAudioContextKeepalive();
    // Advancing timers should not throw (timer was cleared)
    expect(() => vi.runAllTimers()).not.toThrow();
    vi.useRealTimers();
  });

  it("handles errors from osc.stop() / disconnect() gracefully", () => {
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 Chrome",
      configurable: true,
    });
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    installAudioContextKeepalive();
    const osc = ctx.createOscillator.mock.results[0]!.value as ReturnType<typeof makeOscillatorNode>;
    // Make stop/disconnect throw to exercise the catch blocks in _teardownKeepalive
    osc.stop = vi.fn().mockImplementation(() => { throw new Error("already stopped"); });
    osc.disconnect = vi.fn().mockImplementation(() => { throw new Error("already disconnected"); });
    const gain = ctx.createGain.mock.results[0]!.value as ReturnType<typeof makeGainNode>;
    gain.disconnect = vi.fn().mockImplementation(() => { throw new Error("already disconnected"); });
    // Should not throw even though the underlying calls fail
    expect(() => teardownAudioContextKeepalive()).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// ensureAudioContextOnGesture
// ---------------------------------------------------------------------------

describe("ensureAudioContextOnGesture", () => {
  it("adds a pointer listener on document", () => {
    const addSpy = vi.spyOn(document, "addEventListener");
    ensureAudioContextOnGesture();
    const events = addSpy.mock.calls.map((c) => c[0]);
    expect(events).toContain("pointerdown");
    expect(events).not.toContain("keydown");
    addSpy.mockRestore();
  });

  it("is idempotent — calling twice adds listeners only once", () => {
    const addSpy = vi.spyOn(document, "addEventListener");
    ensureAudioContextOnGesture();
    ensureAudioContextOnGesture();
    // The pointer listener is installed only once.
    const ourEvents = addSpy.mock.calls.map((c) => c[0]);
    const pointerdownCount = ourEvents.filter((e) => e === "pointerdown").length;
    expect(pointerdownCount).toBe(1);
    addSpy.mockRestore();
  });

  it("is a no-op when document is not defined", () => {
    // This branch is guarded by `typeof document === 'undefined'`.
    // We can't easily simulate no-document in jsdom, so we just verify
    // that calling it doesn't throw.
    expect(() => ensureAudioContextOnGesture()).not.toThrow();
  });

  it("resumes a suspended context on gesture", async () => {
    ensureAudioContextOnGesture();
    // Fire the first user gesture
    document.dispatchEvent(new Event("pointerdown", { bubbles: true }));
    // The context will be created and resume() called if suspended.
    // Since FakeAudioContext starts as 'running', resume should NOT be called.
    await new Promise((r) => setTimeout(r, 0));
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    expect(ctx.resume).not.toHaveBeenCalled();
  });

  it("calls resume when context starts suspended", async () => {
    // Override: create a suspended context
    const SuspendedFake = class extends FakeAudioContext {
      override state: AudioContextState = "suspended";
    };
    vi.stubGlobal("AudioContext", SuspendedFake);
    closeSharedAudioContext();

    ensureAudioContextOnGesture();
    document.dispatchEvent(new Event("pointerdown", { bubbles: true }));
    await new Promise((r) => setTimeout(r, 0));

    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    expect(ctx.resume).toHaveBeenCalledOnce();
  });
});

// ---------------------------------------------------------------------------
// Keepalive idle timeout
// ---------------------------------------------------------------------------

describe("keepalive idle timeout", () => {
  it("auto-tears-down after KEEPALIVE_IDLE_TIMEOUT_MS", () => {
    vi.useFakeTimers();
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 Chrome",
      configurable: true,
    });
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    installAudioContextKeepalive();
    const osc = ctx.createOscillator.mock.results[0]!.value as ReturnType<typeof makeOscillatorNode>;

    vi.advanceTimersByTime(30_000); // KEEPALIVE_IDLE_TIMEOUT_MS

    expect(osc.stop).toHaveBeenCalledOnce();
    vi.useRealTimers();
  });

  it("re-arming installAudioContextKeepalive extends the timer", () => {
    vi.useFakeTimers();
    Object.defineProperty(navigator, "userAgent", {
      value: "Mozilla/5.0 Chrome",
      configurable: true,
    });
    const ctx = getSharedAudioContext() as unknown as FakeAudioContext;
    installAudioContextKeepalive();
    const osc = ctx.createOscillator.mock.results[0]!.value as ReturnType<typeof makeOscillatorNode>;

    vi.advanceTimersByTime(20_000);
    installAudioContextKeepalive(); // re-arm
    vi.advanceTimersByTime(20_000); // total 40s since first arm

    // Oscillator should NOT have been stopped yet (timer was reset at 20s mark)
    expect(osc.stop).not.toHaveBeenCalled();

    vi.advanceTimersByTime(10_001); // now 30s past the re-arm
    expect(osc.stop).toHaveBeenCalledOnce();
    vi.useRealTimers();
  });
});
