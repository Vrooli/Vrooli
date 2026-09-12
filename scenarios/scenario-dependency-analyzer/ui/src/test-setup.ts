import "@testing-library/jest-dom/vitest";

import { afterEach, beforeEach, vi } from "vitest";

const getContextMock = vi.fn(() => null);

Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
  configurable: true,
  value: getContextMock
});

Object.defineProperty(window, "matchMedia", {
  configurable: true,
  writable: true,
  value: (media: string) => ({
    media,
    matches: media.includes("min-width"),
    onchange: null,
    addListener() {},
    removeListener() {},
    addEventListener() {},
    removeEventListener() {},
    dispatchEvent: () => false,
  }),
});

beforeEach(() => {
  window.localStorage.clear();
  getContextMock.mockClear();
});

afterEach(() => {
  vi.restoreAllMocks();
});
