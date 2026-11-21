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
});
