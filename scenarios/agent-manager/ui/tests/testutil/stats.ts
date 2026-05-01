import type {
  ModelBreakdownResponse,
  ModelUsageRunsResponse,
  ProfileBreakdownResponse,
  RunnerBreakdownResponse,
  SummaryResponse,
  ToolUsageModelsResponse,
  ToolUsageResponse,
  ToolUsageRunsResponse,
} from "../../src/features/stats/api/types.js";

export function makeSummaryResponse(overrides: Partial<SummaryResponse["summary"]> = {}): SummaryResponse {
  return {
    summary: {
      statusCounts: {
        pending: 3,
        running: 2,
        complete: 17,
        failed: 2,
        cancelled: 1,
        needsReview: 0,
        total: 25,
      },
      successRate: 0.875,
      duration: {
        avgMs: 90_000,
        p50Ms: 60_000,
        p95Ms: 180_000,
        p99Ms: 240_000,
        minMs: 10_000,
        maxMs: 300_000,
        count: 20,
      },
      cost: {
        totalCostUsd: 12.345,
        avgCostUsd: 0.61725,
        inputTokens: 12_000,
        outputTokens: 8_000,
        cacheReadTokens: 1_000,
        totalTokens: 21_000,
      },
      runnerBreakdown: [
        {
          runnerType: "claude-code",
          runCount: 24,
          successCount: 21,
          failedCount: 3,
          totalCostUsd: 12.345,
          avgDurationMs: 90_000,
        },
      ],
      ...overrides,
    },
  };
}

export function makeModelBreakdownResponse(
  overrides: Partial<ModelBreakdownResponse> = {},
): ModelBreakdownResponse {
  return {
    models: [
      {
        model: "claude-3-opus",
        runCount: 12,
        successCount: 9,
        totalCostUsd: 3.25,
        totalTokens: 42_000,
      },
      {
        model: "gpt-5-mini",
        runCount: 4,
        successCount: 4,
        totalCostUsd: 0.75,
        totalTokens: 9_500,
      },
    ],
    ...overrides,
  };
}

export function makeRunnerBreakdownResponse(
  overrides: Partial<RunnerBreakdownResponse> = {},
): RunnerBreakdownResponse {
  return {
    runners: [
      {
        runnerType: "codex",
        runCount: 18,
        successCount: 16,
        failedCount: 2,
        totalCostUsd: 8.25,
        avgDurationMs: 90_000,
      },
      {
        runnerType: "claude-code",
        runCount: 7,
        successCount: 7,
        failedCount: 0,
        totalCostUsd: 2.5,
        avgDurationMs: 45_000,
      },
      {
        runnerType: "opencode",
        runCount: 4,
        successCount: 2,
        failedCount: 2,
        totalCostUsd: 0.75,
        avgDurationMs: 15_000,
      },
    ],
    ...overrides,
  };
}

export function makeProfileBreakdownResponse(
  overrides: Partial<ProfileBreakdownResponse> = {},
): ProfileBreakdownResponse {
  return {
    profiles: [
      {
        profileId: "profile-maintenance",
        profileName: "Maintenance Agent",
        runCount: 14,
        successCount: 12,
        failedCount: 2,
        totalCostUsd: 7.5,
      },
      {
        profileId: "profile-implementation",
        profileName: "Implementation Agent",
        runCount: 9,
        successCount: 9,
        failedCount: 0,
        totalCostUsd: 4.25,
      },
      {
        profileId: "profile-audit",
        profileName: "Audit Agent",
        runCount: 3,
        successCount: 1,
        failedCount: 2,
        totalCostUsd: 0.5,
      },
    ],
    ...overrides,
  };
}

export function makeModelUsageRunsResponse(
  overrides: Partial<ModelUsageRunsResponse> = {},
): ModelUsageRunsResponse {
  return {
    runs: [
      {
        runId: "run-model-12345678",
        taskId: "task-1",
        taskTitle: "Audit dependency graph",
        profileId: "profile-1",
        profileName: "Maintenance Agent",
        status: "complete",
        createdAt: "2026-05-01T14:00:00.000Z",
        totalCostUsd: 1.5,
        totalTokens: 18_500,
      },
    ],
    ...overrides,
  };
}

export function makeToolUsageResponse(overrides: Partial<ToolUsageResponse> = {}): ToolUsageResponse {
  return {
    tools: [
      {
        toolName: "Edit",
        callCount: 20,
        successCount: 18,
        failedCount: 2,
      },
      {
        toolName: "Read",
        callCount: 8,
        successCount: 8,
        failedCount: 0,
      },
    ],
    ...overrides,
  };
}

export function makeToolUsageRunsResponse(
  overrides: Partial<ToolUsageRunsResponse> = {},
): ToolUsageRunsResponse {
  return {
    runs: [
      {
        runId: "run-tool-12345678",
        taskId: "task-2",
        taskTitle: "Patch orchestration tests",
        profileId: "profile-2",
        profileName: "Implementation Agent",
        status: "failed",
        createdAt: "2026-05-01T15:00:00.000Z",
        model: "claude-3-opus",
        callCount: 7,
        successCount: 5,
        failedCount: 2,
      },
    ],
    ...overrides,
  };
}

export function makeToolUsageModelsResponse(
  overrides: Partial<ToolUsageModelsResponse> = {},
): ToolUsageModelsResponse {
  return {
    models: [
      {
        model: "claude-3-opus",
        runCount: 6,
        callCount: 12,
        successCount: 10,
        failedCount: 2,
      },
      {
        model: "gpt-5-mini",
        runCount: 3,
        callCount: 8,
        successCount: 8,
        failedCount: 0,
      },
    ],
    ...overrides,
  };
}
