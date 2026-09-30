import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useComponentInspector } from "./useComponentInspector";

function makeRef(): {
  ref: React.RefObject<HTMLIFrameElement | null>;
  posts: Array<{ payload: unknown; targetOrigin: string }>;
} {
  const posts: Array<{ payload: unknown; targetOrigin: string }> = [];
  const fakeWindow = {
    postMessage: (payload: unknown, targetOrigin: string) => {
      posts.push({ payload, targetOrigin });
    },
  } as unknown as Window;
  const ref = {
    current: { contentWindow: fakeWindow } as unknown as HTMLIFrameElement,
  };
  return { ref, posts };
}

function postFromHarness(source: Window, payload: Record<string, unknown>) {
  window.dispatchEvent(new MessageEvent("message", { data: payload, source }));
}

describe("useComponentInspector", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("startInspect posts the START command on the iframe contentWindow", () => {
    const { ref, posts } = makeRef();
    const { result } = renderHook(() => useComponentInspector(ref));
    act(() => {
      result.current.startInspect();
    });
    expect(posts).toHaveLength(1);
    expect(posts[0]?.payload).toEqual({ v: 1, t: "INSPECT", cmd: "START" });
  });

  it("reflects INSPECT_STATE active=true messages", () => {
    const { ref } = makeRef();
    const { result } = renderHook(() => useComponentInspector(ref));
    act(() => {
      postFromHarness(ref.current!.contentWindow!, {
        v: 1,
        t: "INSPECT_STATE",
        active: true,
        reason: "start",
      });
    });
    expect(result.current.active).toBe(true);
    expect(result.current.lastReason).toBe("start");
  });

  it("stores INSPECT_HOVER payload as `hover` and exposes via `selected`", () => {
    const { ref } = makeRef();
    const { result } = renderHook(() => useComponentInspector(ref));
    const payload = {
      meta: {
        tag: "button",
        id: "",
        classes: ["primary"],
        selector: "button.primary",
        label: "",
        ariaLabel: "",
        ariaDescription: "",
        title: "",
        role: "",
        text: "Click me",
      },
      rect: { x: 1, y: 2, width: 100, height: 30 },
      documentRect: { x: 1, y: 2, width: 100, height: 30 },
      ancestors: [],
      selectedAncestorIndex: 0,
    };
    act(() => {
      postFromHarness(ref.current!.contentWindow!, { v: 1, t: "INSPECT_HOVER", payload });
    });
    expect(result.current.hover?.meta.selector).toBe("button.primary");
    expect(result.current.selected?.meta.text).toBe("Click me");
  });

  it("INSPECT_RESULT clears hover, sets result, flips active false", () => {
    const { ref } = makeRef();
    const { result } = renderHook(() => useComponentInspector(ref));
    act(() => {
      postFromHarness(ref.current!.contentWindow!, { v: 1, t: "INSPECT_STATE", active: true });
    });
    const payload = {
      meta: {
        tag: "h1",
        id: "title",
        classes: [],
        selector: "#title",
        label: "",
        ariaLabel: "",
        ariaDescription: "",
        title: "",
        role: "",
        text: "Hello",
      },
      rect: null,
      documentRect: null,
      ancestors: [],
      selectedAncestorIndex: 0,
      method: "pointer",
    };
    act(() => {
      postFromHarness(ref.current!.contentWindow!, { v: 1, t: "INSPECT_RESULT", payload });
    });
    expect(result.current.active).toBe(false);
    expect(result.current.hover).toBeNull();
    expect(result.current.result?.meta.selector).toBe("#title");
    expect(result.current.lastReason).toBe("complete");
  });

  it("ignores messages without v:1 or unknown t", () => {
    const { ref } = makeRef();
    const { result } = renderHook(() => useComponentInspector(ref));
    act(() => {
      postFromHarness(ref.current!.contentWindow!, {
        t: "INSPECT_HOVER",
        payload: { meta: { tag: "x" } },
      });
      postFromHarness(ref.current!.contentWindow!, { v: 1, t: "RANDOM" });
    });
    expect(result.current.hover).toBeNull();
    expect(result.current.active).toBe(false);
  });

  it("stopInspect is a no-op when not active", () => {
    const { ref, posts } = makeRef();
    const { result } = renderHook(() => useComponentInspector(ref));
    act(() => {
      result.current.stopInspect();
    });
    expect(posts).toHaveLength(0);
  });
});
