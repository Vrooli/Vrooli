import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { renderWithQueryClient } from "../../test-utils";
import { BaselinesTab } from "./BaselinesTab";
import { SetBaselineModal } from "./SetBaselineModal";
import { WorkflowsDiff } from "./diffs/WorkflowsDiff";
import * as api from "../../lib/api-baselines";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";
import type { SurfaceDiff } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

vi.mock("../../lib/api-baselines", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api-baselines")>();
  return {
    ...actual,
    listBaselines: vi.fn(),
    snapshotForBaseline: vi.fn(),
    deleteBaseline: vi.fn(),
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
    surfaces: { workflows: { surfaceId: "workflows", kind: "test-genie-run", ref: "r1", capturedAt: "", summary: "" } },
    skipped: {},
    schemaVersion: 1,
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

    const [firstDelete] = screen.getAllByRole("button", { name: "Delete" });
    fireEvent.click(firstDelete as HTMLElement);

    await waitFor(() =>
      expect(api.deleteBaseline).toHaveBeenCalledWith(
        expect.objectContaining({ scenario: "demo", name: "pre-launch" }),
      ),
    );
  });
});

describe("SetBaselineModal", () => {
  it("defaults to all surfaces + Fast and captures with the include list", async () => {
    vi.mocked(api.snapshotForBaseline).mockResolvedValue({} as never);
    const onCreated = vi.fn();
    const onClose = vi.fn();

    renderWithQueryClient(
      <SetBaselineModal isOpen scenario="demo" repoId={null} onClose={onClose} onCreated={onCreated} />,
    );

    // Dirty-tree warning is shown (summary has 2 staged).
    expect(screen.getByText(/working tree is dirty/i)).toBeInTheDocument();
    // All five surface checkboxes start checked.
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(5);
    checkboxes.forEach((c) => expect(c).toBeChecked());

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "ui-demo" } });
    fireEvent.click(screen.getByRole("button", { name: /capture/i }));

    await waitFor(() =>
      expect(api.snapshotForBaseline).toHaveBeenCalledWith(
        expect.objectContaining({
          scenario: "demo",
          name: "ui-demo",
          fast: true,
          include: ["workflows", "tests", "structure", "visuals", "rules"],
        }),
      ),
    );
    expect(onCreated).toHaveBeenCalledWith("ui-demo");
  });
});

describe("WorkflowsDiff", () => {
  it("renders regression and preexisting entity lists", () => {
    const diff = {
      surfaceId: "workflows",
      verdict: "regression",
      regressions: ["login-smoke"],
      newFailures: [],
      preexisting: ["legacy-flow"],
      cleared: [],
      summary: "1 regression",
    } as unknown as SurfaceDiff;

    renderWithQueryClient(<WorkflowsDiff diff={diff} />);
    expect(screen.getByText("Regression")).toBeInTheDocument();
    expect(screen.getByText("login-smoke")).toBeInTheDocument();
    expect(screen.getByText("legacy-flow")).toBeInTheDocument();
  });
});
