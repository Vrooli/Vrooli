/**
 * StateGraph unit tests.
 *
 * We mock `@xyflow/react` so the test surface is the layout we compute,
 * not the canvas-driven React Flow renderer (which needs ResizeObserver
 * and a real layout box). The mock captures the nodes/edges we hand to
 * <ReactFlow> and exposes them as test ids so we can assert what the
 * component produces deterministically.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";

vi.mock("@xyflow/react/dist/style.css", () => ({}));
vi.mock("@xyflow/react", () => ({
  Background: () => null,
  Position: { Left: "left", Right: "right", Top: "top", Bottom: "bottom" },
  ReactFlow: ({
    nodes,
    edges,
  }: {
    nodes: Array<{ id: string; data: { label: string }; style?: Record<string, unknown> }>;
    edges: Array<{ id: string; source: string; target: string; label?: string }>;
  }) => (
    <div data-testid="rf-mock">
      <ul data-testid="rf-nodes">
        {nodes.map((n) => (
          <li
            key={n.id}
            data-testid={`rf-node-${n.id}`}
            data-border={(n.style as { border?: string }).border ?? ""}
          >
            {n.data.label}
          </li>
        ))}
      </ul>
      <ul data-testid="rf-edges">
        {edges.map((e) => (
          <li key={e.id} data-testid={`rf-edge-${e.source}-${e.target}`}>
            {e.label}
          </li>
        ))}
      </ul>
    </div>
  ),
}));

import { StateGraph } from "./StateGraph";
import type { FlowState, FlowTransition } from "../../api/inventory";

const sampleStates: FlowState[] = [
  { id: "draft", quint: "Draft", initial: true },
  { id: "uploading", quint: "Uploading" },
  { id: "uploaded", quint: "Uploaded", terminal: true },
];

const sampleEvents = [{ id: "begin" }, { id: "complete" }, { id: "cancel" }];

const sampleTransitions: FlowTransition[] = [
  { from: "draft", event: "begin", to: "uploading", wantError: false },
  { from: "uploading", event: "complete", to: "uploaded", wantError: false },
  { from: "uploading", event: "cancel", to: "draft", wantError: false },
  // wantError edge — should NOT appear in the graph (table shows it).
  { from: "uploaded", event: "begin", to: "uploaded", wantError: true },
];

describe("StateGraph", () => {
  afterEach(() => cleanup());

  it("renders an empty card when there are no states", () => {
    renderWithProviders(
      <StateGraph
        states={[]}
        events={[]}
        transitions={[]}
        initialState=""
      />,
    );
    expect(screen.getByTestId("state-graph-empty")).toBeInTheDocument();
  });

  it("emits one node per state and skips wantError edges", () => {
    renderWithProviders(
      <StateGraph
        states={sampleStates}
        events={sampleEvents}
        transitions={sampleTransitions}
        initialState="draft"
      />,
    );
    expect(screen.getByTestId("rf-node-draft")).toBeInTheDocument();
    expect(screen.getByTestId("rf-node-uploading")).toBeInTheDocument();
    expect(screen.getByTestId("rf-node-uploaded")).toBeInTheDocument();
    expect(screen.getByTestId("rf-edge-draft-uploading")).toBeInTheDocument();
    expect(screen.getByTestId("rf-edge-uploading-uploaded")).toBeInTheDocument();
    expect(screen.getByTestId("rf-edge-uploading-draft")).toBeInTheDocument();
    // wantError self-loop is excluded.
    expect(screen.queryByTestId("rf-edge-uploaded-uploaded")).not.toBeInTheDocument();
  });

  it("marks the initial state with the green border", () => {
    renderWithProviders(
      <StateGraph
        states={sampleStates}
        events={sampleEvents}
        transitions={sampleTransitions}
        initialState="draft"
      />,
    );
    expect(screen.getByTestId("rf-node-draft").getAttribute("data-border")).toContain(
      "var(--color-success)",
    );
  });

  it("highlights the activeState with the indigo border", () => {
    renderWithProviders(
      <StateGraph
        states={sampleStates}
        events={sampleEvents}
        transitions={sampleTransitions}
        initialState="draft"
        activeState="uploading"
      />,
    );
    expect(
      screen.getByTestId("rf-node-uploading").getAttribute("data-border"),
    ).toContain("var(--color-primary)");
  });
});
