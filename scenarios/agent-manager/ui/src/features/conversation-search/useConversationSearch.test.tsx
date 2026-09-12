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
