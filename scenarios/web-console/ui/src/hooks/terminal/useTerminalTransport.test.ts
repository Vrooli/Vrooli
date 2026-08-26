import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { isCleanWsClose, useTerminalTransport } from "./useTerminalTransport";

class FakeWebSocket {
  static readonly OPEN = 1;
  static readonly CLOSED = 3;
  readyState = 0;
  bufferedAmount = 0;
  sent: string[] = [];
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent<string>) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;

  closeWithFailure(): void {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({ code: 1006, wasClean: false } as CloseEvent);
  }

  close(): void {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({ code: 1000, wasClean: true } as CloseEvent);
  }

  send(data: string): void { this.sent.push(data); }
}

describe("useTerminalTransport", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("classifies only normal and going-away closes as clean", () => {
    expect(isCleanWsClose(1000)).toBe(true);
    expect(isCleanWsClose(1001)).toBe(true);
    expect(isCleanWsClose(1006)).toBe(false);
  });

  it("keeps retrying after a long server restart", async () => {
    const sockets: FakeWebSocket[] = [];
    renderHook(() => useTerminalTransport({
      url: "ws://example.test/terminal",
      createSocket: () => {
        const socket = new FakeWebSocket();
        sockets.push(socket);
        return socket as unknown as WebSocket;
      },
    }));

    // A lifecycle restart can outlast the first five exponential retries.
    // The connection must remain recoverable rather than permanently leaving
    // the mounted terminal at "[Disconnected]".
    for (let attempt = 0; attempt < 6; attempt += 1) {
      act(() => sockets.at(-1)?.closeWithFailure());
      await act(async () => { await vi.advanceTimersByTimeAsync(5_000); });
    }

    expect(sockets).toHaveLength(7);
  });

  it("keeps reliable input on the open socket above the control-lane high-water mark", () => {
    const sockets: FakeWebSocket[] = [];
    const { result } = renderHook(() => useTerminalTransport({
      url: "ws://example.test/terminal",
      createSocket: () => {
        const socket = new FakeWebSocket();
        sockets.push(socket);
        return socket as unknown as WebSocket;
      },
    }));

    act(() => {
      const socket = sockets[0];
      if (!socket) throw new Error("socket was not created");
      socket.readyState = FakeWebSocket.OPEN;
      socket.bufferedAmount = 2 * 1024 * 1024;
      socket.onopen?.(new Event("open"));
    });

    expect(result.current.sendJson({ type: "control", data: "mouse" })).toBe(false);
    expect(result.current.sendReliableJson({ type: "stdin", data: "input", offset: 0 })).toBe(true);
    expect(sockets[0]?.sent).toHaveLength(1);
  });

  it("uses the browser WebSocket when no factory override is supplied", () => {
    const sockets: FakeWebSocket[] = [];
    class BrowserWebSocket extends FakeWebSocket {
      constructor() {
        super();
        sockets.push(this);
      }
    }
    vi.stubGlobal("WebSocket", BrowserWebSocket);

    const { unmount } = renderHook(() => useTerminalTransport({
      url: "ws://example.test/default",
    }));

    expect(sockets).toHaveLength(1);
    unmount();
    vi.unstubAllGlobals();
  });

  it("waits for visibility before reconnecting a hidden page", () => {
    const sockets: FakeWebSocket[] = [];
    const visibility = Object.getOwnPropertyDescriptor(document, "visibilityState");
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "hidden" });
    try {
      renderHook(() => useTerminalTransport({
        url: "ws://example.test/visibility",
        createSocket: () => {
          const socket = new FakeWebSocket();
          sockets.push(socket);
          return socket as unknown as WebSocket;
        },
      }));

      act(() => sockets[0]?.closeWithFailure());
      vi.advanceTimersByTime(5_000);
      expect(sockets).toHaveLength(1);

      Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
      act(() => document.dispatchEvent(new Event("visibilitychange")));
      expect(sockets).toHaveLength(2);
    } finally {
      if (visibility) Object.defineProperty(document, "visibilityState", visibility);
    }
  });

  it("delivers valid messages, ignores malformed frames, and isolates subscriber failures", () => {
    const sockets: FakeWebSocket[] = [];
    const message = vi.fn();
    const state = vi.fn();
    const { result } = renderHook(() => useTerminalTransport({
      url: "ws://example.test/messages",
      createSocket: () => {
        const socket = new FakeWebSocket();
        sockets.push(socket);
        return socket as unknown as WebSocket;
      },
    }));
    const unsubscribeMessage = result.current.subscribe(() => { throw new Error("subscriber failure"); });
    result.current.subscribe(message);
    const unsubscribeState = result.current.onStateChange(() => { throw new Error("state failure"); });
    result.current.onStateChange(state);

    act(() => {
      const socket = sockets[0];
      if (!socket) throw new Error("socket was not created");
      socket.readyState = FakeWebSocket.OPEN;
      socket.onopen?.(new Event("open"));
      socket.onmessage?.({ data: "not-json" } as MessageEvent<string>);
      socket.onmessage?.({ data: JSON.stringify({ type: "stdout", data: "ok" }) } as MessageEvent<string>);
    });
    expect(message).toHaveBeenCalledWith({ type: "stdout", data: "ok" });
    expect(state).toHaveBeenCalledWith("open");
    expect(result.current.sendJson({ type: "control", data: "ok" })).toBe(true);
    unsubscribeMessage();
    unsubscribeState();
  });
});
