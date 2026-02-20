import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalSocket, type TerminalMessage } from "../hooks/useTerminalSocket";
import { ANSI } from "../lib/ansi";
import { FakeWebSocket, createFakeSocketPair, createMockTerminal, findWriteCall } from "../test-utils";
import type { MockTerminal } from "../test-utils";

// [REQ:P0-002b] WebSocket I/O Streaming — hook behavioral tests
// [REQ:P0-004b] api-base WebSocket Integration — socket factory seam

vi.mock("../lib/api", () => ({
  buildSessionWsUrl: vi.fn((id: string) => `ws://test/sessions/${id}/ws`),
}));

describe("useTerminalSocket hook", () => {
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

  it("creates WebSocket with correct URL and sends initial resize on open", () => {
    const onReady = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
        onReady,
      }),
    );

    expect(createSocket).toHaveBeenCalledWith("ws://test/sessions/sess-1/ws");

    // Simulate WebSocket open
    act(() => fakeWs.triggerOpen());

    // Should send resize message with terminal dimensions
    expect(fakeWs.sent).toHaveLength(1);
    const resizeMsg = JSON.parse(fakeWs.sent[0] ?? "{}") as TerminalMessage;
    expect(resizeMsg).toEqual({ type: "resize", cols: 80, rows: 24 });

    // Should call onReady
    expect(onReady).toHaveBeenCalledOnce();
  });

  it("does not create WebSocket when terminal is null", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: null,
        createSocket,
      }),
    );

    expect(createSocket).not.toHaveBeenCalled();
  });

  it("writes stdout data to terminal", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "hello world" }));

    expect(terminal.write).toHaveBeenCalledWith("hello world");
  });

  it("handles exit code 0 with gray label and calls onExit", () => {
    const onExit = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
        onExit,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "exit", code: 0 }));

    // Should write gray "[Session ended]" label
    const writeData = findWriteCall(terminal.write, "[Session ended]");
    expect(writeData).toBeTruthy();
    expect(writeData).toContain(ANSI.gray);
    expect(writeData).not.toContain("exit code");

    expect(onExit).toHaveBeenCalledWith("sess-1");
  });

  it("handles non-zero exit code with red label", () => {
    const onExit = vi.fn();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
        onExit,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "exit", code: 127 }));

    const writeData = findWriteCall(terminal.write, "exit code 127");
    expect(writeData).toBeTruthy();
    expect(writeData).toContain(ANSI.red);
    expect(onExit).toHaveBeenCalledWith("sess-1");
  });

  it("renders error message with recovery hint for known errors", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() =>
      fakeWs.triggerMessage({
        type: "error",
        data: "Terminal process is not accepting input",
      }),
    );

    // Should write error message
    const errorData = findWriteCall(terminal.write, "[Error:");
    expect(errorData).toBeTruthy();

    // Should write recovery hint
    const hintData = findWriteCall(terminal.write, "Close this pane");
    expect(hintData).toBeTruthy();
  });

  it("renders error message without hint for unknown errors", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() =>
      fakeWs.triggerMessage({ type: "error", data: "some unknown error" }),
    );

    const errorData = findWriteCall(terminal.write, "[Error: some unknown error]");
    expect(errorData).toBeTruthy();

    // No recovery hint for unknown errors
    const hintData = findWriteCall(terminal.write, "Close this pane");
    expect(hintData).toBeUndefined();
  });

  it("shows [Disconnected] for clean close (1000)", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerClose(1000));

    const writeData = findWriteCall(terminal.write, "[Disconnected]");
    expect(writeData).toBeTruthy();
    expect(writeData).toContain(ANSI.gray);
  });

  it("shows [Connection lost] with reconnect hint for unclean close", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerClose(1006));

    const writeData = findWriteCall(terminal.write, "[Connection lost]");
    expect(writeData).toBeTruthy();
    expect(writeData).toContain(ANSI.red);
  });

  it("forwards terminal input to WebSocket as stdin messages", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // Clear the initial resize message
    fakeWs.sent = [];

    // Simulate user typing
    act(() => terminal.simulateInput("ls -la\r"));

    expect(fakeWs.sent).toHaveLength(1);
    const msg = JSON.parse(fakeWs.sent[0] ?? "{}") as TerminalMessage;
    expect(msg).toEqual({ type: "stdin", data: "ls -la\r" });
  });

  it("closes WebSocket and disposes input listener on cleanup", () => {
    const { unmount } = renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    const firstResult = terminal.onData.mock.results[0];
    expect(firstResult).toBeDefined();
    const disposable = firstResult?.value as { dispose: ReturnType<typeof vi.fn> };

    unmount();

    expect(fakeWs.closed).toBe(true);
    expect(disposable.dispose).toHaveBeenCalled();
  });

  it("ignores malformed non-JSON messages", () => {
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => {
      fakeWs.onmessage?.(new MessageEvent("message", { data: "not json{{{" }));
    });

    expect(warnSpy).toHaveBeenCalledWith(
      "WebSocket: received non-JSON message",
      "not json{{{",
    );
    // Terminal should not have been written to with the malformed content
    const dataWritten = findWriteCall(terminal.write, "not json");
    expect(dataWritten).toBeUndefined();

    warnSpy.mockRestore();
  });

  it("returns sendInput and sendResize functions", () => {
    const { result } = renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    fakeWs.sent = [];

    act(() => result.current.sendInput("echo hello"));

    expect(fakeWs.sent).toHaveLength(1);
    expect(JSON.parse(fakeWs.sent[0] ?? "{}")).toEqual({ type: "stdin", data: "echo hello" });

    fakeWs.sent = [];

    act(() => result.current.sendResize(120, 40));

    expect(fakeWs.sent).toHaveLength(1);
    expect(JSON.parse(fakeWs.sent[0] ?? "{}")).toEqual({ type: "resize", cols: 120, rows: 40 });
  });
});
