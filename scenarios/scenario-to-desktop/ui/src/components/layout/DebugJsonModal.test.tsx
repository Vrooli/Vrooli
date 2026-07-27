import { act, fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DebugJsonModal } from "./DebugJsonModal";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { usePipelineStore } from "../../store";
import { writeToClipboard } from "../../lib/browser";

vi.mock("../../lib/browser", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../lib/browser")>()),
  writeToClipboard: vi.fn(),
}));

describe("DebugJsonModal", () => {
  beforeEach(() => {
    usePipelineStore.getState().reset();
    act(() => {
      usePipelineStore.setState({
        scenarioName: "canvas-lab",
        pipelineId: "pipe-123",
      });
    });
    vi.clearAllMocks();
    vi.mocked(writeToClipboard).mockResolvedValue({ success: true });
  });

  it("does not mount its portal while closed", () => {
    renderWithProviders(<DebugJsonModal open={false} onClose={vi.fn()} />);
    expect(screen.queryByText("Pipeline Store Debug")).not.toBeInTheDocument();
  });

  it("copies the initial snapshot and refreshes it deliberately", async () => {
    renderWithProviders(<DebugJsonModal open onClose={vi.fn()} />);

    expect(screen.getByText(/"pipelineId": "pipe-123"/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy" }));
    expect(writeToClipboard).toHaveBeenCalledWith(
      expect.stringContaining('"scenarioName": "canvas-lab"'),
    );
    expect(
      await screen.findByRole("button", { name: "Copied" }),
    ).toBeInTheDocument();

    act(() => {
      usePipelineStore.setState({ pipelineId: "pipe-456" });
    });
    expect(
      screen.queryByText(/"pipelineId": "pipe-456"/),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(screen.getByText(/"pipelineId": "pipe-456"/)).toBeInTheDocument();
  });

  it("closes from the explicit control and its backdrop", () => {
    const onClose = vi.fn();
    renderWithProviders(<DebugJsonModal open onClose={onClose} />);

    fireEvent.click(
      screen.getByRole("button", { name: "Close pipeline store debug" }),
    );
    expect(onClose).toHaveBeenCalledOnce();
    const backdrop = screen
      .getByText("Pipeline Store Debug")
      .closest("div.fixed");
    if (!backdrop) throw new Error("debug backdrop is not mounted");
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
