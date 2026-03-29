import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import React from "react";

// Mock lazy-loaded pages — use default export so React.lazy resolves immediately.
vi.mock("../../../pages/SettingsPage", () => ({
  SettingsPage: () => <div data-testid="settings-page" />,
  default: () => <div data-testid="settings-page" />,
}));
vi.mock("../../../pages/PromptsPage", () => ({
  PromptsPage: () => <div data-testid="prompts-page" />,
  default: () => <div data-testid="prompts-page" />,
}));

// Stub floating panel to render children directly.
vi.mock("../../../components/ui/floating-panel", () => ({
  FloatingPanel: ({ children, isOpen }: { children: React.ReactNode; isOpen: boolean }) =>
    isOpen ? <div data-testid="floating-panel">{children}</div> : null,
}));

import { useGraphDataStore, cloneGraphDataInitialState } from "../stores/graph-data-store";
import { makeBacklogNode, makeExecutionNode, makeScenarioNode } from "../test-helpers";
import { SettingsDrawer } from "./SettingsDrawer";

function resetStore() {
  useGraphDataStore.setState(cloneGraphDataInitialState());
  window.localStorage.clear();
}

describe("SettingsDrawer — entity node counts", () => {
  beforeEach(resetStore);

  it("shows the number of nodes per entity type, not the number of status types", async () => {
    // Seed the store with 3 backlog + 2 execution + 1 scenario = 6 nodes
    useGraphDataStore.setState({
      nodes: [
        makeBacklogNode("backlog-item/execute/a", { status: "completed" }),
        makeBacklogNode("backlog-item/execute/b", { status: "backlog" }),
        makeBacklogNode("backlog-item/fix/c", { status: "in_progress" }),
        makeExecutionNode("execution/e1", { status: "running" }),
        makeExecutionNode("execution/e2", { status: "needs_fixup" }),
        makeScenarioNode("scenario/sm", { status: "running" }),
      ],
    });

    render(<SettingsDrawer isOpen onClose={vi.fn()} />);

    // Wait for lazy components to resolve so Suspense reveals content.
    await waitFor(() => expect(screen.getByText("Statuses")).toBeInTheDocument());

    // Status accordion headers live inside the "Statuses" section. Each
    // accordion button contains the entity label + "(N)" where N is now the
    // node count, NOT the number of status types.
    const statusSection = screen.getByText("Statuses").closest("section");
    expect(statusSection).not.toBeNull();
    const accordionButtons = statusSection?.querySelectorAll<HTMLButtonElement>(
      "button:first-child",
    ) ?? [];

    // Find the accordion for each entity type by label
    const findAccordion = (label: string) =>
      Array.from(accordionButtons).find((btn) => btn.textContent?.includes(label));

    const backlogBtn = findAccordion("Backlog");
    expect(backlogBtn?.textContent).toContain("(3)");

    const executionBtn = findAccordion("Execution");
    expect(executionBtn?.textContent).toContain("(2)");

    const scenarioBtn = findAccordion("Scenarios");
    expect(scenarioBtn?.textContent).toContain("(1)");
  });

  it("shows (0) when there are no nodes of that entity type", async () => {
    useGraphDataStore.setState({ nodes: [] });

    render(<SettingsDrawer isOpen onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByText("Statuses")).toBeInTheDocument());

    const statusSection = screen.getByText("Statuses").closest("section");
    expect(statusSection).not.toBeNull();
    const accordionButtons = statusSection?.querySelectorAll<HTMLButtonElement>(
      "button:first-child",
    ) ?? [];
    const backlogBtn = Array.from(accordionButtons).find((btn) =>
      btn.textContent?.includes("Backlog"),
    );
    expect(backlogBtn?.textContent).toContain("(0)");
  });
});
