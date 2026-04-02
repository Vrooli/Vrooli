/**
 * Tests for SidebarHeader.
 *
 * Verifies home button behavior, settings button, and collapse button.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SidebarHeader } from "./SidebarHeader";
import { useDetailSelectionStore } from "../../../../stores/detail-selection-store";
import { useGraphUIStore } from "../../stores/graph-ui-store";

// Mock AgentsDropdown to avoid store complexity.
vi.mock("../../../../components/agents/AgentsDropdown", () => ({
  AgentsDropdown: () => <div data-testid="agents-dropdown" />,
}));

// Mock useCommandPostBadgeCount to avoid QueryClientProvider dependency.
vi.mock("../../../../hooks/useCommandPostBadgeCount", () => ({
  useCommandPostBadgeCount: () => 0,
}));

beforeEach(() => {
  useDetailSelectionStore.setState({ selection: null });
  useGraphUIStore.setState({ sidebarCollapsed: false });
});

function renderHeader(overrides?: Partial<React.ComponentProps<typeof SidebarHeader>>) {
  return render(
    <SidebarHeader
      onSettingsOpen={vi.fn()}
      onCollapse={vi.fn()}
      onViewActivity={vi.fn()}
      onViewBacklog={vi.fn()}
      {...overrides}
    />,
  );
}

describe("SidebarHeader", () => {
  it("renders home button and app title", () => {
    renderHeader();

    expect(screen.getByTestId("sidebar-home")).toBeInTheDocument();
    expect(screen.getByText("Swarm Manager")).toBeInTheDocument();
  });

  it("home button clears detail selection and collapses sidebar", async () => {
    useDetailSelectionStore.setState({
      selection: { entityType: "backlog", kind: "fix", name: "test" },
    });
    useGraphUIStore.setState({ sidebarCollapsed: false });

    renderHeader();

    const user = userEvent.setup();
    await user.click(screen.getByTestId("sidebar-home"));

    expect(useDetailSelectionStore.getState().selection).toBeNull();
    expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
  });

  it("calls onSettingsOpen when settings button is clicked", async () => {
    const onSettingsOpen = vi.fn();
    renderHeader({ onSettingsOpen });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("sidebar-settings"));

    expect(onSettingsOpen).toHaveBeenCalledOnce();
  });

  it("calls onCollapse when collapse button is clicked", async () => {
    const onCollapse = vi.fn();
    renderHeader({ onCollapse });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("sidebar-toggle-close"));

    expect(onCollapse).toHaveBeenCalledOnce();
  });

  it("renders agents dropdown", () => {
    renderHeader();

    expect(screen.getByTestId("agents-dropdown")).toBeInTheDocument();
  });
});
