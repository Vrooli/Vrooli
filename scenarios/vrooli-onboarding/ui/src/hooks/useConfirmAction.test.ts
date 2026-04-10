// [REQ:REQ-P0-004] Two-step confirm flow for destructive actions
import { renderHook, act } from "@testing-library/react";
import { vi } from "vitest";
import { useConfirmAction } from "./useConfirmAction";

describe("useConfirmAction", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts with confirming=false", () => {
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useConfirmAction(onConfirm));
    expect(result.current.confirming).toBe(false);
  });

  it("requestConfirm sets confirming=true", () => {
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useConfirmAction(onConfirm));

    act(() => result.current.requestConfirm());
    expect(result.current.confirming).toBe(true);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("confirm calls onConfirm and resets state", () => {
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useConfirmAction(onConfirm));

    act(() => result.current.requestConfirm());
    act(() => result.current.confirm());
    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(result.current.confirming).toBe(false);
  });

  it("cancel resets confirming without calling onConfirm", () => {
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useConfirmAction(onConfirm));

    act(() => result.current.requestConfirm());
    act(() => result.current.cancel());
    expect(result.current.confirming).toBe(false);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("auto-resets after timeout", () => {
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useConfirmAction(onConfirm, 3000));

    act(() => result.current.requestConfirm());
    expect(result.current.confirming).toBe(true);

    act(() => vi.advanceTimersByTime(2999));
    expect(result.current.confirming).toBe(true);

    act(() => vi.advanceTimersByTime(1));
    expect(result.current.confirming).toBe(false);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("re-requesting resets the timeout", () => {
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useConfirmAction(onConfirm, 3000));

    act(() => result.current.requestConfirm());
    act(() => vi.advanceTimersByTime(2000));

    // Re-request should reset the timer
    act(() => result.current.requestConfirm());
    act(() => vi.advanceTimersByTime(2000));
    expect(result.current.confirming).toBe(true);

    act(() => vi.advanceTimersByTime(1000));
    expect(result.current.confirming).toBe(false);
  });

  it("cleans up timer on unmount", () => {
    const onConfirm = vi.fn();
    const { result, unmount } = renderHook(() => useConfirmAction(onConfirm, 3000));

    act(() => result.current.requestConfirm());
    unmount();

    // Timer should be cleared, no errors from state update after unmount
    act(() => vi.advanceTimersByTime(5000));
  });
});
