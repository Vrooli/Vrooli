import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useStdinStream, type InputSettledListener } from "./useStdinStream";
import type { TerminalMessage } from "../../types/terminal";

describe("useStdinStream", () => {
  function setup() {
    const frames: TerminalMessage[] = [];
    let ready = true;
    const rendered = renderHook(() => useStdinStream({
      sendFrame: (frame) => {
        frames.push(frame);
        return true;
      },
      isSessionReady: () => ready,
    }));
    return { result: rendered.result, frames, setReady: (value: boolean) => { ready = value; } };
  }

  it("uses cumulative UTF-8 byte offsets and does not retransmit on a timer", () => {
    vi.useFakeTimers();
    const { result, frames } = setup();

    act(() => {
      expect(result.current.send("é", "typing")).toEqual({ sent: true, offset: 2 });
      expect(result.current.send("x", "named_key")).toEqual({ sent: true, offset: 3 });
      vi.advanceTimersByTime(5000);
    });

    expect(frames).toEqual([
      expect.objectContaining({ type: "stdin", data: "é", offset: 0 }),
      expect.objectContaining({ type: "stdin", data: "x", offset: 2 }),
    ]);
    vi.useRealTimers();
  });

  it("replays only the unaccepted suffix after reconnect reconciliation", () => {
    const { result, frames, setReady } = setup();
    act(() => {
      result.current.send("one", "typing");
      result.current.send("two", "bulk_text");
      setReady(false);
      result.current.handleClose();
      result.current.resetForNewConnection();
      setReady(true);
      result.current.reconcile(3);
      result.current.replay();
    });

    expect(frames).toEqual([
      expect.objectContaining({ data: "one", offset: 0 }),
      expect.objectContaining({ data: "two", offset: 3 }),
      expect.objectContaining({ type: "hello", have_through: 0 }),
      expect.objectContaining({ data: "two", offset: 3 }),
    ]);
  });

  it("reports a mid-entry server offset as unreconcilable", () => {
    const { result } = setup();
    const events: Array<[number, boolean, string | undefined]> = [];
    const listener: InputSettledListener = (offset, ok, reason) => events.push([offset, ok, reason]);

    act(() => {
      result.current.subscribeInputSettled(listener);
      result.current.send("é", "typing");
      result.current.reconcile(1);
    });

    expect(events).toEqual([[1, false, "unreconcilable"]]);
    expect(result.current.send("later", "typing")).toEqual({ sent: false, reason: "not-ready" });
  });

  it("rejects a server offset that is ahead of the local write head", () => {
    const { result } = setup();
    const events: Array<[number, boolean, string | undefined]> = [];

    act(() => {
      result.current.subscribeInputSettled((offset, ok, reason) => events.push([offset, ok, reason]));
      result.current.send("x", "typing");
      result.current.reconcile(2);
    });

    expect(events).toEqual([[2, false, "unreconcilable"]]);
    expect(result.current.send("later", "typing")).toEqual({ sent: false, reason: "not-ready" });
  });

  it("refuses a server offset below the already released prefix", () => {
    const { result } = setup();
    const events: Array<[number, boolean, string | undefined]> = [];

    act(() => {
      result.current.subscribeInputSettled((offset, ok, reason) => events.push([offset, ok, reason]));
      result.current.send("x", "typing");
      result.current.reconcile(1);
      result.current.reconcile(0);
    });

    expect(events).toEqual([[1, true, undefined], [0, false, "unreconcilable"]]);
    expect(result.current.send("later", "typing")).toEqual({ sent: false, reason: "not-ready" });
  });

  it("holds queued input until the session is ready", () => {
    const { result, frames, setReady } = setup();
    act(() => {
      setReady(false);
      result.current.enqueue("queued", "bulk_text");
      result.current.flush();
      setReady(true);
      result.current.flush();
    });

    expect(frames).toEqual([expect.objectContaining({ data: "queued", offset: 0, intent: "bulk_text" })]);
  });
});
