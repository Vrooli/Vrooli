/**
 * Shared test utilities for useScenarioState tests.
 */

import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import type { UseScenarioStateOptions } from "../useScenarioState";
import type {
  LoadStateResponse,
  SaveStateResponse,
  ScenarioState,
} from "../../lib/api";

const mocks = vi.hoisted(() => ({
  fetchScenarioState: vi.fn(),
  saveScenarioState: vi.fn(),
  deleteScenarioState: vi.fn(),
  checkStateStaleness: vi.fn(),
}));

vi.mock("../../lib/api", () => mocks);

export const mockFetchScenarioState = mocks.fetchScenarioState;
export const mockSaveScenarioState = mocks.saveScenarioState;
export const mockDeleteScenarioState = mocks.deleteScenarioState;
export const mockCheckStateStaleness = mocks.checkStateStaleness;

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

// Helper to create a ScenarioState for testing
export function createMockScenarioState(
  overrides: Partial<ScenarioState> = {},
): ScenarioState {
  return {
    scenario_name: "test-scenario",
    schema_version: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
    hash: "abc123",
    form_state: {
      app_display_name: "Test App",
      app_description: "Test Description",
      deployment_mode: "bundled",
      framework: "electron",
    },
    ...overrides,
  };
}

// Helper to create LoadStateResponse
export function createLoadStateResponse(
  state: ScenarioState | null,
  overrides: Partial<LoadStateResponse> = {},
): LoadStateResponse {
  return {
    state,
    found: state !== null,
    ...overrides,
  };
}

// Helper to create SaveStateResponse
export function createSaveStateResponse(
  overrides: Partial<SaveStateResponse> = {},
): SaveStateResponse {
  return {
    success: true,
    updated_at: new Date().toISOString(),
    hash: "newhash123",
    ...overrides,
  };
}

export const defaultOptions: UseScenarioStateOptions = {
  scenarioName: "test-scenario",
  enabled: true,
  checkStaleness: false, // Disable by default to simplify tests
};
