import { describe, it, expect, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes, useLocation } from "react-router-dom";
import { Network, Activity } from "lucide-react";
import { DetailPageHeader, type DetailPageHeaderProps } from "./DetailPageHeader";
import type { LensOption } from "./lens-options";
import { installMatchMediaMock, renderWithProviders } from "../../test-utils";
import { useGraphUIStore } from "../../surfaces/graph/stores/graph-ui-store";

const testLenses: LensOption[] = [
  { lens: "topology", label: "View Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "operations", label: "View Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

beforeEach(() => {
  installMatchMediaMock();
  // Default to a collapsed sidebar so the existing tests that assert the
  // hamburger is present continue to pass; specific tests override below.
  useGraphUIStore.setState({ sidebarCollapsed: true });
});

function renderHeader(overrides?: Partial<DetailPageHeaderProps>) {
  const defaults: DetailPageHeaderProps = {
    entityType: "backlog",
    title: "Test Item",
    nodeId: "backlog-item/execute/test",
    lenses: testLenses,
    ...overrides,
  };

  return renderWithProviders(<DetailPageHeader {...defaults} />, {
    initialEntries: ["/graph", "/backlog/execute/test"],
    initialIndex: 1,
  });
}

function LocationProbe() {
  const location = useLocation();
  return <span data-testid="location-path">{location.pathname}</span>;
}

describe("DetailPageHeader", () => {
  it("renders entity type badge and title", () => {
    renderHeader();

    expect(screen.getByText("backlog")).toBeInTheDocument();
    expect(screen.getByText("Test Item")).toBeInTheDocument();
  });

  it("opens a popover with the full title and a copy button when the title is clicked", async () => {
    const longTitle = "Define the canonical sandbox auditability contract for agent-manager";
    renderHeader({ title: longTitle });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("detail-title-button"));

    expect(screen.getByTestId("detail-title-popover")).toBeInTheDocument();
    expect(screen.getByTestId("detail-title-popover-text")).toHaveTextContent(longTitle);
    expect(screen.getByTestId("detail-title-copy-button")).toBeInTheDocument();
  });

  it("renders subtitle, status, and action slot when provided", () => {
    renderHeader({
      subtitle: "execute/test",
      status: "in_progress",
      metadata: <span data-testid="custom-metadata">Created by session</span>,
      actions: <button data-testid="custom-action">Run</button>,
    });

    expect(screen.getByText("execute/test")).toBeInTheDocument();
    expect(screen.getByTestId("detail-page-header")).toBeInTheDocument();
    expect(screen.getByTestId("custom-metadata")).toBeInTheDocument();
    expect(screen.getByTestId("custom-action")).toBeInTheDocument();
  });

  it("renders LensBar when nodeId and lenses are provided", () => {
    renderHeader();

    expect(screen.getByTestId("lens-bar")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-topology")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-operations")).toBeInTheDocument();
  });

  it("does not render LensBar when nodeId is null or lenses are empty", () => {
    const { rerender } = renderWithProviders(
      <DetailPageHeader entityType="backlog" title="Test Item" nodeId={null} lenses={testLenses} />,
    );
    expect(screen.queryByTestId("lens-bar")).not.toBeInTheDocument();

    rerender(
      <DetailPageHeader entityType="backlog" title="Test Item" nodeId="backlog-item/execute/test" lenses={[]} />,
    );
    expect(screen.queryByTestId("lens-bar")).not.toBeInTheDocument();
  });

  it("renders tab bar slot when provided", () => {
    renderHeader({ tabBar: <div data-testid="custom-tabs">Tabs</div> });

    expect(screen.getByTestId("custom-tabs")).toBeInTheDocument();
  });

  it("hides the hamburger button when the sidebar is open", () => {
    useGraphUIStore.setState({ sidebarCollapsed: false });
    renderHeader();
    expect(screen.queryByTestId("page-sidebar-button")).toBeNull();
  });

  it("shows the hamburger button when the sidebar is collapsed", () => {
    useGraphUIStore.setState({ sidebarCollapsed: true });
    renderHeader();
    expect(screen.getByTestId("page-sidebar-button")).toBeInTheDocument();
  });

  it("uses route back semantics for the nav button", async () => {
    renderWithProviders(
      <Routes>
        <Route
          path="*"
          element={(
            <>
              <DetailPageHeader entityType="backlog" title="Test Item" nodeId={null} lenses={[]} />
              <LocationProbe />
            </>
          )}
        />
      </Routes>,
      {
        initialEntries: ["/graph", "/backlog/execute/test"],
        initialIndex: 1,
      },
    );

    expect(screen.getByTestId("page-sidebar-button")).toHaveAttribute("aria-label", "Open sidebar");
    expect(screen.getByTestId("detail-nav-button")).toHaveAttribute("aria-label", "Close page");

    const user = userEvent.setup();
    await user.click(screen.getByTestId("detail-nav-button"));

    expect(screen.getByTestId("location-path")).toHaveTextContent("/graph");
  });
});
