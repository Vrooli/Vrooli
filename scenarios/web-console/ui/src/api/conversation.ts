import { createClient } from "@connectrpc/connect";
import { ConversationService } from "@vrooli/proto-types/web-console/v1/conversation/conversation_pb";

import { transport } from "./client";

// conversationClient is the Connect-Web client for ConversationService.
// Consumers should prefer the typed wrappers below, which decode proto
// bigint sequences to JS numbers and surface the snake_case shapes that
// the conversation store and MessagesPane expect.
export const conversationClient = createClient(ConversationService, transport);

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

export interface ConversationEvent {
  id: string;
  sessionId: string;
  source: string;
  role: "assistant" | "user";
  text: string;
  speechParagraphs: string[];
  originalSpeechParagraphs?: string[];
  summarized: boolean;
  createdAt: string;
  sequence: number;
  deliveryState: string;
  ttsState: string;
  consumptionState: string;
}

export interface ConversationCursor {
  lastSeenSequence: number;
  lastListenedSequence: number;
}

export interface ConversationSessionResponse {
  sessionId: string;
  events: ConversationEvent[];
  cursor: ConversationCursor;
  hasMore?: boolean;
  oldestSequence?: number;
  newestSequence?: number;
  totalCount?: number;
}

// ---------------------------------------------------------------------------
// Decoders — proto wire shape → domain shape
// ---------------------------------------------------------------------------

interface ProtoConversationEvent {
  id: string;
  sessionId: string;
  source: string;
  role: string;
  text: string;
  speechParagraphs: string[];
  originalSpeechParagraphs: string[];
  summarized: boolean;
  createdAt: string;
  sequence: bigint;
  deliveryState: string;
  ttsState: string;
  consumptionState: string;
}

interface ProtoConversationCursor {
  lastSeenSequence: bigint;
  lastListenedSequence: bigint;
}

function decodeConversationEvent(e: ProtoConversationEvent): ConversationEvent {
  return {
    id: e.id,
    sessionId: e.sessionId,
    source: e.source,
    role: e.role as ConversationEvent["role"],
    text: e.text,
    speechParagraphs: e.speechParagraphs,
    originalSpeechParagraphs: e.originalSpeechParagraphs.length > 0 ? e.originalSpeechParagraphs : undefined,
    summarized: e.summarized,
    createdAt: e.createdAt,
    sequence: Number(e.sequence),
    deliveryState: e.deliveryState,
    ttsState: e.ttsState,
    consumptionState: e.consumptionState,
  };
}

function decodeConversationCursor(c: ProtoConversationCursor | undefined): ConversationCursor {
  return {
    lastSeenSequence: c ? Number(c.lastSeenSequence) : 0,
    lastListenedSequence: c ? Number(c.lastListenedSequence) : 0,
  };
}

// ---------------------------------------------------------------------------
// Typed wrappers
// ---------------------------------------------------------------------------

export async function getConversationSession(
  sessionId: string,
  opts?: { sinceSequence?: number; limit?: number; beforeSequence?: number },
): Promise<ConversationSessionResponse> {
  const resp = await conversationClient.get({
    sessionId,
    sinceSequence:
      opts?.sinceSequence && opts.sinceSequence > 0 ? BigInt(opts.sinceSequence) : 0n,
    limit: opts?.limit ?? 0,
    beforeSequence: opts?.beforeSequence && opts.beforeSequence > 0 ? BigInt(opts.beforeSequence) : 0n,
  });
  return {
    sessionId: resp.sessionId,
    events: resp.events.map(decodeConversationEvent),
    cursor: decodeConversationCursor(resp.cursor),
    hasMore: resp.hasMore,
    oldestSequence: Number(resp.oldestSequence),
    newestSequence: Number(resp.newestSequence),
    totalCount: Number(resp.totalCount),
  };
}

export async function updateConversationCursor(
  sessionId: string,
  patch: Partial<ConversationCursor>,
): Promise<ConversationCursor> {
  const req: {
    sessionId: string;
    lastSeenSequence?: bigint;
    hasLastSeenSequence?: boolean;
    lastListenedSequence?: bigint;
    hasLastListenedSequence?: boolean;
  } = { sessionId };
  if (patch.lastSeenSequence !== undefined) {
    req.lastSeenSequence = BigInt(patch.lastSeenSequence);
    req.hasLastSeenSequence = true;
  }
  if (patch.lastListenedSequence !== undefined) {
    req.lastListenedSequence = BigInt(patch.lastListenedSequence);
    req.hasLastListenedSequence = true;
  }
  const resp = await conversationClient.updateCursor(req);
  return decodeConversationCursor(resp.cursor);
}

export interface ConversationSearchMatch { eventId: string; sequence: number; excerpt: string }
export interface ConversationSearchResponse { matches: ConversationSearchMatch[]; truncated: boolean; totalMatches: number }

export async function searchConversation(sessionId: string, query: string, limit = 500): Promise<ConversationSearchResponse> {
  const response = await conversationClient.search({ sessionId, query, limit });
  return { matches: response.matches.map((match) => ({ eventId: match.eventId, sequence: Number(match.sequence), excerpt: match.excerpt })), truncated: response.truncated, totalMatches: Number(response.totalMatches) };
}

export interface ArchivedConversationSearchMatch {
  eventId: string;
  sessionId: string;
  sequence: number;
  role: string;
  createdAt: string;
  excerpt: string;
}

export interface ArchivedConversationSearchResponse {
  matches: ArchivedConversationSearchMatch[];
  truncated: boolean;
  totalMatches: number;
  distinctSessions: number;
}

export interface ArchivedConversationSearchFilters {
  agentType?: string;
  role?: string;
  createdAfter?: string;
}

export async function searchArchivedConversations(
  query: string,
  filters: ArchivedConversationSearchFilters = {},
  limit = 100,
): Promise<ArchivedConversationSearchResponse> {
  const response = await conversationClient.searchArchived({
    query,
    limit,
    agentType: filters.agentType ?? "",
    role: filters.role ?? "",
    createdAfter: filters.createdAfter ?? "",
  });
  return {
    matches: response.matches.map((match) => ({
      eventId: match.eventId,
      sessionId: match.sessionId,
      sequence: Number(match.sequence),
      role: match.role,
      createdAt: match.createdAt,
      excerpt: match.excerpt,
    })),
    truncated: response.truncated,
    totalMatches: Number(response.totalMatches),
    distinctSessions: Number(response.distinctSessions),
  };
}

export async function getConversationRange(sessionId: string, fromSequence: number, toSequence: number): Promise<ConversationSessionResponse> {
  const response = await conversationClient.getRange({ sessionId, fromSequence: BigInt(fromSequence), toSequence: BigInt(toSequence) });
  return { sessionId: response.sessionId, events: response.events.map(decodeConversationEvent), cursor: decodeConversationCursor(response.cursor) };
}

export interface SummarizeEventResponse {
  summarized: boolean;
  speechParagraphs?: string[];
  error?: string;
}

export async function summarizeEvent(
  sessionId: string,
  eventId: string,
  signal?: AbortSignal,
): Promise<SummarizeEventResponse> {
  const resp = await conversationClient.summarizeEvent({ sessionId, eventId }, { signal });
  return {
    summarized: resp.summarized,
    speechParagraphs: resp.speechParagraphs.length > 0 ? resp.speechParagraphs : undefined,
    error: resp.error || undefined,
  };
}
