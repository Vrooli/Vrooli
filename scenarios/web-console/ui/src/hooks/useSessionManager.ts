import { useState, useCallback, useRef, useEffect } from "react";
import { createSession, deleteSession, listSessions, toErrorInfo, type SessionInfo, type ErrorInfo } from "../lib/api";
import { DEFAULT_COLS, DEFAULT_ROWS, ERROR_AUTO_DISMISS_MS } from "../consts/config";
import type { TerminalPaneHandle } from "../components/TerminalPane";

// DOC: docs/concepts/ARCHITECTURE.md#session-creation
// DOC: docs/internal/SEAMS.md#1b-session-orchestration
/**
 * ── STABLE CORE: Session lifecycle orchestration. ──
 * Owns pane state, session creation/deletion, pending-command
 * bookkeeping, and terminal ref management. The Workspace component
 * delegates all session logic here and focuses on layout only.
 *
 * Change axes that touch this hook:
 *   - Session creation parameters (cols/rows defaults, shell)
 *   - Session policy controls (P1-001)
 *   - Error semantics (new error codes from the API)
 *
 * [REQ:P0-001b] Independent Pane Session Lifecycle
 * [REQ:P0-006a] Terminal Launch Flow UI
 */

export interface PaneState {
  session: SessionInfo;
}

export function useSessionManager() {
  const [panes, setPanes] = useState<PaneState[]>([]);
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState<ErrorInfo | null>(null);
  const [isHydrated, setIsHydrated] = useState(false);
  const terminalRefs = useRef<Map<string, TerminalPaneHandle>>(new Map());
  const pendingCommands = useRef<Map<string, string>>(new Map());
  const dismissTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const flushPendingCommand = useCallback((sessionId: string) => {
    const command = pendingCommands.current.get(sessionId);
    if (!command) return;
    const handle = terminalRefs.current.get(sessionId);
    if (!handle) return;
    pendingCommands.current.delete(sessionId);
    handle.sendInput(command + "\n");
  }, []);

  // Hydrate workspace panes from existing sessions so reload reconnects to
  // durable sessions instead of showing the empty-state landing screen.
  useEffect(() => {
    let canceled = false;

    const hydratePanes = async () => {
      try {
        const sessions = await listSessions();
        if (canceled) return;
        if (sessions.length > 0) {
          setPanes((prev) => (prev.length > 0 ? prev : sessions.map((session) => ({ session }))));
        }
      } catch {
        // Best-effort hydration: keep empty-state on API/list failures.
      } finally {
        if (!canceled) setIsHydrated(true);
      }
    };

    void hydratePanes();
    return () => {
      canceled = true;
    };
  }, []);

  // Clear the auto-dismiss timer on unmount to prevent setState on dead component
  useEffect(() => {
    return () => {
      if (dismissTimerRef.current !== null) {
        clearTimeout(dismissTimerRef.current);
      }
    };
  }, []);

  const clearError = useCallback(() => {
    if (dismissTimerRef.current !== null) {
      clearTimeout(dismissTimerRef.current);
      dismissTimerRef.current = null;
    }
    setCreateError(null);
  }, []);

  // Guard against concurrent creation requests (e.g., rapid double-click).
  // The `isCreating` state flag drives the UI (button disable), while
  // `createInFlight` prevents the handler itself from executing twice.
  const createInFlight = useRef(false);

  // [REQ:P0-006a] Terminal Launch Flow UI
  const launchSession = useCallback(async (command?: string) => {
    // Replay guard: if a creation is already in-flight, skip silently.
    if (createInFlight.current) return null;
    createInFlight.current = true;
    setIsCreating(true);
    setCreateError(null);
    try {
      const session = await createSession({ cols: DEFAULT_COLS, rows: DEFAULT_ROWS });
      if (command) {
        pendingCommands.current.set(session.id, command);
      }
      setPanes((prev) => [...prev, { session }]);
      return session;
    } catch (err) {
      console.error("Failed to create session:", err);
      setCreateError(toErrorInfo(err));
      // Auto-dismiss error; timer is cleaned up on unmount via dismissTimerRef
      if (dismissTimerRef.current !== null) {
        clearTimeout(dismissTimerRef.current);
      }
      dismissTimerRef.current = setTimeout(() => {
        dismissTimerRef.current = null;
        setCreateError(null);
      }, ERROR_AUTO_DISMISS_MS);
      return null;
    } finally {
      createInFlight.current = false;
      setIsCreating(false);
    }
  }, []);

  const handleTerminalReady = useCallback((sessionId: string) => {
    flushPendingCommand(sessionId);
  }, [flushPendingCommand]);

  const removePane = useCallback(async (sessionId: string) => {
    setPanes((prev) => prev.filter((p) => p.session.id !== sessionId));
    terminalRefs.current.delete(sessionId);
    try {
      await deleteSession(sessionId);
    } catch {
      // Session may already be dead
    }
  }, []);

  const handleExit = useCallback((sessionId: string) => {
    console.log(`Session ${sessionId} exited`);
  }, []);

  const sendToActiveTerminal = useCallback(
    (data: string, targetId?: string) => {
      const target = targetId ?? panes[panes.length - 1]?.session.id;
      if (target) {
        const handle = terminalRefs.current.get(target);
        if (handle) {
          handle.sendInput(data);
        }
      }
    },
    [panes],
  );

  const registerTerminalRef = useCallback(
    (sessionId: string, handle: TerminalPaneHandle | null) => {
      if (handle) {
        terminalRefs.current.set(sessionId, handle);
        // onReady can fire before the ref callback in some mount orders.
        // Flush here too so queued launch commands are never stranded.
        flushPendingCommand(sessionId);
      } else {
        terminalRefs.current.delete(sessionId);
      }
    },
    [flushPendingCommand],
  );

  return {
    panes,
    isHydrated,
    isCreating,
    createError,
    clearError,
    launchSession,
    handleTerminalReady,
    removePane,
    handleExit,
    sendToActiveTerminal,
    registerTerminalRef,
  };
}
