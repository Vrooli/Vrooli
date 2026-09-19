import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import { QueryResponseSchema } from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { SearchResults } from "./SearchResults";

const FALLBACK_ID = "fallback-id";
const MATCHING_SNIPPET = "A matching document";
const DOWN_PROVIDER_NOTE = "temporarily unavailable";
const WEAK_RESULT_TITLE = "Weak result";

describe("SearchResults", () => {
  it("renders the unified ranking with a weak, located result", () => {
    const data = create(QueryResponseSchema, {
      reranked: true,
      ranked: [{
        providerId: "docs",
        providerGroup: "knowledge-observatory",
        id: FALLBACK_ID,
        title: "",
        snippet: MATCHING_SNIPPET,
        path: "",
        locations: ["one.md", "two.md", "three.md"],
        confidence: { weak: true, regime: "hybrid" },
      }],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByTestId(selectors.search.results)).toBeInTheDocument();
    expect(screen.getByText(FALLBACK_ID)).toBeInTheDocument();
    expect(screen.getByText(MATCHING_SNIPPET)).toBeInTheDocument();
    expect(screen.getByText(strings.search.locations)).toBeInTheDocument();
  });

  it("renders degraded, empty, and all-weak provider groups honestly", () => {
    const data = create(QueryResponseSchema, {
      groups: [
        { providerId: "down-provider", degraded: true, note: DOWN_PROVIDER_NOTE },
        { providerId: "empty-provider", count: 0 },
        {
          providerId: "weak-provider",
          count: 1,
          hits: [{
            providerId: "weak-provider",
            id: "weak-id",
            title: WEAK_RESULT_TITLE,
            confidence: { weak: true },
          }],
        },
      ],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByText(DOWN_PROVIDER_NOTE)).toBeInTheDocument();
    expect(screen.getByText(WEAK_RESULT_TITLE)).toBeInTheDocument();
    expect(screen.getAllByText(new RegExp(strings.search.groupEmpty)).length).toBeGreaterThan(0);
    expect(screen.getAllByText(new RegExp(strings.search.noConfidentMatch)).length).toBeGreaterThan(0);
  });
});
