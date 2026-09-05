import { act, fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PipelineSidebar } from "./PipelineSidebar";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { useSidebarStore } from "../../store";

const isMobile = vi.hoisted(() => vi.fn(() => false));

vi.mock("../../hooks/useMediaQuery", () => ({ useIsMobile: isMobile }));
vi.mock("./SidebarHeader", () => ({
  SidebarHeader: () => <div>pipeline header</div>,
}));
vi.mock("./SidebarNavigation", () => ({
  SidebarNavigation: ({ collapsed }: { collapsed?: boolean }) => (
    <div>{collapsed ? "collapsed navigation" : "expanded navigation"}</div>
  ),
}));

describe("PipelineSidebar", () => {
  beforeEach(() => {
    isMobile.mockReturnValue(false);
    act(() => {
      useSidebarStore.setState({ collapsed: false });
    });
  });

  it("toggles between expanded and compact desktop navigation", () => {
    renderWithProviders(<PipelineSidebar onSectionClick={vi.fn()} />);

    expect(screen.getByText("pipeline header")).toBeInTheDocument();
    expect(screen.getByText("expanded navigation")).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Collapse pipeline sidebar" }),
    );
    expect(screen.getByText("collapsed navigation")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Expand pipeline sidebar" }),
    ).toBeInTheDocument();
  });

  it("keeps the sidebar expanded and removes the desktop collapse control on mobile", () => {
    isMobile.mockReturnValue(true);
    act(() => {
      useSidebarStore.setState({ collapsed: true });
    });
    renderWithProviders(<PipelineSidebar onSectionClick={vi.fn()} />);

    expect(screen.getByText("expanded navigation")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /pipeline sidebar/ }),
    ).not.toBeInTheDocument();
  });
});
