import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useExecutionDetailData } from "./useExecutionDetailData";
import type { ExecutionRecord, Finalization } from "../types";

// --- Mocks ---
vi.mock("../services", () => ({
  executionService: {
    get: vi.fn(),
    cancel: vi.fn(),
    retry: vi.fn(),
    triggerReview: vi.fn(),
  },
  promptService: {
    getExecutionPromptTrace: vi.fn(),
  },
}));

vi.mock("../services/review-service", () => ({
  reviewService: {
    listRounds: vi.fn(),
  },
}));

vi.mock("./useActivityTimeline", () => ({
  useActivityTimeline: vi.fn().mockReturnValue({
    entries: [],
    isLoading: false,
    error: null,
  }),
}));

const { executionService, promptService } = await import("../services") as unknown as {
  executionService: {
    get: ReturnType<typeof vi.fn>;
    cancel: ReturnType<typeof vi.fn>;
    retry: ReturnType<typeof vi.fn>;
    triggerReview: ReturnType<typeof vi.fn>;
  };
  promptService: {
    getExecutionPromptTrace: ReturnType<typeof vi.fn>;
  };
};

const { reviewService } = await import("../services/review-service") as unknown as {
  reviewService: {
    listRounds: ReturnType<typeof vi.fn>;
  };
};

// --- Factories ---
const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "fix",
  backlogName: "test-bug",
  status: "completed",
  mode: "manual",
  createdAt: "2026-03-20T00:00:00Z",
  updatedAt: "2026-03-20T01:00:00Z",
  ...overrides,
});

const makeFinalization = (overrides?: Partial<Finalization>): Finalization => ({
  eligible: true,
  status: "completed",
  phase: "completed",
  scopeSource: "sandbox_diff",
  warnings: [],
  affectedScenarios: ["swarm-manager"],
  aggregateClassification: "ready",
  scenarios: [],
  ...overrides,
});

// --- Test helpers ---
function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
      mutations: { retry: false },
    },
  });
}

function createWrapper() {
  const queryClient = createTestQueryClient();
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  executionService.get.mockResolvedValue(makeExecution());
  promptService.getExecutionPromptTrace.mockResolvedValue({
    purpose: "test",
    prompt: "do the thing",
    used_fallback: false,
    captured_at: "2026-03-20T00:00:00Z",
  });
  reviewService.listRounds.mockResolvedValue([]);
});

describe("useExecutionDetailData", () => {
  it("loads execution data", async () => {
    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.execution).toBeDefined();
    });

    expect(result.current.execution!.executionId).toBe("exec-1");
    expect(executionService.get).toHaveBeenCalledWith("exec-1");
  });

  it("fetches prompt trace when execution is available", async () => {
    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.trace).not.toBeNull();
    });

    expect(result.current.trace!.purpose).toBe("test");
    expect(promptService.getExecutionPromptTrace).toHaveBeenCalledWith("exec-1");
  });

  it("fetches review rounds using execution backlog kind/name", async () => {
    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.execution).toBeDefined();
    });

    await waitFor(() => {
      expect(reviewService.listRounds).toHaveBeenCalledWith("fix", "test-bug");
    });
  });

  it("computes isActive correctly for running status", async () => {
    executionService.get.mockResolvedValue(makeExecution({ status: "running" }));

    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.execution).toBeDefined();
    });

    expect(result.current.isActive).toBe(true);
    expect(result.current.isTerminal).toBe(false);
  });

  it("computes isTerminal correctly for completed status", async () => {
    executionService.get.mockResolvedValue(makeExecution({ status: "completed" }));

    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.execution).toBeDefined();
    });

    expect(result.current.isActive).toBe(false);
    expect(result.current.isTerminal).toBe(true);
  });

  it("computes postRunBadgeExecution when finalization exists", async () => {
    const finalization = makeFinalization();
    executionService.get.mockResolvedValue(makeExecution({ finalization }));

    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.postRunBadgeExecution).not.toBeNull();
    });

    expect(result.current.postRunBadgeExecution!.finalization).toBe(finalization);
  });

  it("computes postRunBadgeExecution with synthetic for validating", async () => {
    executionService.get.mockResolvedValue(makeExecution({ status: "validating" }));

    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.postRunBadgeExecution).not.toBeNull();
    });

    expect(result.current.postRunBadgeExecution!.finalization!.status).toBe("running");
  });

  it("returns null postRunBadgeExecution for completed without finalization", async () => {
    executionService.get.mockResolvedValue(makeExecution({ status: "completed" }));

    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.execution).toBeDefined();
    });

    expect(result.current.postRunBadgeExecution).toBeNull();
  });

  it("extracts targetScenarios from finalization", async () => {
    executionService.get.mockResolvedValue(
      makeExecution({
        finalization: makeFinalization({ affectedScenarios: ["app-a", "app-b"] }),
      }),
    );

    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.targetScenarios).toEqual(["app-a", "app-b"]);
    });
  });

  it("detects gathering evidence from review rounds", async () => {
    reviewService.listRounds.mockResolvedValue([
      { round: 1, status: "gathering", evidence: [], generated_at: "", execution_id: "exec-1" },
    ]);

    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.isGatheringEvidence).toBe(true);
    });
  });

  it("does not fetch when executionId is undefined", () => {
    renderHook(
      () => useExecutionDetailData({ executionId: undefined }),
      { wrapper: createWrapper() },
    );

    expect(executionService.get).not.toHaveBeenCalled();
    expect(promptService.getExecutionPromptTrace).not.toHaveBeenCalled();
  });

  it("returns loading state initially", () => {
    executionService.get.mockReturnValue(new Promise(() => {})); // never resolves

    const { result } = renderHook(
      () => useExecutionDetailData({ executionId: "exec-1" }),
      { wrapper: createWrapper() },
    );

    expect(result.current.isLoading).toBe(true);
    expect(result.current.execution).toBeUndefined();
  });
});
