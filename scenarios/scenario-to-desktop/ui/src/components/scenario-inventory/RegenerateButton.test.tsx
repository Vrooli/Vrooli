import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RegenerateButton } from "./RegenerateButton";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { StageName } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const mocks = vi.hoisted(() => ({
  mutation: { isPending: false },
  status: { isBuilding: false, isComplete: false, isFailed: false },
  runPipelineWithConfig: vi.fn(),
  reset: vi.fn(),
  clearBuildId: vi.fn(),
}));

vi.mock("../../hooks", () => ({
  usePipelineMutation: () => ({
    state: { buildId: "build-1" },
    mutation: mocks.mutation,
    runPipelineWithConfig: mocks.runPipelineWithConfig,
    reset: mocks.reset,
    clearBuildId: mocks.clearBuildId,
  }),
  usePipelineStatus: () => mocks.status,
}));

describe("RegenerateButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.mutation.isPending = false;
    Object.assign(mocks.status, {
      isBuilding: false,
      isComplete: false,
      isFailed: false,
    });
  });

  it("requires confirmation and starts a typed generate-only pipeline", () => {
    renderWithProviders(
      <RegenerateButton
        scenarioName="canvas-lab"
        connectionConfig={{
          proxy_url: "http://127.0.0.1:5050",
          deployment_mode: "bundled",
        }}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Regenerate" }));
    expect(screen.getByText("Regenerate desktop app?")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Confirm Regenerate" }));
    expect(mocks.runPipelineWithConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        scenarioName: "canvas-lab",
        proxyUrl: "http://127.0.0.1:5050",
        stopAfterStage: StageName.GENERATE,
      }),
    );
  });

  it("allows cancellation before work begins", () => {
    renderWithProviders(<RegenerateButton scenarioName="canvas-lab" />);
    fireEvent.click(screen.getByRole("button", { name: "Regenerate" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.getByRole("button", { name: "Regenerate" }),
    ).toBeInTheDocument();
    expect(mocks.runPipelineWithConfig).not.toHaveBeenCalled();
  });

  it("shows running, completion, and recoverable failure states", () => {
    mocks.mutation.isPending = true;
    const { rerender } = renderWithProviders(
      <RegenerateButton scenarioName="canvas-lab" />,
    );
    expect(screen.getByText("Regenerating...")).toBeInTheDocument();

    mocks.mutation.isPending = false;
    Object.assign(mocks.status, { isComplete: true });
    rerender(<RegenerateButton scenarioName="canvas-lab" />);
    expect(screen.getByText("Regenerated!")).toBeInTheDocument();

    Object.assign(mocks.status, { isComplete: false, isFailed: true });
    rerender(<RegenerateButton scenarioName="canvas-lab" />);
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(mocks.reset).toHaveBeenCalledOnce();
    Object.assign(mocks.status, { isFailed: false });
    rerender(<RegenerateButton scenarioName="canvas-lab" />);
    expect(screen.getByText("Regenerate desktop app?")).toBeInTheDocument();
  });
});
