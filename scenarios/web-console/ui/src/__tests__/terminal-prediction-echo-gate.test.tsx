import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";

class PredictionSocket {
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
  const alternate = {};
  return {
    cols: 80,
    rows: 24,
    options: {},
    modes: { mouseTrackingMode: "none" },
    buffer: { active: normal, normal, alternate, cursorX: 4, cursorY: 5 },
    reset: vi.fn(),
    clear: vi.fn(),
    write: vi.fn(),
    resize: vi.fn(),
    onData,
  };
}

describe("terminal prediction echo gate", () => {
  it("clears speculative text as soon as server echo state enters alternate or unknown mode", () => {
    const socket = new PredictionSocket();
    const terminal = terminalFixture();
    const predictionContainer = document.createElement("div");
    const screen = document.createElement("div");
    screen.className = "xterm-screen";
    predictionContainer.appendChild(screen);

    const { result, unmount } = renderHook(() => useTerminalSession({
      sessionId: "prediction-session",
      terminal: terminal as never,
      predictionContainer,
      createSocket: () => socket as unknown as WebSocket,
    }));

    const message = (payload: Record<string, unknown>) => act(() => {
      socket.onmessage?.({ data: JSON.stringify(payload) } as MessageEvent<string>);
    });
    act(() => {
      socket.readyState = PredictionSocket.OPEN;
      socket.onopen?.();
    });
    message({ type: "session_ready", accepted_through: 0 });
    message({ type: "echo_state", echo_known: true, echo_enabled: true, in_alt_buffer: false, cursor_at_line_end: true });

    const onData = terminal.onData as ReturnType<typeof vi.fn>;
    act(() => onData.mock.calls[0]?.[0]?.("a"));
    expect(predictionContainer.querySelector("[data-testid='terminal-prediction-overlay']")?.textContent).toBe("a");

    message({ type: "echo_state", echo_known: true, echo_enabled: true, in_alt_buffer: true, cursor_at_line_end: false });
    expect(predictionContainer.querySelector("[data-testid='terminal-prediction-overlay']")?.textContent).toBe("");

    message({ type: "echo_state", echo_known: true, echo_enabled: false, in_alt_buffer: false, cursor_at_line_end: true });
    act(() => onData.mock.calls[0]?.[0]?.("c"));
    expect(predictionContainer.querySelector("[data-testid='terminal-prediction-overlay']")?.textContent).toBe("");

    message({ type: "echo_state", echo_known: false, echo_enabled: false, in_alt_buffer: false, cursor_at_line_end: false });
    act(() => onData.mock.calls[0]?.[0]?.("b"));
    expect(predictionContainer.querySelector("[data-testid='terminal-prediction-overlay']")?.textContent).toBe("");
    expect(result.current.getPendingInputSnapshot()).toHaveLength(0);
    unmount();
  });
});
