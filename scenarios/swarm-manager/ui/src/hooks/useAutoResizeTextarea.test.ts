import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { useAutoResizeTextarea } from "./useAutoResizeTextarea";

function makeTextareaRef(scrollHeight = 100) {
  const el = {
    scrollHeight,
    style: { height: "" },
  } as unknown as HTMLTextAreaElement;
  return { current: el };
}

describe("useAutoResizeTextarea", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("sets height to scrollHeight when under max", () => {
    const ref = makeTextareaRef(80);
    renderHook(() => useAutoResizeTextarea(ref, "hello"));

    expect(ref.current!.style.height).toBe("80px");
  });

  it("clamps height to maxHeight when scrollHeight exceeds it", () => {
    const ref = makeTextareaRef(500);
    renderHook(() => useAutoResizeTextarea(ref, "lots of text", { maxHeight: 200 }));

    expect(ref.current!.style.height).toBe("200px");
  });

  it("uses default maxHeight of 200 when not specified", () => {
    const ref = makeTextareaRef(300);
    renderHook(() => useAutoResizeTextarea(ref, "text"));

    expect(ref.current!.style.height).toBe("200px");
  });

  it("re-measures when value changes", () => {
    const ref = makeTextareaRef(50);
    const { rerender } = renderHook(
      ({ value }) => useAutoResizeTextarea(ref, value),
      { initialProps: { value: "short" } },
    );

    expect(ref.current!.style.height).toBe("50px");

    // Simulate the textarea growing
    (ref.current as { scrollHeight: number }).scrollHeight = 120;
    rerender({ value: "much longer text that wraps" });

    expect(ref.current!.style.height).toBe("120px");
  });

  it("resets height to auto before measuring", () => {
    const ref = makeTextareaRef(80);
    const heightValues: string[] = [];

    // Intercept style.height assignments
    const el = ref.current!;
    let realHeight = "";
    Object.defineProperty(el.style, "height", {
      get: () => realHeight,
      set: (v: string) => {
        heightValues.push(v);
        realHeight = v;
      },
    });

    renderHook(() => useAutoResizeTextarea(ref, "test"));

    // First call should be "auto", second should be the measured height
    expect(heightValues[0]).toBe("auto");
    expect(heightValues[1]).toBe("80px");
  });

  it("does nothing when ref.current is null", () => {
    const ref = { current: null };
    // Should not throw
    expect(() => {
      renderHook(() => useAutoResizeTextarea(ref, "text"));
    }).not.toThrow();
  });
});
