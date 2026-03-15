import { useEffect, useRef, useCallback } from "react";
import type { Terminal } from "@xterm/xterm";
import { buildSessionWsUrl } from "../lib/api";
import { ANSI } from "../lib/ansi";
import { LocalEchoController } from "../lib/localEcho";
import { applyModifiers } from "../consts/toolbar-keys";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
// DOC: docs/internal/SEAMS.md#websocket-factory-seam-ui
/**
 * WebSocket JSON message protocol matching the Go backend (terminal_ws.go).
 *
 * Message directions:
 *   Client → Server: stdin, resize, ping
 *   Server → Client: stdout, exit, error, pong
 *
 * [REQ:P0-002b] WebSocket I/O Streaming
 */
export interface TerminalMessage {
  type: "stdin" | "stdout" | "resize" | "resize_info" | "exit" | "error" | "ping" | "pong" | "sync_warning";
  /** Terminal I/O payload (stdin input or stdout output). */
  data?: string;
  /** New terminal width for resize messages. */
  cols?: number;
  /** New terminal height for resize messages. */
  rows?: number;
  /** Process exit code (sent with "exit" messages). */
  code?: number;
  /** Cumulative coalesced frame count (sent with "sync_warning" messages). */
  coalesced_frames?: number;
}

/** Factory function for creating WebSocket connections. Override in tests. */
export type SocketFactory = (url: string) => WebSocket;

const defaultSocketFactory: SocketFactory = (url) => new WebSocket(url);

/** Maps known WS error messages to user-facing recovery hints. */
const WS_ERROR_RECOVERY: Record<string, string> = {
  "Invalid message format": "A malformed message was sent. This is usually harmless.",
  "Terminal process is not accepting input": "The terminal process has stopped. Close this pane and open a new terminal.",
};

const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_BASE_DELAY_MS = 400;
const RECONNECT_MAX_DELAY_MS = 5000;
const MAX_OUTPUT_PROBE_CHARS = 12000;
const MAX_PENDING_INPUT_MESSAGES = 64;

function appendOutputProbe(sessionId: string, data: string): void {
  if (typeof window === "undefined" || !data) return;
  const probeWindow = window as Window & {
    __wc_terminal_output?: Record<string, string>;
  };
  const probe = probeWindow.__wc_terminal_output ?? {};
  const previous = probe[sessionId] ?? "";
  const next = (previous + data).slice(-MAX_OUTPUT_PROBE_CHARS);
  probe[sessionId] = next;
  probeWindow.__wc_terminal_output = probe;
}

/**
 * WebSocket close codes 1000 (Normal) and 1001 (Going Away) indicate an
 * intentional, expected close — e.g. the user closed the tab or the server
 * shut down gracefully. Any other code signals an unexpected disconnection
 * (network failure, server crash, protocol error, etc.).
 */
export function isCleanWsClose(code: number): boolean {
  return code === 1000 || code === 1001;
}

interface UseTerminalSocketOptions {
  sessionId: string;
  terminal: Terminal | null;
  onExit?: (sessionId: string) => void;
  onReady?: () => void;
  /** Injectable WebSocket factory for testing without real connections. */
  createSocket?: SocketFactory;
}

/**
 * Manages the WebSocket connection for a terminal session.
 * Handles bidirectional I/O (stdin/stdout), resize messages, and lifecycle events.
 *
 * [REQ:P0-002b] WebSocket I/O Streaming
 * [REQ:P0-004b] api-base WebSocket Integration
 */
export function useTerminalSocket({
  sessionId,
  terminal,
  onExit,
  onReady,
  createSocket = defaultSocketFactory,
}: UseTerminalSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingInputRef = useRef<string[]>([]);

  // Store event-handler callbacks in refs so they can be updated without
  // tearing down the WebSocket connection. These are "fire-and-forget"
  // handlers — the connection effect reads them at call time, not setup time.
  const onExitRef = useRef(onExit);
  const onReadyRef = useRef(onReady);
  onExitRef.current = onExit;
  onReadyRef.current = onReady;

  const enqueueInput = useCallback((data: string) => {
    if (!data) return;
    pendingInputRef.current.push(data);
    if (pendingInputRef.current.length > MAX_PENDING_INPUT_MESSAGES) {
      pendingInputRef.current.splice(
        0,
        pendingInputRef.current.length - MAX_PENDING_INPUT_MESSAGES,
      );
    }
  }, []);

  const sendMessage = useCallback((msg: TerminalMessage) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(msg));
      return true;
    }
    return false;
  }, []);

  const sendInput = useCallback(
    (data: string): boolean => {
      if (sendMessage({ type: "stdin", data })) {
        return true;
      }
      // Inputs can be triggered before socket open (launcher shortcuts,
      // restored-session command replay). Queue and flush on connect.
      enqueueInput(data);
      return false;
    },
    [enqueueInput, sendMessage],
  );

  const sendResize = useCallback(
    (cols: number, rows: number) => {
      sendMessage({ type: "resize", cols, rows });
    },
    [sendMessage],
  );

  useEffect(() => {
    if (!terminal) return;

    const wsUrl = buildSessionWsUrl(sessionId);
    const localEcho = new LocalEchoController();
    let disposed = false;
    let reconnectAttempts = 0;
    let connectedAtLeastOnce = false;

    const flushPendingInput = () => {
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) return;
      while (pendingInputRef.current.length > 0) {
        const next = pendingInputRef.current.shift();
        if (!next) continue;
        ws.send(JSON.stringify({ type: "stdin", data: next } satisfies TerminalMessage));
      }
    };

    const connect = () => {
      if (disposed) return;

      const ws = createSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        const wasReconnect = connectedAtLeastOnce;
        connectedAtLeastOnce = true;
        reconnectAttempts = 0;
        localEcho.reset();
        flushPendingInput();
        // Ensure PTY has client dimensions before history replay arrives.
        // onReady may send a refined resize after fit(), but this provides
        // an immediate baseline.
        sendResize(terminal.cols, terminal.rows);
        if (wasReconnect) {
          terminal.reset();
          terminal.write(`\r\n${ANSI.gray}[Reconnected]${ANSI.reset}\r\n`);
        }
        onReadyRef.current?.();
      };

      ws.onmessage = (event) => {
        let msg: TerminalMessage;
        try {
          msg = JSON.parse(event.data as string) as TerminalMessage;
        } catch {
          // Malformed message from server — log but don't crash the handler
          console.warn("WebSocket: received non-JSON message", event.data);
          return;
        }
        switch (msg.type) {
          case "stdout":
            if (msg.data) {
              const processed = localEcho.processOutput(msg.data);
              if (processed) terminal.write(processed);
              appendOutputProbe(sessionId, msg.data);
            }
            break;
          case "exit": {
            const code = msg.code ?? 0;
            const exitLabel = code === 0
              ? `${ANSI.gray}[Session ended]`
              : `${ANSI.red}[Session ended with exit code ${code}]`;
            terminal.write(`\r\n${exitLabel}${ANSI.reset}\r\n`);
            onExitRef.current?.(sessionId);
            break;
          }
          case "error": {
            terminal.write(
              `\r\n${ANSI.red}[Error: ${msg.data}]${ANSI.reset}\r\n`,
            );
            // Provide recovery guidance for known error types
            const recoveryHint = WS_ERROR_RECOVERY[msg.data ?? ""] ?? "";
            if (recoveryHint) {
              terminal.write(
                `${ANSI.gray}  ${recoveryHint}${ANSI.reset}\r\n`,
              );
            }
            break;
          }
          case "sync_warning": {
            // Reset local echo predictions — heavy coalesced output means
            // pending predictions are stale and could suppress legitimate chars.
            localEcho.reset();
            const coalesced = msg.coalesced_frames ?? 0;
            terminal.write(
              `\r\n${ANSI.yellow}[Warning: ${coalesced} output frames coalesced — terminal may lag]${ANSI.reset}\r\n` +
              `${ANSI.gray}  Output will catch up automatically.${ANSI.reset}\r\n`,
            );
            break;
          }
          case "resize_info":
            // Informational: the server reports the effective PTY size.
            // xterm.js handles reflow for smaller viewports automatically.
            break;
        }
      };

      ws.onclose = (event) => {
        localEcho.reset();
        if (wsRef.current === ws) {
          wsRef.current = null;
        }
        if (disposed) {
          return;
        }

        if (isCleanWsClose(event.code)) {
          terminal.write(
            `\r\n${ANSI.gray}[Disconnected]${ANSI.reset}\r\n`,
          );
          return;
        }

        // If the page is hidden (backgrounded browser tab, phone screen locked),
        // defer reconnection until the tab becomes visible again. This prevents
        // the stale-terminal problem where the old bail-out silently abandoned
        // reconnection attempts.
        const isPageHidden = typeof document !== "undefined" && document.visibilityState === "hidden";
        if (isPageHidden) {
          terminal.write(
            `\r\n${ANSI.gray}[Connection lost while backgrounded — will reconnect when tab is active]${ANSI.reset}\r\n`,
          );
          const onVisible = () => {
            if (document.visibilityState !== "visible") return;
            document.removeEventListener("visibilitychange", onVisible);
            visibilityListenerRef = null;
            if (disposed) return;
            reconnectAttempts = 0; // Fresh start on visibility return
            connect();
          };
          document.addEventListener("visibilitychange", onVisible);
          visibilityListenerRef = onVisible;
          return;
        }

        if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts += 1;
          const delay = Math.min(
            RECONNECT_BASE_DELAY_MS * (2 ** (reconnectAttempts - 1)),
            RECONNECT_MAX_DELAY_MS,
          );
          terminal.write(
            `\r\n${ANSI.gray}[Connection lost, reconnecting...]${ANSI.reset}\r\n`,
          );
          reconnectTimerRef.current = setTimeout(connect, delay);
          return;
        }

        terminal.write(
          `\r\n${ANSI.red}[Connection lost]${ANSI.reset}\r\n` +
          `${ANSI.gray}  Reconnect attempts exhausted. Open a new terminal if this persists.${ANSI.reset}\r\n`,
        );
      };
    };

    let visibilityListenerRef: (() => void) | null = null;
    connect();

    // Terminal input -> WebSocket stdin (with local echo for printable chars).
    // When mobile toolbar modifier toggles are active, apply them to the input
    // before sending. Reading from the store directly (not via subscription)
    // ensures we always see the latest modifier state.
    const inputDisposable = terminal.onData((rawData) => {
      const mods = useWorkspaceStore.getState().modifiers;
      const hasModifier = mods.ctrl || mods.alt || mods.shift;
      let data = rawData;
      if (hasModifier) {
        const { data: modified } = applyModifiers(rawData, mods);
        data = modified;
        useWorkspaceStore.getState().clearModifiers();
      }
      const echo = localEcho.handleInput(data);
      if (echo) terminal.write(echo);
      const ws = wsRef.current;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "stdin", data } satisfies TerminalMessage));
      } else {
        enqueueInput(data);
      }
    });

    return () => {
      disposed = true;
      if (reconnectTimerRef.current !== null) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
      if (visibilityListenerRef) {
        document.removeEventListener("visibilitychange", visibilityListenerRef);
        visibilityListenerRef = null;
      }
      inputDisposable.dispose();
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [sessionId, terminal, sendResize, createSocket, enqueueInput]);

  return { sendInput, sendResize };
}
