import "@testing-library/jest-dom";
import { afterEach, beforeEach, vi } from "vitest";

const originalFetch = globalThis.fetch;
const originalConsoleLog = console.log;

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "https://git-control-tower.test/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) => `${opts.baseUrl}${path}`,
  // Connect transport: tests that exercise Connect clients mock the thin api-*
  // wrappers, so the transport only needs to construct, not transport.
  createScenarioConnectTransport: () => ({}),
}));

Element.prototype.scrollIntoView = vi.fn();

if (!window.matchMedia) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
}

beforeEach(() => {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
  vi.spyOn(console, "log").mockImplementation((...args: unknown[]) => {
    if (typeof args[0] === "string" && args[0].startsWith("[api-base]")) {
      return;
    }
    originalConsoleLog(...args);
  });
});

afterEach(() => {
  vi.restoreAllMocks();
  globalThis.fetch = originalFetch;
  window.localStorage.clear();
  window.sessionStorage.clear();
});
