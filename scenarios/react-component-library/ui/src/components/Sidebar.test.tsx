import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { SidebarContent } from "./Sidebar";

describe("SidebarContent", () => {
  afterEach(() => cleanup());

  it("renders the brand without a redundant route list", () => {
    renderWithProviders(<SidebarContent />);
    expect(screen.getByTestId("app-sidebar-content")).toBeInTheDocument();
    expect(screen.getByTestId("app-brand")).toHaveAttribute("href", "/");
    expect(screen.queryByTestId("nav-dashboard")).not.toBeInTheDocument();
    expect(screen.queryByTestId("nav-components")).not.toBeInTheDocument();
    expect(screen.queryByTestId("nav-adoptions")).not.toBeInTheDocument();
  });

  it("renders header and inventory slots when provided", () => {
    renderWithProviders(
      <SidebarContent
        headerSlot={<span data-testid="hdr">H</span>}
        inventorySlot={<span data-testid="inv">I</span>}
      />,
    );
    expect(screen.getByTestId("hdr")).toBeInTheDocument();
    expect(screen.getByTestId("inv")).toBeInTheDocument();
  });
});
