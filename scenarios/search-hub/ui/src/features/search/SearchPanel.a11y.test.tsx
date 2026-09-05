/**
 * SearchPanel accessibility regression tests.
 *
 * The search feature owns its query box, facets, and result states, so a11y
 * coverage lives here. Both the pre-search (empty) and post-search (results)
 * states are checked under the real English locale.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
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

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/search", () => ({
  runQuery: vi.fn(),
  fetchFederationStatus: vi.fn(),
  listActiveProviders: vi.fn(),
}));

import { SearchPanel } from "./SearchPanel";
import * as searchApi from "../../api/search";

function wire() {
  vi.mocked(searchApi.listActiveProviders).mockResolvedValue([
    create(ProviderDescriptorSchema, {
      providerId: "cli-health.commands",
      providerGroup: "cli-health",
      type: "command",
      state: ProviderState.ACTIVE,
    }),
  ]);
  vi.mocked(searchApi.fetchFederationStatus).mockResolvedValue(
    create(StatusResponseSchema, {
      providers: [{ providerId: "cli-health.commands", reachable: true, reachability: "endpoint resolved", indexAge: "not_reported: provider has no status_endpoint" }],
      classifierAvailable: true,
      rerankerAvailable: true,
    }),
  );
  vi.mocked(searchApi.runQuery).mockResolvedValue(
    create(QueryResponseSchema, {
      ranked: [
        {
          providerId: "cli-health.commands",
          providerGroup: "cli-health",
          type: "command",
          id: "scenario-restart",
          title: "scenario restart",
          snippet: "Restart a scenario",
          path: "scenario restart",
          rerankScore: 0.91,
        },
      ],
      groups: [{ providerId: "cli-health.commands", count: 1, hits: [{ id: "scenario-restart", title: "scenario restart" }] }],
      reranked: true,
      latencyMs: 1200n,
    }),
  );
}

describe("SearchPanel accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    wire();
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty/idle state without axe violations", async () => {
    const { container } = renderWithProviders(<SearchPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.search.statusBar)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });

  it("renders the results state without axe violations", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<SearchPanel />);

    await user.type(screen.getByTestId(selectors.search.input), "restart a scenario");
    await user.click(screen.getByTestId(selectors.search.submit));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.search.results)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
