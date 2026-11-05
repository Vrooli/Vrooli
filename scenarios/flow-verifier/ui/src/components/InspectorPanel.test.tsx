import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { InspectorPanel } from "./InspectorPanel";

describe("InspectorPanel", () => {
  afterEach(() => cleanup());

  it("renders nothing when closed", () => {
    renderWithProviders(
      <InspectorPanel open={false} onClose={() => {}} title="Inspector">
        <span data-testid="inspector-child">child</span>
      </InspectorPanel>,
    );
    expect(screen.queryByTestId("inspector-child")).not.toBeInTheDocument();
  });

  it("renders desktop pane with title and child content when open", () => {
    renderWithProviders(
      <InspectorPanel open onClose={() => {}} title="Selected run">
        <span data-testid="inspector-child">child</span>
      </InspectorPanel>,
    );
    expect(screen.getByTestId("inspector-desktop")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-title")).toHaveTextContent("Selected run");
    expect(screen.getByTestId("inspector-child")).toBeInTheDocument();
  });

  it("calls onClose when the close button is clicked", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <InspectorPanel open onClose={onClose} title="t">
        <span />
      </InspectorPanel>,
    );
    await user.click(screen.getByTestId("inspector-close"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose when Escape is pressed", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <InspectorPanel open onClose={onClose} title="t">
        <span />
      </InspectorPanel>,
    );
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });
});
