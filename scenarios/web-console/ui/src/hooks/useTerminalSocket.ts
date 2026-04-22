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
 *   Server → Client: stdout, exit, error, pong, session_ready, stdin_ack
 *
 * [REQ:P0-002b] WebSocket I/O Streaming
 */
export interface TerminalMessage {
  type: "stdin" | "stdout" | "resize" | "resize_info" | "exit" | "error" | "ping" | "pong" | "sync_warning" | "history_end" | "conversation_event" | "conversation_event_ack" | "conversation_event_update" | "session_ready" | "stdin_ack";
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
  /** Server's monotonic output byte count (sent with "history_end"). */
  total_bytes?: number;
  /** True when the server honored the client's resume offset (delta-only). */
  resumed?: boolean;
  eventId?: string;
  source?: string;
  stage?: string;
  backend?: string;
  role?: string;
  createdAt?: string;
  sequence?: number;
  speechParagraphs?: string[];
  originalSpeechParagraphs?: string[];
  summarized?: boolean;
  /** Client-assigned sequence number for stdin messages; echoed in stdin_ack. */
  seq?: number;
  /** Per-message success flag (used by stdin_ack). */
  ok?: boolean;
}

export interface ConversationEventMessage {
  id: string;
  source: string;
  role: "assistant" | "user";
  text: string;
  speechParagraphs?: string[];
  originalSpeechParagraphs?: string[];
  summarized?: boolean;
  createdAt?: string;
  sequence: number;
}

/** Factory function for creating WebSocket connections. Override in tests. */
export type SocketFactory = (url: string) => WebSocket;

const defaultSocketFactory: SocketFactory = (url) => new WebSocket(url);

/** Maps known WS error messages to user-facing recovery hints. */
const WS_ERROR_RECOVERY: Record<string, string> = {
  "Invalid message format": "A malformed message was sent. This is usually harmless.",
  "Terminal process is not accepting input": "The terminal process has stopped. Close this pane and open a new terminal.",
  session_not_ready: "The terminal session did not confirm readiness in time. Reconnect or reopen this pane.",
};

const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_BASE_DELAY_MS = 400;
const RECONNECT_MAX_DELAY_MS = 5000;
const MAX_OUTPUT_PROBE_CHARS = 12000;
const MAX_PENDING_INPUT_MESSAGES = 64;

/**
 * Ack timeout for stdin messages. If the server does not echo stdin_ack
 * within this window, the client treats the send as failed, re-enqueues
 * the payload, and fires inputSettled(false). 2 s gives ~4× headroom
 * over observed tmux attach p99 (~500 ms). Override via build env.
 */
const ACK_TIMEOUT_MS = (() => {
  const env = (import.meta as unknown as { env?: Record<string, string | undefined> }).env;
  const raw = env?.VITE_WC_ACK_TIMEOUT_MS;
  const n = raw ? Number(raw) : NaN;
  return Number.isFinite(n) && n > 0 ? n : 2000;
})();

/**
 * bufferedAmount high-water mark above which we stop issuing ws.send and
 * queue instead. Browsers are more likely to silently drop bytes once the
 * WS send buffer grows past ~1 MiB; refusing the send preserves the
 * preserve-on-failure invariant.
 */
const WS_SEND_HIGH_WATER = 1 * 1024 * 1024;

/**
 * Safety timeout for history replay. If the server never sends a
 * "history_end" message (e.g. protocol mismatch with an older server),
 * the client flushes whatever it has buffered and switches to live
 * pass-through mode after this delay.
 */
const HISTORY_FLUSH_TIMEOUT_MS = 5000;

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

/** Snapshot entry for a queued stdin payload awaiting flush. */
export interface PendingInputEntry {
  data: string;
  /** ms epoch when the payload was first queued. */
  addedAt: number;
}

interface PendingAckEntry {
  data: string;
  addedAt: number;
  timer: ReturnType<typeof setTimeout>;
}

/** Callback signature for input-settlement subscribers. */
export type InputSettledListener = (seq: number, ok: boolean) => void;

interface UseTerminalSocketOptions {
  sessionId: string;
  terminal: Terminal | null;
  onExit?: (sessionId: string) => void;
  onReady?: () => void;
  /** Injectable WebSocket factory for testing without real connections. */
  createSocket?: SocketFactory;
  /** Byte offset for history resume (from terminal cache). */
  historyOffset?: number;
  /** Whether the terminal was restored from a serialized cache entry. */
  hasCachedState?: boolean;
  /** Called when a conversation event arrives from the server. */
  onConversationEvent?: (event: ConversationEventMessage, sendAck: (stage: string, message?: string, backend?: string) => void) => void;
  /** Called when an async update (e.g. summarization) arrives for an existing event. */
  onConversationEventUpdate?: (eventId: string, patch: { speechParagraphs?: string[]; originalSpeechParagraphs?: string[]; summarized?: boolean }) => void;
}

/**
 * Manages the WebSocket connection for a terminal session.
 * Handles bidirectional I/O (stdin/stdout), resize messages, and lifecycle events.
 *
 * Send contract:
 *   - sendInput(data) attempts an immediate ws.send gated on session_ready.
 *     Returns true if the frame was handed to the browser's WS stack AND a
 *     pending-ack timer is armed. Returns false when session_ready has not
 *     arrived, ws is not open, send throws, or bufferedAmount exceeds the
 *     high-water mark — in all of those cases the payload is queued and
 *     will be flushed after the next session_ready.
 *   - Draft-clearing callers must wait for an inputSettled(seq, ok) callback
 *     rather than treating the synchronous boolean as confirmation.
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
  historyOffset,
  hasCachedState,
  onConversationEvent,
  onConversationEventUpdate,
}: UseTerminalSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingInputRef = useRef<PendingInputEntry[]>([]);
  const totalBytesRef = useRef<number>(0);

  // Per-connection state for the stdin-ack protocol.
  const sessionReadyRef = useRef(false);
  const nextSeqRef = useRef(1);
  const pendingAcksRef = useRef<Map<number, PendingAckEntry>>(new Map());
  const inputSettledSubsRef = useRef<Set<InputSettledListener>>(new Set());
  const pendingInputSubsRef = useRef<Set<() => void>>(new Set());

  // Store event-handler callbacks and options in refs so they can be updated
  // without tearing down the WebSocket connection. These are "fire-and-forget"
  // handlers — the connection effect reads them at call time, not setup time.
  const onExitRef = useRef(onExit);
  const onReadyRef = useRef(onReady);
  const hasCachedStateRef = useRef(hasCachedState ?? false);
  const onConversationEventRef = useRef(onConversationEvent);
  const onConversationEventUpdateRef = useRef(onConversationEventUpdate);
  onExitRef.current = onExit;
  onReadyRef.current = onReady;
  onConversationEventRef.current = onConversationEvent;
  onConversationEventUpdateRef.current = onConversationEventUpdate;
  hasCachedStateRef.current = hasCachedState ?? false;

  const notifyPendingChanged = useCallback(() => {
    for (const cb of pendingInputSubsRef.current) {
      try {
        cb();
      } catch (err) {
        console.warn("useTerminalSocket: pendingInput subscriber threw", err);
      }
    }
  }, []);

  const notifyInputSettled = useCallback((seq: number, ok: boolean) => {
    for (const cb of inputSettledSubsRef.current) {
      try {
        cb(seq, ok);
      } catch (err) {
        console.warn("useTerminalSocket: inputSettled subscriber threw", err);
      }
    }
  }, []);

  const enqueueInput = useCallback((data: string) => {
    if (!data) return;
    pendingInputRef.current.push({ data, addedAt: Date.now() });
    if (pendingInputRef.current.length > MAX_PENDING_INPUT_MESSAGES) {
      pendingInputRef.current.splice(
        0,
        pendingInputRef.current.length - MAX_PENDING_INPUT_MESSAGES,
      );
    }
    notifyPendingChanged();
  }, [notifyPendingChanged]);

  // Raw send wrapper with try/catch + bufferedAmount high-water guard.
  // Returns true iff the bytes were accepted by the browser's WS stack.
  const safeSend = useCallback((payload: string): boolean => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return false;
    if (ws.bufferedAmount > WS_SEND_HIGH_WATER) return false;
    try {
      ws.send(payload);
      return true;
    } catch (err) {
      console.warn("useTerminalSocket: ws.send threw", err);
      return false;
    }
  }, []);

  const registerAckTimer = useCallback((seq: number, data: string) => {
    const timer = setTimeout(() => {
      const entry = pendingAcksRef.current.get(seq);
      if (!entry) return;
      pendingAcksRef.current.delete(seq);
      console.warn(`useTerminalSocket: input_ack_timeout seq=${seq} len=${data.length}`);
      enqueueInput(entry.data);
      notifyInputSettled(seq, false);
    }, ACK_TIMEOUT_MS);
    pendingAcksRef.current.set(seq, { data, addedAt: Date.now(), timer });
  }, [enqueueInput, notifyInputSettled]);

  const trySendStdin = useCallback((data: string): { sent: true; seq: number } | { sent: false } => {
    const ws = wsRef.current;
    if (!sessionReadyRef.current || !ws || ws.readyState !== WebSocket.OPEN) {
      return { sent: false };
    }
    const seq = nextSeqRef.current;
    if (!safeSend(JSON.stringify({ type: "stdin", data, seq } satisfies TerminalMessage))) {
      return { sent: false };
    }
    nextSeqRef.current = seq + 1;
    registerAckTimer(seq, data);
    return { sent: true, seq };
  }, [registerAckTimer, safeSend]);

  const sendInput = useCallback(
    (data: string): boolean => {
      if (!data) return false;
      const res = trySendStdin(data);
      if (res.sent) return true;
      enqueueInput(data);
      return false;
    },
    [enqueueInput, trySendStdin],
  );

  const sendResize = useCallback(
    (cols: number, rows: number) => {
      // Resize is a connection-level message — it does not participate in
      // the session_ready gating (resize is safe to drop on the floor; the
      // next real resize will supersede it).
      safeSend(JSON.stringify({ type: "resize", cols, rows } satisfies TerminalMessage));
    },
    [safeSend],
  );

  const subscribeInputSettled = useCallback((cb: InputSettledListener): (() => void) => {
    inputSettledSubsRef.current.add(cb);
    return () => {
      inputSettledSubsRef.current.delete(cb);
    };
  }, []);

  const subscribePendingInput = useCallback((cb: () => void): (() => void) => {
    pendingInputSubsRef.current.add(cb);
    return () => {
      pendingInputSubsRef.current.delete(cb);
    };
  }, []);

  const getPendingInputSnapshot = useCallback((): readonly PendingInputEntry[] => {
    // Returned array is a shallow copy so subscribers can safely compare
    // identity between snapshots. Entries themselves are immutable-by-convention.
    return pendingInputRef.current.slice();
  }, []);

  useEffect(() => {
    if (!terminal) return;

    // Append resume offset to the WS URL when the client has a cached
    // terminal state. The server validates the offset and sends only delta
    // data, or falls back to full history if the offset is stale.
    // DOC: docs/concepts/ARCHITECTURE.md#terminal-history-caching
    const baseWsUrl = buildSessionWsUrl(sessionId);
    const wsUrl = historyOffset && historyOffset > 0
      ? `${baseWsUrl}${baseWsUrl.includes("?") ? "&" : "?"}history_offset=${historyOffset}`
      : baseWsUrl;
    const localEcho = new LocalEchoController();
    let disposed = false;
    let reconnectAttempts = 0;
    let connectedAtLeastOnce = false;

    // Clears every pending-ack timer for the current connection and, when
    // requested, re-enqueues the unacked payloads so they survive the
    // reconnect — a dropped ack must not become a silent lost message.
    const clearPendingAcks = (reenqueue: boolean) => {
      const entries = Array.from(pendingAcksRef.current.entries());
      pendingAcksRef.current.clear();
      for (const [seq, entry] of entries) {
        clearTimeout(entry.timer);
        if (reenqueue) {
          // Preserve original addedAt so the pill shows the real age.
          pendingInputRef.current.push({ data: entry.data, addedAt: entry.addedAt });
        }
        notifyInputSettled(seq, false);
      }
      if (reenqueue && entries.length > 0) {
        notifyPendingChanged();
      }
    };

    const flushPendingInput = () => {
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) return;
      if (!sessionReadyRef.current) return;
      let changed = false;
      while (pendingInputRef.current.length > 0) {
        const next = pendingInputRef.current[0];
        if (!next) {
          pendingInputRef.current.shift();
          changed = true;
          continue;
        }
        const res = trySendStdin(next.data);
        if (!res.sent) break;
        pendingInputRef.current.shift();
        changed = true;
      }
      if (changed) notifyPendingChanged();
    };

    // --- History replay buffering ---
    // On each (re)connect the client buffers stdout messages until the
    // server sends "history_end", then writes everything in one batch.
    // This eliminates the visible fast-forward replay on page load/refresh.
    let replayingHistory = false;
    let historyBuffer: string[] = [];
    let historyTimeoutId: ReturnType<typeof setTimeout> | null = null;

    const flushHistoryBuffer = () => {
      if (!replayingHistory) return;
      replayingHistory = false;
      if (historyTimeoutId !== null) {
        clearTimeout(historyTimeoutId);
        historyTimeoutId = null;
      }
      if (historyBuffer.length > 0) {
        terminal.write(historyBuffer.join(""));
        for (const chunk of historyBuffer) {
          appendOutputProbe(sessionId, chunk);
        }
      }
      historyBuffer = [];
    };

    const connect = () => {
      if (disposed) return;

      const ws = createSocket(wsUrl);
      wsRef.current = ws;
      const sendConversationAck = (event: ConversationEventMessage, stage: string, message?: string, backend?: string) => {
        if (!event.id || !event.source) return;
        safeSend(JSON.stringify({
          type: "conversation_event_ack",
          eventId: event.id,
          source: event.source,
          stage,
          backend,
          data: message,
        } satisfies TerminalMessage));
      };

      ws.onopen = () => {
        const wasReconnect = connectedAtLeastOnce;
        connectedAtLeastOnce = true;
        reconnectAttempts = 0;
        localEcho.reset();

        // New connection: old sequence numbers are meaningless. Reset the
        // ack gate, drop any in-flight ack timers (they'll be re-enqueued
        // and re-sent with fresh seqs after session_ready lands).
        sessionReadyRef.current = false;
        nextSeqRef.current = 1;
        clearPendingAcks(true);

        // Reset history replay state for this connection.
        replayingHistory = true;
        historyBuffer = [];
        if (historyTimeoutId !== null) {
          clearTimeout(historyTimeoutId);
        }
        historyTimeoutId = setTimeout(flushHistoryBuffer, HISTORY_FLUSH_TIMEOUT_MS);

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
          case "session_ready":
            sessionReadyRef.current = true;
            // Flush anything that was queued pre-ready (including payloads
            // re-enqueued from the previous connection's cleared acks).
            flushPendingInput();
            break;
          case "stdin_ack": {
            const seq = msg.seq ?? 0;
            const entry = pendingAcksRef.current.get(seq);
            if (!entry) {
              // Unknown seq — either a duplicate ack, an ack for a payload
              // whose timer already fired, or a stray echo from a prior
              // connection generation. Safe to ignore.
              break;
            }
            clearTimeout(entry.timer);
            pendingAcksRef.current.delete(seq);
            notifyInputSettled(seq, msg.ok === true);
            break;
          }
          case "stdout":
            if (msg.data) {
              // Belt-and-suspenders strip of DEC mode-2026 toggles AND
              // the DECRQM query for mode 2026. The server already strips
              // these in `sanitizeForClient` (see api/ansi_responder.go),
              // but we repeat the strip here so a stale server build or
              // an out-of-order broadcast path can't reintroduce the
              // xterm.js v6 `ReferenceError: r is not defined` crash that
              // silently breaks every subsequent render. See project
              // memory `project_web_console_claude_hang_fix` for the full
              // root-cause write-up.
              // eslint-disable-next-line no-control-regex -- intentional CSI
              const stripped = msg.data.replace(/\x1b\[\?2026(?:[hl]|\$p)/g, "");
              if (replayingHistory) {
                historyBuffer.push(stripped);
              } else {
                const processed = localEcho.processOutput(stripped);
                if (processed) terminal.write(processed);
                appendOutputProbe(sessionId, stripped);
              }
            }
            break;
          case "history_end": {
            const resumed = msg.resumed === true;
            if (!resumed && hasCachedStateRef.current) {
              // Server rejected our offset — cache was stale. Clear the
              // deserialized content before writing fresh full history.
              terminal.reset();
            }
            if (msg.total_bytes !== undefined) {
              totalBytesRef.current = msg.total_bytes;
            }
            // After processing metadata, mark cache as consumed so
            // reconnect within the same session doesn't double-reset.
            hasCachedStateRef.current = false;
            flushHistoryBuffer();
            break;
          }
          case "exit": {
            // Flush any pending history so the user sees terminal state
            // before the exit label.
            flushHistoryBuffer();
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
          case "conversation_event":
            if (msg.data && msg.eventId && msg.source && msg.sequence) {
              const event = {
                id: msg.eventId,
                source: msg.source,
                role: msg.role === "user" ? "user" : "assistant",
                text: msg.data,
                speechParagraphs: msg.speechParagraphs,
                originalSpeechParagraphs: msg.originalSpeechParagraphs,
                summarized: msg.summarized,
                createdAt: msg.createdAt,
                sequence: msg.sequence,
              } satisfies ConversationEventMessage;
              onConversationEventRef.current?.(event, (stage, message, backend) => sendConversationAck(event, stage, message, backend));
            }
            break;
          case "conversation_event_update":
            if (msg.eventId) {
              onConversationEventUpdateRef.current?.(msg.eventId, {
                speechParagraphs: msg.speechParagraphs,
                originalSpeechParagraphs: msg.originalSpeechParagraphs,
                summarized: msg.summarized,
              });
            }
            break;
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
        // Connection is gone — drop pending-ack timers and re-enqueue their
        // payloads so nothing is silently lost.
        sessionReadyRef.current = false;
        clearPendingAcks(true);
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
    //
    // Strip terminal-generated responses (DA1/DA2/DA3, DSR, CPR) from input
    // before forwarding to the PTY. xterm.js emits these in reply to
    // queries that appear in PTY output — including queries replayed from
    // the session history buffer on reconnect, where the original querying
    // program is long gone. Leaving them in the input stream spams the
    // current shell with `1;2c0;276;0c…` garbage at the prompt.
    //
    // DA queries that TUI programs (Claude Code, vim, etc.) legitimately
    // need answered are handled server-side in session.readLoop instead —
    // that path synthesizes responses and writes them directly to the PTY
    // master, bypassing xterm.js entirely. See `ansi_responder.go`.
    //
    // Matched CSI forms:
    //   \e[?…c   DA1 response             \e[>…c   DA2/DA3 response
    //   \e[…n    Device Status Report      \e[…R    Cursor Position Report
    //   \e[?…$y  DECRPM (DECRQM reply)
    // eslint-disable-next-line no-control-regex -- intentionally matches CSI ESC byte
    const RE_TERMINAL_RESPONSE = /\x1b\[[\x30-\x3f]*[\x20-\x2f]*[cnRy]/g;
    const stripTerminalResponses = (s: string): string => {
      if (s.indexOf("\x1b") === -1) return s;
      return s.replace(RE_TERMINAL_RESPONSE, "");
    };

    const inputDisposable = terminal.onData((rawData) => {
      const mods = useWorkspaceStore.getState().modifiers;
      const hasModifier = mods.ctrl || mods.alt || mods.shift;
      let data = stripTerminalResponses(rawData);
      if (data.length === 0) return;
      if (hasModifier) {
        const { data: modified } = applyModifiers(data, mods);
        data = modified;
        useWorkspaceStore.getState().clearModifiers();
      }
      const echo = localEcho.handleInput(data);
      if (echo) terminal.write(echo);
      // Route xterm keystrokes through the same seq/ack pipeline so they
      // are covered by the hardened send + re-enqueue path. Direct xterm
      // input has no UI to surface an ack failure (out of scope per plan
      // §13), but observability still records timeouts.
      const res = trySendStdin(data);
      if (!res.sent) enqueueInput(data);
    });

    return () => {
      disposed = true;
      if (historyTimeoutId !== null) {
        clearTimeout(historyTimeoutId);
        historyTimeoutId = null;
      }
      if (reconnectTimerRef.current !== null) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
      if (visibilityListenerRef) {
        document.removeEventListener("visibilitychange", visibilityListenerRef);
        visibilityListenerRef = null;
      }
      inputDisposable.dispose();
      // Drop any outstanding ack timers on unmount; listeners are fired so
      // they can clean up UI state (e.g. MobileToolbar's sending indicator).
      clearPendingAcks(false);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [
    sessionId,
    terminal,
    sendResize,
    createSocket,
    enqueueInput,
    historyOffset,
    notifyInputSettled,
    notifyPendingChanged,
    safeSend,
    trySendStdin,
  ]);

  return {
    sendInput,
    sendResize,
    totalBytesRef,
    subscribeInputSettled,
    subscribePendingInput,
    getPendingInputSnapshot,
  };
}
