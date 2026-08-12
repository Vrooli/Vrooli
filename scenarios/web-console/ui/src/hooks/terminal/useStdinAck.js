import { useCallback, useRef } from "react";
const MAX_PENDING_INPUT_MESSAGES = 64;
/**
 * Ack timeout for stdin messages. If the server does not echo
 * stdin_ack within this window, the client treats the send as
 * failed, re-enqueues the payload, and fires inputSettled(false).
 * Override via build env VITE_WC_ACK_TIMEOUT_MS.
 */
const ACK_TIMEOUT_MS = (() => {
    const env = import.meta.env;
    const raw = env?.VITE_WC_ACK_TIMEOUT_MS;
    const n = raw ? Number(raw) : NaN;
    return Number.isFinite(n) && n > 0 ? n : 2000;
})();
/**
 * useStdinAck owns the stdin seq/ack protocol and the pending-input
 * queue. The wsGen write barrier lives here: when a WS connection
 * closes, only payloads tagged with the *current* generation are
 * re-enqueued. If a payload was tagged with an older generation, its
 * outcome is committed (either it arrived and the server will respond
 * after reconnect, or it's genuinely lost but the server has likely
 * seen the write; double-sending is worse than losing a single frame).
 */
export function useStdinAck({ sendFrame, isSessionReady, currentGen, }) {
    const nextSeqRef = useRef(1);
    const pendingAcksRef = useRef(new Map());
    const pendingInputRef = useRef([]);
    const inputSettledSubsRef = useRef(new Set());
    const pendingInputSubsRef = useRef(new Set());
    const notifyPending = useCallback(() => {
        for (const cb of pendingInputSubsRef.current) {
            try {
                cb();
            }
            catch (err) {
                console.warn("useStdinAck: pending subscriber threw", err);
            }
        }
    }, []);
    const notifySettled = useCallback((seq, ok, reason) => {
        for (const cb of inputSettledSubsRef.current) {
            try {
                cb(seq, ok, reason);
            }
            catch (err) {
                console.warn("useStdinAck: settled subscriber threw", err);
            }
        }
    }, []);
    const enqueue = useCallback((data, kind) => {
        if (!data)
            return;
        pendingInputRef.current.push({ data, kind, addedAt: Date.now() });
        if (pendingInputRef.current.length > MAX_PENDING_INPUT_MESSAGES) {
            pendingInputRef.current.splice(0, pendingInputRef.current.length - MAX_PENDING_INPUT_MESSAGES);
        }
        notifyPending();
    }, [notifyPending]);
    const registerAckTimer = useCallback((seq, data, gen, kind) => {
        const timer = setTimeout(() => {
            const entry = pendingAcksRef.current.get(seq);
            if (!entry)
                return;
            pendingAcksRef.current.delete(seq);
            console.warn(`useStdinAck: input_ack_timeout seq=${seq} len=${data.length} gen=${gen} kind=${kind}`);
            enqueue(entry.data, entry.kind);
            notifySettled(seq, false, "ack-timeout");
        }, ACK_TIMEOUT_MS);
        pendingAcksRef.current.set(seq, { data, addedAt: Date.now(), timer, gen, kind });
    }, [enqueue, notifySettled]);
    const send = useCallback((data, kind) => {
        if (!data)
            return { sent: false, reason: "not-ready" };
        if (!isSessionReady()) {
            return { sent: false, reason: "not-ready" };
        }
        const seq = nextSeqRef.current;
        const gen = currentGen();
        const ok = sendFrame({ type: "stdin", data, seq, kind });
        if (!ok)
            return { sent: false, reason: "ws-closed" };
        nextSeqRef.current = seq + 1;
        registerAckTimer(seq, data, gen, kind);
        return { sent: true, seq };
    }, [isSessionReady, currentGen, sendFrame, registerAckTimer]);
    const flush = useCallback(() => {
        if (!isSessionReady())
            return;
        let changed = false;
        while (pendingInputRef.current.length > 0) {
            const next = pendingInputRef.current[0];
            if (!next) {
                pendingInputRef.current.shift();
                changed = true;
                continue;
            }
            const res = send(next.data, next.kind);
            if (!res.sent)
                break;
            pendingInputRef.current.shift();
            changed = true;
        }
        if (changed)
            notifyPending();
    }, [isSessionReady, send, notifyPending]);
    const resetForNewConnection = useCallback((_newGen) => {
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
                    kind: entry.kind,
                });
                rePushed = true;
            }
        }
        if (rePushed)
            notifyPending();
    }, [currentGen, notifySettled, notifyPending]);
    const acceptAck = useCallback((seq, ok, reason) => {
        const entry = pendingAcksRef.current.get(seq);
        if (!entry)
            return false;
        clearTimeout(entry.timer);
        pendingAcksRef.current.delete(seq);
        notifySettled(seq, ok, reason);
        return true;
    }, [notifySettled]);
    const subscribeInputSettled = useCallback((cb) => {
        inputSettledSubsRef.current.add(cb);
        return () => {
            inputSettledSubsRef.current.delete(cb);
        };
    }, []);
    const subscribePendingInput = useCallback((cb) => {
        pendingInputSubsRef.current.add(cb);
        return () => {
            pendingInputSubsRef.current.delete(cb);
        };
    }, []);
    const getPendingSnapshot = useCallback(() => pendingInputRef.current.slice(), []);
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
        subscribePendingInput,
        getPendingSnapshot,
        dispose,
    };
}
