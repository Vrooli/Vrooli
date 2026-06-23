// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState, forwardRef, type DragEvent, type ClipboardEvent } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";
import type { GateResult, InputSource } from "./terminal/inputGate";
import { TERMINAL_SCROLLBACK_LINES } from "../lib/terminalConfig";
import { useTerminalTouch } from "../hooks/useTerminalTouch";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { TERMINAL_THEMES, DEFAULT_THEME_ID, TERMINAL_FONT_FAMILY, TERMINAL_FONT_SIZE } from "../consts/config";
import { parseShortcut, matchesShortcut } from "../lib/shortcutParser";
import { useImageUpload } from "../hooks/useImageUpload";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import TerminalContextMenu from "./TerminalContextMenu";
import { useTextToSpeech } from "../hooks/useTextToSpeech";
import { useMobileBackspaceRepeat } from "../hooks/useMobileBackspaceRepeat";
import { useConversationSession } from "../hooks/useConversationSession";
import { useConversationStore } from "../stores/useConversationStore";
import type { ConversationEvent } from "../api/conversation";
import type { TTSPlaybackState } from "../audio-integration";

const EMPTY_CONVERSATION_EVENTS: ConversationEvent[] = [];
const EMPTY_CONVERSATION_CURSOR = { lastSeenSequence: 0, lastListenedSequence: 0 } as const;

// Max age of an assistant event that may still trigger auto-TTS. Live events
// arrive within milliseconds; this excludes a backlog replayed by the global
// SSE channel after a reconnect from being read aloud all at once.
const AUTO_TTS_MAX_AGE_MS = 60_000;

/**
 * Backend synthesis limit in **bytes** (UTF-8). The Go backend checks
 * `len(req.Input)` which is byte length, so we must measure the same way.
 */
const TTS_MAX_CHUNK_BYTES = 4500;

const textEncoder = new TextEncoder();

/** Return UTF-8 byte length of a string (matches Go's `len(s)`). */
function utf8ByteLength(s: string): number {
  return textEncoder.encode(s).length;
}

/**
 * Ensures every paragraph fits within the TTS synthesis byte limit.
 * This is a defensive fallback for events that lack backend-computed
 * speechParagraphs (e.g. cached from an older server version).
 *
 * Measures UTF-8 byte length (not JS string length) because the backend
 * validates with Go's len() which counts bytes. Unicode-heavy content
 * (box-drawing, em dashes) can be 3 bytes per character.
 */
function ensureSpeechChunks(paragraphs: string[]): string[] {
  const result: string[] = [];
  for (const p of paragraphs) {
    if (utf8ByteLength(p) <= TTS_MAX_CHUNK_BYTES) {
      result.push(p);
    } else {
      // Split at word boundaries, measuring byte length
      let remaining = p;
      while (utf8ByteLength(remaining) > TTS_MAX_CHUNK_BYTES) {
        // Binary-search for the split point: find the largest prefix
        // that fits within the byte limit, cutting at a word boundary.
        let lo = 0;
        let hi = remaining.length;
        while (lo < hi) {
          const mid = (lo + hi + 1) >>> 1;
          if (utf8ByteLength(remaining.slice(0, mid)) <= TTS_MAX_CHUNK_BYTES) {
            lo = mid;
          } else {
            hi = mid - 1;
          }
        }
        // lo = max number of chars whose byte length fits.
        // Prefer splitting at a space.
        const spaceAt = remaining.lastIndexOf(" ", lo);
        const cut = spaceAt > 0 ? spaceAt : lo;
        result.push(remaining.slice(0, cut).trim());
        remaining = remaining.slice(cut).trim();
      }
      if (remaining) result.push(remaining);
    }
  }
  return result.filter(Boolean);
}

interface TerminalPaneProps {
  sessionId: string;
  onExit?: (sessionId: string) => void;
  onReady?: () => void;
  onVoiceStart?: () => void;
  onVoiceStop?: () => void;
  /** Called when TTS speaking state changes for this pane. */
  onTtsSpeakingChange?: (speaking: boolean) => void;
  /** Called when the currently-speaking conversation event changes (for summarize controls). */
  onSpeakingEventChange?: (eventId: string | null) => void;
  onConversationEventReceived?: (
    sessionId: string,
    event: ConversationEvent,
    sendAck: (stage: string, message?: string, backend?: string) => void,
  ) => void;
  /**
   * Called when auto-TTS playback is rejected by the browser's autoplay
   * policy and no user gesture has unlocked the audio element yet. Pass
   * `null` to clear (when the condition resolves or the user dismisses).
   * Consumers should render a persistent "Enable voice" affordance and
   * call `enable()` on click — a successful return replays pending events.
   */
  onNeedsUnlock?: (payload: { sessionId: string; enable: () => Promise<boolean> } | null) => void;
}

// [REQ:P0-007b] Terminal Key/Chord Mapping - expose input injection
export interface TerminalPaneHandle {
  /**
   * Submit data to the terminal via the single input gate. Returns a
   * typed GateResult: `sent` (with seq), `queued` (with reason),
   * or `rejected`. Callers wishing to display status (MobileToolbar)
   * inspect the result's status/reason; callers that don't care
   * (Workspace's simple forwarders) can ignore it.
   */
  submitInput: (data: string, source: InputSource) => GateResult;
  /** Focus the xterm.js terminal element. */
  focus: () => void;
  /** Stop TTS playback for this pane. */
  stopTts: () => void;
  /** Stop current TTS, then speak a single text (optionally pre-chunked). */
  speakText: (text: string, paragraphs?: string[], opts?: { eventId?: string; version?: "active" | "original"; initiatedBy?: "auto" | "manual" }) => Promise<string | undefined>;
  /** Pause TTS playback. */
  pauseTts: () => void;
  /** Resume paused TTS playback. */
  resumeTts: () => void;
  /** Seek to a position in seconds. */
  seekTts: (seconds: number) => void;
  /** Set the TTS playback speed. */
  setTtsPlaybackRate: (rate: number) => void;
  /** Set the TTS volume (0\u20131). */
  setTtsVolume: (level: number) => void;
  /** Set TTS muted state (independent of volume). */
  setTtsMuted: (next: boolean) => void;
  /** Return a snapshot of the current TTS playback state, or null. */
  getTtsState: () => TTSPlaybackState | null;
  /**
   * Subscribe to per-send settlement results. The callback fires with
   * `(seq, ok)` once the server's stdin_ack arrives (ok=true) or the
   * client's 2 s timeout elapses (ok=false). Unsubscribes via the returned
   * cleanup function. Used by MobileToolbar to delay clearing the draft
   * until the send is actually confirmed.
   */
  subscribeInputSettled: (cb: (seq: number, ok: boolean) => void) => () => void;
  /** Subscribe to queue-changed notifications for the pending-input pill. */
  subscribePendingInput: (cb: () => void) => () => void;
  /** Snapshot of currently queued (unsent) input payloads. */
  getPendingInputSnapshot: () => readonly { data: string; addedAt: number }[];
}

// [REQ:P0-002d] xterm.js Terminal Rendering
const TerminalPane = forwardRef<TerminalPaneHandle, TerminalPaneProps>(
  function TerminalPane({ sessionId, onExit, onReady, onVoiceStart, onVoiceStop, onTtsSpeakingChange, onSpeakingEventChange, onNeedsUnlock, onConversationEventReceived }, ref) {
    const { t } = useTranslation();
    const containerRef = useRef<HTMLDivElement>(null);
    const fitRef = useRef<FitAddon | null>(null);
    // Last cols/rows actually sent to the server. Used to suppress
    // resize WS messages when the container resize observer fires but
    // the terminal's reflowed dimensions are unchanged. Without this,
    // a viewport tick (mobile keyboard, visualViewport change) fans out
    // to every mounted TerminalPane and emits identical resizes for
    // each one — visible as a `resize_noop` storm in api logs.
    const lastSentSizeRef = useRef<{ cols: number; rows: number } | null>(null);
    const [terminal, setTerminal] = useState<Terminal | null>(null);

    // Per-pane selectors with fallbacks for old persisted data
    const paneFontSize = useWorkspaceStore(
      useCallback((s) => s.panes.find((p) => p.sessionId === sessionId)?.fontSize ?? TERMINAL_FONT_SIZE, [sessionId]),
    );
    const paneThemeId = useWorkspaceStore(
      useCallback((s) => s.panes.find((p) => p.sessionId === sessionId)?.themeId ?? DEFAULT_THEME_ID, [sessionId]),
    );
    const paneTheme = useMemo(() => {
      const fallback = { background: "#0f172a", foreground: "#e2e8f0", cursor: "#38bdf8" } as const;
      return TERMINAL_THEMES[paneThemeId]?.colors ?? TERMINAL_THEMES[DEFAULT_THEME_ID]?.colors ?? fallback;
    }, [paneThemeId]);

    const ttsVoice = useWorkspaceStore((s) => s.ttsVoice);
    const ttsRate = useWorkspaceStore((s) => s.ttsRate);
    const ttsPitch = useWorkspaceStore((s) => s.ttsPitch);
    const kokoroVoice = useWorkspaceStore((s) => s.kokoroVoice);
    const kokoroSpeed = useWorkspaceStore((s) => s.kokoroSpeed);
    const ttsBackendPreference = useWorkspaceStore((s) => s.ttsBackendPreference);
    const ttsSettings = useMemo(
      () => ({
        voice: ttsVoice,
        rate: ttsRate,
        pitch: ttsPitch,
        kokoroVoice,
        kokoroSpeed,
        backendPreference: ttsBackendPreference,
      }),
      [ttsVoice, ttsRate, ttsPitch, kokoroVoice, kokoroSpeed, ttsBackendPreference],
    );
    const {
      speakParagraphs, stop: ttsStop,
      pause: ttsPause, resume: ttsResume, seek: ttsSeek,
      setPlaybackRate: ttsSetPlaybackRate, setVolume: ttsSetVolume,
      setMuted: ttsSetMuted,
      getPlaybackState: ttsGetPlaybackState,
      supported: ttsSupported, isSpeaking: ttsSpeaking,
      needsUnlock, unlockAudio,
    } = useTextToSpeech(ttsSettings, {
      source: "terminal_auto",
      sessionId,
    });

    // Notify parent when TTS speaking state changes.
    // Use a ref for the callback to avoid re-firing the effect when the
    // inline arrow prop changes identity on every Workspace render.
    const onTtsSpeakingChangeRef = useRef(onTtsSpeakingChange);
    onTtsSpeakingChangeRef.current = onTtsSpeakingChange;
    useEffect(() => {
      onTtsSpeakingChangeRef.current?.(ttsSpeaking);
    }, [ttsSpeaking]);

    const activePane = useWorkspaceStore((s) => s.activePane);
    const { persistCursor } = useConversationSession(sessionId);
    const conversationSession = useConversationStore((state) => state.sessions[sessionId]);
    const conversationEvents = conversationSession?.events ?? EMPTY_CONVERSATION_EVENTS;
    const conversationCursor = conversationSession?.cursor ?? EMPTY_CONVERSATION_CURSOR;
    const conversationHydrated = conversationSession?.hydrated ?? false;

    useEffect(() => {
      if (activePane !== sessionId) return;
      const latestSequence = conversationEvents[conversationEvents.length - 1]?.sequence;
      if (latestSequence && latestSequence > conversationCursor.lastSeenSequence) {
        void persistCursor({ lastSeenSequence: latestSequence });
      }
    }, [activePane, conversationCursor.lastSeenSequence, conversationEvents, persistCursor, sessionId]);

    // Bridge needsUnlock → parent. When unlock is required we hand the parent
    // an `enable` callback that unlocks + replays pending events on success.
    // Pass `null` to clear when the condition resolves.
    const onNeedsUnlockRef = useRef(onNeedsUnlock);
    onNeedsUnlockRef.current = onNeedsUnlock;
    useEffect(() => {
      if (!needsUnlock) {
        onNeedsUnlockRef.current?.(null);
        return;
      }
      const enable = async (): Promise<boolean> => {
        const ok = await unlockAudio();
        return ok;
      };
      onNeedsUnlockRef.current?.({ sessionId, enable });
    }, [needsUnlock, sessionId, unlockAudio]);

    // Delegate all WebSocket protocol handling to the session hook
    const { submitInput, sendResize, subscribeInputSettled, subscribePendingInput, getPendingInputSnapshot, sendConversationAck } = useTerminalSession({
      sessionId,
      terminal,
      onExit,
      onReady: () => {
        // After socket connects and snapshot replay completes, re-fit to
        // ensure dimensions match this client's actual container size.
        requestAnimationFrame(() => {
          const fit = fitRef.current;
          if (!fit || !terminal) return;
          fit.fit();
          if (terminal.cols > 0 && terminal.rows > 0) {
            sendResize(terminal.cols, terminal.rows);
            lastSentSizeRef.current = { cols: terminal.cols, rows: terminal.rows };
          }
        });
        onReady?.();
      },
    });

    // Auto-TTS is now driven from the conversation store rather than the
    // terminal WS (conversation events arrive via the global SSE channel).
    // We hold a per-session baseline at the tail seen when the session first
    // hydrates so pre-existing history never auto-plays; only assistant events
    // that land AFTER the baseline, while THIS pane is active, trigger playback.
    // A recency gate prevents an SSE reconnect from auto-reading a backlog of
    // missed messages. Playback telemetry acks still go over this pane's WS.
    const autoTtsBaselineRef = useRef<number | null>(null);
    useEffect(() => {
      if (!conversationHydrated) return;
      const maxSeq = conversationEvents.length > 0
        ? (conversationEvents[conversationEvents.length - 1]?.sequence ?? 0)
        : 0;
      const baseline = autoTtsBaselineRef.current;
      // First hydrate, or this pane isn't the active one: just advance the
      // baseline so nothing pre-existing (or arriving in the background)
      // auto-plays — and switching to this pane later won't replay it.
      if (baseline === null || activePane !== sessionId) {
        autoTtsBaselineRef.current = maxSeq;
        return;
      }
      if (maxSeq <= baseline) return;
      autoTtsBaselineRef.current = maxSeq;
      if (!ttsSupported || !onConversationEventReceived) return;
      // Auto-play only the newest fresh assistant event; the controller's own
      // !isSpeaking gate handles the rest of any burst.
      let latest: ConversationEvent | undefined;
      for (let i = conversationEvents.length - 1; i >= 0; i--) {
        const candidate = conversationEvents[i];
        if (!candidate || candidate.sequence <= baseline) break;
        if (candidate.role === "assistant") { latest = candidate; break; }
      }
      if (!latest) return;
      const ageMs = Date.now() - new Date(latest.createdAt).getTime();
      if (Number.isFinite(ageMs) && ageMs > AUTO_TTS_MAX_AGE_MS) return;
      const event = latest;
      onConversationEventReceived(sessionId, event, (stage, message, backend) =>
        sendConversationAck(event.id, event.source, stage, message, backend),
      );
    }, [conversationHydrated, conversationEvents, activePane, sessionId, ttsSupported, onConversationEventReceived, sendConversationAck]);

    // Pending-input draft round-trip. Offscreen terminals are unmounted to keep
    // cost flat in N; without this, anything typed-but-not-yet-sent would be
    // lost when the pane unmounts. On unmount we stash the unsent input; on the
    // next mount we re-inject it (the gate queues it until the WS is ready).
    const setPendingInputDraft = useWorkspaceStore((s) => s.setPendingInputDraft);
    const consumePendingInputDraft = useWorkspaceStore((s) => s.consumePendingInputDraft);
    const getPendingInputSnapshotRef = useRef(getPendingInputSnapshot);
    getPendingInputSnapshotRef.current = getPendingInputSnapshot;
    const submitInputRef = useRef(submitInput);
    submitInputRef.current = submitInput;
    useEffect(() => {
      const draft = consumePendingInputDraft(sessionId);
      if (draft) submitInputRef.current(draft, "toolbar-submit");
      return () => {
        const text = getPendingInputSnapshotRef.current()
          .map((entry) => entry.data)
          .join("");
        if (text) setPendingInputDraft(sessionId, text);
      };
    }, [sessionId, consumePendingInputDraft, setPendingInputDraft]);

    // Expose submitInput + focus for parent components (mobile toolbar, launcher shortcuts)
    useImperativeHandle(ref, () => ({
      submitInput,
      focus: () => terminal?.focus(),
      stopTts: () => {
        ttsStop();
        onSpeakingEventChange?.(null);
      },
      speakText: (text: string, paragraphs?: string[], opts?: { eventId?: string; version?: "active" | "original"; initiatedBy?: "auto" | "manual" }) => {
        if (opts?.initiatedBy !== "auto") {
          ttsSetMuted(false);
        }
        ttsStop();
        onSpeakingEventChange?.(opts?.eventId ?? null);
        return speakParagraphs(ensureSpeechChunks(paragraphs ?? [text]), opts).finally(() => {
          onSpeakingEventChange?.(null);
        });
      },
      pauseTts: ttsPause,
      resumeTts: () => {
        ttsSetMuted(false);
        ttsResume();
      },
      seekTts: ttsSeek,
      setTtsPlaybackRate: ttsSetPlaybackRate,
      setTtsVolume: ttsSetVolume,
      setTtsMuted: ttsSetMuted,
      getTtsState: ttsGetPlaybackState,
      subscribeInputSettled,
      subscribePendingInput,
      getPendingInputSnapshot,
    }), [submitInput, terminal, ttsStop, speakParagraphs, ttsPause, ttsResume, ttsSeek, ttsSetPlaybackRate, ttsSetVolume, ttsSetMuted, ttsGetPlaybackState, onSpeakingEventChange, subscribeInputSettled, subscribePendingInput, getPendingInputSnapshot]);

    const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null);

    const { hasSelection, copySelection, clearSelection } = useTerminalTouch({
      terminal,
      containerRef,
      submitInput,
      onContextMenu: useCallback((x: number, y: number) => {
        setContextMenu({ x, y });
      }, []),
    });

    // Enable hold-to-delete on mobile virtual keyboards (see hook for details).
    useMobileBackspaceRepeat(terminal);

    const closeContextMenu = useCallback(() => {
      // Clear the selection when dismissing the menu so it doesn't
      // immediately reopen (the selection triggers the context menu).
      clearSelection();
      setContextMenu(null);
    }, [clearSelection]);

    const handleCtxCopy = useCallback(() => {
      copySelection();
      clearSelection();
      setContextMenu(null);
    }, [copySelection, clearSelection]);

    const handleCtxPaste = useCallback((text: string): Promise<
      { status: "ok" } | { status: "failed"; reason: string }
    > => {
      // Submit the paste via the input gate. If the gate sends it
      // immediately (status === "sent") we await the server's
      // stdin_ack before resolving. If queued or rejected, resolve
      // synchronously with the gate's reason so the UI can react
      // without holding the menu open forever.
      const result = submitInput(text, "paste");
      terminal?.focus();
      if (result.status === "rejected") {
        return Promise.resolve({
          status: "failed",
          reason: result.reason,
        });
      }
      if (result.status === "queued") {
        // The payload will be sent once session_ready / mouse-tracking
        // clears. We cannot observe that settlement here without a
        // seq (the gate didn't produce one). Report queued as success
        // to the user — the bytes will be delivered.
        return Promise.resolve({ status: "ok" });
      }
      // result.status === "sent": wait for the matching stdin_ack.
      const { seq } = result;
      return new Promise((resolve) => {
        const unsub = subscribeInputSettled((ackSeq, ok) => {
          if (ackSeq !== seq) return;
          unsub();
          if (ok) {
            resolve({ status: "ok" });
          } else {
            resolve({ status: "failed", reason: "server rejected" });
          }
        });
      });
    }, [submitInput, subscribeInputSettled, terminal]);

    const handleCtxSelectAll = useCallback(() => {
      terminal?.selectAll();
      setContextMenu(null);
    }, [terminal]);

    const handleCtxClear = useCallback(() => {
      terminal?.clear();
      setContextMenu(null);
    }, [terminal]);

    const handleCtxSpeak = useCallback(() => {
      const selection = terminal?.getSelection();
      if (selection) void speakParagraphs(ensureSpeechChunks([selection]));
      setContextMenu(null);
    }, [terminal, speakParagraphs]);

    // Image upload support
    const { uploadAndInject, uploading, error: uploadError } = useImageUpload(sessionId, submitInput);
    const fileInputRef = useRef<HTMLInputElement>(null);
    const [dragOver, setDragOver] = useState(false);

    const handleCtxUploadImage = useCallback(() => {
      setContextMenu(null);
      fileInputRef.current?.click();
    }, []);

    const handleFileInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
      const f = e.target.files?.[0];
      if (f) uploadAndInject(f);
      e.target.value = "";
    }, [uploadAndInject]);

    const handlePaste = useCallback((e: ClipboardEvent) => {
      const items = e.clipboardData?.items;
      if (!items) return;
      for (const item of items) {
        if (item.type.startsWith("image/")) {
          e.preventDefault();
          const blob = item.getAsFile();
          if (blob) uploadAndInject(blob);
          return;
        }
      }
      // No image found — let xterm.js handle text paste
    }, [uploadAndInject]);

    const handleDragOver = useCallback((e: DragEvent) => {
      e.preventDefault();
      setDragOver(true);
    }, []);

    const handleDragLeave = useCallback(() => {
      setDragOver(false);
    }, []);

    const handleDrop = useCallback((e: DragEvent) => {
      e.preventDefault();
      setDragOver(false);
      const files = e.dataTransfer?.files;
      if (!files) return;
      for (const file of files) {
        if (file.type.startsWith("image/")) {
          uploadAndInject(file);
        }
      }
    }, [uploadAndInject]);

    const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
    const { syncPaneUpdate } = useWorkspaceSync();

    // Initialize xterm.js terminal (rendering only)
    useEffect(() => {
      const container = containerRef.current;
      if (!container) return;

      const term = new Terminal({
        cursorBlink: true,
        fontSize: paneFontSize,
        fontFamily: TERMINAL_FONT_FAMILY,
        theme: paneTheme,
        allowProposedApi: true,
        scrollback: TERMINAL_SCROLLBACK_LINES,
      });

      const fitAddon = new FitAddon();
      const webLinksAddon = new WebLinksAddon();
      term.loadAddon(fitAddon);
      term.loadAddon(webLinksAddon);

      term.open(container);
      fitAddon.fit();

      // === Mobile virtual-keyboard suppression ===
      // On mobile, xterm.js focus() focuses a hidden <textarea>, which the
      // browser interprets as "user wants to type" and opens the virtual
      // keyboard. But many focus() calls are programmatic (toolbar button
      // presses, paste, etc.) where the user does NOT intend to type.
      //
      // Setting inputMode="none" on xterm's textarea tells the browser
      // "this element accepts focus but should not trigger the keyboard."
      // We default to "none" and only flip to "" (show keyboard) in
      // useTerminalTouch.ts when the user taps the terminal directly —
      // the clearest signal of typing intent.
      //
      // The blur listener re-arms the guard after every focus/blur cycle,
      // so the next programmatic focus() won't re-open the keyboard.
      const isTouchDevice = "ontouchstart" in window || navigator.maxTouchPoints > 0;
      const xtermTextarea = term.textarea;
      if (isTouchDevice && xtermTextarea) {
        xtermTextarea.inputMode = "none";
      }

      const handleXtermBlur = () => {
        // Re-arm keyboard suppression: when the terminal loses focus,
        // reset inputMode to "none" so the next focus() call (likely
        // programmatic) won't trigger the keyboard. Only a direct tap
        // in useTerminalTouch.ts will flip this back to "" before focusing.
        if (xtermTextarea) xtermTextarea.inputMode = "none";
      };
      if (isTouchDevice && xtermTextarea) {
        xtermTextarea.addEventListener("blur", handleXtermBlur);
      }

      // Listen for title changes from OSC escape sequences (e.g., from Claude Code, vim, ssh)
      const titleDisposable = term.onTitleChange((title) => {
        renamePaneById(sessionId, title);
        syncPaneUpdate(sessionId, { name: title });
      });

      fitRef.current = fitAddon;
      setTerminal(term);

      return () => {
        if (isTouchDevice && xtermTextarea) {
          xtermTextarea.removeEventListener("blur", handleXtermBlur);
        }
        titleDisposable.dispose();
        term.dispose();
        fitRef.current = null;
        setTerminal(null);
      };
      // eslint-disable-next-line react-hooks/exhaustive-deps -- Initial font size only; font updates handled by separate effect
    }, []);

    // Fit terminal while preserving scroll position when the user is scrolled
    // up in history. xterm.js Buffer.resize() adjusts ydisp to keep the cursor
    // visible, which yanks users back to the bottom if they were reading
    // scrollback. We save the offset-from-bottom before fit() and restore it.
    const scrollAwareFit = useCallback(() => {
      const fit = fitRef.current;
      if (!fit || !terminal) return;

      const buf = terminal.buffer.active;
      const offsetFromBottom = buf.baseY - buf.viewportY;

      fit.fit();

      // Restore scroll position only if user was scrolled up (not at bottom)
      if (offsetFromBottom > 0) {
        const newBuf = terminal.buffer.active;
        const targetY = Math.max(0, newBuf.baseY - offsetFromBottom);
        const drift = targetY - newBuf.viewportY;
        if (drift !== 0) {
          terminal.scrollLines(drift);
        }
      }
    }, [terminal]);

    // React to font size changes from store
    useEffect(() => {
      if (!terminal || !fitRef.current) return;
      terminal.options.fontSize = paneFontSize;
      scrollAwareFit();
      const last = lastSentSizeRef.current;
      if (!last || last.cols !== terminal.cols || last.rows !== terminal.rows) {
        sendResize(terminal.cols, terminal.rows);
        lastSentSizeRef.current = { cols: terminal.cols, rows: terminal.rows };
      }
    }, [paneFontSize, terminal, sendResize, scrollAwareFit]);

    // React to theme changes from store
    useEffect(() => {
      if (!terminal) return;
      terminal.options.theme = paneTheme;
    }, [paneTheme, terminal]);

    // Intercept configurable voice shortcut via capture-phase DOM listener.
    // attachCustomKeyEventHandler fires too late — the browser processes
    // shortcuts like Alt+Space before xterm's handler runs. A capture-phase
    // listener on the container intercepts early enough to preventDefault().
    const voiceShortcut = useWorkspaceStore((s) => s.voiceShortcut);
    useEffect(() => {
      const container = containerRef.current;
      if (!container || !onVoiceStart || !onVoiceStop) return;
      const parsed = parseShortcut(voiceShortcut);
      if (!parsed) return;

      const handler = (event: KeyboardEvent) => {
        if (!matchesShortcut(event, parsed)) return;
        event.preventDefault();
        event.stopPropagation();
        if (event.type === "keydown") onVoiceStart();
        if (event.type === "keyup") onVoiceStop();
      };

      container.addEventListener("keydown", handler, { capture: true });
      container.addEventListener("keyup", handler, { capture: true });
      return () => {
        container.removeEventListener("keydown", handler, { capture: true });
        container.removeEventListener("keyup", handler, { capture: true });
      };
    }, [onVoiceStart, onVoiceStop, voiceShortcut]);

    // Handle container resize -> fit terminal -> notify server.
    // Throttled via requestAnimationFrame to avoid flooding the WebSocket
    // with resize messages during continuous drag/resize operations.
    useEffect(() => {
      const container = containerRef.current;
      if (!container || !terminal || !fitRef.current) return;

      let rafId: number | null = null;
      const resizeObserver = new ResizeObserver(() => {
        if (rafId !== null) return; // Already scheduled
        rafId = requestAnimationFrame(() => {
          rafId = null;
          scrollAwareFit();
          const last = lastSentSizeRef.current;
          if (!last || last.cols !== terminal.cols || last.rows !== terminal.rows) {
            sendResize(terminal.cols, terminal.rows);
            lastSentSizeRef.current = { cols: terminal.cols, rows: terminal.rows };
          }
        });
      });
      resizeObserver.observe(container);

      return () => {
        resizeObserver.disconnect();
        if (rafId !== null) cancelAnimationFrame(rafId);
      };
    }, [terminal, sendResize, scrollAwareFit]);

    return (
      <div
        ref={containerRef}
        data-testid="terminal-pane"
        data-session-id={sessionId}
        // overflow-hidden is critical: xterm.js manages its own scrolling via
        // an internal .xterm-viewport element (overflow-y: scroll). Without
        // clipping overflow here, the browser creates a SECOND native scrollbar
        // on this container (or an ancestor) once the terminal buffer grows
        // large enough for xterm's rendered DOM to exceed the container bounds.
        // That phantom outer scrollbar captures touch/wheel events on mobile,
        // making the terminal unscrollable unless the user carefully avoids it.
        className={`h-full w-full overflow-hidden relative p-1${dragOver ? " ring-2 ring-inset ring-blue-400/60" : ""}`}
        onPasteCapture={handlePaste}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        {uploading && (
          <div data-testid="upload-overlay" className="absolute inset-0 z-20 flex items-center justify-center bg-black/50 text-sm text-white">
            {t(strings.terminalPane.uploadingImage)}
          </div>
        )}
        {uploadError && (
          <div data-testid="upload-error" className="absolute top-2 left-2 z-20 rounded bg-red-600/90 px-3 py-1.5 text-xs text-white shadow-lg">
            {uploadError}
          </div>
        )}
        {contextMenu && (
          <TerminalContextMenu
            position={contextMenu}
            hasSelection={hasSelection}
            onCopy={handleCtxCopy}
            onPaste={handleCtxPaste}
            onSelectAll={handleCtxSelectAll}
            onClear={handleCtxClear}
            onUploadImage={handleCtxUploadImage}
            onSpeak={ttsSupported ? handleCtxSpeak : undefined}
            onClose={closeContextMenu}
          />
        )}
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          hidden
          onChange={handleFileInputChange}
        />
      </div>
    );
  },
);

export default TerminalPane;
