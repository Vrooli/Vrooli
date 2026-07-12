import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import { QueryResponseSchema } from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { SearchResults } from "./SearchResults";

describe("SearchResults", () => {
  it("renders the unified ranking with a weak, located result", () => {
    const data = create(QueryResponseSchema, {
      reranked: true,
      ranked: [{
        providerId: "docs",
        providerGroup: "knowledge-observatory",
        id: "fallback-id",
        title: "",
        snippet: "A matching document",
        path: "",
        locations: ["one.md", "two.md", "three.md"],
        confidence: { weak: true, regime: "hybrid" },
      }],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByTestId(selectors.search.results)).toBeInTheDocument();
    expect(screen.getByText("fallback-id")).toBeInTheDocument();
    expect(screen.getByText("A matching document")).toBeInTheDocument();
    expect(screen.getByText("search.locations")).toBeInTheDocument();
  });

  it("renders degraded, empty, and all-weak provider groups honestly", () => {
    const data = create(QueryResponseSchema, {
      groups: [
        { providerId: "down-provider", degraded: true, note: "temporarily unavailable" },
        { providerId: "empty-provider", count: 0 },
        {
          providerId: "weak-provider",
          count: 1,
          hits: [{
            providerId: "weak-provider",
            id: "weak-id",
            title: "Weak result",
            confidence: { weak: true },
          }],
        },
      ],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByText("temporarily unavailable")).toBeInTheDocument();
    expect(screen.getByText("Weak result")).toBeInTheDocument();
    expect(screen.getAllByText(/search\.groupEmpty/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/search\.noConfidentMatch/).length).toBeGreaterThan(0);
  });
});
