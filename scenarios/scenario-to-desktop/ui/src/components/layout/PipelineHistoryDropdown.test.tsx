import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { act, screen, waitFor } from "@testing-library/react";
import { PipelineHistoryDropdown } from "./PipelineHistoryDropdown";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { createPipelineStatus } from "../../test-utils/mocks";
import { usePipelineStore } from "../../store";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const originalLoadPipelineHistory =
  usePipelineStore.getState().loadPipelineHistory;

function setPipelineState(
  state: Partial<ReturnType<typeof usePipelineStore.getState>>,
) {
  act(() => {
    usePipelineStore.setState(state);
  });
}

beforeEach(() => {
  setPipelineState({
    scenarioName: "calculator",
    pipelineId: "pipeline-current",
    loadPipelineHistory: vi.fn(),
  });
});

afterEach(() => {
  setPipelineState({
    scenarioName: null,
    pipelineId: null,
    loadPipelineHistory: originalLoadPipelineHistory,
  });
});

describe("PipelineHistoryDropdown", () => {
  it("does not render or request history while closed", () => {
    renderWithProviders(
      <PipelineHistoryDropdown open={false} onClose={vi.fn()} />,
    );

    expect(screen.queryByText("Pipeline History")).not.toBeInTheDocument();
    expect(
      usePipelineStore.getState().loadPipelineHistory,
    ).not.toHaveBeenCalled();
  });

  it("shows current and failed pipeline evidence after loading history", async () => {
    const loadPipelineHistory = vi.fn().mockResolvedValue([
      createPipelineStatus({
        pipelineId: "pipeline-current",
        status: StageStatus.COMPLETED,
        stageOrder: [StageName.BUNDLE, StageName.BUILD],
        stages: {
          bundle: { stage: StageName.BUNDLE, status: StageStatus.COMPLETED },
          build: { stage: StageName.BUILD, status: StageStatus.COMPLETED },
        },
        startedAt: { seconds: 1_700_000_000n, nanos: 0 },
      }),
      createPipelineStatus({
        pipelineId: "pipeline-failed",
        status: StageStatus.FAILED,
        stageOrder: [StageName.BUNDLE, StageName.BUILD],
        stages: {
          bundle: { stage: StageName.BUNDLE, status: StageStatus.COMPLETED },
          build: { stage: StageName.BUILD, status: StageStatus.FAILED },
        },
        error: "electron-builder did not produce an installer",
      }),
    ]);
    setPipelineState({ loadPipelineHistory });

    renderWithProviders(<PipelineHistoryDropdown open onClose={vi.fn()} />);

    expect(await screen.findByText("pipeline-current")).toBeInTheDocument();
    expect(loadPipelineHistory).toHaveBeenCalledWith(10);
    expect(screen.getByText("Current")).toBeInTheDocument();
    expect(screen.getByText("pipeline-failed")).toBeInTheDocument();
    expect(
      screen.getByText("electron-builder did not produce an installer"),
    ).toBeInTheDocument();
    expect(screen.getAllByTitle("BUILD: FAILED")).toHaveLength(1);
  });

  it("surfaces a failed history request and allows the operator to retry", async () => {
    const loadPipelineHistory = vi
      .fn()
      .mockRejectedValueOnce(new Error("history service unavailable"))
      .mockResolvedValueOnce([]);
    setPipelineState({ loadPipelineHistory });

    renderWithProviders(<PipelineHistoryDropdown open onClose={vi.fn()} />);

    expect(
      await screen.findByText("history service unavailable"),
    ).toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Refresh" }));
    await waitFor(() => {
      expect(loadPipelineHistory).toHaveBeenCalledTimes(2);
    });
    expect(
      await screen.findByText("No pipeline history found."),
    ).toBeInTheDocument();
  });

  it("closes from the backdrop and explicit close control", async () => {
    const onClose = vi.fn();
    setPipelineState({
      loadPipelineHistory: vi.fn().mockResolvedValue([]),
    });
    renderWithProviders(<PipelineHistoryDropdown open onClose={onClose} />);

    await screen.findByText("No pipeline history found.");
    const user = userEvent.setup();
    const backdrop = screen.getByText("Pipeline History").closest("div.fixed");
    if (!backdrop) throw new Error("Pipeline history backdrop is not mounted");
    await user.click(backdrop);
    expect(onClose).toHaveBeenCalledOnce();

    await user.click(
      screen.getByRole("button", { name: "Close pipeline history" }),
    );
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
