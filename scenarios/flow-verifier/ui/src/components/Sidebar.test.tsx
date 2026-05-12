import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { Sidebar } from "./Sidebar";

describe("Sidebar", () => {
  afterEach(() => cleanup());

  it("renders brand and primary nav targets", () => {
    renderWithProviders(<Sidebar />);
    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("app-brand")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("nav-dashboard")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("nav-flows")).toHaveAttribute("href", "/flows");
    expect(screen.getByTestId("nav-settings")).toHaveAttribute("href", "/settings");
  });

  it("renders header and inventory slots and a resize handle when provided", () => {
    renderWithProviders(
      <Sidebar
        width={320}
        resizeHandleProps={{ role: "separator", "aria-orientation": "vertical" }}
        headerSlot={<span data-testid="hdr">H</span>}
        inventorySlot={<span data-testid="inv">I</span>}
      />,
    );
    expect(screen.getByTestId("hdr")).toBeInTheDocument();
    expect(screen.getByTestId("inv")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-resize-handle")).toBeInTheDocument();
  });
});
