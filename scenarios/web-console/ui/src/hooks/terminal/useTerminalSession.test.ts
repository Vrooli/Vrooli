import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { appendOutputProbe, stripTerminalResponses, useTerminalSession } from "./useTerminalSession";

class SessionSocket {
  static readonly OPEN = 1;
  readyState = 0;
  bufferedAmount = 0;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent<string>) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  send(data: string): void { this.sent.push(data); }
  close(): void { this.readyState = 0; this.onclose?.({ code: 1000 } as CloseEvent); }
}

function terminalFixture() {
  const onData = vi.fn();
  onData.mockReturnValue({ dispose: vi.fn() });
  const normal = {};
  const alternate = {};
  return {
    cols: 80,
    rows: 24,
    options: {},
    modes: { mouseTrackingMode: "none" },
    element: null,
    buffer: { active: normal, normal, alternate, cursorX: 4, cursorY: 5 },
    reset: vi.fn(),
    clear: vi.fn(),
    write: vi.fn(),
    resize: vi.fn(),
    onData,
    scrollLines: vi.fn(),
  };
}

describe("stripTerminalResponses", () => {
  it("removes terminal query replies while preserving ordinary input", () => {
    expect(stripTerminalResponses("plain input")).toBe("plain input");
    expect(stripTerminalResponses("before\x1b[0nafter")).toBe("beforeafter");
    expect(stripTerminalResponses("before\x1b]10;rgb:ffff/ffff/ffff\x07after")).toBe("beforeafter");
  });

  it("routes the session protocol through one socket and reports state transitions", () => {
    const socket = new SessionSocket();
    const terminal = terminalFixture();
    const statuses: unknown[] = [];
    const exits: string[] = [];
    const onData = terminal.onData as ReturnType<typeof vi.fn>;
    const createSocket = () => socket as unknown as WebSocket;
    const { result, unmount } = renderHook(() => useTerminalSession({
      sessionId: "session-1",
      terminal: terminal as never,
      createSocket,
      onStatus: (status) => statuses.push(status),
      onExit: (id) => exits.push(id),
    }));

    act(() => {
      socket.readyState = SessionSocket.OPEN;
      socket.onopen?.();
    });
    expect(socket.sent.map((entry) => JSON.parse(entry))).toContainEqual(expect.objectContaining({ type: "resize", cols: 80, rows: 24 }));

    const message = (payload: Record<string, unknown>) => act(() => {
      socket.onmessage?.({ data: JSON.stringify(payload) } as MessageEvent<string>);
    });
    message({ type: "session_ready", gen: 2, mouse_mode_known: true, mouse_mode: true, accepted_through: 0 });
    message({ type: "session_ready" });
    message({ type: "stdin_ack", accepted_through: 0, ok: false, reason: "rejected" });
    message({ type: "stdin_ack", accepted_through: 0, ok: false, reason: "unreconcilable" });
    message({ type: "stdin_ack", accepted_through: 0, ok: true });
    message({ type: "echo_state", echo_known: true, echo_enabled: true, in_alt_buffer: false, cursor_at_line_end: true });
    message({ type: "echo_state", echo_known: false, in_alt_buffer: true });
    message({ type: "stdout", data: "" });
    message({ type: "stdout", data: "snapshot" });
    message({ type: "history_end" });
    message({ type: "stdout", data: "live" });
    message({ type: "size_info", cols: 100, rows: 30, holdsLease: false, leaderDevice: "tablet", viewerCount: 2 });
    message({ type: "size_info", cols: 0, rows: 0 });
    expect(terminal.write).toHaveBeenNthCalledWith(1, "snapshot");
    expect(terminal.write).toHaveBeenNthCalledWith(2, "live");
    expect(terminal.resize).toHaveBeenCalledWith(100, 30);
    expect(result.current.serverSize).toEqual({ cols: 100, rows: 30 });
    expect(result.current.isFollower).toBe(true);
    expect(result.current.leaderDevice).toBe("tablet");
    message({ type: "presence", holdsLease: false, leaderDevice: "laptop", viewerCount: 3 });
    expect(result.current.viewerCount).toBe(3);
    expect(result.current.leaderDevice).toBe("laptop");
    message({ type: "size_info", cols: 100, rows: 30, holdsLease: true });

    act(() => {
      result.current.submitInput("a", "typing");
      result.current.submitInput("b", "typing");
    });

    message({ type: "mouse_mode", data: "off" });
    expect(result.current.mouseMode).toBe(false);
    message({ type: "mouse_mode", data: "unsupported" });
    expect(result.current.mouseMode).toBeNull();
    message({ type: "snapshot_notice" });
    message({ type: "snapshot_notice", data: "truncated" });
    message({ type: "resync" });
    message({ type: "exit", code: 7 });
    message({ type: "exit", code: 0 });
    message({ type: "error", data: "Invalid message format" });
    message({ type: "error", data: "Terminal process is not accepting input" });
    message({ type: "error", data: "session_not_ready" });
    message({ type: "error", data: "" });
    message({ type: "sync_warning", data: "coalesced" });
    message({ type: "unknown_future_message" });
    expect(terminal.reset).toHaveBeenCalled();
    expect(terminal.clear).toHaveBeenCalled();
    expect(exits).toEqual(["session-1", "session-1"]);
    expect(statuses).toEqual(expect.arrayContaining([
      { kind: "resynced", detail: "Scrollback was truncated for replay" },
      { kind: "resynced", detail: "truncated" },
      { kind: "resynced" },
      { kind: "input-desynced", detail: "Reliable input is out of sync at byte 0. Reconnect or reopen this pane to recover." },
      { kind: "session-ended", detail: "Session ended with exit code 7" },
      { kind: "error", detail: "The terminal session did not confirm readiness in time. Reconnect or reopen this pane." },
      { kind: "error", detail: "Terminal error" },
    ]));

    act(() => {
      result.current.sendResize(0, 0);
      result.current.sendResize(120, 40);
      result.current.sendConversationAck("", "ui", "played");
      result.current.sendConversationAck("event-1", "ui", "played", "done", "browser");
      result.current.takeLease();
    });
    expect(socket.sent.map((entry) => JSON.parse(entry)).filter((entry) => entry.type === "resize")).toContainEqual({ type: "resize", cols: 120, rows: 40 });
    expect(socket.sent.map((entry) => JSON.parse(entry))).toContainEqual(expect.objectContaining({ type: "conversation_event_ack", eventId: "event-1", backend: "browser", data: "done" }));
    expect(socket.sent.map((entry) => JSON.parse(entry)).some((entry) => entry.type === "take_lease")).toBe(true);
    expect(onData).toHaveBeenCalled();
    unmount();
  });
});

describe("reconnect presentation", () => {
  it("reports reconnect in pane status without writing operator text to xterm", () => {
    const socket = new SessionSocket();
    const terminal = terminalFixture();
    const statuses: unknown[] = [];
    const { unmount } = renderHook(() => useTerminalSession({
      sessionId: "reconnect-session",
      terminal: terminal as never,
      createSocket: () => socket as unknown as WebSocket,
      onStatus: (status) => statuses.push(status),
    }));

    act(() => {
      socket.readyState = SessionSocket.OPEN;
      socket.onopen?.();
    });
    const message = (payload: Record<string, unknown>) => act(() => {
      socket.onmessage?.({ data: JSON.stringify(payload) } as MessageEvent<string>);
    });
    message({ type: "session_ready", accepted_through: 0 });
    message({ type: "stdout", data: "server-screen" });

    act(() => socket.onclose?.({ code: 1006 } as CloseEvent));
    act(() => socket.onopen?.());

    expect(statuses).toContainEqual({ kind: "disconnected" });
    expect(statuses).toContainEqual({ kind: "reconnected" });
    expect(terminal.write).toHaveBeenCalledWith("server-screen");
    expect(terminal.write).not.toHaveBeenCalledWith(expect.stringContaining("Disconnected"));
    expect(terminal.write).not.toHaveBeenCalledWith(expect.stringContaining("Reconnected"));
    unmount();
  });
});

describe("terminal debug output probe", () => {
  it("returns before touching window when terminal debug is disabled", () => {
    const descriptor = Object.getOwnPropertyDescriptor(globalThis, "window");
    Object.defineProperty(globalThis, "window", {
      configurable: true,
      get: () => { throw new Error("window should not be read"); },
    });
    try {
      expect(() => appendOutputProbe("session-1", "output")).not.toThrow();
    } finally {
      if (descriptor) Object.defineProperty(globalThis, "window", descriptor);
    }
  });
});
