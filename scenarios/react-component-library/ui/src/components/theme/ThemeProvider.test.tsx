// provider-free-exception: this suite mounts ThemeProvider itself; the
// canonical renderWithProviders wrapper already includes a ThemeProvider, and
// nesting a second one would have both instances fighting over
// documentElement state.
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "./ThemeProvider";
import { useTheme } from "./useTheme";

function Probe() {
  const { theme, resolved, setTheme } = useTheme();
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <span data-testid="resolved">{resolved}</span>
      <button data-testid="to-dark" onClick={() => setTheme("dark")}>
        dark
      </button>
      <button data-testid="to-system" onClick={() => setTheme("system")}>
        system
      </button>
    </div>
  );
}

function installMatchMedia(dark: boolean) {
  const listeners = new Set<(e: MediaQueryListEvent) => void>();
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (q: string) =>
      ({
        matches: dark,
        media: q,
        addEventListener: (_t: string, l: (e: MediaQueryListEvent) => void) =>
          listeners.add(l),
        removeEventListener: (_t: string, l: (e: MediaQueryListEvent) => void) =>
          listeners.delete(l),
        dispatchEvent: () => true,
      }) as unknown as MediaQueryList,
  });
  return {
    flip(next: boolean) {
      listeners.forEach((l) =>
        l({ matches: next, media: "" } as MediaQueryListEvent),
      );
    },
  };
}

describe("ThemeProvider", () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.removeAttribute("data-resolved-theme");
    document.documentElement.classList.remove("dark");
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("defaults to system and resolves to OS preference", () => {
    installMatchMedia(true);
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );
    expect(screen.getByTestId("theme").textContent).toBe("system");
    expect(screen.getByTestId("resolved").textContent).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("setTheme writes the dark class on <html>", async () => {
    installMatchMedia(false);
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    await user.click(screen.getByTestId("to-dark"));
    expect(screen.getByTestId("resolved").textContent).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("system mode tracks OS toggles", () => {
    const mm = installMatchMedia(false);
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );
    expect(screen.getByTestId("resolved").textContent).toBe("light");
    act(() => mm.flip(true));
    expect(screen.getByTestId("resolved").textContent).toBe("dark");
  });

  it("persists to localStorage for first-paint cache", async () => {
    installMatchMedia(false);
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );
    await user.click(screen.getByTestId("to-dark"));
    expect(window.localStorage.getItem("react-component-library.theme.v1")).toBe("dark");
  });

});
