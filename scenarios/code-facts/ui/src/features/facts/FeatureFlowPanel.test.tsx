import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { searchFeatureFlows } from "../../api/featureFlows";
import { renderWithProviders } from "../../test-utils";
import { FeatureFlowPanel } from "./FeatureFlowPanel";

vi.mock("../../api/featureFlows", () => ({ searchFeatureFlows: vi.fn() }));

describe("FeatureFlowPanel", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders bounded source path, line, graph identity, partial attestation, and runtime limits", async () => {
    vi.mocked(searchFeatureFlows).mockResolvedValue({
      query: "read provider configuration",
      scenario: "search-hub",
      path: "api/handlers",
      results: [{
        id: "code-facts:flow:Search:handlers.go",
        title: "Search source call path",
        text: "Search → loadProvider; DERIVED/PARTIAL: source call edges do not prove runtime reachability",
        score: 0.8,
        path: "api/handlers/query/handlers.go",
        paths: ["api/handlers/query/handlers.go", "api/internal/registry/registry.go"],
        graph_hash: "graph-sha-256",
        attestation: "DERIVED/PARTIAL",
        steps: [
          {
            from: "Search",
            to: "loadProvider",
            call: "loadProvider",
            path: "api/handlers/query/handlers.go",
            to_path: "api/internal/registry/registry.go",
            line: 42,
            graph_hash: "source-graph-1",
            to_graph_hash: "target-graph-1",
            resolution: "source_symbol_resolved",
          },
          {
            from: "loadProvider",
            call: "lookupDynamicOwner",
            path: "api/internal/registry/registry.go",
            line: 81,
            graph_hash: "target-graph-1",
            resolution: "callee_symbol_unmatched",
          },
        ],
      }],
      degradations: [],
    });

    renderWithProviders(<FeatureFlowPanel />);
    fireEvent.change(screen.getByLabelText("Scenario slug"), { target: { value: "search-hub" } });
    fireEvent.change(screen.getByLabelText("Relative source path"), { target: { value: "api/handlers" } });
    fireEvent.change(screen.getByLabelText("Flow question"), { target: { value: "read provider configuration" } });
    fireEvent.click(screen.getByRole("button", { name: "Search source flows" }));

    expect(await screen.findByText("Search source call path")).toBeInTheDocument();
    expect(screen.getByTestId("feature-flow-scope")).toHaveTextContent("scenario search-hub · subtree api/handlers · query read provider configuration");
    expect(screen.getByText(/Caller source api\/handlers\/query\/handlers\.go:42/)).toBeInTheDocument();
    expect(screen.getByText(/target source api\/internal\/registry\/registry\.go · target graph target-graph-1/)).toBeInTheDocument();
    expect(screen.getByText(/graph source-graph-1 · Search calls loadProvider/)).toBeInTheDocument();
    expect(screen.getByText(/registry\/registry\.go:81/)).toBeInTheDocument();
    expect(screen.getByText(/unresolved callee; no target path \(callee_symbol_unmatched\)/)).toBeInTheDocument();
    expect(screen.getByText(/Source graph: graph-sha-256 · attestation: DERIVED\/PARTIAL/)).toBeInTheDocument();
    expect(screen.getByText(/Up to 5 results and 3 source call-site edges in one selected scenario subtree/)).toBeInTheDocument();
    expect(screen.getByText(/Each edge shows its caller location and graph identity, plus a resolved target location when available/)).toBeInTheDocument();
    expect(screen.getByText(/Static edges are partial evidence: they do not prove runtime entry, dynamic dispatch, execution conditions, complete flow, or reachability/)).toBeInTheDocument();
    await waitFor(() => expect(searchFeatureFlows).toHaveBeenCalledWith("read provider configuration", "search-hub", "api/handlers", 5));
  });

  it.each([
    ["no_matching_flow_entry", "No source function matched this query."],
    ["call_graph_unavailable", "The selected source graph has no function and call nodes."],
  ])("shows %s degradation without claiming runtime absence", async (code, message) => {
    vi.mocked(searchFeatureFlows).mockResolvedValue({
      query: "unrelated feature",
      scenario: "search-hub",
      path: "api",
      results: [],
      degradations: [{ code, message }],
    });

    renderWithProviders(<FeatureFlowPanel />);
    fireEvent.click(screen.getByRole("button", { name: "Search source flows" }));

    expect(await screen.findByText(/No matching source flow was found in this bounded graph/)).toBeInTheDocument();
    expect(screen.getByText(`${code}: ${message}`)).toBeInTheDocument();
    expect(screen.getByText(/This is not proof that no runtime path exists/)).toBeInTheDocument();
    expect(screen.getByTestId("feature-flow-scope")).toHaveTextContent("scenario search-hub · subtree api · query unrelated feature");
  });
});
