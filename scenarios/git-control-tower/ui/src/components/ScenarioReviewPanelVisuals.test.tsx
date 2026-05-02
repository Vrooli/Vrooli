import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ScreenshotsTab } from "./ScenarioReviewPanelScreenshots";
import { WorkflowsTab } from "./ScenarioReviewPanelWorkflows";
import type {
  CapturePreset,
  ExecutionMode,
  SnapshotSetMeta,
  WorkflowCaptureResult,
} from "../lib/api";

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

function workflowCapture(role: "baseline" | "capture", overrides: Partial<WorkflowCaptureResult> = {}): WorkflowCaptureResult {
  return {
    id: `${role}-workflow`,
    scenarioSlug: "git-control-tower",
    role,
    createdAt: role === "baseline" ? "2026-05-01T12:00:00Z" : "2026-05-01T12:12:00Z",
    status: "complete",
    sizeBytes: 4096,
    workflowResults: [
      {
        workflowName: role === "baseline" ? "Open dashboard" : "Review changed files",
        executionMode: "observer",
        executionId: `${role}-exec-1`,
        status: role === "baseline" ? "passed" : "failed",
        error: role === "capture" ? "Expected review panel to load" : undefined,
        durationMs: role === "baseline" ? 1100 : 2300,
        videoCount: role === "capture" ? 1 : 0,
      },
      {
        workflowName: "Commit dry run",
        executionMode: "mutating",
        executionId: `${role}-exec-2`,
        status: "skipped",
        durationMs: 0,
        videoCount: 0,
      },
    ],
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
  it("runs a baseline with the selected execution modes from the empty state", () => {
    const onBaseline = vi.fn();
    const onSelectedModesChange = vi.fn();

    render(
      <WorkflowsTab
        scenarioSlug="git-control-tower"
        basAvailable
        isRunning={false}
        onBaseline={onBaseline}
        onCapture={vi.fn()}
        onSelectedModesChange={onSelectedModesChange}
      />,
    );

    fireEvent.click(screen.getByLabelText("mutating"));
    fireEvent.click(screen.getByRole("button", { name: /set baseline/i }));

    expect(onSelectedModesChange).toHaveBeenCalledWith(["observer", "mutating"]);
    expect(onBaseline).toHaveBeenCalledWith(["observer", "mutating"]);
  });

  it("switches capture detail, expands workflow errors, and opens videos", () => {
    const onViewRoleChange = vi.fn();

    render(
      <WorkflowsTab
        baseline={workflowCapture("baseline")}
        capture={workflowCapture("capture", { status: "failed" })}
        captureStaleness={{ isStale: true }}
        scenarioSlug="git-control-tower"
        basAvailable
        isRunning={false}
        onBaseline={vi.fn()}
        onCapture={vi.fn()}
        onViewRoleChange={onViewRoleChange}
      />,
    );

    expect(screen.getByText("Review changed files")).toBeInTheDocument();
    expect(screen.getByText(/files have changed since this capture/i)).toBeInTheDocument();

    const captureSummary = screen.getByRole("button", { name: /capture.*1 failed.*1 skipped/i });
    expect(captureSummary).toHaveTextContent("Stale");
    expect(captureSummary).toHaveTextContent("Failed");

    fireEvent.click(screen.getByText("Review changed files"));

    expect(screen.getByText(/expected review panel to load/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /watch/i }));

    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
    const videoSrc = document.querySelector("video")?.getAttribute("src");
    expect(videoSrc).toContain("/repo/workflow-captures/capture-workflow/video/");
    expect(videoSrc).toContain("Review%20changed%20files");

    fireEvent.click(screen.getByRole("button", { name: /baseline.*1 passed.*1 skipped/i }));

    expect(onViewRoleChange).toHaveBeenCalledWith("baseline");
    expect(screen.getByText("Open dashboard")).toBeInTheDocument();

    const table = screen.getByRole("table");
    expect(within(table).queryByText("Review changed files")).not.toBeInTheDocument();
  });

  it("does not start workflows when all execution modes are deselected", () => {
    const onBaseline = vi.fn();

    render(
      <WorkflowsTab
        scenarioSlug="git-control-tower"
        basAvailable
        isRunning={false}
        onBaseline={onBaseline}
        onCapture={vi.fn()}
        initialSelectedModes={["observer"] satisfies ExecutionMode[]}
      />,
    );

    fireEvent.click(screen.getByLabelText("observer"));

    expect(screen.getByRole("button", { name: /set baseline/i })).toBeDisabled();
    expect(onBaseline).not.toHaveBeenCalled();
  });
});
