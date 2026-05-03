import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { useResizablePanel, type UseResizablePanelOptions } from "./useResizablePanel";
import type { PointerEvent } from "react";

/** jsdom doesn't provide PointerEvent — polyfill with MouseEvent + pointer fields. */
class PointerEventPolyfill extends MouseEvent {
  readonly pointerId: number;
  constructor(type: string, init?: PointerEventInit & MouseEventInit) {
    super(type, init);
    this.pointerId = init?.pointerId ?? 0;
  }
}
if (typeof globalThis.PointerEvent === "undefined") {
  (globalThis as Record<string, unknown>).PointerEvent = PointerEventPolyfill;
}

function makeContainerRef(width = 1000) {
  const el = {
    getBoundingClientRect: () => ({ left: 0, top: 0, width, height: 600, right: width, bottom: 600, x: 0, y: 0, toJSON: () => ({}) }),
  } as HTMLDivElement;
  return { current: el };
}

function makeTargetRef() {
  const style: Record<string, string> = {};
  const el = { style } as unknown as HTMLDivElement;
  return { current: el, style };
}

function defaultOptions(overrides: Partial<UseResizablePanelOptions> = {}): UseResizablePanelOptions {
  return {
    containerRef: makeContainerRef(),
    targetRef: makeTargetRef(),
    minSize: 200,
    maxSize: 500,
    defaultSize: 320,
    adjacentMinSize: 300,
    handleWidth: 8,
    ...overrides,
  };
}

describe("useResizablePanel", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.spyOn(document.body.style, "cursor", "set");
    vi.spyOn(document.body.style, "userSelect", "set");
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns the default size initially", () => {
    const { result } = renderHook(() => useResizablePanel(defaultOptions()));
    expect(result.current.size).toBe(320);
    expect(result.current.isResizing).toBe(false);
  });

  it("restores persisted size and persists final size on pointerup", async () => {
    window.localStorage.setItem("test-panel-width", "410");
    const { result } = renderHook(() => useResizablePanel(defaultOptions({ storageKey: "test-panel-width" })));

    expect(result.current.size).toBe(410);

    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 0,
        preventDefault: vi.fn(),
      } as unknown as PointerEvent<HTMLDivElement>);
    });

    act(() => {
      window.dispatchEvent(new PointerEvent("pointermove", { clientX: 360 }));
      window.dispatchEvent(new PointerEvent("pointerup"));
    });

    expect(result.current.size).toBe(360);
    await waitFor(() => {
      expect(window.localStorage.getItem("test-panel-width")).toBe("360");
    });
  });

  it("provides correct aria attributes on resizeHandleProps", () => {
    const { result } = renderHook(() => useResizablePanel(defaultOptions()));
    const props = result.current.resizeHandleProps;
    expect(props.role).toBe("separator");
    expect(props["aria-orientation"]).toBe("vertical");
    expect(props["aria-valuenow"]).toBe(320);
    expect(props["aria-valuemin"]).toBe(200);
    expect(props["aria-valuemax"]).toBe(500);
  });

  it("ignores non-primary button presses", () => {
    const { result } = renderHook(() => useResizablePanel(defaultOptions()));
    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 2,
        preventDefault: vi.fn(),
      } as unknown as PointerEvent<HTMLDivElement>);
    });
    expect(result.current.isResizing).toBe(false);
  });

  it("starts resizing on primary button pointerdown", () => {
    const { result } = renderHook(() => useResizablePanel(defaultOptions()));
    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 0,
        preventDefault: vi.fn(),
      } as unknown as PointerEvent<HTMLDivElement>);
    });
    expect(result.current.isResizing).toBe(true);
  });

  it("writes target.style.width during pointermove and commits size on pointerup", () => {
    const target = makeTargetRef();
    const { result } = renderHook(() => useResizablePanel(defaultOptions({ targetRef: target })));

    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 0,
        preventDefault: vi.fn(),
      } as unknown as PointerEvent<HTMLDivElement>);
    });

    // pointermove writes the DOM directly; React state stays put.
    act(() => {
      window.dispatchEvent(new PointerEvent("pointermove", { clientX: 400 }));
    });
    expect(target.style.width).toBe("400px");
    expect(result.current.size).toBe(320);

    act(() => {
      window.dispatchEvent(new PointerEvent("pointermove", { clientX: 450 }));
    });
    expect(target.style.width).toBe("450px");
    expect(result.current.size).toBe(320);

    // pointerup commits the final size.
    act(() => {
      window.dispatchEvent(new PointerEvent("pointerup"));
    });
    expect(result.current.size).toBe(450);
    expect(result.current.isResizing).toBe(false);
  });

  it("clamps size to minSize", () => {
    const target = makeTargetRef();
    const { result } = renderHook(() => useResizablePanel(defaultOptions({ targetRef: target })));

    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 0,
        preventDefault: vi.fn(),
      } as unknown as PointerEvent<HTMLDivElement>);
    });

    act(() => {
      window.dispatchEvent(new PointerEvent("pointermove", { clientX: 50 }));
      window.dispatchEvent(new PointerEvent("pointerup"));
    });
    expect(result.current.size).toBe(200);
    expect(target.style.width).toBe("200px");
  });

  it("clamps size to effective max (container - adjacent - handle)", () => {
    const target = makeTargetRef();
    const { result } = renderHook(() => useResizablePanel(defaultOptions({ targetRef: target })));

    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 0,
        preventDefault: vi.fn(),
      } as unknown as PointerEvent<HTMLDivElement>);
    });

    act(() => {
      window.dispatchEvent(new PointerEvent("pointermove", { clientX: 800 }));
      window.dispatchEvent(new PointerEvent("pointerup"));
    });
    expect(result.current.size).toBe(500); // maxSize
  });

  it("clamps to effective max when container is narrow", () => {
    const target = makeTargetRef();
    const opts = defaultOptions({ containerRef: makeContainerRef(600), targetRef: target });
    const { result } = renderHook(() => useResizablePanel(opts));

    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 0,
        preventDefault: vi.fn(),
      } as unknown as PointerEvent<HTMLDivElement>);
    });

    act(() => {
      window.dispatchEvent(new PointerEvent("pointermove", { clientX: 450 }));
      window.dispatchEvent(new PointerEvent("pointerup"));
    });
    expect(result.current.size).toBe(292); // effective max = 600 - 300 - 8
  });

  it("sets col-resize cursor during drag and restores on release", () => {
    document.body.style.cursor = "default";
    document.body.style.userSelect = "auto";

    const { result } = renderHook(() => useResizablePanel(defaultOptions()));

    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 0,
        preventDefault: vi.fn(),
      } as unknown as PointerEvent<HTMLDivElement>);
    });
    expect(document.body.style.cursor).toBe("col-resize");
    expect(document.body.style.userSelect).toBe("none");

    act(() => {
      window.dispatchEvent(new PointerEvent("pointerup"));
    });
    expect(document.body.style.cursor).toBe("default");
    expect(document.body.style.userSelect).toBe("auto");
  });
});
