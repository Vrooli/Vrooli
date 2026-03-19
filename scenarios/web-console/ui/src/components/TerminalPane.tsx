// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState, forwardRef, type DragEvent, type ClipboardEvent } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SerializeAddon } from "@xterm/addon-serialize";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { useTerminalSocket } from "../hooks/useTerminalSocket";
import { loadTerminalCache, saveTerminalCache } from "../lib/terminalCache";
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
import type { ConversationEvent } from "../lib/api";
import type { TTSPlaybackState } from "../hooks/tts/types";

const EMPTY_CONVERSATION_EVENTS: ConversationEvent[] = [];
const EMPTY_CONVERSATION_CURSOR = { lastSeenSequence: 0, lastListenedSequence: 0 } as const;

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
}

// [REQ:P0-007b] Terminal Key/Chord Mapping - expose input injection
export interface TerminalPaneHandle {
  /** Send data to the terminal. Returns true if sent immediately, false if queued. */
  sendInput: (data: string) => boolean;
  /** Focus the xterm.js terminal element. */
  focus: () => void;
  /** Stop TTS playback for this pane. */
  stopTts: () => void;
  /** Stop current TTS, then speak a single text (optionally pre-chunked). */
  speakText: (text: string, paragraphs?: string[]) => void;
  /** Stop current TTS, then speak texts sequentially, calling onProgress(i) before each. */
  speakSequence: (texts: string[], onProgress: (index: number) => void) => Promise<void>;
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
  /** Return a snapshot of the current TTS playback state, or null. */
  getTtsState: () => TTSPlaybackState | null;
}

// [REQ:P0-002d] xterm.js Terminal Rendering
const TerminalPane = forwardRef<TerminalPaneHandle, TerminalPaneProps>(
  function TerminalPane({ sessionId, onExit, onReady, onVoiceStart, onVoiceStop, onTtsSpeakingChange }, ref) {
    const containerRef = useRef<HTMLDivElement>(null);
    const fitRef = useRef<FitAddon | null>(null);
    const serializeRef = useRef<SerializeAddon | null>(null);
    const cachedOffsetRef = useRef<number | undefined>(undefined);
    const hadCacheRef = useRef(false);
    const livePlaybackEventRef = useRef<string | null>(null);
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
      getPlaybackState: ttsGetPlaybackState,
      supported: ttsSupported, error: ttsError, backend, isSpeaking: ttsSpeaking,
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

    // Auto-dismiss TTS error after 5 seconds
    const [showTtsError, setShowTtsError] = useState<string | null>(null);
    useEffect(() => {
      if (!ttsError) {
        setShowTtsError(null);
        return;
      }
      setShowTtsError(ttsError);
      const timer = setTimeout(() => setShowTtsError(null), 5000);
      return () => clearTimeout(timer);
    }, [ttsError]);

    const autoTtsEnabled = useWorkspaceStore((s) => s.autoTtsEnabled);
    const activePane = useWorkspaceStore((s) => s.activePane);
    const { appendConversationEvent, persistCursor } = useConversationSession(sessionId);
    const conversationSession = useConversationStore((state) => state.sessions[sessionId]);
    const conversationEvents = conversationSession?.events ?? EMPTY_CONVERSATION_EVENTS;
    const conversationCursor = conversationSession?.cursor ?? EMPTY_CONVERSATION_CURSOR;

    const handleConversationEvent = useCallback(async (
      event: { id: string; source: string; role: "assistant"; text: string; speechParagraphs?: string[]; createdAt?: string; sequence: number },
      sendAck: (stage: string, message?: string, backend?: string) => void,
    ) => {
      appendConversationEvent({
        id: event.id,
        sessionId,
        source: event.source,
        role: event.role,
        text: event.text,
        speechParagraphs: event.speechParagraphs ?? [event.text],
        createdAt: event.createdAt ?? new Date().toISOString(),
        sequence: event.sequence,
        deliveryState: "received",
        ttsState: "idle",
        consumptionState: "unseen",
      } satisfies ConversationEvent);
      sendAck("received");
      const isActivePane = activePane === sessionId;
      if (isActivePane) {
        void persistCursor({ lastSeenSequence: event.sequence });
        sendAck("seen");
      }
      if (!autoTtsEnabled || !isActivePane) {
        return;
      }
      if (!ttsSupported) {
        sendAck("rejected", "No TTS backend is available in this tab");
        return;
      }
      livePlaybackEventRef.current = event.id;
      ttsStop();
      const paragraphs = ensureSpeechChunks(event.speechParagraphs ?? [event.text]);
      sendAck("playback_started", undefined, backend);
      try {
        const usedBackend = await speakParagraphs(paragraphs);
        sendAck("playback_succeeded", undefined, usedBackend ?? backend);
        await persistCursor({ lastListenedSequence: event.sequence, lastSeenSequence: event.sequence });
      } catch (err) {
        const message = err instanceof Error ? err.message : "Speech failed";
        sendAck("playback_failed", message, backend);
      } finally {
        if (livePlaybackEventRef.current === event.id) {
          livePlaybackEventRef.current = null;
        }
      }
    }, [activePane, appendConversationEvent, autoTtsEnabled, backend, persistCursor, sessionId, speakParagraphs, ttsStop, ttsSupported]);

    useEffect(() => {
      if (activePane !== sessionId) return;
      const latestSequence = conversationEvents[conversationEvents.length - 1]?.sequence;
      if (latestSequence && latestSequence > conversationCursor.lastSeenSequence) {
        void persistCursor({ lastSeenSequence: latestSequence });
      }
    }, [activePane, conversationCursor.lastSeenSequence, conversationEvents, persistCursor, sessionId]);

    useEffect(() => {
      if (activePane !== sessionId || !autoTtsEnabled || !ttsSupported) return;
      const pending = conversationEvents.filter((event) =>
        event.role === "assistant" && event.sequence > conversationCursor.lastListenedSequence && event.id !== livePlaybackEventRef.current,
      );
      if (pending.length === 0 || ttsSpeaking) return;

      let cancelled = false;
      const playPending = async () => {
        for (const event of pending) {
          if (cancelled) return;
          ttsStop();
          const paragraphs = ensureSpeechChunks(event.speechParagraphs ?? [event.text]);
          try {
            await speakParagraphs(paragraphs);
            await persistCursor({ lastListenedSequence: event.sequence, lastSeenSequence: event.sequence });
          } catch {
            return
          }
        }
      };
      void playPending();
      return () => {
        cancelled = true;
      };
    }, [activePane, autoTtsEnabled, backend, conversationCursor.lastListenedSequence, conversationEvents, persistCursor, sessionId, speakParagraphs, ttsSpeaking, ttsStop, ttsSupported]);

    // Delegate all WebSocket protocol handling to the socket hook
    const { sendInput, sendResize, totalBytesRef } = useTerminalSocket({
      sessionId,
      terminal,
      onExit,
      onReady: () => {
        // After socket connects and replay completes, re-fit to ensure
        // dimensions match this client's actual container size.
        requestAnimationFrame(() => {
          const fit = fitRef.current;
          if (!fit || !terminal) return;
          fit.fit();
          if (terminal.cols > 0 && terminal.rows > 0) {
            sendResize(terminal.cols, terminal.rows);
          }
        });
        onReady?.();
      },
      historyOffset: cachedOffsetRef.current,
      hasCachedState: hadCacheRef.current,
      onConversationEvent: handleConversationEvent,
    });

    // Expose sendInput + focus for parent components (mobile toolbar, launcher shortcuts)
    useImperativeHandle(ref, () => ({
      sendInput,
      focus: () => terminal?.focus(),
      stopTts: ttsStop,
      speakText: (text: string, paragraphs?: string[]) => {
        ttsStop();
        void speakParagraphs(ensureSpeechChunks(paragraphs ?? [text]));
      },
      speakSequence: async (texts: string[], onProgress: (index: number) => void) => {
        ttsStop();
        onProgress(0);
        const allChunks = texts.flatMap((t) => t ? ensureSpeechChunks([t]) : []);
        await speakParagraphs(allChunks);
      },
      pauseTts: ttsPause,
      resumeTts: ttsResume,
      seekTts: ttsSeek,
      setTtsPlaybackRate: ttsSetPlaybackRate,
      setTtsVolume: ttsSetVolume,
      getTtsState: ttsGetPlaybackState,
    }), [sendInput, terminal, ttsStop, speakParagraphs, ttsPause, ttsResume, ttsSeek, ttsSetPlaybackRate, ttsSetVolume, ttsGetPlaybackState]);

    const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null);

    const { hasSelection, copySelection, clearSelection } = useTerminalTouch({
      terminal,
      containerRef,
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

    const handleCtxPaste = useCallback((text: string) => {
      sendInput(text);
      setContextMenu(null);
      terminal?.focus();
    }, [sendInput, terminal]);

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
    const { uploadAndInject, uploading, error: uploadError } = useImageUpload(sessionId, sendInput);
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
      });

      const fitAddon = new FitAddon();
      const serializeAddon = new SerializeAddon();
      const webLinksAddon = new WebLinksAddon();
      term.loadAddon(fitAddon);
      term.loadAddon(serializeAddon);
      term.loadAddon(webLinksAddon);

      term.open(container);
      fitAddon.fit();

      // Restore cached terminal state for instant visual display on refresh.
      // The cache entry includes the byte offset so useTerminalSocket can
      // request only delta output from the server.
      // DOC: docs/concepts/ARCHITECTURE.md#terminal-history-caching
      const cached = loadTerminalCache(sessionId);
      if (cached) {
        term.write(cached.serialized);
        cachedOffsetRef.current = cached.totalBytes;
        hadCacheRef.current = true;
      } else {
        cachedOffsetRef.current = undefined;
        hadCacheRef.current = false;
      }

      serializeRef.current = serializeAddon;

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
        serializeRef.current = null;
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
      sendResize(terminal.cols, terminal.rows);
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

    // Save terminal state to sessionStorage on visibility change (tab
    // backgrounded) and beforeunload (page refresh). On next mount the
    // cached state is written to xterm instantly, and the byte offset is
    // sent to the server for delta-only history replay.
    // DOC: docs/concepts/ARCHITECTURE.md#terminal-history-caching
    useEffect(() => {
      if (!terminal || !serializeRef.current) return;

      const save = () => {
        const addon = serializeRef.current;
        if (!addon) return;
        const serialized = addon.serialize();
        saveTerminalCache(sessionId, {
          serialized,
          totalBytes: totalBytesRef.current,
          savedAt: Date.now(),
        });
      };

      const handleVisibilityChange = () => {
        if (document.visibilityState === "hidden") save();
      };
      const handleBeforeUnload = () => save();

      document.addEventListener("visibilitychange", handleVisibilityChange);
      window.addEventListener("beforeunload", handleBeforeUnload);
      return () => {
        save(); // Final save on unmount
        document.removeEventListener("visibilitychange", handleVisibilityChange);
        window.removeEventListener("beforeunload", handleBeforeUnload);
      };
    }, [terminal, sessionId, totalBytesRef]);

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
          sendResize(terminal.cols, terminal.rows);
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
            Uploading image…
          </div>
        )}
        {uploadError && (
          <div data-testid="upload-error" className="absolute top-2 left-2 z-20 rounded bg-red-600/90 px-3 py-1.5 text-xs text-white shadow-lg">
            {uploadError}
          </div>
        )}
        {showTtsError && (
          <div data-testid="tts-error" className="absolute top-2 right-2 z-20 rounded bg-amber-600/90 px-3 py-1.5 text-xs text-white shadow-lg">
            TTS: {showTtsError}
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
