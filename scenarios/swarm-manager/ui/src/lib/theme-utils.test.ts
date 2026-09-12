import { afterEach, describe, expect, it, vi } from "vitest";
import { applyTheme, resolveTheme, watchSystemTheme } from "./theme-utils";
import { installMatchMediaMock } from "../test-utils/browser";

afterEach(() => {
  const root = document.documentElement;
  delete root.dataset.theme;
  delete root.dataset.resolvedTheme;
  root.style.colorScheme = "";
  vi.restoreAllMocks();
});

describe("applyTheme", () => {
  it("applies an explicit dark theme to the document root", () => {
    const resolved = applyTheme("dark");

    expect(resolved).toBe("dark");
    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(document.documentElement.dataset.resolvedTheme).toBe("dark");
    expect(document.documentElement.style.colorScheme).toBe("dark");
  });

  it("applies an explicit light theme to the document root", () => {
    const resolved = applyTheme("light");

    expect(resolved).toBe("light");
    expect(document.documentElement.dataset.theme).toBe("light");
    expect(document.documentElement.dataset.resolvedTheme).toBe("light");
    expect(document.documentElement.style.colorScheme).toBe("light");
  });

  it("resolves the system preference to the matched media query", () => {
    installMatchMediaMock(true); // prefers-color-scheme: dark matches
    expect(resolveTheme("system")).toBe("dark");
    expect(applyTheme("system")).toBe("dark");
    expect(document.documentElement.dataset.theme).toBe("system");
    expect(document.documentElement.dataset.resolvedTheme).toBe("dark");

    installMatchMediaMock(false); // does not match -> light
    expect(resolveTheme("system")).toBe("light");
    expect(applyTheme("system")).toBe("light");
    expect(document.documentElement.dataset.resolvedTheme).toBe("light");
  });

  it("dispatches a theme-change event so subscribers can react", () => {
    const handler = vi.fn();
    window.addEventListener("vrooli-theme-change", handler);

    applyTheme("light");

    expect(handler).toHaveBeenCalledOnce();
    const event = handler.mock.calls[0]?.[0] as CustomEvent<string>;
    expect(event.detail).toBe("light");

    window.removeEventListener("vrooli-theme-change", handler);
  });
});

describe("watchSystemTheme", () => {
  it("subscribes to prefers-color-scheme changes and reports the resolved theme", () => {
    const handle: { registered: (() => void) | null } = { registered: null };
    let matches = false;
    const removeEventListener = vi.fn();
    const matchMedia = vi.fn().mockImplementation((query: string) => ({
      get matches() {
        return matches;
      },
      media: query,
      addEventListener: (_: string, cb: () => void) => {
        handle.registered = cb;
      },
      removeEventListener,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));
    Object.defineProperty(window, "matchMedia", { writable: true, value: matchMedia });

    const onChange = vi.fn();
    const unsubscribe = watchSystemTheme(onChange);

    expect(handle.registered).not.toBeNull();

    matches = true;
    handle.registered?.();
    expect(onChange).toHaveBeenLastCalledWith("dark");

    matches = false;
    handle.registered?.();
    expect(onChange).toHaveBeenLastCalledWith("light");

    unsubscribe();
    expect(removeEventListener).toHaveBeenCalledOnce();
  });
});
