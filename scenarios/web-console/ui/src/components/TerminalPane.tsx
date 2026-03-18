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
import TerminalContextMenu from "./TerminalContextMenu";
import { useTextToSpeech } from "../hooks/useTextToSpeech";
import { useMobileBackspaceRepeat } from "../hooks/useMobileBackspaceRepeat";
import { waitForTerminalCandidateMatch } from "../lib/terminalTtsMatch";

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
}

// [REQ:P0-002d] xterm.js Terminal Rendering
const TerminalPane = forwardRef<TerminalPaneHandle, TerminalPaneProps>(
  function TerminalPane({ sessionId, onExit, onReady, onVoiceStart, onVoiceStop, onTtsSpeakingChange }, ref) {
    const containerRef = useRef<HTMLDivElement>(null);
    const fitRef = useRef<FitAddon | null>(null);
    const serializeRef = useRef<SerializeAddon | null>(null);
    const cachedOffsetRef = useRef<number | undefined>(undefined);
    const hadCacheRef = useRef(false);
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
    const { speak, speakParagraphs, stop: ttsStop, supported: ttsSupported, error: ttsError, backend, isSpeaking: ttsSpeaking } = useTextToSpeech(ttsSettings, {
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

    // Auto-TTS: correlate routed TTS candidates against this terminal's
    // rendered buffer, then play them if they match.
    const autoTtsEnabled = useWorkspaceStore((s) => s.autoTtsEnabled);

    const handleTTSCandidate = useCallback(async (
      candidate: { eventId: string; source: string; text: string },
      sendAck: (stage: string, message?: string, backend?: string) => void,
    ) => {
      sendAck("received");
      if (!autoTtsEnabled) {
        sendAck("rejected", "Auto-TTS is disabled in this tab");
        return;
      }
      if (!ttsSupported || !terminal) {
        sendAck("rejected", "No TTS backend is available in this tab");
        return;
      }
      if (!await waitForTerminalCandidateMatch(terminal, candidate.text)) {
        sendAck("rejected", "Assistant text did not match the rendered terminal buffer");
        return;
      }
      sendAck("correlated");
      ttsStop();
      // Split on double-newlines first; if that yields very long blocks,
      // sub-split on single newlines so utterances stay manageable.
      const raw = candidate.text.split(/\n\n+/).filter((p) => p.trim());
      const paragraphs: string[] = [];
      for (const block of raw) {
        if (block.length > 500) {
          paragraphs.push(...block.split(/\n/).filter((l) => l.trim()));
        } else {
          paragraphs.push(block);
        }
      }
      sendAck("playback_started", undefined, backend);
      try {
        const usedBackend = await speakParagraphs(paragraphs);
        sendAck("playback_succeeded", undefined, usedBackend ?? backend);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Speech failed";
        sendAck("playback_failed", message, backend);
      }
    }, [autoTtsEnabled, backend, speakParagraphs, terminal, ttsStop, ttsSupported]);

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
      onTTSCandidate: handleTTSCandidate,
    });

    // Expose sendInput + focus for parent components (mobile toolbar, launcher shortcuts)
    useImperativeHandle(ref, () => ({
      sendInput,
      focus: () => terminal?.focus(),
      stopTts: ttsStop,
    }), [sendInput, terminal, ttsStop]);

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
      if (selection) speak(selection);
      setContextMenu(null);
    }, [terminal, speak]);

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
        className={`h-full w-full min-h-[200px] relative p-1${dragOver ? " ring-2 ring-inset ring-blue-400/60" : ""}`}
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
