import { Code, ConnectError } from "@connectrpc/connect";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type {
  ConversationSearchHit,
  GetConversationContextResponse,
  GetConversationIndexStatusResponse,
  SearchConversationsResponse,
} from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";
import {
  ConversationContentClass,
  ConversationSearchInteractionKind,
  ConversationSearchMode,
  ConversationSearchSort,
} from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";
import { conversationSearchClient } from "./api/conversationSearchClient";

export interface ConversationSearchFiltersState {
  mode: ConversationSearchMode;
  sort: ConversationSearchSort;
  role: string;
  harness: string;
  project: string;
  model: string;
  profile: string;
  runStatus: string;
  contentClass: ConversationContentClass;
  after: string;
  before: string;
  includeToolEvents: boolean;
}

export const DEFAULT_CONVERSATION_FILTERS: ConversationSearchFiltersState = {
  mode: ConversationSearchMode.HYBRID,
  sort: ConversationSearchSort.RELEVANCE,
  role: "",
  harness: "",
  project: "",
  model: "",
  profile: "",
  runStatus: "",
  contentClass: ConversationContentClass.UNSPECIFIED,
  after: "",
  before: "",
  includeToolEvents: false,
};

export type ConversationSearchErrorKind =
  | "invalid"
  | "permission"
  | "admission"
  | "generic";

export interface ConversationSearchError {
  kind: ConversationSearchErrorKind;
  message: string;
}

export function classifyConversationSearchError(error: unknown): ConversationSearchError {
  const connectError = ConnectError.from(error);
  if (connectError.code === Code.InvalidArgument) return { kind: "invalid", message: connectError.rawMessage || "Check the query and filters." };
  if (connectError.code === Code.PermissionDenied || connectError.code === Code.Unauthenticated) return { kind: "permission", message: "You do not have access to search this conversation history." };
  if (connectError.code === Code.ResourceExhausted || connectError.code === Code.Unavailable) return { kind: "admission", message: "Search is busy or temporarily unavailable. Try again shortly." };
  return { kind: "generic", message: connectError.rawMessage || "Conversation search failed." };
}

function parseDate(value: string) {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? undefined : timestampFromDate(date);
}

function splitValue(value: string): string[] {
  return value.trim() ? [value.trim()] : [];
}

export function useConversationSearch(
  query: string,
  filters: ConversationSearchFiltersState,
  pageSize = 20,
) {
  const [pages, setPages] = useState<SearchConversationsResponse[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<ConversationSearchError | null>(null);
  const [status, setStatus] = useState<GetConversationIndexStatusResponse | null>(null);
  const requestRef = useRef(0);
  const abortRef = useRef<AbortController | null>(null);
  const telemetrySessionRef = useRef(globalThis.crypto?.randomUUID?.() ?? `search-${Date.now()}-${Math.random().toString(36).slice(2)}`);
  const lastCompletedRef = useRef<{ query: string; requestId: string } | null>(null);

  const normalizedQuery = query.trim();
  const search = useCallback(async (cursor = "", append = false) => {
    if (!normalizedQuery) {
      abortRef.current?.abort();
      setPages([]);
      setError(null);
      setLoading(false);
      setLoadingMore(false);
      return;
    }

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    const requestId = ++requestRef.current;
    if (append) setLoadingMore(true);
    else setLoading(true);
    setError(null);

    const previous = lastCompletedRef.current;
    if (!append && previous?.requestId && previous.query !== normalizedQuery) {
      void conversationSearchClient.recordConversationSearchInteraction({
        requestId: previous.requestId,
        telemetrySessionToken: telemetrySessionRef.current,
        kind: ConversationSearchInteractionKind.REFORMULATED,
      }).catch(() => undefined);
      lastCompletedRef.current = null;
    }

    try {
      const response = await conversationSearchClient.searchConversations({
        query: normalizedQuery,
        mode: filters.mode,
        sort: filters.sort,
        pageSize,
        pageCursor: cursor,
        telemetrySessionToken: telemetrySessionRef.current,
        filters: {
          roles: splitValue(filters.role),
          harnesses: splitValue(filters.harness),
          providerOrigins: [],
          projectScopes: splitValue(filters.project),
          cwdScopes: [],
          runners: [],
          models: splitValue(filters.model),
          profiles: splitValue(filters.profile),
          runStatuses: splitValue(filters.runStatus),
          tags: [],
          workloads: [],
          occurredAfter: parseDate(filters.after),
          occurredBefore: parseDate(filters.before),
          contentClasses: filters.contentClass === ConversationContentClass.UNSPECIFIED ? [] : [filters.contentClass],
          includeToolEvents: filters.includeToolEvents,
        },
      }, { signal: controller.signal });
      if (requestRef.current !== requestId) return;
      setPages((current) => append ? [...current, response] : [response]);
      if (!append && response.requestId) lastCompletedRef.current = { query: normalizedQuery, requestId: response.requestId };
    } catch (caught) {
      if (controller.signal.aborted || requestRef.current !== requestId) return;
      setError(classifyConversationSearchError(caught));
      if (!append) setPages([]);
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false);
        setLoadingMore(false);
      }
    }
  }, [filters, normalizedQuery, pageSize]);

  useEffect(() => {
    const timer = window.setTimeout(() => void search(), 250);
    return () => {
      window.clearTimeout(timer);
      abortRef.current?.abort();
    };
  }, [search]);

  useEffect(() => {
    if (!normalizedQuery) return;
    const controller = new AbortController();
    void conversationSearchClient.getConversationIndexStatus({}, { signal: controller.signal })
      .then(setStatus)
      .catch(() => setStatus(null));
    return () => controller.abort();
  }, [normalizedQuery]);

  const hits = useMemo(() => pages.flatMap((page) => page.hits), [pages]);
  const requestByHit = useMemo(() => {
    const requests = new Map<string, string>();
    for (const page of pages) {
      if (!page.requestId) continue;
      for (const hit of page.hits) requests.set(hit.stableHitId, page.requestId);
    }
    return requests;
  }, [pages]);
  const lastPage = pages.at(-1) ?? null;
  const loadMore = useCallback(() => {
    if (lastPage?.nextPageCursor && !loadingMore) void search(lastPage.nextPageCursor, true);
  }, [lastPage?.nextPageCursor, loadingMore, search]);
  const retry = useCallback(() => void search(), [search]);
  const recordSelection = useCallback((hit: ConversationSearchHit, rank: number) => {
    const requestId = requestByHit.get(hit.stableHitId);
    if (!requestId || rank < 1) return;
    void conversationSearchClient.recordConversationSearchInteraction({
      requestId,
      telemetrySessionToken: telemetrySessionRef.current,
      kind: ConversationSearchInteractionKind.SELECTED,
      stableHitId: hit.stableHitId,
      selectedRank: rank,
    }).catch(() => undefined);
  }, [requestByHit]);

  return {
    hits,
    loading,
    loadingMore,
    error,
    status,
    response: lastPage,
    hasMore: Boolean(lastPage?.nextPageCursor),
    loadMore,
    retry,
    recordSelection,
  };
}

export async function getConversationContext(
  hit: ConversationSearchHit,
  signal?: AbortSignal,
): Promise<GetConversationContextResponse> {
  return conversationSearchClient.getConversationContext({
    stableHitId: hit.stableHitId,
    beforeEvents: 2,
    afterEvents: 3,
  }, { signal });
}
