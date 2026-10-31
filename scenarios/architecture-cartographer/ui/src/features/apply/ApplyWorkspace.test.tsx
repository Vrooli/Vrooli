import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { OperationKind } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

vi.mock("./controllers/useApplyController", () => ({
  useApplyHistory: vi.fn(),
  useBuildBaseline: vi.fn(),
  usePlanApply: vi.fn(),
  useRunApply: vi.fn(),
}));

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { ApplyWorkspace } from "./ApplyWorkspace";
import {
  useApplyHistory,
  useBuildBaseline,
  usePlanApply,
  useRunApply,
} from "./controllers/useApplyController";

const plan = {
  id: "plan-1",
  operations: [
    {
      id: "op-1",
      kind: OperationKind.MOVE_FILE,
      fromPath: "api/internal/old.go",
      toPath: "api/internal/new.go",
    },
  ],
};

afterEach(() => {
  cleanup();
  vi.mocked(useApplyHistory).mockReset();
  vi.mocked(useBuildBaseline).mockReset();
  vi.mocked(usePlanApply).mockReset();
  vi.mocked(useRunApply).mockReset();
});

function mockWorkspaceState({
  baselineGreen = true,
  baselineToolchain = "go test",
  planPending = false,
  runPending = false,
  runError = false,
  planMutate = vi.fn().mockResolvedValue({ plan }),
  runMutate = vi.fn().mockResolvedValue({ run: { id: "run-1" } }),
}: {
  baselineGreen?: boolean;
  baselineToolchain?: string;
  planPending?: boolean;
  runPending?: boolean;
  runError?: boolean;
  planMutate?: ReturnType<typeof vi.fn>;
  runMutate?: ReturnType<typeof vi.fn>;
} = {}) {
  vi.mocked(useBuildBaseline).mockReturnValue({
    data: { baseline: { green: baselineGreen, toolchain: baselineToolchain } },
  } as unknown as ReturnType<typeof useBuildBaseline>);
  vi.mocked(useApplyHistory).mockReturnValue({
    isPending: false,
    isError: false,
    data: { runs: [] },
    error: null,
    refetch: vi.fn(),
  } as unknown as ReturnType<typeof useApplyHistory>);
  vi.mocked(usePlanApply).mockReturnValue({
    isPending: planPending,
    mutateAsync: planMutate,
  } as unknown as ReturnType<typeof usePlanApply>);
  vi.mocked(useRunApply).mockReturnValue({
    isPending: runPending,
    isError: runError,
    mutateAsync: runMutate,
  } as unknown as ReturnType<typeof useRunApply>);
  return { planMutate, runMutate };
}

describe("ApplyWorkspace", () => {
  it("plans, dry-runs, and applies when the baseline is green", async () => {
    const { planMutate, runMutate } = mockWorkspaceState();
    renderWithProviders(<ApplyWorkspace scenario="demo" domain="graph" />);

    expect(screen.getByTestId(selectors.features.apply.plan.dryRunButton)).toBeDisabled();
    fireEvent.click(screen.getByTestId(selectors.features.apply.plan.planButton));

    await waitFor(() => expect(planMutate).toHaveBeenCalledWith({ scenario: "demo", domain: "graph", dryRun: false }));
    expect(screen.getByTestId(selectors.features.apply.plan.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.apply.dryRun.root)).toBeInTheDocument();

    fireEvent.click(screen.getByTestId(selectors.features.apply.plan.dryRunButton));
    await waitFor(() => expect(planMutate).toHaveBeenCalledWith({ scenario: "demo", domain: "graph", dryRun: true }));

    fireEvent.click(screen.getByTestId(selectors.features.apply.plan.applyButton));
    await waitFor(() => expect(runMutate).toHaveBeenCalledWith({ scenario: "demo", domain: "graph", planId: "plan-1" }));
    expect(screen.queryByTestId(selectors.features.apply.confirmDialog.root)).not.toBeInTheDocument();
  });

  it("requires confirmation before applying on a red baseline", async () => {
    const { runMutate } = mockWorkspaceState({ baselineGreen: false });
    renderWithProviders(<ApplyWorkspace scenario="demo" domain="graph" />);

    fireEvent.click(screen.getByTestId(selectors.features.apply.plan.planButton));
    await screen.findByTestId(selectors.features.apply.plan.root);

    fireEvent.click(screen.getByTestId(selectors.features.apply.plan.applyButton));
    expect(screen.getByTestId(selectors.features.apply.confirmDialog.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.apply.confirmDialog.confirmButton)).toBeDisabled();

    fireEvent.change(screen.getByTestId(selectors.features.apply.confirmDialog.noteInput), {
      target: { value: "checked red baseline" },
    });
    fireEvent.click(screen.getByTestId(selectors.features.apply.confirmDialog.confirmButton));

    await waitFor(() => expect(runMutate).toHaveBeenCalledWith({ scenario: "demo", domain: "graph", planId: "plan-1" }));
  });

  it("surfaces pending and run-error states", () => {
    mockWorkspaceState({ baselineToolchain: "", planPending: true, runPending: true, runError: true });
    renderWithProviders(<ApplyWorkspace scenario="demo" domain="graph" />);

    expect(screen.getByTestId(selectors.features.apply.plan.planButton)).toBeDisabled();
    expect(screen.getByTestId(selectors.features.apply.plan.applyButton)).toBeDisabled();
    expect(screen.getAllByTestId(selectors.shared.emptyState.root).at(1)).toHaveTextContent(
      "pages.targetApply.applyUnimplementedTitle",
    );
    expect(screen.getByTestId(selectors.features.apply.overview.baseline)).toHaveTextContent(
      "pages.targetApply.baselineUnknown",
    );
  });
});
