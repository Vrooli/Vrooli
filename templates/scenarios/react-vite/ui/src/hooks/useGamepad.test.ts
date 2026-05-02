/**
 * Unit tests for useGamepad.
 *
 * The hook wraps `GamepadInputManager` from `@vrooli/iframe-bridge/spatial`.
 * Tests pin the three contracts the hook exists to provide:
 *
 *   1. mount instantiates the manager and starts it
 *   2. unmount disposes the manager
 *   3. the latest `onAction` callback is invoked even after rerender
 *      (the `callbackRef` indirection — without it, every action would
 *      go to the *first* callback the component ever rendered with)
 *
 * Vitest hoists `vi.mock(path, factory)` above all imports. We declare
 * the shared mock-state holder via `vi.hoisted(...)` so the factory
 * (which runs at import time) and the test bodies (which run later)
 * touch the same object. `beforeEach` resets the slots so tests stay
 * isolated even though they share the `instance` reference.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";

const mockState = vi.hoisted(() => {
  const instance = {
    start: vi.fn(),
    dispose: vi.fn(),
    onAction: undefined as ((a: unknown) => void) | undefined,
  };
  const Ctor = vi
    .fn()
    .mockImplementation((opts: { onAction?: (a: unknown) => void }) => {
      instance.onAction = opts.onAction;
      return instance;
    });
  return { instance, Ctor };
});

vi.mock("@vrooli/iframe-bridge/spatial", () => ({
  GamepadInputManager: mockState.Ctor,
}));

import { useGamepad } from "./useGamepad";

describe("useGamepad", () => {
  beforeEach(() => {
    mockState.instance.start.mockReset();
    mockState.instance.dispose.mockReset();
    mockState.instance.onAction = undefined;
    mockState.Ctor.mockClear();
  });

  afterEach(() => {
    // renderHook's auto-cleanup unmounts the hook, which calls dispose.
    // Tests that read dispose call counts should do so *before* this
    // hook fires; this afterEach resets state for the next test.
    mockState.instance.start.mockReset();
    mockState.instance.dispose.mockReset();
    mockState.Ctor.mockClear();
  });

  it("instantiates GamepadInputManager and calls start on mount", () => {
    const handler = vi.fn();
    renderHook(() => useGamepad(handler));

    expect(mockState.Ctor).toHaveBeenCalledTimes(1);
    expect(mockState.instance.start).toHaveBeenCalledTimes(1);
  });

  it("calls dispose on unmount", () => {
    const handler = vi.fn();
    const { unmount } = renderHook(() => useGamepad(handler));

    expect(mockState.instance.dispose).not.toHaveBeenCalled();
    unmount();
    expect(mockState.instance.dispose).toHaveBeenCalledTimes(1);
  });

  it("invokes the latest onAction callback after rerender (callbackRef indirection)", () => {
    const first = vi.fn();
    const second = vi.fn();

    const { rerender } = renderHook(({ cb }: { cb: () => void }) => useGamepad(cb), {
      initialProps: { cb: first },
    });

    rerender({ cb: second });

    expect(mockState.instance.onAction).toBeDefined();
    mockState.instance.onAction!({ kind: "action" });

    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
  });
});
