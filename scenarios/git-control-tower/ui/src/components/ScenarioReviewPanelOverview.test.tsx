import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { OverviewTab } from "./ScenarioReviewPanelOverview";
import { renderWithQueryClient } from "../test-utils";
import type { AgentContextItem, RepoFileStats, SnapshotSetMeta, TestExecutionResult } from "../lib/api";

const mocks = vi.hoisted(() => ({
  useTestExecutions: vi.fn(),
  useTidinessScore: vi.fn(),
  useTidinessStaleness: vi.fn(),
  useScenarios: vi.fn(),
  useReviewSummary: vi.fn(),
  useTriggerReviewRun: vi.fn(),
  useReviewJobStatus: vi.fn(),
  fetchExternalUrl: vi.fn(),
}));

vi.mock("../lib/hooks", () => ({
  useTestExecutions: mocks.useTestExecutions,
  useTidinessScore: mocks.useTidinessScore,
  useTidinessStaleness: mocks.useTidinessStaleness,
  useScenarios: mocks.useScenarios,
  useReviewSummary: mocks.useReviewSummary,
  useTriggerReviewRun: mocks.useTriggerReviewRun,
  useReviewJobStatus: mocks.useReviewJobStatus,
}));

vi.mock("../lib/api-internals", () => ({
  fetchExternalUrl: mocks.fetchExternalUrl,
}));

function latestTest(overrides: Partial<TestExecutionResult> = {}): TestExecutionResult {
  return {
    executionId: "exec-1",
    scenarioName: "git-control-tower",
    success: false,
    startedAt: "2026-05-01T12:00:00Z",
    completedAt: "2026-05-01T12:01:00Z",
    phases: [
      {
        name: "smoke",
        status: "failed",
        durationSeconds: 8,
        error: "Iframe bridge never signaled ready",
        remediation: "Start the UI before smoke.",
      },
    ],
    phaseSummary: {
      total: 2,
      passed: 1,
      failed: 1,
      durationSeconds: 60,
      observationCount: 0,
    },
    ...overrides,
  };
}

function snapshot(overrides: Partial<SnapshotSetMeta> = {}): SnapshotSetMeta {
  return {
    id: "capture-1",
    scenarioSlug: "git-control-tower",
    role: "capture",
    triggerType: "manual",
    pages: ["/"],
    screenshotCount: 2,
    videoCount: 0,
    createdAt: "2026-05-01T12:00:00Z",
    sizeBytes: 2048,
    status: "complete",
    presets: [],
    ...overrides,
  };
}

function fileStats(): RepoFileStats {
  return {
    staged: {
      "scenarios/git-control-tower/ui/src/App.tsx": {
        additions: 10,
        deletions: 3,
        files: 1,
      },
    },
    unstaged: {
      "scenarios/git-control-tower/api/main.go": {
        additions: 2,
        deletions: 1,
        files: 1,
      },
    },
  };
}

function renderOverview(overrides: Partial<Parameters<typeof OverviewTab>[0]> = {}) {
  const onAttachToAgent = vi.fn();
  const onCapture = vi.fn();

  renderWithQueryClient(
    <OverviewTab
      scenarioSlug="git-control-tower"
      repoId="repo-1"
      basAvailable
      testGenieAvailable
      tidinessAvailable
      isCapturing={false}
      onCapture={onCapture}
      agentManagerAvailable
      onAttachToAgent={onAttachToAgent}
      {...overrides}
    />,
  );

  return { onAttachToAgent, onCapture };
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.fetchExternalUrl.mockReturnValue(new Promise(() => {}));
  mocks.useScenarios.mockReturnValue({
    data: [
      {
        name: "git-control-tower",
        display_name: "Git Control Tower",
        status: "running",
        health_status: "healthy",
      },
    ],
  });
  mocks.useTestExecutions.mockReturnValue({
    data: { items: [latestTest()], count: 1 },
    isLoading: false,
  });
  mocks.useTidinessScore.mockReturnValue({
    data: {
      scenario: "git-control-tower",
      score: 82,
      violations: 3,
    },
    isLoading: false,
    error: null,
  });
  mocks.useTidinessStaleness.mockReturnValue({
    data: {
      last_scan_at: "2026-05-01T11:00:00Z",
      is_stale: true,
      modified_files: 2,
      stale_reason: "2 files changed since the last scan",
    },
  });
  mocks.useReviewSummary.mockReturnValue({
    data: { readiness: "green" },
  });
  mocks.useTriggerReviewRun.mockReturnValue({
    mutate: vi.fn((_request, options) => options?.onSuccess?.({ jobId: "review-job-1" })),
    isPending: false,
  });
  mocks.useReviewJobStatus.mockReturnValue({ data: null });
});

describe("OverviewTab", () => {
  it("renders scenario readiness, external URL, visual status, tests, quality, and change summary", async () => {
    mocks.fetchExternalUrl.mockResolvedValue("https://git-control-tower.local/");

    renderOverview({
      capture: snapshot({
        pageDiscoveryMethod: "fallback",
      }),
      captureStaleness: {
        isStale: true,
        lastFileChange: "2026-05-01T12:30:00Z",
        captureCreatedAt: "2026-05-01T12:00:00Z",
      },
      fileStats: fileStats(),
    });

    expect(screen.getByText("Git Control Tower")).toBeInTheDocument();
    expect(screen.getByText("running")).toBeInTheDocument();
    expect(screen.getByText("Ready")).toBeInTheDocument();
    expect(screen.getByText("Change Summary")).toBeInTheDocument();
    expect(screen.getByText("12")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
    expect(screen.getByText("2 files changed")).toBeInTheDocument();
    expect(screen.getByText("Visual Status")).toBeInTheDocument();
    expect(screen.getByText("Screenshots")).toBeInTheDocument();
    expect(screen.getByText("Test Status")).toBeInTheDocument();
    expect(screen.getByText("Failed")).toBeInTheDocument();
    expect(screen.getByText("1/2 passed")).toBeInTheDocument();
    expect(screen.getByText("Code Quality")).toBeInTheDocument();
    expect(screen.getByText("Good")).toBeInTheDocument();
    expect(screen.getByText("82/100")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText(/2 files changed since the last scan/i)).toBeInTheDocument();
    expect(screen.getByText(/pages discovered via fallback/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText("https://git-control-tower.local/")).toBeInTheDocument();
    });
  });

  it("routes overview context attachments to the agent tab seam", () => {
    const { onAttachToAgent } = renderOverview({
      capture: snapshot(),
      fileStats: fileStats(),
    });

    const attachButtons = screen.getAllByText("+ Agent");
    expect(attachButtons).toHaveLength(3);
    const [changeAttach, testAttach, qualityAttach] = attachButtons;
    if (!changeAttach || !testAttach || !qualityAttach) {
      throw new Error("expected three overview agent attachment buttons");
    }
    fireEvent.click(changeAttach);
    fireEvent.click(testAttach);
    fireEvent.click(qualityAttach);

    const attachedItems = onAttachToAgent.mock.calls.map(([item]) => item as AgentContextItem);
    expect(attachedItems).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ id: expect.stringContaining("change-summary") }),
        expect.objectContaining({ kind: "test-failure" }),
        expect.objectContaining({ id: expect.stringContaining("scenario-quality") }),
      ]),
    );
  });

  it("starts a unified review run and shows in-progress check state", async () => {
    const mutate = vi.fn((_request, options) => options?.onSuccess?.({ jobId: "review-job-1" }));
    mocks.useTriggerReviewRun.mockReturnValue({
      mutate,
      isPending: false,
    });
    mocks.useReviewJobStatus.mockImplementation((jobId: string | null) => ({
      data: jobId
        ? {
            status: "running",
            checks: {
              tests: "running",
              visuals: "completed",
              quality: "pending",
            },
          }
        : null,
    }));

    renderOverview();

    fireEvent.click(screen.getByRole("button", { name: /rerun all checks/i }));

    await waitFor(() => {
      expect(mutate).toHaveBeenCalledWith(
        { scenarioName: "git-control-tower" },
        expect.objectContaining({ onSuccess: expect.any(Function) }),
      );
      expect(screen.getByText(/review run in progress/i)).toBeInTheDocument();
    });
    expect(screen.getByText("tests: running")).toBeInTheDocument();
    expect(screen.getByText("visuals: completed")).toBeInTheDocument();
    expect(screen.getByText("quality: pending")).toBeInTheDocument();
  });

  it("falls back to capability guidance when dependent services are unavailable", () => {
    mocks.useTestExecutions.mockReturnValue({ data: undefined, isLoading: false });
    mocks.useTidinessScore.mockReturnValue({ data: undefined, isLoading: false, error: null });
    mocks.useTidinessStaleness.mockReturnValue({ data: undefined });
    mocks.useReviewSummary.mockReturnValue({ data: undefined });

    renderOverview({
      basAvailable: false,
      testGenieAvailable: false,
      tidinessAvailable: false,
      agentManagerAvailable: false,
      capture: undefined,
    });

    expect(screen.getByText("No data")).toBeInTheDocument();
    expect(screen.getByText(/start browser-automation-studio/i)).toBeInTheDocument();
    expect(screen.getByText(/start test-genie/i)).toBeInTheDocument();
    expect(screen.getByText(/start tidiness-manager/i)).toBeInTheDocument();
    expect(screen.queryByText("+ Agent")).not.toBeInTheDocument();
    // Decision 1: no "set baseline" vocabulary for screenshot snapshots.
    expect(screen.queryByRole("button", { name: /set baseline/i })).not.toBeInTheDocument();
  });
});
