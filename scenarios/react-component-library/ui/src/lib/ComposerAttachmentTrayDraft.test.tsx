import { useState } from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { ComposerAttachmentTray } from "../../../library/components/ComposerAttachmentTray/versions/1.0.4/ComposerAttachmentTray";

describe("ComposerAttachmentTray draft", () => {
  it("keeps keyboard reorder actions and position announcements", () => {
    const onReorder = vi.fn();
    renderWithProviders(
      <ComposerAttachmentTray
        items={[
          { id: "brief", name: "Brief.pdf" },
          { id: "notes", name: "Notes.md" },
        ]}
        onReorder={onReorder}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Move Notes.md earlier" }));
    expect(onReorder).toHaveBeenCalledWith(["notes", "brief"]);
    expect(screen.getByText("Notes.md moved to position 1 of 2.")).toHaveAttribute("role", "status");
  });

  it("retains the failed attachment error and retry action", () => {
    const onRetry = vi.fn();
    renderWithProviders(
      <ComposerAttachmentTray
        items={[
          {
            id: "failed",
            name: "Notes.md",
            status: "error",
            errorMessage: "The file needs another upload.",
          },
        ]}
        onRetry={onRetry}
      />,
    );

    const errorState = screen
      .getByText("The file needs another upload.")
      .closest("[data-rcl-attachment-preview-status]");
    expect(errorState).toHaveAttribute("role", "alert");
    expect(errorState).toHaveTextContent("The file needs another upload.");
    fireEvent.click(screen.getByRole("button", { name: "Retry upload" }));
    expect(onRetry).toHaveBeenCalledWith("failed");
  });

  it("moves focus to the remaining attachment after removing the focused card", async () => {
    function RemovableTray() {
      const [items, setItems] = useState([
        { id: "brief", name: "Brief.pdf" },
        { id: "notes", name: "Notes.md" },
      ]);
      return (
        <ComposerAttachmentTray
          items={items}
          onRemove={(id) => setItems((rows) => rows.filter((row) => row.id !== id))}
        />
      );
    }

    renderWithProviders(<RemovableTray />);
    const firstRemoveButton = screen.getAllByRole("button", { name: "Remove" }).at(0);
    if (!firstRemoveButton) throw new Error("expected the first attachment remove button");
    fireEvent.click(firstRemoveButton);

    await waitFor(() => expect(screen.queryByText("Brief.pdf")).not.toBeInTheDocument());
    await waitFor(() => expect(screen.getByRole("button", { name: "Remove" })).toHaveFocus());
  });
});
