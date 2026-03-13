// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useEffect, useRef, useImperativeHandle, forwardRef, useState, useCallback, useMemo } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { useTerminalSocket } from "../hooks/useTerminalSocket";
import { useTerminalTouch } from "../hooks/useTerminalTouch";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { TERMINAL_THEMES, DEFAULT_THEME_ID, TERMINAL_FONT_FAMILY, TERMINAL_FONT_SIZE } from "../consts/config";
import { parseShortcut, matchesShortcut } from "../lib/shortcutParser";

interface TerminalPaneProps {
  sessionId: string;
  onExit?: (sessionId: string) => void;
  onReady?: () => void;
  onVoiceStart?: () => void;
  onVoiceStop?: () => void;
}

// [REQ:P0-007b] Terminal Key/Chord Mapping - expose input injection
export interface TerminalPaneHandle {
  /** Send data to the terminal. Returns true if sent immediately, false if queued. */
  sendInput: (data: string) => boolean;
  /** Focus the xterm.js terminal element. */
  focus: () => void;
}

// [REQ:P0-002d] xterm.js Terminal Rendering
const TerminalPane = forwardRef<TerminalPaneHandle, TerminalPaneProps>(
  function TerminalPane({ sessionId, onExit, onReady, onVoiceStart, onVoiceStop }, ref) {
    const containerRef = useRef<HTMLDivElement>(null);
    const fitRef = useRef<FitAddon | null>(null);
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

    // Delegate all WebSocket protocol handling to the socket hook
    const { sendInput, sendResize } = useTerminalSocket({
      sessionId,
      terminal,
      onExit,
      onReady,
    });

    // Expose sendInput + focus for parent components (mobile toolbar, launcher shortcuts)
    useImperativeHandle(ref, () => ({
      sendInput,
      focus: () => terminal?.focus(),
    }), [sendInput, terminal]);

    const { hasSelection, copySelection, clearSelection } = useTerminalTouch({
      terminal,
      containerRef,
    });

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
      const webLinksAddon = new WebLinksAddon();
      term.loadAddon(fitAddon);
      term.loadAddon(webLinksAddon);

      term.open(container);
      fitAddon.fit();

      // Listen for title changes from OSC escape sequences (e.g., from Claude Code, vim, ssh)
      const titleDisposable = term.onTitleChange((title) => {
        renamePaneById(sessionId, title);
      });

      fitRef.current = fitAddon;
      setTerminal(term);

      return () => {
        titleDisposable.dispose();
        term.dispose();
        fitRef.current = null;
        setTerminal(null);
      };
      // eslint-disable-next-line react-hooks/exhaustive-deps -- Initial font size only; font updates handled by separate effect
    }, []);

    // React to font size changes from store
    useEffect(() => {
      if (!terminal || !fitRef.current) return;
      terminal.options.fontSize = paneFontSize;
      fitRef.current.fit();
      sendResize(terminal.cols, terminal.rows);
    }, [paneFontSize, terminal, sendResize]);

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
      const fit = fitRef.current;
      if (!container || !terminal || !fit) return;

      let rafId: number | null = null;
      const resizeObserver = new ResizeObserver(() => {
        if (rafId !== null) return; // Already scheduled
        rafId = requestAnimationFrame(() => {
          rafId = null;
          fit.fit();
          sendResize(terminal.cols, terminal.rows);
        });
      });
      resizeObserver.observe(container);

      return () => {
        resizeObserver.disconnect();
        if (rafId !== null) cancelAnimationFrame(rafId);
      };
    }, [terminal, sendResize]);

    return (
      <div
        ref={containerRef}
        data-testid="terminal-pane"
        data-session-id={sessionId}
        className="h-full w-full min-h-[200px] relative p-1"
      >
        {hasSelection && (
          <button
            data-testid="touch-copy-btn"
            className="absolute top-2 right-2 z-10 rounded bg-[rgb(var(--wc-accent))] px-3 py-1.5 text-xs font-medium text-slate-900 shadow-lg active:opacity-80"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={() => {
              copySelection();
              clearSelection();
            }}
          >
            Copy
          </button>
        )}
      </div>
    );
  },
);

export default TerminalPane;
