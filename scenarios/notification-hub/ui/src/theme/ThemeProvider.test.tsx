/**
 * ThemeProvider tests — verify the data-theme attribute toggles in response to
 * user choice and that the choice persists to localStorage. `system` is the
 * only branch that consults matchMedia; covered by stubbing matchMedia.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";

import { readLocalPreference, ThemeProvider, usePreferenceStorageUnavailable, useTheme, type ThemeChoice } from "./ThemeProvider";

const STORAGE_KEY = "vrooli.theme";

const wrapper =
  (initialChoice?: ThemeChoice) =>
  ({ children }: { children: ReactNode }) =>
    <ThemeProvider initialChoice={initialChoice}>{children}</ThemeProvider>;

describe("ThemeProvider", () => {
  beforeEach(() => {
    window.localStorage.clear();
    readLocalPreference("theme", STORAGE_KEY);
    readLocalPreference("locale", "vrooli.locale");
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-resolved-theme");
  });

  afterEach(() => {
    vi.restoreAllMocks();
    cleanup();
  });

  it("uses system mode and reports a denied initial storage read without crashing render", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });

    const { result } = renderHook(() => ({ theme: useTheme(), unavailable: usePreferenceStorageUnavailable() }), { wrapper: wrapper() });

    expect(result.current.theme.choice).toBe("system");
    expect(result.current.unavailable).toBe(true);
    expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
  });

  it("sets data-theme on the html element for an explicit light choice", () => {
    renderHook(() => useTheme(), { wrapper: wrapper("light") });
    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
    expect(document.documentElement.dataset.resolvedTheme).toBe("light");
  });

  it("sets data-theme to dark when the user chooses dark", () => {
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("light") });
    act(() => result.current.setTheme("dark"));
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(result.current.choice).toBe("dark");
    expect(result.current.resolved).toBe("dark");
    expect(document.documentElement.dataset.resolvedTheme).toBe("dark");
  });

  it("removes data-theme when the user chooses system", () => {
    // matchMedia in jsdom defaults to no-match; resolved should be "light".
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("light") });
    act(() => result.current.setTheme("system"));
    expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
    expect(document.documentElement.dataset.resolvedTheme).toBe("light");
    expect(result.current.choice).toBe("system");
  });

  it("persists the choice to localStorage", () => {
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("light") });
    act(() => result.current.setTheme("dark"));
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe("dark");
  });

  it("applies the selected theme in memory when a storage write is denied", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });
    const { result } = renderHook(() => ({ theme: useTheme(), unavailable: usePreferenceStorageUnavailable() }), { wrapper: wrapper("light") });

    act(() => result.current.theme.setTheme("dark"));

    expect(result.current.theme.choice).toBe("dark");
    expect(result.current.theme.resolved).toBe("dark");
    expect(result.current.unavailable).toBe(true);
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
  });

  it("persists the latest theme and clears its warning after storage recovers", () => {
    const nativeSetItem = Storage.prototype.setItem;
    let denyWrites = true;
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(function (this: Storage, key: string, value: string) {
      if (denyWrites) throw new DOMException("blocked", "SecurityError");
      return nativeSetItem.call(this, key, value);
    });
    const { result } = renderHook(() => ({ theme: useTheme(), unavailable: usePreferenceStorageUnavailable() }), { wrapper: wrapper("light") });

    act(() => result.current.theme.setTheme("dark"));
    expect(result.current.unavailable).toBe(true);

    denyWrites = false;
    act(() => result.current.theme.setTheme("system"));

    expect(result.current.theme.choice).toBe("system");
    expect(result.current.unavailable).toBe(false);
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe("system");
  });

  it("resolves system to dark when prefers-color-scheme matches", () => {
    let matches = true;
    let onChange: ((event: MediaQueryListEvent) => void) | undefined;
    const matchMediaSpy = vi.spyOn(window, "matchMedia").mockImplementation((q) => ({
      get matches() { return q === "(prefers-color-scheme: dark)" && matches; },
      media: q,
      onchange: null,
      addEventListener: vi.fn((_type, listener) => { onChange = listener as (event: MediaQueryListEvent) => void; }),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("system") });
    expect(result.current.resolved).toBe("dark");
    expect(document.documentElement.dataset.resolvedTheme).toBe("dark");

    matches = false;
    act(() => onChange?.(new Event("change") as MediaQueryListEvent));
    expect(result.current.resolved).toBe("light");
    expect(document.documentElement.dataset.resolvedTheme).toBe("light");

    matchMediaSpy.mockRestore();
  });
});
