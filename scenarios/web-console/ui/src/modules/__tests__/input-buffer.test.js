// @ts-check

import { describe, expect, it, beforeEach, vi } from "vitest";

if (typeof globalThis.WebSocket === "undefined") {
  class TestWebSocket {
    static CONNECTING = 0;
    static OPEN = 1;
    static CLOSING = 2;
    static CLOSED = 3;

    constructor() {
      this.readyState = TestWebSocket.CLOSED;
    }

    addEventListener() {}
    removeEventListener() {}
    send() {}
    close() {}
  }

  globalThis.WebSocket = /** @type {any} */ (TestWebSocket);
}

import {
  queueInput,
  flushPendingWrites,
  transmitInput,
  ensureInputSequence,
} from "../input-buffer.js";

function createTab() {
  return /** @type {import("../types.d.ts").TerminalTab} */ ({
    id: "tab-1",
    label: "tab-1",
    defaultLabel: "tab-1",
    colorId: "sky",
    socket: null,
    pendingWrites: [],
    telemetry: { queued: 0, batches: 0, sent: 0, typed: 0, lastBatchSize: 0 },
    transcript: [],
    transcriptByteSize: 0,
    transcriptHydrated: false,
    transcriptHydrating: false,
    suppressed: {},
    errorMessage: "",
    events: [],
    term: /** @type {any} */ ({
      reset: vi.fn(),
      write: vi.fn(),
      focus: vi.fn(),
      scrollToBottom: vi.fn(),
      refresh: vi.fn(),
      cols: 80,
      rows: 24,
    }),
    fitAddon: /** @type {any} */ ({ fit: vi.fn() }),
    container: /** @type {any} */ ({}),
    phase: "idle",
    socketState: "disconnected",
    wasDetached: false,
    inputSeq: 0,
    inputBatch: null,
    inputBatchScheduled: false,
    replayPending: false,
    replayComplete: true,
    lastReplayCount: 0,
    lastReplayTruncated: false,
    hasReceivedLiveOutput: false,
    hasEverConnected: false,
    reconnecting: false,
    localEchoBuffer: [],
    lastSentSize: { cols: 0, rows: 0 },
    session: null,
    heartbeatInterval: null,
    domItem: null,
    domButton: null,
    domClose: null,
    domLabel: null,
    domStatus: null,
  });
}

describe("input buffer", () => {
  /** @type {import("../types.d.ts").TerminalTab} */
  let tab;

  beforeEach(() => {
    tab = createTab();
  });

  it("queues input and tracks pending writes with sequencing metadata", () => {
    queueInput(tab, 'echo "hi"', { appendNewline: true });
    queueInput(tab, "ls", {});

    expect(tab.pendingWrites).toHaveLength(2);
    expect(tab.telemetry.queued).toBe(2);
    expect(tab.pendingWrites[0].meta.batchSize).toBe('echo "hi"'.length);
    const firstSeq = Number(tab.pendingWrites[0].meta.seq);
    const secondSeq = Number(tab.pendingWrites[1].meta.seq);
    expect(Number.isFinite(firstSeq)).toBe(true);
    expect(secondSeq).toBe(firstSeq + 1);
  });

  it("flushes pending writes through an open socket and records transcript entries", () => {
    const send = vi.fn();
    tab.socket = /** @type {WebSocket} */ (
      /** @type {unknown} */ ({
        readyState: WebSocket.OPEN,
        send,
        close: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(() => true),
        url: "",
        protocol: "",
        extensions: "",
        bufferedAmount: 0,
        binaryType: "arraybuffer",
        onclose: null,
        onerror: null,
        onmessage: null,
        onopen: null,
      })
    );

    queueInput(tab, "echo test", { appendNewline: true });
    flushPendingWrites(tab);

    expect(tab.pendingWrites).toHaveLength(0);
    expect(send).toHaveBeenCalledTimes(1);
    const payload = send.mock.calls[0][0];
    expect(payload).toBeInstanceOf(Uint8Array);
    expect(tab.telemetry.sent).toBe(1);
    expect(Array.isArray(tab.transcript)).toBe(true);
    expect(tab.transcript).toHaveLength(1);
  });

  it("refuses to transmit when socket is not open", () => {
    tab.socket = /** @type {WebSocket} */ (
      /** @type {unknown} */ ({
        readyState: WebSocket.CLOSING,
        send: vi.fn(),
        close: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(() => true),
        url: "",
        protocol: "",
        extensions: "",
        bufferedAmount: 0,
        binaryType: "arraybuffer",
        onclose: null,
        onerror: null,
        onmessage: null,
        onopen: null,
      })
    );
    const meta = { appendNewline: false };
    const result = transmitInput(tab, "noop", meta);
    expect(result).toBe(false);
    const socket = tab.socket;
    expect(socket).not.toBeNull();
    if (!socket) throw new Error("socket should be set for test");
    expect(socket.send).not.toHaveBeenCalled();
  });

  it("ensures input sequences increment even with explicit meta values", () => {
    const meta = { seq: 10 };
    const seq = ensureInputSequence(tab, meta);
    expect(seq).toBe(10);
    const next = ensureInputSequence(tab, {});
    expect(next).toBe(1);
  });
});
