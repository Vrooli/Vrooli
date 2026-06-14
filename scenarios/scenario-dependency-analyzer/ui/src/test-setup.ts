import "@testing-library/jest-dom/vitest";

import { afterEach, beforeEach, vi } from "vitest";

const getContextMock = vi.fn(() => null);

Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
  configurable: true,
  value: getContextMock
});

beforeEach(() => {
  window.localStorage.clear();
  getContextMock.mockClear();
});

afterEach(() => {
  vi.restoreAllMocks();
});
