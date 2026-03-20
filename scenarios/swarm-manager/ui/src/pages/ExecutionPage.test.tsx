import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ExecutionPage } from "./ExecutionPage";
import { useExecutionStore } from "../stores";

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
}));

import { executionService, promptService } from "../services";
import type { PromptTrace } from "../types";

describe("ExecutionPage", () => {
  const mockPromptTrace: PromptTrace = {
    skill_id: "swarm-manager-process-execute",
    purpose: "Execution trace",
    prompt: "Execution prompt",
    used_fallback: false,
    captured_at: "2026-02-12T10:00:00.000Z",
  };

  beforeEach(() => {
    vi.clearAllMocks();
    useExecutionStore.getState().reset();
    vi.mocked(promptService.getExecutionPromptTrace).mockResolvedValue(mockPromptTrace);
  });

  afterEach(() => {
    useExecutionStore.getState().reset();
  });

  it("renders the execution page with tabs and controls", async () => {
    vi.mocked(executionService.list).mockResolvedValue([]);

    render(<MemoryRouter><ExecutionPage /></MemoryRouter>);

    expect(screen.getByTestId("execution-page")).toBeInTheDocument();
    expect(screen.getByTestId("execution-tabs")).toBeInTheDocument();
    expect(screen.getByTestId("execution-search")).toBeInTheDocument();
    expect(screen.getByTestId("execution-filter")).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByTestId("execution-empty")).toBeInTheDocument();
    });
  });

  it("renders run cards when data exists", async () => {
    vi.mocked(executionService.list).mockResolvedValue([
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

    render(<MemoryRouter><ExecutionPage /></MemoryRouter>);

    await waitFor(() => {
      expect(screen.getByTestId("execution-grid")).toBeInTheDocument();
    });

    expect(screen.getAllByText("Execute: deploy-health-check").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Running").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Cancel").length).toBeGreaterThanOrEqual(1);
  });
});
