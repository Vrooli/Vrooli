/**
 * Shared test utilities for usePipelineButton tests.
 */

import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

// Mock the API module
vi.mock("../../lib/api", () => ({
  runPipeline: vi.fn(),
  getPipelineStatus: vi.fn(),
  checkWineStatus: vi.fn(),
}));

// Import mocks after setting up vi.mock
import { runPipeline, getPipelineStatus, checkWineStatus } from "../../lib/api";

export const mockRunPipeline = runPipeline as ReturnType<typeof vi.fn>;
export const mockGetPipelineStatus = getPipelineStatus as ReturnType<typeof vi.fn>;
export const mockCheckWineStatus = checkWineStatus as ReturnType<typeof vi.fn>;

// Create a wrapper with QueryClientProvider
export function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

// Mock localStorage
export const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, "localStorage", { value: localStorageMock });
