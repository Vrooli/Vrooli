import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { createTerminalStub } from "../test-utils";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";

class ScrollSocket {
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

function openSession(terminal: ReturnType<typeof createTerminalStub>) {
  const socket = new ScrollSocket();
  const view = renderHook(() => useTerminalSession({
    sessionId: "scroll-session",
    terminal: terminal as never,
    createSocket: () => socket as unknown as WebSocket,
  }));
  act(() => {
    socket.readyState = ScrollSocket.OPEN;
    socket.onopen?.();
    socket.onmessage?.({ data: JSON.stringify({ type: "session_ready", accepted_through: 0 }) } as MessageEvent<string>);
  });
  const frames = () => socket.sent.map((entry) => JSON.parse(entry) as { type: string; lines?: number; data?: string });
  return { socket, view, frames };
}

describe("terminal server scroll lane", () => {
  it("asks the server to scroll when the pane has no scrollback and no mouse tracking", async () => {
    // The tmux case: xterm sits in the alternate buffer for the whole session
    // and the program requested no mouse tracking, so neither the browser nor
    // the program can scroll. The pane's tmux history is the only history.
    const terminal = createTerminalStub({ mouseTrackingMode: "none", onAltBuffer: true });
    const { view, frames } = openSession(terminal);

    act(() => { view.result.current.scrollBy(-4, "wheel"); });

    // The controller batches onto one animation frame so a fast gesture
    // becomes one request rather than one per event.
    await waitFor(() => {
      expect(frames().filter((frame) => frame.type === "scroll")).toHaveLength(1);
    });
    expect(frames().find((frame) => frame.type === "scroll")?.lines).toBe(-4);
    // The regression: scrollLines() against the alternate buffer did nothing
    // at all, so a touch drag produced no movement and no network traffic.
    expect(terminal.scrollLines).not.toHaveBeenCalled();
    view.unmount();
  });

  it("keeps scrolling locally when the terminal owns real scrollback", () => {
    const terminal = createTerminalStub({ mouseTrackingMode: "none" });
    const { view, frames } = openSession(terminal);

    act(() => { view.result.current.scrollBy(-4, "wheel"); });

    expect(terminal.scrollLines).toHaveBeenCalledWith(-4);
    expect(frames().some((frame) => frame.type === "scroll")).toBe(false);
    view.unmount();
  });

  it("leaves the wheel to the program when it requested mouse tracking", async () => {
    const terminal = createTerminalStub({ mouseTrackingMode: "sgr", onAltBuffer: true });
    const { view, frames } = openSession(terminal);

    act(() => { view.result.current.scrollBy(-1, "wheel"); });

    await waitFor(() => {
      expect(frames().some((frame) => frame.type === "control" && frame.data?.includes("\x1b[<64;"))).toBe(true);
    });
    expect(frames().some((frame) => frame.type === "scroll")).toBe(false);
    view.unmount();
  });

  it("stops asking after the backend reports it owns no history", async () => {
    const terminal = createTerminalStub({ mouseTrackingMode: "none", onAltBuffer: true });
    const { socket, view, frames } = openSession(terminal);

    act(() => { view.result.current.scrollBy(-2, "wheel"); });
    await waitFor(() => {
      expect(frames().filter((frame) => frame.type === "scroll")).toHaveLength(1);
    });

    act(() => {
      socket.onmessage?.({
        data: JSON.stringify({ type: "scroll", data: "unsupported", reason: "unsupported" }),
      } as MessageEvent<string>);
    });

    // Without the latch every later gesture would re-send a frame that can
    // only fail, and each failure would hold the acknowledgement gate until
    // the watchdog cleared it.
    act(() => { view.result.current.scrollBy(-2, "wheel"); });
    act(() => { view.result.current.scrollBy(-2, "wheel"); });
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(frames().filter((frame) => frame.type === "scroll")).toHaveLength(1);
    view.unmount();
  });
});
