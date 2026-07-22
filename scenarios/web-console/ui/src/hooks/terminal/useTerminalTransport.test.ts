import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useTerminalTransport } from "./useTerminalTransport";

class FakeWebSocket {
  static readonly OPEN = 1;
  static readonly CLOSED = 3;
  readyState = 0;
  bufferedAmount = 0;
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

  send(): void {}
}

describe("useTerminalTransport", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

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
});
