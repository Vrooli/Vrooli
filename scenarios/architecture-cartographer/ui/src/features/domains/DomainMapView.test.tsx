import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { DomainSource } from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";

vi.mock("./controllers/useDomainsController", () => ({
  useGetDomainMap: vi.fn(),
  useExtractDomains: vi.fn(),
}));

vi.mock("../graph/controllers/useGraphController", () => ({
  useSearchDomainSlice: vi.fn(),
}));

vi.mock("./ConvergenceReport", () => ({
  ConvergenceReport: ({ scenario }: { scenario: string }) => (
    <div data-testid="convergence-report">{scenario}</div>
  ),
}));

vi.mock("./BoundaryHealth", () => ({
  BoundaryHealth: ({ scenario }: { scenario: string }) => (
    <div data-testid="boundary-health">{scenario}</div>
  ),
}));

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { useExtractDomains, useGetDomainMap } from "./controllers/useDomainsController";
import { useSearchDomainSlice } from "../graph/controllers/useGraphController";
import { DomainMapView } from "./DomainMapView";

afterEach(() => {
  cleanup();
  vi.mocked(useGetDomainMap).mockReset();
  vi.mocked(useExtractDomains).mockReset();
  vi.mocked(useSearchDomainSlice).mockReset();
});

function mockDomainMap(state: Partial<ReturnType<typeof useGetDomainMap>>) {
  vi.mocked(useGetDomainMap).mockReturnValue({
    isPending: false,
    isError: false,
    data: { domainMap: undefined },
    error: null,
    refetch: vi.fn(),
    ...state,
  } as unknown as ReturnType<typeof useGetDomainMap>);
}

function mockExtract(state: Partial<ReturnType<typeof useExtractDomains>> = {}) {
  vi.mocked(useSearchDomainSlice).mockReturnValue({
    isPending: false,
    isError: false,
    data: { results: [], degradations: [] },
  } as never);
  const mutate = vi.fn();
  vi.mocked(useExtractDomains).mockReturnValue({
    isPending: false,
    mutate,
    ...state,
  } as unknown as ReturnType<typeof useExtractDomains>);
  return mutate;
}

describe("DomainMapView", () => {
  it("renders loading, error, and empty states", () => {
    mockExtract();
    mockDomainMap({ isPending: true });
    const { rerender } = renderWithProviders(<DomainMapView scenario="demo" />);
    expect(screen.getByTestId(selectors.features.domains.view.loading)).toBeInTheDocument();

    mockDomainMap({ isError: true, error: new Error("domains unavailable") });
    rerender(<DomainMapView scenario="demo" />);
    expect(screen.getByTestId(selectors.features.domains.view.error)).toHaveTextContent(
      "domains unavailable",
    );

    mockDomainMap({ data: { domainMap: undefined } as never });
    rerender(<DomainMapView scenario="demo" />);
    expect(screen.getByTestId(selectors.features.domains.view.empty)).toBeInTheDocument();
  });

  it("renders a populated map and triggers extraction", async () => {
    const user = userEvent.setup();
    const mutate = mockExtract();
    mockDomainMap({
      data: {
        domainMap: {
          domains: [
            {
              name: "graph",
              paths: ["api/internal/graph"],
              archetype: "core",
              provenance: [DomainSource.API_MANIFEST, DomainSource.DOMAINS_DOC],
            },
            {
              name: "signals",
              paths: [],
              archetype: "",
              provenance: [
                DomainSource.API_FOLDERS,
                DomainSource.CLI_GROUPS,
                DomainSource.UI_FEATURES,
                DomainSource.UNSPECIFIED,
              ],
            },
          ],
          sharedSubstrate: ["api/internal/httpc"],
          nonDomains: ["tmp"],
          declarations: [
            {
              source: DomainSource.API_MANIFEST,
              authoritative: false,
              domainNames: ["graph"],
            },
            {
              source: DomainSource.DOMAINS_DOC,
              authoritative: true,
              domainNames: ["signals"],
            },
            {
              source: DomainSource.API_FOLDERS,
              authoritative: false,
              domainNames: [],
            },
            {
              source: DomainSource.CLI_GROUPS,
              authoritative: false,
              domainNames: [],
            },
            {
              source: DomainSource.UI_FEATURES,
              authoritative: false,
              domainNames: [],
            },
            {
              source: DomainSource.UNSPECIFIED,
              authoritative: false,
              domainNames: [],
            },
          ],
        },
      } as never,
    });

    renderWithProviders(<DomainMapView scenario="demo" />);

    const root = screen.getByTestId(selectors.features.domains.view.root);
    expect(root).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.domains.table.root)).toBeInTheDocument();
    expect(root).toHaveTextContent("graph");
    expect(root).toHaveTextContent("api/internal/graph");
    expect(root).toHaveTextContent("api/internal/httpc");
    expect(root).toHaveTextContent("tmp");
    expect(root).toHaveTextContent("api manifest");
    expect(root).toHaveTextContent("DOMAINS.md");
    expect(root).toHaveTextContent("api/internal folders");
    expect(root).toHaveTextContent("cli groups");
    expect(root).toHaveTextContent("ui features");
    expect(root).toHaveTextContent("unspecified");
    expect(screen.getByTestId("convergence-report")).toHaveTextContent("demo");
    expect(screen.getByTestId("boundary-health")).toHaveTextContent("demo");

    await user.click(screen.getByTestId(selectors.features.domains.view.extractButton));
    expect(mutate).toHaveBeenCalledTimes(1);
  });

  it("renders five-rung evidence, declared ownership, and bounded layer handoffs", () => {
    mockExtract();
    mockDomainMap({ data: { domainMap: {
      domains: [{ name: "graph", paths: ["api/internal/graph"], responsibility: "Own graph snapshots and imports", purpose: "Map domain code", ownsData: "Graph snapshots", archetypes: [], provenance: [DomainSource.API_MANIFEST, DomainSource.DOMAINS_DOC] }],
      sharedSubstrate: [], nonDomains: [], declarations: [],
    } } as never });
    vi.mocked(useSearchDomainSlice).mockReturnValue({
      isPending: false,
      isError: false,
      data: {
        query: "graph", scope: "scenario:demo",
        results: [{ id: "demo/graph", title: "demo/graph implementation slice", score: 1, text: "DERIVED/PARTIAL", path: "api/internal/graph/service.go", paths: ["api/internal/graph/service.go"], scenario: "demo", domain: "graph", snapshot_id: "snap:demo:h1", slice: {
          scenario: "demo", domain: "graph", snapshot_id: "snap:demo:h1", surfaces: ["api"],
          rungs: ["proto", "handler", "internal", "cli", "ui"].map((name) => ({ name, present: name === "handler", files: [{ path: `src/${name}.go`, lines: 12, is_test: false }], symbols: [{ name: `${name}Entry`, kind: "func", file: `src/${name}.go` }], evidence: [{ kind: "package", value: `api/internal/${name}`, source: "code" }] })),
          layer_edges: [{ from_rung: "handler", to_rung: "internal", kind: "import" }],
          attestation: { claim: "A partial graph-derived vertical slice.", basis: 1, sufficiency: 2, citations: [
            { locator: "api/internal/internal", kind: "code", note: "Package evidence" },
            { locator: "outside/bounded-sample.go", kind: "code", note: "Citation omitted from bounded rung rows" },
          ], gaps: ["Symbol-level links are not resolved."], suggested_follow_ups: [] },
        } }], degradations: [],
      },
    } as never);

    renderWithProviders(<DomainMapView scenario="demo" />);

    const region = screen.getByRole("region", { name: "Implementation slice anatomy" });
    expect(region).toHaveTextContent("DERIVED / PARTIAL");
    expect(region).toHaveTextContent("Owns data: Graph snapshots");
    expect(region).toHaveTextContent("api manifest, DOMAINS.md");
    expect(region).toHaveTextContent("proto · missing");
    expect(region).toHaveTextContent("handler · present");
    expect(region).toHaveTextContent("Graph snapshots");
    expect(region).toHaveTextContent("graphEntry (func) — src/graph.go");
    expect(region).toHaveTextContent("api/internal/handler · code");
    expect(region).toHaveTextContent("handler → internal: import observed");
    expect(region).toHaveTextContent("no direct edge reported");
    expect(region).toHaveTextContent("Symbol-level links are not resolved.");
    expect(screen.getByRole("link", { name: "internal rung evidence row" })).toHaveAttribute("href", "#slice-evidence-graph-internal-0");
    expect(region).toHaveTextContent("Not linked to a returned rung evidence row (bounded sample).");
    expect(region).toHaveTextContent("Snapshot age/freshness and current coverage are not established by this view.");
    expect(useSearchDomainSlice).toHaveBeenCalledWith("demo", "graph");
  });

  it("surfaces the no-cached-snapshot degradation without offering extraction", () => {
    mockExtract();
    mockDomainMap({ data: { domainMap: {
      domains: [{ name: "graph", paths: [], archetypes: [], provenance: [] }],
      sharedSubstrate: [], nonDomains: [], declarations: [],
    } } as never });
    vi.mocked(useSearchDomainSlice).mockReturnValue({
      isPending: false,
      isError: false,
      data: { query: "graph", scope: "scenario:demo", results: [], degradations: [{ code: "graph_snapshot_unavailable", message: "No cached graph snapshot exists." }] },
    } as never);

    renderWithProviders(<DomainMapView scenario="demo" />);

    const region = screen.getByRole("region", { name: "Implementation slice anatomy" });
    expect(region).toHaveTextContent("graph_snapshot_unavailable");
    expect(region).toHaveTextContent("No extraction was attempted by this search view.");
  });

  it("keeps unavailable snapshot requests degraded when the owner query fails", () => {
    mockExtract();
    mockDomainMap({ data: { domainMap: {
      domains: [{ name: "graph", paths: [], archetypes: [], provenance: [] }],
      sharedSubstrate: [], nonDomains: [], declarations: [],
    } } as never });
    vi.mocked(useSearchDomainSlice).mockReturnValue({
      isPending: false,
      isError: true,
      error: new Error("cached graph snapshot is stale or unavailable"),
      data: undefined,
    } as never);

    renderWithProviders(<DomainMapView scenario="demo" />);

    const region = screen.getByRole("region", { name: "Implementation slice anatomy" });
    expect(region).toHaveTextContent("Slice search unavailable: cached graph snapshot is stale or unavailable");
    expect(region).toHaveTextContent("Snapshot age/freshness and current coverage are not established by this view.");
    expect(region).not.toHaveTextContent("No matching slice means no implementation exists");
  });

});
