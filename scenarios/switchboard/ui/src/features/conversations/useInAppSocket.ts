import { useCallback, useEffect, useRef, useState } from "react";

import { inAppSocketUrl, type MessageMedia } from "../../api/console";

export type SocketState = "idle" | "connecting" | "open" | "closed" | "error";

export interface InboundSocketPayload {
  text?: string;
  media?: MessageMedia[];
  error?: string;
  reply_to_remote_id?: string;
}

interface UseInAppSocketOptions {
  /** Connect only when a thread key is present; `undefined` means idle. */
  threadKey: string | undefined;
  onMessage: (payload: InboundSocketPayload) => void;
}

/**
 * One WebSocket to the in-app channel adapter for the open thread. Reconnects
 * with backoff while the thread stays open, and reports its state so the
 * composer can disable itself with a reason instead of failing silently.
 */
export function useInAppSocket({ threadKey, onMessage }: UseInAppSocketOptions) {
  const [state, setState] = useState<SocketState>("idle");
  const socket = useRef<WebSocket | null>(null);
  const handler = useRef(onMessage);
  handler.current = onMessage;

  useEffect(() => {
    if (!threadKey) {
      setState("idle");
      return undefined;
    }
    let attempt = 0;
    let timer: number | undefined;
    let disposed = false;

    const connect = () => {
      if (disposed) return;
      setState("connecting");
      const connection = new WebSocket(inAppSocketUrl(threadKey));
      socket.current = connection;
      connection.onopen = () => {
        attempt = 0;
        setState("open");
      };
      connection.onmessage = (event) => {
        try {
          handler.current(JSON.parse(String(event.data)) as InboundSocketPayload);
        } catch {
          handler.current({ error: "unreadable" });
        }
      };
      connection.onerror = () => setState("error");
      connection.onclose = () => {
        socket.current = null;
        if (disposed) return;
        setState("closed");
        attempt += 1;
        timer = window.setTimeout(connect, Math.min(10_000, 500 * 2 ** attempt));
      };
    };
    connect();
    return () => {
      disposed = true;
      if (timer) window.clearTimeout(timer);
      socket.current?.close();
      socket.current = null;
    };
  }, [threadKey]);

  const send = useCallback((payload: Record<string, unknown>): boolean => {
    const connection = socket.current;
    if (!connection || connection.readyState !== WebSocket.OPEN) return false;
    connection.send(JSON.stringify(payload));
    return true;
  }, []);

  return { state, send };
}
