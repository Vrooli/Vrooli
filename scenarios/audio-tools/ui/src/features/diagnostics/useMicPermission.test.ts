import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, act, cleanup } from "@testing-library/react";

import { useMicPermission } from "./useMicPermission";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

/** Helper that installs a mock permissions API returning the given initial state. */
function mockPermissions(initialState: "granted" | "denied" | "prompt") {
  let onchangeCallback: (() => void) | null = null;
  const status = {
    state: initialState,
    set onchange(cb: (() => void) | null) {
      onchangeCallback = cb;
    },
    get onchange() { return onchangeCallback; },
  };
  const query = vi.fn().mockResolvedValue(status);
  Object.defineProperty(navigator, "permissions", {
    value: { query },
    configurable: true,
    writable: true,
  });
  return { status, triggerChange: (newState: "granted" | "denied" | "prompt") => {
    status.state = newState;
    onchangeCallback?.();
  }};
}

describe("useMicPermission", () => {
  it("starts as 'unknown' before the permissions query resolves", () => {
    // Return a never-resolving promise so the initial state is captured.
    Object.defineProperty(navigator, "permissions", {
      value: { query: () => new Promise(() => {}) },
      configurable: true,
      writable: true,
    });
    const { result } = renderHook(() => useMicPermission());
    expect(result.current).toBe("unknown");
  });

  it("resolves to 'granted' once the permissions API resolves", async () => {
    mockPermissions("granted");
    const { result } = renderHook(() => useMicPermission());
    // Wait for the async query to settle
    await act(async () => {
      await Promise.resolve();
    });
    expect(result.current).toBe("granted");
  });

  it("resolves to 'denied' when permission is denied", async () => {
    mockPermissions("denied");
    const { result } = renderHook(() => useMicPermission());
    await act(async () => { await Promise.resolve(); });
    expect(result.current).toBe("denied");
  });

  it("resolves to 'prompt' when permission is pending", async () => {
    mockPermissions("prompt");
    const { result } = renderHook(() => useMicPermission());
    await act(async () => { await Promise.resolve(); });
    expect(result.current).toBe("prompt");
  });

  it("updates state when the permission status changes via onchange", async () => {
    const { triggerChange } = mockPermissions("prompt");
    const { result } = renderHook(() => useMicPermission());
    await act(async () => { await Promise.resolve(); });
    expect(result.current).toBe("prompt");

    // Simulate permission change
    await act(async () => {
      triggerChange("granted");
      await Promise.resolve();
    });
    expect(result.current).toBe("granted");
  });

  it("stays 'unknown' when navigator.permissions is not available", () => {
    // Remove permissions API (older WebViews)
    Object.defineProperty(navigator, "permissions", {
      value: undefined,
      configurable: true,
      writable: true,
    });
    const { result } = renderHook(() => useMicPermission());
    expect(result.current).toBe("unknown");
  });

  it("stays 'unknown' when permissions.query is not a function", () => {
    Object.defineProperty(navigator, "permissions", {
      value: { query: null },
      configurable: true,
      writable: true,
    });
    const { result } = renderHook(() => useMicPermission());
    expect(result.current).toBe("unknown");
  });

  it("ignores the resolved state if the component unmounted before the query settled", async () => {
    // Use a promise we control
    let resolveQuery!: (v: unknown) => void;
    const queryPromise = new Promise((res) => { resolveQuery = res; });
    Object.defineProperty(navigator, "permissions", {
      value: { query: () => queryPromise },
      configurable: true,
      writable: true,
    });

    const { result, unmount } = renderHook(() => useMicPermission());
    // Unmount before the promise resolves
    unmount();
    // Now resolve — should not cause any state update
    await act(async () => {
      resolveQuery({ state: "granted", onchange: null });
      await Promise.resolve();
    });
    // State was never set to "granted" because cancelled=true
    expect(result.current).toBe("unknown");
  });

  it("silently catches errors from the permissions query", async () => {
    Object.defineProperty(navigator, "permissions", {
      value: { query: vi.fn().mockRejectedValue(new Error("unsupported descriptor")) },
      configurable: true,
      writable: true,
    });
    const { result } = renderHook(() => useMicPermission());
    await act(async () => { await Promise.resolve(); });
    // Should remain "unknown" and not throw
    expect(result.current).toBe("unknown");
  });
});
