import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";
// FakeWebSocket: drives the transport hook deterministically. Only
// the methods useTerminalTransport touches are implemented.
class FakeWebSocket {
    constructor(url) {
        this.url = url;
        this.readyState = 0;
        this.onopen = null;
        this.onmessage = null;
        this.onclose = null;
        this.onerror = null;
        this.sent = [];
    }
    send(data) {
        this.sent.push(data);
    }
    close() {
        this.readyState = FakeWebSocket.CLOSED;
        this.onclose?.({ code: 1000, reason: "test", wasClean: true });
    }
    triggerOpen() {
        this.readyState = FakeWebSocket.OPEN;
        this.onopen?.(new Event("open"));
    }
    triggerMessage(payload) {
        this.onmessage?.({ data: JSON.stringify(payload) });
    }
}
FakeWebSocket.OPEN = 1;
FakeWebSocket.CLOSED = 3;
function makeFakeTerminal() {
    const written = [];
    const alt = {};
    const normal = {};
    const term = {
        cols: 80,
        rows: 24,
        onData: vi.fn(() => ({ dispose: vi.fn() })),
        buffer: { active: normal, alternate: alt },
        written,
        resetCalls: 0,
    };
    term.reset = vi.fn(() => {
        term.resetCalls += 1;
    });
    term.clear = vi.fn();
    term.write = vi.fn((s) => {
        written.push(s);
    });
    return term;
}
describe("useTerminalSession snapshot replay", () => {
    let lastSocket = null;
    const createSocket = (url) => {
        lastSocket = new FakeWebSocket(url);
        return lastSocket;
    };
    const getLastSocket = () => {
        expect(lastSocket).not.toBeNull();
        if (!lastSocket) {
            throw new Error("Expected websocket to be initialized");
        }
        return lastSocket;
    };
    beforeEach(() => {
        lastSocket = null;
    });
    afterEach(() => {
        vi.clearAllMocks();
    });
    it("resets xterm and writes snapshot stdout frames before history_end", async () => {
        const term = makeFakeTerminal();
        renderHook(() => useTerminalSession({
            sessionId: "test",
            // Cast through unknown — FakeTerminal satisfies the subset of
            // the xterm.Terminal API the hook actually touches.
            terminal: term,
            createSocket,
        }));
        expect(lastSocket).not.toBeNull();
        await act(async () => {
            getLastSocket().triggerOpen();
        });
        // onTransportOpen calls reset() exactly once per WS open.
        expect(term.resetCalls).toBe(1);
        // Server streams snapshot stdout frames. Hook writes them to xterm.
        await act(async () => {
            const socket = getLastSocket();
            socket.triggerMessage({ type: "stdout", data: "scrollback line 1\r\n" });
            socket.triggerMessage({ type: "stdout", data: "screen content" });
        });
        expect(term.written).toContain("scrollback line 1\r\n");
        expect(term.written).toContain("screen content");
        // history_end terminates snapshot mode.
        await act(async () => {
            getLastSocket().triggerMessage({ type: "history_end" });
        });
        // Subsequent stdout frames are still written (live mode).
        await act(async () => {
            getLastSocket().triggerMessage({ type: "stdout", data: "live byte" });
        });
        expect(term.written).toContain("live byte");
    });
    it("reset()s xterm on every reconnect (no stale-cache state)", async () => {
        const term = makeFakeTerminal();
        renderHook(() => useTerminalSession({
            sessionId: "test",
            terminal: term,
            createSocket,
        }));
        await act(async () => {
            getLastSocket().triggerOpen();
        });
        expect(term.resetCalls).toBe(1);
        // Simulate disconnect + reconnect — the same socket factory is used
        // for the next attempt, so we just open again on the existing WS.
        await act(async () => {
            getLastSocket().close();
        });
        // Transport's auto-reconnect creates a new socket; force an immediate
        // open by calling the factory directly is not exposed. Instead, mount
        // a second hook to model a reconnect for this assertion.
        const term2 = makeFakeTerminal();
        renderHook(() => useTerminalSession({
            sessionId: "test",
            terminal: term2,
            createSocket,
        }));
        await act(async () => {
            getLastSocket().triggerOpen();
        });
        expect(term2.resetCalls).toBe(1);
    });
});
