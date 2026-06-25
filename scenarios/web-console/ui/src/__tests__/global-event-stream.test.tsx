import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import {
  dispatchGlobalEvent,
  useGlobalEventStream,
} from "../hooks/useGlobalEventStream";
import { useConversationStore, getSessionUnreadCount, getSessionConversationEvents } from "../stores/useConversationStore";

const { mockRefresh } = vi.hoisted(() => ({ mockRefresh: vi.fn().mockResolvedValue(true) }));
vi.mock("../hooks/useConversationSession", () => ({
  refreshConversationSession: mockRefresh,
}));

// Minimal EventSource fake: captures per-kind listeners and lets tests emit.
class FakeEventSource {
  listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
  closed = false;
  onerror: ((event: Event) => void) | null = null;
  constructor(public url: string) {}
  addEventListener(type: string, cb: EventListener) {
    (this.listeners[type] ||= []).push(cb as (e: MessageEvent) => void);
  }
  removeEventListener(type: string, cb: EventListener) {
    this.listeners[type] = (this.listeners[type] || []).filter((f) => f !== cb);
  }
  close() { this.closed = true; }
  emit(type: string, data: unknown) {
    for (const cb of this.listeners[type] || []) cb({ data: JSON.stringify(data) } as MessageEvent);
  }
}

function conversationEnvelope(id: number, sessionId: string, sequence: number, role: "assistant" | "user" = "assistant") {
  return {
    id,
    session_id: sessionId,
    kind: "conversation_event" as const,
    sequence,
    payload: {
      id: `${sessionId}-evt-${sequence}`,
      source: "claude_hook",
      role,
      text: `Message ${sequence}`,
      sequence,
    },
  };
}

beforeEach(() => {
  useConversationStore.setState({ sessions: {}, viewModes: {} });
  mockRefresh.mockClear();
  vi.restoreAllMocks();
});

describe("dispatchGlobalEvent (Layer 1)", () => {
  it("appends a conversation_event into the store and counts it unread", () => {
    dispatchGlobalEvent(conversationEnvelope(1, "A", 1));
    expect(getSessionConversationEvents(useConversationStore.getState(), "A")).toHaveLength(1);
    expect(getSessionUnreadCount(useConversationStore.getState(), "A")).toBe(1);
  });

  it("dedupes a re-delivered event by event id (store-level)", () => {
    dispatchGlobalEvent(conversationEnvelope(1, "A", 1));
    dispatchGlobalEvent(conversationEnvelope(2, "A", 1)); // different globalId, same payload id
    expect(getSessionConversationEvents(useConversationStore.getState(), "A")).toHaveLength(1);
  });

  it("applies a conversation_event_update and surfaces summarizeError", () => {
    dispatchGlobalEvent(conversationEnvelope(1, "A", 1));
    const onSummarizeError = vi.fn();
    dispatchGlobalEvent({
      id: 2,
      session_id: "A",
      kind: "conversation_event_update",
      sequence: 1,
      payload: { id: "A-evt-1", speechParagraphs: ["short"], summarized: true, summarizeError: "boom" },
    }, onSummarizeError);

    const events = getSessionConversationEvents(useConversationStore.getState(), "A");
    expect(events[0]?.speechParagraphs).toEqual(["short"]);
    expect(events[0]?.summarized).toBe(true);
    expect(onSummarizeError).toHaveBeenCalledWith("A", "A-evt-1", "boom");
  });

  it("refetches the affected session on conversation_out_of_sync", () => {
    dispatchGlobalEvent({ id: 5, session_id: "A", kind: "conversation_out_of_sync", sequence: 0, payload: {} });
    expect(mockRefresh).toHaveBeenCalledWith("A");
  });
});

describe("useGlobalEventStream idempotency (Layer 1)", () => {
  it("does not double-apply an event replayed with the same global id on reconnect", () => {
    const sources: FakeEventSource[] = [];
    renderHook(() => useGlobalEventStream({ createEventSource: (url) => {
      const s = new FakeEventSource(url);
      sources.push(s);
      return s as unknown as EventSource;
    } }));
    const es = sources[0];
    expect(es).toBeDefined();

    const env = conversationEnvelope(7, "A", 1);
    act(() => es?.emit("conversation_event", env));
    act(() => es?.emit("conversation_event", env)); // replayed overlap — same global id

    expect(getSessionConversationEvents(useConversationStore.getState(), "A")).toHaveLength(1);
    expect(getSessionUnreadCount(useConversationStore.getState(), "A")).toBe(1);
  });

  it("processes every out-of-sync nudge (id:0 must not be deduped)", () => {
    const sources: FakeEventSource[] = [];
    renderHook(() => useGlobalEventStream({ createEventSource: (url) => {
      const s = new FakeEventSource(url);
      sources.push(s);
      return s as unknown as EventSource;
    } }));
    const es = sources[0];
    const nudge = { id: 0, session_id: "A", kind: "conversation_out_of_sync", sequence: 0, payload: {} };
    act(() => es?.emit("conversation_out_of_sync", nudge));
    act(() => es?.emit("conversation_out_of_sync", nudge));
    expect(mockRefresh).toHaveBeenCalledTimes(2);
  });

  it("closes the EventSource on unmount", () => {
    const sources: FakeEventSource[] = [];
    const { unmount } = renderHook(() => useGlobalEventStream({ createEventSource: (url) => {
      const s = new FakeEventSource(url);
      sources.push(s);
      return s as unknown as EventSource;
    } }));
    const es = sources[0];
    expect(es).toBeDefined();
    expect(es?.closed).toBe(false);
    unmount();
    expect(es?.closed).toBe(true);
  });

  it("logs stream errors without closing the EventSource", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const sources: FakeEventSource[] = [];
    renderHook(() => useGlobalEventStream({ createEventSource: (url) => {
      const s = new FakeEventSource(url);
      sources.push(s);
      return s as unknown as EventSource;
    } }));
    const es = sources[0];
    expect(es?.onerror).toBeTypeOf("function");

    act(() => es?.onerror?.(new Event("error")));

    expect(warn).toHaveBeenCalled();
    const [message, details] = warn.mock.calls[0] as [string, { url: string }];
    expect(message).toBe("[web-console] global event stream error");
    expect(details.url).toContain("/events/stream");
    expect(es?.closed).toBe(false);
  });
});
