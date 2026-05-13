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
}

export interface FileReferenceResolveResponse {
  input_path: string;
  resolved_path: string;
  line?: number;
  exists: boolean;
  resolution_basis: "session_cwd" | "project_root" | "absolute_allowed" | "session_upload";
  category: "markdown" | "code" | "text" | "binary";
  can_preview: boolean;
}

export interface FileReferenceContentResponse {
  path: string;
  line?: number;
  category: "markdown" | "code" | "text" | "binary";
  content_type: string;
  content: string;
  truncated: boolean;
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
  opts?: { sinceSequence?: number },
): Promise<ConversationSessionResponse> {
  const resp = await conversationClient.get({
    sessionId,
    sinceSequence:
      opts?.sinceSequence && opts.sinceSequence > 0 ? BigInt(opts.sinceSequence) : 0n,
  });
  return {
    sessionId: resp.sessionId,
    events: resp.events.map(decodeConversationEvent),
    cursor: decodeConversationCursor(resp.cursor),
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

export async function resolveFileReference(
  sessionId: string,
  path: string,
): Promise<FileReferenceResolveResponse> {
  const resp = await conversationClient.resolveFileReference({ sessionId, path });
  return {
    input_path: resp.inputPath,
    resolved_path: resp.resolvedPath,
    line: resp.hasLine ? resp.line : undefined,
    exists: resp.exists,
    resolution_basis: resp.resolutionBasis as FileReferenceResolveResponse["resolution_basis"],
    category: resp.category as FileReferenceResolveResponse["category"],
    can_preview: resp.canPreview,
  };
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

export async function getFileReferenceContent(
  sessionId: string,
  path: string,
): Promise<FileReferenceContentResponse> {
  const resp = await conversationClient.getFileReferenceContent({ sessionId, path });
  return {
    path: resp.path,
    line: resp.hasLine ? resp.line : undefined,
    category: resp.category as FileReferenceContentResponse["category"],
    content_type: resp.contentType,
    content: resp.content,
    truncated: resp.truncated,
  };
}
