import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalSocket } from "../hooks/useTerminalSocket";
import { FakeWebSocket, createFakeSocketPair, createMockTerminal } from "../test-utils";
import type { MockTerminal } from "../test-utils";

vi.mock("../lib/api", () => ({
  buildSessionWsUrl: vi.fn((id: string) => `ws://test/sessions/${id}/ws`),
}));

describe("useTerminalSocket — TTS message handling", () => {
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

  it("invokes onTTS callback when a tts message arrives with data", () => {
    const mockOnTTS = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTS: mockOnTTS,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "tts", data: "Hello from AI" }));

    expect(mockOnTTS).toHaveBeenCalledOnce();
    expect(mockOnTTS).toHaveBeenCalledWith("Hello from AI");
  });

  it("does not invoke onTTS when tts message has no data", () => {
    const mockOnTTS = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTS: mockOnTTS,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "tts" }));

    expect(mockOnTTS).not.toHaveBeenCalled();
  });

  it("does not invoke onTTS when tts message has empty string data", () => {
    const mockOnTTS = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTS: mockOnTTS,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "tts", data: "" }));

    expect(mockOnTTS).not.toHaveBeenCalled();
  });

  it("does not crash when no onTTS callback is provided", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // Should not throw
    expect(() => {
      act(() => fakeWs.triggerMessage({ type: "tts", data: "No handler" }));
    }).not.toThrow();
  });

  it("does not write tts data to the terminal", () => {
    const mockOnTTS = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTS: mockOnTTS,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "history_end" }));

    // Clear any writes from history_end
    terminal.write.mockClear();

    act(() => fakeWs.triggerMessage({ type: "tts", data: "Speech only text" }));

    // TTS messages should NOT be written to the terminal
    const writeCalls = terminal.write.mock.calls as string[][];
    const hasTtsContent = writeCalls.some(
      (c) => typeof c[0] === "string" && c[0].includes("Speech only text"),
    );
    expect(hasTtsContent).toBe(false);
  });

  it("invokes onTTS for multiple successive tts messages", () => {
    const mockOnTTS = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-tts",
        terminal: terminal as never,
        createSocket,
        onTTS: mockOnTTS,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "tts", data: "First message" }));
    act(() => fakeWs.triggerMessage({ type: "tts", data: "Second message" }));

    expect(mockOnTTS).toHaveBeenCalledTimes(2);
    expect(mockOnTTS).toHaveBeenNthCalledWith(1, "First message");
    expect(mockOnTTS).toHaveBeenNthCalledWith(2, "Second message");
  });

  it("uses latest onTTS callback via ref (no stale closure)", () => {
    const firstCallback = vi.fn();
    const secondCallback = vi.fn();

    const { rerender } = renderHook(
      ({ onTTS }) =>
        useTerminalSocket({
          sessionId: "sess-tts",
          terminal: terminal as never,
          createSocket,
          onTTS,
        }),
      { initialProps: { onTTS: firstCallback } },
    );

    act(() => fakeWs.triggerOpen());

    // Rerender with a new callback
    rerender({ onTTS: secondCallback });

    act(() => fakeWs.triggerMessage({ type: "tts", data: "After rerender" }));

    expect(firstCallback).not.toHaveBeenCalled();
    expect(secondCallback).toHaveBeenCalledWith("After rerender");
  });
});
