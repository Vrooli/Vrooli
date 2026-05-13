import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";
import { FakeWebSocket, createFakeSocketPair, createMockTerminal } from "../test-utils";
import type { MockTerminal } from "../test-utils";
import type { ConversationEventMessage } from "../types/terminal";

vi.mock("../api/sessions", () => ({
  buildSessionWsUrl: vi.fn((id: string) => `ws://test/sessions/${id}/ws`),
}));

describe("useTerminalSession — conversation event handling", () => {
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

  it("invokes onConversationEvent callback when an event arrives with data", () => {
    const mockOnConversationEvent = vi.fn();

    renderHook(() =>
      useTerminalSession({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onConversationEvent: mockOnConversationEvent,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "conversation_event", eventId: "evt-1", source: "claude_hook", role: "assistant", sequence: 1, data: "Hello from AI" }));

    expect(mockOnConversationEvent).toHaveBeenCalledOnce();
    expect(mockOnConversationEvent).toHaveBeenCalledWith(
      { id: "evt-1", source: "claude_hook", role: "assistant", text: "Hello from AI", createdAt: undefined, sequence: 1 },
      expect.any(Function),
    );
  });

  it("does not invoke onConversationEvent when event message is incomplete", () => {
    const mockOnConversationEvent = vi.fn();

    renderHook(() =>
      useTerminalSession({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onConversationEvent: mockOnConversationEvent,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "conversation_event", data: "missing metadata" }));

    expect(mockOnConversationEvent).not.toHaveBeenCalled();
  });

  it("does not crash when no onConversationEvent callback is provided", () => {
    renderHook(() =>
      useTerminalSession({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    expect(() => {
      act(() => fakeWs.triggerMessage({ type: "conversation_event", eventId: "evt-2", source: "codex_tailer", role: "assistant", sequence: 2, data: "No handler" }));
    }).not.toThrow();
  });

  it("does not write conversation event data to the terminal", () => {
    const mockOnConversationEvent = vi.fn();

    renderHook(() =>
      useTerminalSession({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onConversationEvent: mockOnConversationEvent,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "history_end" }));

    // Clear any writes from history_end
    terminal.write.mockClear();

    act(() => fakeWs.triggerMessage({ type: "conversation_event", eventId: "evt-3", source: "claude_hook", role: "assistant", sequence: 3, data: "Speech only text" }));

    // TTS messages should NOT be written to the terminal
    const writeCalls = terminal.write.mock.calls as string[][];
    const hasTtsContent = writeCalls.some(
      (c) => typeof c[0] === "string" && c[0].includes("Speech only text"),
    );
    expect(hasTtsContent).toBe(false);
  });

  it("invokes onConversationEvent for multiple successive event messages", () => {
    const mockOnConversationEvent = vi.fn();

    renderHook(() =>
      useTerminalSession({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onConversationEvent: mockOnConversationEvent,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "conversation_event", eventId: "evt-4", source: "claude_hook", role: "assistant", sequence: 4, data: "First message" }));
    act(() => fakeWs.triggerMessage({ type: "conversation_event", eventId: "evt-5", source: "claude_hook", role: "assistant", sequence: 5, data: "Second message" }));

    expect(mockOnConversationEvent).toHaveBeenCalledTimes(2);
    expect(mockOnConversationEvent).toHaveBeenNthCalledWith(1, { id: "evt-4", source: "claude_hook", role: "assistant", text: "First message", createdAt: undefined, sequence: 4 }, expect.any(Function));
    expect(mockOnConversationEvent).toHaveBeenNthCalledWith(2, { id: "evt-5", source: "claude_hook", role: "assistant", text: "Second message", createdAt: undefined, sequence: 5 }, expect.any(Function));
  });

  it("uses latest onConversationEvent callback via ref (no stale closure)", () => {
    const firstCallback = vi.fn();
    const secondCallback = vi.fn();

    const { rerender } = renderHook(
      ({ onConversationEvent }) =>
        useTerminalSession({
          sessionId: "sess-tts",
          terminal: terminal as never,
          createSocket,
          onConversationEvent,
        }),
      { initialProps: { onConversationEvent: firstCallback } },
    );

    act(() => fakeWs.triggerOpen());

    // Rerender with a new callback
    rerender({ onConversationEvent: secondCallback });

    act(() => fakeWs.triggerMessage({ type: "conversation_event", eventId: "evt-6", source: "claude_hook", role: "assistant", sequence: 6, data: "After rerender" }));

    expect(firstCallback).not.toHaveBeenCalled();
    expect(secondCallback).toHaveBeenCalledWith(
      { id: "evt-6", source: "claude_hook", role: "assistant", text: "After rerender", createdAt: undefined, sequence: 6 },
      expect.any(Function),
    );
  });

  it("sends conversation event acknowledgments over the websocket", () => {
    let ackFn: ((stage: string, message?: string, backend?: string) => void) | undefined;

    renderHook(() =>
      useTerminalSession({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onConversationEvent: (_event: ConversationEventMessage, sendAck: (stage: string, message?: string, backend?: string) => void) => { ackFn = sendAck; },
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "conversation_event", eventId: "evt-ack", source: "claude_hook", role: "assistant", sequence: 7, data: "Ack me" }));
    act(() => ackFn?.("playback_succeeded", "ok", "browser"));

    const sent = fakeWs.sent.map((raw) => JSON.parse(raw) as { type: string; eventId?: string; stage?: string; backend?: string; data?: string; source?: string });
    expect(sent.some((msg) =>
      msg.type === "conversation_event_ack" &&
      msg.eventId === "evt-ack" &&
      msg.source === "claude_hook" &&
      msg.stage === "playback_succeeded" &&
      msg.backend === "browser" &&
      msg.data === "ok")).toBe(true);
  });
});
