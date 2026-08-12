import { useCallback, useEffect, useRef } from "react";
// Module-level so the default has stable identity across renders; a
// default parameter would allocate a new function every call and churn
// the connect effect below.
const defaultSocketFactory = (u) => new WebSocket(u);
/**
 * WebSocket close codes 1000 (Normal) and 1001 (Going Away) indicate
 * an intentional, expected close — e.g. the user closed the tab or
 * the server shut down gracefully. Any other code signals an
 * unexpected disconnection (network failure, server crash, etc.).
 */
export function isCleanWsClose(code) {
    return code === 1000 || code === 1001;
}
const RECONNECT_BASE_DELAY_MS = 400;
const RECONNECT_MAX_DELAY_MS = 5000;
/** bufferedAmount high-water mark above which sends are refused. */
const WS_SEND_HIGH_WATER = 1 * 1024 * 1024;
/**
 * useTerminalTransport owns the WebSocket: connect, reconnect with
 * backoff, message delivery to subscribers, and a per-connection
 * generation counter that callers use for write-barrier decisions.
 *
 * This hook deliberately knows nothing about the message *protocol*.
 * It just marshals JSON frames in and out. Protocol semantics
 * (session_ready, stdin_ack, history_end, ...) live in
 * useTerminalSession.
 */
export function useTerminalTransport({ url, createSocket = defaultSocketFactory, onOpen, onClose, }) {
    const wsRef = useRef(null);
    const genRef = useRef(0);
    const stateRef = useRef("idle");
    const messageSubsRef = useRef(new Set());
    const stateSubsRef = useRef(new Set());
    const onOpenRef = useRef(onOpen);
    const onCloseRef = useRef(onClose);
    onOpenRef.current = onOpen;
    onCloseRef.current = onClose;
    const setState = (next) => {
        if (stateRef.current === next)
            return;
        stateRef.current = next;
        for (const cb of stateSubsRef.current) {
            try {
                cb(next);
            }
            catch (err) {
                console.warn("useTerminalTransport: state subscriber threw", err);
            }
        }
    };
    useEffect(() => {
        let disposed = false;
        let reconnectAttempts = 0;
        let connectedAtLeastOnce = false;
        let visibilityListener = null;
        let reconnectTimer = null;
        const connect = () => {
            if (disposed)
                return;
            setState("connecting");
            const ws = createSocket(url);
            wsRef.current = ws;
            ws.onopen = () => {
                const wasReconnect = connectedAtLeastOnce;
                connectedAtLeastOnce = true;
                reconnectAttempts = 0;
                genRef.current += 1;
                setState("open");
                onOpenRef.current?.(wasReconnect, genRef.current);
            };
            ws.onmessage = (event) => {
                let msg;
                try {
                    msg = JSON.parse(event.data);
                }
                catch {
                    console.warn("useTerminalTransport: non-JSON message", event.data);
                    return;
                }
                for (const cb of messageSubsRef.current) {
                    try {
                        cb(msg);
                    }
                    catch (err) {
                        console.warn("useTerminalTransport: message subscriber threw", err);
                    }
                }
            };
            ws.onclose = (event) => {
                if (wsRef.current === ws)
                    wsRef.current = null;
                setState("closed");
                onCloseRef.current?.(event);
                if (disposed || isCleanWsClose(event.code))
                    return;
                const isPageHidden = typeof document !== "undefined" && document.visibilityState === "hidden";
                if (isPageHidden) {
                    const onVisible = () => {
                        if (document.visibilityState !== "visible")
                            return;
                        document.removeEventListener("visibilitychange", onVisible);
                        visibilityListener = null;
                        if (disposed)
                            return;
                        reconnectAttempts = 0;
                        connect();
                    };
                    document.addEventListener("visibilitychange", onVisible);
                    visibilityListener = onVisible;
                    return;
                }
                reconnectAttempts += 1;
                const delay = Math.min(RECONNECT_BASE_DELAY_MS * 2 ** (reconnectAttempts - 1), RECONNECT_MAX_DELAY_MS);
                reconnectTimer = setTimeout(connect, delay);
            };
        };
        connect();
        return () => {
            disposed = true;
            if (reconnectTimer !== null) {
                clearTimeout(reconnectTimer);
                reconnectTimer = null;
            }
            if (visibilityListener) {
                document.removeEventListener("visibilitychange", visibilityListener);
                visibilityListener = null;
            }
            wsRef.current?.close();
            wsRef.current = null;
        };
    }, [url, createSocket]);
    const sendJson = useCallback((msg) => {
        const ws = wsRef.current;
        if (!ws || ws.readyState !== WebSocket.OPEN)
            return false;
        if (ws.bufferedAmount > WS_SEND_HIGH_WATER)
            return false;
        try {
            ws.send(JSON.stringify(msg));
            return true;
        }
        catch (err) {
            console.warn("useTerminalTransport: ws.send threw", err);
            return false;
        }
    }, []);
    const subscribe = useCallback((listener) => {
        messageSubsRef.current.add(listener);
        return () => {
            messageSubsRef.current.delete(listener);
        };
    }, []);
    const onStateChange = useCallback((listener) => {
        stateSubsRef.current.add(listener);
        return () => {
            stateSubsRef.current.delete(listener);
        };
    }, []);
    const currentGen = useCallback(() => genRef.current, []);
    const state = useCallback(() => stateRef.current, []);
    return { sendJson, currentGen, subscribe, onStateChange, state };
}
