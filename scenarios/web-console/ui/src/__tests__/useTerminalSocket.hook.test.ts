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

  it("creates WebSocket with correct URL and calls onReady on open", () => {
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

    // Resize message sent immediately on open with terminal dimensions
    expect(fakeWs.sent).toHaveLength(1);
    expect(JSON.parse(fakeWs.sent[0] ?? "{}")).toEqual({ type: "resize", cols: 80, rows: 24 });

    // Should call onReady
    expect(onReady).toHaveBeenCalledOnce();
  });

  it("sends resize with terminal dimensions immediately on open", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // Resize message should be sent on open with terminal's current dimensions
    const resizeMsg = fakeWs.sent.find((raw) => {
      try {
        const msg = JSON.parse(raw) as TerminalMessage;
        return msg.type === "resize";
      } catch {
        return false;
      }
    });
    expect(resizeMsg).toBeDefined();
    expect(JSON.parse(resizeMsg ?? "{}")).toEqual({ type: "resize", cols: 80, rows: 24 });
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
    act(() => fakeWs.triggerMessage({ type: "history_end" }));
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
    expect(terminal.reset).toHaveBeenCalledTimes(1);
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
    fakeWs.sent = []; // clear the resize-on-open message

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

    // On open: flushed stdin + resize message
    const stdinMsg = fakeWs.sent.find((raw) => {
      const m = JSON.parse(raw) as TerminalMessage;
      return m.type === "stdin";
    });
    expect(stdinMsg).toBeDefined();
    expect(JSON.parse(stdinMsg ?? "{}")).toEqual({ type: "stdin", data: "echo queued" });
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
    act(() => fakeWs.triggerMessage({ type: "history_end" }));
    act(() =>
      fakeWs.triggerMessage({ type: "sync_warning", coalesced_frames: 7 }),
    );

    const warningData = findWriteCall(terminal.write, "7 output frames coalesced");
    expect(warningData).toBeTruthy();
    expect(warningData).toContain(ANSI.yellow);
    expect(warningData).toContain("terminal may lag");

    // Verify stdout after sync_warning is passed through unchanged
    // (local echo predictions are reset on sync_warning to prevent suppression)
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "post-coalesce output" }));
    const postOutput = findWriteCall(terminal.write, "post-coalesce output");
    expect(postOutput).toBeTruthy();
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
    const multiFactory = vi.fn(() => {
      const ws = new FakeWebSocket();
      sockets.push(ws);
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

    const firstWs = sockets[0] as FakeWebSocket;
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

    const firstWs = sockets[0] as FakeWebSocket;
    act(() => firstWs.triggerOpen());

    // Do 3 failed reconnections to burn attempts
    for (let i = 0; i < 3; i++) {
      act(() => (sockets[sockets.length - 1] as FakeWebSocket).triggerClose(1006));
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
    act(() => (sockets[sockets.length - 1] as FakeWebSocket).triggerClose(1006));

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

// --- History replay batching tests ---

describe("useTerminalSocket — history replay batching", () => {
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

  it("buffers stdout during history replay and flushes on history_end", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // Send multiple stdout chunks (simulating history replay).
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "chunk1" }));
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "chunk2" }));
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "chunk3" }));

    // Terminal should NOT have been written to yet (history is buffered).
    const stdoutWrite = findWriteCall(terminal.write, "chunk");
    expect(stdoutWrite).toBeUndefined();

    // Send history_end — buffer should flush as a single write.
    act(() => fakeWs.triggerMessage({ type: "history_end" }));

    const flushed = findWriteCall(terminal.write, "chunk1chunk2chunk3");
    expect(flushed).toBe("chunk1chunk2chunk3");
  });

  it("passes stdout through after history_end", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // Empty history — send history_end immediately.
    act(() => fakeWs.triggerMessage({ type: "history_end" }));

    // Now send live stdout.
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "live output" }));

    // Should be written directly (not buffered).
    const liveWrite = findWriteCall(terminal.write, "live output");
    expect(liveWrite).toBeTruthy();
  });

  it("safety timeout flushes history buffer if history_end never arrives", () => {
    vi.useFakeTimers();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // Send stdout without history_end.
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "stale-server-data" }));

    // Not written yet.
    expect(findWriteCall(terminal.write, "stale-server-data")).toBeUndefined();

    // Advance past the safety timeout (5000ms).
    act(() => {
      vi.advanceTimersByTime(5000);
    });

    // Should be flushed now.
    expect(findWriteCall(terminal.write, "stale-server-data")).toBeTruthy();
  });

  it("reconnect resets history replay state", () => {
    vi.useFakeTimers();

    const sockets: FakeWebSocket[] = [];
    const multiFactory = vi.fn(() => {
      const ws = new FakeWebSocket();
      sockets.push(ws);
      return ws as unknown as WebSocket;
    });

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket: multiFactory,
      }),
    );

    const first = sockets[0] as FakeWebSocket;

    // First connection: complete history replay.
    act(() => first.triggerOpen());
    act(() => first.triggerMessage({ type: "history_end" }));
    act(() => first.triggerMessage({ type: "stdout", data: "live1" }));
    expect(findWriteCall(terminal.write, "live1")).toBeTruthy();

    // Disconnect and reconnect.
    act(() => first.triggerClose(1006));
    act(() => {
      vi.advanceTimersByTime(400);
    });

    const second = sockets[1] as FakeWebSocket;
    act(() => second.triggerOpen());

    // New stdout should be buffered (replaying history again).
    act(() => second.triggerMessage({ type: "stdout", data: "history-on-reconnect" }));
    expect(findWriteCall(terminal.write, "history-on-reconnect")).toBeUndefined();

    // history_end flushes.
    act(() => second.triggerMessage({ type: "history_end" }));
    expect(findWriteCall(terminal.write, "history-on-reconnect")).toBeTruthy();
  });

  it("exit during history replay flushes buffer first", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // Send history stdout, then exit (no history_end).
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "final-output" }));
    act(() => fakeWs.triggerMessage({ type: "exit", code: 0 }));

    // History should have been flushed before the exit label.
    const historyWrite = findWriteCall(terminal.write, "final-output");
    expect(historyWrite).toBeTruthy();
    const exitWrite = findWriteCall(terminal.write, "[Session ended]");
    expect(exitWrite).toBeTruthy();

    // Verify ordering: history flush before exit label.
    const historyIdx = terminal.write.mock.invocationCallOrder[
      terminal.write.mock.calls.findIndex(
        (c: unknown[]) => typeof c[0] === "string" && (c[0] as string).includes("final-output"),
      )
    ] as number;
    const exitIdx = terminal.write.mock.invocationCallOrder[
      terminal.write.mock.calls.findIndex(
        (c: unknown[]) => typeof c[0] === "string" && (c[0] as string).includes("[Session ended]"),
      )
    ] as number;
    expect(historyIdx).toBeLessThan(exitIdx);
  });

  it("cleans up history timeout on unmount", () => {
    vi.useFakeTimers();

    const { unmount } = renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "buffered" }));

    // Unmount while history is still buffered.
    unmount();

    // Advancing past the timeout should not throw.
    act(() => {
      vi.advanceTimersByTime(10000);
    });

    // Terminal should NOT have been written to after unmount.
    expect(findWriteCall(terminal.write, "buffered")).toBeUndefined();
  });

  it("history_end with empty history immediately enables pass-through", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // history_end arrives immediately (no stdout before it).
    act(() => fakeWs.triggerMessage({ type: "history_end" }));

    // Subsequent stdout should go straight to terminal.write.
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "immediate" }));
    expect(findWriteCall(terminal.write, "immediate")).toBeTruthy();
  });
});

// --- History resume caching tests ---

describe("useTerminalSocket — history resume caching", () => {
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

  it("appends history_offset query param when historyOffset provided", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
        historyOffset: 5000,
      }),
    );

    expect(createSocket).toHaveBeenCalledTimes(1);
    const url = (createSocket as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(url).toContain("?history_offset=5000");
  });

  it("does not append history_offset when zero or undefined", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
        historyOffset: 0,
      }),
    );

    expect(createSocket).toHaveBeenCalledTimes(1);
    const url = (createSocket as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(url).not.toContain("history_offset");
  });

  it("resets terminal on history_end with resumed=false and cached state", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
        hasCachedState: true,
        historyOffset: 5000,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "history-data" }));
    act(() => fakeWs.triggerMessage({ type: "history_end", resumed: false, total_bytes: 10000 }));

    // terminal.reset() should have been called when the server rejected
    // the cached offset (resumed=false with hasCachedState=true).
    // Note: reset is also called on reconnect, but this is the first connection,
    // so any reset call here is from the history_end handler.
    expect(terminal.reset).toHaveBeenCalled();
  });

  it("does NOT reset terminal on history_end with resumed=true", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
        hasCachedState: true,
        historyOffset: 5000,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "delta-data" }));
    act(() => fakeWs.triggerMessage({ type: "history_end", resumed: true, total_bytes: 10000 }));

    // terminal.reset() should NOT have been called — the server honored
    // the offset and sent only delta data.
    expect(terminal.reset).not.toHaveBeenCalled();
  });

  it("does NOT reset terminal on history_end without cached state", () => {
    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "full-history" }));
    act(() => fakeWs.triggerMessage({ type: "history_end", resumed: false, total_bytes: 10000 }));

    // No cached state means no reset needed — this is a fresh connection.
    expect(terminal.reset).not.toHaveBeenCalled();
  });

  it("exposes totalBytesRef updated from history_end", () => {
    const { result } = renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "history_end", total_bytes: 42000 }));

    expect(result.current.totalBytesRef.current).toBe(42000);
  });

  it("handles history_end without total_bytes field (old server compat)", () => {
    const { result } = renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() => fakeWs.triggerMessage({ type: "history_end" }));

    // Should not crash and totalBytesRef stays at default (0).
    expect(result.current.totalBytesRef.current).toBe(0);
  });
});
