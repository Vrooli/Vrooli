import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";
import { FakeWebSocket, createFakeSocketPair, createMockTerminal } from "../test-utils";
import type { MockTerminal } from "../test-utils";

vi.mock("../api/sessions", () => ({
  buildSessionWsUrl: vi.fn((id: string) => `ws://test/sessions/${id}/ws`),
}));

// Conversation events themselves now arrive via the global SSE channel
// (useGlobalEventStream), NOT the terminal WebSocket. The only conversation
// concern still on the terminal WS is the client→server playback-telemetry
// ack, exposed as sendConversationAck.
describe("useTerminalSession — conversation acks", () => {
  let fakeWs: FakeWebSocket;
  let createSocket: ReturnType<typeof createFakeSocketPair>["createSocket"];
  let terminal: MockTerminal;

  beforeEach(() => {
    vi.clearAllMocks();
    const pair = createFakeSocketPair();
    fakeWs = pair.fakeWs;
    createSocket = pair.createSocket;
    terminal = createMockTerminal();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("sends a conversation_event_ack frame over the websocket", () => {
    const { result } = renderHook(() =>
      useTerminalSession({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => result.current.sendConversationAck("evt-ack", "claude_hook", "playback_succeeded", "ok", "browser"));

    const sent = fakeWs.sent.map((raw) => JSON.parse(raw) as {
      type: string; eventId?: string; stage?: string; backend?: string; data?: string; source?: string;
    });
    expect(sent.some((msg) =>
      msg.type === "conversation_event_ack" &&
      msg.eventId === "evt-ack" &&
      msg.source === "claude_hook" &&
      msg.stage === "playback_succeeded" &&
      msg.backend === "browser" &&
      msg.data === "ok")).toBe(true);
  });

  it("ignores an ack with a missing eventId or source", () => {
    const { result } = renderHook(() =>
      useTerminalSession({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => result.current.sendConversationAck("", "claude_hook", "playback_succeeded"));
    act(() => result.current.sendConversationAck("evt-x", "", "playback_succeeded"));

    const acks = fakeWs.sent
      .map((raw) => JSON.parse(raw) as { type: string })
      .filter((msg) => msg.type === "conversation_event_ack");
    expect(acks).toHaveLength(0);
  });

  it("declares this device's grid before an explicit take-over", () => {
    const { result } = renderHook(() =>
      useTerminalSession({
        sessionId: "sess-takeover",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => result.current.takeLease());

    const sent = fakeWs.sent.map((raw) => JSON.parse(raw) as { type: string; cols?: number; rows?: number });
    expect(sent.slice(-2)).toEqual([
      { type: "resize", cols: terminal.cols, rows: terminal.rows },
      { type: "take_lease" },
    ]);
  });

  it("routes local scrolling through the shared scroll seam", () => {
    const { result } = renderHook(() =>
      useTerminalSession({
        sessionId: "sess-scroll",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => result.current.scrollBy(2, "wheel"));

    expect(terminal.scrollLines).toHaveBeenCalledWith(2);
  });
});
