/**
 * ThemeProvider tests — verify the data-theme / data-resolved-theme attributes toggle in response to
 * user choice and that the choice persists to localStorage. `system` is the
 * only branch that consults matchMedia; covered by stubbing matchMedia.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";

import { ThemeProvider, useTheme, type ThemeChoice } from "./ThemeProvider";

const STORAGE_KEY = "vrooli.theme";

const wrapper =
  (initialChoice?: ThemeChoice) =>
  ({ children }: { children: ReactNode }) =>
    <ThemeProvider initialChoice={initialChoice}>{children}</ThemeProvider>;

describe("ThemeProvider", () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-resolved-theme");
  });

  afterEach(() => {
    cleanup();
  });

  it("sets data-theme on the html element for an explicit light choice", () => {
    renderHook(() => useTheme(), { wrapper: wrapper("light") });
    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
    expect(document.documentElement.getAttribute("data-resolved-theme")).toBe("light");
  });

  it("sets data-theme to dark when the user chooses dark", () => {
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("light") });
    act(() => result.current.setTheme("dark"));
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    // The design kit's dark palette keys on this attribute, so an explicit
    // choice must reach it even when the OS prefers light.
    expect(document.documentElement.getAttribute("data-resolved-theme")).toBe("dark");
    expect(result.current.choice).toBe("dark");
    expect(result.current.resolved).toBe("dark");
  });

  it("removes data-theme when the user chooses system", () => {
    // matchMedia in jsdom defaults to no-match; resolved should be "light".
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("light") });
    act(() => result.current.setTheme("system"));
    expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
    expect(document.documentElement.getAttribute("data-resolved-theme")).toBe("light");
    expect(result.current.choice).toBe("system");
  });

  it("persists the choice to localStorage", () => {
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("light") });
    act(() => result.current.setTheme("dark"));
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe("dark");
  });

  it("resolves system to dark when prefers-color-scheme matches", () => {
    const matchMediaSpy = vi.spyOn(window, "matchMedia").mockImplementation((q) => ({
      matches: q === "(prefers-color-scheme: dark)",
      media: q,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("system") });
    expect(result.current.resolved).toBe("dark");

    matchMediaSpy.mockRestore();
  });

  it("uses system when stored preferences are invalid and follows later media changes", () => {
    window.localStorage.setItem(STORAGE_KEY, "dark");
    const stored = renderHook(() => useTheme(), { wrapper: wrapper() });
    expect(stored.result.current.choice).toBe("dark");
    stored.unmount();
    window.localStorage.setItem(STORAGE_KEY, "sepia");
    let onChange: (() => void) | undefined;
    let prefersDark = true;
    const addEventListener = vi.fn((_type: string, listener: () => void) => { onChange = listener; });
    const removeEventListener = vi.fn();
    const matchMediaSpy = vi.spyOn(window, "matchMedia").mockImplementation((q) => ({
      get matches() { return q === "(prefers-color-scheme: dark)" && prefersDark; },
      media: q, onchange: null, addEventListener, removeEventListener,
      addListener: vi.fn(), removeListener: vi.fn(), dispatchEvent: vi.fn(),
    }));
    const { result, unmount } = renderHook(() => useTheme(), { wrapper: wrapper() });
    expect(result.current.choice).toBe("system");
    expect(result.current.resolved).toBe("dark");
    prefersDark = false;
    act(() => onChange?.());
    expect(result.current.resolved).toBe("light");
    unmount();
    expect(removeEventListener).toHaveBeenCalledWith("change", onChange);
    matchMediaSpy.mockRestore();
  });

  it("falls back to light when system media queries are unavailable", () => {
    const descriptor = Object.getOwnPropertyDescriptor(window, "matchMedia");
    Object.defineProperty(window, "matchMedia", { configurable: true, value: undefined });
    const { result, unmount } = renderHook(() => useTheme(), { wrapper: wrapper("system") });
    expect(result.current.resolved).toBe("light");
    unmount();
    if (descriptor) Object.defineProperty(window, "matchMedia", descriptor);
  });

  it("rejects use outside its provider", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);
    try {
      expect(() => renderHook(() => useTheme())).toThrow("useTheme must be called inside <ThemeProvider>");
    } finally {
      consoleError.mockRestore();
    }
  });
});
