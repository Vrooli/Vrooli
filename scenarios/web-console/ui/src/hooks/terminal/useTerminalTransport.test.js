import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useTerminalTransport } from "./useTerminalTransport";
class FakeWebSocket {
    constructor() {
        this.readyState = 0;
        this.bufferedAmount = 0;
        this.onopen = null;
        this.onmessage = null;
        this.onclose = null;
    }
    closeWithFailure() {
        this.readyState = FakeWebSocket.CLOSED;
        this.onclose?.({ code: 1006, wasClean: false });
    }
    close() {
        this.readyState = FakeWebSocket.CLOSED;
        this.onclose?.({ code: 1000, wasClean: true });
    }
    send() { }
}
FakeWebSocket.OPEN = 1;
FakeWebSocket.CLOSED = 3;
describe("useTerminalTransport", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());
    it("keeps retrying after a long server restart", async () => {
        const sockets = [];
        renderHook(() => useTerminalTransport({
            url: "ws://example.test/terminal",
            createSocket: () => {
                const socket = new FakeWebSocket();
                sockets.push(socket);
                return socket;
            },
        }));
        // A lifecycle restart can outlast the first five exponential retries.
        // The connection must remain recoverable rather than permanently leaving
        // the mounted terminal at "[Disconnected]".
        for (let attempt = 0; attempt < 6; attempt += 1) {
            act(() => sockets.at(-1)?.closeWithFailure());
            await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
        }
        expect(sockets).toHaveLength(7);
    });
});
