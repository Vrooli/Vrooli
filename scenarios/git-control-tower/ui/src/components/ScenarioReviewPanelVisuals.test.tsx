import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
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
  workflowVideoUrl: (scenario: string, runId: string, artifactId: string) =>
    `/api/v1/repo/workflow-runs/${runId}/video?scenario=${scenario}&artifact_id=${artifactId}`,
}));

// Baselines come from BaselinesService; the surface bar/selector list them. The
// tests below exercise the loose-capture paths, so an empty baseline list is
// the realistic default.
vi.mock("../lib/api-baselines", async () => {
  const actual = await vi.importActual<typeof import("../lib/api-baselines")>("../lib/api-baselines");
  return { ...actual, listBaselines: vi.fn().mockResolvedValue([]) };
});

beforeEach(() => window.localStorage.clear());

function run(overrides: Partial<RunSummary> = {}): RunSummary {
  return {
    runId: "run-123",
    status: "passed",
    startedAt: "2026-05-26T12:00:00Z",
    completedAt: "2026-05-26T12:01:00Z",
    gitSha: "abc12345def",
    gitBranch: "agi",
    gitDirty: false,
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

function capture(overrides: Partial<SnapshotSetMeta> = {}): SnapshotSetMeta {
  return {
    id: "capture-snap",
    scenarioSlug: "git-control-tower",
    role: "capture",
    triggerType: "manual",
    pages: ["/", "/settings"],
    screenshotCount: 4,
    videoCount: 0,
    createdAt: "2026-05-01T12:10:00Z",
    sizeBytes: 2048,
    status: "complete",
    presets: [desktopPreset, mobilePreset],
    pageDiscoveryMethod: "lighthouse",
    ...overrides,
  };
}

describe("ScreenshotsTab", () => {
  it("shows the service message (no baseline vocabulary) when browser automation is unavailable", () => {
    renderWithQueryClient(
      <ScreenshotsTab
        scenarioSlug="git-control-tower"
        isMobile={false}
        basAvailable={false}
        isCapturing={false}
        onCapture={vi.fn()}
        onOpenBaselines={vi.fn()}
        presetConfig={[desktopPreset]}
        onPresetConfigChange={vi.fn()}
      />,
    );

    expect(screen.getByText("No visuals captured yet")).toBeInTheDocument();
    expect(screen.getByText(/start browser-automation-studio/i)).toBeInTheDocument();
    // Decision 1: no "set/reset baseline" vocabulary for screenshot snapshots.
    expect(screen.queryByRole("button", { name: /set baseline/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /reset baseline/i })).not.toBeInTheDocument();
  });

  it("offers two capture intents (loose vs baseline) when nothing is captured", () => {
    const onCapture = vi.fn();
    renderWithQueryClient(
      <ScreenshotsTab
        scenarioSlug="git-control-tower"
        isMobile={false}
        basAvailable
        isCapturing={false}
        onCapture={onCapture}
        onOpenBaselines={vi.fn()}
        presetConfig={[desktopPreset]}
        onPresetConfigChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /capture screenshots/i }));
    expect(onCapture).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: /capture baseline/i })).toBeInTheDocument();
  });

  it("renders the current capture and switches presets/pages while preserving caller state", () => {
    const onCapture = vi.fn();
    const onPresetIndexChange = vi.fn();
    const onSelectedPageChange = vi.fn();
    const onAttachToAgent = vi.fn();

    renderWithQueryClient(
      <ScreenshotsTab
        capture={capture()}
        captureStaleness={{ isStale: true }}
        scenarioSlug="git-control-tower"
        isMobile={false}
        basAvailable
        isCapturing={false}
        onCapture={onCapture}
        onOpenBaselines={vi.fn()}
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
    // Decision 1: even with data present, no "set/reset baseline" snapshot vocabulary.
    expect(screen.queryByRole("button", { name: /set baseline/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /reset baseline/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /^re-capture$/i }));
    expect(onCapture).toHaveBeenCalledOnce();

    const attachButtons = screen.getAllByText("+ Agent");
    const attachButton = attachButtons[0];
    if (!attachButton) {
      throw new Error("Expected screenshot attach button to render");
    }
    fireEvent.click(attachButton);

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
  it("shows the two-action empty state (loose run + capture baseline)", async () => {
    vi.mocked(listRecentRuns).mockResolvedValue([]);

    renderWithQueryClient(
      <WorkflowsTab
        scenarioSlug="git-control-tower"
        repoId={null}
        testGenieAvailable
        onOpenBaselines={vi.fn()}
      />,
    );

    expect(await screen.findByText("No workflows captured yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /capture workflow evidence/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /capture baseline/i })).toBeInTheDocument();
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

  it("lists runs with typed workflow evidence and opens a recording by opaque id", async () => {
    vi.mocked(listRecentRuns).mockResolvedValue([run({ runId: "run-abc" })]);
    vi.mocked(getRunDetail).mockResolvedValue({
      run: run({ runId: "run-abc" }),
      artifacts: [{ id: "opaque-video-id", kind: "workflow.video", label: "login-smoke", sizeBytes: 10n }],
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
    expect(videoSrc).toContain("artifact_id=opaque-video-id");
    expect(videoSrc).not.toContain("automation/login-smoke/video/a.webm");
  });
});
