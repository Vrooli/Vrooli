import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { SidebarContent } from "./Sidebar";

describe("SidebarContent", () => {
  afterEach(() => cleanup());

  it("renders brand and primary nav targets", () => {
    renderWithProviders(<SidebarContent />);
    expect(screen.getByTestId("app-sidebar-content")).toBeInTheDocument();
    expect(screen.getByTestId("app-brand")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("nav-dashboard")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("nav-components")).toHaveAttribute("href", "/components");
    expect(screen.getByTestId("nav-adoptions")).toHaveAttribute("href", "/adoptions");
    expect(screen.getByTestId("nav-settings")).toHaveAttribute("href", "/settings");
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
