import { fireEvent, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useFirstUserGesture } from "./useFirstUserGesture";

describe("useFirstUserGesture", () => {
  it("fires once for the first user activation", () => {
    const onGesture = vi.fn();
    renderHook(() => useFirstUserGesture(onGesture));

    fireEvent.pointerDown(window);
    fireEvent.keyDown(window);

    expect(onGesture).toHaveBeenCalledOnce();
  });
});
