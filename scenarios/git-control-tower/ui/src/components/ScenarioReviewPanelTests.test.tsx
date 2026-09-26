import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { RunInfo } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import { TestsTab } from "./ScenarioReviewPanelTests";
import { renderWithQueryClient } from "../test-utils";
import { getRun, listRuns, startRun } from "../lib/api-evidence";
import { listBaselines } from "../lib/api-baselines";

vi.mock("../lib/api-evidence", () => ({
  listRuns: vi.fn(),
  startRun: vi.fn(),
  listEvidence: vi.fn(),
  getRun: vi.fn(),
  runArtifactUrl: vi.fn(),
}));

vi.mock("../lib/api-baselines", async () => {
  const actual = await vi.importActual<typeof import("../lib/api-baselines")>("../lib/api-baselines");
  return { ...actual, listBaselines: vi.fn().mockResolvedValue([]) };
});

function run(overrides: Partial<RunInfo> = {}): RunInfo {
  return {
    runId: "run-latest",
    scenario: "git-control-tower",
    status: "failed",
    startedAt: "2026-05-01T12:00:00Z",
    completedAt: "2026-05-01T12:01:20Z",
    preset: "comprehensive",
    phases: [
      { name: "unit", status: "passed", durationSeconds: 12 },
      { name: "future-health", status: "failed", durationSeconds: 8, findingsSummary: { blockers: 1, errors: 2, warnings: 3, infos: 0, total: 6 } },
    ],
    plannedPhases: ["unit", "future-health"],
    pins: [],
    descriptorSnapshot: {
      schemaVersion: 1,
      digest: "sha256:catalog",
      phases: [
        { phase: "unit", displayName: "Unit", provider: "unit-health", dimensions: [], evidenceKinds: [], aliases: [], supersedes: [] },
        { phase: "future-health", displayName: "Future Health", description: "An unknown future phase", provider: "future-provider", phaseClass: "future-class", dimensions: ["novel"], evidenceKinds: ["future.evidence"], aliases: [], supersedes: [] },
      ],
    },
    ...overrides,
  } as RunInfo;
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(listBaselines).mockResolvedValue([]);
  vi.mocked(listRuns).mockResolvedValue({ runs: [run(), run({ runId: "run-previous", status: "passed", completedAt: "2026-05-01T11:00:00Z" })], total: 2, limit: 10, offset: 0, hasMore: false } as never);
  vi.mocked(startRun).mockResolvedValue({ runId: "run-new", scenario: "git-control-tower" } as never);
  vi.mocked(getRun).mockResolvedValue({
    run: run(),
    artifacts: [{ id: "artifact-1", kind: "future.report", label: "Future report", producingPhase: "future-health", mediaType: "application/json" }],
    artifactTotal: 1,
  } as never);
});

describe("TestsTab", () => {
  it("shows a service unavailable state when test-genie is not available", () => {
    renderWithQueryClient(<TestsTab scenarioSlug="git-control-tower" testGenieAvailable={false} />);
    expect(screen.getByText(/test genie is not available/i)).toBeInTheDocument();
  });

  it("[REQ:GCT-DESCRIPTOR-REVIEW-P2] renders captured descriptors and preserves unknown phase metadata", async () => {
    renderWithQueryClient(<TestsTab scenarioSlug="git-control-tower" repoId="repo-1" testGenieAvailable />);
    expect(await screen.findByText("Failed")).toBeInTheDocument();
    expect(screen.getByText("1 passed")).toBeInTheDocument();
    expect(screen.getByText("1 failed")).toBeInTheDocument();
    expect(screen.getByText("Future Health")).toBeInTheDocument();
    expect(screen.getAllByText("future-provider").length).toBeGreaterThan(0);
    fireEvent.click(screen.getByText("Future Health"));
    expect(screen.getByText("An unknown future phase")).toBeInTheDocument();
    expect(screen.getAllByText("future-class", { exact: false }).length).toBeGreaterThan(0);
    expect(screen.getByText(/6 findings/)).toBeInTheDocument();
    expect(await screen.findByText("Future report")).toBeInTheDocument();
    expect(getRun).toHaveBeenCalledWith("git-control-tower", "run-latest", expect.objectContaining({ limit: 100 }));
    expect(screen.getByText("Run history")).toBeInTheDocument();
  });

  it("keeps clean phases compact, filters descriptor metadata, and opens historical runs", async () => {
    renderWithQueryClient(<TestsTab scenarioSlug="git-control-tower" repoId="repo-1" testGenieAvailable />);
    expect(await screen.findByText("Needs attention")).toBeInTheDocument();
    expect(screen.queryByText("Unit")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /clean phases/i }));
    expect(screen.getByText("Unit")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "future-provider" } });
    expect(screen.getByText("Future Health")).toBeInTheDocument();
    expect(screen.queryByText("Unit")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /run-previous/i }));
    expect(screen.getByText("All Passed")).toBeInTheDocument();
  });

  it("opens an exact historical clean phase from an evidence workbench", async () => {
    renderWithQueryClient(<TestsTab scenarioSlug="git-control-tower" repoId="repo-1" testGenieAvailable target={{ runId: "run-previous", phase: "unit" }} />);
    expect(await screen.findByText("All Passed")).toBeInTheDocument();
    expect(screen.getByText("Unit")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /unit/i })).toHaveAttribute("aria-expanded", "true");
  });

  it("bounds large clean catalogs with incremental rendering", async () => {
    const phases = Array.from({ length: 50 }, (_, index) => ({ name: `clean-${index}`, status: "passed", durationSeconds: 1 }));
    const descriptors = phases.map((phase, index) => ({ phase: phase.name, displayName: `Clean ${index}`, provider: "synthetic-provider", dimensions: ["large-catalog"], evidenceKinds: [], aliases: [], supersedes: [] }));
    vi.mocked(listRuns).mockResolvedValue({ runs: [run({ status: "passed", phases: phases as never, plannedPhases: phases.map((phase) => phase.name), descriptorSnapshot: { schemaVersion: 1, digest: "sha256:large", phases: descriptors } as never })], total: 1, limit: 50, offset: 0, hasMore: false } as never);

    renderWithQueryClient(<TestsTab scenarioSlug="git-control-tower" repoId="repo-1" testGenieAvailable />);
    const cleanToggle = await screen.findByRole("button", { name: /clean phases/i });
    expect(screen.queryByText("Clean 0")).not.toBeInTheDocument();
    fireEvent.click(cleanToggle);
    expect(screen.getByText("Clean 0")).toBeInTheDocument();
    expect(screen.getByText("Clean 19")).toBeInTheDocument();
    expect(screen.queryByText("Clean 20")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /show 20 more clean phases/i }));
    expect(screen.getByText("Clean 20")).toBeInTheDocument();
    expect(screen.queryByText("Clean 40")).not.toBeInTheDocument();
  });

  it("renders explicit zero-phase and legacy descriptor degradation states", async () => {
    vi.mocked(listRuns).mockResolvedValue({ runs: [run({ phases: [], plannedPhases: [], descriptorSnapshot: undefined, descriptorSnapshotDigest: "" })], total: 1, limit: 50, offset: 0, hasMore: false } as never);
    renderWithQueryClient(<TestsTab scenarioSlug="git-control-tower" testGenieAvailable />);
    expect(await screen.findByText("No phase records were captured for this run.")).toBeInTheDocument();
    expect(screen.getByText("legacy catalog metadata unavailable")).toBeInTheDocument();
  });

  it("starts a durable run and invalidates canonical run history", async () => {
    renderWithQueryClient(<TestsTab scenarioSlug="git-control-tower" repoId="repo-1" testGenieAvailable />);
    fireEvent.click(await screen.findByRole("button", { name: /run tests/i }));
    await waitFor(() => expect(startRun).toHaveBeenCalledWith({ scenario: "git-control-tower" }));
    await waitFor(() => expect(listRuns).toHaveBeenCalledTimes(2));
  });
});
