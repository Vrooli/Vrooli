/**
 * ThemeProvider tests — verify the data-theme attribute toggles in response to
 * user choice and that the choice persists to localStorage. `system` is the
 * only branch that consults matchMedia; covered by stubbing matchMedia.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { readFileSync } from "node:fs";

import { ThemeProvider, useTheme, type ThemeChoice } from "./ThemeProvider";

const STORAGE_KEY = "vrooli.theme";
const designTokens = readFileSync("src/design-tokens.css", "utf8");

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

describe("theme contrast", () => {
  it.each(["light", "dark"] as const)("keeps %s text and semantic controls at WCAG AA", (theme) => {
    const tokens = themeTokens(theme);
    const value = (name: string) => token(tokens, name);
    const surface = value("color-surface");
    const background = value("color-background");
    const pairs: Array<[string, string]> = [
      [value("color-foreground"), surface],
      [value("color-muted-foreground"), surface],
      [value("color-primary-foreground"), value("color-primary")],
      [value("color-primary-foreground"), value("color-danger")],
      [value("color-warning"), composite(value("color-warning"), surface, 0.1)],
      [value("color-danger"), composite(value("color-danger"), background, 0.1)],
      [value("color-success"), composite(value("color-success"), surface, 0.1)],
      [value("color-info"), composite(value("color-info"), surface, 0.1)],
    ];

    for (const [foreground, backdrop] of pairs) {
      expect(contrast(foreground, backdrop), `${foreground} on ${backdrop}`).toBeGreaterThanOrEqual(4.5);
    }
  });
});

function themeTokens(theme: "light" | "dark") {
  const selector = theme === "light" ? ":root" : ':root[data-resolved-theme="dark"]';
  const start = designTokens.indexOf(selector);
  const open = designTokens.indexOf("{", start);
  const body = designTokens.slice(open, designTokens.indexOf("}", open));
  return Object.fromEntries(
    [...body.matchAll(/--([\w-]+):\s*(#[\da-f]{6});/gi)].map((match) => [match[1], match[2]]),
  ) as Record<string, string>;
}

function token(tokens: Record<string, string>, name: string) {
  const value = tokens[name];
  if (!value) throw new Error(`Missing design token --${name}`);
  return value;
}

function composite(foreground: string, background: string, alpha: number) {
  const fg = rgb(foreground);
  const bg = rgb(background);
  return `#${fg.map((channel, index) => Math.round(channel * alpha + bg[index]! * (1 - alpha)).toString(16).padStart(2, "0")).join("")}`;
}

function contrast(first: string, second: string) {
  const values = [luminance(first), luminance(second)].sort((a, b) => b - a);
  return (values[0]! + 0.05) / (values[1]! + 0.05);
}

function luminance(color: string) {
  const channels = rgb(color).map((channel) => {
    const normalized = channel / 255;
    return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * channels[0]! + 0.7152 * channels[1]! + 0.0722 * channels[2]!;
}

function rgb(color: string) {
  return (color.match(/[\da-f]{2}/gi) ?? []).map((channel) => Number.parseInt(channel, 16));
}
