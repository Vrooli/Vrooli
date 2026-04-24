import { useCallback, useEffect, useRef } from "react";
import type { Terminal } from "@xterm/xterm";
import { buildSessionWsUrl } from "../../lib/api";
import { ANSI } from "../../lib/ansi";
import { LocalEchoController } from "../../lib/localEcho";
import { applyModifiers } from "../../consts/toolbar-keys";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import {
  createInputGate,
  type GateResult,
  type InputSource,
  type TerminalInputGate,
} from "../../components/terminal/inputGate";
import { getTerminalDebugProbe } from "../../components/terminal/debug";
import type {
  ConversationEventMessage,
  TerminalMessage,
} from "../../types/terminal";
import {
  useTerminalTransport,
  type SocketFactory,
} from "./useTerminalTransport";
import {
  useStdinAck,
  type InputSettledListener,
  type PendingInputEntry,
} from "./useStdinAck";

const MAX_OUTPUT_PROBE_CHARS = 12000;
const HISTORY_FLUSH_TIMEOUT_MS = 5000;

/** Maps known WS error messages to user-facing recovery hints. */
const WS_ERROR_RECOVERY: Record<string, string> = {
  "Invalid message format": "A malformed message was sent. This is usually harmless.",
  "Terminal process is not accepting input":
    "The terminal process has stopped. Close this pane and open a new terminal.",
  session_not_ready:
    "The terminal session did not confirm readiness in time. Reconnect or reopen this pane.",
};

/**
 * byteLengthUTF8 returns the UTF-8 byte length of a JSON-decoded
 * stdout payload. The server's totalOutputBytes counts raw bytes from
 * the PTY, not JavaScript string code units — a multi-byte character
 * is multiple bytes on the wire. TextEncoder gives us the correct
 * count in one call. Falls back to `.length` only if TextEncoder is
 * unavailable (should never happen in supported browsers / vitest).
 */
function byteLengthUTF8(s: string): number {
  if (typeof TextEncoder !== "undefined") {
    try {
      return new TextEncoder().encode(s).byteLength;
    } catch {
      // fallthrough
    }
  }
  return s.length;
}

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
 * stripTerminalResponses removes terminal-generated replies (DA1/DA2/DA3,
 * DSR, CPR) from input payloads before they reach the PTY. xterm.js
 * emits these in reply to queries that appear in PTY output — including
 * queries replayed from session history, where the original querying
 * program is long gone. Leaving them in the input stream spams the
 * current shell with garbage. DA queries that TUI programs (Claude
 * Code, vim, etc.) legitimately need answered are handled server-side
 * in session.readLoop instead (see api/ansi_responder.go).
 *
 * Matched CSI forms:
 *   \e[?…c   DA1 response        \e[>…c   DA2/DA3 response
 *   \e[…n    Device Status Report \e[…R   Cursor Position Report
 *   \e[?…$y  DECRPM (DECRQM reply)
 */
// eslint-disable-next-line no-control-regex -- intentionally matches CSI ESC byte
const RE_TERMINAL_RESPONSE = /\x1b\[[\x30-\x3f]*[\x20-\x2f]*[cnRy]/g;
function stripTerminalResponses(s: string): string {
  if (s.indexOf("\x1b") === -1) return s;
  return s.replace(RE_TERMINAL_RESPONSE, "");
}

export interface UseTerminalSessionOptions {
  sessionId: string;
  terminal: Terminal | null;
  onExit?: (sessionId: string) => void;
  onReady?: () => void;
  createSocket?: SocketFactory;
  /** Byte offset for history resume (from terminal cache). */
  historyOffset?: number;
  /** Whether the terminal was restored from a serialized cache entry. */
  hasCachedState?: boolean;
  onConversationEvent?: (
    event: ConversationEventMessage,
    sendAck: (stage: string, message?: string, backend?: string) => void,
  ) => void;
  onConversationEventUpdate?: (
    eventId: string,
    patch: {
      speechParagraphs?: string[];
      originalSpeechParagraphs?: string[];
      summarized?: boolean;
    },
  ) => void;
}

export interface UseTerminalSessionResult {
  /** Single-path input entry used by every UI source. */
  submitInput: (data: string, source: InputSource) => GateResult;
  gate: TerminalInputGate;
  sendResize: (cols: number, rows: number) => void;
  totalBytesRef: { current: number };
  subscribeInputSettled: (cb: InputSettledListener) => () => void;
  subscribePendingInput: (cb: () => void) => () => void;
  getPendingInputSnapshot: () => readonly PendingInputEntry[];
}

/**
 * useTerminalSession is the protocol orchestrator for a single
 * terminal pane. It composes:
 *   - useTerminalTransport (WebSocket lifecycle)
 *   - useStdinAck         (seq/ack, queue, wsGen write barrier)
 *   - LocalEchoController (predictive echo)
 *   - TerminalInputGate   (single-path input decision layer)
 *
 * Responsibilities kept inside this hook (nowhere else):
 *   - session_ready gating
 *   - history_end replay buffering
 *   - pty_state → local-echo enable/disable
 *   - conversation side-channel fan-out
 *
 * Everything else — raw sends, message decoding, reconnect backoff,
 * stdin protocol — lives in the component hooks.
 *
 * [REQ:P0-002b] WebSocket I/O Streaming
 * [REQ:P0-004b] api-base WebSocket Integration
 */
export function useTerminalSession({
  sessionId,
  terminal,
  onExit,
  onReady,
  createSocket,
  historyOffset,
  hasCachedState,
  onConversationEvent,
  onConversationEventUpdate,
}: UseTerminalSessionOptions): UseTerminalSessionResult {
  const totalBytesRef = useRef<number>(0);
  const sessionReadyRef = useRef(false);
  const wsGenAtReadyRef = useRef(0);
  // Cached-state flag evaluated per-connection (not once per hook lifetime).
  const hasCachedStateForConnectionRef = useRef(hasCachedState ?? false);
  const altBufferRef = useRef(false);

  const onExitRef = useRef(onExit);
  const onReadyRef = useRef(onReady);
  const onConversationEventRef = useRef(onConversationEvent);
  const onConversationEventUpdateRef = useRef(onConversationEventUpdate);
  onExitRef.current = onExit;
  onReadyRef.current = onReady;
  onConversationEventRef.current = onConversationEvent;
  onConversationEventUpdateRef.current = onConversationEventUpdate;

  const baseWsUrl = buildSessionWsUrl(sessionId);
  const wsUrl = historyOffset && historyOffset > 0
    ? `${baseWsUrl}${baseWsUrl.includes("?") ? "&" : "?"}history_offset=${historyOffset}`
    : baseWsUrl;

  const localEchoRef = useRef(new LocalEchoController());

  // --- History replay state (lives for the lifetime of the hook;
  // reset per-connection via transport onOpen). ---
  const replayingHistoryRef = useRef(false);
  const historyBufferRef = useRef<string[]>([]);
  const historyTimeoutIdRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Transport is constructed first; the stdin-ack and session layer
  // consult transport.currentGen() and transport.sendJson().
  const transportRef = useRef<ReturnType<typeof useTerminalTransport> | null>(
    null,
  );

  const isSessionReady = useCallback(() => sessionReadyRef.current, []);
  const currentGen = useCallback(
    () => transportRef.current?.currentGen() ?? 0,
    [],
  );
  const sendFrame = useCallback(
    (msg: TerminalMessage): boolean => transportRef.current?.sendJson(msg) ?? false,
    [],
  );

  const stdin = useStdinAck({
    sendFrame,
    isSessionReady,
    currentGen,
  });

  const gate: TerminalInputGate = useRef(
    createInputGate({
      transport: {
        send: stdin.send,
        enqueue: stdin.enqueue,
      },
      getTerminal: () => terminalRef.current,
    }),
  ).current;

  // Hold the terminal in a ref so the gate's getTerminal closure always
  // sees the latest instance.
  const terminalRef = useRef<Terminal | null>(terminal);
  terminalRef.current = terminal;

  const flushHistoryBuffer = useCallback(() => {
    if (!replayingHistoryRef.current) return;
    replayingHistoryRef.current = false;
    if (historyTimeoutIdRef.current !== null) {
      clearTimeout(historyTimeoutIdRef.current);
      historyTimeoutIdRef.current = null;
    }
    const t = terminalRef.current;
    if (!t) {
      historyBufferRef.current = [];
      return;
    }
    if (historyBufferRef.current.length > 0) {
      t.write(historyBufferRef.current.join(""));
      for (const chunk of historyBufferRef.current) {
        appendOutputProbe(sessionId, chunk);
      }
    }
    historyBufferRef.current = [];
  }, [sessionId]);

  const onTransportOpen = useCallback((wasReconnect: boolean, _gen: number) => {
    sessionReadyRef.current = false;
    localEchoRef.current.reset();
    stdin.resetForNewConnection(currentGen());
    // Per-connection cache flag resets to the initial cached-state
    // value so a reconnect with a stale offset still triggers reset().
    hasCachedStateForConnectionRef.current = hasCachedState ?? false;
    // Snap the live byte counter to the offset we're asking the
    // server to resume from (or zero for a fresh connection). This
    // keeps the counter + rendered xterm state consistent for the
    // cache save path: after history_end the counter === server's
    // total_bytes, and every subsequent live stdout frame advances
    // it in lockstep with xterm.write.
    totalBytesRef.current = historyOffset ?? 0;
    // History replay state per connection.
    replayingHistoryRef.current = true;
    historyBufferRef.current = [];
    if (historyTimeoutIdRef.current !== null) {
      clearTimeout(historyTimeoutIdRef.current);
    }
    historyTimeoutIdRef.current = setTimeout(flushHistoryBuffer, HISTORY_FLUSH_TIMEOUT_MS);

    const t = terminalRef.current;
    if (t) {
      transportRef.current?.sendJson({ type: "resize", cols: t.cols, rows: t.rows });
      if (wasReconnect) {
        t.reset();
        t.write(`\r\n${ANSI.gray}[Reconnected]${ANSI.reset}\r\n`);
      }
    }
    onReadyRef.current?.();
  }, [hasCachedState, historyOffset, flushHistoryBuffer, stdin, currentGen]);

  const onTransportClose = useCallback(() => {
    sessionReadyRef.current = false;
    localEchoRef.current.reset();
    stdin.handleClose();
    // Surface the disconnect to the user.
    const t = terminalRef.current;
    if (t) {
      t.write(`\r\n${ANSI.gray}[Disconnected]${ANSI.reset}\r\n`);
    }
  }, [stdin]);

  const transport = useTerminalTransport({
    url: wsUrl,
    createSocket,
    onOpen: onTransportOpen,
    onClose: onTransportClose,
  });
  transportRef.current = transport;

  const sendConversationAck = useCallback(
    (
      event: ConversationEventMessage,
      stage: string,
      message?: string,
      backend?: string,
    ) => {
      if (!event.id || !event.source) return;
      transport.sendJson({
        type: "conversation_event_ack",
        eventId: event.id,
        source: event.source,
        stage,
        backend,
        data: message,
      });
    },
    [transport],
  );

  // Incoming message handler. Installed as a transport subscriber.
  useEffect(() => {
    const unsubscribe = transport.subscribe((msg) => {
      switch (msg.type) {
        case "session_ready": {
          sessionReadyRef.current = true;
          wsGenAtReadyRef.current = msg.gen ?? transport.currentGen();
          stdin.flush();
          break;
        }
        case "stdin_ack": {
          stdin.acceptAck(msg.seq ?? 0, msg.ok === true);
          break;
        }
        case "stdout": {
          if (!msg.data) break;
          // Live byte counter — must advance on EVERY stdout frame
          // (history replay AND live), not only on history_end. The
          // cached sessionStorage entry uses this counter to declare
          // a resume offset; if the counter lags behind the
          // serialized xterm state, reconnecting duplicates the
          // trailing bytes in the scrollback. Bug C regression guard:
          // see __tests__/terminal-scrollback-dedup.test.tsx.
          totalBytesRef.current += byteLengthUTF8(msg.data);
          if (replayingHistoryRef.current) {
            historyBufferRef.current.push(msg.data);
          } else {
            const t = terminalRef.current;
            if (!t) break;
            const processed = localEchoRef.current.processOutput(msg.data);
            if (processed) t.write(processed);
            appendOutputProbe(sessionId, msg.data);
          }
          break;
        }
        case "history_end": {
          const resumed = msg.resumed === true;
          const t = terminalRef.current;
          if (!resumed && hasCachedStateForConnectionRef.current && t) {
            // Cache was stale — reset xterm before writing fresh history.
            t.reset();
            // Reset live byte counter because we're replacing every
            // byte xterm had. The subsequent history chunks (already
            // written above via the stdout branch) advanced the
            // counter; we snap it back to the server's authoritative
            // value so cache saves stay consistent.
          }
          if (msg.total_bytes !== undefined) {
            // Server's authoritative total_bytes at subscribe time.
            // After a full replay this matches our live counter (we
            // counted every replayed stdout frame). After a delta
            // replay (resumed=true) our counter already equals this
            // because we started from the cached offset and counted
            // only the delta. In dev, warn on drift.
            if (
              import.meta.env?.DEV &&
              totalBytesRef.current !== msg.total_bytes
            ) {
              console.warn(
                "useTerminalSession: totalBytes drift — " +
                  `live=${totalBytesRef.current} server=${msg.total_bytes} resumed=${resumed}`,
              );
            }
            totalBytesRef.current = msg.total_bytes;
          }
          hasCachedStateForConnectionRef.current = false;
          flushHistoryBuffer();
          break;
        }
        case "pty_state": {
          const altBuffer = msg.altBuffer === true;
          altBufferRef.current = altBuffer;
          // Local echo produces visible flicker and wrong predictions
          // inside alt-buffer TUIs (Claude Code, vim). Disable
          // unconditionally while the session is in alt-buffer.
          localEchoRef.current.enabled = !altBuffer;
          break;
        }
        case "exit": {
          flushHistoryBuffer();
          const code = msg.code ?? 0;
          const label =
            code === 0
              ? `${ANSI.gray}[Session ended]`
              : `${ANSI.red}[Session ended with exit code ${code}]`;
          terminalRef.current?.write(`\r\n${label}${ANSI.reset}\r\n`);
          onExitRef.current?.(sessionId);
          break;
        }
        case "error": {
          terminalRef.current?.write(
            `\r\n${ANSI.red}[Error: ${msg.data}]${ANSI.reset}\r\n`,
          );
          const hint = WS_ERROR_RECOVERY[msg.data ?? ""] ?? "";
          if (hint) {
            terminalRef.current?.write(
              `${ANSI.gray}  ${hint}${ANSI.reset}\r\n`,
            );
          }
          break;
        }
        case "sync_warning": {
          localEchoRef.current.reset();
          const coalesced = msg.coalesced_frames ?? 0;
          terminalRef.current?.write(
            `\r\n${ANSI.yellow}[Warning: ${coalesced} output frames coalesced — terminal may lag]${ANSI.reset}\r\n` +
              `${ANSI.gray}  Output will catch up automatically.${ANSI.reset}\r\n`,
          );
          break;
        }
        case "conversation_event": {
          if (msg.data && msg.eventId && msg.source && msg.sequence) {
            const event: ConversationEventMessage = {
              id: msg.eventId,
              source: msg.source,
              role: msg.role === "user" ? "user" : "assistant",
              text: msg.data,
              speechParagraphs: msg.speechParagraphs,
              originalSpeechParagraphs: msg.originalSpeechParagraphs,
              summarized: msg.summarized,
              createdAt: msg.createdAt,
              sequence: msg.sequence,
            };
            onConversationEventRef.current?.(event, (stage, message, backend) =>
              sendConversationAck(event, stage, message, backend),
            );
          }
          break;
        }
        case "conversation_event_update": {
          if (msg.eventId) {
            onConversationEventUpdateRef.current?.(msg.eventId, {
              speechParagraphs: msg.speechParagraphs,
              originalSpeechParagraphs: msg.originalSpeechParagraphs,
              summarized: msg.summarized,
            });
          }
          break;
        }
        case "resize_info": {
          // Informational; xterm reflows on its own.
          break;
        }
      }
    });
    return unsubscribe;
  }, [transport, sessionId, sendConversationAck, stdin, flushHistoryBuffer]);

  // xterm.onData → gate.submit (single path). localEcho prediction
  // still runs on printable single chars before dispatch.
  useEffect(() => {
    if (!terminal) return;
    const disposable = terminal.onData((rawData) => {
      const mods = useWorkspaceStore.getState().modifiers;
      const hasModifier = mods.ctrl || mods.alt || mods.shift;
      let data = stripTerminalResponses(rawData);
      if (data.length === 0) return;
      if (hasModifier) {
        const { data: modified } = applyModifiers(data, mods);
        data = modified;
        useWorkspaceStore.getState().clearModifiers();
      }
      const echo = localEchoRef.current.handleInput(data);
      if (echo) terminal.write(echo);
      gate.submit(data, "xterm");
    });
    return () => {
      disposable.dispose();
    };
  }, [terminal, gate]);

  // Cleanup on unmount.
  useEffect(() => {
    return () => {
      gate.dispose();
      stdin.dispose();
      if (historyTimeoutIdRef.current !== null) {
        clearTimeout(historyTimeoutIdRef.current);
        historyTimeoutIdRef.current = null;
      }
      getTerminalDebugProbe().remove(sessionId);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Debug probe publishing. Updates on every state-bearing transition.
  useEffect(() => {
    const probe = getTerminalDebugProbe();
    const publish = () => {
      probe.update({
        sessionId,
        connectionState: transport.state(),
        wsGen: transport.currentGen(),
        pendingInput: stdin.getPendingSnapshot().length,
        pendingAcks: 0,
        altBuffer: altBufferRef.current,
      });
    };
    publish();
    const unsubPending = stdin.subscribePendingInput(publish);
    const unsubState = transport.onStateChange(publish);
    return () => {
      unsubPending();
      unsubState();
    };
  }, [sessionId, transport, stdin]);

  const sendResize = useCallback(
    (cols: number, rows: number) => {
      transport.sendJson({ type: "resize", cols, rows });
    },
    [transport],
  );

  const submitInput = useCallback(
    (data: string, source: InputSource): GateResult => gate.submit(data, source),
    [gate],
  );

  return {
    submitInput,
    gate,
    sendResize,
    totalBytesRef,
    subscribeInputSettled: stdin.subscribeInputSettled,
    subscribePendingInput: stdin.subscribePendingInput,
    getPendingInputSnapshot: stdin.getPendingSnapshot,
  };
}
