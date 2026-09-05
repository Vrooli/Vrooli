/**
 * GraphPage degradation tests.
 *
 * The graph surface consumes scenario-dependency-analyzer (SDA) via the
 * API. When SDA is down or returns an empty graph, the page MUST still
 * paint meaningful, visible content (heading + metrics + an explicit
 * error/empty panel) rather than going blank. These tests lock in that
 * graceful-degradation contract so a smoke capture never sees a blank
 * solid-color render.
 *
 * describeTechTree is mocked so the tests drive the error and empty paths
 * deterministically without a live SDA. The GraphCanvas (ReactFlow/ELK)
 * heavy path is intentionally not exercised here — the empty path renders
 * the lightweight empty panel, and the error path never mounts the canvas.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";

const describeTechTree = vi.fn();

vi.mock("../../api/techTree", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/techTree")>();
  return {
    ...actual,
    describeTechTree: (...args: unknown[]) => describeTechTree(...args),
    exportTechTree: vi.fn(),
  };
});

import { GraphPage } from "./GraphPage";

afterEach(() => {
  cleanup();
  describeTechTree.mockReset();
});

describe("GraphPage graceful degradation", () => {
  it("always paints the heading and metrics even before data resolves", () => {
    describeTechTree.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<GraphPage />);

    expect(screen.getByTestId(selectors.pages.graph)).toBeInTheDocument();
    expect(screen.getByText(strings.graph.title)).toBeInTheDocument();
    // Four metric tiles render unconditionally — never a blank canvas.
    expect(screen.getByText(strings.graph.metrics.live)).toBeInTheDocument();
    expect(screen.getByText(strings.graph.metrics.warnings)).toBeInTheDocument();
  });

  it("renders a visible error panel when the SDA-backed query fails", async () => {
    describeTechTree.mockRejectedValue(new Error("scenario-dependency-analyzer unavailable"));
    renderWithProviders(<GraphPage />);

    await waitFor(() => {
      expect(screen.getByText(strings.graph.states.error)).toBeInTheDocument();
    });
    // Heading still present — the page degrades, it does not blank out.
    expect(screen.getByText(strings.graph.title)).toBeInTheDocument();
  });

  it("renders the empty-state panel when SDA returns a graph with no nodes", async () => {
    describeTechTree.mockResolvedValue({
      graph: { nodes: [], edges: [], errors: [] },
    });
    renderWithProviders(<GraphPage />);

    await waitFor(() => {
      expect(screen.getByText(strings.graph.states.empty)).toBeInTheDocument();
    });
    expect(screen.getByText(strings.graph.title)).toBeInTheDocument();
  });

  it("surfaces source warnings (GraphError) when SDA reports them", async () => {
    describeTechTree.mockResolvedValue({
      graph: {
        nodes: [],
        edges: [],
        errors: [{ source: "sda", scenario: "web-console", message: "parse failed" }],
      },
    });
    renderWithProviders(<GraphPage />);

    await waitFor(() => {
      expect(screen.getByText(strings.graph.warnings.title)).toBeInTheDocument();
    });
  });
});
