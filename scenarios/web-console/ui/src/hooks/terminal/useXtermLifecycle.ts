import { useCallback, useEffect, useRef, useState } from "react";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Terminal, type ITheme } from "@xterm/xterm";
import { TERMINAL_FONT_FAMILY } from "../../consts/config";
import { TERMINAL_SCROLLBACK_LINES } from "../../lib/terminalConfig";
import { scrollTerminalLines } from "../../lib/terminalScroll";

export interface XtermLifecycleOptions {
  sessionId: string;
  paneFontSize: number;
  paneTheme: ITheme;
  wheelScrollSensitivity: number;
  sendResize: (cols: number, rows: number) => void;
  getServerSize: () => { cols: number; rows: number } | null;
  isFollower: () => boolean;
  renamePaneById: (sessionId: string, name: string) => void;
  syncPaneUpdate: (sessionId: string, patch: { name?: string }) => void;
}

function maybeSendResize(
  terminal: Terminal,
  sendResize: (cols: number, rows: number) => void,
  lastSentSizeRef: { current: { cols: number; rows: number } | null },
  getServerSize: () => { cols: number; rows: number } | null,
): void {
  const last = lastSentSizeRef.current;
  const authoritative = getServerSize();
  const dimensionsChanged = !last || last.cols !== terminal.cols || last.rows !== terminal.rows;
  const serverDiffers = !authoritative || authoritative.cols !== terminal.cols || authoritative.rows !== terminal.rows;
  if (serverDiffers && dimensionsChanged) {
    sendResize(terminal.cols, terminal.rows);
    lastSentSizeRef.current = { cols: terminal.cols, rows: terminal.rows };
  }
}

/** Window a burst of terminal title changes collapses into one pane write. */
const TITLE_SYNC_DEBOUNCE_MS = 750;

export async function waitForTerminalFont(fontSize: number): Promise<void> {
  if (typeof document !== "undefined" && "fonts" in document) {
    await document.fonts.load(`${String(fontSize)}px ${TERMINAL_FONT_FAMILY}`);
  }
}

export function useXtermLifecycle(options: XtermLifecycleOptions) {
  const {
    sessionId,
    paneFontSize,
    paneTheme,
    wheelScrollSensitivity,
    sendResize,
    getServerSize,
    isFollower,
    renamePaneById,
    syncPaneUpdate,
  } = options;
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalHostRef = useRef<HTMLDivElement>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const lastSentSizeRef = useRef<{ cols: number; rows: number } | null>(null);
  const pendingTitleRef = useRef<string | null>(null);
  const syncedTitleRef = useRef<string | null>(null);
  const titleFlushRef = useRef<number | null>(null);
  const [terminal, setTerminal] = useState<Terminal | null>(null);
  const [paneSize, setPaneSize] = useState({ width: 0, height: 0 });

  useEffect(() => {
    const terminalHost = terminalHostRef.current;
    if (!terminalHost) return;
    const term = new Terminal({
      cursorBlink: true,
      fontSize: paneFontSize,
      fontFamily: TERMINAL_FONT_FAMILY,
      theme: paneTheme,
      allowProposedApi: true,
      scrollback: TERMINAL_SCROLLBACK_LINES,
      scrollSensitivity: wheelScrollSensitivity,
    });
    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.open(terminalHost);
    let disposed = false;
    const fitAfterFontLoad = async () => {
      await waitForTerminalFont(paneFontSize);
      if (!disposed) fitAddon.fit();
    };
    void fitAfterFontLoad();

    const isTouchDevice = "ontouchstart" in window || navigator.maxTouchPoints > 0;
    const xtermTextarea = term.textarea;
    if (isTouchDevice && xtermTextarea) xtermTextarea.inputMode = "none";
    const handleXtermBlur = () => {
      if (xtermTextarea) xtermTextarea.inputMode = "none";
    };
    if (isTouchDevice && xtermTextarea) xtermTextarea.addEventListener("blur", handleXtermBlur);
    // The title is a stream, not an event: a shell rewrites it on every
    // prompt and a working agent rewrites it continuously, which turned one
    // idle pane into a steady run of pane writes — each a network round trip.
    // The local rename stays immediate so the tab label keeps up; only the
    // persisted copy is coalesced, and an unchanged title never travels.
    const flushTitle = () => {
      titleFlushRef.current = null;
      const next = pendingTitleRef.current;
      if (next === null || next === syncedTitleRef.current) return;
      syncedTitleRef.current = next;
      syncPaneUpdate(sessionId, { name: next });
    };
    const titleDisposable = term.onTitleChange((title) => {
      renamePaneById(sessionId, title);
      pendingTitleRef.current = title;
      // Trailing-edge only: while titles keep arriving the window keeps the
      // pending value and one write lands after they stop.
      if (titleFlushRef.current !== null) return;
      titleFlushRef.current = window.setTimeout(flushTitle, TITLE_SYNC_DEBOUNCE_MS);
    });
    fitRef.current = fitAddon;
    setTerminal(term);
    return () => {
      disposed = true;
      if (isTouchDevice && xtermTextarea) xtermTextarea.removeEventListener("blur", handleXtermBlur);
      titleDisposable.dispose();
      if (titleFlushRef.current !== null) {
        window.clearTimeout(titleFlushRef.current);
        titleFlushRef.current = null;
      }
      term.dispose();
      fitRef.current = null;
      setTerminal(null);
    };
    // The initial options are intentionally captured; later updates have their
    // own effects so xterm is not reconstructed on every store change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const scrollAwareFit = useCallback(() => {
    const fit = fitRef.current;
    if (!fit || !terminal) return;
    const buffer = terminal.buffer.active;
    const offsetFromBottom = buffer.baseY - buffer.viewportY;
    fit.fit();
    if (offsetFromBottom > 0) {
      const next = terminal.buffer.active;
      const targetY = Math.max(0, next.baseY - offsetFromBottom);
      const drift = targetY - next.viewportY;
      scrollTerminalLines(terminal, drift);
    }
  }, [terminal]);

  useEffect(() => {
    if (terminal) terminal.options.scrollSensitivity = wheelScrollSensitivity;
  }, [terminal, wheelScrollSensitivity]);

  useEffect(() => {
    if (!terminal || !fitRef.current || isFollower()) return;
    terminal.options.fontSize = paneFontSize;
    scrollAwareFit();
    maybeSendResize(terminal, sendResize, lastSentSizeRef, getServerSize);
  }, [paneFontSize, terminal, isFollower, sendResize, scrollAwareFit, getServerSize]);

  useEffect(() => {
    if (!terminal) return;
    terminal.options.theme = paneTheme;
  }, [paneTheme, terminal]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !terminal || !fitRef.current) return;
    let rafId: number | null = null;
    const resizeObserver = new ResizeObserver(() => {
      if (rafId !== null) return;
      rafId = requestAnimationFrame(() => {
        rafId = null;
        const nextSize = { width: container.clientWidth, height: container.clientHeight };
        setPaneSize((previous) => previous.width === nextSize.width && previous.height === nextSize.height ? previous : nextSize);
        if (isFollower()) return;
        scrollAwareFit();
        maybeSendResize(terminal, sendResize, lastSentSizeRef, getServerSize);
      });
    });
    setPaneSize({ width: container.clientWidth, height: container.clientHeight });
    resizeObserver.observe(container);
    // ResizeObserver's first notification is asynchronous. Fit once in the
    // current layout as well, otherwise the accessibility capture can observe
    // xterm's default 80-column screen before the observer callback runs.
    if (!isFollower()) {
      scrollAwareFit();
      maybeSendResize(terminal, sendResize, lastSentSizeRef, getServerSize);
    }
    return () => {
      resizeObserver.disconnect();
      if (rafId !== null) cancelAnimationFrame(rafId);
    };
  }, [terminal, sendResize, scrollAwareFit, getServerSize, isFollower]);

  return { containerRef, terminalHostRef, fitRef, terminal, paneSize, scrollAwareFit };
}
