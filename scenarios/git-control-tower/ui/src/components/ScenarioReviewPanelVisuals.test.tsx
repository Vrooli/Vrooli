import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ScreenshotsTab } from "./ScenarioReviewPanelScreenshots";
import { WorkflowsTab } from "./ScenarioReviewPanelWorkflows";
import { ArtifactEvidenceRenderer, artifactRendererKind } from "./ArtifactEvidenceRenderer";
import { renderWithQueryClient } from "../test-utils";
import { listEvidence, startRun } from "../lib/api-evidence";
import { listBaselines } from "../lib/api-baselines";
import type { ArtifactRef, RunInfo } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";

vi.mock("../lib/api-evidence", () => ({
  listEvidence: vi.fn(),
  listRuns: vi.fn(),
  getRun: vi.fn(),
  startRun: vi.fn(),
  runArtifactUrl: (scenario: string, runId: string, artifactId: string) =>
    `/api/v1/repo/test-runs/${runId}/artifacts/${artifactId}?scenario=${scenario}`,
}));

vi.mock("../lib/api-baselines", async () => {
  const actual = await vi.importActual<typeof import("../lib/api-baselines")>("../lib/api-baselines");
  return { ...actual, listBaselines: vi.fn().mockResolvedValue([]) };
});

function evidence(kind: string, producingPhase: string, id: string, label: string): { run: RunInfo; artifact: ArtifactRef } {
  return {
    run: {
      runId: "run-abc",
      scenario: "git-control-tower",
      status: "passed",
      startedAt: "2026-07-10T12:00:00Z",
      completedAt: "2026-07-10T12:01:00Z",
      gitSha: "abc12345def",
      phases: [],
      pins: [],
      plannedPhases: [],
    },
    artifact: {
      id,
      kind,
      label,
      producingPhase,
      mediaType: kind === "workflow.video" ? "video/webm" : "image/png",
      sizeBytes: 10n,
      metadata: {},
      relationships: [],
      accessCapability: 1,
      provenance: 1,
    },
  } as unknown as { run: RunInfo; artifact: ArtifactRef };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(listBaselines).mockResolvedValue([]);
  vi.mocked(startRun).mockResolvedValue({ runId: "run-new", scenario: "git-control-tower" } as never);
  vi.mocked(listEvidence).mockResolvedValue({ items: [], total: 0, limit: 100, offset: 0, hasMore: false, degradedReasons: [] } as never);
});

describe("ScreenshotsTab", () => {
  it("uses the typed evidence empty state", async () => {
    renderWithQueryClient(<ScreenshotsTab scenarioSlug="git-control-tower" testGenieAvailable onOpenBaselines={vi.fn()} />);
    expect(await screen.findByText("No visuals captured yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /capture screenshots/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /capture baseline/i })).toBeInTheDocument();
  });

  it("[REQ:GCT-DESCRIPTOR-REVIEW-P2] renders screenshots from arbitrary producer phases and attaches opaque identity", async () => {
    vi.mocked(listEvidence).mockResolvedValue({
      items: [evidence("screenshot", "future-visual-provider", "opaque-image", "Settings page")],
      total: 1, limit: 100, offset: 0, hasMore: false, degradedReasons: [],
    } as never);
    const onAttachToAgent = vi.fn();
    renderWithQueryClient(<ScreenshotsTab scenarioSlug="git-control-tower" testGenieAvailable agentManagerAvailable onAttachToAgent={onAttachToAgent} onOpenBaselines={vi.fn()} />);
    expect(await screen.findByText("Settings page")).toBeInTheDocument();
    expect(screen.getByText("future-visual-provider")).toBeInTheDocument();
    fireEvent.click(screen.getByText("+ Agent"));
    expect(onAttachToAgent).toHaveBeenCalledWith(expect.objectContaining({ id: "artifact:run-abc:opaque-image", markdown: expect.stringContaining("future-visual-provider") }));
    expect(document.querySelector("img")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /preview/i }));
    expect(document.querySelector("img")?.getAttribute("src")).toContain("/test-runs/run-abc/artifacts/opaque-image");
  });
});

describe("WorkflowsTab", () => {
  it("reports when test-genie is unavailable", () => {
    renderWithQueryClient(<WorkflowsTab scenarioSlug="git-control-tower" testGenieAvailable={false} onOpenBaselines={vi.fn()} />);
    expect(screen.getByText(/test-genie is not available/i)).toBeInTheDocument();
    expect(listEvidence).not.toHaveBeenCalled();
  });

  it("[REQ:GCT-DESCRIPTOR-REVIEW-P2] discovers workflow media by kind regardless of producing phase", async () => {
    vi.mocked(listEvidence).mockResolvedValue({
      items: [evidence("workflow.video", "future-smoke-provider", "opaque-video", "Login recording")],
      total: 1, limit: 100, offset: 0, hasMore: false, degradedReasons: [],
    } as never);
    renderWithQueryClient(<WorkflowsTab scenarioSlug="git-control-tower" testGenieAvailable onOpenBaselines={vi.fn()} />);
    fireEvent.click(await screen.findByText("run-abc"));
    expect(document.querySelector("video")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /preview/i }));
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
    expect(document.querySelector("video")?.getAttribute("src")).toContain("/test-runs/run-abc/artifacts/opaque-video");
    expect(screen.getAllByText(/future-smoke-provider/).length).toBeGreaterThan(0);
  });

  it("attaches workflow artifacts and opens the exact producer phase", async () => {
    vi.mocked(listEvidence).mockResolvedValue({
      items: [evidence("trace", "non-workflow-producer", "opaque-trace", "Login trace")],
      total: 1, limit: 40, offset: 0, hasMore: false, degradedReasons: [],
    } as never);
    const onAttachToAgent = vi.fn();
    const onOpenTests = vi.fn();
    renderWithQueryClient(<WorkflowsTab scenarioSlug="git-control-tower" testGenieAvailable agentManagerAvailable onAttachToAgent={onAttachToAgent} onOpenTests={onOpenTests} onOpenBaselines={vi.fn()} />);
    fireEvent.click(await screen.findByText("run-abc"));
    fireEvent.click(screen.getByText("+ Agent"));
    expect(onAttachToAgent).toHaveBeenCalledWith(expect.objectContaining({ id: "artifact:run-abc:opaque-trace" }));
    fireEvent.click(screen.getByRole("button", { name: /open exact test phase/i }));
    expect(onOpenTests).toHaveBeenCalledWith("run-abc", "non-workflow-producer");
  });
});

describe("ArtifactEvidenceRenderer", () => {
  it("[REQ:GCT-DESCRIPTOR-REVIEW-P2] registers known stable kinds and safely falls back for an unknown future kind", () => {
    expect(artifactRendererKind("coverage.report")).toBe("coverage");
    expect(artifactRendererKind("future.binary.evidence")).toBe("generic");
    const item = evidence("future.binary.evidence", "future-provider", "opaque-future", "Future evidence");
    item.artifact.accessCapability = 0;
    item.artifact.provenance = 2;
    item.artifact.metadata = { summary: "inspectable without bytes" };
    item.artifact.relationships = [{ type: "derived_from", targetArtifactId: "opaque-source" } as never];
    renderWithQueryClient(<ArtifactEvidenceRenderer scenario="git-control-tower" run={item.run as never} artifact={item.artifact as never} />);
    expect(screen.getByText("Artifact · future.binary.evidence")).toBeInTheDocument();
    expect(screen.getByText("Legacy discovery")).toBeInTheDocument();
    expect(screen.getByText("Artifact bytes unavailable")).toBeInTheDocument();
    expect(screen.getByText("opaque-source")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /open evidence/i })).not.toBeInTheDocument();
  });

  it("labels visual changes as advisory", () => {
    const item = evidence("visual.diff", "visual-provider", "opaque-diff", "Dashboard comparison");
    item.artifact.metadata = { changed_fraction: "0.125" };
    renderWithQueryClient(<ArtifactEvidenceRenderer scenario="git-control-tower" run={item.run as never} artifact={item.artifact as never} />);
    expect(screen.getByText(/12.5% changed/i)).toBeInTheDocument();
    expect(screen.getByText(/do not alter the test verdict/i)).toBeInTheDocument();
  });
});
