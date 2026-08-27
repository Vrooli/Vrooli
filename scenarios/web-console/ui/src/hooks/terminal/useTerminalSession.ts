import { useCallback, useEffect, useRef, useState } from "react";
import type { Terminal } from "@xterm/xterm";
import { buildSessionWsUrl } from "../../api/sessions";
import { refreshConversationSession } from "../../hooks/useConversationSession";
import { isMouseTrackingSequence } from "../../lib/terminalKeys";
import { createScrollController } from "../../lib/terminalScroll";
import { createPredictionOverlay, type PredictionOverlay } from "../../lib/predictionOverlay";
import { applyModifiers } from "../../consts/toolbar-keys";
import { useWorkspaceStore, type TerminalPaneStatus } from "../../stores/useWorkspaceStore";
import {
  createInputGate,
  type GateResult,
  type InputIntent,
  type TerminalInputGate,
} from "../../components/terminal/inputGate";
import { getTerminalDebugProbe } from "../../components/terminal/debug";
import { deviceIdentity } from "../../lib/deviceIdentity";
import type { TerminalMessage } from "../../types/terminal";
import { initialTerminalProtocolState, reduceTerminalMessage } from "../../lib/terminalProtocol";
import {
  useTerminalTransport,
  type SocketFactory,
} from "./useTerminalTransport";
import {
  useStdinStream,
  type InputSettledListener,
  type InputSettlementCallback,
  type PendingInputEntry,
} from "./useStdinStream";

const MAX_OUTPUT_PROBE_CHARS = 12000;
const TERMINAL_DEBUG_ENABLED = (() => {
  const env = (import.meta as unknown as { env?: Record<string, string | undefined> }).env;
  return env?.VITE_WC_TERMINAL_DEBUG === "1" || env?.VITE_WC_TERMINAL_DEBUG === "true";
})();

/** Maps known WS error messages to user-facing recovery hints. */
const WS_ERROR_RECOVERY: Record<string, string> = {
  "Invalid message format": "A malformed message was sent. This is usually harmless.",
  "Terminal process is not accepting input":
    "The terminal process has stopped. Close this pane and open a new terminal.",
  session_not_ready:
    "The terminal session did not confirm readiness in time. Reconnect or reopen this pane.",
};

export function appendOutputProbe(sessionId: string, data: string): void {
  if (!TERMINAL_DEBUG_ENABLED || typeof window === "undefined" || !data) return;
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
const RE_TERMINAL_RESPONSE = /\x1b\[[\x30-\x3f]*[\x20-\x2f]*[cnRypY]/g;
// eslint-disable-next-line no-control-regex -- intentionally matches OSC framing
const RE_OSC_COLOR_REPLY = /\x1b\][^\x07\x1b]*?rgb:[^\x07\x1b]*(?:\x07|\x1b\\)/g;
export function stripTerminalResponses(s: string): string {
  if (s.indexOf("\x1b") === -1) return s;
  return s.replace(RE_TERMINAL_RESPONSE, "").replace(RE_OSC_COLOR_REPLY, "");
}

export interface UseTerminalSessionOptions {
  sessionId: string;
  terminal: Terminal | null;
  onExit?: (sessionId: string) => void;
  onReady?: () => void;
  onStatus?: (status: TerminalPaneStatus | null) => void;
  predictionContainer?: HTMLElement | null;
  createSocket?: SocketFactory;
}

export type { TerminalPaneStatus } from "../../stores/useWorkspaceStore";

export interface UseTerminalSessionResult {
  /** Single-path input entry used by every UI source. */
  submitInput: (data: string, intent: Exclude<InputIntent, "control">) => GateResult;
  /** Send best-effort synthetic terminal bytes outside the reliable lane. */
  sendControl: (data: string) => boolean;
  /** Set tmux mouse capture for this persistent pane; null means unsupported. */
  setMouseMode: (enabled: boolean) => boolean;
  mouseMode: boolean | null;
  scrollBy: (lines: number, source: "touch" | "wheel" | "programmatic") => void;
  gate: TerminalInputGate;
  sendResize: (cols: number, rows: number) => void;
	getServerSize: () => { cols: number; rows: number } | null;
	serverSize: { cols: number; rows: number } | null;
	isFollower: boolean;
	leaderDevice: string;
	/** Leader-declared device family, used to frame a follower's view. */
	leaderClass: string;
	/** The leader's virtual keyboard covers part of its viewport. */
	leaderKbOpen: boolean;
	viewerCount: number;
	takeLease: () => void;
	/** Declare this client's virtual-keyboard state to the session. */
	setKeyboardOpen: (open: boolean) => void;
  subscribeInputSettled: (cb: InputSettledListener) => () => void;
  awaitOffset: (offset: number, cb: InputSettlementCallback) => () => void;
  subscribePendingInput: (cb: () => void) => () => void;
  getPendingInputSnapshot: () => readonly PendingInputEntry[];
  discardPendingInput: (index: number) => void;
  discardAllPendingInput: () => void;
  flushPendingInputNow: () => void;
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
 *   - useStdinStream      (cumulative-offset queue and reconnect replay)
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
  onStatus,
  predictionContainer,
  createSocket,
}: UseTerminalSessionOptions): UseTerminalSessionResult {
  const sessionReadyRef = useRef(false);
  const wsGenAtReadyRef = useRef(0);
  // True from terminal.reset() until history_end. While true, local-echo
  // stays disabled so the snapshot's alt-buffer markers / TUI paint render
  // cleanly.
  const inSnapshotRef = useRef(true);
	const outputCursorRef = useRef(0);
	const protocolStateRef = useRef(initialTerminalProtocolState);
  const predictionOverlayRef = useRef<PredictionOverlay | null>(null);
	const predictionSentAtRef = useRef(new Map<number, number>());
	const predictionLatencyEmaRef = useRef(0);
	const echoStateRef = useRef({ known: false, enabled: false, inAltBuffer: false, cursorAtLineEnd: false });
	const serverSizeRef = useRef<{ cols: number; rows: number } | null>(null);
	// This is deliberately separate from serverSizeRef. A follower renders the
	// leader's grid, but must retain its own most recently declared grid so an
	// explicit Take over can resize the PTY for the device that requested it.
	const declaredSizeRef = useRef<{ cols: number; rows: number } | null>(null);
	const leaseRequestInFlightRef = useRef(false);
  const [serverSize, setServerSize] = useState<{ cols: number; rows: number } | null>(null);
	const [mouseMode, setMouseMode] = useState<boolean | null>(null);
	const [holdsLease, setHoldsLease] = useState(true);
	const [leaderDevice, setLeaderDevice] = useState("");
	const [leaderClass, setLeaderClass] = useState("");
	const [leaderKbOpen, setLeaderKbOpen] = useState(false);
	const [viewerCount, setViewerCount] = useState(1);

  const onExitRef = useRef(onExit);
  const onReadyRef = useRef(onReady);
  onExitRef.current = onExit;
  onReadyRef.current = onReady;

  const wsUrl = buildSessionWsUrl(sessionId, typeof window === "undefined" ? undefined : deviceIdentity());

  // Transport is constructed first; the stdin stream and session layer
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
    (msg: TerminalMessage): boolean => transportRef.current?.sendReliableJson(msg) ?? false,
    [],
  );

  const stdin = useStdinStream({
    sendFrame,
    isSessionReady,
    onUnreconcilable: useCallback((offset: number) => {
      onStatus?.({
        kind: "input-desynced",
        detail: `Reliable input is out of sync at byte ${offset}. Reconnect or reopen this pane to recover.`,
      });
    }, [onStatus]),
  });

  const gate: TerminalInputGate = useRef(
    createInputGate({
      transport: {
        send: stdin.send,
        enqueue: stdin.enqueue,
      },
    }),
  ).current;

  // Hold the terminal in a ref so transport callbacks always see the latest
  // instance.
  const terminalRef = useRef<Terminal | null>(terminal);
  terminalRef.current = terminal;

  const onTransportOpen = useCallback((wasReconnect: boolean, _gen: number) => {
    sessionReadyRef.current = false;
		echoStateRef.current = { known: false, enabled: false, inAltBuffer: false, cursorAtLineEnd: false };
    stdin.resetForNewConnection(outputCursorRef.current);
    inSnapshotRef.current = !wasReconnect;

    const t = terminalRef.current;
    if (t && !wasReconnect) {
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
      declaredSizeRef.current = { cols: t.cols, rows: t.rows };
      transportRef.current?.sendJson({ type: "resize", cols: t.cols, rows: t.rows });
      predictionOverlayRef.current?.clear();
		predictionSentAtRef.current.clear();
		predictionLatencyEmaRef.current = 0;
    }
	if (t && wasReconnect) {
	  predictionOverlayRef.current?.clear();
	  predictionSentAtRef.current.clear();
	  if (onStatus) onStatus({ kind: "reconnected" });
	}
    if (wasReconnect) {
      void refreshConversationSession(sessionId);
    }
    onReadyRef.current?.();
  }, [onStatus, sessionId, stdin]);

  const onTransportClose = useCallback(() => {
    sessionReadyRef.current = false;
    stdin.handleClose();
    predictionOverlayRef.current?.clear();
	predictionSentAtRef.current.clear();
    onStatus?.({ kind: "disconnected" });
  }, [onStatus, stdin]);

  const transport = useTerminalTransport({
    url: wsUrl,
    createSocket,
    onOpen: onTransportOpen,
    onClose: onTransportClose,
  });
  transportRef.current = transport;

  const sendControl = useCallback(
    (data: string): boolean => transport.sendJson({ type: "control", data }),
    [transport],
  );

  const requestMouseMode = useCallback(
    (enabled: boolean): boolean => transport.sendJson({ type: "mouse_mode", data: enabled ? "on" : "off" }),
    [transport],
  );

  const scrollControllerRef = useRef<ReturnType<typeof createScrollController> | null>(null);
  if (scrollControllerRef.current === null) {
    scrollControllerRef.current = createScrollController(() => terminalRef.current, sendControl, {
      getSensitivity: (source) => {
        const state = useWorkspaceStore.getState();
        return source === "touch" ? state.touchScrollSensitivity : state.wheelScrollSensitivity;
      },
    });
  }
  const scrollController = scrollControllerRef.current;

  const requestLease = useCallback((explicit = false) => {
		if (!explicit && leaseRequestInFlightRef.current) return;
		// Frames on one WebSocket are ordered. Refresh our declaration first so
		// AcquireLease applies this device's grid even when it has only ever
		// rendered as a follower.
		const declared = declaredSizeRef.current;
		if (declared) transport.sendJson({ type: "resize", cols: declared.cols, rows: declared.rows });
    const sent = transport.sendJson({ type: "take_lease" });
		// Do not leave a follower stuck waiting when the tap happened during a
		// mobile reconnect and there was no open socket to carry the request.
		leaseRequestInFlightRef.current = sent;
  }, [transport]);

  // The visible Take over control is an explicit operator action. It must be
  // retryable even when a preceding automatic input request is awaiting a
  // server response (for example after a mobile reconnect).
  const takeLease = useCallback(() => requestLease(true), [requestLease]);

  // Presentational only: followers draw the leader's keyboard rather than
  // inferring it from a grid that shrinks for many reasons. Sent on change,
  // so the keyboard-animation polls upstream cost nothing.
  const keyboardOpenRef = useRef(false);
  const setKeyboardOpen = useCallback((open: boolean) => {
    if (keyboardOpenRef.current === open) return;
    keyboardOpenRef.current = open;
    transport.sendJson({ type: "device_state", kbOpen: open });
  }, [transport]);

  const submitInput = useCallback(
    (data: string, intent: Exclude<InputIntent, "control">): GateResult => {
      if (!holdsLease) requestLease();
      return gate.submit(data, intent);
    },
    [gate, holdsLease, requestLease],
  );

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
    // size_info and presence carry the same leader-presentation fields, so
    // they apply them through one function rather than two copies that drift.
    const applyLeaderPresentation = (msg: TerminalMessage) => {
      setLeaderDevice(msg.leaderDevice ?? "");
      setLeaderClass(msg.deviceClass ?? "");
      setLeaderKbOpen(msg.kbOpen === true);
      setViewerCount(msg.viewerCount ?? 1);
      useWorkspaceStore.getState().setViewerCount(sessionId, msg.viewerCount ?? 1);
    };
    const unsubscribe = transport.subscribe((msg) => {
	  protocolStateRef.current = reduceTerminalMessage(protocolStateRef.current, msg);
      switch (msg.type) {
        case "session_ready": {
          sessionReadyRef.current = true;
          wsGenAtReadyRef.current = msg.gen ?? transport.currentGen();
			setMouseMode(msg.mouse_mode_known ? msg.mouse_mode === true : null);
          stdin.reconcile(msg.accepted_through ?? 0);
          predictionOverlayRef.current?.retireThrough(msg.accepted_through ?? 0);
          stdin.replay();
          stdin.flush();
          break;
        }
        case "stdin_ack": {
          const acceptedThrough = msg.accepted_through ?? 0;
          const now = performance.now();
          for (const [offset, sentAt] of predictionSentAtRef.current) {
            if (offset <= acceptedThrough) {
              const sample = Math.max(0, now - sentAt);
              predictionLatencyEmaRef.current = predictionLatencyEmaRef.current === 0
                ? sample
                : predictionLatencyEmaRef.current * 0.75 + sample * 0.25;
              predictionSentAtRef.current.delete(offset);
            }
          }
          stdin.acknowledge(msg.accepted_through ?? 0, msg.ok === true, msg.reason);
		  if (msg.ok) {
			const active = terminalRef.current?.buffer.active;
			predictionOverlayRef.current?.retireThrough(
			  acceptedThrough,
			  active && typeof active.cursorX === "number" && typeof active.cursorY === "number"
				? { col: active.cursorX, row: active.cursorY }
				: undefined,
			);
		  } else {
			predictionOverlayRef.current?.clear();
		  }
          break;
        }
		case "echo_state": {
			const previous = echoStateRef.current;
			echoStateRef.current = {
				known: msg.echo_known === true,
				enabled: msg.echo_enabled === true,
				inAltBuffer: msg.in_alt_buffer === true,
				cursorAtLineEnd: msg.cursor_at_line_end === true,
			};
			if (
				previous.known !== echoStateRef.current.known ||
				previous.enabled !== echoStateRef.current.enabled ||
				previous.inAltBuffer !== echoStateRef.current.inAltBuffer ||
				previous.cursorAtLineEnd !== echoStateRef.current.cursorAtLineEnd
			) {
				predictionOverlayRef.current?.clear();
				predictionSentAtRef.current.clear();
			}
			break;
		}
		case "mouse_mode": {
			if (msg.data === "on" || msg.data === "off") setMouseMode(msg.data === "on");
			else if (msg.data === "unsupported") setMouseMode(null);
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
            appendOutputProbe(sessionId, msg.data);
            t.write(msg.data);
          }
		  if (typeof msg.output_cursor === "number") outputCursorRef.current = msg.output_cursor;
          scrollController.notifyOutput();
          break;
        }
        case "history_end": {
          inSnapshotRef.current = false;
		  if (typeof msg.output_cursor === "number") outputCursorRef.current = msg.output_cursor;
          break;
        }
        case "resync": {
          const t = terminalRef.current;
          if (t) {
            t.reset();
            t.clear();
          }
          predictionOverlayRef.current?.clear();
		  predictionSentAtRef.current.clear();
          inSnapshotRef.current = true;
          onStatus?.({ kind: "resynced" });
          break;
        }
        case "snapshot_notice": {
          onStatus?.({ kind: "resynced", detail: msg.data ?? "Scrollback was truncated for replay" });
          break;
        }
        case "exit": {
          inSnapshotRef.current = false;
          const code = msg.code ?? 0;
          onStatus?.({
            kind: "session-ended",
            detail: code === 0 ? "Session ended" : `Session ended with exit code ${code}`,
          });
          onExitRef.current?.(sessionId);
          break;
        }
        case "error": {
          const hint = WS_ERROR_RECOVERY[msg.data ?? ""] ?? "";
          onStatus?.({ kind: "error", detail: hint || msg.data || "Terminal error" });
          break;
        }
        case "sync_warning": {
          // Coalescing is an internal recovery signal. It is deliberately
          // not written into the emulator/xterm stream; a resync frame owns
          // recovery and pane chrome will own operator-facing status.
          break;
        }
        case "size_info": {
			if (!msg.cols || !msg.rows) break;
			serverSizeRef.current = { cols: msg.cols, rows: msg.rows };
			setServerSize(serverSizeRef.current);
			if (msg.holdsLease !== undefined) {
				const nextHoldsLease = msg.holdsLease === true;
				setHoldsLease(nextHoldsLease);
				if (nextHoldsLease || !leaseRequestInFlightRef.current) leaseRequestInFlightRef.current = false;
			}
			applyLeaderPresentation(msg);
			const t = terminalRef.current;
			if (t && (t.cols !== msg.cols || t.rows !== msg.rows)) t.resize(msg.cols, msg.rows);
			break;
		}
		case "presence": {
			const nextHoldsLease = msg.holdsLease === true;
			setHoldsLease(nextHoldsLease);
			applyLeaderPresentation(msg);
			if (nextHoldsLease || !leaseRequestInFlightRef.current) leaseRequestInFlightRef.current = false;
			break;
		}
		default:
			// Ignore forward-compatible message types without affecting the
			// terminal stream or reporting a spurious error.
			break;
      }
    });
    return unsubscribe;
  }, [onStatus, sessionId, stdin, transport]);

  useEffect(() => {
    if (!terminal || !predictionContainer) return;
    const overlay = createPredictionOverlay(terminal, predictionContainer);
    predictionOverlayRef.current = overlay;
    return () => {
      if (predictionOverlayRef.current === overlay) predictionOverlayRef.current = null;
      overlay.dispose();
    };
  }, [predictionContainer, terminal]);

  // xterm.onData → gate.submit (single path). Prediction is rendered as a
  // non-mutating overlay elsewhere; xterm remains the server-state renderer.
  useEffect(() => {
    if (!terminal) return;
    const disposable = terminal.onData((rawData) => {
	  if (isMouseTrackingSequence(rawData)) {
	    sendControl(rawData);
	    return;
	  }
      const mods = useWorkspaceStore.getState().modifiers;
      const hasModifier = mods.ctrl || mods.alt || mods.shift;
      let data = stripTerminalResponses(rawData);
      if (data.length === 0) return;
      if (hasModifier) {
        const { data: modified } = applyModifiers(data, mods);
        data = modified;
        useWorkspaceStore.getState().clearModifiers();
      }
	  const result = submitInput(data, "typing");
      const currentTerminal = terminalRef.current;
      if (
        result.status === "sent" &&
        data.length === 1 &&
        data >= " " &&
        data !== "\x7f" &&
		  currentTerminal?.buffer.active === currentTerminal?.buffer.normal &&
			echoStateRef.current.known &&
			echoStateRef.current.enabled &&
			echoStateRef.current.cursorAtLineEnd
      ) {
        if (!currentTerminal) return;
        const active = currentTerminal.buffer.active;
		predictionSentAtRef.current.set(result.offset, performance.now());
		const threshold = useWorkspaceStore.getState().predictionLatencyThresholdMs;
		predictionOverlayRef.current?.add(
		  data,
		  active.cursorX,
		  active.cursorY,
		  result.offset,
		  predictionLatencyEmaRef.current > threshold,
		);
      }
    });
    return () => {
      disposable.dispose();
    };
	}, [terminal, sendControl, submitInput]);

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
        pendingReliableInput: stdin.getPendingInputCount(),
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
			if (cols <= 0 || rows <= 0) return;
			declaredSizeRef.current = { cols, rows };
      transport.sendJson({ type: "resize", cols, rows });
    },
    [transport],
  );

	const getServerSize = useCallback(() => serverSizeRef.current, []);

  return {
    submitInput,
    sendControl,
    setMouseMode: requestMouseMode,
    mouseMode,
    scrollBy: scrollController.scrollBy,
    gate,
		sendResize,
		getServerSize,
		serverSize,
		isFollower: !holdsLease,
		leaderDevice,
		leaderClass,
		leaderKbOpen,
		viewerCount,
		takeLease,
		setKeyboardOpen,
    subscribeInputSettled: stdin.subscribeInputSettled,
    awaitOffset: stdin.awaitOffset,
    subscribePendingInput: stdin.subscribePendingInput,
    getPendingInputSnapshot: stdin.getPendingSnapshot,
    discardPendingInput: stdin.discardEntry,
    discardAllPendingInput: stdin.discardAll,
    flushPendingInputNow: stdin.flushNow,
    sendConversationAck,
  };
}
