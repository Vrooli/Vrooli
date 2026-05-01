import "@testing-library/jest-dom";
import { afterEach, beforeEach, vi } from "vitest";

const originalFetch = globalThis.fetch;
const originalConsoleLog = console.log;

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "https://git-control-tower.test/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) => `${opts.baseUrl}${path}`,
}));

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
