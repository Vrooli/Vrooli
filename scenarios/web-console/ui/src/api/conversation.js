import { createClient } from "@connectrpc/connect";
import { ConversationService } from "@vrooli/proto-types/web-console/v1/conversation/conversation_pb";
import { transport } from "./client";
// conversationClient is the Connect-Web client for ConversationService.
// Consumers should prefer the typed wrappers below, which decode proto
// bigint sequences to JS numbers and surface the snake_case shapes that
// the conversation store and MessagesPane expect.
export const conversationClient = createClient(ConversationService, transport);
function decodeConversationEvent(e) {
    return {
        id: e.id,
        sessionId: e.sessionId,
        source: e.source,
        role: e.role,
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
function decodeConversationCursor(c) {
    return {
        lastSeenSequence: c ? Number(c.lastSeenSequence) : 0,
        lastListenedSequence: c ? Number(c.lastListenedSequence) : 0,
    };
}
// ---------------------------------------------------------------------------
// Typed wrappers
// ---------------------------------------------------------------------------
export async function getConversationSession(sessionId, opts) {
    const resp = await conversationClient.get({
        sessionId,
        sinceSequence: opts?.sinceSequence && opts.sinceSequence > 0 ? BigInt(opts.sinceSequence) : 0n,
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
export async function updateConversationCursor(sessionId, patch) {
    const req = { sessionId };
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
export async function searchConversation(sessionId, query, limit = 500) {
    const response = await conversationClient.search({ sessionId, query, limit });
    return { matches: response.matches.map((match) => ({ eventId: match.eventId, sequence: Number(match.sequence), excerpt: match.excerpt })), truncated: response.truncated, totalMatches: Number(response.totalMatches) };
}
export async function getConversationRange(sessionId, fromSequence, toSequence) {
    const response = await conversationClient.getRange({ sessionId, fromSequence: BigInt(fromSequence), toSequence: BigInt(toSequence) });
    return { sessionId: response.sessionId, events: response.events.map(decodeConversationEvent), cursor: decodeConversationCursor(response.cursor) };
}
export async function summarizeEvent(sessionId, eventId, signal) {
    const resp = await conversationClient.summarizeEvent({ sessionId, eventId }, { signal });
    return {
        summarized: resp.summarized,
        speechParagraphs: resp.speechParagraphs.length > 0 ? resp.speechParagraphs : undefined,
        error: resp.error || undefined,
    };
}
