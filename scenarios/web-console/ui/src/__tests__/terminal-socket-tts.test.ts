import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalSocket } from "../hooks/useTerminalSocket";
import { FakeWebSocket, createFakeSocketPair, createMockTerminal } from "../test-utils";
import type { MockTerminal } from "../test-utils";

vi.mock("../lib/api", () => ({
  buildSessionWsUrl: vi.fn((id: string) => `ws://test/sessions/${id}/ws`),
}));

describe("useTerminalSocket — TTS candidate handling", () => {
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

  it("invokes onTTSCandidate callback when a candidate arrives with data", () => {
    const mockOnTTSCandidate = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTSCandidate: mockOnTTSCandidate,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "tts_candidate", eventId: "evt-1", source: "claude_hook", data: "Hello from AI" }));

    expect(mockOnTTSCandidate).toHaveBeenCalledOnce();
    expect(mockOnTTSCandidate).toHaveBeenCalledWith(
      { eventId: "evt-1", source: "claude_hook", text: "Hello from AI" },
      expect.any(Function),
    );
  });

  it("does not invoke onTTSCandidate when candidate message is incomplete", () => {
    const mockOnTTSCandidate = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTSCandidate: mockOnTTSCandidate,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "tts_candidate", data: "missing metadata" }));

    expect(mockOnTTSCandidate).not.toHaveBeenCalled();
  });

  it("does not crash when no onTTSCandidate callback is provided", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    expect(() => {
      act(() => fakeWs.triggerMessage({ type: "tts_candidate", eventId: "evt-2", source: "codex_tailer", data: "No handler" }));
    }).not.toThrow();
  });

  it("does not write TTS candidate data to the terminal", () => {
    const mockOnTTSCandidate = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTSCandidate: mockOnTTSCandidate,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "history_end" }));

    // Clear any writes from history_end
    terminal.write.mockClear();

    act(() => fakeWs.triggerMessage({ type: "tts_candidate", eventId: "evt-3", source: "claude_hook", data: "Speech only text" }));

    // TTS messages should NOT be written to the terminal
    const writeCalls = terminal.write.mock.calls as string[][];
    const hasTtsContent = writeCalls.some(
      (c) => typeof c[0] === "string" && c[0].includes("Speech only text"),
    );
    expect(hasTtsContent).toBe(false);
  });

  it("invokes onTTSCandidate for multiple successive candidate messages", () => {
    const mockOnTTSCandidate = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTSCandidate: mockOnTTSCandidate,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "tts_candidate", eventId: "evt-4", source: "claude_hook", data: "First message" }));
    act(() => fakeWs.triggerMessage({ type: "tts_candidate", eventId: "evt-5", source: "claude_hook", data: "Second message" }));

    expect(mockOnTTSCandidate).toHaveBeenCalledTimes(2);
    expect(mockOnTTSCandidate).toHaveBeenNthCalledWith(1, { eventId: "evt-4", source: "claude_hook", text: "First message" }, expect.any(Function));
    expect(mockOnTTSCandidate).toHaveBeenNthCalledWith(2, { eventId: "evt-5", source: "claude_hook", text: "Second message" }, expect.any(Function));
  });

  it("uses latest onTTSCandidate callback via ref (no stale closure)", () => {
    const firstCallback = vi.fn();
    const secondCallback = vi.fn();

    const { rerender } = renderHook(
      ({ onTTSCandidate }) =>
        useTerminalSocket({
          sessionId: "sess-tts",
          terminal: terminal as never,
          createSocket,
          onTTSCandidate,
        }),
      { initialProps: { onTTSCandidate: firstCallback } },
    );

    act(() => fakeWs.triggerOpen());

    // Rerender with a new callback
    rerender({ onTTSCandidate: secondCallback });

    act(() => fakeWs.triggerMessage({ type: "tts_candidate", eventId: "evt-6", source: "claude_hook", data: "After rerender" }));

    expect(firstCallback).not.toHaveBeenCalled();
    expect(secondCallback).toHaveBeenCalledWith(
      { eventId: "evt-6", source: "claude_hook", text: "After rerender" },
      expect.any(Function),
    );
  });

  it("sends TTS acknowledgments over the websocket", () => {
    let ackFn: ((stage: string, message?: string, backend?: string) => void) | undefined;

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTSCandidate: (_candidate, sendAck) => { ackFn = sendAck; },
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "tts_candidate", eventId: "evt-ack", source: "claude_hook", data: "Ack me" }));
    act(() => ackFn?.("playback_succeeded", "ok", "browser"));

    const sent = fakeWs.sent.map((raw) => JSON.parse(raw) as { type: string; eventId?: string; stage?: string; backend?: string; data?: string; source?: string });
    expect(sent.some((msg) =>
      msg.type === "tts_ack" &&
      msg.eventId === "evt-ack" &&
      msg.source === "claude_hook" &&
      msg.stage === "playback_succeeded" &&
      msg.backend === "browser" &&
      msg.data === "ok")).toBe(true);
  });
});
