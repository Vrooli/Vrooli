// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useState, useCallback, useEffect, useMemo, useRef, type ChangeEvent } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { Loader2, Menu, MessageSquareText, Plus, Settings, TerminalSquare } from "lucide-react";
import { useShallow } from "zustand/react/shallow";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { SPLITTER_SIZE_PX, MIN_COLUMN_PX, MIN_ROW_PX } from "../consts/config";
import { useSessionManager } from "../hooks/useSessionManager";
import { useGlobalEventStream } from "../hooks/useGlobalEventStream";
import { useConversationHydration } from "../hooks/useConversationHydration";
import { useVoiceInput } from "../hooks/useVoiceInput";
import { useAppViewport } from "../hooks/useAppViewport";
import { useTouchControls } from "../hooks/useTouchControls";
import { useWakeLock } from "../hooks/useWakeLock";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { usePressGesture } from "../hooks/usePressGesture";
import { useWakeLockStatus } from "../stores/useWakeLockStatus";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import {
  resolveWorkspaceLayout,
  reconcileTrackFractions,
  buildGridTrackTemplate,
  updateAdjacentFractions,
  fractionsMatch,
} from "../lib/gridLayout";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";
import type { GateResult, InputSource } from "./terminal/inputGate";
import { uploadFile } from "../api/uploads";
import { fetchCapabilities } from "../api/capabilities";
import { getSessionDefaults } from "../api/settings";
import { getSession, type BackendOption, type BackendID, type ExpirationPolicy } from "../api/sessions";
import type { LaunchOptions } from "./TerminalLauncher";
import ErrorBanner from "./ErrorBanner";
import GridSplitter from "./GridSplitter";
import TerminalLauncher from "./TerminalLauncher";
import MobileToolbar from "./MobileToolbar";
import type { MobileToolbarHandle } from "./MobileToolbar";
import AiInput from "./AiInput";
import FloatingToolbar from "./FloatingToolbar";
import VoiceRejectionBanner from "./VoiceRejectionBanner";
import WorkspaceMinimap from "./WorkspaceMinimap";
import SettingsModal from "./SettingsModal";
import AppearanceModal from "./AppearanceModal";
import ConfirmCloseDialog from "./ConfirmCloseDialog";
import WorkspacePaneShell from "./WorkspacePaneShell";
import TabBar from "./TabBar";
import SessionSidebar from "./SessionSidebar";
import AudioPlayerBar from "./AudioPlayerBar";
import SummarizeErrorBanner, { type SummarizeErrorState } from "./SummarizeErrorBanner";
import EnableAudioBanner from "./EnableAudioBanner";
import RecoverableSessionsBanner from "./RecoverableSessionsBanner";
import { useConversationStore, type PaneViewMode } from "../stores/useConversationStore";
import type { TTSPlaybackState } from "../audio-integration";
import { useTtsPlaybackController } from "../domains/tts-playback/useTtsPlaybackController";
import { isTabLikeDisplayMode } from "../lib/workspaceDisplayMode";
import { buildWorkspaceNavigationItems, countWorkspaceUnreadMessages } from "../lib/workspaceNavigation";
import { useTabLikeNavigationShortcuts } from "../hooks/useTabLikeNavigationShortcuts";

type ActiveResize = {
  axis: "column" | "row";
  index: number;
  startPointer: number;
  containerSize: number;
  startValues: number[];
};

type ActiveArrangeDrag = { paneId: string; dropIndex: number };

const SIDEBAR_HEADER_LONG_PRESS_MS = 500;
const SIDEBAR_HEADER_PRESS_MOVE_THRESHOLD = 8;

/**
 * ── STABLE CORE: Pane layout and control wiring. ──
 * This component owns ONLY visual layout (grid, header, empty state)
 * and wires child components to the session lifecycle hook.
 *
 * Session orchestration lives in useSessionManager.
 * Error display lives in ErrorBanner.
 * Shortcut data lives in consts/shortcuts.ts.
 *
 * [REQ:P0-001a] Responsive Pane Grid Layout
 * [REQ:P0-001b] Independent Pane Session Lifecycle
 */
export default function Workspace() {
  const { t } = useTranslation();
  const {
    panes: sessionPanes,
    isHydrated,
    isCreating,
    createError,
    hydrationError,
    clearError,
    clearHydrationError,
    launchSession,
    handleTerminalReady,
    removePane: removeSessionPane,
    handleExit: sessionHandleExit,
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
  } = useSessionManager();

  const workspace = useWorkspaceStore(useShallow((state) => ({
    panes: state.panes,
    columnFractions: state.columnFractions,
    rowFractions: state.rowFractions,
    activePane: state.activePane,
    appearanceModalPane: state.appearanceModalPane,
    isMinimapVisible: state.isMinimapVisible,
    displayMode: state.displayMode,
    settingsModalOpen: state.settingsModalOpen,
    aiModalOpen: state.aiModalOpen,
    aiSuggestActive: state.aiSuggestActive,
    autoTtsEnabled: state.autoTtsEnabled,
    startMutedOnLoad: state.startMutedOnLoad,
    keepScreenAwake: state.keepScreenAwake,
    vadAutoStop: state.vadAutoStop,
    groups: state.groups,
    sidebarSortMode: state.sidebarSortMode,
    plusButtonBehavior: state.plusButtonBehavior,
    defaultHeaderColor: state.defaultHeaderColor,
    defaultThemeId: state.defaultThemeId,
    defaultFontSize: state.defaultFontSize,
    addPane: state.addPane,
    removePane: state.removePane,
    setActivePane: state.setActivePane,
    movePaneToIndex: state.movePaneToIndex,
    setColumnFractions: state.setColumnFractions,
    setRowFractions: state.setRowFractions,
    setSettingsModalOpen: state.setSettingsModalOpen,
    setAiModalOpen: state.setAiModalOpen,
    setAiSuggestActive: state.setAiSuggestActive,
    setTabContextMenu: state.setTabContextMenu,
  })));
  const workspacePanes = workspace.panes;
  const activeWorkspacePane = workspace.activePane;
  const addWorkspacePane = workspace.addPane;
  const removeWorkspacePane = workspace.removePane;
  const setActiveWorkspacePane = workspace.setActivePane;
  const vadAutoStop = workspace.vadAutoStop;
  const { syncActivePane, syncPaneUpdate } = useWorkspaceSync();
  const conversationState = useConversationStore(useShallow((state) => ({
    sessions: state.sessions,
    viewModes: state.viewModes,
    setViewMode: state.setViewMode,
    clearSession: state.clearSession,
    activeViewMode: workspace.activePane ? (state.viewModes[workspace.activePane] ?? "terminal") : "terminal",
  })));
  const {
    sessions: conversationSessions,
    viewModes: conversationViewModes,
    setViewMode: setConversationViewMode,
    clearSession: clearConversationSession,
    activeViewMode,
  } = conversationState;

  // Fetch available backends once on mount (they don't change at runtime)
  const [availableBackends, setAvailableBackends] = useState<BackendOption[]>();
  useEffect(() => {
    let cancelled = false;
    fetchCapabilities().then((caps) => {
      if (cancelled) return;
      if (caps.session_backends) setAvailableBackends(caps.session_backends);
    }).catch(() => {});
    return () => { cancelled = true; };
  }, []);

  const gridRef = useRef<HTMLDivElement>(null);
  const sidebarLayoutRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const activeResizeRef = useRef<ActiveResize | null>(null);
  useAppViewport();
  const needsTouchControls = useTouchControls();
  const wakeLockStatus = useWakeLock(workspace.keepScreenAwake);
  const setWakeLockStatus = useWakeLockStatus((s) => s.setStatus);
  useEffect(() => { setWakeLockStatus(wakeLockStatus); }, [wakeLockStatus, setWakeLockStatus]);
  const mobileToolbarRef = useRef<MobileToolbarHandle>(null);

  const [launcherOpen, setLauncherOpen] = useState(false);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [lastVisitedBySession, setLastVisitedBySession] = useState<Record<string, string>>({});

  // Fetch session defaults on mount AND each time the launcher opens so
  // the dropdown shows the correct server default immediately.
  const [defaultBackend, setDefaultBackend] = useState<BackendID>("standard");
  const [defaultPolicy, setDefaultPolicy] = useState<ExpirationPolicy>();
  const fetchDefaults = useCallback(() => {
    getSessionDefaults().then((d) => {
      if (d.default_backend) setDefaultBackend(d.default_backend as BackendID);
      if (d.default_policy) setDefaultPolicy(d.default_policy);
    }).catch(() => {});
  }, []);
  // Fetch on mount to avoid the "standard" initial state persisting.
  useEffect(() => { fetchDefaults(); }, [fetchDefaults]);
  // Re-fetch when launcher opens so Settings changes are reflected.
  useEffect(() => {
    if (launcherOpen) fetchDefaults();
  }, [launcherOpen, fetchDefaults]);
  const pendingActivePaneRef = useRef<string | null>(null);
  const [pendingClose, setPendingClose] = useState<string | null>(null);
  const exitedSessionsRef = useRef<Set<string>>(new Set());

  const activatePane = useCallback((sessionId: string) => {
    setActiveWorkspacePane(sessionId);
    syncActivePane(workspacePanes.map((pane) => pane.sessionId), sessionId);
  }, [setActiveWorkspacePane, syncActivePane, workspacePanes]);

  const isTabLikeMode = isTabLikeDisplayMode(workspace.displayMode);

  // --- Mobile single-column layout ---
  const [isMobile, setIsMobile] = useState(
    () => typeof window !== "undefined" && window.innerWidth < 768,
  );
  useEffect(() => {
    const onResize = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  // --- Pane drag-and-drop reordering ---
  const [activeArrangeDrag, setActiveArrangeDrag] =
    useState<ActiveArrangeDrag | null>(null);

  const startArrangeDrag = useCallback(
    (paneId: string, e: ReactPointerEvent) => {
      const idx = workspacePanes.findIndex((p) => p.sessionId === paneId);
      if (idx === -1) return;
      e.preventDefault();
      e.stopPropagation();
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
      setActiveArrangeDrag({ paneId, dropIndex: idx });
    },
    [workspacePanes],
  );

  useEffect(() => {
    if (activeArrangeDrag === null) return;

    const handleMove = (e: PointerEvent) => {
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const paneEl = el?.closest("[data-pane-index]");
      if (paneEl) {
        const idx = Number(paneEl.getAttribute("data-pane-index"));
        if (Number.isFinite(idx)) {
          setActiveArrangeDrag((prev) =>
            prev ? { ...prev, dropIndex: idx } : null,
          );
        }
      }
    };

    const handleUp = () => {
      setActiveArrangeDrag((prev) => {
        if (prev) workspace.movePaneToIndex(prev.paneId, prev.dropIndex);
        return null;
      });
    };

    window.addEventListener("pointermove", handleMove);
    window.addEventListener("pointerup", handleUp);
    window.addEventListener("pointercancel", handleUp);
    return () => {
      window.removeEventListener("pointermove", handleMove);
      window.removeEventListener("pointerup", handleUp);
      window.removeEventListener("pointercancel", handleUp);
    };
  }, [activeArrangeDrag, workspace]);

  // Reconcile session manager panes with workspace store.
  // Only remove stale store panes after session hydration completes —
  // otherwise the initial empty sessionPanes would nuke persisted metadata.
  useEffect(() => {
    const sessionIds = new Set(sessionPanes.map((p) => p.session.id));
    const storeIds = new Set(workspacePanes.map((p) => p.sessionId));

    // Add new sessions to store (auto-activate if user just created it)
    for (const sp of sessionPanes) {
      if (!storeIds.has(sp.session.id)) {
        const shouldActivate = pendingActivePaneRef.current === sp.session.id;
        if (shouldActivate) pendingActivePaneRef.current = null;
        addWorkspacePane(sp.session.id, sp.session.shell ?? "terminal", shouldActivate, sp.supportsMessagesView);
        // Persist new pane metadata (including supportsMessagesView) to the backend
        syncPaneUpdate(sp.session.id, {
          name: sp.session.shell?.split("/").pop() ?? "terminal",
          header_color: workspace.defaultHeaderColor,
          theme_id: workspace.defaultThemeId,
          font_size: workspace.defaultFontSize,
          supports_messages_view: sp.supportsMessagesView,
        });
      }
    }
    // Remove deleted sessions from store (only after hydration)
    if (isHydrated) {
      for (const sid of storeIds) {
        if (!sessionIds.has(sid)) {
          removeWorkspacePane(sid);
        }
      }
    }
  }, [
    addWorkspacePane,
    isHydrated,
    removeWorkspacePane,
    sessionPanes,
    syncPaneUpdate,
    workspace.defaultFontSize,
    workspace.defaultHeaderColor,
    workspace.defaultThemeId,
    workspacePanes,
  ]);

  // Auto-set active pane if none is set or the persisted value is stale
  useEffect(() => {
    if (workspacePanes.length === 0) return;
    const activePaneExists = workspace.activePane !== null && workspacePanes.some((p) => p.sessionId === workspace.activePane);
    if (!activePaneExists) {
      const lastPane = workspacePanes[workspacePanes.length - 1];
      if (lastPane) activatePane(lastPane.sessionId);
    }
  }, [workspace.activePane, workspacePanes, activatePane]);

  const openLauncher = useCallback(() => setLauncherOpen(true), []);
  const closeLauncher = useCallback(() => setLauncherOpen(false), []);

  const handleLaunch = useCallback(
    async (opts?: LaunchOptions) => {
      const session = await launchSession(opts);
      if (session) {
        setLauncherOpen(false);
        // Mark session for auto-activation. The reconciliation effect
        // will add the pane and activate it atomically in a single
        // zustand set(), avoiding cross-system state races.
        pendingActivePaneRef.current = session.id;
      }
    },
    [launchSession],
  );

  const handleRetry = useCallback(() => {
    clearError();
    handleLaunch();
  }, [clearError, handleLaunch]);

  // Stable callback for the (memoized) TabBar so a conversation event landing
  // in the store doesn't re-render the whole tab strip via an inline arrow.
  const handleNewTerminal = useCallback(() => { handleLaunch(); }, [handleLaunch]);

  const doRemovePane = useCallback(
    (sessionId: string) => {
      removeSessionPane(sessionId);
      removeWorkspacePane(sessionId);
      clearConversationSession(sessionId);
      exitedSessionsRef.current.delete(sessionId);
      try { localStorage.removeItem(`wc-mobile-draft-${sessionId}`); } catch { /* ignore */ }
    },
    [clearConversationSession, removeSessionPane, removeWorkspacePane],
  );

  const handleRequestClose = useCallback(
    async (sessionId: string) => {
      // Skip confirmation for sessions whose process already exited
      if (exitedSessionsRef.current.has(sessionId)) {
        doRemovePane(sessionId);
        return;
      }
      // Ask the server whether the shell has a running child process
      try {
        const info = await getSession(sessionId);
        if (!info.busy) {
          doRemovePane(sessionId);
          return;
        }
      } catch {
        // If the fetch fails (e.g. session already gone), close without dialog
        doRemovePane(sessionId);
        return;
      }
      setPendingClose(sessionId);
    },
    [doRemovePane],
  );

  const handleConfirmClose = useCallback(() => {
    if (pendingClose) doRemovePane(pendingClose);
    setPendingClose(null);
  }, [pendingClose, doRemovePane]);

  const handleCancelClose = useCallback(() => {
    setPendingClose(null);
  }, []);

  useTabLikeNavigationShortcuts({
    enabled: isTabLikeMode,
    panes: workspacePanes,
    activePane: workspace.activePane,
    onActivatePane: activatePane,
    onClosePane: handleRequestClose,
  });

  useEffect(() => {
    if (!workspace.activePane) return;
    setLastVisitedBySession((prev) => ({
      ...prev,
      [workspace.activePane as string]: new Date().toISOString(),
    }));
  }, [workspace.activePane]);

  const handleExit = useCallback((sessionId: string) => {
    exitedSessionsRef.current.add(sessionId);
    sessionHandleExit(sessionId);
  }, [sessionHandleExit]);

  const handleSendToTerminal = useCallback(
    (data: string, source: InputSource): GateResult => {
      return submitToActiveTerminal(data, source, workspace.activePane ?? undefined);
    },
    [submitToActiveTerminal, workspace.activePane],
  );

  const handleSubscribeInputSettled = useCallback(
    (cb: (seq: number, ok: boolean) => void) =>
      subscribeActiveInputSettled(workspace.activePane ?? undefined, cb),
    [subscribeActiveInputSettled, workspace.activePane],
  );

  const handleSubscribePendingInput = useCallback(
    (cb: () => void) =>
      subscribeActivePendingInput(workspace.activePane ?? undefined, cb),
    [subscribeActivePendingInput, workspace.activePane],
  );

  const handleGetPendingInputSnapshot = useCallback(
    () => getActivePendingInputSnapshot(workspace.activePane ?? undefined),
    [getActivePendingInputSnapshot, workspace.activePane],
  );

  const handleFocusTerminal = useCallback(() => {
    focusActiveTerminal(workspace.activePane ?? undefined);
  }, [focusActiveTerminal, workspace.activePane]);

  // Switch the active pane from messages view back to terminal view.
  // Used by MobileToolbar to auto-switch after sending a command.
  const handleSwitchToTerminal = useCallback(() => {
    if (workspace.activePane) {
      setConversationViewMode(workspace.activePane, "terminal");
    }
  }, [setConversationViewMode, workspace.activePane]);

  // ── Stop TTS on the previous pane when switching tabs ──
  // Each TerminalPane manages its own TTS playback.  When the user switches
  // tabs, the newly-active pane may auto-play pending conversation events
  // (the auto-TTS effect in TerminalPane.tsx).  Without stopping the old
  // pane's TTS first, both audio streams play simultaneously — a jarring
  // experience.  We track the previous active pane and stop its TTS before
  // the new pane's auto-play effect fires.
  const prevActivePaneRef = useRef<string | null>(workspace.activePane);
  useEffect(() => {
    const prev = prevActivePaneRef.current;
    prevActivePaneRef.current = workspace.activePane;
    // Stop TTS on the pane we just left (if any)
    if (prev && prev !== workspace.activePane) {
      stopActiveTts(prev);
    }
  }, [workspace.activePane, stopActiveTts]);

  // ── Desktop auto-focus: focus terminal when active tab changes ──
  // On mobile we MUST NOT do this — focusing xterm's hidden <textarea> opens
  // the virtual keyboard, which shrinks the viewport and forces the user to
  // dismiss it before they can even read the terminal output.  The
  // MobileToolbar provides an alternative input path that doesn't require
  // terminal focus.
  //
  // On desktop there is no virtual keyboard, so auto-focusing the terminal
  // after a tab switch is safe and expected — it lets the user start typing
  // immediately without an extra click.
  //
  // requestAnimationFrame ensures the focus call runs after the browser has
  // painted the visibility change (tabs use `visibility: hidden/visible`).
  // Without this, the focus call can silently fail on some browsers because
  // the target element isn't yet visually rendered.
  //
  // IMPORTANT — REGRESSION GUARD: Do NOT remove the `isMobile` check.
  // It has been intentionally added twice before and each time was lost to
  // unrelated refactors. On mobile, terminal focus === keyboard popup.
  useEffect(() => {
    if (!workspace.activePane || isMobile) return;
    // Don't steal focus from open modals
    if (workspace.settingsModalOpen || workspace.aiModalOpen || workspace.aiSuggestActive || workspace.appearanceModalPane !== null) return;
    const paneId = workspace.activePane;
    const rafId = requestAnimationFrame(() => {
      focusActiveTerminal(paneId);
    });
    return () => cancelAnimationFrame(rafId);
  }, [workspace.activePane, isMobile, workspace.settingsModalOpen, workspace.aiModalOpen, workspace.aiSuggestActive, workspace.appearanceModalPane, focusActiveTerminal]);

  const handleVoiceTranscript = useCallback((text: string) => {
    if (isMobile) {
      // On mobile, inject into the toolbar text box for review before sending
      mobileToolbarRef.current?.appendText(text);
    } else {
      handleSendToTerminal(text, "voice");
    }
  }, [isMobile, handleSendToTerminal]);

  const voiceInput = useVoiceInput(handleVoiceTranscript);

  const handleVoiceStart = useCallback((opts?: { vadEnabled?: boolean }) => {
    // Always stop TTS before starting voice recording — the user wants to
    // speak, so any playing audio should yield.  This is unconditional
    // because the isTtsSpeaking flag can lag behind actual playback due to
    // the async propagation chain (useEffect in TerminalPane → Workspace
    // Set state).  Calling stop when nothing is playing is a no-op.
    stopActiveTts(workspace.activePane ?? undefined);
    voiceInput.startRecording({ vadEnabled: vadAutoStop && opts?.vadEnabled });
  }, [vadAutoStop, workspace.activePane, stopActiveTts, voiceInput]);

  const handleVoiceStop = useCallback(() => {
    voiceInput.stopRecording();
  }, [voiceInput]);

  const handleVoiceCancel = useCallback(() => {
    voiceInput.cancelTranscription();
  }, [voiceInput]);

  const handleVoiceCommandConfirm = useCallback((suggestion: { commandId: string; rawText: string; args: Record<string, unknown> }) => {
    voiceInput.dismissCommandSuggestion();
    // Command execution is handled by the command vocabulary in commands.ts.
    import("../hooks/voice/commands").then(({ VOICE_COMMANDS }) => {
      const cmd = VOICE_COMMANDS.find((c) => c.id === suggestion.commandId);
      if (!cmd) return;
      cmd.execute({
        createTab: () => handleLaunch(),
        switchToTab: (index: number) => {
          const pane = workspacePanes[index - 1];
          if (pane) setActiveWorkspacePane(pane.sessionId);
        },
        closeTab: () => {
          const active = activeWorkspacePane;
          if (active) doRemovePane(active);
        },
        sendToTerminal: (data: string) => { handleSendToTerminal(data, "voice"); },
        exitVoiceMode: () => voiceInput.stopRecording(),
      }, suggestion.args);
    });
  }, [activeWorkspacePane, voiceInput, handleSendToTerminal, handleLaunch, doRemovePane, setActiveWorkspacePane, workspacePanes]);

  const handleVoiceCommandDismiss = useCallback(() => {
    voiceInput.dismissCommandSuggestion();
  }, [voiceInput]);

  // ── Post-mic-permission focus ──
  // After the user grants microphone permission (or the browser permission
  // dialog closes), the browser leaves focus in limbo — nothing is focused.
  // This watches for the voice state to transition into "recording" and
  // moves focus to a useful target:
  //   • Mobile  → MobileToolbar textarea (so the transcript will appear there)
  //   • Desktop → terminal (so the user can keep typing)
  const prevVoiceState = useRef(voiceInput.voiceState);
  useEffect(() => {
    const wasNotRecording = prevVoiceState.current !== "recording";
    prevVoiceState.current = voiceInput.voiceState;
    if (voiceInput.voiceState !== "recording" || !wasNotRecording) return;

    // Small delay: the permission dialog may still be visually dismissing,
    // and focus calls during that animation are sometimes swallowed.
    requestAnimationFrame(() => {
      if (isMobile) {
        mobileToolbarRef.current?.focusInput();
      } else if (workspace.activePane) {
        focusActiveTerminal(workspace.activePane);
      }
    });
  }, [voiceInput.voiceState, isMobile, workspace.activePane, focusActiveTerminal]);

  // --- TTS speaking state ---
  // Track which panes are currently speaking so voice input can stop active TTS
  // before recording. Playback presentation belongs to the audio bar.
  const [ttsSpeakingPanes, setTtsSpeakingPanes] = useState<Set<string>>(new Set());
  const handleTtsSpeakingChange = useCallback((sessionId: string, speaking: boolean) => {
    setTtsSpeakingPanes(prev => {
      const has = prev.has(sessionId);
      if (speaking && has) return prev;    // already tracked — no state change
      if (!speaking && !has) return prev;  // already absent — no state change
      const next = new Set(prev);
      if (speaking) next.add(sessionId);
      else next.delete(sessionId);
      return next;
    });
  }, []);
  const isTtsSpeaking = workspace.activePane ? ttsSpeakingPanes.has(workspace.activePane) : false;

  // ── TTS playback state polling for AudioPlayerBar ──
  //
  // IMPORTANT — why visibility is driven by `isTtsSpeaking` alone, NOT
  // `isTtsSpeaking && ttsPlayback`:
  //
  // `isTtsSpeaking` flips to true synchronously via the React
  // onTtsSpeakingChange callback the moment audio starts.  `ttsPlayback`
  // is populated by *polling* the provider at 100 ms intervals.  The
  // polling useEffect only starts running after React re-renders and
  // paints (because `isTtsSpeaking` is in its dependency array), then
  // the first setInterval tick fires 100 ms later — so there is always
  // a 100–200 ms window where audio is audible but the bar is invisible.
  // For very short TTS messages the audio can finish before the first
  // poll ever fires, meaning the bar never appears at all.
  //
  // Fix: when auto-TTS is enabled, the AudioPlayerBar renders whenever
  // `isTtsSpeaking` is true, using `FALLBACK_TTS_PLAYBACK` when the poll
  // hasn't returned yet. The fallback has sensible defaults (not paused,
  // no duration, playbackRate 1, volume 1) and exposes all capabilities
  // so every control is visible — the real provider values replace it
  // within the first poll tick.
  const FALLBACK_TTS_PLAYBACK: TTSPlaybackState = {
    currentTime: 0,
    duration: null,
    isPaused: false,
    playbackRate: 1,
    volume: 1,
    // Match the user's "start muted on load" preference until the first poll
    // tick reports the real per-session mute state, so the bar shows the
    // muted icon immediately on first speak rather than briefly flashing
    // unmuted.
    isMuted: workspace.startMutedOnLoad,
    capabilities: { canPause: true, canSeek: false, canAdjustSpeed: true, canAdjustVolume: true },
  };
  const [ttsPlayback, setTtsPlayback] = useState<TTSPlaybackState | null>(null);
  useEffect(() => {
    if (!isTtsSpeaking || !workspace.activePane) {
      setTtsPlayback(null);
      return;
    }
    const activePane = workspace.activePane;
    // Poll immediately on start — don't wait for the first interval tick.
    // This closes the gap where audio is playing but the bar is invisible.
    const poll = () => {
      const state = getTtsStateOnPane(activePane);
      if (state) setTtsPlayback(state);
    };
    poll();
    const id = setInterval(poll, 100);
    return () => clearInterval(id);
  }, [isTtsSpeaking, workspace.activePane, getTtsStateOnPane]);

  const handleTtsPause = useCallback(() => {
    if (workspace.activePane) pauseTtsOnPane(workspace.activePane);
  }, [workspace.activePane, pauseTtsOnPane]);

  const handleTtsResume = useCallback(() => {
    if (workspace.activePane) resumeTtsOnPane(workspace.activePane);
  }, [workspace.activePane, resumeTtsOnPane]);

  const handleTtsSeek = useCallback((seconds: number) => {
    if (workspace.activePane) seekTtsOnPane(workspace.activePane, seconds);
  }, [workspace.activePane, seekTtsOnPane]);

  const handleTtsSetPlaybackRate = useCallback((rate: number) => {
    if (workspace.activePane) setTtsPlaybackRateOnPane(workspace.activePane, rate);
  }, [workspace.activePane, setTtsPlaybackRateOnPane]);

  const handleTtsSetVolume = useCallback((level: number) => {
    if (workspace.activePane) setTtsVolumeOnPane(workspace.activePane, level);
  }, [workspace.activePane, setTtsVolumeOnPane]);

  const handleTtsSetMuted = useCallback((next: boolean) => {
    if (workspace.activePane) setTtsMutedOnPane(workspace.activePane, next);
  }, [workspace.activePane, setTtsMutedOnPane]);

  const applySummarizeResult = useCallback((sessionId: string, eventId: string, speechParagraphs: string[]) => {
    const convState = useConversationStore.getState();
    const session = convState.sessions[sessionId];
    if (!session) return;
    const updatedEvents = session.events.map((event) =>
      event.id === eventId
        ? {
            ...event,
            summarized: true,
            originalSpeechParagraphs: event.originalSpeechParagraphs ?? event.speechParagraphs,
            speechParagraphs,
          }
        : event,
    );
    useConversationStore.setState({
      sessions: { ...convState.sessions, [sessionId]: { ...session, events: updatedEvents } },
    });
  }, []);

  // --- Messages View / replay-bar TTS controls ---
  const [summarizeError, setSummarizeError] = useState<SummarizeErrorState | null>(null);
  // Enable-audio affordance: set by TerminalPane when auto-TTS is rejected by
  // the browser's autoplay policy. `suppressed` is session-scoped — once the
  // user dismisses the banner it never reappears until the tab reloads.
  const [enableAudio, setEnableAudio] = useState<{ sessionId: string; enable: () => Promise<boolean> } | null>(null);
  const [enableAudioSuppressed, setEnableAudioSuppressed] = useState(false);

  const handleNeedsUnlock = useCallback((payload: { sessionId: string; enable: () => Promise<boolean> } | null) => {
    setEnableAudio(payload);
  }, []);

  const handleEnableAudio = useCallback(async (): Promise<boolean> => {
    if (!enableAudio) return false;
    const ok = await enableAudio.enable();
    if (ok) setEnableAudio(null);
    return ok;
  }, [enableAudio]);

  const handleDismissEnableAudio = useCallback(() => {
    setEnableAudioSuppressed(true);
    setEnableAudio(null);
  }, []);

  const handleSummarizeFailed = useCallback((sessionId: string, eventId: string, message: string, source: "auto" | "on-demand") => {
    setSummarizeError({ sessionId, eventId, message, source, status: "failed" });
  }, []);

  const handleDismissSummarizeError = useCallback(() => {
    setSummarizeError(null);
  }, []);

  const handleSummarizeSucceeded = useCallback((sessionId: string, eventId: string) => {
    setSummarizeError((prev) => (
      prev && prev.sessionId === sessionId && prev.eventId === eventId ? null : prev
    ));
  }, []);

  const ttsPlaybackController = useTtsPlaybackController({
    conversationSessions,
    activePaneId: workspace.activePane,
    autoTtsEnabled: workspace.autoTtsEnabled,
    audioState: { playback: ttsPlayback, isSpeaking: isTtsSpeaking },
    setViewMode: setConversationViewMode,
    speakText: (sessionId, text, paragraphs, opts) => {
      return speakTextOnPane(sessionId, text, paragraphs, opts);
    },
    stopPlayback: stopActiveTts,
    applySummarizeResult,
    onSummarizeFailed: (sessionId, eventId, message) => handleSummarizeFailed(sessionId, eventId, message, "on-demand"),
    onSummarizeSucceeded: handleSummarizeSucceeded,
  });
  const handlePlaybackTransportStopped = ttsPlaybackController.handleTransportStopped;
  const handlePaneTransportEventStart = ttsPlaybackController.handleTransportEventStart;
  const getSelectedPlaybackVersion = ttsPlaybackController.getSelectedVersion;
  const getPlaybackSummarizeError = ttsPlaybackController.getSummarizeError;
  const clearPlaybackSummarizeError = ttsPlaybackController.clearSummarizeError;
  const playPaneEvent = ttsPlaybackController.playEvent;
  const playPaneFromHere = ttsPlaybackController.playFromHere;
  const togglePanePlaybackVersion = ttsPlaybackController.toggleVersion;
  const changePaneSummarizeLevel = ttsPlaybackController.changeSummarizeLevel;
  const playbackFocusRequest = ttsPlaybackController.focusRequest;
  const handleConversationEventReceived = ttsPlaybackController.handleIncomingEvent;
  const handleTtsStop = useCallback(() => {
    ttsPlaybackController.stopPlayback(workspace.activePane);
  }, [ttsPlaybackController, workspace.activePane]);

  const handlePaneToggleView = useCallback((sessionId: string, viewMode: PaneViewMode) => {
    setConversationViewMode(sessionId, viewMode === "terminal" ? "messages" : "terminal");
  }, [setConversationViewMode]);

  // Tracks which pane (if any) is mid view-switch so the tabs-mode floating
  // toggle button can show a loading spinner. Each pane shell reports its own
  // transition state; we only surface the active pane's.
  const [viewSwitchPendingPane, setViewSwitchPendingPane] = useState<string | null>(null);
  const handleViewSwitchPendingChange = useCallback((sessionId: string, pending: boolean) => {
    setViewSwitchPendingPane((prev) => {
      if (pending) return sessionId;
      return prev === sessionId ? null : prev;
    });
  }, []);

  const handlePaneTransportSpeakingEvent = useCallback((sessionId: string, eventId: string | null) => {
    if (sessionId === workspace.activePane) {
      handlePaneTransportEventStart(sessionId, eventId);
    }
  }, [handlePaneTransportEventStart, workspace.activePane]);

  const handlePaneSummarizeError = useCallback((sessionId: string, eventId: string, message: string) => {
    handleSummarizeFailed(sessionId, eventId, message, "auto");
  }, [handleSummarizeFailed]);

  const handleRetrySummarize = useCallback(() => {
    setSummarizeError((prev) => {
      if (!prev) return prev;
      const { sessionId, eventId } = prev;
      const session = useConversationStore.getState().sessions[sessionId];
      const event = session?.events.find((candidate) => candidate.id === eventId);
      if (event) {
        ttsPlaybackController.changeSummarizeLevel(sessionId, eventId, ttsPlaybackController.summarizeLevel);
      }
      return { ...prev, status: "retrying" };
    });
  }, [ttsPlaybackController]);

  useEffect(() => {
    if (!isTtsSpeaking) {
      handlePlaybackTransportStopped();
    }
  }, [handlePlaybackTransportStopped, isTtsSpeaking]);

  // --- Mobile image upload ---
  const mobileFileInputRef = useRef<HTMLInputElement>(null);

  const handleMobileUploadImage = useCallback(() => {
    mobileFileInputRef.current?.click();
  }, []);

  const handleMobileFileChange = useCallback(async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file || !workspace.activePane) return;
    try {
      const path = await uploadFile(workspace.activePane, file);
      submitToActiveTerminal(path + "\n", "upload", workspace.activePane);
    } catch {
      // Upload errors are transient — user can retry
    }
  }, [workspace.activePane, submitToActiveTerminal]);

  // --- Resize logic ---
  const startResize = useCallback(
    (axis: "column" | "row", index: number) =>
      (e: ReactPointerEvent) => {
        const container = gridRef.current;
        if (!container) return;
        const rect = container.getBoundingClientRect();
        activeResizeRef.current = {
          axis,
          index,
          startPointer: axis === "column" ? e.clientX : e.clientY,
          containerSize: axis === "column" ? rect.width : rect.height,
          startValues:
            axis === "column"
              ? workspace.columnFractions
              : workspace.rowFractions,
        };
      },
    [workspace.columnFractions, workspace.rowFractions],
  );

  useEffect(() => {
    const handleMove = (e: PointerEvent) => {
      const resize = activeResizeRef.current;
      if (!resize) return;
      e.preventDefault();
      const delta =
        (resize.axis === "column" ? e.clientX : e.clientY) -
        resize.startPointer;
      const splitterCount = resize.startValues.length - 1;
      const updated = updateAdjacentFractions({
        startValues: resize.startValues,
        index: resize.index,
        delta,
        containerSize: resize.containerSize,
        splitterCount,
        minTrackPx: resize.axis === "column" ? MIN_COLUMN_PX : MIN_ROW_PX,
        splitterSize: SPLITTER_SIZE_PX,
      });
      if (resize.axis === "column") {
        workspace.setColumnFractions(updated);
      } else {
        workspace.setRowFractions(updated);
      }
    };

    const handleUp = () => {
      activeResizeRef.current = null;
    };

    window.addEventListener("pointermove", handleMove, { passive: false });
    window.addEventListener("pointerup", handleUp);
    return () => {
      window.removeEventListener("pointermove", handleMove);
      window.removeEventListener("pointerup", handleUp);
    };
  }, [workspace]);

  // Compute layout
  const orderedPanes = workspacePanes;

  // Single SSE subscription for the whole app: conversation events + unread
  // updates for ALL sessions flow here, decoupled from any terminal WS. Auto-
  // summarize failures surface through the active pane's banner handler.
  useGlobalEventStream({ onSummarizeError: handlePaneSummarizeError });
  // Keep every session's conversation hydrated for badges even when its
  // terminal pane is unmounted (offscreen panes no longer hydrate themselves).
  useConversationHydration(orderedPanes.map((pane) => pane.sessionId));

  // --- Warm set: which sessions stay MOUNTED in tab-like modes ---
  // The core scaling fix: instead of mounting all N panes (and paying for N
  // xterms / WebSockets / observers) and hiding the inactive ones with
  // visibility:hidden, we mount only a small LRU warm set. Cost is flat in N.
  // K=2 (active + most-recent-previous) keeps the common back-and-forth toggle
  // instant while still bounding live terminals to a constant. Grid mode is
  // unchanged (it intentionally shows every visible cell).
  const WARM_SET_SIZE = 2;
  const [recentSessions, setRecentSessions] = useState<string[]>([]);
  useEffect(() => {
    const active = workspace.activePane;
    if (!active) return;
    setRecentSessions((prev) => {
      if (prev[0] === active) return prev;
      return [active, ...prev.filter((id) => id !== active)].slice(0, WARM_SET_SIZE);
    });
  }, [workspace.activePane]);
  const mountedTabSessions = useMemo(() => {
    const existing = new Set(orderedPanes.map((p) => p.sessionId));
    const ids = new Set<string>();
    if (workspace.activePane && existing.has(workspace.activePane)) ids.add(workspace.activePane);
    for (const id of recentSessions) {
      if (ids.size >= WARM_SET_SIZE) break;
      if (existing.has(id)) ids.add(id);
    }
    return ids;
  }, [orderedPanes, recentSessions, workspace.activePane]);

  const maxColumns = isMobile ? 1 : 2;
  const layout = resolveWorkspaceLayout(orderedPanes.length, maxColumns);
  const colFractions = reconcileTrackFractions(
    workspace.columnFractions,
    layout.columns,
  );
  const rowFractions = reconcileTrackFractions(
    workspace.rowFractions,
    layout.rows,
  );

  // Persist reconciled fractions if they differ.
  //
  // IMPORTANT — fractionsMatch uses an epsilon tolerance instead of strict
  // equality (===).  normalizeFractions divides each value by the sum of all
  // values.  For certain track counts (6, 7, 9, …) the sum is not exactly
  // 1.0 in IEEE 754 floating-point, so re-normalizing already-normalized
  // values produces slightly different results on each pass.  With strict
  // equality the effect detects a "difference", writes the new values to the
  // store (which triggers a re-render), re-normalizes (producing yet another
  // slightly different result), and loops — exceeding React's maximum update
  // depth (error #185).  An epsilon of 1e-12 is far below any user-visible
  // precision while absorbing the ~1 ULP drift that normalization introduces.
  //
  // Additionally, in tab-like display modes the grid is never rendered, so
  // fraction reconciliation is skipped entirely to avoid unnecessary store
  // writes.
  useEffect(() => {
    if (isTabLikeMode) return;
    if (!fractionsMatch(colFractions, workspace.columnFractions)) {
      workspace.setColumnFractions(colFractions);
    }
  }, [colFractions, isTabLikeMode, workspace]);

  useEffect(() => {
    if (isTabLikeMode) return;
    if (!fractionsMatch(rowFractions, workspace.rowFractions)) {
      workspace.setRowFractions(rowFractions);
    }
  }, [isTabLikeMode, rowFractions, workspace]);

  const colTemplate = buildGridTrackTemplate(colFractions, SPLITTER_SIZE_PX);
  const rowTemplate = buildGridTrackTemplate(rowFractions, SPLITTER_SIZE_PX);

  // Scale grid min-height by row count so each row can grow up to viewport height.
  // Without this, fr tracks share a single-viewport container and panes shrink
  // as rows are added instead of extending the scrollable surface.
  const viewportPaneHeight = typeof window === "undefined"
    ? MIN_ROW_PX
    : Math.max(MIN_ROW_PX, window.innerHeight);
  const rowSplittersHeight = Math.max(0, layout.rows - 1) * SPLITTER_SIZE_PX;
  const minimumGridHeightPx = (viewportPaneHeight * layout.rows) + rowSplittersHeight;
  const sidebarHeaderPressGesture = usePressGesture<string>({
    longPressMs: SIDEBAR_HEADER_LONG_PRESS_MS,
    moveThresholdPx: SIDEBAR_HEADER_PRESS_MOVE_THRESHOLD,
    onTap: () => {},
    onLongPress: (sessionId, point) => {
      workspace.setTabContextMenu({ sessionId, position: { x: point.x, y: point.y } });
    },
  });

  // While session hydration is in flight, show a loading screen to prevent
  // the empty state ("New Terminal" button) from flashing before we know
  // whether any sessions exist.
  if (!isHydrated) {
    return (
      <div className="flex h-wc-app items-center justify-center bg-wc-surface-base text-wc-text-muted">
        Loading...
      </div>
    );
  }

  // Empty state
  if (sessionPanes.length === 0) {
    return (
      <div className="flex h-wc-app flex-col bg-wc-surface-base text-wc-text-primary">
        <RecoverableSessionsBanner
          onRecovered={(result) => {
            pendingActivePaneRef.current = result.new_session_id;
          }}
        />
        <div className="flex flex-1 items-center justify-center">
          <div className="text-center">
            <h1 className="text-2xl font-semibold mb-4">{t(strings.app.title)}</h1>
            <p className="text-wc-text-muted mb-6">
              {t(strings.workspace.tagline)}
            </p>
            {hydrationError && (
              <ErrorBanner
                error={hydrationError}
                onDismiss={clearHydrationError}
                onRetry={hydrationError.retry ? () => window.location.reload() : undefined}
                className="mb-4"
              />
            )}
            {createError && (
              <ErrorBanner
                error={createError}
                onDismiss={clearError}
                onRetry={createError.retry ? handleRetry : undefined}
                className="mb-4"
              />
            )}
            <Button
              data-testid="new-terminal-button"
              onClick={openLauncher}
              disabled={isCreating}
              size="lg"
            >
              <Plus className="me-2 h-5 w-5" />
              {isCreating ? t(strings.workspace.creating) : t(strings.workspace.newTerminalButton)}
            </Button>
          </div>
        </div>
        <TerminalLauncher
          open={launcherOpen}
          onClose={closeLauncher}
          onLaunch={handleLaunch}
          isCreating={isCreating}
          defaultBackend={defaultBackend}
          defaultPolicy={defaultPolicy}
          availableBackends={availableBackends}
        />
      </div>
    );
  }

  // Build splitter elements
  const columnSplitters: React.ReactNode[] = [];
  for (let i = 0; i < colFractions.length - 1; i++) {
    const gridCol = `${2 + i * 2}`;
    columnSplitters.push(
      <GridSplitter
        key={`col-${i}`}
        axis="column"
        gridColumn={gridCol}
        gridRow={`1 / -1`}
        onPointerDown={startResize("column", i)}
        label={`Resize column ${i + 1}`}
      />,
    );
  }

  const rowSplitters: React.ReactNode[] = [];
  for (let i = 0; i < rowFractions.length - 1; i++) {
    const gridRow = `${2 + i * 2}`;
    rowSplitters.push(
      <GridSplitter
        key={`row-${i}`}
        axis="row"
        gridColumn={`1 / -1`}
        gridRow={gridRow}
        onPointerDown={startResize("row", i)}
        label={`Resize row ${i + 1}`}
      />,
    );
  }

  // Map panes into grid cells
  const isDragging = activeArrangeDrag !== null;
  const paneCells = orderedPanes.map((paneMeta, idx) => {
    const col = idx % layout.columns;
    const row = Math.floor(idx / layout.columns);
    // Grid positions account for splitter tracks: content is at odd positions (1, 3, 5, ...)
    const gridColumn = `${1 + col * 2}`;
    const gridRow = `${1 + row * 2}`;

    const isBeingDragged = activeArrangeDrag?.paneId === paneMeta.sessionId;
    const isDropTarget =
      isDragging &&
      !isBeingDragged &&
      activeArrangeDrag?.dropIndex === idx;

    return (
      <WorkspacePaneShell
        key={paneMeta.sessionId}
        paneMeta={paneMeta}
        layoutMode="grid"
        gridColumn={gridColumn}
        gridRow={gridRow}
        paneIndex={idx}
        isActive={workspace.activePane === paneMeta.sessionId}
        isBeingDragged={isBeingDragged}
        isDropTarget={isDropTarget}
        isTtsSpeaking={isTtsSpeaking && workspace.activePane === paneMeta.sessionId}
        activeSpeakingEventId={workspace.activePane === paneMeta.sessionId ? ttsPlaybackController.activeEventId : null}
        loadingEventId={workspace.activePane === paneMeta.sessionId ? ttsPlaybackController.loadingEventId : null}
        summarizeLevel={ttsPlaybackController.summarizeLevel}
        summarizingEventId={ttsPlaybackController.summarizingEventId}
        getSummarizeError={getPlaybackSummarizeError}
        onClearSummarizeError={clearPlaybackSummarizeError}
        onToggleSummarized={togglePanePlaybackVersion}
        onChangeLevel={changePaneSummarizeLevel}
        selectedVersionForEvent={getSelectedPlaybackVersion}
        playbackState={ttsPlayback ?? FALLBACK_TTS_PLAYBACK}
        onSetPlaybackRate={handleTtsSetPlaybackRate}
        onSetVolume={handleTtsSetVolume}
        onSetMuted={handleTtsSetMuted}
        playbackFocusRequest={workspace.activePane === paneMeta.sessionId ? playbackFocusRequest : null}
        onActivate={activatePane}
        onRequestClose={handleRequestClose}
        onToggleView={handlePaneToggleView}
        onViewSwitchPendingChange={handleViewSwitchPendingChange}
        onStartArrangeDrag={startArrangeDrag}
        onTerminalReady={handleTerminalReady}
        onTerminalExit={handleExit}
        onTerminalRef={registerTerminalRef}
        onVoiceStart={voiceInput.supported ? voiceInput.startRecording : undefined}
        onVoiceStop={voiceInput.supported ? voiceInput.stopRecording : undefined}
        onTtsSpeakingChange={handleTtsSpeakingChange}
        onSpeakingEventChange={handlePaneTransportSpeakingEvent}
        onConversationEventReceived={handleConversationEventReceived}
        onNeedsUnlock={handleNeedsUnlock}
        onPlayFromHere={playPaneFromHere}
        onPlayEvent={playPaneEvent}
      />
    );
  });

  const navigationItems = buildWorkspaceNavigationItems({
    panes: orderedPanes,
    groups: workspace.groups ?? [],
    activePane: workspace.activePane,
    conversationSessions,
    viewModes: conversationViewModes,
    lastVisitedBySession,
    sortMode: workspace.sidebarSortMode,
  });
  const activeNavigationItem = navigationItems.find(
    (item) => item.kind === "pane" && item.pane.sessionId === workspace.activePane,
  );
  const activeSidebarPane = activeNavigationItem?.kind === "pane" ? activeNavigationItem : null;
  const sidebarUnreadCount = countWorkspaceUnreadMessages(orderedPanes, conversationSessions);

  // h-wc-app maps to var(--wc-app-height, 100dvh) — the actual visible
  // viewport height set by useAppViewport(). This is the root layout
  // container; all descendants use flex to fill this height.
  // See src/hooks/useAppViewport.ts for the full design rationale.
  return (
    <div
      className="flex flex-col bg-wc-surface-base text-wc-text-primary h-wc-app"
    >
      {/* Floating toolbar — hidden on mobile tab mode where TabBar
       * already provides the plus button and we move settings there. */}
      <FloatingToolbar
        hidden={isMobile && isTabLikeMode}
        onOpenSettings={() => workspace.setSettingsModalOpen(true)}
        onOpenAi={() => workspace.setAiModalOpen(true)}
        onNewTerminal={() => handleLaunch()}
        onOpenLauncher={openLauncher}
        isCreating={isCreating}
        voiceSupported={voiceInput.supported}
        voicePreparing={voiceInput.isPreparing}
        voiceRecording={voiceInput.isRecording}
        voiceListening={voiceInput.isListening}
        voiceTranscribing={voiceInput.isTranscribing}
        voiceError={voiceInput.error}
        voiceLevel={voiceInput.audioLevel}
        voiceActivity={voiceInput.voiceActivity}
        voicePartialTranscript={voiceInput.partialTranscript}
        voiceBackend={voiceInput.backend}
        onVoiceStart={handleVoiceStart}
        onVoiceStop={handleVoiceStop}
        onVoiceCancel={handleVoiceCancel}
        isTtsSpeaking={isTtsSpeaking}
        onTtsStop={handleTtsStop}
      />

      {/* Voice fallback notice */}
      {voiceInput.fallbackNotice && (
        <div className="py-1.5 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))] text-xs text-amber-300 bg-amber-500/10 border-b border-amber-500/30">
          {voiceInput.fallbackNotice}
        </div>
      )}
      {voiceInput.rejectedAudio && (
        <VoiceRejectionBanner
          rejection={voiceInput.rejectedAudio}
          onRetry={voiceInput.retryWithoutFilter}
          onDismiss={voiceInput.dismissRejection}
        />
      )}

      {summarizeError && (
        <SummarizeErrorBanner
          state={summarizeError}
          onRetry={handleRetrySummarize}
          onDismiss={handleDismissSummarizeError}
        />
      )}

      {enableAudio && !enableAudioSuppressed && (
        <EnableAudioBanner
          onEnable={handleEnableAudio}
          onDismiss={handleDismissEnableAudio}
        />
      )}

      {/* Error banner */}
      {createError && (
        <ErrorBanner
          error={createError}
          onDismiss={clearError}
          onRetry={createError.retry ? handleRetry : undefined}
          className="border-b border-wc-error"
        />
      )}

      {/* Recoverable sessions surface — banner is hidden when nothing is
          awaiting recovery, so steady-state UI is unchanged. */}
      <RecoverableSessionsBanner
        onRecovered={(result) => {
          pendingActivePaneRef.current = result.new_session_id;
        }}
      />

      {/* Tab bar (only in tabs mode) */}
      {workspace.displayMode === "tabs" && (
        <TabBar
          panes={orderedPanes}
          activePane={workspace.activePane}
          onNewTerminal={handleNewTerminal}
          onOpenLauncher={openLauncher}
          onClosePane={handleRequestClose}
          isCreating={isCreating}
          trailingActions={isMobile ? (
            <Button
              data-testid="tabbar-settings"
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0 mx-1 self-center"
              onClick={() => workspace.setSettingsModalOpen(true)}
              title={t(strings.workspace.settingsTitle)}
            >
              <Settings className="h-4 w-4" />
            </Button>
          ) : undefined}
        />
      )}

      {workspace.displayMode === "sidebar" && (
        <div
          data-testid="workspace-sidebar-topbar"
          className="flex h-[calc(2.5rem+var(--wc-safe-top,0px))] shrink-0 items-center gap-2 border-b border-wc-default bg-wc-surface-header pt-[var(--wc-safe-top,0px)] ps-[max(0.5rem,var(--wc-safe-left,0px))] pe-[max(0.5rem,var(--wc-safe-right,0px))] md:hidden"
        >
          <Button
            data-testid="workspace-sidebar-toggle"
            variant="ghost"
            size="icon"
            className="relative h-8 w-8"
            onClick={() => setMobileSidebarOpen(true)}
            title={t(strings.sessionSidebar.open)}
          >
            <Menu className="h-4 w-4" />
            {sidebarUnreadCount > 0 && (
              <span
                data-testid="workspace-sidebar-toggle-unread"
                className="absolute -end-1 -top-1 min-w-4 rounded-full bg-wc-accent px-1 text-[10px] font-semibold leading-4 text-black"
              >
                {sidebarUnreadCount > 99 ? "99+" : sidebarUnreadCount}
              </span>
            )}
          </Button>
          <div
            data-testid="workspace-sidebar-active-title"
            className="min-w-0 flex-1 select-none truncate text-sm font-medium touch-manipulation"
            title={activeSidebarPane?.pane.name ?? t(strings.sessionSidebar.title)}
            {...(activeSidebarPane
              ? sidebarHeaderPressGesture.getGestureHandlers(activeSidebarPane.pane.sessionId)
              : {})}
          >
            {activeSidebarPane?.pane.name ?? t(strings.sessionSidebar.title)}
          </div>
          {activeSidebarPane && activeSidebarPane.unreadCount > 0 && (
            <span className="rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-black">
              {activeSidebarPane.unreadCount}
            </span>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            disabled={isCreating}
            onClick={() => {
              if (workspace.plusButtonBehavior === "launcher") {
                openLauncher();
              } else {
                handleLaunch();
              }
            }}
            title={workspace.plusButtonBehavior === "launcher" ? t(strings.floatingToolbar.launcherFirstTitle) : t(strings.floatingToolbar.terminalFirstTitle)}
          >
            <Plus className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => workspace.setSettingsModalOpen(true)}
            title={t(strings.workspace.settingsTitle)}
          >
            <Settings className="h-4 w-4" />
          </Button>
        </div>
      )}

      {/* Main content area */}
      {isTabLikeMode ? (
        <div
          ref={workspace.displayMode === "sidebar" ? sidebarLayoutRef : undefined}
          className="flex flex-1 min-h-0 overflow-hidden"
        >
          {workspace.displayMode === "sidebar" && (
            <SessionSidebar
              items={navigationItems}
              containerRef={sidebarLayoutRef}
              isMobile={isMobile}
              mobileOpen={mobileSidebarOpen}
              isCreating={isCreating}
              onCloseMobile={() => setMobileSidebarOpen(false)}
              onActivatePane={activatePane}
              onClosePane={handleRequestClose}
              onNewTerminal={() => handleLaunch()}
              onOpenLauncher={openLauncher}
              onOpenSettings={() => workspace.setSettingsModalOpen(true)}
            />
          )}
          {/* Tab-like modes: stacked panes with hidden inactive panes */}
          <div className="relative flex-1 min-h-0 overflow-hidden">
          {/* Toggle between terminal and messages view.
           * Shows the icon for the view you'll switch TO (not the current view):
           *   • In terminal mode → show chat icon (click to switch to messages)
           *   • In messages mode → show terminal icon (click to switch back)
           * Circular icon button with a translucent background so it doesn't
           * obscure too much terminal content but is still easy to tap. */}
          {workspace.activePane && workspace.panes.find((pane) => pane.sessionId === workspace.activePane)?.supportsMessagesView && (
            <div className="absolute end-[max(0.75rem,var(--wc-safe-right,0px))] top-[max(0.75rem,var(--wc-safe-top,0px))] z-20">
              <button
                className="flex items-center justify-center h-8 w-8 rounded-full bg-wc-surface-raised/80 border border-wc-default text-wc-text-secondary hover:text-wc-text-primary hover:bg-wc-surface-input transition-colors backdrop-blur-sm"
                onClick={() => {
                  if (workspace.activePane) {
                    handlePaneToggleView(workspace.activePane, activeViewMode);
                  }
                }}
                title={activeViewMode === "terminal" ? t(strings.workspace.switchToMessagesTitle) : t(strings.workspace.switchToTerminalTitle)}
              >
                {viewSwitchPendingPane === workspace.activePane
                  ? <Loader2 data-testid="workspace-toggle-view-pending" className="h-3.5 w-3.5 animate-spin" />
                  : activeViewMode === "terminal"
                    ? <MessageSquareText className="h-3.5 w-3.5" />
                    : <TerminalSquare className="h-3.5 w-3.5" />}
              </button>
            </div>
          )}
            {orderedPanes.filter((paneMeta) => mountedTabSessions.has(paneMeta.sessionId)).map((paneMeta) => {
              return (
                <WorkspacePaneShell
                  key={paneMeta.sessionId}
                  paneMeta={paneMeta}
                  layoutMode="tabs"
                  isActive={paneMeta.sessionId === workspace.activePane}
                  isVisible={paneMeta.sessionId === workspace.activePane}
                  isTtsSpeaking={isTtsSpeaking && workspace.activePane === paneMeta.sessionId}
                  activeSpeakingEventId={workspace.activePane === paneMeta.sessionId ? ttsPlaybackController.activeEventId : null}
                  loadingEventId={workspace.activePane === paneMeta.sessionId ? ttsPlaybackController.loadingEventId : null}
                  summarizeLevel={ttsPlaybackController.summarizeLevel}
                  summarizingEventId={ttsPlaybackController.summarizingEventId}
                  getSummarizeError={getPlaybackSummarizeError}
                  onClearSummarizeError={clearPlaybackSummarizeError}
                  onToggleSummarized={togglePanePlaybackVersion}
                  onChangeLevel={changePaneSummarizeLevel}
                  selectedVersionForEvent={getSelectedPlaybackVersion}
                  playbackState={ttsPlayback ?? FALLBACK_TTS_PLAYBACK}
                  onSetPlaybackRate={handleTtsSetPlaybackRate}
                  onSetVolume={handleTtsSetVolume}
                  onSetMuted={handleTtsSetMuted}
                  playbackFocusRequest={workspace.activePane === paneMeta.sessionId ? playbackFocusRequest : null}
                  onActivate={activatePane}
                  onRequestClose={handleRequestClose}
                  onToggleView={handlePaneToggleView}
                  onViewSwitchPendingChange={handleViewSwitchPendingChange}
                  onTerminalReady={handleTerminalReady}
                  onTerminalExit={handleExit}
                  onTerminalRef={registerTerminalRef}
                  onVoiceStart={voiceInput.supported ? voiceInput.startRecording : undefined}
                  onVoiceStop={voiceInput.supported ? voiceInput.stopRecording : undefined}
                  onTtsSpeakingChange={handleTtsSpeakingChange}
                  onSpeakingEventChange={handlePaneTransportSpeakingEvent}
                  onConversationEventReceived={handleConversationEventReceived}
                  onNeedsUnlock={handleNeedsUnlock}
                  onPlayFromHere={playPaneFromHere}
                  onPlayEvent={playPaneEvent}
                />
              );
            })}
          </div>
        </div>
      ) : (
        /* Grid mode: original grid layout with minimap */
        <div className="relative flex-1 min-h-0 overflow-hidden">
          <div
            ref={scrollContainerRef}
            className={cn("absolute inset-0 overflow-auto wc-hide-scrollbar", workspace.isMinimapVisible && "right-[34px]")}
          >
            <div
              ref={gridRef}
              data-testid="pane-grid"
              className={cn(
                "grid gap-0 p-1",
                isDragging && "select-none cursor-grabbing [&_.xterm]:pointer-events-none",
              )}
              style={{
                gridTemplateColumns: colTemplate,
                gridTemplateRows: rowTemplate,
                height: `${minimumGridHeightPx}px`,
                minHeight: `${minimumGridHeightPx}px`,
              }}
            >
              {paneCells}
              {columnSplitters}
              {rowSplitters}
            </div>
          </div>

          {/* Minimap (only in grid mode) */}
          <WorkspaceMinimap scrollRef={scrollContainerRef} rowCount={layout.rows} />
        </div>
      )}

      {/* Bottom bar */}
      <div className="relative z-10 shrink-0">
        {/* TTS player bar — visible for active manual playback, or when
         * auto-TTS is enabled and there is playback/replay context.
         *
         * When actively speaking, the bar shows live playback state (polled
         * at 100 ms).  When idle with a replayable event, it shows a
         * "stopped" state where the play button triggers a replay.
         *
         * Uses isTtsSpeaking plus controller context so the bar appears
         * the instant eligible audio starts. ttsPlayback is
         * populated by polling; if the first poll hasn't fired yet we
         * fall back to FALLBACK_TTS_PLAYBACK (see comment above the
         * polling effect). */}
        {(() => {
          const pb = isTtsSpeaking
            ? (ttsPlayback ?? FALLBACK_TTS_PLAYBACK)
            : { ...(ttsPlayback ?? FALLBACK_TTS_PLAYBACK), isPaused: true };
          const context = ttsPlaybackController.buildBarContext(
            workspace.activePane,
            workspace.autoTtsEnabled,
            { playback: ttsPlayback, isSpeaking: isTtsSpeaking },
          );
          if (!context?.event || !context.sessionId) return null;
          const activeEvent = context.event;
          const isReplayMode = !isTtsSpeaking;
          const hasOriginal = (activeEvent.originalSpeechParagraphs?.length ?? 0) > 0;
          const canRequestSummarize = activeEvent.role === "assistant";
          const isPlayingSummarized = context.version === "active" && hasOriginal;
          return (
            <AudioPlayerBar
              isPaused={pb.isPaused}
              currentTime={pb.currentTime}
              duration={pb.duration}
              playbackRate={pb.playbackRate}
              volume={pb.volume}
              isMuted={pb.isMuted}
              capabilities={pb.capabilities}
              isSummarized={isPlayingSummarized}
              hasOriginalVersion={hasOriginal}
              canSummarize={canRequestSummarize}
              isSummarizing={ttsPlaybackController.summarizingEventId === activeEvent.id}
              isLoading={ttsPlaybackController.loadingEventId === activeEvent.id}
              currentLevel={ttsPlaybackController.summarizeLevel}
              currentMessageLabel={context.queueLabel}
              currentMessageId={activeEvent.id}
              messageSelectorEvents={conversationSessions[context.sessionId]?.events ?? []}
              hasQueuedNext={context.hasQueuedNext}
              onPause={() => {
                ttsPlaybackController.pausePlayback(context.sessionId);
                handleTtsPause();
              }}
              onResume={isReplayMode ? () => {
                ttsPlaybackController.resumePlayback(context.sessionId);
              } : () => {
                ttsPlaybackController.resumePlayback(context.sessionId);
                handleTtsResume();
              }}
              onSeek={handleTtsSeek}
              onSetPlaybackRate={handleTtsSetPlaybackRate}
              onSetVolume={handleTtsSetVolume}
              onSetMuted={handleTtsSetMuted}
              onSelectMessage={(eventId) => {
                ttsPlaybackController.playEvent(context.sessionId as string, eventId);
              }}
              onToggleSummarized={hasOriginal && activeEvent && workspace.activePane ? (useSummarized) => {
                ttsPlaybackController.toggleVersion(context.sessionId as string, activeEvent.id, useSummarized);
              } : undefined}
              onChangeLevel={canRequestSummarize && activeEvent && workspace.activePane ? (level) => {
                ttsPlaybackController.changeSummarizeLevel(context.sessionId as string, activeEvent.id, level);
              } : undefined}
              onDismiss={!workspace.autoTtsEnabled ? () => {
                ttsPlaybackController.stopPlayback(context.sessionId);
                stopActiveTts(context.sessionId as string);
              } : undefined}
            />
          );
        })()}
        {/* Mobile toolbar */}
        <MobileToolbar
          ref={mobileToolbarRef}
          visible={needsTouchControls}
          onInput={handleSendToTerminal}
          subscribeInputSettled={handleSubscribeInputSettled}
          subscribePendingInput={handleSubscribePendingInput}
          getPendingInputSnapshot={handleGetPendingInputSnapshot}
          onFocusTerminal={handleFocusTerminal}
          activeSessionId={workspace.activePane}
          voiceSupported={voiceInput.supported}
          voicePreparing={voiceInput.isPreparing}
          voiceRecording={voiceInput.isRecording}
          voiceListening={voiceInput.isListening}
          voiceTranscribing={voiceInput.isTranscribing}
          voiceError={voiceInput.error}
          voiceLevel={voiceInput.audioLevel}
          voiceActivity={voiceInput.voiceActivity}
          voicePartialTranscript={voiceInput.partialTranscript}
          voiceBackend={voiceInput.backend}
          onVoiceStart={handleVoiceStart}
          onVoiceStop={handleVoiceStop}
          onVoiceCancel={handleVoiceCancel}
          voiceCommandSuggestion={voiceInput.commandSuggestion}
          onVoiceCommandConfirm={handleVoiceCommandConfirm}
          onVoiceCommandDismiss={handleVoiceCommandDismiss}
          onUploadImage={handleMobileUploadImage}
          onOpenAi={() => workspace.setAiSuggestActive(!workspace.aiSuggestActive)}
          aiSuggestActive={workspace.aiSuggestActive}
          onAiSuggestExecute={(cmd) => {
            handleSendToTerminal(cmd, "toolbar-submit");
            mobileToolbarRef.current?.clearInput();
            workspace.setAiSuggestActive(false);
          }}
          isTtsSpeaking={isTtsSpeaking}
          onTtsStop={handleTtsStop}
          viewMode={activeViewMode}
          onSwitchToTerminal={handleSwitchToTerminal}
        />
        <input
          ref={mobileFileInputRef}
          type="file"
          accept="image/*"
          hidden
          onChange={handleMobileFileChange}
        />
      </div>

      {/* Terminal Launcher */}
      <TerminalLauncher
        open={launcherOpen}
        onClose={closeLauncher}
        onLaunch={handleLaunch}
        isCreating={isCreating}
        defaultBackend={defaultBackend}
        defaultPolicy={defaultPolicy}
        availableBackends={availableBackends}
      />

      {/* Settings Modal */}
      <SettingsModal
        sessions={sessionPanes}
        onDeleteSession={handleRequestClose}
      />

      {/* Appearance Modal */}
      <AppearanceModal />

      {/* AI Modal */}
      <AiInput onExecute={(cmd) => { handleSendToTerminal(cmd, "toolbar-submit"); }} />

      {/* Close confirmation dialog */}
      <ConfirmCloseDialog
        open={pendingClose !== null}
        sessionName={workspace.panes.find((p) => p.sessionId === pendingClose)?.name ?? "terminal"}
        onConfirm={handleConfirmClose}
        onCancel={handleCancelClose}
      />
    </div>
  );
}
