import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useTransientHighlight } from "./useTransientHighlight";

describe("useTransientHighlight", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    document.body.innerHTML = "";
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("adds the class on mount and removes it after the duration", () => {
    const target = document.createElement("div");
    target.dataset.target = "x";
    document.body.appendChild(target);
    renderHook(() =>
      useTransientHighlight({
        targetSelector: "[data-target='x']",
        highlightClass: "ring-glow",
        durationMs: 500,
        scrollIntoView: false,
      }),
    );
    expect(target.classList.contains("ring-glow")).toBe(true);
    vi.advanceTimersByTime(500);
    expect(target.classList.contains("ring-glow")).toBe(false);
  });

  it("supports multi-class tokens (whitespace-separated)", () => {
    const target = document.createElement("div");
    target.dataset.target = "y";
    document.body.appendChild(target);
    renderHook(() =>
      useTransientHighlight({
        targetSelector: "[data-target='y']",
        highlightClass: "ring-2 ring-cyan-400/60",
        durationMs: 100,
        scrollIntoView: false,
      }),
    );
    expect(target.classList.contains("ring-2")).toBe(true);
    expect(target.classList.contains("ring-cyan-400/60")).toBe(true);
  });

  it("cleans up the timer + class on unmount", () => {
    const target = document.createElement("div");
    target.dataset.target = "z";
    document.body.appendChild(target);
    const { unmount } = renderHook(() =>
      useTransientHighlight({
        targetSelector: "[data-target='z']",
        highlightClass: "ring-glow",
        durationMs: 500,
        scrollIntoView: false,
      }),
    );
    expect(target.classList.contains("ring-glow")).toBe(true);
    unmount();
    expect(target.classList.contains("ring-glow")).toBe(false);
  });

  it("does nothing when targetSelector is null", () => {
    expect(() =>
      renderHook(() =>
        useTransientHighlight({
          targetSelector: null,
          highlightClass: "ring-glow",
          durationMs: 500,
          scrollIntoView: false,
        }),
      ),
    ).not.toThrow();
  });

  it("does nothing when no element matches the selector", () => {
    expect(() =>
      renderHook(() =>
        useTransientHighlight({
          targetSelector: "[data-target='missing']",
          highlightClass: "ring-glow",
          durationMs: 500,
          scrollIntoView: false,
        }),
      ),
    ).not.toThrow();
  });
});
