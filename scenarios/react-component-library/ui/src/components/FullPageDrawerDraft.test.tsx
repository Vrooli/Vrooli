import { renderWithProviders } from "../test-utils";
import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { FullPageDrawer } from "../../../library/components/FullPageDrawer/versions/1.5.3-draft.1/FullPageDrawer";
import type { OverlayActionList } from "../../../library/components/OverlayActionBar/versions/1.0.0/OverlayActionBar";

const attachActions: OverlayActionList = [
  { id: "attach", label: "Attach files", primary: true, onSelect: () => undefined },
  { id: "cancel", label: "Cancel", primary: false, onSelect: () => undefined },
];

describe("FullPageDrawer draft footer composition", () => {
  it("keeps contextual footer content and renders the shared action row", () => {
    renderWithProviders(
      <FullPageDrawer
        open
        title="Choose files"
        closeLabel="Close drawer"
        footer={<p data-testid="footer-selection-count">2 files selected</p>}
        actions={attachActions}
      >
        <p>Choose files from this workspace.</p>
      </FullPageDrawer>,
    );

    expect(screen.getByTestId("footer-selection-count")).toHaveTextContent("2 files selected");
    expect(screen.getByRole("group", { name: "Actions" })).toBeInTheDocument();
    expect(within(screen.getByRole("group", { name: "Actions" })).getAllByRole("button")).toHaveLength(2);
  });
});
