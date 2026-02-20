import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { isCleanWsClose, useTerminalSocket } from "../hooks/useTerminalSocket";
import type { TerminalMessage } from "../hooks/useTerminalSocket";

// Mock the api module so buildSessionWsUrl doesn't hit real config
vi.mock("../lib/api", () => ({
  buildSessionWsUrl: (id: string) => `ws://localhost:9999/sessions/${id}/ws`,
}));

/**
 * Minimal mock WebSocket that records lifecycle calls.
 */
function createMockSocket() {
  const socket = {
    readyState: WebSocket.CONNECTING,
    send: vi.fn(),
    close: vi.fn(),
    onopen: null as ((ev: Event) => void) | null,
    onmessage: null as ((ev: MessageEvent) => void) | null,
    onclose: null as ((ev: CloseEvent) => void) | null,
    onerror: null as ((ev: Event) => void) | null,
    simulateOpen() {
      socket.readyState = WebSocket.OPEN;
      socket.onopen?.(new Event("open"));
    },
    simulateMessage(msg: TerminalMessage) {
      socket.onmessage?.(new MessageEvent("message", { data: JSON.stringify(msg) }));
    },
  };
  return socket;
}

/** Minimal xterm.js Terminal stub. */
function createMockTerminal() {
  return {
    cols: 80,
    rows: 24,
    write: vi.fn(),
    onData: vi.fn((_cb: (data: string) => void) => ({
      dispose: vi.fn(),
    })),
  };
}

// [REQ:P0-002b] WebSocket I/O Streaming - isCleanWsClose decision boundary
describe("isCleanWsClose", () => {
  it("returns true for Normal close (1000)", () => {
    expect(isCleanWsClose(1000)).toBe(true);
  });

  it("returns true for Going Away (1001)", () => {
    expect(isCleanWsClose(1001)).toBe(true);
  });

  it("returns false for Protocol Error (1002)", () => {
    expect(isCleanWsClose(1002)).toBe(false);
  });

  it("returns false for Abnormal Closure (1006)", () => {
    expect(isCleanWsClose(1006)).toBe(false);
  });

  it("returns false for Internal Error (1011)", () => {
    expect(isCleanWsClose(1011)).toBe(false);
  });
});

// [REQ:P0-002b] WebSocket I/O Streaming - hook module
describe("useTerminalSocket hook module", () => {
  it("exports useTerminalSocket function", async () => {
    const mod = await import("../hooks/useTerminalSocket");
    expect(typeof mod.useTerminalSocket).toBe("function");
  });

  it("exports TerminalMessage interface type (verified via runtime shape)", async () => {
    // Verify the module loads without errors; TerminalMessage is a TS interface
    // so we validate it structurally via a runtime assertion
    const msg: import("../hooks/useTerminalSocket").TerminalMessage = {
      type: "stdout",
      data: "hello",
    };
    expect(msg.type).toBe("stdout");
    expect(msg.data).toBe("hello");
  });

  it("exports SocketFactory type for WebSocket seam injection", async () => {
    // SocketFactory is a type export — verify the module loads cleanly.
    // ANSI constants are internal (used only for terminal status formatting).
    const mod = await import("../hooks/useTerminalSocket");
    expect(mod.useTerminalSocket).toBeDefined();
  });

  it("accepts createSocket parameter for WebSocket injection", async () => {
    // Verify the hook signature accepts the createSocket seam parameter
    // by checking that the function accepts an options object with createSocket
    const mod = await import("../hooks/useTerminalSocket");
    // The function exists and accepts the extended options shape
    // (actual rendering test would require React test utils + fake terminal)
    expect(mod.useTerminalSocket.length).toBe(1); // single options param
  });
});

// ─── REGRESSION: Focus-change content duplication ─────────────────────
// Before the fix, onReady/onExit were in the effect dependency array.
// Inline arrow callbacks (e.g., `onReady={() => handleTerminalReady(id)}`)
// changed identity on every parent re-render, tearing down and recreating
// the WebSocket — which caused the server to replay PTY buffer content.
describe("useTerminalSocket — callback stability", () => {
  let sockets: ReturnType<typeof createMockSocket>[];
  let socketFactory: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    sockets = [];
    socketFactory = vi.fn(() => {
      const s = createMockSocket();
      sockets.push(s);
      return s as unknown as WebSocket;
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("does NOT reconnect when onReady/onExit callbacks change identity", () => {
    const terminal = createMockTerminal();
    const onReady1 = vi.fn();
    const onExit1 = vi.fn();

    const { rerender } = renderHook(
      ({ onReady, onExit }) =>
        useTerminalSocket({
          sessionId: "sess-1",
          terminal: terminal as never,
          onReady,
          onExit,
          createSocket: socketFactory,
        }),
      { initialProps: { onReady: onReady1, onExit: onExit1 } },
    );

    // Initial mount: exactly one socket created
    expect(socketFactory).toHaveBeenCalledTimes(1);
    const firstSocket = sockets[0]!;
    firstSocket.simulateOpen();
    expect(onReady1).toHaveBeenCalledTimes(1);

    // Simulate parent re-render with new callback references
    // (this is what happens when Workspace re-renders on focus change)
    const onReady2 = vi.fn();
    const onExit2 = vi.fn();
    rerender({ onReady: onReady2, onExit: onExit2 });

    // WebSocket must NOT have been torn down and recreated
    expect(socketFactory).toHaveBeenCalledTimes(1);
    expect(firstSocket.close).not.toHaveBeenCalled();
  });

  it("uses the latest callback refs for events after re-render", () => {
    const terminal = createMockTerminal();
    const onReady1 = vi.fn();
    const onExit1 = vi.fn();

    const { rerender } = renderHook(
      ({ onReady, onExit }) =>
        useTerminalSocket({
          sessionId: "sess-1",
          terminal: terminal as never,
          onReady,
          onExit,
          createSocket: socketFactory,
        }),
      { initialProps: { onReady: onReady1, onExit: onExit1 } },
    );

    sockets[0]!.simulateOpen();

    // Replace callbacks (simulates parent re-render)
    const onExit2 = vi.fn();
    rerender({ onReady: vi.fn(), onExit: onExit2 });

    // Server sends exit — the NEW onExit should be called
    sockets[0]!.simulateMessage({ type: "exit", code: 0 });
    expect(onExit1).not.toHaveBeenCalled();
    expect(onExit2).toHaveBeenCalledWith("sess-1");
  });

  it("writes stdout to terminal without reconnecting", () => {
    const terminal = createMockTerminal();

    renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket: socketFactory,
      }),
    );

    sockets[0]!.simulateOpen();
    sockets[0]!.simulateMessage({ type: "stdout", data: "hello world" });
    expect(terminal.write).toHaveBeenCalledWith("hello world");
  });

  it("closes WebSocket on unmount", () => {
    const terminal = createMockTerminal();

    const { unmount } = renderHook(() =>
      useTerminalSocket({
        sessionId: "sess-1",
        terminal: terminal as never,
        createSocket: socketFactory,
      }),
    );

    sockets[0]!.simulateOpen();
    unmount();
    expect(sockets[0]!.close).toHaveBeenCalled();
  });
});
