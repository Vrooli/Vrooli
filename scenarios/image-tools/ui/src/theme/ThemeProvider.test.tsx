/**
 * ThemeProvider tests — verify the data-theme attribute toggles in response to
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
  });

  afterEach(() => {
    cleanup();
  });

  it("sets data-theme on the html element for an explicit light choice", () => {
    renderHook(() => useTheme(), { wrapper: wrapper("light") });
    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
  });

  it("sets data-theme to dark when the user chooses dark", () => {
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("light") });
    act(() => result.current.setTheme("dark"));
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(result.current.choice).toBe("dark");
    expect(result.current.resolved).toBe("dark");
  });

  it("removes data-theme when the user chooses system", () => {
    // matchMedia in jsdom defaults to no-match; resolved should be "light".
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("light") });
    act(() => result.current.setTheme("system"));
    expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
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

  it("reads a valid stored choice on first load (no initialChoice)", () => {
    window.localStorage.setItem(STORAGE_KEY, "dark");
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper() });
    expect(result.current.choice).toBe("dark");
    expect(result.current.resolved).toBe("dark");
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
  });

  it("falls back to system when the stored value is not a valid choice", () => {
    window.localStorage.setItem(STORAGE_KEY, "neon");
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper() });
    expect(result.current.choice).toBe("system");
  });

  it("resolves system to light when matchMedia is absent (guard branch)", () => {
    const original = window.matchMedia;
    // Delete the global so resolveChoice's `if (!matchMedia)` guard fires.
    delete (window as { matchMedia?: typeof window.matchMedia }).matchMedia;
    try {
      const { result } = renderHook(() => useTheme(), { wrapper: wrapper("system") });
      expect(result.current.resolved).toBe("light");
      expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
    } finally {
      window.matchMedia = original;
    }
  });

  it("subscribes to the media query while on system and re-resolves on change", () => {
    let changeHandler: (() => void) | undefined;
    const removeEventListener = vi.fn();
    const mql = {
      matches: false,
      media: "(prefers-color-scheme: dark)",
      onchange: null,
      addEventListener: vi.fn((_evt: string, cb: () => void) => {
        changeHandler = cb;
      }),
      removeEventListener,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    };
    const matchMediaSpy = vi.spyOn(window, "matchMedia").mockReturnValue(mql);

    const { result, unmount } = renderHook(() => useTheme(), { wrapper: wrapper("system") });
    expect(result.current.resolved).toBe("light");
    expect(mql.addEventListener).toHaveBeenCalledWith("change", expect.any(Function));

    // OS flips to dark → the handler reads mql.matches and re-resolves.
    mql.matches = true;
    act(() => changeHandler?.());
    expect(result.current.resolved).toBe("dark");

    unmount();
    expect(removeEventListener).toHaveBeenCalledWith("change", expect.any(Function));

    matchMediaSpy.mockRestore();
  });

  it("does not subscribe to the media query for an explicit (non-system) choice", () => {
    const addEventListener = vi.fn();
    const matchMediaSpy = vi.spyOn(window, "matchMedia").mockReturnValue({
      matches: false,
      media: "(prefers-color-scheme: dark)",
      onchange: null,
      addEventListener,
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    });

    renderHook(() => useTheme(), { wrapper: wrapper("light") });
    // The effect's `choice !== "system"` guard returns before subscribing.
    expect(addEventListener).not.toHaveBeenCalled();

    matchMediaSpy.mockRestore();
  });

  it("tears down the system listener when the user switches away from system", () => {
    const removeEventListener = vi.fn();
    const matchMediaSpy = vi.spyOn(window, "matchMedia").mockReturnValue({
      matches: false,
      media: "(prefers-color-scheme: dark)",
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    });

    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("system") });
    act(() => result.current.setTheme("light"));
    expect(removeEventListener).toHaveBeenCalledWith("change", expect.any(Function));

    matchMediaSpy.mockRestore();
  });
});
