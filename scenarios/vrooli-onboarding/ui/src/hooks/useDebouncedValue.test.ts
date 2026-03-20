// [REQ:REQ-P2-002] Debounced search input for glossary
import { renderHook, act } from "@testing-library/react";
import { vi } from "vitest";
import { useDebouncedValue } from "./useDebouncedValue";

describe("useDebouncedValue", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns initial value immediately with no pending state", () => {
    const { result } = renderHook(() => useDebouncedValue("hello", 300));
    expect(result.current.debounced).toBe("hello");
    expect(result.current.isPending).toBe(false);
  });

  it("delays value update by the specified delay", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      { initialProps: { value: "a", delay: 300 } },
    );

    rerender({ value: "ab", delay: 300 });
    expect(result.current.debounced).toBe("a");
    expect(result.current.isPending).toBe(true);

    act(() => vi.advanceTimersByTime(299));
    expect(result.current.debounced).toBe("a");

    act(() => vi.advanceTimersByTime(1));
    expect(result.current.debounced).toBe("ab");
    expect(result.current.isPending).toBe(false);
  });

  it("resets timer when value changes before delay expires", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      { initialProps: { value: "a", delay: 300 } },
    );

    rerender({ value: "ab", delay: 300 });
    act(() => vi.advanceTimersByTime(200));

    rerender({ value: "abc", delay: 300 });
    act(() => vi.advanceTimersByTime(200));
    // Still waiting — total 400ms but only 200ms since last change
    expect(result.current.debounced).toBe("a");

    act(() => vi.advanceTimersByTime(100));
    expect(result.current.debounced).toBe("abc");
  });

  it("is not pending when reverting to the debounced value", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      { initialProps: { value: "a", delay: 300 } },
    );

    rerender({ value: "ab", delay: 300 });
    expect(result.current.isPending).toBe(true);

    // Revert to original value before timer fires
    rerender({ value: "a", delay: 300 });
    expect(result.current.isPending).toBe(false);
    expect(result.current.debounced).toBe("a");
  });

  it("handles empty string values", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      { initialProps: { value: "search", delay: 300 } },
    );

    rerender({ value: "", delay: 300 });
    act(() => vi.advanceTimersByTime(300));
    expect(result.current.debounced).toBe("");
    expect(result.current.isPending).toBe(false);
  });
});
