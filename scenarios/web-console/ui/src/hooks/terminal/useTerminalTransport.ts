import { useCallback, useEffect, useMemo, useRef } from "react";
import type { TerminalMessage } from "../../types/terminal";

/** Factory for creating WebSocket connections. Override in tests. */
export type SocketFactory = (url: string) => WebSocket;

// Module-level so the default has stable identity across renders; a
// default parameter would allocate a new function every call and churn
// the connect effect below.
const defaultSocketFactory: SocketFactory = (u) => new WebSocket(u);

/**
 * WebSocket close codes 1000 (Normal) and 1001 (Going Away) indicate
 * an intentional, expected close — e.g. the user closed the tab or
 * the server shut down gracefully. Any other code signals an
 * unexpected disconnection (network failure, server crash, etc.).
 */
export function isCleanWsClose(code: number): boolean {
  return code === 1000 || code === 1001;
}

const RECONNECT_BASE_DELAY_MS = 400;
const RECONNECT_MAX_DELAY_MS = 5000;

/** bufferedAmount high-water mark above which sends are refused. */
const WS_SEND_HIGH_WATER = 1 * 1024 * 1024;

/** ConnectionState is the read-only view of transport liveness. */
export type ConnectionState = "idle" | "connecting" | "open" | "closed";

export interface TransportHandle {
  /** Attempt an immediate ws.send. Returns true iff the browser accepted it. */
  sendJson: (msg: TerminalMessage) => boolean;
  /** Send reliable frames whenever the socket is open; cumulative offsets provide the replay barrier. */
  sendReliableJson: (msg: TerminalMessage) => boolean;
  /** Returns the current generation (monotonic per successful open). */
  currentGen: () => number;
  /** Subscribe to incoming messages. Returns an unsubscribe function. */
  subscribe: (listener: (msg: TerminalMessage) => void) => () => void;
  /** Subscribe to connect/disconnect lifecycle. */
  onStateChange: (listener: (s: ConnectionState) => void) => () => void;
  /** Current connection state. */
  state: () => ConnectionState;
}

export interface UseTerminalTransportOptions {
  url: string;
  /** Injectable WebSocket factory for testing without real connections. */
  createSocket?: SocketFactory;
  /**
   * Called once per successful connection (onopen). Use this to reset
   * per-connection state (history buffers, ack maps) in caller hooks.
   * `wasReconnect` is true for every open after the first.
   */
  onOpen?: (wasReconnect: boolean, gen: number) => void;
  /**
   * Called when the socket closes. `cleanClose` is true for 1000/1001.
   * Callers should re-enqueue in-flight work and decide whether to
   * wait for visibility before reconnecting (the transport handles
   * backoff, but is oblivious to UI visibility).
   */
  onClose?: (event: CloseEvent) => void;
}

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
export function useTerminalTransport({
  url,
  createSocket = defaultSocketFactory,
  onOpen,
  onClose,
}: UseTerminalTransportOptions): TransportHandle {
  const wsRef = useRef<WebSocket | null>(null);
  const genRef = useRef(0);
  const stateRef = useRef<ConnectionState>("idle");
  const messageSubsRef = useRef<Set<(msg: TerminalMessage) => void>>(new Set());
  const stateSubsRef = useRef<Set<(s: ConnectionState) => void>>(new Set());
  const onOpenRef = useRef(onOpen);
  const onCloseRef = useRef(onClose);
  onOpenRef.current = onOpen;
  onCloseRef.current = onClose;

  const setState = (next: ConnectionState) => {
    if (stateRef.current === next) return;
    stateRef.current = next;
    for (const cb of stateSubsRef.current) {
      try {
        cb(next);
      } catch (err) {
        console.warn("useTerminalTransport: state subscriber threw", err);
      }
    }
  };

  useEffect(() => {
    let disposed = false;
    let reconnectAttempts = 0;
    let connectedAtLeastOnce = false;
    let visibilityListener: (() => void) | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    const connect = () => {
      if (disposed) return;
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
        let msg: TerminalMessage;
        try {
          msg = JSON.parse(event.data as string) as TerminalMessage;
        } catch {
          console.warn("useTerminalTransport: non-JSON message", event.data);
          return;
        }
        for (const cb of messageSubsRef.current) {
          try {
            cb(msg);
          } catch (err) {
            console.warn("useTerminalTransport: message subscriber threw", err);
          }
        }
      };

      ws.onclose = (event) => {
        if (wsRef.current === ws) wsRef.current = null;
        setState("closed");
        onCloseRef.current?.(event);
        if (disposed || isCleanWsClose(event.code)) return;
        const isPageHidden =
          typeof document !== "undefined" && document.visibilityState === "hidden";
        if (isPageHidden) {
          const onVisible = () => {
            if (document.visibilityState !== "visible") return;
            document.removeEventListener("visibilitychange", onVisible);
            visibilityListener = null;
            if (disposed) return;
            reconnectAttempts = 0;
            connect();
          };
          document.addEventListener("visibilitychange", onVisible);
          visibilityListener = onVisible;
          return;
        }
        reconnectAttempts += 1;
        const delay = Math.min(
          RECONNECT_BASE_DELAY_MS * 2 ** (reconnectAttempts - 1),
          RECONNECT_MAX_DELAY_MS,
        );
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

  const send = useCallback((msg: TerminalMessage, respectHighWater: boolean): boolean => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return false;
    if (respectHighWater && ws.bufferedAmount > WS_SEND_HIGH_WATER) return false;
    try {
      ws.send(JSON.stringify(msg));
      return true;
    } catch (err) {
      console.warn("useTerminalTransport: ws.send threw", err);
      return false;
    }
  }, []);

  const sendJson = useCallback((msg: TerminalMessage): boolean => send(msg, true), [send]);
  const sendReliableJson = useCallback((msg: TerminalMessage): boolean => send(msg, false), [send]);

  const subscribe = useCallback((listener: (msg: TerminalMessage) => void) => {
    messageSubsRef.current.add(listener);
    return () => {
      messageSubsRef.current.delete(listener);
    };
  }, []);

  const onStateChange = useCallback((listener: (s: ConnectionState) => void) => {
    stateSubsRef.current.add(listener);
    return () => {
      stateSubsRef.current.delete(listener);
    };
  }, []);

  const currentGen = useCallback(() => genRef.current, []);
  const state = useCallback(() => stateRef.current, []);

  return useMemo(
    () => ({ sendJson, sendReliableJson, currentGen, subscribe, onStateChange, state }),
    [sendJson, sendReliableJson, currentGen, subscribe, onStateChange, state],
  );
}
