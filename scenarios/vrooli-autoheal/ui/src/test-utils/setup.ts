import "@testing-library/jest-dom";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

if (!("ResizeObserver" in globalThis)) {
  (globalThis as typeof globalThis & { ResizeObserver: typeof ResizeObserver })
    .ResizeObserver = vi.fn().mockImplementation(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }));
}

// axe-core probes canvas in jsdom; keep the browser-only API deterministic.
vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
