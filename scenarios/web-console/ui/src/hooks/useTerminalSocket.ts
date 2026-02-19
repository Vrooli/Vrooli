import { useEffect, useRef, useCallback } from "react";
import type { Terminal } from "@xterm/xterm";
import { buildSessionWsUrl } from "../lib/api";
import { ANSI } from "../lib/ansi";

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
  type: "stdin" | "stdout" | "resize" | "exit" | "error" | "ping" | "pong";
  /** Terminal I/O payload (stdin input or stdout output). */
  data?: string;
  /** New terminal width for resize messages. */
  cols?: number;
  /** New terminal height for resize messages. */
  rows?: number;
  /** Process exit code (sent with "exit" messages). */
  code?: number;
}

/** Factory function for creating WebSocket connections. Override in tests. */
export type SocketFactory = (url: string) => WebSocket;

const defaultSocketFactory: SocketFactory = (url) => new WebSocket(url);

/** Maps known WS error messages to user-facing recovery hints. */
const WS_ERROR_RECOVERY: Record<string, string> = {
  "Invalid message format": "A malformed message was sent. This is usually harmless.",
  "Terminal process is not accepting input": "The terminal process has stopped. Close this pane and open a new terminal.",
};

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

  const sendMessage = useCallback((msg: TerminalMessage) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(msg));
    }
  }, []);

  const sendInput = useCallback(
    (data: string) => {
      sendMessage({ type: "stdin", data });
    },
    [sendMessage],
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
    const ws = createSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      sendResize(terminal.cols, terminal.rows);
      onReady?.();
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
          if (msg.data) terminal.write(msg.data);
          break;
        case "exit": {
          const code = msg.code ?? 0;
          const exitLabel = code === 0
            ? `${ANSI.gray}[Session ended]`
            : `${ANSI.red}[Session ended with exit code ${code}]`;
          terminal.write(`\r\n${exitLabel}${ANSI.reset}\r\n`);
          onExit?.(sessionId);
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
      }
    };

    ws.onclose = (event) => {
      if (isCleanWsClose(event.code)) {
        terminal.write(
          `\r\n${ANSI.gray}[Disconnected]${ANSI.reset}\r\n`,
        );
      } else {
        terminal.write(
          `\r\n${ANSI.red}[Connection lost]${ANSI.reset}\r\n` +
          `${ANSI.gray}  Close this pane and open a new terminal to reconnect.${ANSI.reset}\r\n`,
        );
      }
    };

    // Terminal input -> WebSocket stdin
    const inputDisposable = terminal.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "stdin", data } satisfies TerminalMessage));
      }
    });

    return () => {
      inputDisposable.dispose();
      ws.close();
      wsRef.current = null;
    };
  }, [sessionId, terminal, onExit, onReady, sendResize, createSocket]);

  return { sendInput, sendResize };
}
