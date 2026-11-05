import "@testing-library/jest-dom";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";
import { queryClient } from "../providers/QueryProvider";

afterEach(() => {
  cleanup();
  queryClient.clear();
  vi.clearAllMocks();
  localStorage.clear();
  sessionStorage.clear();
});

class ResizeObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}

globalThis.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver;

globalThis.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
  root: null,
  rootMargin: "",
  thresholds: [],
})) as unknown as typeof IntersectionObserver;

Element.prototype.scrollIntoView = vi.fn();

// A plain function, not vi.fn(): suites that reset or restore all mocks must
// not leave shared components (overlays read prefers-reduced-motion) without it.
const noop = () => undefined;
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: noop,
    removeListener: noop,
    addEventListener: noop,
    removeEventListener: noop,
    dispatchEvent: () => false,
  }),
});
