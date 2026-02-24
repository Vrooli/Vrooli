// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useEffect, useRef, useImperativeHandle, forwardRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { useTerminalSocket } from "../hooks/useTerminalSocket";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { TERMINAL_THEME, TERMINAL_FONT_FAMILY } from "../consts/config";

interface TerminalPaneProps {
  sessionId: string;
  onExit?: (sessionId: string) => void;
  onReady?: () => void;
}

// [REQ:P0-007b] Terminal Key/Chord Mapping - expose input injection
export interface TerminalPaneHandle {
  sendInput: (data: string) => void;
}

// [REQ:P0-002d] xterm.js Terminal Rendering
const TerminalPane = forwardRef<TerminalPaneHandle, TerminalPaneProps>(
  function TerminalPane({ sessionId, onExit, onReady }, ref) {
    const containerRef = useRef<HTMLDivElement>(null);
    const fitRef = useRef<FitAddon | null>(null);
    const [terminal, setTerminal] = useState<Terminal | null>(null);
    const terminalFontSize = useWorkspaceStore((s) => s.terminalFontSize);

    // Delegate all WebSocket protocol handling to the socket hook
    const { sendInput, sendResize } = useTerminalSocket({
      sessionId,
      terminal,
      onExit,
      onReady,
    });

    // Expose sendInput for parent components (mobile toolbar, launcher shortcuts)
    useImperativeHandle(ref, () => ({ sendInput }), [sendInput]);

    const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);

    // Initialize xterm.js terminal (rendering only)
    useEffect(() => {
      const container = containerRef.current;
      if (!container) return;

      const term = new Terminal({
        cursorBlink: true,
        fontSize: terminalFontSize,
        fontFamily: TERMINAL_FONT_FAMILY,
        theme: TERMINAL_THEME,
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
      terminal.options.fontSize = terminalFontSize;
      fitRef.current.fit();
      sendResize(terminal.cols, terminal.rows);
    }, [terminalFontSize, terminal, sendResize]);

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
        className="h-full w-full min-h-[200px]"
      />
    );
  },
);

export default TerminalPane;
