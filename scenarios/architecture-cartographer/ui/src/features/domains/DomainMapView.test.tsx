import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { DomainSource } from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";

vi.mock("./controllers/useDomainsController", () => ({
  useGetDomainMap: vi.fn(),
  useExtractDomains: vi.fn(),
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
import { DomainMapView } from "./DomainMapView";

afterEach(() => {
  cleanup();
  vi.mocked(useGetDomainMap).mockReset();
  vi.mocked(useExtractDomains).mockReset();
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
          sharedSubstrate: ["api/internal/httpx"],
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
    expect(root).toHaveTextContent("api/internal/httpx");
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
});
