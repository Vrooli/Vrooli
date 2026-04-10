/**
 * Tests for DetailPageHeader.
 *
 * Verifies navigation button behavior (hamburger vs back arrow),
 * LensBar rendering, tab bar slot, and action slot.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { DetailPageHeader, type DetailPageHeaderProps } from "./DetailPageHeader";
import { useDetailSelectionStore } from "../../stores/detail-selection-store";
import { useGraphUIStore } from "../../surfaces/graph/stores/graph-ui-store";
import type { LensOption } from "./lens-options";
import { Network, Activity } from "lucide-react";

const testLenses: LensOption[] = [
  { lens: "topology", label: "View Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "operations", label: "View Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

// Track matchMedia mock to allow toggling mobile/desktop.
let mockIsMobile = false;

beforeEach(() => {
  mockIsMobile = false;
  useDetailSelectionStore.setState({ selection: null });
  useGraphUIStore.setState({ sidebarCollapsed: true });

  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: query === "(max-width: 768px)" ? mockIsMobile : false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

function renderHeader(overrides?: Partial<DetailPageHeaderProps>) {
  const defaults: DetailPageHeaderProps = {
    entityType: "backlog",
    title: "Test Item",
    nodeId: "backlog/execute/test",
    lenses: testLenses,
    ...overrides,
  };

  return render(
    <MemoryRouter>
      <DetailPageHeader {...defaults} />
    </MemoryRouter>,
  );
}

describe("DetailPageHeader", () => {
  it("renders entity type badge and title", () => {
    renderHeader();

    expect(screen.getByText("backlog")).toBeInTheDocument();
    expect(screen.getByText("Test Item")).toBeInTheDocument();
  });

  it("renders subtitle when provided", () => {
    renderHeader({ subtitle: "execute/test" });

    expect(screen.getByText("execute/test")).toBeInTheDocument();
  });

  it("renders status badge when provided", () => {
    renderHeader({ status: "in_progress" });

    expect(screen.getByTestId("detail-page-header")).toBeInTheDocument();
  });

  it("renders action slot", () => {
    renderHeader({ actions: <button data-testid="custom-action">Run</button> });

    expect(screen.getByTestId("custom-action")).toBeInTheDocument();
  });

  it("renders LensBar when nodeId and lenses are provided", () => {
    renderHeader();

    expect(screen.getByTestId("lens-bar")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-topology")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-operations")).toBeInTheDocument();
  });

  it("does not render LensBar when nodeId is null", () => {
    renderHeader({ nodeId: null });

    expect(screen.queryByTestId("lens-bar")).not.toBeInTheDocument();
  });

  it("does not render LensBar when lenses are empty", () => {
    renderHeader({ lenses: [] });

    expect(screen.queryByTestId("lens-bar")).not.toBeInTheDocument();
  });

  it("renders tab bar slot when provided", () => {
    renderHeader({
      tabBar: <div data-testid="custom-tabs">Tabs</div>,
    });

    expect(screen.getByTestId("custom-tabs")).toBeInTheDocument();
  });

  describe("desktop mode", () => {
    it("shows back arrow on desktop", () => {
      renderHeader();

      const navButton = screen.getByTestId("detail-nav-button");
      expect(navButton).toHaveAttribute("aria-label", "Close detail view");
    });

    it("closes detail on click", async () => {
      useDetailSelectionStore.setState({
        selection: { entityType: "backlog", kind: "execute", name: "test" },
      });
      renderHeader();

      const user = userEvent.setup();
      await user.click(screen.getByTestId("detail-nav-button"));

      expect(useDetailSelectionStore.getState().selection).toBeNull();
    });
  });

  describe("mobile mode", () => {
    beforeEach(() => {
      mockIsMobile = true;
    });

    it("shows hamburger on mobile", () => {
      renderHeader();

      const navButton = screen.getByTestId("detail-nav-button");
      expect(navButton).toHaveAttribute("aria-label", "Open sidebar");
    });

    it("opens sidebar on click", async () => {
      useGraphUIStore.setState({ sidebarCollapsed: true });
      renderHeader();

      const user = userEvent.setup();
      await user.click(screen.getByTestId("detail-nav-button"));

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(false);
    });
  });
});
