import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  ConversationHighlightSchema,
  ConversationSearchCoverageSchema,
  ConversationSearchDegradationReason,
  ConversationSearchDegradationSchema,
  ConversationSearchHitSchema,
  ConversationSearchLeg,
  ConversationRunSummarySchema,
  ConversationSourceProvenanceSchema,
  SearchConversationsResponseSchema,
} from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";
import { ConversationSearchResults, SafeHighlight, safeHighlightParts } from "./ConversationSearchResults";
import { DEFAULT_CONVERSATION_FILTERS, useConversationSearch } from "./useConversationSearch";
import { renderWithProviders } from "../../test-utils";

vi.mock("./useConversationSearch", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./useConversationSearch")>();
  return { ...actual, useConversationSearch: vi.fn(), getConversationContext: vi.fn() };
});

const hit = create(ConversationSearchHitSchema, {
  stableHitId: "hit-1",
  runId: "run-older-than-page",
  eventId: "event-7",
  eventSequence: 7n,
  role: "assistant",
  occurredAt: timestampFromDate(new Date("2026-09-01T12:00:00Z")),
  snippet: "Corrected <img src=x onerror=alert(1)> analysis",
  highlights: [create(ConversationHighlightSchema, { startGrapheme: 10, endGrapheme: 15, field: "snippet" })],
  provenance: create(ConversationSourceProvenanceSchema, { harness: "codex", sourceSessionId: "session-1" }),
  run: create(ConversationRunSummarySchema, { runId: "run-older-than-page", label: "Recovered analysis", status: "complete", runner: "codex", model: "gpt-5" }),
  deepLink: "/runs/untrusted",
});

function mockSearch(overrides: Record<string, unknown> = {}) {
  vi.mocked(useConversationSearch).mockReturnValue({
    hits: [hit],
    loading: false,
    loadingMore: false,
    error: null,
    status: null,
    response: create(SearchConversationsResponseSchema, {
      hits: [hit],
      coverage: create(ConversationSearchCoverageSchema, { lexicalRatio: 1, semanticRatio: 0.5, pendingDocuments: 2n }),
      degradations: [create(ConversationSearchDegradationSchema, {
        reason: ConversationSearchDegradationReason.VECTOR_STORE_UNAVAILABLE,
        leg: ConversationSearchLeg.DENSE,
        detail: "Semantic retrieval is offline.",
        retryable: true,
      })],
    }),
    hasMore: false,
    loadMore: vi.fn(),
    retry: vi.fn(),
    recordSelection: vi.fn(),
    ...overrides,
  });
}

afterEach(() => vi.clearAllMocks());

describe("safe conversation highlighting", () => {
  test("uses grapheme ranges and renders hostile markup as text", () => {
    expect(safeHighlightParts("A🙂BC", [create(ConversationHighlightSchema, { startGrapheme: 1, endGrapheme: 2, field: "snippet" })]))
      .toEqual([{ text: "A", highlighted: false }, { text: "🙂", highlighted: true }, { text: "BC", highlighted: false }]);
    const { container } = renderWithProviders(<SafeHighlight hit={hit} />);
    expect(container.textContent).toContain("<img src=x onerror=alert(1)>");
    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("mark")?.textContent).toBe("<img ");
  });
});

test("groups attributable results, announces degradation, and opens the typed hit", async () => {
  const recordSelection = vi.fn();
  mockSearch({ recordSelection });
  const onOpenHit = vi.fn();
  const onResultCount = vi.fn();
  const { container } = renderWithProviders(<ConversationSearchResults query="corrected analysis" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={vi.fn()} onOpenHit={onOpenHit} onResultCount={onResultCount} />);

  expect(await screen.findByRole("heading", { name: "Recovered analysis" })).toBeInTheDocument();
  expect(screen.getByText(/Partial search coverage/)).toBeInTheDocument();
  expect(screen.getByText(/Semantic ranking is temporarily unavailable/)).toBeInTheDocument();
  expect(screen.queryByText(/vector query returned status/)).not.toBeInTheDocument();
  expect(screen.getByText(/Source: codex · session session-1/)).toBeInTheDocument();
  expect(screen.getByText(/Coverage: 100% lexical · 50% semantic · 2 pending/)).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: /Open matched event/i }));
  expect(onOpenHit).toHaveBeenCalledWith(hit);
  expect(recordSelection).toHaveBeenCalledWith(hit, 1);
  expect(onResultCount).toHaveBeenCalledWith(1);
  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
});

test("renders honest invalid-query and weak/no-result states", () => {
  mockSearch({ hits: [], response: null, error: { kind: "invalid", message: "regular expression is invalid" } });
  const { rerender } = renderWithProviders(<ConversationSearchResults query="[" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={vi.fn()} onOpenHit={vi.fn()} />);
  expect(screen.getByRole("alert")).toHaveTextContent("Search query is invalid");

  mockSearch({ hits: [{ ...hit, weak: true }], response: null, error: null });
  rerender(<ConversationSearchResults query="vague" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={vi.fn()} onOpenHit={vi.fn()} />);
  expect(screen.getByText(/Only weak matches/)).toBeInTheDocument();
});

test("advanced controls remain usable at a narrow viewport", async () => {
  mockSearch();
  Object.defineProperty(window, "innerWidth", { configurable: true, value: 360 });
  const onFiltersChange = vi.fn();
  renderWithProviders(<ConversationSearchResults query="history" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={onFiltersChange} onOpenHit={vi.fn()} />);
  await userEvent.click(screen.getByRole("button", { name: /Advanced conversation filters/ }));
  expect(screen.getByRole("group", { name: "Advanced conversation filters" })).toHaveClass("grid-cols-1");
  fireEvent.change(screen.getByLabelText("Harness"), { target: { value: "codex" } });
  expect(onFiltersChange).toHaveBeenCalledWith(expect.objectContaining({ harness: "codex" }));
});
