import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

// Mock noVNC — it uses top-level await which breaks in jsdom/vitest
vi.mock("@novnc/novnc/lib/rfb", () => ({
  default: vi.fn().mockImplementation(() => ({
    scaleViewport: false,
    resizeSession: false,
    clipViewport: false,
    disconnect: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })),
}));

// Polyfill window.matchMedia for jsdom (returns non-matching by default).
// Individual tests can override this with vi.fn() if they need specific behaviour.
if (typeof window !== "undefined" && !window.matchMedia) {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  });
}

// Cleanup after each test
afterEach(() => {
  cleanup();
});
