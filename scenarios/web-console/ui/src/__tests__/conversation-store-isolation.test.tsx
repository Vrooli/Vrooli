import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, beforeEach } from "vitest";
import { act } from "@testing-library/react";
import {
  useConversationStore,
  getSessionConversationEvents,
  getSessionUnreadCount,
} from "../stores/useConversationStore";
import type { ConversationEvent } from "../api/conversation";

function makeEvent(sessionId: string, sequence: number, role: "assistant" | "user" = "assistant"): ConversationEvent {
  return {
    id: `${sessionId}-evt-${sequence}`,
    sessionId,
    source: "claude_hook",
    role,
    text: `Message ${sequence}`,
    speechParagraphs: [`Message ${sequence}`],
    summarized: false,
    createdAt: new Date(0).toISOString(),
    sequence,
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "unseen",
  };
}

/** Subscribes to one session's events the way MessagesPane does, counting renders. */
function EventsProbe({ sessionId, onRender }: { sessionId: string; onRender: () => void }) {
  const events = useConversationStore((s) => getSessionConversationEvents(s, sessionId));
  onRender();
  return <div data-testid={`probe-${sessionId}`}>{events.length}</div>;
}

/** Subscribes to one session's unread count the way TabUnreadBadge does. */
function BadgeProbe({ sessionId, onRender }: { sessionId: string; onRender: () => void }) {
  const unread = useConversationStore((s) => getSessionUnreadCount(s, sessionId));
  onRender();
  return <div data-testid={`badge-${sessionId}`}>{unread}</div>;
}

beforeEach(() => {
  useConversationStore.setState({ sessions: {}, viewModes: {} });
});

describe("conversation store subscription isolation (Layer 0.1)", () => {
  it("an event in session A does not re-render a subscriber to session B", () => {
    let aRenders = 0;
    let bRenders = 0;
    render(
      <>
        <EventsProbe sessionId="A" onRender={() => { aRenders++; }} />
        <EventsProbe sessionId="B" onRender={() => { bRenders++; }} />
      </>,
    );
    const aBaseline = aRenders;
    const bBaseline = bRenders;

    act(() => useConversationStore.getState().appendEvent(makeEvent("A", 1)));

    expect(aRenders).toBeGreaterThan(aBaseline); // A updated
    expect(bRenders).toBe(bBaseline); // B untouched
  });

  it("an event in session A does not re-render session B's unread badge", () => {
    let bBadgeRenders = 0;
    render(<BadgeProbe sessionId="B" onRender={() => { bBadgeRenders++; }} />);
    const baseline = bBadgeRenders;

    act(() => useConversationStore.getState().appendEvent(makeEvent("A", 1)));

    expect(bBadgeRenders).toBe(baseline);
  });

  it("the events selector is referentially stable when that session is unchanged", () => {
    act(() => useConversationStore.getState().appendEvent(makeEvent("A", 1)));
    const before = getSessionConversationEvents(useConversationStore.getState(), "A");

    // An unrelated session's update must not change A's slice identity.
    act(() => useConversationStore.getState().appendEvent(makeEvent("B", 1)));
    const after = getSessionConversationEvents(useConversationStore.getState(), "A");

    expect(after).toBe(before);
  });

  it("returns a stable empty array for an unknown session", () => {
    const first = getSessionConversationEvents(useConversationStore.getState(), "missing");
    const second = getSessionConversationEvents(useConversationStore.getState(), "missing");
    expect(first).toBe(second);
    expect(first).toHaveLength(0);
  });

  it("badge increments from a store event with no terminal mounted (Layer 1 enabler)", () => {
    // No TerminalPane / WS exists in this test — the badge must still count the
    // unread assistant event purely from store state.
    act(() => useConversationStore.getState().appendEvent(makeEvent("A", 1, "assistant")));
    expect(getSessionUnreadCount(useConversationStore.getState(), "A")).toBe(1);

    act(() => useConversationStore.getState().appendEvent(makeEvent("A", 2, "user")));
    expect(getSessionUnreadCount(useConversationStore.getState(), "A")).toBe(1); // user msgs don't count
  });
});
