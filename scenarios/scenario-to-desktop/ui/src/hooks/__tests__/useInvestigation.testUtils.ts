/**
 * Shared test utilities for useInvestigation tests.
 */

import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import type {
  Investigation,
  InvestigationSummary,
} from "../../types/investigation";

// Mock the API module
vi.mock("../../lib/api", () => ({
  getAgentManagerStatus: vi.fn(),
  createTask: vi.fn(),
  listTasks: vi.fn(),
  getTask: vi.fn(),
  stopTask: vi.fn(),
}));

// Import mocks after setting up vi.mock
import {
  getAgentManagerStatus,
  createTask,
  listTasks,
  getTask,
  stopTask,
} from "../../lib/api";

export const mockGetAgentManagerStatus = getAgentManagerStatus as ReturnType<typeof vi.fn>;
export const mockCreateTask = createTask as ReturnType<typeof vi.fn>;
export const mockListTasks = listTasks as ReturnType<typeof vi.fn>;
export const mockGetTask = getTask as ReturnType<typeof vi.fn>;
export const mockStopTask = stopTask as ReturnType<typeof vi.fn>;

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

// Helper to create mock investigation
export function createMockInvestigation(overrides: Partial<Investigation> = {}): Investigation {
  return {
    id: "task-123",
    pipeline_id: "pipeline-456",
    status: "completed",
    findings: "Found 2 issues",
    progress: 100,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T01:00:00Z",
    ...overrides,
  };
}

// Helper to create mock investigation summary
export function createMockInvestigationSummary(
  overrides: Partial<InvestigationSummary> = {}
): InvestigationSummary {
  return {
    id: "task-123",
    pipeline_id: "pipeline-456",
    status: "completed",
    task_type: "investigate",
    progress: 100,
    created_at: "2024-01-01T00:00:00Z",
    ...overrides,
  };
}
