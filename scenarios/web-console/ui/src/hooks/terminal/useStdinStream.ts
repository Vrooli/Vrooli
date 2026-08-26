import { useCallback, useRef } from "react";
import type { QueuedReason, RawSendResult, InputIntent } from "../../components/terminal/inputGate";
import type { StdinAckReason, TerminalMessage } from "../../types/terminal";

export interface PendingInputEntry {
  data: string;
  addedAt: number;
  intent: Exclude<InputIntent, "control">;
}

export type InputFailureReason = StdinAckReason | "unreconcilable";
export type InputSettledListener = (offset: number, ok: boolean, reason?: InputFailureReason) => void;
export type InputSettlementCallback = (ok: boolean, reason?: InputFailureReason) => void;

interface BufferedEntry {
  start: number;
  end: number;
  data: string;
  intent: Exclude<InputIntent, "control">;
}

export interface StdinStreamHandle {
  send: (data: string, intent: Exclude<InputIntent, "control">) => RawSendResult;
  enqueue: (data: string, intent: Exclude<InputIntent, "control">) => void;
  flush: () => void;
  resetForNewConnection: () => void;
  handleClose: () => void;
  reconcile: (acceptedThrough: number) => void;
  replay: () => void;
  acknowledge: (acceptedThrough: number, ok: boolean, reason?: InputFailureReason) => void;
  subscribeInputSettled: (cb: InputSettledListener) => () => void;
  awaitOffset: (offset: number, cb: InputSettlementCallback) => () => void;
  subscribePendingInput: (cb: () => void) => () => void;
  getPendingSnapshot: () => readonly PendingInputEntry[];
  getPendingInputCount: () => number;
  dispose: () => void;
}

export interface UseStdinStreamOptions {
  sendFrame: (msg: TerminalMessage) => boolean;
  isSessionReady: () => boolean;
}

/** Ordered, cumulative-offset stdin stream. No timer retransmits bytes. */
export function useStdinStream({ sendFrame, isSessionReady }: UseStdinStreamOptions): StdinStreamHandle {
  const writeHeadRef = useRef(0);
  const releasedThroughRef = useRef(0);
  const bufferedRef = useRef<BufferedEntry[]>([]);
  const queuedRef = useRef<PendingInputEntry[]>([]);
  const unreconcilableRef = useRef(false);
  const settledSubsRef = useRef<Set<InputSettledListener>>(new Set());
  const pendingSubsRef = useRef<Set<() => void>>(new Set());
  const waitersRef = useRef(new Map<number, Set<InputSettlementCallback>>());

  const notifyPending = useCallback(() => {
    pendingSubsRef.current.forEach((cb) => cb());
  }, []);

  const notifySettled = useCallback((offset: number, ok: boolean, reason?: InputFailureReason) => {
    settledSubsRef.current.forEach((cb) => cb(offset, ok, reason));
    for (const [target, callbacks] of waitersRef.current) {
      if (target > offset && ok) continue;
      waitersRef.current.delete(target);
      callbacks.forEach((cb) => cb(ok, reason));
    }
  }, []);

  const enqueue = useCallback((data: string, intent: Exclude<InputIntent, "control">) => {
    if (!data) return;
    queuedRef.current.push({ data, intent, addedAt: Date.now() });
    notifyPending();
  }, [notifyPending]);

  const send = useCallback((data: string, intent: Exclude<InputIntent, "control">): RawSendResult => {
    if (!data) return { sent: false, reason: "not-ready" satisfies QueuedReason };
    if (!isSessionReady() || unreconcilableRef.current) return { sent: false, reason: "not-ready" };
    const start = writeHeadRef.current;
    const end = start + new TextEncoder().encode(data).byteLength;
    if (!sendFrame({ type: "stdin", data, offset: start, intent })) {
      return { sent: false, reason: "ws-closed" };
    }
    bufferedRef.current.push({ start, end, data, intent });
    writeHeadRef.current = end;
    notifyPending();
    return { sent: true, offset: end };
  }, [isSessionReady, sendFrame, notifyPending]);

  const flush = useCallback(() => {
    if (!isSessionReady() || unreconcilableRef.current) return;
    while (queuedRef.current.length > 0) {
      const next = queuedRef.current[0];
      if (!next || !send(next.data, next.intent).sent) break;
      queuedRef.current.shift();
    }
    notifyPending();
  }, [isSessionReady, notifyPending, send]);

  const resetForNewConnection = useCallback(() => {
    sendFrame({ type: "hello", have_through: releasedThroughRef.current });
  }, [sendFrame]);

  const handleClose = useCallback(() => {
    // Sent bytes remain in bufferedRef. They are replayed only after the
    // server reports its accepted offset on the next connection.
  }, []);

  const reconcile = useCallback((acceptedThrough: number) => {
    if (acceptedThrough < releasedThroughRef.current || acceptedThrough > writeHeadRef.current) {
      unreconcilableRef.current = true;
      notifySettled(acceptedThrough, false, "unreconcilable");
      return;
    }
    const matching = bufferedRef.current.every((entry) => entry.end <= acceptedThrough || entry.start >= acceptedThrough);
    if (!matching) {
      unreconcilableRef.current = true;
      notifySettled(acceptedThrough, false, "unreconcilable");
      return;
    }
    releasedThroughRef.current = Math.max(releasedThroughRef.current, acceptedThrough);
    bufferedRef.current = bufferedRef.current.filter((entry) => entry.end > acceptedThrough);
    notifySettled(acceptedThrough, true);
  }, [notifySettled]);

  const replay = useCallback(() => {
    if (!isSessionReady() || unreconcilableRef.current) return;
    for (const entry of bufferedRef.current) {
      if (!sendFrame({ type: "stdin", data: entry.data, offset: entry.start, intent: entry.intent })) break;
    }
  }, [isSessionReady, sendFrame]);

  const acknowledge = useCallback((acceptedThrough: number, ok: boolean, reason?: InputFailureReason) => {
    if (!ok) {
      if (reason === "unreconcilable") unreconcilableRef.current = true;
      notifySettled(acceptedThrough, false, reason ?? "unreconcilable");
      return;
    }
    reconcile(acceptedThrough);
  }, [notifySettled, reconcile]);
  const subscribeInputSettled = useCallback((cb: InputSettledListener) => {
    settledSubsRef.current.add(cb);
    return () => settledSubsRef.current.delete(cb);
  }, []);
  const awaitOffset = useCallback((offset: number, cb: InputSettlementCallback) => {
    if (offset <= releasedThroughRef.current) {
      cb(true);
      return () => {};
    }
    const callbacks = waitersRef.current.get(offset) ?? new Set<InputSettlementCallback>();
    callbacks.add(cb);
    waitersRef.current.set(offset, callbacks);
    return () => {
      callbacks.delete(cb);
      if (callbacks.size === 0) waitersRef.current.delete(offset);
    };
  }, []);
  const subscribePendingInput = useCallback((cb: () => void) => {
    pendingSubsRef.current.add(cb);
    return () => pendingSubsRef.current.delete(cb);
  }, []);
  const getPendingSnapshot = useCallback(() => queuedRef.current.slice(), []);
  const getPendingInputCount = useCallback(() => bufferedRef.current.length, []);
  const dispose = useCallback(() => {
    queuedRef.current = [];
    bufferedRef.current = [];
    settledSubsRef.current.clear();
    pendingSubsRef.current.clear();
    waitersRef.current.clear();
  }, []);

  return { send, enqueue, flush, resetForNewConnection, handleClose, reconcile, replay, acknowledge, subscribeInputSettled, awaitOffset, subscribePendingInput, getPendingSnapshot, getPendingInputCount, dispose };
}
