/**
 * SettingsProvider tests — verify the context persists changes to localStorage,
 * applies them to <html>, and re-applies "auto" text direction when the locale
 * changes (so an Arabic switch flips the document to RTL).
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { act, cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";

import { SettingsProvider, useSettings } from "./SettingsProvider";
import { DEFAULT_SETTINGS, SETTINGS_STORAGE_KEY, type SettingsState } from "./useSettings";
import { setLocale } from "../../i18n";

const root = () => document.documentElement;

const wrapper =
  (initialSettings?: SettingsState) =>
  ({ children }: { children: ReactNode }) =>
    <SettingsProvider initialSettings={initialSettings}>{children}</SettingsProvider>;

describe("SettingsProvider", () => {
  beforeEach(async () => {
    window.localStorage.clear();
    await setLocale("en");
    for (const attr of ["data-font-scale", "data-reduced-motion", "data-handedness"]) {
      root().removeAttribute(attr);
    }
  });

  afterEach(async () => {
    cleanup();
    await setLocale("en");
  });

  it("applies the initial settings to <html> on mount", () => {
    renderHook(() => useSettings(), {
      wrapper: wrapper({ ...DEFAULT_SETTINGS, fontScale: "large" }),
    });
    expect(root().getAttribute("data-font-scale")).toBe("large");
  });

  it("persists and applies a preference change", () => {
    const { result } = renderHook(() => useSettings(), { wrapper: wrapper(DEFAULT_SETTINGS) });

    act(() => result.current.setSettings({ reducedMotion: "always" }));

    expect(result.current.settings.reducedMotion).toBe("always");
    expect(root().getAttribute("data-reduced-motion")).toBe("always");
    const persisted = JSON.parse(window.localStorage.getItem(SETTINGS_STORAGE_KEY) ?? "{}");
    expect(persisted.reducedMotion).toBe("always");
  });

  it("merges partial patches without dropping other prefs", () => {
    const { result } = renderHook(() => useSettings(), {
      wrapper: wrapper({ ...DEFAULT_SETTINGS, handedness: "left" }),
    });

    act(() => result.current.setSettings({ fontScale: "small" }));

    expect(result.current.settings).toEqual({
      ...DEFAULT_SETTINGS,
      handedness: "left",
      fontScale: "small",
    });
  });

  it("re-applies auto text direction (→ rtl) when the locale switches to Arabic", async () => {
    renderHook(() => useSettings(), { wrapper: wrapper(DEFAULT_SETTINGS) });
    expect(root().dir).toBe("ltr");

    await act(async () => {
      await setLocale("ar");
    });

    expect(root().dir).toBe("rtl");
  });

  it("keeps an explicit ltr override even after switching to Arabic", async () => {
    renderHook(() => useSettings(), {
      wrapper: wrapper({ ...DEFAULT_SETTINGS, textDirection: "ltr" }),
    });

    await act(async () => {
      await setLocale("ar");
    });

    expect(root().dir).toBe("ltr");
  });
});
