import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, cleanup, screen, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { ExecutionPage } from "./ExecutionPage";
import { useExecutionStore } from "../stores";
import { createTestQueryClient, renderWithProviders } from "../test-utils";

vi.mock("../services", () => ({
  executionService: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    start: vi.fn(),
    cancel: vi.fn(),
    retry: vi.fn(),
  },
  promptService: {
    getExecutionPromptTrace: vi.fn(),
  },
  agentManagerService: {
    getStatus: vi.fn().mockResolvedValue({ available: false }),
  },
  embeddedService: {
    getExternalUrl: vi.fn(() => new Promise(() => undefined)),
  },
}));

vi.mock("../services/gct-service", () => ({
  gctService: {
    getStatus: vi.fn(() => new Promise(() => undefined)),
  },
}));

import { executionService, promptService } from "../services";
import type { ExecutionRecord, PromptTrace } from "../types";

describe("ExecutionPage", () => {
  let queryClient: QueryClient;

  const mockPromptTrace: PromptTrace = {
    purpose: "Execution trace",
    prompt: "Execution prompt",
    used_fallback: false,
    captured_at: "2026-02-12T10:00:00.000Z",
  };

  beforeEach(() => {
    vi.clearAllMocks();
    useExecutionStore.getState().reset();
    queryClient = createTestQueryClient();
    vi.mocked(promptService.getExecutionPromptTrace).mockResolvedValue(mockPromptTrace);
  });

  afterEach(() => {
    cleanup();
    useExecutionStore.getState().reset();
    queryClient.clear();
  });

  const renderPage = async (executions: ExecutionRecord[] = []) => {
    let resolveExecutions: (value: ExecutionRecord[]) => void = () => undefined;
    vi.mocked(executionService.list).mockReturnValue(
      new Promise((resolve) => {
        resolveExecutions = resolve;
      }),
    );

    let result: ReturnType<typeof renderWithProviders> | undefined;
    await act(async () => {
      result = renderWithProviders(<ExecutionPage />, { queryClient });
    });
    await act(async () => {
      resolveExecutions(executions);
    });
    if (!result) {
      throw new Error("ExecutionPage render did not complete.");
    }
    return result;
  };

  it("renders the execution page with tabs and controls", async () => {
    await renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("execution-page")).toBeInTheDocument();
      expect(screen.getByTestId("execution-tabs")).toBeInTheDocument();
      expect(screen.getByTestId("execution-search")).toBeInTheDocument();
      expect(screen.getByTestId("execution-filter")).toBeInTheDocument();
      expect(screen.getByTestId("execution-empty")).toBeInTheDocument();
    });
  });

  it("renders run cards when data exists", async () => {
    await renderPage([
      {
        executionId: "exec_123",
        backlogKind: "execute",
        backlogName: "deploy-health-check",
        status: "running",
        mode: "manual",
        operation: "improver",
        startedBy: "swarm-manager-ui",
        runId: "run_456",
        taskId: "task_789",
        createdAt: "2026-02-12T10:00:00.000Z",
        updatedAt: "2026-02-12T10:05:00.000Z",
      },
    ]);

    await waitFor(() => {
      expect(screen.getByTestId("execution-grid")).toBeInTheDocument();
      expect(screen.getAllByText("Execute: deploy-health-check").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Running").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Cancel").length).toBeGreaterThanOrEqual(1);
    });
  });
});
