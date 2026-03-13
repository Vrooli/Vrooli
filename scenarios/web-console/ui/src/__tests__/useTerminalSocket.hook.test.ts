import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
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

  afterEach(() => {
    vi.useRealTimers();
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

  it("shows reconnecting hint and retries for unclean close", () => {
    vi.useFakeTimers();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerClose(1006));

    const writeData = findWriteCall(terminal.write, "[Connection lost, reconnecting...]");
    expect(writeData).toBeTruthy();

    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(createSocket).toHaveBeenCalledTimes(2);
  });

  it("writes [Reconnected] after a successful reconnect", () => {
    vi.useFakeTimers();

    const first = new FakeWebSocket();
    const second = new FakeWebSocket();
    const sockets: FakeWebSocket[] = [first, second];
    let idx = 0;
    const reconnectFactory = vi.fn(() => {
      const ws = sockets[idx] ?? sockets[sockets.length - 1];
      idx += 1;
      return ws as unknown as WebSocket;
    });

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket: reconnectFactory,
      }),
    );

    act(() => first.triggerOpen());
    act(() => first.triggerClose(1006));
    act(() => {
      vi.advanceTimersByTime(400);
    });
    act(() => second.triggerOpen());

    const writeData = findWriteCall(terminal.write, "[Reconnected]");
    expect(writeData).toBeTruthy();
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

  it("queues sendInput before socket opens and flushes on open", () => {
    const { result } = renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => result.current.sendInput("echo queued"));
    expect(fakeWs.sent).toHaveLength(0);

    act(() => fakeWs.triggerOpen());

    expect(fakeWs.sent).toHaveLength(2);
    expect(JSON.parse(fakeWs.sent[0] ?? "{}")).toEqual({ type: "resize", cols: 80, rows: 24 });
    expect(JSON.parse(fakeWs.sent[1] ?? "{}")).toEqual({ type: "stdin", data: "echo queued" });
  });

  it("shows sync_warning with drop count in yellow", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() =>
      fakeWs.triggerMessage({ type: "sync_warning", dropped_frames: 7 }),
    );

    const warningData = findWriteCall(terminal.write, "7 output frames dropped");
    expect(warningData).toBeTruthy();
    expect(warningData).toContain(ANSI.yellow);
    expect(warningData).toContain("out of sync");
  });

  it("handles resize_info message without crashing", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    // resize_info is informational — should not write to terminal or throw
    act(() =>
      fakeWs.triggerMessage({ type: "resize_info", cols: 120, rows: 40 }),
    );

    const resizeData = findWriteCall(terminal.write, "resize");
    expect(resizeData).toBeUndefined();
  });

  // --- Visibility-aware reconnection tests ---

  it("defers reconnection when page is hidden on unclean close", () => {
    vi.useFakeTimers();

    // Simulate hidden page
    Object.defineProperty(document, "visibilityState", {
      value: "hidden",
      writable: true,
      configurable: true,
    });

    const addListenerSpy = vi.spyOn(document, "addEventListener");

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerClose(1006));

    // Should NOT have attempted reconnection (no timer-based retry)
    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(createSocket).toHaveBeenCalledTimes(1); // only the initial connection

    // Should show backgrounded message
    const bgMsg = findWriteCall(terminal.write, "will reconnect when tab is active");
    expect(bgMsg).toBeTruthy();

    // Should have registered a visibilitychange listener
    expect(addListenerSpy).toHaveBeenCalledWith(
      "visibilitychange",
      expect.any(Function),
    );

    // Restore
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });
    addListenerSpy.mockRestore();
  });

  it("reconnects when page becomes visible after background deferral", () => {
    vi.useFakeTimers();

    const sockets: FakeWebSocket[] = [];
    let socketIdx = 0;
    const multiFactory = vi.fn(() => {
      const ws = new FakeWebSocket();
      sockets.push(ws);
      socketIdx++;
      return ws as unknown as WebSocket;
    });

    // Start hidden
    Object.defineProperty(document, "visibilityState", {
      value: "hidden",
      writable: true,
      configurable: true,
    });

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket: multiFactory,
      }),
    );

    const firstWs = sockets[0]!;
    act(() => firstWs.triggerOpen());
    act(() => firstWs.triggerClose(1006));

    // Only initial connection so far
    expect(multiFactory).toHaveBeenCalledTimes(1);

    // Simulate tab becoming visible
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    // Should have created a new connection
    expect(multiFactory).toHaveBeenCalledTimes(2);

    // Restore
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });
  });

  it("resets reconnect attempts on visibility return", () => {
    vi.useFakeTimers();

    const sockets: FakeWebSocket[] = [];
    const multiFactory = vi.fn(() => {
      const ws = new FakeWebSocket();
      sockets.push(ws);
      return ws as unknown as WebSocket;
    });

    // Start visible — do some reconnect attempts first
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket: multiFactory,
      }),
    );

    const firstWs = sockets[0]!;
    act(() => firstWs.triggerOpen());

    // Do 3 failed reconnections to burn attempts
    for (let i = 0; i < 3; i++) {
      act(() => sockets[sockets.length - 1]!.triggerClose(1006));
      act(() => {
        vi.advanceTimersByTime(10000);
      });
    }

    const connectionsBeforeHide = multiFactory.mock.calls.length;

    // Now go hidden during a disconnect
    Object.defineProperty(document, "visibilityState", {
      value: "hidden",
      writable: true,
      configurable: true,
    });
    act(() => sockets[sockets.length - 1]!.triggerClose(1006));

    // Come back visible — should reset attempts and reconnect
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    // A new connection should have been created
    expect(multiFactory.mock.calls.length).toBeGreaterThan(connectionsBeforeHide);

    // Restore
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });
  });

  it("cleans up visibility listener on unmount", () => {
    vi.useFakeTimers();

    const removeListenerSpy = vi.spyOn(document, "removeEventListener");

    // Start hidden
    Object.defineProperty(document, "visibilityState", {
      value: "hidden",
      writable: true,
      configurable: true,
    });

    const { unmount } = renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerClose(1006));

    // Visibility listener should be registered
    unmount();

    // Should have cleaned up the listener
    expect(removeListenerSpy).toHaveBeenCalledWith(
      "visibilitychange",
      expect.any(Function),
    );

    // Restore
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });
    removeListenerSpy.mockRestore();
  });
});
