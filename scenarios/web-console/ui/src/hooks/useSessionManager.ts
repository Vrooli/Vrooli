import { useState, useCallback, useRef, useEffect } from "react";
import { toErrorInfo, type ErrorInfo } from "../lib/errors";
import { getWorkspaceLayout, updateWorkspacePane } from "../api/workspace";
import { createSession, deleteSession, listSessions, type SessionInfo, type BackendID, type PolicyMode, type AgentType } from "../api/sessions";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { DEFAULT_COLS, DEFAULT_ROWS, ERROR_AUTO_DISMISS_MS } from "../consts/config";
import type { TerminalPaneHandle } from "../components/TerminalPane";
import type { GateResult, InputSource } from "../components/terminal/inputGate";
import { ttsPlaybackRegistry } from "../audio-integration";

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
  supportsMessagesView: boolean;
}

function supportsMessagesViewForCommand(command?: string): boolean {
  return agentTypeFromCommand(command) !== "none";
}

// agentTypeFromCommand classifies the launch command into the closed
// AgentType set persisted server-side. Used to populate the create-session
// payload so the recovery flow can later reattach the agent. Exported for unit
// coverage of the classification table.
export function agentTypeFromCommand(command?: string): AgentType {
  if (!command) return "none";
  const trimmed = command.trim().toLowerCase();
  if (trimmed === "claude" || trimmed.startsWith("claude ")) return "claude";
  if (trimmed === "codex" || trimmed.startsWith("codex ")) return "codex";
  if (trimmed === "opencode" || trimmed.startsWith("opencode ")) return "opencode";
  if (trimmed === "grok" || trimmed.startsWith("grok ")) return "grok";
  return "none";
}

export function useSessionManager() {
  const [panes, setPanes] = useState<PaneState[]>([]);
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState<ErrorInfo | null>(null);
  const [hydrationError, setHydrationError] = useState<ErrorInfo | null>(null);
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
    handle.submitInput(command + "\n", "toolbar-submit");
  }, []);

  // Hydrate workspace panes from existing sessions so reload reconnects to
  // durable sessions instead of showing the empty-state landing screen.
  useEffect(() => {
    let canceled = false;

    const hydratePanes = async () => {
      try {
        const [sessionsResult, layoutResult] = await Promise.allSettled([
          listSessions(),
          getWorkspaceLayout(),
        ]);

        const sessions = sessionsResult.status === "fulfilled" ? sessionsResult.value : [];
        const layout = layoutResult.status === "fulfilled" ? layoutResult.value : null;

        if (canceled) return;

        // Surface per-call failures. Logging always (so a single-call
        // regression is visible without devtools-on-mobile); the UI banner
        // fires only when both calls fail, because a single failure usually
        // means a degraded surface rather than a totally broken hydration
        // (e.g. layout fetch fails but sessions still render via the
        // fallback path below).
        if (sessionsResult.status === "rejected") {
          console.error("hydratePanes: listSessions failed", sessionsResult.reason);
        }
        if (layoutResult.status === "rejected") {
          console.error("hydratePanes: getWorkspaceLayout failed", layoutResult.reason);
        }
        if (sessionsResult.status === "rejected" && layoutResult.status === "rejected") {
          setHydrationError({
            ...toErrorInfo(sessionsResult.reason),
            retry: true,
          });
        }

        if (sessions.length > 0) {
          // Build a lookup of supports_messages_view from the backend layout
          const layoutSmv = new Map<string, boolean>();
          if (layout) {
            for (const p of layout.panes) {
              layoutSmv.set(p.session_id, p.supports_messages_view ?? false);
            }
          }
          setPanes((prev) => (prev.length > 0 ? prev : sessions.map((session) => ({
            session,
            supportsMessagesView: layoutSmv.get(session.id) ?? false,
          }))));
        }

        // Sync workspace store from backend layout
        const store = useWorkspaceStore.getState();
        if (layout) {
          // Map backend panes to PaneMetadata
          const sessionSet = new Set(sessions.map((s) => s.id));
          const paneMetadata = layout.panes
            .filter((p) => sessionSet.has(p.session_id))
            .map((p) => ({
              sessionId: p.session_id,
              name: p.name,
              headerColor: p.header_color,
              themeId: p.theme_id,
              fontSize: p.font_size,
              groupId: p.group_id,
              supportsMessagesView: p.supports_messages_view ?? false,
            }));

          // Sessions without pane metadata get defaults
          const knownPanes = new Set(layout.panes.map((p) => p.session_id));
          for (const session of sessions) {
            if (!knownPanes.has(session.id)) {
              const newPane = {
                sessionId: session.id,
                name: session.shell.split("/").pop() ?? "terminal",
                headerColor: store.defaultHeaderColor,
                themeId: store.defaultThemeId,
                fontSize: store.defaultFontSize,
                groupId: null,
                supportsMessagesView: false,
              };
              paneMetadata.push(newPane);
              // Persist new pane to backend (fire-and-forget)
              updateWorkspacePane(session.id, {
                name: newPane.name,
                header_color: newPane.headerColor,
                theme_id: newPane.themeId,
                font_size: newPane.fontSize,
                sort_order: paneMetadata.length - 1,
              }).catch(() => {});
            }
          }

          // Update store
          if (store.panes.length === 0) useWorkspaceStore.setState({
            panes: paneMetadata,
            activePane: layout.active_pane || paneMetadata[0]?.sessionId || null,
            groups: layout.groups.map((g) => ({
              id: g.id,
              name: g.name,
              color: g.color,
              isCollapsed: g.is_collapsed,
            })),
          });
        } else if (sessions.length > 0 && store.panes.length === 0) {
          // Fallback: no layout from backend, build from sessions.
          // Without layout data, supportsMessagesView defaults to false.
          useWorkspaceStore.setState({
            panes: sessions.map((s) => ({
              sessionId: s.id,
              name: s.shell.split("/").pop() ?? "terminal",
              headerColor: store.defaultHeaderColor,
              themeId: store.defaultThemeId,
              fontSize: store.defaultFontSize,
              groupId: null,
              supportsMessagesView: false,
            })),
            activePane: sessions[0]?.id || null,
          });
        }
      } catch (err) {
        // Unexpected synchronous failure inside the hydration pipeline
        // (decoder bug, type mismatch, etc.). The per-call rejection paths
        // above already cover network failures; reaching this catch means
        // something is structurally wrong, so surface it.
        if (!canceled) {
          console.error("hydratePanes: unexpected failure", err);
          setHydrationError({ ...toErrorInfo(err), retry: true });
        }
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

  const clearHydrationError = useCallback(() => {
    setHydrationError(null);
  }, []);

  // Guard against concurrent creation requests (e.g., rapid double-click).
  // The `isCreating` state flag drives the UI (button disable), while
  // `createInFlight` prevents the handler itself from executing twice.
  const createInFlight = useRef(false);

  // [REQ:P0-006a] Terminal Launch Flow UI
  const launchSession = useCallback(async (opts?: {
    command?: string;
    backend?: BackendID;
    policy?: { mode: PolicyMode; duration?: string };
  }) => {
    const command = opts?.command;
    // Replay guard: if a creation is already in-flight, skip silently.
    if (createInFlight.current) return null;
    createInFlight.current = true;
    setIsCreating(true);
    setCreateError(null);
    try {
      const session = await createSession({
        cols: DEFAULT_COLS,
        rows: DEFAULT_ROWS,
        backend: opts?.backend,
        policy: opts?.policy,
        launch_command: command,
        agent_type: agentTypeFromCommand(command),
      });
      if (command) {
        pendingCommands.current.set(session.id, command);
      }
      setPanes((prev) => [...prev, { session, supportsMessagesView: supportsMessagesViewForCommand(command) }]);
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
    // Session-end is a genuine stop intent: tear down any TTS provider held for
    // this session in the playback registry, even if it was handed off (still
    // speaking) when its pane last unmounted, so ending a session stops its
    // audio rather than leaving an orphaned tail playing.
    ttsPlaybackRegistry.stop(sessionId);
    try {
      await deleteSession(sessionId);
    } catch {
      // Session may already be dead
    }
  }, []);

  const handleExit = useCallback((sessionId: string) => {
    console.log(`Session ${sessionId} exited`);
  }, []);

  const submitToActiveTerminal = useCallback(
    (data: string, source: InputSource, targetId?: string): GateResult => {
      const target = targetId ?? panes[panes.length - 1]?.session.id;
      if (target) {
        const handle = terminalRefs.current.get(target);
        if (handle) {
          return handle.submitInput(data, source);
        }
      }
      return { status: "rejected", reason: "disposed" };
    },
    [panes],
  );

  const subscribeActiveInputSettled = useCallback(
    (targetId: string | undefined, cb: (seq: number, ok: boolean) => void): () => void => {
      const target = targetId ?? panes[panes.length - 1]?.session.id;
      if (!target) return () => {};
      const handle = terminalRefs.current.get(target);
      if (!handle) return () => {};
      return handle.subscribeInputSettled(cb);
    },
    [panes],
  );

  const subscribeActivePendingInput = useCallback(
    (targetId: string | undefined, cb: () => void): () => void => {
      const target = targetId ?? panes[panes.length - 1]?.session.id;
      if (!target) return () => {};
      const handle = terminalRefs.current.get(target);
      if (!handle) return () => {};
      return handle.subscribePendingInput(cb);
    },
    [panes],
  );

  const getActivePendingInputSnapshot = useCallback(
    (targetId?: string): readonly { data: string; addedAt: number }[] => {
      const target = targetId ?? panes[panes.length - 1]?.session.id;
      if (!target) return [];
      const handle = terminalRefs.current.get(target);
      return handle?.getPendingInputSnapshot() ?? [];
    },
    [panes],
  );

  const focusActiveTerminal = useCallback(
    (targetId?: string) => {
      const target = targetId ?? panes[panes.length - 1]?.session.id;
      if (target) {
        terminalRefs.current.get(target)?.focus();
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

  const stopActiveTts = useCallback(
    (targetId?: string) => {
      const target = targetId ?? panes[panes.length - 1]?.session.id;
      if (target) {
        terminalRefs.current.get(target)?.stopTts();
      }
    },
    [panes],
  );

  const speakTextOnPane = useCallback(
    (sessionId: string, text: string, paragraphs?: string[], opts?: { eventId?: string; version?: "active" | "original"; initiatedBy?: "auto" | "manual" }) => {
      return terminalRefs.current.get(sessionId)?.speakText(text, paragraphs, opts) ?? Promise.resolve(undefined);
    },
    [],
  );

  const pauseTtsOnPane = useCallback(
    (sessionId: string) => {
      terminalRefs.current.get(sessionId)?.pauseTts();
    },
    [],
  );

  const resumeTtsOnPane = useCallback(
    (sessionId: string) => {
      terminalRefs.current.get(sessionId)?.resumeTts();
    },
    [],
  );

  const seekTtsOnPane = useCallback(
    (sessionId: string, seconds: number) => {
      terminalRefs.current.get(sessionId)?.seekTts(seconds);
    },
    [],
  );

  const setTtsPlaybackRateOnPane = useCallback(
    (sessionId: string, rate: number) => {
      terminalRefs.current.get(sessionId)?.setTtsPlaybackRate(rate);
    },
    [],
  );

  const setTtsVolumeOnPane = useCallback(
    (sessionId: string, level: number) => {
      terminalRefs.current.get(sessionId)?.setTtsVolume(level);
    },
    [],
  );

  const setTtsMutedOnPane = useCallback(
    (sessionId: string, next: boolean) => {
      terminalRefs.current.get(sessionId)?.setTtsMuted(next);
    },
    [],
  );

  const getTtsStateOnPane = useCallback(
    (sessionId: string) => {
      return terminalRefs.current.get(sessionId)?.getTtsState() ?? null;
    },
    [],
  );

  return {
    panes,
    isHydrated,
    isCreating,
    createError,
    hydrationError,
    clearError,
    clearHydrationError,
    launchSession,
    handleTerminalReady,
    removePane,
    handleExit,
    submitToActiveTerminal,
    subscribeActiveInputSettled,
    subscribeActivePendingInput,
    getActivePendingInputSnapshot,
    focusActiveTerminal,
    registerTerminalRef,
    stopActiveTts,
    speakTextOnPane,
    pauseTtsOnPane,
    resumeTtsOnPane,
    seekTtsOnPane,
    setTtsPlaybackRateOnPane,
    setTtsVolumeOnPane,
    setTtsMutedOnPane,
    getTtsStateOnPane,
  };
}
