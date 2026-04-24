import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";
import { createFakeSocketPair, createMockTerminal } from "../test-utils";
import type { FakeWebSocket, MockTerminal } from "../test-utils";

vi.mock("../lib/api", () => ({
  buildSessionWsUrl: vi.fn((id: string) => `ws://test/sessions/${id}/ws`),
}));

/**
 * Regression test for Bug C — scrollback duplication after reconnect
 * with a stale sessionStorage cache.
 *
 * The root cause was that useTerminalSession's totalBytesRef was only
 * updated from `history_end` messages, not from live `stdout` frames.
 * When the browser saved the cache mid-session (on visibility change
 * or beforeunload), the saved `totalBytes` lagged behind the rendered
 * xterm state. On reconnect, the server's `history_offset=<stale>`
 * caused a delta replay that overlapped with the already-restored
 * serialized xterm content, duplicating the trailing bytes.
 *
 * The fix counts bytes on every stdout frame (both replay and live).
 * These tests lock in the new invariant directly against the hook's
 * public `totalBytesRef`.
 */
describe("useTerminalSession — live byte counting (Bug C regression)", () => {
  let fakeWs: FakeWebSocket;
  let createSocket: ReturnType<typeof createFakeSocketPair>["createSocket"];
  let terminal: MockTerminal;

  beforeEach(() => {
    vi.clearAllMocks();
    const pair = createFakeSocketPair();
    fakeWs = pair.fakeWs;
    createSocket = pair.createSocket;
    terminal = createMockTerminal();
  });

  it("advances totalBytesRef on every live stdout frame, not only history_end", () => {
    const { result } = renderHook(() =>
      useTerminalSession({
        sessionId: "sess-bugc-1",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    // History replay: three chunks totalling 20 bytes, then history_end.
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "aaaaaaa" })); // 7 bytes
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "bbbbb" })); //   5 bytes
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "cccccccc" })); // 8 bytes
    act(() =>
      fakeWs.triggerMessage({
        type: "history_end",
        total_bytes: 20,
        resumed: false,
      }),
    );

    expect(result.current.totalBytesRef.current).toBe(20);

    // Now simulate live output arriving after history_end. Without the
    // fix, totalBytesRef would stay at 20 while xterm rendered new
    // bytes, yielding a stale cache.
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "hello" })); // 5 bytes
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "!" })); //      1 byte

    expect(result.current.totalBytesRef.current).toBe(26);
  });

  it("counts UTF-8 multi-byte payloads by byte length, not code-unit length", () => {
    const { result } = renderHook(() =>
      useTerminalSession({
        sessionId: "sess-bugc-2",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    act(() =>
      fakeWs.triggerMessage({
        type: "history_end",
        total_bytes: 0,
        resumed: false,
      }),
    );

    // 'é' is 2 UTF-8 bytes; '漢' is 3; '🚀' is 4. String.length would
    // report 1, 1, and 2 respectively — all wrong. The hook must use
    // TextEncoder byte length.
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "é" }));
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "漢" }));
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "🚀" }));

    expect(result.current.totalBytesRef.current).toBe(2 + 3 + 4);
  });

  it("resets totalBytesRef to historyOffset on WS (re)open", () => {
    const { result } = renderHook(() =>
      useTerminalSession({
        sessionId: "sess-bugc-3",
        terminal: terminal as never,
        createSocket,
        historyOffset: 500,
        hasCachedState: true,
      }),
    );

    act(() => fakeWs.triggerOpen());

    // Before any server message, the counter snaps to the resume
    // offset. A subsequent delta replay will increment it with each
    // stdout frame, converging on the server's authoritative value
    // when history_end is received.
    expect(result.current.totalBytesRef.current).toBe(500);

    // Simulate a 30-byte delta replay.
    act(() =>
      fakeWs.triggerMessage({ type: "stdout", data: "x".repeat(30) }),
    );
    act(() =>
      fakeWs.triggerMessage({
        type: "history_end",
        total_bytes: 530,
        resumed: true,
      }),
    );

    expect(result.current.totalBytesRef.current).toBe(530);
  });

  it("history_end reconciliation snaps the counter to server's authoritative value", () => {
    const { result } = renderHook(() =>
      useTerminalSession({
        sessionId: "sess-bugc-4",
        terminal: terminal as never,
        createSocket,
      }),
    );

    act(() => fakeWs.triggerOpen());
    // If server and client somehow diverge during replay (should not
    // happen in practice), history_end wins — this keeps future
    // resume offsets aligned with the server's ring.
    act(() => fakeWs.triggerMessage({ type: "stdout", data: "abcd" })); // 4 bytes
    act(() =>
      fakeWs.triggerMessage({
        type: "history_end",
        total_bytes: 999,
        resumed: false,
      }),
    );
    expect(result.current.totalBytesRef.current).toBe(999);
  });
});
