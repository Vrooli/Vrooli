import { describe, it, expect, beforeAll, vi, afterEach } from "vitest";
import { render, cleanup } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { InitiativeDependencyGraph } from "./InitiativeDependencyGraph";
import type { BacklogStatus } from "../../types";

beforeAll(() => {
  // ReactFlow needs ResizeObserver + getBoundingClientRect.
  class ROShim {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  (globalThis as unknown as { ResizeObserver: typeof ROShim }).ResizeObserver = ROShim;
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  });
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
    const { container } = render(
      <MemoryRouter>
        <InitiativeDependencyGraph items={items} />
      </MemoryRouter>,
    );
    // ReactFlow mounts a container div; overlay path still produces output.
    expect(container.firstChild).toBeTruthy();
  });

  it("accepts an overlay with added/archived/status changes without crashing", () => {
    const items = [item("a"), item("b", ["execute/a"])];
    const { container } = render(
      <MemoryRouter>
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
        />
      </MemoryRouter>,
    );
    expect(container.firstChild).toBeTruthy();
    // With overlay adds, the "No dependencies" placeholder should not render.
    expect(container.textContent).not.toContain("No dependencies between items");
  });

  it("renders the diff badge label when a status_change is present", () => {
    const items = [item("a"), item("b", ["execute/a"])];
    const { container } = render(
      <MemoryRouter>
        <InitiativeDependencyGraph
          items={items}
          overlay={{
            statusChanges: { "execute/a": "ready" },
          }}
        />
      </MemoryRouter>,
    );
    // ReactFlow renders the custom node content as HTML — the "Status" label
    // should appear in the DOM somewhere under the node markup.
    expect(container.textContent).toContain("Status");
  });
});
