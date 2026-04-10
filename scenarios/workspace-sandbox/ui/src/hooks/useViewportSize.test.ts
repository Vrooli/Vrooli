import { describe, it, expect, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useViewportSize, useIsMobile } from "./useViewportSize";

describe("useViewportSize", () => {
  const originalInnerWidth = window.innerWidth;
  const originalInnerHeight = window.innerHeight;

  function setViewport(width: number, height = 768) {
    Object.defineProperty(window, "innerWidth", { value: width, writable: true, configurable: true });
    Object.defineProperty(window, "innerHeight", { value: height, writable: true, configurable: true });
  }

  afterEach(() => {
    setViewport(originalInnerWidth, originalInnerHeight);
  });

  it("returns desktop breakpoint when width >= 1024", () => {
    setViewport(1024);
    const { result } = renderHook(() => useViewportSize());
    expect(result.current.breakpoint).toBe("desktop");
    expect(result.current.isMobile).toBe(false);
    expect(result.current.isTablet).toBe(false);
    expect(result.current.isDesktop).toBe(true);
  });

  it("returns tablet breakpoint when width is 640-1023", () => {
    setViewport(800);
    const { result } = renderHook(() => useViewportSize());
    expect(result.current.breakpoint).toBe("tablet");
    expect(result.current.isTablet).toBe(true);
  });

  it("returns mobile breakpoint when width < 640", () => {
    setViewport(375);
    const { result } = renderHook(() => useViewportSize());
    expect(result.current.breakpoint).toBe("mobile");
    expect(result.current.isMobile).toBe(true);
  });

  it("responds to resize events", () => {
    setViewport(1024);
    const { result } = renderHook(() => useViewportSize());
    expect(result.current.breakpoint).toBe("desktop");

    act(() => {
      setViewport(375);
      window.dispatchEvent(new Event("resize"));
    });

    expect(result.current.breakpoint).toBe("mobile");
    expect(result.current.isMobile).toBe(true);
  });

  it("includes width and height values", () => {
    setViewport(1200, 900);
    const { result } = renderHook(() => useViewportSize());
    expect(result.current.width).toBe(1200);
    expect(result.current.height).toBe(900);
  });
});

describe("useIsMobile", () => {
  const originalInnerWidth = window.innerWidth;

  function setWidth(width: number) {
    Object.defineProperty(window, "innerWidth", { value: width, writable: true, configurable: true });
  }

  afterEach(() => {
    setWidth(originalInnerWidth);
  });

  it("returns true when width < 640", () => {
    setWidth(375);
    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(true);
  });

  it("returns false when width >= 640", () => {
    setWidth(1024);
    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(false);
  });

  it("responds to resize events", () => {
    setWidth(1024);
    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(false);

    act(() => {
      setWidth(375);
      window.dispatchEvent(new Event("resize"));
    });

    expect(result.current).toBe(true);
  });
});
