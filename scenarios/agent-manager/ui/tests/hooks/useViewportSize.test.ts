import assert from "node:assert/strict";
import { act, renderHook } from "@testing-library/react";
import { afterEach, test } from "vitest";
import { useIsMobile, useViewportSize } from "../../src/hooks/useViewportSize.js";

function setViewport(width: number, height = 800) {
  Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
  Object.defineProperty(window, "innerHeight", { configurable: true, value: height });
}

afterEach(() => setViewport(1024));

test("viewport hooks classify responsive breakpoints and update on resize and orientation", () => {
  setViewport(500, 700);
  const viewport = renderHook(() => useViewportSize());
  const mobile = renderHook(() => useIsMobile());
  assert.equal(viewport.result.current.breakpoint, "mobile");
  assert.equal(viewport.result.current.isMobile, true);
  assert.equal(mobile.result.current, true);
  setViewport(800, 600);
  act(() => window.dispatchEvent(new Event("resize")));
  assert.equal(viewport.result.current.breakpoint, "tablet");
  assert.equal(viewport.result.current.isTablet, true);
  assert.equal(mobile.result.current, false);
  setViewport(1280, 900);
  act(() => window.dispatchEvent(new Event("orientationchange")));
  assert.equal(viewport.result.current.breakpoint, "desktop");
  assert.equal(viewport.result.current.isDesktop, true);
});
