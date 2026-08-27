import { beforeEach, describe, expect, it } from "vitest";
import type { ConversationEvent } from "../api/conversation";
import type { MessageCaptureStatus } from "../api/messageCapture";
import { UNKNOWN_CAPTURE } from "../api/messageCapture";
import {
  createConversationSessionState,
  getSessionConversationView,
  useConversationStore,
} from "./useConversationStore";

const event = (id: string, sequence: number): ConversationEvent => ({
  id,
  sessionId: "s",
  source: "claude_tailer",
  role: "assistant",
  text: id,
  speechParagraphs: [id],
  summarized: false,
  createdAt: "2026-08-27T00:00:00Z",
  sequence,
  deliveryState: "received",
  ttsState: "idle",
  consumptionState: "unseen",
});

const capture = (overrides: Partial<MessageCaptureStatus>): MessageCaptureStatus => ({
  ...UNKNOWN_CAPTURE,
  ...overrides,
});

const CAPTURING = capture({ state: "capturing", summary: "Messages are being captured." });

describe("conversation view state", () => {
  beforeEach(() => {
    useConversationStore.setState({ sessions: {}, viewModes: {} });
  });

  it("shows loading, not emptiness, before the first load resolves", () => {
    // The regression this guards: the pane tested events.length first, so the
    // empty state flashed on every open while the request was still in flight.
    expect(getSessionConversationView(useConversationStore.getState(), "s")).toEqual({ kind: "loading" });

    useConversationStore.getState().beginLoad("s");
    expect(getSessionConversationView(useConversationStore.getState(), "s")).toEqual({ kind: "loading" });
  });

  it("distinguishes a failed load from an empty conversation", () => {
    useConversationStore.getState().failLoad("s", { message: "Web Console couldn't reach the server.", code: "unavailable", retryable: true });

    const view = getSessionConversationView(useConversationStore.getState(), "s");
    expect(view.kind).toBe("failed");
    if (view.kind !== "failed") throw new Error("expected failed view");
    expect(view.error.retryable).toBe(true);
  });

  it("reports unavailable capture rather than an empty conversation", () => {
    useConversationStore.getState().hydrateSession("s", [], { lastSeenSequence: 0, lastListenedSequence: 0 }, {
      oldestSequence: 0,
      hasOlder: false,
      totalCount: 0,
      capture: capture({ state: "unavailable", reasonCode: "hook_not_registered", summary: "Messages aren't being captured." }),
    });

    const view = getSessionConversationView(useConversationStore.getState(), "s");
    expect(view.kind).toBe("unavailable");
    if (view.kind !== "unavailable") throw new Error("expected unavailable view");
    expect(view.capture.reasonCode).toBe("hook_not_registered");
  });

  it("reports a plain terminal as not-applicable", () => {
    useConversationStore.getState().hydrateSession("s", [], { lastSeenSequence: 0, lastListenedSequence: 0 }, {
      oldestSequence: 0,
      hasOlder: false,
      totalCount: 0,
      capture: capture({ state: "not_applicable", reasonCode: "no_agent" }),
    });

    expect(getSessionConversationView(useConversationStore.getState(), "s").kind).toBe("not-applicable");
  });

  it("only reports empty when capture is healthy and there is nothing to show", () => {
    useConversationStore.getState().hydrateSession("s", [], { lastSeenSequence: 0, lastListenedSequence: 0 }, {
      oldestSequence: 0,
      hasOlder: false,
      totalCount: 0,
      capture: CAPTURING,
    });

    expect(getSessionConversationView(useConversationStore.getState(), "s").kind).toBe("empty");
  });

  it("shows messages whenever any exist, regardless of capture state", () => {
    useConversationStore.getState().hydrateSession("s", [event("e1", 1)], { lastSeenSequence: 0, lastListenedSequence: 0 }, {
      oldestSequence: 1,
      hasOlder: false,
      totalCount: 1,
      capture: capture({ state: "unavailable", reasonCode: "transcript_missing" }),
    });

    // History we already have is still worth showing; the capture fault only
    // explains why nothing new is arriving.
    expect(getSessionConversationView(useConversationStore.getState(), "s").kind).toBe("messages");
  });
});

describe("mergeEvents paging preservation", () => {
  beforeEach(() => {
    useConversationStore.setState({ sessions: {}, viewModes: {} });
  });

  it("keeps paging metadata across a refresh", () => {
    // The regression this guards: mergeEvents rebuilt the session object
    // without spreading the existing one, wiping hasOlder/windowOldestSequence/
    // totalCount. Because the pane refreshes on mount and on every window
    // focus, "load older messages" stopped working almost immediately.
    useConversationStore.getState().hydrateSession("s", [event("e2", 2)], { lastSeenSequence: 0, lastListenedSequence: 0 }, {
      oldestSequence: 2,
      hasOlder: true,
      totalCount: 900,
      capture: CAPTURING,
    });

    useConversationStore.getState().mergeEvents("s", [event("e3", 3)], { lastSeenSequence: 2, lastListenedSequence: 0 }, CAPTURING);

    const session = useConversationStore.getState().sessions.s;
    expect(session?.events.map((e) => e.id)).toEqual(["e2", "e3"]);
    expect(session?.hasOlder).toBe(true);
    expect(session?.windowOldestSequence).toBe(2);
    expect(session?.totalCount).toBe(900);
  });

  it("carries the newest capture diagnosis and clears a prior error", () => {
    useConversationStore.getState().failLoad("s", { message: "offline", code: "network", retryable: true });
    useConversationStore.getState().mergeEvents("s", [event("e1", 1)], undefined, CAPTURING);

    const session = useConversationStore.getState().sessions.s;
    expect(session?.status).toBe("loaded");
    expect(session?.error).toBeUndefined();
    expect(session?.capture.state).toBe("capturing");
  });

  it("createConversationSessionState produces a view-ready slice", () => {
    // Seeding the store with a hand-built object used to be possible and
    // produced a slice with no status, which resolved as "loaded and empty".
    const slice = createConversationSessionState({ events: [event("e1", 1)] });
    expect(slice.status).toBe("loaded");
    expect(slice.capture).toEqual(UNKNOWN_CAPTURE);
  });
});
