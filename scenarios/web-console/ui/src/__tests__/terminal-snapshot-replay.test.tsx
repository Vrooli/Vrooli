import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";

// FakeWebSocket: drives the transport hook deterministically. Only
// the methods useTerminalTransport touches are implemented.
class FakeWebSocket {
  static OPEN = 1;
  static CLOSED = 3;
  readyState = 0;
  onopen: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent<string>) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  sent: string[] = [];
  constructor(public url: string) {}
  send(data: string): void {
    this.sent.push(data);
  }
  close(): void {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({ code: 1000, reason: "test", wasClean: true } as CloseEvent);
  }
  triggerOpen(): void {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }
  triggerMessage(payload: object): void {
    this.onmessage?.({ data: JSON.stringify(payload) } as MessageEvent<string>);
  }
}

interface FakeTerminal {
  cols: number;
  rows: number;
  reset: () => void;
  write: (s: string) => void;
  buffer: { active: object; alternate: object };
  written: string[];
  resetCalls: number;
}

function makeFakeTerminal(): FakeTerminal {
  const written: string[] = [];
  const alt = {};
  const normal = {};
  const term = {
    cols: 80,
    rows: 24,
    onData: vi.fn(() => ({ dispose: vi.fn() })),
    buffer: { active: normal, alternate: alt },
    written,
    resetCalls: 0,
  } as unknown as FakeTerminal & { onData: () => { dispose: () => void } };
  (term as unknown as { reset: () => void }).reset = vi.fn(() => {
    term.resetCalls += 1;
  });
  (term as unknown as { clear: () => void }).clear = vi.fn();
  (term as unknown as { write: (s: string) => void }).write = vi.fn((s: string) => {
    written.push(s);
  });
  return term as unknown as FakeTerminal;
}

describe("useTerminalSession snapshot replay", () => {
  let lastSocket: FakeWebSocket | null = null;
  const createSocket = (url: string) => {
    lastSocket = new FakeWebSocket(url);
    return lastSocket as unknown as WebSocket;
  };

  beforeEach(() => {
    lastSocket = null;
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("resets xterm and writes snapshot stdout frames before history_end", async () => {
    const term = makeFakeTerminal();
    renderHook(() =>
      useTerminalSession({
        sessionId: "test",
        // Cast through unknown — FakeTerminal satisfies the subset of
        // the xterm.Terminal API the hook actually touches.
        terminal: term as unknown as Parameters<typeof useTerminalSession>[0]["terminal"],
        createSocket,
      }),
    );

    expect(lastSocket).not.toBeNull();
    await act(async () => {
      lastSocket!.triggerOpen();
    });

    // onTransportOpen calls reset() exactly once per WS open.
    expect(term.resetCalls).toBe(1);

    // Server streams snapshot stdout frames. Hook writes them to xterm.
    await act(async () => {
      lastSocket!.triggerMessage({ type: "stdout", data: "scrollback line 1\r\n" });
      lastSocket!.triggerMessage({ type: "stdout", data: "screen content" });
    });
    expect(term.written).toContain("scrollback line 1\r\n");
    expect(term.written).toContain("screen content");

    // history_end terminates snapshot mode.
    await act(async () => {
      lastSocket!.triggerMessage({ type: "history_end" });
    });

    // Subsequent stdout frames are still written (live mode).
    await act(async () => {
      lastSocket!.triggerMessage({ type: "stdout", data: "live byte" });
    });
    expect(term.written).toContain("live byte");
  });

  it("reset()s xterm on every reconnect (no stale-cache state)", async () => {
    const term = makeFakeTerminal();
    renderHook(() =>
      useTerminalSession({
        sessionId: "test",
        terminal: term as unknown as Parameters<typeof useTerminalSession>[0]["terminal"],
        createSocket,
      }),
    );

    await act(async () => {
      lastSocket!.triggerOpen();
    });
    expect(term.resetCalls).toBe(1);

    // Simulate disconnect + reconnect — the same socket factory is used
    // for the next attempt, so we just open again on the existing WS.
    await act(async () => {
      lastSocket!.close();
    });
    // Transport's auto-reconnect creates a new socket; force an immediate
    // open by calling the factory directly is not exposed. Instead, mount
    // a second hook to model a reconnect for this assertion.
    const term2 = makeFakeTerminal();
    renderHook(() =>
      useTerminalSession({
        sessionId: "test",
        terminal: term2 as unknown as Parameters<typeof useTerminalSession>[0]["terminal"],
        createSocket,
      }),
    );
    await act(async () => {
      lastSocket!.triggerOpen();
    });
    expect(term2.resetCalls).toBe(1);
  });
});
