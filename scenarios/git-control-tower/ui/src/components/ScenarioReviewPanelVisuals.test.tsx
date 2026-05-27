import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ScreenshotsTab } from "./ScenarioReviewPanelScreenshots";
import { WorkflowsTab } from "./ScenarioReviewPanelWorkflows";
import { renderWithQueryClient } from "../test-utils";
import { listRecentRuns, getRunDetail } from "../lib/api-workflowreplay";
import type { RunSummary, GetRunDetailResponse } from "../lib/api-workflowreplay";
import type { CapturePreset, SnapshotSetMeta } from "../lib/api";

// The Workflows tab reads runs through the WorkflowReplayService client; mock
// the thin api wrappers so tests drive the component deterministically.
vi.mock("../lib/api-workflowreplay", () => ({
  listRecentRuns: vi.fn(),
  getRunDetail: vi.fn(),
  workflowVideoUrl: (scenario: string, runId: string, relPath: string) =>
    `/api/v1/repo/workflow-runs/${runId}/video?scenario=${scenario}&path=${relPath}`,
}));

function run(overrides: Partial<RunSummary> = {}): RunSummary {
  return {
    runId: "run-123",
    status: "passed",
    startedAt: "2026-05-26T12:00:00Z",
    completedAt: "2026-05-26T12:01:00Z",
    gitSha: "abc12345def",
    gitBranch: "agi",
    gitDirty: false,
    playbooksStatus: "passed",
    playbooksDurationSeconds: 12,
    ...overrides,
  } as unknown as RunSummary;
}

const desktopPreset: CapturePreset = {
  name: "Desktop Light",
  width: 1440,
  height: 900,
  theme: "light",
};

const mobilePreset: CapturePreset = {
  name: "Mobile Dark",
  width: 390,
  height: 844,
  theme: "dark",
};

function snapshot(role: "baseline" | "capture", overrides: Partial<SnapshotSetMeta> = {}): SnapshotSetMeta {
  return {
    id: `${role}-snap`,
    scenarioSlug: "git-control-tower",
    role,
    triggerType: "manual",
    pages: ["/", "/settings"],
    screenshotCount: 4,
    videoCount: 0,
    createdAt: role === "baseline" ? "2026-05-01T12:00:00Z" : "2026-05-01T12:10:00Z",
    sizeBytes: 2048,
    status: "complete",
    presets: [desktopPreset, mobilePreset],
    pageDiscoveryMethod: "lighthouse",
    ...overrides,
  };
}

describe("ScreenshotsTab", () => {
  it("shows the unavailable empty state without browser automation", () => {
    render(
      <ScreenshotsTab
        scenarioSlug="git-control-tower"
        isMobile={false}
        basAvailable={false}
        isCapturing={false}
        onBaseline={vi.fn()}
        onCapture={vi.fn()}
        presetConfig={[desktopPreset]}
        onPresetConfigChange={vi.fn()}
      />,
    );

    expect(screen.getByText("No captures yet")).toBeInTheDocument();
    expect(screen.getByText(/start browser-automation-studio/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /set baseline/i })).not.toBeInTheDocument();
  });

  it("switches captured presets and pages while preserving caller state", () => {
    const onBaseline = vi.fn();
    const onCapture = vi.fn();
    const onPresetIndexChange = vi.fn();
    const onSelectedPageChange = vi.fn();
    const onAttachToAgent = vi.fn();

    render(
      <ScreenshotsTab
        baseline={snapshot("baseline")}
        capture={snapshot("capture")}
        captureStaleness={{ isStale: true }}
        scenarioSlug="git-control-tower"
        isMobile={false}
        basAvailable
        isCapturing={false}
        onBaseline={onBaseline}
        onCapture={onCapture}
        presetConfig={[desktopPreset, mobilePreset]}
        onPresetConfigChange={vi.fn()}
        agentManagerAvailable
        onAttachToAgent={onAttachToAgent}
        onPresetIndexChange={onPresetIndexChange}
        onSelectedPageChange={onSelectedPageChange}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Mobile Dark" }));
    fireEvent.click(screen.getByRole("button", { name: "/settings" }));

    expect(onPresetIndexChange).toHaveBeenCalledWith(1);
    expect(onSelectedPageChange).toHaveBeenCalledWith(1);
    expect(screen.getByText(/files have changed since this capture/i)).toBeInTheDocument();
    expect(screen.getByText("Page:")).toHaveTextContent("Page: /settings");

    fireEvent.click(screen.getByRole("button", { name: /reset baseline/i }));
    fireEvent.click(screen.getByRole("button", { name: /^re-capture$/i }));

    expect(onBaseline).toHaveBeenCalledOnce();
    expect(onCapture).toHaveBeenCalledOnce();

    const attachButtons = screen.getAllByText("+ Agent");
    const baselineAttachButton = attachButtons[0];
    if (!baselineAttachButton) {
      throw new Error("Expected screenshot attach button to render");
    }
    fireEvent.click(baselineAttachButton);

    expect(onAttachToAgent).toHaveBeenCalledWith(
      expect.objectContaining({
        kind: "screenshot",
        label: "Screenshot: /settings",
        markdown: expect.stringContaining("390x844"),
      }),
    );
  });
});

describe("WorkflowsTab", () => {
  it("shows the empty state and routes 'Set baseline' to the Baselines tab", async () => {
    vi.mocked(listRecentRuns).mockResolvedValue([]);
    const onOpenBaselines = vi.fn();

    renderWithQueryClient(
      <WorkflowsTab
        scenarioSlug="git-control-tower"
        repoId={null}
        testGenieAvailable
        onOpenBaselines={onOpenBaselines}
      />,
    );

    expect(await screen.findByText(/no playbooks runs yet/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /set baseline/i }));
    expect(onOpenBaselines).toHaveBeenCalled();
  });

  it("reports when test-genie is unavailable", () => {
    renderWithQueryClient(
      <WorkflowsTab
        scenarioSlug="git-control-tower"
        repoId={null}
        testGenieAvailable={false}
        onOpenBaselines={vi.fn()}
      />,
    );
    expect(screen.getByText(/test-genie is not available/i)).toBeInTheDocument();
    expect(listRecentRuns).not.toHaveBeenCalled();
  });

  it("lists playbooks runs and opens a recorded video on expand", async () => {
    vi.mocked(listRecentRuns).mockResolvedValue([run({ runId: "run-abc" })]);
    vi.mocked(getRunDetail).mockResolvedValue({
      run: run({ runId: "run-abc" }),
      videos: [{ workflow: "login-smoke", relPath: "automation/login-smoke/video/a.webm", sizeBytes: 10 }],
    } as unknown as GetRunDetailResponse);

    renderWithQueryClient(
      <WorkflowsTab
        scenarioSlug="git-control-tower"
        repoId={null}
        testGenieAvailable
        onOpenBaselines={vi.fn()}
      />,
    );

    // Run row renders once the list resolves.
    fireEvent.click(await screen.findByText("run-abc"));

    // Expanding fetches detail and renders a Watch button per video.
    fireEvent.click(await screen.findByRole("button", { name: /login-smoke/i }));

    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
    const videoSrc = document.querySelector("video")?.getAttribute("src");
    expect(videoSrc).toContain("/repo/workflow-runs/run-abc/video");
    expect(videoSrc).toContain("automation/login-smoke/video/a.webm");
  });
});
