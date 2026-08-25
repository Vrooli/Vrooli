import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useStdinAck, type InputSettledListener } from "./useStdinAck";
import type { TerminalMessage } from "../../types/terminal";

describe("useStdinAck", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  function setup() {
    const frames: TerminalMessage[] = [];
    let ready = true;
    let generation = 1;
    const rendered = renderHook(() => useStdinAck({
      sendFrame: (frame) => {
        frames.push(frame);
        return true;
      },
      isSessionReady: () => ready,
      currentGen: () => generation,
    }));
    return { result: rendered.result, frames, setReady: (value: boolean) => { ready = value; } };
  }

  it("assigns connection-local sequences and preserves each intent in FIFO order", () => {
    const { result, frames } = setup();

    let first: ReturnType<typeof result.current.send>;
    let second: ReturnType<typeof result.current.send>;
    act(() => {
      first = result.current.send("a", "typing");
      second = result.current.send("b", "bulk_text");
    });

    expect(first!).toEqual({ sent: true, seq: 1 });
    expect(second!).toEqual({ sent: true, seq: 2 });
    expect(frames).toEqual([
      expect.objectContaining({ type: "stdin", data: "a", seq: 1, intent: "typing" }),
      expect.objectContaining({ type: "stdin", data: "b", seq: 2, intent: "bulk_text" }),
    ]);
  });

  it("settles only the sequence awaited by a caller", () => {
    const { result } = setup();
    const settled: Array<[boolean, string | undefined]> = [];
    let seq1 = 0;
    let seq2 = 0;

    act(() => {
      seq1 = result.current.send("one", "typing").sent ? 1 : 0;
      seq2 = result.current.send("two", "named_key").sent ? 2 : 0;
      result.current.awaitSeq(seq2, (ok, reason) => settled.push([ok, reason]));
    });

    act(() => {
      expect(result.current.acceptAck(seq1, true)).toBe(true);
    });
    expect(settled).toEqual([]);

    act(() => {
      expect(result.current.acceptAck(seq2, false, "not_ready")).toBe(true);
    });
    expect(settled).toEqual([[false, "not_ready"]]);
    expect(result.current.acceptAck(999, true)).toBe(false);
  });

  it("requeues timed-out input and reports the timeout reason", () => {
    vi.useFakeTimers();
    const { result } = setup();
    const events: Array<[number, boolean, string | undefined]> = [];
    const listener: InputSettledListener = (seq, ok, reason) => events.push([seq, ok, reason]);
    let seq = 0;

    act(() => {
      result.current.subscribeInputSettled(listener);
      const sent = result.current.send("retry me", "bulk_text");
      if (!sent.sent) throw new Error("send unexpectedly failed");
      if (sent.seq === undefined) throw new Error("send returned no sequence");
      seq = sent.seq;
      vi.advanceTimersByTime(2000);
    });

    expect(events).toEqual([[seq, false, "ack-timeout"]]);
    expect(result.current.getPendingSnapshot()).toEqual([
      expect.objectContaining({ data: "retry me", intent: "bulk_text" }),
    ]);
  });

  it("resets sequence numbers for a new connection and caps queued input", () => {
    const { result, frames, setReady } = setup();
    act(() => {
      result.current.send("old", "typing");
      result.current.resetForNewConnection(2);
    });
    expect(result.current.send("new", "typing")).toEqual({ sent: true, seq: 1 });
    expect(frames.at(-1)).toEqual(expect.objectContaining({ data: "new", seq: 1 }));

    act(() => {
      setReady(false);
      for (let i = 0; i < 70; i += 1) result.current.enqueue(`queued-${i}`, "bulk_text");
    });
    const pending = result.current.getPendingSnapshot();
    expect(pending).toHaveLength(64);
    expect(pending[0]?.data).toBe("queued-6");
    expect(pending.at(-1)?.data).toBe("queued-69");
  });

  it("flushes queued input only when ready and preserves the close generation barrier", () => {
    const { result, frames, setReady } = setup();
    let generation = 1;
    const settled: Array<[number, boolean, string | undefined]> = [];
    act(() => {
      result.current.subscribeInputSettled((seq, ok, reason) => settled.push([seq, ok, reason]));
      setReady(false);
      result.current.enqueue("queued", "bulk_text");
      result.current.flush();
    });
    expect(frames).toHaveLength(0);
    setReady(true);
    act(() => result.current.flush());
    expect(frames.at(-1)).toEqual(expect.objectContaining({ data: "queued", intent: "bulk_text" }));
    act(() => result.current.handleClose());
    expect(settled.at(-1)?.[2]).toBe("connection-closed");
    expect(result.current.getPendingSnapshot()).toEqual([expect.objectContaining({ data: "queued" })]);

    generation = 2;
    act(() => {
      result.current.resetForNewConnection(generation);
      result.current.flush();
    });
    expect(result.current.getPendingSnapshot()).toHaveLength(0);
  });

  it("notifies pending subscribers, supports unsubscribe, and disposes timers", () => {
    vi.useFakeTimers();
    const { result } = setup();
    const pending = vi.fn();
    const settled = vi.fn();
    let removePending = () => {};
    act(() => {
      removePending = result.current.subscribePendingInput(pending);
      result.current.subscribeInputSettled(settled);
      result.current.enqueue("a", "typing");
    });
    expect(pending).toHaveBeenCalled();
    act(() => removePending());
    act(() => result.current.enqueue("b", "typing"));
    expect(pending).toHaveBeenCalledTimes(1);
    act(() => {
      result.current.send("timer", "typing");
      result.current.dispose();
      vi.advanceTimersByTime(2500);
    });
    expect(settled).not.toHaveBeenCalled();
  });
});
