import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { Node } from "@xyflow/react";
import { Inspector } from "./Inspector";
import { useGraphDataStore, graphDataInitialState } from "../stores/graph-data-store";
import type { GraphLens } from "../stores/graph-data-store";

// jsdom doesn't provide matchMedia.
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

function resetStore(lens: GraphLens = "topology") {
  useGraphDataStore.setState({
    ...graphDataInitialState,
    lens,
    entityFilters: { ...graphDataInitialState.entityFilters },
  });
}

const makeNode = (id: string, entityType: string, status?: string, kind?: string): Node => ({
  id,
  type: entityType,
  position: { x: 0, y: 0 },
  data: { label: `Test ${id}`, entityType, status, kind },
});

function renderInspector(node: Node, lens: GraphLens = "topology") {
  resetStore(lens);
  return render(
    <MemoryRouter initialEntries={["/graph?lens=" + lens]}>
      <Inspector isOpen={true} onClose={vi.fn()} selectedNode={node} />
    </MemoryRouter>,
  );
}

describe("Inspector", () => {
  beforeEach(() => resetStore());

  it("renders nothing when not open", () => {
    const node = makeNode("scenario/test", "scenario", "running");
    render(
      <MemoryRouter>
        <Inspector isOpen={false} onClose={vi.fn()} selectedNode={node} />
      </MemoryRouter>,
    );
    expect(screen.queryByTestId("inspector")).not.toBeInTheDocument();
  });

  it("renders nothing when no selected node", () => {
    render(
      <MemoryRouter>
        <Inspector isOpen={true} onClose={vi.fn()} selectedNode={null} />
      </MemoryRouter>,
    );
    expect(screen.queryByTestId("inspector")).not.toBeInTheDocument();
  });

  it("renders node metadata", () => {
    const node = makeNode("scenario/test", "scenario", "running");
    renderInspector(node);
    const content = screen.getByTestId("inspector-content");
    expect(content).toBeInTheDocument();
    expect(content.textContent).toContain("Test scenario/test");
    expect(content.textContent).toContain("running");
    expect(content.textContent).toContain("scenario");
  });

  // Topology lens: shows actions for all 4 node types.
  it("shows scenario actions on topology lens", () => {
    const node = makeNode("scenario/test", "scenario", "running");
    renderInspector(node, "topology");
    expect(screen.getByTestId("inspector-actions")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-view-scenario-files")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-edit-scenario")).toBeInTheDocument();
  });

  it("shows capture actions on topology lens", () => {
    const node = makeNode("capture/cap-1", "capture", "classified");
    renderInspector(node, "topology");
    expect(screen.getByTestId("inspector-actions")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-classify")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-create-item")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-delete-capture")).toBeInTheDocument();
  });

  it("shows backlog actions on topology lens", () => {
    const node = makeNode("backlog-item/execute/task-a", "backlog", "ready", "execute");
    renderInspector(node, "topology");
    expect(screen.getByTestId("inspector-actions")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-edit-backlog")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-queue")).toBeInTheDocument();
  });

  it("shows initiative actions on topology lens", () => {
    const node = makeNode("initiative/init-1", "initiative", "active");
    renderInspector(node, "topology");
    expect(screen.getByTestId("inspector-actions")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-edit-initiative")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-manage-members")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-archive-initiative")).toBeInTheDocument();
  });

  // Flow lens: shows actions for backlog and execution.
  it("shows queue and view-details for backlog on flow lens", () => {
    const node = makeNode("execute/my-feature", "backlog", "ready", "execute");
    renderInspector(node, "flow");
    expect(screen.getByTestId("inspector-actions")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-queue")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-view-backlog-details")).toBeInTheDocument();
  });

  it("shows execution actions on flow lens", () => {
    const node = makeNode("execution/abc-123", "execution", "completed");
    renderInspector(node, "flow");
    expect(screen.getByTestId("inspector-action-view-execution-details")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-view-prompt-trace")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-follow-up")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-retry")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-trigger-review")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-cancel")).toBeInTheDocument();
  });

  // Operations lens: shows actions for scenario and execution.
  it("shows start/stop/restart for scenario on operations lens", () => {
    const node = makeNode("scenario/swarm-manager", "scenario", "running");
    renderInspector(node, "operations");
    expect(screen.getByTestId("inspector-action-start")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-stop")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-restart")).toBeInTheDocument();
    expect(screen.getByTestId("inspector-action-view-scenario-details")).toBeInTheDocument();
  });

  it("shows stop for agent-run on operations lens", () => {
    const node = makeNode("agent-run/run-456", "agent-run", "running");
    renderInspector(node, "operations");
    expect(screen.getByTestId("inspector-action-stop-run")).toBeInTheDocument();
  });
});
