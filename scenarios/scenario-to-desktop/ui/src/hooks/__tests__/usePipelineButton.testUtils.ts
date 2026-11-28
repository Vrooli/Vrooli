/**
 * Shared test utilities for usePipelineButton tests.
 */

import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

const mocks = vi.hoisted(() => ({
  runPipeline: vi.fn(),
  getPipelineStatus: vi.fn(),
  checkWineStatus: vi.fn(),
}));

vi.mock("../../lib/api", () => mocks);

export const mockRunPipeline = mocks.runPipeline;
export const mockGetPipelineStatus = mocks.getPipelineStatus;
export const mockCheckWineStatus = mocks.checkWineStatus;

// Create a wrapper with QueryClientProvider
export function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(
      QueryClientProvider,
      { client: queryClient },
      children,
    );
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
      Reflect.deleteProperty(store, key);
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, "localStorage", { value: localStorageMock });
