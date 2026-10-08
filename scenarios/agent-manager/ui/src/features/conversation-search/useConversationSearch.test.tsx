import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { ConversationSearchHitSchema, SearchConversationsResponseSchema } from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";
import { conversationSearchClient } from "./api/conversationSearchClient";
import {
  classifyConversationSearchError,
  DEFAULT_CONVERSATION_FILTERS,
  useConversationSearch,
} from "./useConversationSearch";

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

test("classifies actionable Connect failures without exposing transport details", () => {
  expect(classifyConversationSearchError(new ConnectError("bad regex", Code.InvalidArgument))).toEqual({ kind: "invalid", message: "bad regex" });
  expect(classifyConversationSearchError(new ConnectError("secret", Code.PermissionDenied))).toEqual({ kind: "permission", message: "You do not have access to search this conversation history." });
  expect(classifyConversationSearchError(new ConnectError("overloaded", Code.ResourceExhausted))).toEqual({ kind: "admission", message: "Search is busy or temporarily unavailable. Try again shortly." });
});

test("cancels stale server requests when the query changes", async () => {
  const signals: AbortSignal[] = [];
  vi.spyOn(conversationSearchClient, "getConversationIndexStatus").mockResolvedValue({} as never);
  vi.spyOn(conversationSearchClient, "searchConversations").mockImplementation((_request, options) => {
    if (options?.signal) signals.push(options.signal);
    return new Promise(() => undefined);
  });

  const { rerender, unmount } = renderHook(({ query }) => useConversationSearch(query, DEFAULT_CONVERSATION_FILTERS), { initialProps: { query: "first clue" } });
  await waitFor(() => expect(signals).toHaveLength(1));
  rerender({ query: "second clue" });
  await waitFor(() => expect(signals).toHaveLength(2));
  expect(signals[0]?.aborted).toBe(true);
  unmount();
  expect(signals[1]?.aborted).toBe(true);
});

test("appends stable cursor pages without replacing earlier hits", async () => {
  vi.spyOn(conversationSearchClient, "getConversationIndexStatus").mockResolvedValue({} as never);
  const firstHit = create(ConversationSearchHitSchema, { stableHitId: "first-page-hit" });
  const secondHit = create(ConversationSearchHitSchema, { stableHitId: "second-page-hit" });
  const search = vi.spyOn(conversationSearchClient, "searchConversations")
    .mockResolvedValueOnce(create(SearchConversationsResponseSchema, { requestId: "first-page", nextPageCursor: "next", hits: [firstHit] }))
    .mockResolvedValueOnce(create(SearchConversationsResponseSchema, { requestId: "second-page", hits: [secondHit] }));
  const interaction = vi.spyOn(conversationSearchClient, "recordConversationSearchInteraction").mockResolvedValue({ accepted: true } as never);
  const { result } = renderHook(() => useConversationSearch("remembered language", DEFAULT_CONVERSATION_FILTERS));
  await waitFor(() => expect(result.current.hasMore).toBe(true));
  act(() => result.current.loadMore());
  await waitFor(() => expect(search).toHaveBeenCalledTimes(2));
  expect(search.mock.calls[1]?.[0].pageCursor).toBe("next");
  act(() => result.current.recordSelection(firstHit, 1));
  expect(interaction).toHaveBeenCalledWith(expect.objectContaining({ requestId: "first-page", stableHitId: "first-page-hit" }));
});

test("emits content-free reformulation and selection telemetry", async () => {
  vi.spyOn(conversationSearchClient, "getConversationIndexStatus").mockResolvedValue({} as never);
  const hit = create(ConversationSearchHitSchema, { stableHitId: "stable-hit-one" });
  vi.spyOn(conversationSearchClient, "searchConversations")
    .mockResolvedValueOnce(create(SearchConversationsResponseSchema, { requestId: "request-one", hits: [hit] }))
    .mockResolvedValueOnce(create(SearchConversationsResponseSchema, { requestId: "request-two" }));
  const interaction = vi.spyOn(conversationSearchClient, "recordConversationSearchInteraction").mockResolvedValue({ accepted: true } as never);
  const { result, rerender } = renderHook(({ query }) => useConversationSearch(query, DEFAULT_CONVERSATION_FILTERS), { initialProps: { query: "first clue" } });
  await waitFor(() => expect(result.current.response?.requestId).toBe("request-one"));
  act(() => result.current.recordSelection(hit, 1));
  expect(interaction).toHaveBeenCalledWith(expect.objectContaining({ requestId: "request-one", stableHitId: "stable-hit-one", selectedRank: 1 }));
  rerender({ query: "second clue" });
  await waitFor(() => expect(result.current.response?.requestId).toBe("request-two"));
  expect(interaction).toHaveBeenCalledWith(expect.objectContaining({ requestId: "request-one", kind: expect.any(Number) }));
  for (const [payload] of interaction.mock.calls) {
    expect(payload).not.toHaveProperty("query");
    expect(payload).not.toHaveProperty("snippet");
  }
});

test("late responses cannot replace a newer query and blank input clears retained results", async () => {
  vi.spyOn(conversationSearchClient, "getConversationIndexStatus").mockRejectedValue(new Error("index offline"));
  const completions: Array<(value: ReturnType<typeof create<typeof SearchConversationsResponseSchema>>) => void> = [];
  const search = vi.spyOn(conversationSearchClient, "searchConversations").mockImplementation(() => new Promise((resolve) => completions.push(resolve)));
  const { result, rerender } = renderHook(({ query }) => useConversationSearch(query, DEFAULT_CONVERSATION_FILTERS), { initialProps: { query: "old" } });
  await waitFor(() => expect(search).toHaveBeenCalledTimes(1));
  rerender({ query: "new" });
  await waitFor(() => expect(search).toHaveBeenCalledTimes(2));
  await act(async () => completions[1]?.(create(SearchConversationsResponseSchema, { requestId: "new", hits: [create(ConversationSearchHitSchema, { stableHitId: "new-hit" })] })));
  await act(async () => completions[0]?.(create(SearchConversationsResponseSchema, { requestId: "old", hits: [create(ConversationSearchHitSchema, { stableHitId: "old-hit" })] })));
  expect(result.current.hits.map((hit) => hit.stableHitId)).toEqual(["new-hit"]);
  expect(result.current.loading).toBe(false);
  expect(result.current.status).toBeNull();
  rerender({ query: "  " });
  await waitFor(() => expect(result.current.hits).toEqual([]));
  expect(result.current.error).toBeNull();
  expect(search).toHaveBeenCalledTimes(2);
});

test("cursor failure preserves prior hits, guards double loads and retry clears an initial failure", async () => {
  vi.spyOn(conversationSearchClient, "getConversationIndexStatus").mockResolvedValue({} as never);
  const hit = create(ConversationSearchHitSchema, { stableHitId: "retained" });
  let failPage!: (error: Error) => void;
  const search = vi.spyOn(conversationSearchClient, "searchConversations")
    .mockResolvedValueOnce(create(SearchConversationsResponseSchema, { hits: [hit], nextPageCursor: "next" }))
    .mockImplementationOnce(() => new Promise((_resolve, reject) => { failPage = reject; }))
    .mockRejectedValueOnce(new ConnectError("retryable", Code.Unavailable))
    .mockResolvedValueOnce(create(SearchConversationsResponseSchema, { hits: [hit] }));
  const interaction = vi.spyOn(conversationSearchClient, "recordConversationSearchInteraction").mockResolvedValue({ accepted: true } as never);
  const { result } = renderHook(() => useConversationSearch("pages", DEFAULT_CONVERSATION_FILTERS));
  await waitFor(() => expect(result.current.hasMore).toBe(true));
  act(() => result.current.loadMore());
  await waitFor(() => expect(result.current.loadingMore).toBe(true));
  act(() => result.current.loadMore());
  expect(search).toHaveBeenCalledTimes(2);
  await act(async () => failPage(new ConnectError("denied", Code.PermissionDenied)));
  expect(result.current.hits).toEqual([hit]);
  expect(result.current.error?.kind).toBe("permission");
  expect(result.current.loadingMore).toBe(false);
  act(() => result.current.recordSelection(hit, 1));
  expect(interaction).not.toHaveBeenCalled(); // requestless response is not attributable telemetry
  act(() => result.current.retry());
  await waitFor(() => expect(result.current.error?.kind).toBe("admission"));
  expect(result.current.hits).toEqual([]);
  act(() => result.current.retry());
  await waitFor(() => expect(result.current.hits).toEqual([hit]));
  expect(result.current.error).toBeNull();
  act(() => result.current.loadMore());
  expect(search).toHaveBeenCalledTimes(4);
});

test("filtered requests trim scopes, reject invalid dates and keep selection telemetry content-free", async () => {
  vi.spyOn(conversationSearchClient, "getConversationIndexStatus").mockResolvedValue({} as never);
  const hit = create(ConversationSearchHitSchema, { stableHitId: "filtered" });
  const search = vi.spyOn(conversationSearchClient, "searchConversations").mockResolvedValue(create(SearchConversationsResponseSchema, { requestId: "filtered-request", hits: [hit] }));
  const interaction = vi.spyOn(conversationSearchClient, "recordConversationSearchInteraction").mockRejectedValue(new Error("telemetry unavailable"));
  const filters = { ...DEFAULT_CONVERSATION_FILTERS, role: " assistant ", harness: " codex ", project: " repo ", model: " model ", profile: " profile ", runStatus: " complete ", after: "not-a-date", before: "2026-09-01", contentClass: 1, includeToolEvents: true };
  const { result } = renderHook(() => useConversationSearch("  bounded clue  ", filters, 7));
  await waitFor(() => expect(result.current.response?.requestId).toBe("filtered-request"));
  const request = search.mock.calls[0]?.[0];
  expect(request?.query).toBe("bounded clue");
  expect(request?.pageSize).toBe(7);
  expect(request?.filters).toMatchObject({ roles: ["assistant"], harnesses: ["codex"], projectScopes: ["repo"], models: ["model"], profiles: ["profile"], runStatuses: ["complete"], occurredAfter: undefined, contentClasses: [1], includeToolEvents: true });
  expect(request?.filters?.occurredBefore).toBeDefined();
  act(() => result.current.recordSelection(hit, 0));
  expect(interaction).not.toHaveBeenCalled();
  await act(async () => result.current.recordSelection(hit, 1));
  expect(interaction.mock.calls[0]?.[0]).toMatchObject({ requestId: "filtered-request", stableHitId: "filtered", selectedRank: 1 });
  expect(interaction.mock.calls[0]?.[0]).not.toHaveProperty("query");
  expect(result.current.error).toBeNull();
});

test("aborted request errors stay invisible and authentication/availability messages stay bounded", async () => {
  vi.spyOn(conversationSearchClient, "getConversationIndexStatus").mockResolvedValue({} as never);
  let fail!: (error: Error) => void;
  vi.spyOn(conversationSearchClient, "searchConversations").mockImplementationOnce(() => new Promise((_resolve, reject) => { fail = reject; })).mockResolvedValue(create(SearchConversationsResponseSchema));
  const { result, rerender } = renderHook(({ query }) => useConversationSearch(query, DEFAULT_CONVERSATION_FILTERS), { initialProps: { query: "first" } });
  await waitFor(() => expect(fail).toBeDefined());
  rerender({ query: "replacement" });
  await act(async () => fail(new ConnectError("cancelled transport detail", Code.Unknown)));
  expect(result.current.error).toBeNull();
  expect(classifyConversationSearchError(new ConnectError("private", Code.Unauthenticated)).kind).toBe("permission");
  expect(classifyConversationSearchError(new ConnectError("private", Code.Unavailable)).kind).toBe("admission");
  expect(classifyConversationSearchError(new ConnectError("", Code.InvalidArgument))).toEqual({ kind: "invalid", message: "Check the query and filters." });
  expect(classifyConversationSearchError(new ConnectError("", Code.Unknown))).toEqual({ kind: "generic", message: "Conversation search failed." });
  expect(classifyConversationSearchError(new Error("retained explanation"))).toEqual({ kind: "generic", message: "retained explanation" });
});
