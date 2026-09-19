/**
 * Shared test utilities for useInvestigation tests.
 */

import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { create } from "@bufbuild/protobuf";
import {
  InvestigationSchema,
  InvestigationStatus,
  InvestigationSummarySchema,
  type Investigation,
  type InvestigationSummary,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/tasks_pb";

type LegacyStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";
type InvestigationOverrides = Omit<
  Partial<Investigation>,
  "status" | "details" | "$typeName" | "$unknown"
> & {
  status?: InvestigationStatus | LegacyStatus;
  details?: unknown;
};
type InvestigationSummaryOverrides = Omit<
  Partial<InvestigationSummary>,
  "status" | "$typeName" | "$unknown"
> & {
  status?: InvestigationStatus | LegacyStatus;
  task_type?: string;
  created_at?: string;
};

const investigationStatus = (
  status: InvestigationStatus | LegacyStatus | undefined,
) => {
  if (typeof status === "number") return status;
  return {
    pending: InvestigationStatus.PENDING,
    running: InvestigationStatus.RUNNING,
    completed: InvestigationStatus.COMPLETED,
    failed: InvestigationStatus.FAILED,
    cancelled: InvestigationStatus.CANCELLED,
  }[status ?? "completed"];
};

const mocks = vi.hoisted(() => ({
  getAgentManagerStatus: vi.fn(),
  createTask: vi.fn(),
  listTasks: vi.fn(),
  getTask: vi.fn(),
  stopTask: vi.fn(),
}));

vi.mock("../../lib/api", () => mocks);

export const mockGetAgentManagerStatus = mocks.getAgentManagerStatus;
export const mockCreateTask = mocks.createTask;
export const mockListTasks = mocks.listTasks;
export const mockGetTask = mocks.getTask;
export const mockStopTask = mocks.stopTask;

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

// Helper to create mock investigation
export function createMockInvestigation(
  overrides: InvestigationOverrides = {},
): Investigation {
  const { status, details, ...rest } = overrides;
  return create(InvestigationSchema, {
    id: "task-123",
    pipelineId: "pipeline-456",
    status: investigationStatus(status),
    findings: "Found 2 issues",
    progress: 100,
    createdAt: { seconds: 1704067200n, nanos: 0 },
    updatedAt: { seconds: 1704070800n, nanos: 0 },
    details: details as Investigation["details"],
    ...rest,
  });
}

// Helper to create mock investigation summary
export function createMockInvestigationSummary(
  overrides: InvestigationSummaryOverrides = {},
): InvestigationSummary {
  const {
    status,
    task_type: _taskType,
    created_at: _createdAt,
    ...rest
  } = overrides;
  return create(InvestigationSummarySchema, {
    id: "task-123",
    pipelineId: "pipeline-456",
    status: investigationStatus(status),
    progress: 100,
    createdAt: { seconds: 1704067200n, nanos: 0 },
    ...rest,
  });
}
