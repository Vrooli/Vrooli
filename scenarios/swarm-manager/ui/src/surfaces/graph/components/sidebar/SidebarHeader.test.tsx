/**
 * Tests for SidebarHeader.
 *
 * Verifies home button behavior, settings button, and collapse button.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SidebarHeader } from "./SidebarHeader";

// Mock AgentsDropdown to avoid store complexity.
vi.mock("../../../../components/agents/AgentsDropdown", () => ({
  AgentsDropdown: () => <div data-testid="agents-dropdown" />,
}));

// Mock useCommandPostBadgeCount to avoid QueryClientProvider dependency.
vi.mock("../../../../hooks/useCommandPostBadgeCount", () => ({
  useCommandPostBadgeCount: () => 0,
}));

// Mock react-query to avoid QueryClientProvider dependency.
vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual("@tanstack/react-query");
  return { ...actual, useQuery: () => ({ data: undefined, isLoading: false }) };
});

function renderHeader(overrides?: Partial<React.ComponentProps<typeof SidebarHeader>>) {
  return render(
    <SidebarHeader
      onSettingsOpen={vi.fn()}
      onCollapse={vi.fn()}
      onViewActivity={vi.fn()}
      onViewBacklog={vi.fn()}
      onGoHome={vi.fn()}
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

  it("calls onGoHome when home button is clicked", async () => {
    const onGoHome = vi.fn();
    renderHeader({ onGoHome });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("sidebar-home"));

    expect(onGoHome).toHaveBeenCalledOnce();
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
