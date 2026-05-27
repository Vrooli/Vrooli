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

beforeEach(() => {
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
