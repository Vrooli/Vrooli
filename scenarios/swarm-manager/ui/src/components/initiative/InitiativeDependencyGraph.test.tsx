import { describe, it, expect, beforeAll, afterEach } from "vitest";
import { cleanup } from "@testing-library/react";
import { InitiativeDependencyGraph } from "./InitiativeDependencyGraph";
import type { BacklogStatus } from "../../types";
import {
  installMatchMediaMock,
  installResizeObserverMock,
  renderWithProviders,
} from "../../test-utils";

beforeAll(() => {
  // ReactFlow needs ResizeObserver, and the graph shell checks matchMedia.
  installResizeObserverMock();
  installMatchMediaMock();
});

afterEach(() => cleanup());

function item(name: string, dependsOn: string[] = [], status: BacklogStatus = "backlog") {
  return {
    kind: "execute",
    name,
    title: name,
    status,
    dependsOn,
  };
}

describe("InitiativeDependencyGraph overlay", () => {
  it("renders with no overlay (baseline path)", () => {
    const items = [item("a"), item("b", ["execute/a"])];
    const { container } = renderWithProviders(<InitiativeDependencyGraph items={items} />);
    // ReactFlow mounts a container div; overlay path still produces output.
    expect(container.firstChild).toBeTruthy();
  });

  it("accepts an overlay with added/archived/status changes without crashing", () => {
    const items = [item("a"), item("b", ["execute/a"])];
    const { container } = renderWithProviders(
      <InitiativeDependencyGraph
        items={items}
        overlay={{
          addedNodes: [{ id: "execute/new", kind: "execute", name: "new", title: "New" }],
          addedNodeIds: ["execute/new"],
          archivedNodeIds: ["execute/a"],
          statusChanges: { "execute/b": "ready" },
          addedEdges: [{ from: "execute/new", to: "execute/b" }],
          removedEdges: [{ from: "execute/a", to: "execute/b" }],
        }}
      />,
    );
    expect(container.firstChild).toBeTruthy();
    // With overlay adds, the "No dependencies" placeholder should not render.
    expect(container.textContent).not.toContain("No dependencies between items");
  });

  it("renders the diff badge label when a status_change is present", () => {
    const items = [item("a"), item("b", ["execute/a"])];
    const { container } = renderWithProviders(
      <InitiativeDependencyGraph
        items={items}
        overlay={{
          statusChanges: { "execute/a": "ready" },
        }}
      />,
    );
    // ReactFlow renders the custom node content as HTML — the "Status" label
    // should appear in the DOM somewhere under the node markup.
    expect(container.textContent).toContain("Status");
  });
});
