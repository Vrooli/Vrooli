/**
 * ThemeProvider tests — verify the data-theme / data-resolved-theme attributes toggle in response to
 * user choice and that the choice persists to localStorage. `auto` is the
 * only branch that follows the local Observatory day/night window.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";

import { ThemeProvider, useTheme, type ThemeChoice } from "./ThemeProvider";
import { OBSERVATORY_NIGHT_START_KEY, OBSERVATORY_PREFERENCES_EVENT } from "./observatoryAppearance";

const STORAGE_KEY = "vrooli.theme";

const wrapper =
  (initialChoice?: ThemeChoice) =>
  ({ children }: { children: ReactNode }) =>
    <ThemeProvider initialChoice={initialChoice}>{children}</ThemeProvider>;

describe("ThemeProvider", () => {
  beforeEach(() => {
    window.localStorage.clear();
    window.history.replaceState({}, "", "/");
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-resolved-theme");
    vi.useRealTimers();
  });

  afterEach(() => {
    cleanup();
  });

  it("sets data-theme on the html element for an explicit day choice", () => {
    renderHook(() => useTheme(), { wrapper: wrapper("day") });
    expect(document.documentElement.getAttribute("data-theme")).toBe("day");
    expect(document.documentElement.getAttribute("data-resolved-theme")).toBe("light");
    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute("content")).toBe("rgb(235 229 216)");
  });

  it("resolves a URL appearance override before the shell mounts", () => {
    window.history.replaceState({}, "", "/?appearance=night");
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper() });
    expect(result.current.choice).toBe("night");
    expect(result.current.resolved).toBe("dark");
    expect(document.documentElement.getAttribute("data-theme")).toBe("night");
  });

  it("sets data-theme to night when the user chooses night", () => {
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("day") });
    act(() => result.current.setTheme("night"));
    expect(document.documentElement.getAttribute("data-theme")).toBe("night");
    // The design kit's dark palette keys on this attribute, so an explicit
    // choice must reach it even when the OS prefers light.
    expect(document.documentElement.getAttribute("data-resolved-theme")).toBe("dark");
    expect(result.current.choice).toBe("night");
    expect(result.current.resolved).toBe("dark");
    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute("content")).toBe("rgb(23 22 52)");
  });

  it("keeps media-specific browser chrome colors aligned with the resolved appearance", async () => {
    const darkVariant = document.createElement("meta");
    darkVariant.name = "theme-color";
    darkVariant.setAttribute("media", "(prefers-color-scheme: dark)");
    document.head.appendChild(darkVariant);
    renderHook(() => useTheme(), { wrapper: wrapper("night") });
    await waitFor(() => expect(darkVariant.content).toBe("rgb(23 22 52)"));
    darkVariant.remove();
  });

  it("removes data-theme when the user chooses auto", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T12:00:00"));
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("day") });
    act(() => result.current.setTheme("auto"));
    expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
    expect(document.documentElement.getAttribute("data-resolved-theme")).toBe("light");
    expect(result.current.choice).toBe("auto");
  });

  it("persists the choice to localStorage", () => {
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("day") });
    act(() => result.current.setTheme("night"));
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe("night");
  });

  it("resolves auto to dark during the local night window", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T20:00:00"));
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("auto") });
    expect(result.current.resolved).toBe("dark");
  });

  it("updates the auto theme when the local night boundary changes", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T20:00:00"));
    window.localStorage.setItem(OBSERVATORY_NIGHT_START_KEY, "19:00");
    const { result } = renderHook(() => useTheme(), { wrapper: wrapper("auto") });
    expect(result.current.resolved).toBe("dark");

    window.localStorage.setItem(OBSERVATORY_NIGHT_START_KEY, "21:00");
    act(() => window.dispatchEvent(new Event(OBSERVATORY_PREFERENCES_EVENT)));
    expect(result.current.resolved).toBe("light");
  });
});
