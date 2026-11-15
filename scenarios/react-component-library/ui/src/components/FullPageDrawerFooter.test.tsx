import { renderWithProviders } from "../test-utils";
import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { FullPageDrawer } from "../../../library/components/FullPageDrawer/versions/1.5.3/FullPageDrawer";
import type { OverlayActionList } from "../../../library/components/OverlayActionBar/versions/1.0.1/OverlayActionBar";

const attachActions: OverlayActionList = [
  { id: "attach", label: "Attach files", primary: true, onSelect: () => undefined },
  { id: "cancel", label: "Cancel", primary: false, onSelect: () => undefined },
];

/** The drawer root carries the data-has-* layout flags; testId is on its dialog panel. */
const drawerRoot = () => screen.getByTestId("drawer").closest<HTMLElement>("[data-rcl-full-page-drawer]") as HTMLElement;

function renderDrawer(props: { footer?: React.ReactNode; actions?: OverlayActionList }) {
  return renderWithProviders(
    <FullPageDrawer open title="Choose files" closeLabel="Close drawer" testId="drawer" {...props}>
      <p>Choose files from this workspace.</p>
    </FullPageDrawer>,
  );
}

describe("FullPageDrawer footer composition", () => {
  it("keeps contextual footer content and renders the shared action row", () => {
    renderDrawer({ footer: <p data-testid="footer-selection-count">2 files selected</p>, actions: attachActions });

    expect(screen.getByTestId("footer-selection-count")).toHaveTextContent("2 files selected");
    const actions = screen.getByRole("group", { name: "Actions" });
    expect(within(actions).getAllByRole("button")).toHaveLength(2);
    // Both live in the one footer, context above the action row.
    const footer = screen.getByTestId("footer-selection-count").closest("footer");
    expect(footer).not.toBeNull();
    expect(footer).toContainElement(actions);
    expect(drawerRoot()).toHaveAttribute("data-has-footer");
    expect(drawerRoot()).toHaveAttribute("data-has-actions");
  });

  it("renders only the action row when no footer is passed", () => {
    renderDrawer({ actions: attachActions });

    expect(screen.getByRole("group", { name: "Actions" })).toBeInTheDocument();
    expect(drawerRoot()).toHaveAttribute("data-has-actions");
  });

  it("renders only the footer when no actions are passed", () => {
    renderDrawer({ footer: <p data-testid="footer-only">Footer</p> });

    expect(screen.getByTestId("footer-only")).toBeInTheDocument();
    expect(screen.queryByRole("group", { name: "Actions" })).toBeNull();
    expect(drawerRoot()).not.toHaveAttribute("data-has-actions");
  });
});
