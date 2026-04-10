// Test setup file for Vitest
// [REQ:P0-001] Reference Scenario Registry - Test infrastructure
import "@testing-library/jest-dom";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

// Cleanup after each test
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

// Mock ResizeObserver (not available in jsdom)
(globalThis as typeof globalThis & { ResizeObserver: unknown }).ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn()
}));

// Mock fetch globally for API tests
(globalThis as typeof globalThis & { fetch: unknown }).fetch = vi.fn();
