import { act, renderHook } from "@testing-library/react";
import { useRef } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { useResizablePanel } from "./useResizablePanel";

const STORAGE_KEY = "flow-verifier.sidebar.width.test";

function setup(initialWidth = 320) {
  return renderHook(() => {
    const containerRef = useRef<HTMLElement>(document.body);
    const targetRef = useRef<HTMLElement>(document.createElement("aside"));
    const hook = useResizablePanel({
      containerRef,
      targetRef,
      minSize: 260,
      maxSize: 480,
      defaultSize: initialWidth,
      storageKey: STORAGE_KEY,
    });
    return { hook, containerRef, targetRef };
  });
}

describe("useResizablePanel", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });
  afterEach(() => {
    window.localStorage.clear();
  });

  it("starts at defaultSize when nothing is persisted", () => {
    const { result } = setup(310);
    expect(result.current.hook.size).toBe(310);
    expect(result.current.hook.isResizing).toBe(false);
  });

  it("clamps a persisted value to [min,max]", () => {
    window.localStorage.setItem(STORAGE_KEY, "9999");
    const { result } = setup(320);
    expect(result.current.hook.size).toBe(480);
  });

  it("exposes ARIA separator props", () => {
    const { result } = setup(320);
    const p = result.current.hook.resizeHandleProps;
    expect(p.role).toBe("separator");
    expect(p["aria-orientation"]).toBe("vertical");
    expect(p["aria-valuenow"]).toBe(320);
    expect(p["aria-valuemin"]).toBe(260);
    expect(p["aria-valuemax"]).toBe(480);
  });

  it("enters resizing state on pointerdown", () => {
    const { result } = setup(320);
    const fakeEvent = {
      button: 0,
      preventDefault: () => undefined,
    } as unknown as React.PointerEvent<HTMLDivElement>;
    act(() => result.current.hook.resizeHandleProps.onPointerDown(fakeEvent));
    expect(result.current.hook.isResizing).toBe(true);
  });
});
