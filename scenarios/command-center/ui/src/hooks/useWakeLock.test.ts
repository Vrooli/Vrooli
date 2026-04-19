import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useWakeLock } from "./useWakeLock";

interface FakeSentinel {
  release: () => Promise<void>;
  addEventListener: (event: string, handler: () => void) => void;
  triggerRelease: () => void;
}

function createFakeSentinel(): FakeSentinel {
  const listeners: (() => void)[] = [];
  const sentinel: FakeSentinel = {
    release: vi.fn(() => {
      sentinel.triggerRelease();
      return Promise.resolve();
    }),
    addEventListener: (event: string, handler: () => void) => {
      if (event === "release") listeners.push(handler);
    },
    triggerRelease: () => {
      for (const l of listeners) l();
    },
  };
  return sentinel;
}

interface MockNavigator {
  wakeLock: {
    request: ReturnType<typeof vi.fn>;
  };
}

describe("useWakeLock", () => {
  const originalNavigator = globalThis.navigator;
  let mockNav: MockNavigator;

  beforeEach(() => {
    mockNav = {
      wakeLock: {
        request: vi.fn(() => Promise.resolve(createFakeSentinel() as unknown as WakeLockSentinel)),
      },
    };
    Object.defineProperty(globalThis, "navigator", {
      configurable: true,
      value: mockNav,
    });
  });

  afterEach(() => {
    Object.defineProperty(globalThis, "navigator", {
      configurable: true,
      value: originalNavigator,
    });
    vi.clearAllMocks();
  });

  it("starts inactive and does not auto-request", () => {
    const { result } = renderHook(() => useWakeLock());
    expect(result.current.isActive).toBe(false);
    expect(mockNav.wakeLock.request).not.toHaveBeenCalled();
  });

  it("transitions to active after request() and back to inactive after release()", async () => {
    const { result } = renderHook(() => useWakeLock());

    await act(async () => {
      await result.current.request();
    });
    expect(result.current.isActive).toBe(true);

    act(() => {
      result.current.release();
    });
    expect(result.current.isActive).toBe(false);
  });

  it("flips back to inactive when the sentinel fires its release event", async () => {
    const fake = createFakeSentinel();
    mockNav.wakeLock.request = vi.fn(() => Promise.resolve(fake as unknown as WakeLockSentinel));

    const { result } = renderHook(() => useWakeLock());
    await act(async () => {
      await result.current.request();
    });
    expect(result.current.isActive).toBe(true);

    act(() => {
      fake.triggerRelease();
    });
    expect(result.current.isActive).toBe(false);
  });
});
