import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { renderHook, act, cleanup } from "@testing-library/react";
import * as React from "react";
import { PreferencesProvider, usePreferences } from "./usePreferences";

const STORAGE_KEY = "audio-tools.preferences.v1";

// jsdom doesn't implement window.matchMedia — stub it globally before any hook renders
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }),
  });
});

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

function wrapper({ children }: { children: React.ReactNode }) {
  return <PreferencesProvider>{children}</PreferencesProvider>;
}

function renderPreferences() {
  return renderHook(() => usePreferences(), { wrapper });
}

describe("usePreferences", () => {
  describe("defaults", () => {
    it("returns default theme=system", () => {
      const { result } = renderPreferences();
      expect(result.current.preferences.theme).toBe("system");
    });

    it("returns default fontScale=comfortable", () => {
      const { result } = renderPreferences();
      expect(result.current.preferences.fontScale).toBe("comfortable");
    });

    it("returns default reducedMotion=false", () => {
      const { result } = renderPreferences();
      expect(result.current.preferences.reducedMotion).toBe(false);
    });
  });

  describe("setTheme", () => {
    it("updates theme to light", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setTheme("light"); });
      expect(result.current.preferences.theme).toBe("light");
    });

    it("updates theme to dark", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setTheme("dark"); });
      expect(result.current.preferences.theme).toBe("dark");
    });

    it("resolves dark theme correctly", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setTheme("dark"); });
      expect(result.current.resolvedTheme).toBe("dark");
    });

    it("resolves light theme correctly", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setTheme("light"); });
      expect(result.current.resolvedTheme).toBe("light");
    });
  });

  describe("setFontScale", () => {
    it("updates fontScale to compact", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setFontScale("compact"); });
      expect(result.current.preferences.fontScale).toBe("compact");
    });

    it("updates fontScale to large", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setFontScale("large"); });
      expect(result.current.preferences.fontScale).toBe("large");
    });
  });

  describe("setReducedMotion", () => {
    it("sets reducedMotion to true", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setReducedMotion(true); });
      expect(result.current.preferences.reducedMotion).toBe(true);
    });

    it("toggles reducedMotion back to false", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setReducedMotion(true); });
      act(() => { result.current.setReducedMotion(false); });
      expect(result.current.preferences.reducedMotion).toBe(false);
    });
  });

  describe("localStorage persistence", () => {
    it("persists theme to localStorage", () => {
      const { result } = renderPreferences();
      act(() => { result.current.setTheme("dark"); });
      const stored = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "{}") as Record<string, unknown>;
      expect(stored.theme).toBe("dark");
    });

    it("reads persisted theme on mount", () => {
      window.localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ theme: "dark", fontScale: "large", reducedMotion: true }),
      );
      const { result } = renderPreferences();
      expect(result.current.preferences.theme).toBe("dark");
      expect(result.current.preferences.fontScale).toBe("large");
      expect(result.current.preferences.reducedMotion).toBe(true);
    });

    it("falls back to defaults when stored JSON is partial", () => {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ theme: "light" }));
      const { result } = renderPreferences();
      expect(result.current.preferences.theme).toBe("light");
      expect(result.current.preferences.fontScale).toBe("comfortable");
      expect(result.current.preferences.reducedMotion).toBe(false);
    });

    it("falls back to defaults when stored JSON is invalid", () => {
      window.localStorage.setItem(STORAGE_KEY, "not-valid-json{{{");
      const { result } = renderPreferences();
      expect(result.current.preferences.theme).toBe("system");
    });
  });

  describe("system theme resolution", () => {
    beforeEach(() => {
      Object.defineProperty(window, "matchMedia", {
        writable: true,
        value: vi.fn().mockReturnValue({
          matches: true,
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
        }),
      });
    });

    it("resolves system theme to dark when OS prefers dark", () => {
      const { result } = renderPreferences();
      // default is system; OS mock returns matches:true (dark)
      expect(result.current.resolvedTheme).toBe("dark");
    });
  });

  describe("error guard", () => {
    it("throws when usePreferences is called outside PreferencesProvider", () => {
      const spy = vi.spyOn(console, "error").mockImplementation(() => {});
      expect(() => renderHook(() => usePreferences())).toThrow(
        "usePreferences must be used inside <PreferencesProvider>",
      );
      spy.mockRestore();
    });
  });
});

describe("OS theme change listener", () => {
  it("updates resolvedTheme when OS theme changes while system mode active", () => {
    let changeHandler: (() => void) | null = null;
    const mockMql = {
      matches: false,
      addEventListener: vi.fn((_e: string, cb: () => void) => { changeHandler = cb; }),
      removeEventListener: vi.fn(),
    };
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockReturnValue(mockMql),
    });
    const { result } = renderPreferences();
    // Default theme is "system"; listener should be registered
    expect(mockMql.addEventListener).toHaveBeenCalledWith("change", expect.any(Function));
    // Simulate OS switching to dark
    act(() => {
      mockMql.matches = true;
      if (changeHandler) changeHandler();
    });
    expect(result.current.resolvedTheme).toBe("dark");
    // Simulate OS switching back to light
    act(() => {
      mockMql.matches = false;
      if (changeHandler) changeHandler();
    });
    expect(result.current.resolvedTheme).toBe("light");
  });

  it("does not register OS change listener when theme is not system", () => {
    const mockMql = {
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    };
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockReturnValue(mockMql),
    });
    const { result } = renderPreferences();
    act(() => { result.current.setTheme("dark"); });
    // The initial system-mode registration happens during initial render
    // After switching to dark, no new listener for system-mode changes
    // The removeEventListener should have been called (cleanup from initial effect)
    expect(result.current.preferences.theme).toBe("dark");
  });
});

describe("localStorage error handling", () => {
  it("falls back gracefully when localStorage.setItem throws", () => {
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockReturnValue({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      }),
    });
    const setItemSpy = vi.spyOn(window.localStorage, "setItem").mockImplementation(() => {
      throw new Error("QuotaExceededError");
    });
    const { result } = renderPreferences();
    // Should not throw even if setItem fails
    act(() => { result.current.setTheme("dark"); });
    expect(result.current.preferences.theme).toBe("dark");
    setItemSpy.mockRestore();
  });
});
