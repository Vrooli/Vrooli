import "@testing-library/jest-dom";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import { i18n } from "../i18n";
import { configureTestProviders } from "@vrooli/api-base/testing";
import { I18nextProvider } from "react-i18next";
import { createElement } from "react";

configureTestProviders((children) => createElement(I18nextProvider, { i18n }, children));

// Default every test into i18next's `cimode` pseudo-locale. In cimode,
// `t("app.title")` returns the *key* (`"app.title"`) rather than translated
// copy, so component tests can assert against `strings.app.title` from the
// typed registry instead of brittle string literals. Translators rewriting
// English copy never break tests; only structural changes to keys do.
//
// Tests that specifically validate translation behaviour (locale switcher,
// real English/Arabic rendering, RTL pipeline) opt out via their own
// `beforeEach(() => setLocale("en"))` — the file-local hook runs after this
// process-wide one, so per-file overrides win.
beforeEach(async () => {
  window.localStorage.clear();
  await i18n.changeLanguage("cimode");
});

// jsdom does not implement matchMedia, which the shared component library's
// useMediaQuery hook calls through useSyncExternalStore. Without this, every
// test rendering an RCL component that reads a breakpoint throws before its
// first assertion. Defaults to "does not match" so tests see the desktop
// layout unless they install their own stub.
Object.defineProperty(window, "matchMedia", {
  configurable: true,
  writable: true,
  value: vi.fn((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(() => false),
  })),
});

Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
  configurable: true,
  value: vi.fn(() => ({})),
});

// jsdom doesn't implement scrollTo or scrollIntoView
Element.prototype.scrollTo = vi.fn() as unknown as typeof Element.prototype.scrollTo;
Element.prototype.scrollIntoView = vi.fn();
HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
HTMLMediaElement.prototype.pause = vi.fn();
HTMLMediaElement.prototype.load = vi.fn();

// Automatic cleanup after each test — prevents leaked DOM state
// between tests that use @testing-library/react render().
afterEach(() => {
  cleanup();
});
