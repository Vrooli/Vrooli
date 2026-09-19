import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ConfirmDialog } from "./confirm-dialog";

afterEach(() => {
  vi.restoreAllMocks();
});

function renderStrong(overrides = {}) {
  return render(
    <ConfirmDialog
      isOpen
      onClose={vi.fn()}
      onConfirm={vi.fn()}
      title="Delete Scenario"
      description="This cannot be undone."
      confirmationText="my-scenario"
      testIds={{ confirmButton: "confirm", copyButton: "copy" }}
      {...overrides}
    />,
  );
}

describe("ConfirmDialog copy button", () => {
  it("copies the confirmation text to the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    renderStrong();
    fireEvent.click(screen.getByTestId("copy"));

    expect(writeText).toHaveBeenCalledWith("my-scenario");
    await waitFor(() => expect(screen.getByTestId("copy")).toHaveTextContent("Copied"));
  });

  it("falls back to pre-filling the input when clipboard is unavailable", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("denied"));
    Object.assign(navigator, { clipboard: { writeText } });

    renderStrong();
    const confirm = screen.getByTestId("confirm");
    expect(confirm).toBeDisabled();

    fireEvent.click(screen.getByTestId("copy"));

    // Fallback fills the input with the exact confirmation text, enabling confirm.
    await waitFor(() => expect(confirm).not.toBeDisabled());
  });

  it("does not render a confirmation input (or copy button) for simple deletes", () => {
    render(
      <ConfirmDialog
        isOpen
        onClose={vi.fn()}
        onConfirm={vi.fn()}
        title="Delete Session"
        description="This cannot be undone."
        testIds={{ copyButton: "copy" }}
      />,
    );
    expect(screen.queryByTestId("copy")).toBeNull();
  });

  it("uses the bottom-sheet presentation when requested", () => {
    renderStrong({
      presentation: "bottom-sheet",
      testIds: { dialog: "delete-sheet", confirmButton: "confirm", copyButton: "copy" },
    });

    expect(screen.getByTestId("delete-sheet")).toHaveAttribute("role", "dialog");
    expect(screen.getByTestId("delete-sheet")).toHaveTextContent("Delete Scenario");
  });
});
