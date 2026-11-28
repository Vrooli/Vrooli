import type { ConversationAuthor, ConversationEvent, ConversationKind, ConversationThreadSummary, ConversationWorkSpan } from "../api/conversation";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#what-a-row-is
//
// One place answers "what is this row?" for the store, the list, the reader,
// playback, search and export. Rows from a server that predates
// classification carry no kind and read as messages, exactly as before.

const KINDS: readonly string[] = ["message", "attachment", "notice", "context", "tool_call", "tool_result", "reasoning", "work"] satisfies ConversationKind[];
const AUTHORS: readonly string[] = ["human", "agent", "harness", "automation", "peer"] satisfies ConversationAuthor[];

function isKind(value: string | undefined): value is ConversationKind {
  return value != null && KINDS.includes(value);
}

function isAuthor(value: string | undefined): value is ConversationAuthor {
  return value != null && AUTHORS.includes(value);
}

/**
 * The classification fields of a row from either transport (Connect or the
 * SSE hub). Unknown values are dropped rather than trusted, so a newer server
 * never puts a kind the view cannot place into the list.
 */
export function decodeClassification(e: { kind?: string; author?: string; authorName?: string; label?: string; threadId?: string }): Pick<ConversationEvent, "kind" | "author" | "authorName" | "label" | "threadId"> {
  return {
    ...(isKind(e.kind) ? { kind: e.kind } : {}),
    ...(isAuthor(e.author) ? { author: e.author } : {}),
    ...(e.authorName ? { authorName: e.authorName } : {}),
    ...(e.label ? { label: e.label } : {}),
    ...(e.threadId ? { threadId: e.threadId } : {}),
  };
}

type RowShape = Pick<ConversationEvent, "role"> & Partial<Pick<ConversationEvent, "kind" | "author" | "threadId" | "work">>;

/** A row the Messages view lists: a message, an attachment or a notice. */
export function isConversationRow(event: RowShape): boolean {
  return event.kind == null || event.kind === "message" || event.kind === "attachment" || event.kind === "notice";
}

/** A message someone wrote (not a notice, not work). */
export function isMessageRow(event: RowShape): boolean {
  return event.kind == null || event.kind === "message" || event.kind === "attachment";
}

/** A projection row standing for the work between two conversation rows. */
export function isWorkRow(event: RowShape): event is RowShape & { work: ConversationWorkSpan } {
  return event.kind === "work" && event.work != null;
}

/**
 * The session agent's own reply on its main thread: what the reader steps
 * between, what counts as unread, and what plays as it arrives.
 */
export function isAgentReply(event: RowShape): boolean {
  return event.role === "assistant"
    && (event.kind == null || event.kind === "message")
    && (event.author == null || event.author === "agent")
    && !event.threadId;
}

/** Written by the person at the keyboard (rendered as "You"). */
export function isHumanMessage(event: RowShape): boolean {
  if (event.author != null) return event.author === "human" && isMessageRow(event);
  return event.role === "user" && isMessageRow(event);
}

/** A row playback reads aloud: a message, never work, a notice or an attachment chip. */
export function isSpeakable(event: RowShape): boolean {
  return event.kind == null || event.kind === "message";
}

/** The first sequence a row covers: a work row stands for a whole range. */
export function firstSequence(event: Pick<ConversationEvent, "sequence"> & { work?: ConversationWorkSpan }): number {
  return event.work?.fromSequence ?? event.sequence;
}

/** True when the row covers `sequence` (a work row covers its range). */
export function coversSequence(event: Pick<ConversationEvent, "sequence"> & { work?: ConversationWorkSpan }, sequence: number): boolean {
  return firstSequence(event) <= sequence && sequence <= event.sequence;
}

/** A work row that summarises nothing of this thread (only other threads' rows): kept for paging, never shown. */
export function isEmptyWork(work: ConversationWorkSpan): boolean {
  return work.toolCalls + work.toolResults + work.reasoning + work.context === 0;
}

export function workEventId(threadId: string, fromSequence: number): string {
  return `work:${threadId}:${String(fromSequence)}`;
}

/**
 * Folds one live row into the tail of a projected list. A conversation row is
 * appended; anything else extends the work row at the tail (or opens one), so
 * a live view never holds raw tool rows and stays contiguous by sequence. A
 * row of another thread only extends the tail's range: it belongs to that
 * thread's group, not to this thread's work.
 */
export function foldLiveRow(events: readonly ConversationEvent[], incoming: ConversationEvent, threadId = ""): ConversationEvent[] {
  const ownThread = (incoming.threadId ?? "") === threadId;
  if (ownThread && isConversationRow(incoming)) return [...events, incoming];
  const tail = events[events.length - 1];
  const at = incoming.occurredAt || incoming.createdAt;
  if (tail && isWorkRow(tail) && tail.sequence + 1 === incoming.sequence) {
    const work = ownThread ? countInto(tail.work, incoming.kind, at) : tail.work;
    return [...events.slice(0, -1), { ...tail, sequence: incoming.sequence, work: { ...work, toSequence: incoming.sequence } }];
  }
  const empty: ConversationWorkSpan = { fromSequence: incoming.sequence, toSequence: incoming.sequence, toolCalls: 0, toolResults: 0, reasoning: 0, context: 0 };
  const row: ConversationEvent = {
    id: workEventId(threadId, incoming.sequence),
    sessionId: incoming.sessionId,
    source: incoming.source,
    role: "assistant",
    text: "",
    speechParagraphs: [],
    summarized: false,
    createdAt: incoming.createdAt,
    ...(incoming.occurredAt ? { occurredAt: incoming.occurredAt } : {}),
    sequence: incoming.sequence,
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
    kind: "work",
    ...(threadId ? { threadId } : {}),
    work: ownThread ? countInto(empty, incoming.kind, at) : empty,
  };
  return [...events, row];
}

function countInto(work: ConversationWorkSpan, kind: ConversationEvent["kind"], at: string | undefined): ConversationWorkSpan {
  const next = { ...work };
  if (kind === "tool_call") next.toolCalls += 1;
  else if (kind === "tool_result") next.toolResults += 1;
  else if (kind === "reasoning") next.reasoning += 1;
  else next.context += 1;
  if (at) {
    if (!next.startedAt || at < next.startedAt) next.startedAt = at;
    if (!next.endedAt || at > next.endedAt) next.endedAt = at;
  }
  return next;
}

/** Counts a live row of another thread into that thread's summary (adding it when it is new). */
export function foldThreadRow(threads: readonly ConversationThreadSummary[], incoming: ConversationEvent): ConversationThreadSummary[] {
  const threadId = incoming.threadId;
  if (!threadId) return [...threads];
  const at = incoming.occurredAt || incoming.createdAt;
  const conversation = isConversationRow(incoming);
  const index = threads.findIndex((thread) => thread.threadId === threadId);
  if (index < 0) {
    return [...threads, {
      threadId,
      kind: threadId.startsWith("teammate:") ? "teammate" : "subagent",
      label: incoming.authorName ?? "",
      description: "",
      parentThreadId: "",
      firstSequence: incoming.sequence,
      lastSequence: incoming.sequence,
      messageCount: conversation ? 1 : 0,
      workCount: conversation ? 0 : 1,
      ...(at ? { startedAt: at, endedAt: at } : {}),
    }];
  }
  return threads.map((thread, position) => (position !== index ? thread : {
    ...thread,
    lastSequence: Math.max(thread.lastSequence, incoming.sequence),
    messageCount: thread.messageCount + (conversation ? 1 : 0),
    workCount: thread.workCount + (conversation ? 0 : 1),
    ...(at && (!thread.endedAt || at > thread.endedAt) ? { endedAt: at } : {}),
  }));
}

/**
 * Merges a server page into loaded rows. Work rows are replaced by id: a
 * later read of the same span (it grew while the page was in flight, or live
 * rows extended it) is authoritative. Everything else is added once.
 */
export function mergeProjectedRows(existing: ConversationEvent[], incoming: readonly ConversationEvent[]): { events: ConversationEvent[]; added: ConversationEvent[] } {
  const byId = new Map(existing.map((event) => [event.id, event] as const));
  const added: ConversationEvent[] = [];
  let replaced = false;
  for (const event of incoming) {
    const current = byId.get(event.id);
    if (current == null) {
      byId.set(event.id, event);
      added.push(event);
    } else if (isWorkRow(event) && current.sequence !== event.sequence) {
      byId.set(event.id, event);
      replaced = true;
    }
  }
  if (added.length === 0 && !replaced) return { events: existing, added };
  return { events: [...byId.values()].sort((left, right) => left.sequence - right.sequence), added };
}
