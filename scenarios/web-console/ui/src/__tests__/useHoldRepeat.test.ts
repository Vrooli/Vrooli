import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useHoldRepeat } from "../hooks/useHoldRepeat";

describe("useHoldRepeat", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const makePointerEvent = (overrides?: Partial<React.PointerEvent>) => ({
    pointerType: "touch",
    button: 0,
    preventDefault: vi.fn(),
    ...overrides,
  } as unknown as React.PointerEvent);

  it("fires once immediately on pointerdown", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() => useHoldRepeat({ onFire }));

    act(() => result.current.onPointerDown(makePointerEvent()));

    expect(onFire).toHaveBeenCalledTimes(1);
  });

  it("calls preventDefault on pointerdown to preserve focus", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() => useHoldRepeat({ onFire }));

    const event = makePointerEvent();
    act(() => result.current.onPointerDown(event));

    expect(event.preventDefault).toHaveBeenCalled();
  });

  it("does not repeat on a quick tap (pointerup before initial delay)", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() => useHoldRepeat({ onFire }));

    act(() => result.current.onPointerDown(makePointerEvent()));
    act(() => {
      vi.advanceTimersByTime(200);
    });
    act(() => result.current.onPointerUp());
    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(onFire).toHaveBeenCalledTimes(1);
  });

  it("begins repeating after initialDelayMs and continues at repeatIntervalMs", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() =>
      useHoldRepeat({ onFire, initialDelayMs: 400, repeatIntervalMs: 40 }),
    );

    act(() => result.current.onPointerDown(makePointerEvent()));
    expect(onFire).toHaveBeenCalledTimes(1);

    // At the initial delay boundary the setTimeout has fired (arming the
    // interval) but the first interval tick hasn't landed yet.
    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(onFire).toHaveBeenCalledTimes(1);

    // One repeat interval later — first repeat fires.
    act(() => {
      vi.advanceTimersByTime(40);
    });
    expect(onFire).toHaveBeenCalledTimes(2);

    // Five more intervals — five more fires.
    act(() => {
      vi.advanceTimersByTime(40 * 5);
    });
    expect(onFire).toHaveBeenCalledTimes(7);
  });

  it("stops repeating on pointerup", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() =>
      useHoldRepeat({ onFire, initialDelayMs: 400, repeatIntervalMs: 40 }),
    );

    act(() => result.current.onPointerDown(makePointerEvent()));
    act(() => {
      vi.advanceTimersByTime(500); // 1 initial + 1 repeat (at 440)
    });
    const countAtRelease = onFire.mock.calls.length;

    act(() => result.current.onPointerUp());
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(onFire).toHaveBeenCalledTimes(countAtRelease);
  });

  it("stops repeating on pointercancel", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() =>
      useHoldRepeat({ onFire, initialDelayMs: 400, repeatIntervalMs: 40 }),
    );

    act(() => result.current.onPointerDown(makePointerEvent()));
    act(() => {
      vi.advanceTimersByTime(500);
    });
    const countAtCancel = onFire.mock.calls.length;

    act(() => result.current.onPointerCancel());
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(onFire).toHaveBeenCalledTimes(countAtCancel);
  });

  it("stops repeating on pointerleave (finger dragged off button)", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() =>
      useHoldRepeat({ onFire, initialDelayMs: 400, repeatIntervalMs: 40 }),
    );

    act(() => result.current.onPointerDown(makePointerEvent()));
    act(() => {
      vi.advanceTimersByTime(500);
    });
    const countAtLeave = onFire.mock.calls.length;

    act(() => result.current.onPointerLeave());
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(onFire).toHaveBeenCalledTimes(countAtLeave);
  });

  it("ignores pointerdown from non-primary mouse buttons", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() => useHoldRepeat({ onFire }));

    act(() =>
      result.current.onPointerDown(
        makePointerEvent({ pointerType: "mouse", button: 2 }),
      ),
    );
    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(onFire).not.toHaveBeenCalled();
  });

  it("uses the latest onFire without re-subscribing timers", () => {
    const first = vi.fn();
    const second = vi.fn();
    const { result, rerender } = renderHook(
      ({ fn }: { fn: () => void }) => useHoldRepeat({ onFire: fn }),
      { initialProps: { fn: first } },
    );

    // Start a hold with the first onFire.
    act(() => result.current.onPointerDown(makePointerEvent()));
    expect(first).toHaveBeenCalledTimes(1);

    // Swap onFire mid-hold.
    rerender({ fn: second });

    // Let the initial delay elapse and one repeat fire.
    act(() => {
      vi.advanceTimersByTime(440);
    });

    // The repeat should route to the new onFire, not the stale one.
    expect(second).toHaveBeenCalledTimes(1);
    expect(first).toHaveBeenCalledTimes(1);
  });

  it("clears timers on unmount so no stray repeats fire", () => {
    const onFire = vi.fn();
    const { result, unmount } = renderHook(() => useHoldRepeat({ onFire }));

    act(() => result.current.onPointerDown(makePointerEvent()));
    expect(onFire).toHaveBeenCalledTimes(1);

    unmount();
    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(onFire).toHaveBeenCalledTimes(1);
  });

  it("a second pointerdown restarts the timing instead of stacking", () => {
    const onFire = vi.fn();
    const { result } = renderHook(() =>
      useHoldRepeat({ onFire, initialDelayMs: 400, repeatIntervalMs: 40 }),
    );

    act(() => result.current.onPointerDown(makePointerEvent()));
    act(() => {
      vi.advanceTimersByTime(300); // still within initial delay
    });
    // Second press while first is pending — should cancel and restart.
    act(() => result.current.onPointerDown(makePointerEvent()));

    // Two initial fires so far (one per pointerdown), no repeats.
    expect(onFire).toHaveBeenCalledTimes(2);

    // Advance to the first repeat from the *second* press (initial delay
    // plus one interval tick).
    act(() => {
      vi.advanceTimersByTime(440);
    });
    expect(onFire).toHaveBeenCalledTimes(3);
  });
});
