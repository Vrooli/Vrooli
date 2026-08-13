import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SidebarHeader } from "./SidebarHeader";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { usePipelineStore } from "../../store";
import { writeToClipboard } from "../../lib/browser";

vi.mock("../../lib/browser", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../lib/browser")>()),
  writeToClipboard: vi.fn(),
}));

vi.mock("./DebugJsonModal", () => ({
  DebugJsonModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="debug-modal" /> : null,
}));

vi.mock("./PipelineHistoryDropdown", () => ({
  PipelineHistoryDropdown: ({ open }: { open: boolean }) =>
    open ? <div data-testid="history-dropdown" /> : null,
}));

function setPipelineState(
  state: Partial<ReturnType<typeof usePipelineStore.getState>>,
) {
  act(() => {
    usePipelineStore.setState(state);
  });
}

describe("SidebarHeader", () => {
  beforeEach(() => {
    usePipelineStore.getState().reset();
    vi.clearAllMocks();
    vi.mocked(writeToClipboard).mockResolvedValue({ success: true });
    setPipelineState({
      scenarioName: "canvas-lab",
      pipelineId: "pipe-123",
      createNewPipelineForScenario: vi.fn().mockResolvedValue("pipe-456"),
    });
  });

  it("opens the history and debug views through named controls", () => {
    renderWithProviders(<SidebarHeader />);

    fireEvent.click(
      screen.getByRole("button", { name: "Open pipeline history" }),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Open pipeline debug JSON" }),
    );

    expect(screen.getByTestId("history-dropdown")).toBeInTheDocument();
    expect(screen.getByTestId("debug-modal")).toBeInTheDocument();
  });

  it("copies the active pipeline ID and confirms the completed action", async () => {
    renderWithProviders(<SidebarHeader />);

    fireEvent.click(screen.getByRole("button", { name: "Copy pipeline ID" }));

    expect(writeToClipboard).toHaveBeenCalledWith("pipe-123");
    expect(
      await screen.findByRole("button", { name: "Pipeline ID copied" }),
    ).toBeInTheDocument();
  });

  it("creates a new pipeline when idle and prevents it while running", async () => {
    const createNewPipelineForScenario = vi.fn().mockResolvedValue("pipe-456");
    setPipelineState({ createNewPipelineForScenario });
    renderWithProviders(<SidebarHeader />);

    fireEvent.click(screen.getByRole("button", { name: "New" }));
    await waitFor(() => {
      expect(createNewPipelineForScenario).toHaveBeenCalledOnce();
    });

    setPipelineState({ runStatus: "running" });
    expect(screen.getByRole("button", { name: "New" })).toBeDisabled();
  });
});
