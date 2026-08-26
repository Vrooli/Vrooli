import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";

class WheelSocket {
  static readonly OPEN = 1;
  readyState = 0;
  bufferedAmount = 0;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent<string>) => void) | null = null;
  onclose: (() => void) | null = null;
  send(data: string): void { this.sent.push(data); }
  close(): void { this.readyState = 0; this.onclose?.(); }
}

function terminalFixture() {
  const onData = vi.fn();
  onData.mockReturnValue({ dispose: vi.fn() });
  const normal = {};
  return {
    cols: 80,
    rows: 24,
    options: {},
    modes: { mouseTrackingMode: "sgr" },
    buffer: { active: normal, normal, alternate: {}, cursorX: 0, cursorY: 0 },
    onData,
    reset: vi.fn(),
    clear: vi.fn(),
    write: vi.fn(),
    resize: vi.fn(),
    scrollLines: vi.fn(),
  };
}

describe("terminal wheel control lane", () => {
  it("sends a tracked wheel report as control without stdin sequencing or queueing", () => {
    const socket = new WheelSocket();
    const terminal = terminalFixture();
    const { result, unmount } = renderHook(() => useTerminalSession({
      sessionId: "wheel-session",
      terminal: terminal as never,
      createSocket: () => socket as unknown as WebSocket,
    }));

    act(() => {
      socket.readyState = WheelSocket.OPEN;
      socket.onopen?.();
      socket.onmessage?.({ data: JSON.stringify({ type: "session_ready", accepted_through: 0 }) } as MessageEvent<string>);
    });

    const onData = terminal.onData as ReturnType<typeof vi.fn>;
    act(() => onData.mock.calls[0]?.[0]?.("\x1b[<65;40;12M"));

    const frames = socket.sent.map((entry) => JSON.parse(entry) as { type: string; data?: string; seq?: number });
    expect(frames).toContainEqual({ type: "control", data: "\x1b[<65;40;12M" });
    expect(frames.some((frame) => frame.type === "stdin")).toBe(false);
    expect(frames.find((frame) => frame.type === "control")).not.toHaveProperty("seq");
    expect(result.current.getPendingInputSnapshot()).toHaveLength(0);

    act(() => onData.mock.calls[0]?.[0]?.("a"));
    expect(socket.sent.map((entry) => JSON.parse(entry)).some((frame) => frame.type === "stdin")).toBe(true);
    expect(result.current.getPendingInputSnapshot()).toHaveLength(0);
    unmount();
  });
});
