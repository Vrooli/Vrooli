/**
 * Tests for SidebarHeader.
 *
 * Verifies home, settings, and collapse handlers fire, and pins that the
 * home button carries the running-agent badge (no standalone agents pill).
 */

import { beforeEach, describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { SidebarHeader } from "./SidebarHeader";
import { useOperationsStore } from "../../../../stores/operations-store";

// Mock useCommandPostBadgeCount to avoid QueryClientProvider dependency.
vi.mock("../../../../hooks/useCommandPostBadgeCount", () => ({
  useCommandPostBadgeCount: () => 0,
}));

function renderHeader(overrides?: Partial<React.ComponentProps<typeof SidebarHeader>>) {
  return render(
    <MemoryRouter>
      <SidebarHeader
        onSettingsOpen={vi.fn()}
        onCollapse={vi.fn()}
        onGoHome={vi.fn()}
        {...overrides}
      />
    </MemoryRouter>,
  );
}

describe("SidebarHeader", () => {
  beforeEach(() => {
    useOperationsStore.getState().reset();
  });

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

  it("badges the home button with the active agent count", () => {
    useOperationsStore.setState({
      view: {
        lanes: [],
        queue: { depth: 0, maxDepth: 50 },
        activities: [{ runId: "run-1" }, { runId: "run-2" }],
        recentlyFinished: [],
        generatedAt: "2026-07-07T00:00:00Z",
        windowSeconds: 10800,
      } as never,
    });

    renderHeader();

    expect(screen.getByTestId("sidebar-home-agent-badge")).toHaveTextContent("2");
  });

  it("hides the home badge and renders no agents pill when idle", () => {
    renderHeader();

    expect(screen.queryByTestId("sidebar-home-agent-badge")).toBeNull();
    expect(screen.queryByTestId("layout-ops-trigger-button")).toBeNull();
    expect(screen.queryByTestId("agents-badge")).toBeNull();
    expect(screen.queryByTestId("agents-dropdown")).toBeNull();
  });
});
