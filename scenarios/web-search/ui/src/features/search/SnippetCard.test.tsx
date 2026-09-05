import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { SearchResultSchema } from "@vrooli/proto-types/web-search/v1/livesearch/livesearch_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { SnippetCard } from "./SnippetCard";

const fullResult = create(SearchResultSchema, {
  url: "https://example.com/page",
  title: "Example page",
  snippet: "An example snippet.",
  engine: "duckduckgo",
  score: 0.912,
  category: "general",
});

describe("SnippetCard", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders title, URL, snippet, score and source engine from a complete result", () => {
    renderWithProviders(
      <ul>
        <SnippetCard result={fullResult} />
      </ul>,
    );

    const card = screen.getByTestId(selectors.search.result);
    const link = screen.getByRole("link", { name: strings.search.openResult });
    expect(link).toHaveAttribute("href", "https://example.com/page");
    expect(link).toHaveTextContent("Example page");
    expect(card).toHaveTextContent("An example snippet.");
    // cimode renders the raw registry key for parameterised strings, so the
    // engine + score provenance line is asserted via its key paths.
    expect(card).toHaveTextContent(strings.search.resultEngine);
    expect(card).toHaveTextContent(strings.search.resultScore);
  });

  it("renders without crashing when the snippet is absent", () => {
    const noSnippet = create(SearchResultSchema, {
      url: "https://example.com/page",
      title: "Example page",
      engine: "duckduckgo",
      score: 0.5,
    });

    renderWithProviders(
      <ul>
        <SnippetCard result={noSnippet} />
      </ul>,
    );

    const card = screen.getByTestId(selectors.search.result);
    expect(card).toBeInTheDocument();
    expect(card).toHaveTextContent("Example page");
    expect(card).not.toHaveTextContent("An example snippet.");
  });

  it("falls back to the URL as the link text when the title is empty", () => {
    const untitled = create(SearchResultSchema, {
      url: "https://example.com/untitled",
      score: 0.2,
    });

    renderWithProviders(
      <ul>
        <SnippetCard result={untitled} />
      </ul>,
    );

    const link = screen.getByRole("link", { name: strings.search.openResult });
    expect(link).toHaveTextContent("https://example.com/untitled");
  });
});
