import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  ConversationContextEventSchema,
  GetConversationContextResponseSchema,
  ConversationHighlightSchema,
  ConversationSearchCoverageSchema,
  ConversationSearchDegradationReason,
  ConversationSearchDegradationSchema,
  ConversationSearchHitSchema,
  ConversationIndexState,
  ConversationSearchMode,
  ConversationSearchSort,
  ConversationContentClass,
  GetConversationIndexStatusResponseSchema,
  ConversationSearchLeg,
  ConversationRunSummarySchema,
  ConversationSourceProvenanceSchema,
  SearchConversationsResponseSchema,
} from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";
import { ConversationSearchResults, SafeHighlight, safeHighlightParts } from "./ConversationSearchResults";
import { DEFAULT_CONVERSATION_FILTERS, getConversationContext, useConversationSearch } from "./useConversationSearch";
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

test("nearby context stays bounded, retains attribution and can be hidden and reloaded", async () => {
  mockSearch();
  let complete!: (value: ReturnType<typeof create<typeof GetConversationContextResponseSchema>>) => void;
  const pending = new Promise<ReturnType<typeof create<typeof GetConversationContextResponseSchema>>>((resolve) => { complete = resolve; });
  vi.mocked(getConversationContext).mockReturnValueOnce(pending).mockResolvedValueOnce(create(GetConversationContextResponseSchema));
  renderWithProviders(<ConversationSearchResults query="context" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={vi.fn()} onOpenHit={vi.fn()} />);
  fireEvent.click(screen.getByRole("button", { name: "Show nearby context" }));
  expect(getConversationContext).toHaveBeenCalledWith(hit, expect.any(AbortSignal));
  expect(screen.getByRole("button", { name: "Show nearby context" }).querySelector(".animate-spin")).not.toBeNull();
  await act(async () => complete(create(GetConversationContextResponseSchema, { truncated: true, events: [
    create(ConversationContextEventSchema, { eventId: "matched", eventSequence: 7n, role: "assistant", boundedContent: "matched retained event", matched: true }),
    create(ConversationContextEventSchema, { eventId: "nearby", eventSequence: 8n, boundedContent: "bounded nearby event" }),
  ] })));
  expect(screen.getByText("assistant · sequence 7")).toBeInTheDocument();
  expect(screen.getByText("event · sequence 8")).toBeInTheDocument();
  expect(screen.getByText("matched retained event").parentElement).toHaveClass("border-primary");
  expect(screen.getByText("bounded nearby event").parentElement).toHaveClass("border-border");
  expect(screen.getByText(/Context is bounded/)).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Hide context" }));
  expect(screen.queryByText("matched retained event")).toBeNull();
  expect(getConversationContext).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByRole("button", { name: "Show nearby context" }));
  await screen.findByRole("button", { name: "Hide context" });
  expect(getConversationContext).toHaveBeenCalledTimes(2);
  expect(screen.queryByText(/Context is bounded/)).toBeNull();
});

test("context failures explain the unavailable result and unmount aborts pending work", async () => {
  mockSearch();
  vi.mocked(getConversationContext).mockRejectedValueOnce(new Error("retained context is unavailable")).mockRejectedValueOnce("opaque failure");
  const view = renderWithProviders(<ConversationSearchResults query="context" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={vi.fn()} onOpenHit={vi.fn()} />);
  fireEvent.click(screen.getByRole("button", { name: "Show nearby context" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("retained context is unavailable");
  fireEvent.click(screen.getByRole("button", { name: "Show nearby context" }));
  await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Context is unavailable."));
  let reject!: (reason: Error) => void;
  vi.mocked(getConversationContext).mockReturnValueOnce(new Promise((_resolve, fail) => { reject = fail; }));
  fireEvent.click(screen.getByRole("button", { name: "Show nearby context" }));
  const signal = vi.mocked(getConversationContext).mock.calls.at(-1)?.[1];
  view.unmount();
  expect(signal?.aborted).toBe(true);
  await act(async () => reject(new Error("cancelled")));
});

test("highlighting merges overlapping ranges, excludes foreign fields and safely falls back without Segmenter", () => {
  const range = (startGrapheme: number, endGrapheme: number, field = "snippet") => create(ConversationHighlightSchema, { startGrapheme, endGrapheme, field });
  expect(safeHighlightParts("ABCDE", [range(2, 99), range(1, 3), range(0, 1), range(3, 3), range(0, 2, "content")]))
    .toEqual([{ text: "ABCDE", highlighted: true }]);
  expect(safeHighlightParts("", [])).toEqual([{ text: "", highlighted: false }]);
  const original = Object.getOwnPropertyDescriptor(Intl, "Segmenter");
  try {
    Object.defineProperty(Intl, "Segmenter", { configurable: true, value: undefined });
    expect(safeHighlightParts("A🙂B", [range(1, 2, "")])).toEqual([
      { text: "A", highlighted: false }, { text: "🙂", highlighted: true }, { text: "B", highlighted: false },
    ]);
  } finally {
    if (original) Object.defineProperty(Intl, "Segmenter", original);
  }
});

test("advanced filters send precise changes and reset instead of retaining hidden filter state", () => {
  mockSearch();
  const change = vi.fn();
  renderWithProviders(<ConversationSearchResults query="filters" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={change} onOpenHit={vi.fn()} />);
  fireEvent.click(screen.getByRole("button", { name: /Advanced conversation filters/ }));
  for (const [label, key, value] of [["Mode", "mode", ConversationSearchMode.REGEX], ["Sort", "sort", ConversationSearchSort.NEWEST], ["Content class", "contentClass", ConversationContentClass.PROSE]] as const) {
    fireEvent.change(screen.getByLabelText(label), { target: { value: String(value) } });
    expect(change.mock.calls.at(-1)?.[0]).toEqual({ ...DEFAULT_CONVERSATION_FILTERS, [key]: value });
  }
  for (const [label, key, value] of [["Role", "role", "assistant"], ["Project", "project", "project"], ["Model", "model", "model"], ["Profile", "profile", "profile"], ["Runstatus", "runStatus", "complete"], ["After", "after", "2026-09-01"], ["Before", "before", "2026-09-02"]] as const) {
    fireEvent.change(screen.getByLabelText(label), { target: { value } });
    expect(change.mock.calls.at(-1)?.[0]).toEqual({ ...DEFAULT_CONVERSATION_FILTERS, [key]: value });
  }
  fireEvent.click(screen.getByRole("checkbox", { name: "Include tool events" }));
  expect(change.mock.calls.at(-1)?.[0].includeToolEvents).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "Reset" }));
  expect(change.mock.calls.at(-1)?.[0]).toEqual(DEFAULT_CONVERSATION_FILTERS);
  fireEvent.click(screen.getByRole("button", { name: /Advanced conversation filters/ }));
  expect(screen.queryByRole("group", { name: "Advanced conversation filters" })).toBeNull();
});

test("search status preserves honest loading, restricted, admission, retry and pagination states", () => {
  const retry = vi.fn();
  const more = vi.fn();
  mockSearch({ hits: [], loading: true, response: null, status: null });
  const view = renderWithProviders(<ConversationSearchResults query="status" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={vi.fn()} onOpenHit={vi.fn()} />);
  expect(screen.getByText("Searching all retained conversations…")).toBeInTheDocument();
  expect(screen.queryByText("No conversation matches")).toBeNull();
  const render = () => view.rerender(<ConversationSearchResults query="status" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={vi.fn()} onOpenHit={vi.fn()} />);
  for (const [kind, heading] of [["permission", "Conversation search is restricted"], ["admission", "Search is temporarily busy"], ["generic", "Conversation search failed"]]) {
    mockSearch({ hits: [], response: null, error: { kind, message: "bounded explanation" }, retry });
    render();
    expect(screen.getByRole("alert")).toHaveTextContent(heading);
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  }
  expect(retry).toHaveBeenCalledTimes(3);
  mockSearch({ hits: [], response: null, status: create(GetConversationIndexStatusResponseSchema, { state: ConversationIndexState.STALE }), hasMore: true, loadMore: more });
  render();
  expect(screen.getByText("No conversation matches")).toBeInTheDocument();
  expect(screen.getByText(/index may not include the newest messages/)).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Load more results" }));
  expect(more).toHaveBeenCalledTimes(1);
  mockSearch({ hits: [{ ...hit, run: undefined, role: "", occurredAt: undefined, provenance: undefined }], response: null, hasMore: true, loadingMore: true });
  render();
  expect(screen.getByText("unknown status · unknown runner · unknown model")).toBeInTheDocument();
  expect(screen.getByText("time unavailable")).toBeInTheDocument();
  expect(screen.getByText("Source: unknown harness")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Loading more" })).toBeDisabled();
});


test.each([
  [ConversationSearchDegradationReason.SEMANTIC_UNAVAILABLE, "Semantic ranking is temporarily unavailable."],
  [ConversationSearchDegradationReason.EMBEDDING_UNAVAILABLE, "Semantic ranking is temporarily unavailable."],
  [ConversationSearchDegradationReason.INDEX_STALE, "The newest retained messages may still be indexing."],
  [ConversationSearchDegradationReason.INDEX_LAYOUT_MISMATCH, "Semantic index maintenance is required."],
  [ConversationSearchDegradationReason.CANDIDATE_LIMIT, "The search candidate limit was reached; refine the query for more precise results."],
  [ConversationSearchDegradationReason.DEADLINE, "One search path exceeded its time budget."],
  [ConversationSearchDegradationReason.AUTHORIZATION_FILTERED, "Some results were excluded by access policy."],
  [ConversationSearchDegradationReason.RERANK_UNAVAILABLE, "Advanced result ordering is temporarily unavailable."],
  [ConversationSearchDegradationReason.UNSPECIFIED, "One search path is temporarily unavailable."],
])("explains degraded retrieval reason %s without exposing provider details", (reason, message) => {
  mockSearch({ response: create(SearchConversationsResponseSchema, {
    hits: [hit], degradations: [create(ConversationSearchDegradationSchema, { reason: Number(reason), detail: "secret provider failure" })],
  }) });
  renderWithProviders(<ConversationSearchResults query="history" filters={DEFAULT_CONVERSATION_FILTERS} onFiltersChange={vi.fn()} onOpenHit={vi.fn()} />);
  expect(screen.getByText((text) => text.includes(String(message)))).toBeInTheDocument();
  expect(screen.queryByText("secret provider failure")).not.toBeInTheDocument();
});
