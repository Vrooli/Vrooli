/**
 * SearchPanel tests — the federated-search surface in isolation.
 *
 * Renders <SearchPanel /> directly with the ./api/search boundary mocked, so
 * failures point at search-feature behaviour rather than transport. Copy is
 * asserted through the strings registry / test ids (cimode), never translated
 * literals.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  QueryResponseSchema,
  StatusResponseSchema,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";
import {
  ProviderDescriptorSchema,
  ProviderState,
} from "@vrooli/proto-types/search-hub/v1/registry/registry_pb";

import { renderWithProviders } from "../../test-utils";

// Provider-supplied result data (NOT user-facing copy) — a const so the
// copy-driven-query lint rule (which forbids string *literals* in TL queries)
// is satisfied while still asserting the hit title threads through to the DOM.
const HIT_TITLE = "scenario restart";

vi.mock("../../api/search", () => ({
  runQuery: vi.fn(),
  fetchFederationStatus: vi.fn(),
  listActiveProviders: vi.fn(),
}));

import { SearchPanel } from "./SearchPanel";
import { selectors } from "../../consts/selectors";
import * as searchApi from "../../api/search";

const provider = (providerId: string, type: string) =>
  create(ProviderDescriptorSchema, {
    providerId,
    providerGroup: providerId.split(".")[0],
    type,
    state: ProviderState.ACTIVE,
  });

const status = () =>
  create(StatusResponseSchema, {
    providers: [{ providerId: "cli-health.commands", reachable: true, reachability: "endpoint resolved", indexAge: "not_reported: provider has no status_endpoint" }],
    classifierAvailable: true,
    rerankerAvailable: true,
  });

const rerankedResponse = () =>
  create(QueryResponseSchema, {
    ranked: [
      {
        providerId: "cli-health.commands",
        providerGroup: "cli-health",
        type: "command",
        id: "scenario-restart",
        title: HIT_TITLE,
        snippet: "Restart a scenario",
        path: "scenario restart",
        rerankScore: 0.91,
      },
    ],
    groups: [{ providerId: "cli-health.commands", count: 1, hits: [{ id: "scenario-restart", title: HIT_TITLE }] }],
    reranked: true,
    latencyMs: 1200n,
  });

function wireDefaults() {
  vi.mocked(searchApi.listActiveProviders).mockResolvedValue([
    provider("cli-health.commands", "command"),
    provider("knowledge-observatory.docs", "doc"),
  ]);
  vi.mocked(searchApi.fetchFederationStatus).mockResolvedValue(status());
  vi.mocked(searchApi.runQuery).mockResolvedValue(rerankedResponse());
}

describe("SearchPanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the query box, submit, and expand toggle", () => {
    wireDefaults();
    renderWithProviders(<SearchPanel />);
    expect(screen.getByTestId(selectors.search.input)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.search.submit)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.search.expandToggle)).toBeInTheDocument();
  });

  it("shows the empty state before any search", () => {
    wireDefaults();
    renderWithProviders(<SearchPanel />);
    expect(screen.getByTestId(selectors.search.empty)).toBeInTheDocument();
  });

  it("builds type facets from the active registry providers", async () => {
    wireDefaults();
    renderWithProviders(<SearchPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.search.typeFacet({ type: "command" }))).toBeInTheDocument();
      expect(screen.getByTestId(selectors.search.typeFacet({ type: "doc" }))).toBeInTheDocument();
    });
  });

  it("runs a query on submit and renders the unified ranked results + provenance", async () => {
    wireDefaults();
    const user = userEvent.setup();
    renderWithProviders(<SearchPanel />);

    await user.type(screen.getByTestId(selectors.search.input), "restart a scenario");
    await user.click(screen.getByTestId(selectors.search.submit));

    await waitFor(() => {
      expect(searchApi.runQuery).toHaveBeenCalledWith(
        expect.objectContaining({ query: "restart a scenario", all: false }),
      );
    });
    const results = await screen.findByTestId(selectors.search.results);
    expect(within(results).getByText(HIT_TITLE)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.search.summary)).toBeInTheDocument();
  });

  it("expand toggle sends all=true and ignores type facets", async () => {
    wireDefaults();
    const user = userEvent.setup();
    renderWithProviders(<SearchPanel />);

    await user.type(screen.getByTestId(selectors.search.input), "anything");
    await user.click(screen.getByTestId(selectors.search.expandToggle));
    await user.click(screen.getByTestId(selectors.search.submit));

    await waitFor(() => {
      expect(searchApi.runQuery).toHaveBeenCalledWith(
        expect.objectContaining({ all: true, types: [] }),
      );
    });
  });

  it("does not search on an empty query", async () => {
    wireDefaults();
    const user = userEvent.setup();
    renderWithProviders(<SearchPanel />);
    await user.click(screen.getByTestId(selectors.search.submit));
    expect(searchApi.runQuery).not.toHaveBeenCalled();
  });

  it("surfaces an error state when the query fails", async () => {
    wireDefaults();
    vi.mocked(searchApi.runQuery).mockRejectedValueOnce(new Error("boom"));
    const user = userEvent.setup();
    renderWithProviders(<SearchPanel />);

    await user.type(screen.getByTestId(selectors.search.input), "restart");
    await user.click(screen.getByTestId(selectors.search.submit));

    expect(await screen.findByTestId(selectors.search.error)).toBeInTheDocument();
  });
});
