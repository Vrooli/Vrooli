/**
 * Vitest setup file
 *
 * 1. Registers @testing-library/jest-dom matchers (`.toBeInTheDocument()`, …).
 * 2. Defaults i18next into `cimode` before every test so `t('app.title')`
 *    returns the *key* (`"app.title"`), not translated copy. This makes
 *    component tests robust against any wording change in any locale —
 *    assertions reference the typed `strings.*` registry, not brittle
 *    string literals.
 *
 * Tests that specifically need to validate translation behaviour (locale
 * switcher, real English/Japanese rendering, etc.) opt back via their own
 * `beforeEach`:
 *
 *   beforeEach(async () => { await setLocale("en"); });
 *
 * The setup-file `beforeEach` runs first, so per-file overrides win.
 *
 * This file owns *setup-side-effects only*. Helpers tests reach for
 * (e.g. `interp`) live in `@/test-utils` so the setup-file remains a
 * pure registration surface — moving them out keeps test-setup.ts
 * single-purpose and makes the import graph for individual tests
 * easier to follow.
 */
import "@testing-library/jest-dom/vitest";
import { afterEach, beforeEach, vi, type MockInstance } from "vitest";
import { i18n, i18nReady } from "./i18n";

// jsdom does not implement matchMedia. The shared VoiceInputButton and the
// application preference hooks both use it, so keep the browser contract
// available to every component test instead of duplicating local stubs.
if (typeof window.matchMedia !== "function") {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: (query: string): MediaQueryList => {
      let matches = false;
      type MatchMediaListener = (event: MediaQueryListEvent) => void;
      const listeners = new Set<MatchMediaListener>();
      return {
        get matches() {
          return matches;
        },
        media: query,
        onchange: null,
        addEventListener: (_type: string, listener: EventListenerOrEventListenerObject | null) => {
          if (typeof listener === "function") listeners.add(listener);
        },
        removeEventListener: (_type: string, listener: EventListenerOrEventListenerObject | null) => {
          if (typeof listener === "function") listeners.delete(listener);
        },
        addListener: (listener: MatchMediaListener) => listeners.add(listener),
        removeListener: (listener: MatchMediaListener) => listeners.delete(listener),
        dispatchEvent: (event) => {
          matches = (event as MediaQueryListEvent).matches;
          for (const listener of listeners) listener(event as MediaQueryListEvent);
          return true;
        },
      };
    },
  });
}

// jsdom does not provide ResizeObserver. Components that virtualize or size
// scrollable content only need the registration lifecycle in most unit tests;
// layout-specific behavior belongs in a browser-backed test.
if (typeof globalThis.ResizeObserver === "undefined") {
  class TestResizeObserver implements ResizeObserver {
    constructor(private readonly callback: ResizeObserverCallback) {}
    observe(): void {
      this.callback([], this);
    }
    unobserve(): void {}
    disconnect(): void {}
  }
  Object.defineProperty(globalThis, "ResizeObserver", {
    configurable: true,
    value: TestResizeObserver,
  });
}

type ConsoleMethod = (...args: unknown[]) => void;
let consoleError: MockInstance<ConsoleMethod>;
let consoleWarn: MockInstance<ConsoleMethod>;

beforeEach(async () => {
  window.localStorage.clear();
  await i18nReady;
  await i18n.changeLanguage("cimode");
  consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
  consoleWarn = vi.spyOn(console, "warn").mockImplementation(() => {});
});

afterEach(() => {
  const errorCalls = consoleError.mock.calls;
  const warnCalls = consoleWarn.mock.calls;
  consoleError.mockRestore();
  consoleWarn.mockRestore();

  if (errorCalls.length > 0 || warnCalls.length > 0) {
    const formatted = [
      ...errorCalls.map((args: unknown[]) => `console.error: ${args.map(String).join(" ")}`),
      ...warnCalls.map((args: unknown[]) => `console.warn: ${args.map(String).join(" ")}`),
    ].join("\n");
    throw new Error(`Unexpected console output during test:\n${formatted}`);
  }
});

// Process-wide spy by intent — axe-core probes canvas during every a11y
// run, and jsdom doesn't implement HTMLCanvasElement.getContext. We do
// NOT call vi.restoreAllMocks() between tests because the canvas mock
// is part of the test environment, not per-test arrangement. If a
// future test legitimately needs a real canvas surface, it should
// explicitly `vi.spyOn(HTMLCanvasElement.prototype, "getContext")` in
// its own beforeEach and restore it on teardown — opt-in override
// rather than process-wide unwiring.
vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
