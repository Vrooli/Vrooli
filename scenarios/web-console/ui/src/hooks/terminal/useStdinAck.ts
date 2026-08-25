import { useCallback, useRef } from "react";
import type {
  QueuedReason,
  RawSendResult,
  InputIntent,
} from "../../components/terminal/inputGate";
import type { StdinAckReason, TerminalMessage } from "../../types/terminal";

const MAX_PENDING_INPUT_MESSAGES = 64;

/**
 * Ack timeout for stdin messages. If the server does not echo
 * stdin_ack within this window, the client treats the send as
 * failed, re-enqueues the payload, and fires inputSettled(false).
 * Override via build env VITE_WC_ACK_TIMEOUT_MS.
 */
const ACK_TIMEOUT_MS = (() => {
  const env = (import.meta as unknown as { env?: Record<string, string | undefined> }).env;
  const raw = env?.VITE_WC_ACK_TIMEOUT_MS;
  const n = raw ? Number(raw) : NaN;
  return Number.isFinite(n) && n > 0 ? n : 2000;
})();

/** Snapshot entry for a queued stdin payload awaiting flush. */
export interface PendingInputEntry {
  data: string;
  /** ms epoch when the payload was first queued. */
  addedAt: number;
  /** Intent preserved so drain re-sends via the right server path. */
  intent: Exclude<InputIntent, "control">;
}

interface PendingAckEntry {
  data: string;
  addedAt: number;
  timer: ReturnType<typeof setTimeout>;
  /** Generation of the WS connection that sent this payload. */
  gen: number;
  /** Preserved for re-enqueue on timeout or close. */
  intent: Exclude<InputIntent, "control">;
}

/**
 * Why an input payload settled unsuccessfully.
 *
 * Server-sent codes come from StdinAckReason (the wire contract in
 * types/terminal.ts, mirroring api/terminal_ws.go). The two extra codes
 * are client-side: they describe failures the server never got to
 * answer, so they have no wire representation.
 */
export type InputFailureReason =
  | StdinAckReason
  | "ack-timeout"
  | "connection-closed";

/** Callback signature for input-settlement subscribers. */
export type InputSettledListener = (
  seq: number,
  ok: boolean,
  reason?: InputFailureReason,
) => void;

/** Callback for one specific stdin sequence. */
export type InputSettlementCallback = (ok: boolean, reason?: InputFailureReason) => void;

export interface StdinAckHandle {
  /** Try to send data immediately. Does not enqueue on failure. */
  send: (data: string, intent: Exclude<InputIntent, "control">) => RawSendResult;
  /** Queue data for later flush. */
  enqueue: (data: string, intent: Exclude<InputIntent, "control">) => void;
  /** Attempt to drain the queue; sends until send() returns false. */
  flush: () => void;
  /**
   * Reset all per-connection state except the pending queue. Called on
   * WS (re)open. The queue survives so queued payloads are re-sent.
   */
  resetForNewConnection: (newGen: number) => void;
  /**
   * Handle a connection close event. Unacked payloads whose generation
   * matches the current one are re-enqueued (they might have been
   * dropped by the network); payloads from older generations are
   * discarded (they have a committed write-barrier outcome already).
   */
  handleClose: () => void;
  /**
   * Accept an incoming stdin_ack. Returns true if the ack matched a
   * pending timer.
   */
  acceptAck: (seq: number, ok: boolean, reason?: StdinAckReason) => boolean;
  /**
   * Subscribe to settlement events (ack or timeout). Returns unsubscribe.
   */
  subscribeInputSettled: (cb: InputSettledListener) => () => void;
  /** Wait for the settlement of exactly one sequence, ignoring all others. */
  awaitSeq: (seq: number, cb: InputSettlementCallback) => () => void;
  /**
   * Subscribe to queue-changed notifications (pill visibility). Returns
   * unsubscribe.
   */
  subscribePendingInput: (cb: () => void) => () => void;
  /** Return a shallow copy of the current pending queue. */
  getPendingSnapshot: () => readonly PendingInputEntry[];
  /** Dispose: clear timers and subscribers. */
  dispose: () => void;
}

export interface UseStdinAckOptions {
  /**
   * Send a stdin frame to the transport. Returns true iff the browser
   * WebSocket accepted it. Called only when the session is ready.
   */
  sendFrame: (msg: TerminalMessage) => boolean;
  /** True when the server has emitted session_ready on the current gen. */
  isSessionReady: () => boolean;
  /** Current transport generation. */
  currentGen: () => number;
}

/**
 * useStdinAck owns the stdin seq/ack protocol and the pending-input
 * queue. The wsGen write barrier lives here: when a WS connection
 * closes, only payloads tagged with the *current* generation are
 * re-enqueued. If a payload was tagged with an older generation, its
 * outcome is committed (either it arrived and the server will respond
 * after reconnect, or it's genuinely lost but the server has likely
 * seen the write; double-sending is worse than losing a single frame).
 */
export function useStdinAck({
  sendFrame,
  isSessionReady,
  currentGen,
}: UseStdinAckOptions): StdinAckHandle {
  const nextSeqRef = useRef(1);
  const pendingAcksRef = useRef<Map<number, PendingAckEntry>>(new Map());
  const pendingInputRef = useRef<PendingInputEntry[]>([]);
  const inputSettledSubsRef = useRef<Set<InputSettledListener>>(new Set());
  const pendingInputSubsRef = useRef<Set<() => void>>(new Set());

  const notifyPending = useCallback(() => {
    for (const cb of pendingInputSubsRef.current) {
      try {
        cb();
      } catch (err) {
        console.warn("useStdinAck: pending subscriber threw", err);
      }
    }
  }, []);

  const notifySettled = useCallback((seq: number, ok: boolean, reason?: InputFailureReason) => {
    for (const cb of inputSettledSubsRef.current) {
      try {
        cb(seq, ok, reason);
      } catch (err) {
        console.warn("useStdinAck: settled subscriber threw", err);
      }
    }
  }, []);

  const enqueue = useCallback((data: string, intent: Exclude<InputIntent, "control">) => {
    if (!data) return;
    pendingInputRef.current.push({ data, intent, addedAt: Date.now() });
    if (pendingInputRef.current.length > MAX_PENDING_INPUT_MESSAGES) {
      pendingInputRef.current.splice(
        0,
        pendingInputRef.current.length - MAX_PENDING_INPUT_MESSAGES,
      );
    }
    notifyPending();
  }, [notifyPending]);

  const registerAckTimer = useCallback(
    (seq: number, data: string, gen: number, intent: Exclude<InputIntent, "control">) => {
      const timer = setTimeout(() => {
        const entry = pendingAcksRef.current.get(seq);
        if (!entry) return;
        pendingAcksRef.current.delete(seq);
        console.warn(
          `useStdinAck: input_ack_timeout seq=${seq} len=${data.length} gen=${gen} intent=${intent}`,
        );
        enqueue(entry.data, entry.intent);
        notifySettled(seq, false, "ack-timeout");
      }, ACK_TIMEOUT_MS);
      pendingAcksRef.current.set(seq, { data, addedAt: Date.now(), timer, gen, intent });
    },
    [enqueue, notifySettled],
  );

  const send = useCallback(
    (data: string, intent: Exclude<InputIntent, "control">): RawSendResult => {
      if (!data) return { sent: false, reason: "not-ready" satisfies QueuedReason };
      if (!isSessionReady()) {
        return { sent: false, reason: "not-ready" };
      }
      const seq = nextSeqRef.current;
      const gen = currentGen();
      const ok = sendFrame({ type: "stdin", data, seq, intent } satisfies TerminalMessage);
      if (!ok) return { sent: false, reason: "ws-closed" };
      nextSeqRef.current = seq + 1;
      registerAckTimer(seq, data, gen, intent);
      return { sent: true, seq };
    },
    [isSessionReady, currentGen, sendFrame, registerAckTimer],
  );

  const flush = useCallback(() => {
    if (!isSessionReady()) return;
    let changed = false;
    while (pendingInputRef.current.length > 0) {
      const next = pendingInputRef.current[0];
      if (!next) {
        pendingInputRef.current.shift();
        changed = true;
        continue;
      }
      const res = send(next.data, next.intent);
      if (!res.sent) break;
      pendingInputRef.current.shift();
      changed = true;
    }
    if (changed) notifyPending();
  }, [isSessionReady, send, notifyPending]);

  const resetForNewConnection = useCallback((_newGen: number) => {
    // Seqs are per-connection on the server; reset for the new one.
    nextSeqRef.current = 1;
    // Pending-ack timers from the prior connection are moot: the old
    // server will never ack them. handleClose() already re-enqueued
    // matching-generation payloads; anything left is spurious.
    for (const entry of pendingAcksRef.current.values()) {
      clearTimeout(entry.timer);
    }
    pendingAcksRef.current.clear();
  }, []);

  const handleClose = useCallback(() => {
    const gen = currentGen();
    const entries = Array.from(pendingAcksRef.current.entries());
    pendingAcksRef.current.clear();
    let rePushed = false;
    for (const [seq, entry] of entries) {
      clearTimeout(entry.timer);
      notifySettled(seq, false, "connection-closed");
      // wsGen write barrier: only re-enqueue payloads sent on the
      // current generation. Payloads from prior generations were
      // already retried (or deliberately dropped) during the previous
      // close cycle; double-sending them creates duplicate shell
      // commands on flaky networks.
      if (entry.gen === gen) {
        pendingInputRef.current.push({
          data: entry.data,
          addedAt: entry.addedAt,
          intent: entry.intent,
        });
        rePushed = true;
      }
    }
    if (rePushed) notifyPending();
  }, [currentGen, notifySettled, notifyPending]);

  const acceptAck = useCallback((seq: number, ok: boolean, reason?: StdinAckReason): boolean => {
    const entry = pendingAcksRef.current.get(seq);
    if (!entry) return false;
    clearTimeout(entry.timer);
    pendingAcksRef.current.delete(seq);
    notifySettled(seq, ok, reason);
    return true;
  }, [notifySettled]);

  const subscribeInputSettled = useCallback((cb: InputSettledListener) => {
    inputSettledSubsRef.current.add(cb);
    return () => {
      inputSettledSubsRef.current.delete(cb);
    };
  }, []);

  const awaitSeq = useCallback((seq: number, cb: InputSettlementCallback) => {
    let active = true;
    const unsubscribe = subscribeInputSettled((ackSeq, ok, reason) => {
      if (!active || ackSeq !== seq) return;
      active = false;
      unsubscribe();
      cb(ok, reason);
    });
    return () => {
      active = false;
      unsubscribe();
    };
  }, [subscribeInputSettled]);

  const subscribePendingInput = useCallback((cb: () => void) => {
    pendingInputSubsRef.current.add(cb);
    return () => {
      pendingInputSubsRef.current.delete(cb);
    };
  }, []);

  const getPendingSnapshot = useCallback(
    (): readonly PendingInputEntry[] => pendingInputRef.current.slice(),
    [],
  );

  const dispose = useCallback(() => {
    for (const entry of pendingAcksRef.current.values()) {
      clearTimeout(entry.timer);
    }
    pendingAcksRef.current.clear();
    inputSettledSubsRef.current.clear();
    pendingInputSubsRef.current.clear();
  }, []);

  return {
    send,
    enqueue,
    flush,
    resetForNewConnection,
    handleClose,
    acceptAck,
    subscribeInputSettled,
    awaitSeq,
    subscribePendingInput,
    getPendingSnapshot,
    dispose,
  };
}
