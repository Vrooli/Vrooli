import { useCallback, useEffect, useRef } from "react";
import type { Terminal } from "@xterm/xterm";
import { buildSessionWsUrl } from "../../api/sessions";
import { refreshConversationSession } from "../../hooks/useConversationSession";
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
import type { TerminalMessage } from "../../types/terminal";
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

/** Maps known WS error messages to user-facing recovery hints. */
const WS_ERROR_RECOVERY: Record<string, string> = {
  "Invalid message format": "A malformed message was sent. This is usually harmless.",
  "Terminal process is not accepting input":
    "The terminal process has stopped. Close this pane and open a new terminal.",
  session_not_ready:
    "The terminal session did not confirm readiness in time. Reconnect or reopen this pane.",
};

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
 * stripTerminalResponses removes terminal-generated replies from input
 * payloads before they reach the PTY. xterm.js emits these in reply to
 * queries that appear in PTY output — including queries replayed from the
 * snapshot, where the original querying program is long gone. Leaving them
 * in the input stream spams the current shell with garbage. Queries that
 * TUI programs (Claude Code, vim, etc.) legitimately need answered are
 * handled server-side in session.readLoop instead (see
 * api/ansi_responder.go).
 */
// eslint-disable-next-line no-control-regex -- intentionally matches CSI ESC byte
const RE_TERMINAL_RESPONSE = /\x1b\[[\x30-\x3f]*[\x20-\x2f]*[cnRy]/g;
// eslint-disable-next-line no-control-regex -- intentionally matches OSC framing
const RE_OSC_COLOR_REPLY = /\x1b\][^\x07\x1b]*?rgb:[^\x07\x1b]*(?:\x07|\x1b\\)/g;
function stripTerminalResponses(s: string): string {
  if (s.indexOf("\x1b") === -1) return s;
  return s.replace(RE_TERMINAL_RESPONSE, "").replace(RE_OSC_COLOR_REPLY, "");
}

export interface UseTerminalSessionOptions {
  sessionId: string;
  terminal: Terminal | null;
  onExit?: (sessionId: string) => void;
  onReady?: () => void;
  createSocket?: SocketFactory;
}

export interface UseTerminalSessionResult {
  /** Single-path input entry used by every UI source. */
  submitInput: (data: string, source: InputSource) => GateResult;
  gate: TerminalInputGate;
  sendResize: (cols: number, rows: number) => void;
  subscribeInputSettled: (cb: InputSettledListener) => () => void;
  subscribePendingInput: (cb: () => void) => () => void;
  getPendingInputSnapshot: () => readonly PendingInputEntry[];
  /**
   * Sends a conversation_event_ack frame (client→server playback telemetry)
   * over this pane's terminal WebSocket. Conversation events themselves now
   * arrive via the global SSE channel, but acks stay on the per-pane WS since
   * playback only happens on the mounted/active pane.
   */
  sendConversationAck: (
    eventId: string,
    source: string,
    stage: string,
    message?: string,
    backend?: string,
  ) => void;
}

/**
 * useTerminalSession is the protocol orchestrator for a single
 * terminal pane. It composes:
 *   - useTerminalTransport (WebSocket lifecycle)
 *   - useStdinAck         (seq/ack, queue, wsGen write barrier)
 *   - LocalEchoController (predictive echo)
 *   - TerminalInputGate   (single-path input decision layer)
 *
 * Wire flow on every WS open (fresh OR reconnect):
 *   1. terminal.reset()
 *   2. write every {type:"stdout"} frame directly to xterm
 *      (these are the snapshot, encoding screen + alt-buffer + scrollback)
 *   3. on {type:"history_end"} flip to live mode
 *   4. write every subsequent {type:"stdout"} as live PTY output
 *
 * The xterm instance is a pure renderer; the server's emulator is the
 * source of truth.
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
}: UseTerminalSessionOptions): UseTerminalSessionResult {
  const sessionReadyRef = useRef(false);
  const wsGenAtReadyRef = useRef(0);
  // True from terminal.reset() until history_end. While true, local-echo
  // stays disabled so the snapshot's alt-buffer markers / TUI paint render
  // cleanly.
  const inSnapshotRef = useRef(true);

  const onExitRef = useRef(onExit);
  const onReadyRef = useRef(onReady);
  onExitRef.current = onExit;
  onReadyRef.current = onReady;

  const wsUrl = buildSessionWsUrl(sessionId);

  const localEchoRef = useRef(new LocalEchoController());

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

  const onTransportOpen = useCallback((wasReconnect: boolean, _gen: number) => {
    sessionReadyRef.current = false;
    localEchoRef.current.reset();
    // Local echo stays disabled during snapshot replay; the server's
    // emulator state — including alt-buffer — is restored as bytes hit
    // xterm, so any echo decision must wait until live mode.
    localEchoRef.current.enabled = false;
    stdin.resetForNewConnection(currentGen());
    inSnapshotRef.current = true;

    const t = terminalRef.current;
    if (t) {
      // Wipe xterm.js buffers BEFORE the snapshot streams in. Two calls
      // are necessary because:
      //   - reset() is a soft reset (DECSTR) — clears modes/charsets but
      //     in xterm.js v5+ does not wipe scrollback or buffer cells.
      //   - clear() empties the buffer, including scrollback. Without
      //     this, scrollback content from before a WS reconnect or API
      //     restart layers underneath the new snapshot, producing the
      //     "scroll up shows last page repeated" symptom.
      t.reset();
      t.clear();
      transportRef.current?.sendJson({ type: "resize", cols: t.cols, rows: t.rows });
      if (wasReconnect) {
        t.write(`\r\n${ANSI.gray}[Reconnected]${ANSI.reset}\r\n`);
      }
    }
    if (wasReconnect) {
      void refreshConversationSession(sessionId);
    }
    onReadyRef.current?.();
  }, [stdin, currentGen, sessionId]);

  const onTransportClose = useCallback(() => {
    sessionReadyRef.current = false;
    localEchoRef.current.reset();
    stdin.handleClose();
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
      eventId: string,
      source: string,
      stage: string,
      message?: string,
      backend?: string,
    ) => {
      if (!eventId || !source) return;
      transport.sendJson({
        type: "conversation_event_ack",
        eventId,
        source,
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
          const t = terminalRef.current;
          if (!t) break;
          if (inSnapshotRef.current) {
            // Snapshot bytes from the server — write verbatim. xterm
            // applies the encoded \x1bc reset, scrollback rows, and any
            // \x1b[?1049h alt-buffer enter so its state matches the
            // server's emulator state at subscribe time.
            t.write(msg.data);
          } else {
            const processed = localEchoRef.current.processOutput(msg.data);
            if (processed) t.write(processed);
            appendOutputProbe(sessionId, msg.data);
          }
          break;
        }
        case "history_end": {
          inSnapshotRef.current = false;
          // Re-enable local echo only when not in alt-buffer. xterm
          // exposes the active buffer; we read it after the snapshot has
          // been drained so the decision reflects post-replay state.
          const t = terminalRef.current;
          if (t) {
            const inAlt = t.buffer.active === t.buffer.alternate;
            localEchoRef.current.enabled = !inAlt;
          }
          break;
        }
        case "exit": {
          inSnapshotRef.current = false;
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
        case "resize_info": {
          break;
        }
      }
    });
    return unsubscribe;
  }, [transport, sessionId, stdin]);

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
      getTerminalDebugProbe().remove(sessionId);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Debug probe publishing. Updates on every state-bearing transition.
  useEffect(() => {
    const probe = getTerminalDebugProbe();
    const publish = () => {
      const t = terminalRef.current;
      probe.update({
        sessionId,
        connectionState: transport.state(),
        wsGen: transport.currentGen(),
        pendingInput: stdin.getPendingSnapshot().length,
        pendingAcks: 0,
        altBuffer: t ? t.buffer.active === t.buffer.alternate : false,
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
    subscribeInputSettled: stdin.subscribeInputSettled,
    subscribePendingInput: stdin.subscribePendingInput,
    getPendingInputSnapshot: stdin.getPendingSnapshot,
    sendConversationAck,
  };
}
