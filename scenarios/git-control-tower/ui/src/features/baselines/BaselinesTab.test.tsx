import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { renderWithQueryClient } from "../../test-utils";
import { BaselinesTab } from "./BaselinesTab";
import { SetBaselineModal } from "./SetBaselineModal";
import * as api from "../../lib/api-baselines";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

vi.mock("../../lib/api-baselines", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api-baselines")>();
  return {
    ...actual,
    listBaselines: vi.fn(),
    snapshotForBaseline: vi.fn(),
    deleteBaseline: vi.fn(),
    diffBaseline: vi.fn(),
  };
});

// SetBaselineModal reads repo status for branch + dirty warning; stub it.
vi.mock("../../lib/hooks-core", () => ({
  useRepoStatus: () => ({
    data: { branch: { head: "agi" }, summary: { staged: 2, unstaged: 0, untracked: 0, conflicts: 0 } },
    isLoading: false,
  }),
}));

function manifest(name: string, overrides: Partial<BaselineManifest> = {}): BaselineManifest {
  return {
    name,
    scenario: "demo",
    branch: "agi",
    createdAt: "2026-05-26T12:00:00Z",
    createdBy: "ui",
    git: { sha: "abc12345def", branch: "agi", dirty: false, dirtySummary: "" },
    run: {
      runId: "r1",
      capturedAt: "2026-05-26T12:00:00Z",
      captureProfile: "baseline",
      treeDigest: "td:tree",
      phaseSetDigest: "ps:set",
      descriptorSnapshotRef: "test-genie-run:r1#descriptor-snapshot",
      descriptorSnapshotDigest: "ds:catalog",
      descriptorSnapshotSchemaVersion: 1,
    },
    schemaVersion: 2,
    ...overrides,
  } as unknown as BaselineManifest;
}

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
});

describe("BaselinesTab", () => {
  it("shows the empty state with a first-baseline CTA", async () => {
    vi.mocked(api.listBaselines).mockResolvedValue([]);
    renderWithQueryClient(<BaselinesTab scenarioSlug="demo" repoId={null} />);
    expect(await screen.findByText(/no baselines yet/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /set first baseline/i })).toBeInTheDocument();
  });

  it("lists baselines and deletes one", async () => {
    vi.mocked(api.listBaselines).mockResolvedValue([manifest("pre-launch"), manifest("plan-7c3")]);
    vi.mocked(api.deleteBaseline).mockResolvedValue(true);

    renderWithQueryClient(<BaselinesTab scenarioSlug="demo" repoId={null} />);

    expect(await screen.findByText("pre-launch")).toBeInTheDocument();
    expect(screen.getByText("plan-7c3")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();

    const [firstDelete] = screen.getAllByRole("button", { name: "Delete" });
    fireEvent.click(firstDelete as HTMLElement);

    await waitFor(() =>
      expect(api.deleteBaseline).toHaveBeenCalledWith(
        expect.objectContaining({ scenario: "demo", name: "pre-launch" }),
      ),
    );
  });

  it("[REQ:GCT-DESCRIPTOR-REVIEW-P2] summarizes dynamic comparison outcomes and catalog evolution", async () => {
    vi.mocked(api.listBaselines).mockResolvedValue([manifest("pre-launch")]);
    vi.mocked(api.diffBaseline).mockResolvedValue({
      verdict: "regression",
      baseline: manifest("pre-launch"),
      currentGit: { sha: "def67890", dirty: true },
      staleness: { likelyStale: true, commitsSince: 3, filesChanged: 7 },
      evidence: { baseRunId: "r1", currentRunId: "r2", visualDeltas: [{ page: "/", status: "changed", changedFraction: 0.1 }], degradedReasons: [] },
      phases: [
        { phase: "future-health", verdict: "regression", statusA: "passed", statusB: "failed", regressions: ["check-a"], newFailures: [], preexistingFailures: [], clearedFailures: [], reasons: [{ code: 1, detail: "New catalog phase" }], descriptorB: { displayName: "Future Health", provider: "future-provider" } },
        { phase: "unit", verdict: "clean", statusA: "passed", statusB: "passed", regressions: [], newFailures: [], preexistingFailures: [], clearedFailures: [], reasons: [] },
      ],
    } as never);

    renderWithQueryClient(<BaselinesTab scenarioSlug="demo" repoId={null} />);
    fireEvent.click(await screen.findByRole("button", { name: "Compare" }));
    fireEvent.click(screen.getByRole("button", { name: /compare against working tree/i }));

    expect(await screen.findByText("Regressions: 1")).toBeInTheDocument();
    expect(screen.getByText("Catalog changed: 1 new / 0 retired")).toBeInTheDocument();
    expect(screen.getByText(/Baseline likely stale/)).toBeInTheDocument();
    expect(screen.getByText("Future Health")).toBeInTheDocument();
    expect(screen.queryByText("unit")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /clean phases/i }));
    expect(screen.getAllByText("unit").length).toBeGreaterThan(0);
  });
});

describe("SetBaselineModal", () => {
  it("captures one comprehensive run without surface selection", async () => {
    vi.mocked(api.snapshotForBaseline).mockResolvedValue({} as never);
    const onCreated = vi.fn();
    const onClose = vi.fn();

    renderWithQueryClient(
      <SetBaselineModal isOpen scenario="demo" repoId={null} onClose={onClose} onCreated={onCreated} />,
    );

    // Dirty-tree warning is shown (summary has 2 staged).
    expect(screen.getByText(/working tree is dirty/i)).toBeInTheDocument();
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    expect(screen.getByText(/one comprehensive run/i)).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "ui-demo" } });
    fireEvent.click(screen.getByRole("button", { name: /capture/i }));

    await waitFor(() =>
      expect(api.snapshotForBaseline).toHaveBeenCalledWith(
        expect.objectContaining({
          scenario: "demo",
          name: "ui-demo",
        }),
      ),
    );
    expect(onCreated).toHaveBeenCalledWith("ui-demo");
  });
});
