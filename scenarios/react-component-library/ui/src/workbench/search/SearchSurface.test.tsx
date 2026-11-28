import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useLocation } from "react-router-dom";
import { create } from "@bufbuild/protobuf";
import {
  SearchResponseSchema,
  SearchResultSchema,
  SearchStatusSchema,
  AskAnswerSchema,
  AskCitationSchema,
  SemanticLegState,
} from "@vrooli/proto-types/react-component-library/v1/search/search_pb";

import {
  SearchAvailability,
  SearchKind,
  searchWorkbench,
  type SearchResponse,
} from "../../api/search";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
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
  it("keeps an untouched surface quiet and requests compact results when a query is entered", async () => {
    renderWithProviders(<SearchSurface compact />);
    expect(screen.getByTestId("search-surface-results")).toHaveAttribute("hidden");
    expect(screen.getByTestId("search-surface")).toHaveAttribute("data-compact", "true");
    await new Promise((resolve) => setTimeout(resolve, 200));
    expect(mockedSearch).not.toHaveBeenCalled();

    const input = screen.getByRole("searchbox", { name: "Search the library" });
    await userEvent.type(input, "landing");
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "landing", limit: 12 }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    });
  });

  it("renders server-ranked kinds in outcome order", async () => {
    renderWithProviders(<SearchSurface initialQuery="landing page" />);
    expect(await screen.findByText(/LandingPage/)).toBeInTheDocument();
    expect(screen.getByText(/Page types/)).toBeInTheDocument();
    expect(screen.getByText(/Assets/)).toBeInTheDocument();
    const rows = screen.getAllByRole("button", { name: /LandingPage|Button/ });
    expect(rows[0]).toHaveTextContent("LandingPage");
    expect(screen.getByRole("list").querySelectorAll("li")).toHaveLength(2);
  });

  it("sends typed queries to the server instead of filtering a local list", async () => {
    renderWithProviders(<SearchSurface />);
    const input = screen.getByRole("searchbox", { name: "Search the library" });
    expect(input.closest("[data-rcl-search-input]")).toBeInTheDocument();
    await userEvent.type(input, "onboarding");
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "onboarding" }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    });
  });

  it("uses the shared RCL tabs to filter Start results by server-owned result kind", async () => {
    const onClearKinds = vi.fn();
    renderWithProviders(<SearchSurface compact onClearKinds={onClearKinds} />);

    await userEvent.click(screen.getByRole("tab", { name: "Assets" }));
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "", kinds: [SearchKind.ASSET], limit: 12 }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    });

    await userEvent.click(screen.getByRole("tab", { name: "Pages" }));
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({
          query: "",
          kinds: [SearchKind.PAGE, SearchKind.PAGE_TYPE],
          limit: 12,
        }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    });

    expect(onClearKinds).toHaveBeenCalledTimes(2);
    expect(screen.getByRole("tab", { name: "Pages" })).toHaveAttribute("aria-selected", "true");
  });

  it("keeps keyboard selection on the server-ranked result list", async () => {
    renderWithProviders(<SearchSurface initialQuery="landing page" />);
    const input = screen.getByRole("searchbox", { name: "Search the library" });
    await screen.findByText(/LandingPage/);

    await userEvent.type(input, "{ArrowDown}");

    expect(input).toHaveAttribute("aria-activedescendant", "search-result-1");
    expect(document.getElementById("search-result-1")).toHaveTextContent("Button");
  });

  it("keeps prior RCL results and keyboard targets while refreshing and after an error", async () => {
    let rejectRefresh: (reason: Error) => void = () => {};
    const refresh = new Promise<SearchResponse>((_, reject) => {
      rejectRefresh = reject;
    });
    mockedSearch.mockResolvedValueOnce(response).mockReturnValueOnce(refresh);
    renderWithProviders(<SearchSurface initialQuery="landing page" />);
    const input = screen.getByRole("searchbox", { name: "Search the library" });
    expect(await screen.findByText(/LandingPage/)).toBeInTheDocument();

    await userEvent.clear(input);
    await userEvent.type(input, "new query");
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "new query" }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    });
    await userEvent.keyboard("{ArrowDown}");

    expect(await screen.findByText(i18n.t(strings.workbench.search.loading))).toBeInTheDocument();
    expect(input).toHaveAttribute("aria-activedescendant", "search-result-1");
    expect(screen.getByRole("button", { name: /Button/ })).toBeInTheDocument();

    rejectRefresh(new Error("connection refused"));
    expect(await screen.findByRole("alert")).toHaveTextContent(/unavailable/i);
    expect(screen.getByRole("button", { name: /Button/ })).toBeInTheDocument();
    expect(input).toHaveAttribute("aria-activedescendant", "search-result-1");
  });

  it("opens a server-backed command-only browse state without a query", async () => {
    mockedSearch.mockResolvedValue(
      create(SearchResponseSchema, {
        ...response,
        query: "",
        total: 1,
        results: [
          create(SearchResultSchema, {
            id: "command.new-workspace",
            kind: SearchKind.COMMAND,
            title: "New workspace",
            subtitle: "Create · N",
            snippet: "Start a new workspace in Design.",
            href: "/design?new=1",
            badges: ["N"],
          }),
        ],
      }),
    );

    renderWithProviders(<SearchSurface initialKinds={[SearchKind.COMMAND]} />);

    const commandResults = await screen.findByTestId("search-surface-results");
    expect(commandResults).not.toHaveAttribute("hidden");
    await waitFor(() => expect(commandResults).toHaveTextContent("New workspace"));
    expect(mockedSearch).toHaveBeenCalledWith(
      expect.objectContaining({ query: "", kinds: [SearchKind.COMMAND] }),
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(screen.getByTestId("search-clear-kind-filter")).toBeInTheDocument();
  });

  it("clears an initial kind filter back to a quiet browse surface", async () => {
    const onClearKinds = vi.fn();
    renderWithProviders(
      <SearchSurface initialKinds={[SearchKind.COMMAND]} onClearKinds={onClearKinds} />,
    );
    await screen.findByText(/LandingPage/);

    await userEvent.click(screen.getByRole("button", { name: "All results" }));

    expect(onClearKinds).toHaveBeenCalledOnce();
    await new Promise((resolve) => setTimeout(resolve, 200));
    expect(mockedSearch).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("search-surface-results")).toHaveAttribute("hidden");
    expect(screen.queryByTestId("search-clear-kind-filter")).not.toBeInTheDocument();
  });

  it("submits changed Ask text with Enter and opens a selected result", async () => {
    function LocationOutput() {
      const location = useLocation();
      return <output data-testid="current-location">{location.pathname + location.search}</output>;
    }
    const onNavigate = vi.fn();
    renderWithProviders(
      <>
        <SearchSurface initialQuery="landing page" onNavigate={onNavigate} />
        <LocationOutput />
      </>,
    );
    const input = screen.getByRole("searchbox", { name: "Search the library" });
    await screen.findByText(/LandingPage/);
    await userEvent.click(screen.getByRole("button", { name: "Ask" }));
    await userEvent.clear(input);
    await userEvent.type(input, "how do landing pages work?");

    await new Promise((resolve) => globalThis.setTimeout(resolve, 180));
    expect(mockedSearch).toHaveBeenCalledTimes(1);
    await userEvent.keyboard("{Enter}");
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "how do landing pages work?", ask: true }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    });

    await userEvent.click(screen.getByRole("button", { name: "Search" }));
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "how do landing pages work?", ask: false }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    });
    await userEvent.click(await screen.findByRole("button", { name: /LandingPage/ }));
    expect(onNavigate).toHaveBeenCalledOnce();
    expect(screen.getByTestId("current-location")).toHaveTextContent(
      "/library?kind=page-template&q=LandingPage",
    );
  });

  it("uses explicit Ask mode and renders only linked indexed citations", async () => {
    mockedSearch.mockResolvedValue(
      create(SearchResponseSchema, {
        ...response,
        answer: create(AskAnswerSchema, {
          text: "Landing pages combine a clear value proposition with a call to action.",
          citations: [create(AskCitationSchema, { resultId: "templates.landing-page" })],
        }),
      }),
    );
    renderWithProviders(<SearchSurface initialQuery="landing page" />);
    await userEvent.click(screen.getByRole("button", { name: "Ask" }));
    await userEvent.click(screen.getByRole("button", { name: "Get answer" }));
    await waitFor(() => {
      expect(mockedSearch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "landing page", ask: true }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    });
    expect(
      await screen.findByText(/Landing pages combine a clear value proposition/),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Source 1: LandingPage/ })).toBeInTheDocument();
  });

  it("reports an Ask failure without presenting lexical matches as an answer", async () => {
    mockedSearch.mockRejectedValue(new Error("gateway unavailable"));
    const { container } = renderWithProviders(<SearchSurface initialQuery="landing page" />);
    await userEvent.click(screen.getByRole("button", { name: "Ask" }));
    await userEvent.click(screen.getByRole("button", { name: "Get answer" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/answer is unavailable/i);
    expect(screen.queryByLabelText(/Answer from indexed results/i)).not.toBeInTheDocument();
    expect(container.querySelector(".wb-command-group")).toBeNull();
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
    expect(await screen.findByText(/The index is empty/i)).toBeInTheDocument();
  });

  it("keeps the workbench no-match state distinct from SearchResults empty state", async () => {
    mockedSearch.mockResolvedValue(
      create(SearchResponseSchema, {
        query: "nothing",
        total: 0,
        results: [],
        status: create(SearchStatusSchema, {
          indexed: true,
          documentCount: 480,
          semanticLeg: SemanticLegState.DISABLED,
        }),
      }),
    );
    renderWithProviders(<SearchSurface initialQuery="nothing" />);

    expect(await screen.findByText(i18n.t(strings.workbench.search.noMatch))).toBeInTheDocument();
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
  });

  it("explains that keyword results are shown when semantic search is degraded", async () => {
    mockedSearch.mockResolvedValue(
      create(SearchResponseSchema, {
        query: "button",
        total: 1,
        results: response.results.slice(1, 2),
        status: create(SearchStatusSchema, {
          indexed: true,
          documentCount: 480,
          semanticLeg: SemanticLegState.DEGRADED,
        }),
      }),
    );
    renderWithProviders(<SearchSurface initialQuery="button" />);
    expect(await screen.findByRole("status")).toHaveTextContent(/keyword matches/i);
    const buttonTitle = response.results.find((result) => result.id.endsWith(":Button"))?.title;
    expect(buttonTitle).toBeDefined();
    expect(screen.getByText(buttonTitle ?? "")).toBeInTheDocument();
  });

  it("reports a search failure without hiding the recovery message", async () => {
    mockedSearch.mockRejectedValue(new Error("connection refused"));
    renderWithProviders(<SearchSurface initialQuery="landing" />);
    expect(await screen.findByRole("alert")).toHaveTextContent(/unavailable/i);
  });
});
