import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  SearchResponseSchema,
  SearchResultSchema,
  SearchStatusSchema,
  SemanticLegState,
} from "@vrooli/proto-types/react-component-library/v1/search/search_pb";

import { SearchAvailability, SearchKind, searchWorkbench } from "../../api/search";
import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { SearchSurface } from "./SearchSurface";

vi.mock("../../api/search", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/search")>();
  return { ...actual, searchWorkbench: vi.fn() };
});

const mockedSearch = vi.mocked(searchWorkbench);

const response = create(SearchResponseSchema, {
  query: "landing page",
  total: 2,
  tookMs: 4n,
  status: create(SearchStatusSchema, {
    indexed: true,
    documentCount: 480,
    indexedAt: "2026-11-19T12:00:00Z",
    stale: false,
    lastError: "",
    semanticLeg: SemanticLegState.DISABLED,
  }),
  results: [
    create(SearchResultSchema, {
      id: "templates.landing-page",
      kind: SearchKind.PAGE_TYPE,
      title: "LandingPage",
      subtitle: "Page type · templates",
      snippet: "Hero, proof, pricing and call to action regions.",
      href: "/library?kind=page-template&q=LandingPage",
      score: 3.2,
      availability: SearchAvailability.PLANNED,
      badges: ["page type"],
      relations: [],
    }),
    create(SearchResultSchema, {
      id: "react-component-library:Button",
      kind: SearchKind.ASSET,
      title: "Button",
      subtitle: "controls · component",
      snippet: "Primary action surface.",
      href: "/assets/react-component-library%3AButton",
      score: 1.1,
      availability: SearchAvailability.AVAILABLE,
      badges: ["2.2.14"],
      relations: [],
    }),
  ],
});

beforeEach(async () => {
  await setLocale("en");
  mockedSearch.mockReset();
  mockedSearch.mockResolvedValue(response);
});

afterEach(() => cleanup());

describe("SearchSurface", () => {
  it("renders server-ranked kinds in outcome order", async () => {
    renderWithProviders(<SearchSurface initialQuery="landing page" />);
    expect(await screen.findByText(/LandingPage/)).toBeInTheDocument();
    expect(screen.getByText(/Page types/)).toBeInTheDocument();
    expect(screen.getByText(/Assets/)).toBeInTheDocument();
    const rows = screen.getAllByRole("button");
    expect(rows[0]).toHaveTextContent("LandingPage");
  });

  it("sends typed queries to the server instead of filtering a local list", async () => {
    renderWithProviders(<SearchSurface />);
    const input = screen.getByRole("textbox");
    await userEvent.type(input, "onboarding");
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "onboarding" }),
      );
    });
  });

  it("distinguishes an empty index from a no-match query", async () => {
    mockedSearch.mockResolvedValue(
      create(SearchResponseSchema, {
        query: "nothing",
        total: 0,
        results: [],
        status: create(SearchStatusSchema, {
          indexed: false,
          documentCount: 0,
          semanticLeg: SemanticLegState.DISABLED,
        }),
      }),
    );
    renderWithProviders(<SearchSurface initialQuery="nothing" />);
    expect(
      await screen.findByText(/The index is empty/i),
    ).toBeInTheDocument();
  });

  it("reports a search failure without hiding the recovery message", async () => {
    mockedSearch.mockRejectedValue(new Error("connection refused"));
    renderWithProviders(<SearchSurface initialQuery="landing" />);
    expect(await screen.findByRole("alert")).toHaveTextContent(/unavailable/i);
  });
});
