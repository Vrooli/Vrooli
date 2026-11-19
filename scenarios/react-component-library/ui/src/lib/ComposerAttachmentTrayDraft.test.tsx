import { useState } from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { ComposerAttachmentTray } from "../../../library/components/ComposerAttachmentTray/versions/1.0.4/ComposerAttachmentTray";

// Fixture attachment names and the library's own announcement/error copy,
// named once so the assertions survive a wording change in one place.
const briefName = "Brief.pdf";
const notesName = "Notes.md";
const reorderStatus = "Notes.md moved to position 1 of 2.";
const uploadErrorMessage = "The file needs another upload.";

describe("ComposerAttachmentTray draft", () => {
  it("keeps keyboard reorder actions and position announcements", () => {
    const onReorder = vi.fn();
    renderWithProviders(
      <ComposerAttachmentTray
        items={[
          { id: "brief", name: briefName },
          { id: "notes", name: notesName },
        ]}
        onReorder={onReorder}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: `Move ${notesName} earlier` }));
    expect(onReorder).toHaveBeenCalledWith(["notes", "brief"]);
    expect(screen.getByText(reorderStatus)).toHaveAttribute("role", "status");
  });

  it("retains the failed attachment error and retry action", () => {
    const onRetry = vi.fn();
    renderWithProviders(
      <ComposerAttachmentTray
        items={[
          {
            id: "failed",
            name: notesName,
            status: "error",
            errorMessage: uploadErrorMessage,
          },
        ]}
        onRetry={onRetry}
      />,
    );

    const errorState = screen
      .getByText(uploadErrorMessage)
      .closest("[data-rcl-attachment-preview-status]");
    expect(errorState).toHaveAttribute("role", "alert");
    expect(errorState).toHaveTextContent(uploadErrorMessage);
    fireEvent.click(screen.getByRole("button", { name: "Retry upload" }));
    expect(onRetry).toHaveBeenCalledWith("failed");
  });

  it("moves focus to the remaining attachment after removing the focused card", async () => {
    function RemovableTray() {
      const [items, setItems] = useState([
        { id: "brief", name: briefName },
        { id: "notes", name: notesName },
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

    await waitFor(() => expect(screen.queryByText(briefName)).not.toBeInTheDocument());
    await waitFor(() => expect(screen.getByRole("button", { name: "Remove" })).toHaveFocus());
  });
});
