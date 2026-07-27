import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeleteButton } from "./DeleteButton";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

const mocks = vi.hoisted(() => ({ deleteDesktopBuild: vi.fn() }));
vi.mock("../../lib/api", () => ({
  deleteDesktopBuild: mocks.deleteDesktopBuild,
}));

describe("DeleteButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("requires the exact scenario name before permanent deletion", () => {
    renderWithProviders(<DeleteButton scenarioName="canvas-lab" />);
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    const confirmation = screen.getByLabelText(/Type the scenario name/i);
    expect(
      screen.getByRole("button", { name: "Confirm Delete" }),
    ).toBeDisabled();
    fireEvent.change(confirmation, { target: { value: "canvas" } });
    expect(
      screen.getByRole("button", { name: "Confirm Delete" }),
    ).toBeDisabled();
    fireEvent.change(confirmation, { target: { value: "canvas-lab" } });
    expect(
      screen.getByRole("button", { name: "Confirm Delete" }),
    ).toBeEnabled();
  });

  it("deletes only after confirmation and shows completion", async () => {
    mocks.deleteDesktopBuild.mockResolvedValue(undefined);
    renderWithProviders(<DeleteButton scenarioName="canvas-lab" />);
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    fireEvent.change(screen.getByLabelText(/Type the scenario name/i), {
      target: { value: "canvas-lab" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Confirm Delete" }));

    await waitFor(() => {
      expect(mocks.deleteDesktopBuild).toHaveBeenCalledWith("canvas-lab");
    });
    expect(await screen.findByText("Deleted")).toBeInTheDocument();
  });

  it("allows cancellation and recovery from a failed delete", async () => {
    mocks.deleteDesktopBuild.mockRejectedValueOnce(
      new Error("permission denied"),
    );
    renderWithProviders(<DeleteButton scenarioName="canvas-lab" />);
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    fireEvent.change(screen.getByLabelText(/Type the scenario name/i), {
      target: { value: "canvas-lab" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Confirm Delete" }));
    expect(await screen.findByText("Failed")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(
      screen.getByLabelText(/Type the scenario name/i),
    ).toBeInTheDocument();
  });
});
