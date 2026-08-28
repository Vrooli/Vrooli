// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// DOC: docs/internal/SEAMS.md#1-entry-presentation
import { useState, useCallback, useEffect, useMemo, useRef, type ChangeEvent } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { Loader2, Menu, MessageSquareText, MonitorSmartphone, Plus, Settings, TerminalSquare, X } from "lucide-react";
import { useShallow } from "zustand/react/shallow";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { SPLITTER_SIZE_PX, MIN_COLUMN_PX, MIN_ROW_PX, TERMINAL_THEMES, DEFAULT_THEME_ID } from "../consts/config";
import { chromeTheme } from "../lib/chromeTheme";
import { useSessionManager } from "../hooks/useSessionManager";
import { useGlobalEventStream } from "../hooks/useGlobalEventStream";
import { useConversationHydration } from "../hooks/useConversationHydration";
import { useScenarioVoiceInput as useVoiceInput } from "../audio-integration";
import { useAppViewport } from "../hooks/useAppViewport";
import { useTouchControls } from "../hooks/useTouchControls";
import { useWakeLock } from "../hooks/useWakeLock";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { usePressGesture } from "../hooks/usePressGesture";
import { useWakeLockStatus } from "../stores/useWakeLockStatus";
import { useWorkspaceStore, type PaneMetadata } from "../stores/useWorkspaceStore";
import {
  resolveWorkspaceLayout,
  reconcileTrackFractions,
  buildGridTrackTemplate,
  updateAdjacentFractions,
  fractionsMatch,
} from "../lib/gridLayout";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";
import type { GateResult, InputIntent } from "./terminal/inputGate";
import { uploadFile } from "../api/uploads";
import { fetchCapabilities } from "../api/capabilities";
import { getSessionDefaults } from "../api/settings";
import { getSession, type BackendOption, type BackendID, type ExpirationPolicy, type RecoverResult, type SessionOriginName } from "../api/sessions";
import { listTargetCatalog, type TargetCatalog, type TerminalTarget } from "../api/targets";
import type { LaunchOptions } from "./TerminalLauncher";
import ErrorBanner from "./ErrorBanner";
import GridSplitter from "./GridSplitter";
import TerminalLauncher from "./TerminalLauncher";
import MachinesDrawer from "./machines/MachinesDrawer";
import MobileToolbar from "./MobileToolbar";
import type { MobileToolbarHandle } from "./MobileToolbar";
import AiInput from "./AiInput";
import FloatingToolbar from "./FloatingToolbar";
import FullScreenComposer from "./FullScreenComposer";
import VoiceMicButton from "./VoiceMicButton";
import { useComposerDraft } from "../hooks/useComposerDraft";
import { useComposerAttachments } from "../hooks/useComposerAttachments";
import { useComposerHotkey } from "../hooks/useComposerHotkey";
import { useWindowKeyDown } from "../hooks/useKeyboardListeners";
import BannerRegion from "./banners/BannerRegion";
import type { MaybeBanner } from "./banners/types";
import {
  createErrorBanner,
  enableAudioBanner,
  summarizeErrorBanner,
  trackingDegradedBanner,
  ttsSpeakingBanner,
  voiceErrorBanner,
  voiceFallbackBanner,
  voiceRejectionBanner,
  voiceStaleMicBanner,
  voiceTranscribingBanner,
} from "./banners/descriptors";
import {
  useCrashRecoveryBanner,
  useSessionRecoveryBanner,
} from "./banners/useRecoveryBanners";
import WorkspaceMinimap from "./WorkspaceMinimap";
import SettingsModal from "./SettingsModal";
import AppearanceModal from "./AppearanceModal";
import ManageGroupsDrawer from "./ManageGroupsDrawer";
import CloseGroupDialog from "./CloseGroupDialog";
import HandoffComposer from "./handoff/HandoffComposer";
import type { RoleMeta } from "../stores/useWorkspaceStore";
import GroupUndoBanner from "./GroupUndoBanner";
import RoleMenu from "./RoleMenu";
import { useGroupActions } from "../hooks/useGroupActions";
import { useRoleActions } from "../hooks/useRoleActions";
import { sendHandoff, targetsForSession, type HandoffTarget } from "../hooks/useHandoff";
import type { GroupCreationRequest } from "./launcher/GroupModePanel";
import { listGroupTemplates, upsertGroupTemplate } from "../api/grouptemplates";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import WorkspacePaneShell from "./WorkspacePaneShell";
import TabBar from "./TabBar";
import SessionSidebar from "./SessionSidebar";
import AudioPlayerBar from "./AudioPlayerBar";
import type { SummarizeErrorState } from "../types/summarize";
import ArchiveDrawer from "./ArchiveDrawer";
import TopSafeArea from "./TopSafeArea";
import { useConversationStore, type PaneViewMode } from "../stores/useConversationStore";
import type { TTSPlaybackState } from "../audio-integration";

const FALLBACK_TTS_PLAYBACK: Omit<TTSPlaybackState, "isMuted"> = {
  currentTime: 0,
  duration: null,
  isPaused: false,
  playbackRate: 1,
  volume: 1,
  capabilities: { canPause: true, canSeek: false, canAdjustSpeed: true, canAdjustVolume: true },
};
import { useTtsPlaybackController } from "../domains/tts-playback/useTtsPlaybackController";
import { isTabLikeDisplayMode } from "../lib/workspaceDisplayMode";
import { buildWorkspaceNavigationItems, buildOriginBucketedNavigation, countWorkspaceUnreadMessages } from "../lib/workspaceNavigation";
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

interface WorkspaceProps {
  /**
   * Notices raised by App (connection health, audio-tools reachability).
   * They are arbitrated together with Workspace's own in a single
   * `BannerRegion`, so there is exactly one top-chrome owner and one
   * safe-area owner.
   */
  appBanners?: readonly MaybeBanner[];
}

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
export default function Workspace({ appBanners = [] }: WorkspaceProps = {}) {
  const { t } = useTranslation();
  const [targetCatalog, setTargetCatalog] = useState<TargetCatalog>({
    status: "ready",
    targets: [],
  });
  const [targetsLoading, setTargetsLoading] = useState(true);
  const refreshTargetCatalog = useCallback(async () => {
    setTargetsLoading(true);
    try {
      setTargetCatalog(await listTargetCatalog());
    } catch (error) {
      console.error("target catalog unavailable", error);
      setTargetCatalog({
        status: "registry-error",
        targets: [],
        message: "The target catalog could not be loaded.",
        recovery_action: "Check Web Console and Bridge health, then try again.",
      });
    } finally {
      setTargetsLoading(false);
    }
  }, []);
  useEffect(() => {
    void refreshTargetCatalog();
  }, [refreshTargetCatalog]);
  const availableTargets: TerminalTarget[] = targetCatalog.targets;
  const {
    panes: sessionPanes,
    isHydrated,
    isCreating,
    createError,
    hydrationError,
    clearError,
    clearHydrationError,
    launchSession,
    removePane: removeSessionPane,
    undoArchive,
    deletePanePermanently,
    mergeExternalSession,
    endExternalSession,
    handleExit: sessionHandleExit,
    submitToActiveTerminal,
    subscribeActiveInputSettled,
    awaitActiveInputOffset,
    subscribeActivePendingInput,
    getActivePendingInputSnapshot,
    discardActivePendingInput,
    discardAllActivePendingInput,
    flushActivePendingInputNow,
    copySelectionOnPane,
    pasteFromClipboardOnPane,
    scrollTerminalOnPane,
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
    persistentMode: state.persistentMode,
    groups: state.groups,
    roles: state.roles,
    sidebarSortMode: state.sidebarSortMode,
    adaptiveChrome: state.adaptiveChrome,
    plusButtonBehavior: state.plusButtonBehavior,
    defaultHeaderColor: state.defaultHeaderColor,
    defaultThemeId: state.defaultThemeId,
    defaultFontSize: state.defaultFontSize,
    addPane: state.addPane,
    setPaneGroup: state.setPaneGroup,
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
  const workspaceGroups = workspace.groups;
  const activeWorkspacePane = workspace.activePane;
  const activeSessionTrackingDegraded = sessionPanes.find((pane) => pane.session.id === workspace.activePane)?.session.tracking_degraded;
  const addWorkspacePane = workspace.addPane;
  const setWorkspacePaneGroup = workspace.setPaneGroup;
  const removeWorkspacePane = workspace.removePane;
  const setActiveWorkspacePane = workspace.setActivePane;
  const vadAutoStop = workspace.vadAutoStop;
  const { syncActivePane, syncPaneUpdate, syncPaneOrder, syncPaneMove } = useWorkspaceSync();
  const { createNamedGroup, closeGroup, closeGroupIfFinished, restoreClosedGroup, dismissClosedGroupUndo } = useGroupActions();
  const { createRole, updateRole, removeRole, setRoleSession } = useRoleActions();
  const updateWorkspaceGroup = useWorkspaceStore((s) => s.updateGroup);
  const { syncUpdateGroup } = useWorkspaceSync();

  // Text waiting for a session that does not have a mounted terminal yet.
  // Mirrors pendingGroupBySessionRef exactly, and is drained by the same
  // session-reconcile effect — a second draining mechanism would be a second
  // place for a handoff to go missing.
  const pendingHandoffBySessionRef = useRef<Map<string, string>>(new Map());
  // The role a just-started session belongs to, attached when its pane lands.
  const pendingRoleBySessionRef = useRef<Map<string, string>>(new Map());
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
  // Publish this device's keyboard state so each pane can declare it to its
  // session; followers draw the leader's keyboard rather than guessing at it.
  const setKeyboardOpen = useWorkspaceStore((state) => state.setKeyboardOpen);
  useAppViewport({ onKeyboardChange: setKeyboardOpen });
  const needsTouchControls = useTouchControls();
  const wakeLockStatus = useWakeLock(workspace.keepScreenAwake);
  const setWakeLockStatus = useWakeLockStatus((s) => s.setStatus);
  useEffect(() => { setWakeLockStatus(wakeLockStatus); }, [wakeLockStatus, setWakeLockStatus]);
  const mobileToolbarRef = useRef<MobileToolbarHandle>(null);

  // Single shared per-session draft: the collapsed toolbar input and the
  // full-screen composer read/write ONE value that cannot diverge.
  const composerDraft = useComposerDraft(workspace.activePane);
  const composerAttachments = useComposerAttachments();
  const [composerOpen, setComposerOpen] = useState(false);
  const [archiveDrawerOpen, setArchiveDrawerOpen] = useState(false);
  const [archivePreferOrphans, setArchivePreferOrphans] = useState(false);
  const [archiveInitialSessionId, setArchiveInitialSessionId] = useState<string | null>(null);
  const openCrashArchive = useCallback(() => {
    setArchiveInitialSessionId(null);
    setArchivePreferOrphans(true);
    setArchiveDrawerOpen(true);
  }, []);
  const closeArchiveDrawer = useCallback(() => {
    setArchiveDrawerOpen(false);
    setArchivePreferOrphans(false);
    setArchiveInitialSessionId(null);
  }, []);
  const openComposer = useCallback(() => { setComposerOpen(true); }, []);
  const closeComposer = useCallback(() => { setComposerOpen(false); }, []);
  // Desktop keyboard shortcut (Ctrl/Cmd+Shift+K) opens the composer.
  useComposerHotkey(openComposer);
  useWindowKeyDown(true, useCallback((event: KeyboardEvent) => {
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key.toLocaleLowerCase() === "h") {
      event.preventDefault();
      setArchiveDrawerOpen(true);
    }
  }, []));

  const sendArchivedMessageToComposer = useCallback((text: string) => {
    const target = composerDraft.getSessionId();
    if (!target || target !== useWorkspaceStore.getState().activePane) return;
    composerDraft.appendAtCaret(text);
    setArchiveDrawerOpen(false);
    setComposerOpen(true);
  }, [composerDraft]);

  const handleArchiveReopened = useCallback((result: RecoverResult) => {
    pendingActivePaneRef.current = result.new_session_id;
    void getSession(result.new_session_id).then((session) => {
      mergeExternalSession(session, true);
    }).catch(() => {
      // The session.created event is the authoritative fallback. Keeping the
      // pending target makes that later merge focus the reopened pane.
    });
  }, [mergeExternalSession]);

  const [launcherOpen, setLauncherOpen] = useState(false);
  const [machinesOpen, setMachinesOpen] = useState(false);
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
  const pendingGroupBySessionRef = useRef<Map<string, string>>(new Map());
  // The launcher's destination lives in STATE, not a ref. As a ref it could
  // never reach the dialog, which is why the dialog could not state where the
  // session it was about to create would go.
  const [launcherGroupId, setLauncherGroupId] = useState<string | null>(null);
  // handleLaunch is a stable callback, so it cannot read launcherGroupId
  // directly without going stale. The ref mirrors the state for that one read.
  const launcherGroupRef = useRef<string | null>(null);
  useEffect(() => { launcherGroupRef.current = launcherGroupId; }, [launcherGroupId]);
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);
  const [archiveUndo, setArchiveUndo] = useState<{
    sessionId: string;
    pane: PaneMetadata;
    index: number;
  } | null>(null);
  const exitedSessionsRef = useRef<Set<string>>(new Set());

  const activatePane = useCallback((sessionId: string) => {
    // Activation clears a manual unread flag — but only on a real transition,
    // so flagging the session you are looking at survives until you leave and
    // come back. See the store's setActivePane.
    const clearedUnread = setActiveWorkspacePane(sessionId);
    syncActivePane(workspacePanes.map((pane) => pane.sessionId), sessionId);
    if (clearedUnread) syncPaneUpdate(sessionId, { manually_unread: false });
  }, [setActiveWorkspacePane, syncActivePane, syncPaneUpdate, workspacePanes]);

  const isTabLikeMode = isTabLikeDisplayMode(workspace.displayMode);

  // Adaptive app-chrome: tell the imperative chrome controller which pane owns
  // the chrome (the focused pane in single-focus modes), whether tinting is
  // active, and the owner's configured theme background (the detection
  // fallback). The per-pane detector feeds detected colors straight to the
  // controller; this effect only carries the low-frequency config, so it never
  // re-renders on color changes. See lib/chromeTheme.ts.
  const activeChromeThemeId = workspace.panes.find(
    (p) => p.sessionId === workspace.activePane,
  )?.themeId;
  useEffect(() => {
    const enabled = workspace.adaptiveChrome && isTabLikeMode && !!workspace.activePane;
    const fallbackColor = enabled
      ? (TERMINAL_THEMES[activeChromeThemeId ?? DEFAULT_THEME_ID]?.colors.background
        ?? TERMINAL_THEMES[DEFAULT_THEME_ID]?.colors.background
        ?? null)
      : null;
    chromeTheme.setConfig({
      enabled,
      ownerSessionId: enabled ? workspace.activePane : null,
      fallbackColor,
    });
  }, [workspace.adaptiveChrome, isTabLikeMode, workspace.activePane, activeChromeThemeId]);

  // --- Mobile single-column layout ---
  const [isMobile, setIsMobile] = useState(
    () => typeof window !== "undefined" && window.innerWidth < 768,
  );
  useEffect(() => {
    const onResize = () => { setIsMobile(window.innerWidth < 768); };
    window.addEventListener("resize", onResize);
    return () => { window.removeEventListener("resize", onResize); };
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
        if (prev) {
          workspace.movePaneToIndex(prev.paneId, prev.dropIndex);
          // Grid arrange-drag used to mutate the local order and never persist
          // it, so a rearranged grid reverted on reload while every other
          // reorder surface stuck.
          syncPaneMove(prev.paneId);
        }
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
  }, [activeArrangeDrag, syncPaneMove, workspace]);

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
        const pendingGroupId = pendingGroupBySessionRef.current.get(sp.session.id) ?? null;
        if (pendingGroupId) pendingGroupBySessionRef.current.delete(sp.session.id);
        addWorkspacePane(sp.session.id, sp.session.shell ?? "terminal", shouldActivate, sp.supportsMessagesView);
        if (pendingGroupId) {
          setWorkspacePaneGroup(sp.session.id, pendingGroupId);
        }
        // Persist new pane metadata (including supportsMessagesView) to the
        // backend. Read the color and position back out of the store rather
        // than assuming the defaults: joining a group seeds the color from
        // the group, and the pane's index is its sort_order.
        //
        // sort_order is load-bearing. Omitting it left every new pane at the
        // server's zero value, so on the next reload it sorted to the TOP of
        // the list (the query is `ORDER BY sort_order, created_at`) — landing
        // inside whichever group happened to be up there and splitting it.
        const { panes: afterAdd, activePane: activeAfterAdd } = useWorkspaceStore.getState();
        const created = afterAdd.find((pane) => pane.sessionId === sp.session.id);
        syncPaneUpdate(sp.session.id, {
          name: sp.session.shell?.split("/").pop() ?? "terminal",
          header_color: created?.headerColor ?? workspace.defaultHeaderColor,
          theme_id: workspace.defaultThemeId,
          font_size: workspace.defaultFontSize,
          sort_order: Math.max(0, afterAdd.findIndex((pane) => pane.sessionId === sp.session.id)),
          supports_messages_view: sp.supportsMessagesView,
          ...(pendingGroupId ? { group_id: pendingGroupId } : {}),
        });
        if (pendingGroupId) {
          syncPaneOrder(afterAdd.map((pane) => pane.sessionId), activeAfterAdd);
        }

        // A role that was just started now has its session. Attaching here
        // rather than at launch time is what makes the role running only once
        // the process actually exists.
        const pendingRoleId = pendingRoleBySessionRef.current.get(sp.session.id);
        if (pendingRoleId) {
          pendingRoleBySessionRef.current.delete(sp.session.id);
          setRoleSession(pendingRoleId, sp.session.id);
        }

        // Deliver any handoff text held for this session. The terminal is
        // still mounting, so this goes through the same pending-input queue
        // the operator can see, discard, and flush — never dropped silently.
        const pendingText = pendingHandoffBySessionRef.current.get(sp.session.id);
        if (pendingText) {
          pendingHandoffBySessionRef.current.delete(sp.session.id);
          submitToActiveTerminal(pendingText, "bulk_text", sp.session.id);
        }
      }
    }
    // Remove deleted sessions from store (only after hydration)
    if (isHydrated) {
      for (const sid of storeIds) {
        if (!sessionIds.has(sid)) {
          // Remember the group BEFORE the pane goes, so the auto-close check
          // below knows which group to reconsider. Reading it afterwards
          // would always find nothing.
          const departingGroupId = workspacePanes.find((p) => p.sessionId === sid)?.groupId ?? null;
          removeWorkspacePane(sid);
          // A role pointing at a session that no longer exists would be a
          // handoff target aimed at nothing, so it returns to waiting.
          const orphaned = useWorkspaceStore.getState().roles.find((r) => r.sessionId === sid);
          if (orphaned) setRoleSession(orphaned.id, null);
          // Scoped to the one group the pane left: a scan across every group
          // would cost on every session change.
          closeGroupIfFinished(departingGroupId);
        }
      }
    }
  }, [
    addWorkspacePane,
    setWorkspacePaneGroup,
    isHydrated,
    removeWorkspacePane,
    sessionPanes,
    syncPaneOrder,
    syncPaneUpdate,
    setRoleSession,
    submitToActiveTerminal,
    closeGroupIfFinished,
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

  // ---------------------------------------------------------------------
  // Roles and handoffs
  // ---------------------------------------------------------------------

  const [handoffState, setHandoffState] = useState<{
    sourceSessionId: string;
    sourceLabel: string;
    payload: string;
    initialSelection?: string[];
  } | null>(null);
  const [roleMenu, setRoleMenu] = useState<{ role: RoleMeta; position: { x: number; y: number } } | null>(null);

  /** Start a waiting role's command, and remember which role to attach. */
  const handleStartRole = useCallback(async (role: RoleMeta): Promise<string | null> => {
    const session = await launchSession({
      command: role.command || undefined,
      workingDir: role.workingDir || undefined,
      backend: role.backend ? (role.backend as BackendID) : undefined,
      target: role.targetId ? availableTargets.find((candidate) => candidate.id === role.targetId) : undefined,
    });
    if (!session) return null;
    pendingActivePaneRef.current = session.id;
    // The role's group is the pane's group: a started role stays where it was.
    pendingGroupBySessionRef.current.set(session.id, role.groupId);
    pendingRoleBySessionRef.current.set(session.id, role.id);
    return session.id;
  }, [availableTargets, launchSession]);

  const openHandoff = useCallback((sourceSessionId: string, payload: string, initialSelection?: string[]) => {
    const pane = useWorkspaceStore.getState().panes.find((p) => p.sessionId === sourceSessionId);
    setHandoffState({
      sourceSessionId,
      sourceLabel: pane?.name ?? sourceSessionId,
      payload,
      initialSelection,
    });
  }, []);

  const handoffTargets = useMemo(() => {
    if (!handoffState) return [];
    const pane = workspacePanes.find((p) => p.sessionId === handoffState.sourceSessionId);
    return targetsForSession(handoffState.sourceSessionId, pane?.groupId ?? null);
  }, [handoffState, workspacePanes]);

  /**
   * Deliver a handoff.
   *
   * Every seam this needs already exists: submitToActiveTerminal reaches a
   * named pane, launchSession creates the process, and the pending map carries
   * text a not-yet-mounted terminal cannot take. Nothing here polls for
   * readiness, and nothing here knows what a payload is.
   */
  const runHandoff = useCallback(
    (targets: readonly HandoffTarget[], textFor: (target: HandoffTarget) => string) =>
      sendHandoff(targets, textFor, {
        submit: (data, intent, targetId) => submitToActiveTerminal(data, intent, targetId),
        launch: async (options) => {
          const session = await launchSession({
            command: options.command,
            workingDir: options.workingDir,
            backend: options.backend ? (options.backend as BackendID) : undefined,
            target: options.targetId ? availableTargets.find((c) => c.id === options.targetId) : undefined,
          });
          if (!session) return null;
          const sourceGroupId = useWorkspaceStore.getState().panes
            .find((p) => p.sessionId === handoffState?.sourceSessionId)?.groupId;
          if (sourceGroupId) pendingGroupBySessionRef.current.set(session.id, sourceGroupId);
          return session.id;
        },
        queueForSession: (sessionId, text) => {
          // If a terminal is already mounted, hand it over now; otherwise the
          // reconcile effect delivers it the moment the pane appears.
          const immediate = submitToActiveTerminal(text, "bulk_text", sessionId);
          if (immediate.status === "rejected") {
            pendingHandoffBySessionRef.current.set(sessionId, text);
          }
        },
        attachRole: (roleId, sessionId) => {
          pendingRoleBySessionRef.current.set(sessionId, roleId);
        },
      }),
    [availableTargets, handoffState?.sourceSessionId, launchSession, submitToActiveTerminal],
  );

  // Creating a group from inside the launcher is server-first, like every
  // other group creation: the backend mints the id and the dialog adopts it.
  const createLauncherGroup = useCallback(async (name: string): Promise<string | null> => {
    try {
      const group = await createNamedGroup(name);
      return group.id;
    } catch (error) {
      console.error("Failed to create group from launcher:", error);
      return null;
    }
  }, [createNamedGroup]);

  // What a new session will actually look like. The reconcile effect applies
  // these after the session exists; naming them here is what removes the
  // surprise.
  const launcherAppearance = useMemo(() => ({
    headerColor: workspace.defaultHeaderColor,
    themeId: workspace.defaultThemeId,
    fontSize: workspace.defaultFontSize,
  }), [workspace.defaultFontSize, workspace.defaultHeaderColor, workspace.defaultThemeId]);

  const startRoleFromSurface = useCallback((role: RoleMeta) => {
    void handleStartRole(role);
  }, [handleStartRole]);

  const handoffToRole = useCallback((role: RoleMeta) => {
    // Hand off FROM the group's active session TO this role. Falling back to
    // any member keeps the control working when the active pane is elsewhere.
    const { panes, activePane } = useWorkspaceStore.getState();
    const source = panes.find((p) => p.sessionId === activePane && p.groupId === role.groupId)
      ?? panes.find((p) => p.groupId === role.groupId);
    if (!source) return;
    openHandoff(source.sessionId, "", [role.id]);
  }, [openHandoff]);

  const openRoleMenu = useCallback((role: RoleMeta, position: { x: number; y: number }) => {
    setRoleMenu({ role, position });
  }, []);

  /**
   * Create a group, its roles, and its eager sessions in one action.
   *
   * The launches are SEQUENCED, not fired together. launchSession guards
   * against concurrent creation with an in-flight flag and returns null on the
   * second overlapping call, so a template with three eager roles would
   * silently start one. Awaiting each in turn is the whole fix, and it is why
   * this cannot be a Promise.all.
   */
  const createGroupFromRoles = useCallback(async (request: GroupCreationRequest) => {
    try {
      const group = await createNamedGroup(request.name);
      if (request.color) {
        updateWorkspaceGroup(group.id, { color: request.color });
        syncUpdateGroup(group.id, { color: request.color });
      }

      // Roles are created before anything starts, so a failure part-way
      // through leaves a group the operator can see and finish by hand
      // rather than a half-started set of processes with no home.
      const created: (RoleMeta | null)[] = [];
      for (const [index, role] of request.roles.entries()) {
        created.push(await createRole({
          groupId: group.id,
          label: role.label || `Role ${String(index + 1)}`,
          command: role.command,
          workingDir: role.working_dir,
          incomingPrompt: role.incoming_prompt,
          backend: role.backend,
          targetId: role.target_id,
        }));
      }

      // Only eager roles cost a process.
      for (const [index, role] of request.roles.entries()) {
        if (role.start_mode !== "eager") continue;
        const meta = created[index];
        if (!meta) continue;
        await handleStartRole(meta);
      }

      // The counter moves only after the group actually exists, so a failed
      // create does not inflate how often a template looks used.
      if (request.templateId) {
        const template = (await listGroupTemplates()).find((tpl) => tpl.id === request.templateId);
        if (template) {
          await upsertGroupTemplate({
            id: template.id,
            name: template.name,
            color: template.color,
            roles: template.roles,
            use_count: template.use_count + 1,
          });
        }
      }

      setLauncherOpen(false);
    } catch (error) {
      console.error("Failed to create group from roles:", error);
    }
  }, [createNamedGroup, createRole, handleStartRole, syncUpdateGroup, updateWorkspaceGroup]);

  const openLauncher = useCallback(() => { setLauncherOpen(true); }, []);
  // Opening machines from the launcher replaces it rather than stacking two
  // overlays: one surface on screen at a time.
  const openMachines = useCallback(() => {
    setLauncherOpen(false);
    setMachinesOpen(true);
  }, []);
  /**
   * Hand the launcher off to the settings surface that OWNS the list it is
   * rendering. The launcher shows agents and templates; neither is editable
   * there, and leaving the operator to find the right settings tab is the
   * kind of gap that makes a dialog feel unfinished.
   */
  const openSettingsTab = useCallback((tab: string) => {
    setLauncherOpen(false);
    const state = useWorkspaceStore.getState();
    state.setSettingsInitialTab(tab);
    state.setSettingsModalOpen(true);
  }, []);
  const openShortcutSettings = useCallback(() => { openSettingsTab("shortcuts"); }, [openSettingsTab]);
  const openTemplateSettings = useCallback(() => { openSettingsTab("templates"); }, [openSettingsTab]);
  const closeLauncher = useCallback(() => {
    setLauncherGroupId(null);
    setLauncherOpen(false);
  }, []);

  const handleLaunch = useCallback(
    async (opts?: LaunchOptions) => {
      try {
        const session = await launchSession(opts);
        if (session) {
          setLauncherOpen(false);
          // Mark session for auto-activation. The reconciliation effect
          // will add the pane and activate it atomically in a single
          // zustand set(), avoiding cross-system state races.
          pendingActivePaneRef.current = session.id;
          // Read the destination from the ref mirror rather than the state,
          // because this callback may be holding a render-old closure and the
          // operator can change the destination inside the dialog.
          const pendingGroupId = launcherGroupRef.current;
          if (pendingGroupId) {
            pendingGroupBySessionRef.current.set(session.id, pendingGroupId);
          }
          setLauncherGroupId(null);
        } else {
          setLauncherGroupId(null);
        }
      } catch (error) {
        setLauncherGroupId(null);
        throw error;
      }
    },
    [launchSession],
  );

  const handleNewSessionInGroup = useCallback((groupId: string) => {
    setLauncherGroupId(groupId);
    setMobileSidebarOpen(false);
    setLauncherOpen(true);
  }, []);

  const handleRetry = useCallback(() => {
    clearError();
    handleLaunch();
  }, [clearError, handleLaunch]);

  // Stable callback for the (memoized) TabBar so a conversation event landing
  // in the store doesn't re-render the whole tab strip via an inline arrow.
  const handleNewTerminal = useCallback(() => { handleLaunch(); }, [handleLaunch]);

  const releaseWorkspacePane = useCallback(
    (sessionId: string) => {
      removeWorkspacePane(sessionId);
      clearConversationSession(sessionId);
      exitedSessionsRef.current.delete(sessionId);
      try { localStorage.removeItem(`wc-mobile-draft-${sessionId}`); } catch { /* ignore */ }
    },
    [clearConversationSession, removeWorkspacePane],
  );

  const handleRequestClose = useCallback(
    async (sessionId: string) => {
      const index = workspace.panes.findIndex((pane) => pane.sessionId === sessionId);
      const pane = workspace.panes[index];
      const outcome = await removeSessionPane(sessionId);
      if (outcome === "failed") return;
      releaseWorkspacePane(sessionId);
      if (outcome === "undoable" && pane) {
        setArchiveUndo({ sessionId, pane, index });
      }
    },
    [releaseWorkspacePane, removeSessionPane, workspace.panes],
  );

  useEffect(() => {
    if (!archiveUndo) return;
    const timer = window.setTimeout(() => { setArchiveUndo(null); }, 8_000);
    return () => { window.clearTimeout(timer); };
  }, [archiveUndo]);

  const handleUndoArchive = useCallback(async () => {
    if (!archiveUndo) return;
    if (await undoArchive(archiveUndo.sessionId)) {
      useWorkspaceStore.setState((state) => {
        if (state.panes.some((pane) => pane.sessionId === archiveUndo.sessionId)) return state;
        const panes = [...state.panes];
        panes.splice(Math.min(archiveUndo.index, panes.length), 0, archiveUndo.pane);
        return { panes, activePane: archiveUndo.sessionId };
      });
    }
    setArchiveUndo(null);
  }, [archiveUndo, undoArchive]);

  const handleConfirmDelete = useCallback(async () => {
    if (!pendingDelete) return;
    await deletePanePermanently(pendingDelete);
    releaseWorkspacePane(pendingDelete);
    setPendingDelete(null);
  }, [deletePanePermanently, pendingDelete, releaseWorkspacePane]);

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
    (data: string, intent: Exclude<InputIntent, "control">): GateResult => {
      return submitToActiveTerminal(data, intent, workspace.activePane ?? undefined);
    },
    [submitToActiveTerminal, workspace.activePane],
  );

  const handleSubscribeInputSettled = useCallback(
    (cb: (offset: number, ok: boolean) => void) =>
      subscribeActiveInputSettled(workspace.activePane ?? undefined, cb),
    [subscribeActiveInputSettled, workspace.activePane],
  );

  const handleAwaitInputOffset = useCallback(
    (offset: number, cb: Parameters<NonNullable<React.ComponentProps<typeof MobileToolbar>["awaitOffset"]>>[1]) =>
      awaitActiveInputOffset(workspace.activePane ?? undefined, offset, cb),
    [awaitActiveInputOffset, workspace.activePane],
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

  const handleDiscardPendingInput = useCallback(
    (index: number) => { discardActivePendingInput(index, workspace.activePane ?? undefined); },
    [discardActivePendingInput, workspace.activePane],
  );
  const handleDiscardAllPendingInput = useCallback(
    () => { discardAllActivePendingInput(workspace.activePane ?? undefined); },
    [discardAllActivePendingInput, workspace.activePane],
  );
  const handleFlushPendingInputNow = useCallback(
    () => { flushActivePendingInputNow(workspace.activePane ?? undefined); },
    [flushActivePendingInputNow, workspace.activePane],
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
    // Don't steal focus from open modals or the full-screen composer (which
    // owns focus while open and restores it to the opener on close).
    if (workspace.settingsModalOpen || workspace.aiModalOpen || workspace.aiSuggestActive || workspace.appearanceModalPane !== null || composerOpen) return;
    const paneId = workspace.activePane;
    const rafId = requestAnimationFrame(() => {
      focusActiveTerminal(paneId);
    });
    return () => { cancelAnimationFrame(rafId); };
  }, [workspace.activePane, isMobile, workspace.settingsModalOpen, workspace.aiModalOpen, workspace.aiSuggestActive, workspace.appearanceModalPane, composerOpen, focusActiveTerminal]);

  const handleVoiceTranscript = useCallback((text: string) => {
    if (composerOpen) {
      // Dictating into the full-screen composer — insert at its caret.
      composerDraft.appendAtCaret(text);
    } else if (isMobile) {
      // On mobile, inject into the toolbar text box for review before sending
      mobileToolbarRef.current?.appendText(text);
    } else {
      handleSendToTerminal(text, "bulk_text");
    }
  }, [composerOpen, composerDraft, isMobile, handleSendToTerminal]);

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
          if (active) void handleRequestClose(active);
        },
        sendToTerminal: (data: string) => { handleSendToTerminal(data, "bulk_text"); },
        copySelection: () => { void copySelectionOnPane(activeWorkspacePane ?? undefined); },
        pasteFromClipboard: () => { void pasteFromClipboardOnPane(activeWorkspacePane ?? undefined); },
        scrollTerminal: (lines: number) => { scrollTerminalOnPane(lines, activeWorkspacePane ?? undefined); },
        exitVoiceMode: () => { voiceInput.stopRecording(); },
      }, suggestion.args);
    });
  }, [activeWorkspacePane, voiceInput, handleSendToTerminal, handleLaunch, handleRequestClose, setActiveWorkspacePane, workspacePanes, copySelectionOnPane, pasteFromClipboardOnPane, scrollTerminalOnPane]);

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
  const fallbackTtsPlayback = useMemo<TTSPlaybackState>(() => ({
    ...FALLBACK_TTS_PLAYBACK,
    isMuted: workspace.startMutedOnLoad,
  }), [workspace.startMutedOnLoad]);
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
    return () => { clearInterval(id); };
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

  // In-flight guard for the unlock gesture. It has to live here rather than
  // inside the banner because the banner is now a descriptor, not a component
  // — and the button must stay disabled across the await either way.
  const [enablingAudio, setEnablingAudio] = useState(false);

  const handleEnableAudio = useCallback(async (): Promise<boolean> => {
    if (!enableAudio || enablingAudio) return false;
    setEnablingAudio(true);
    try {
      const ok = await enableAudio.enable();
      if (ok) setEnableAudio(null);
      return ok;
    } finally {
      setEnablingAudio(false);
    }
  }, [enableAudio, enablingAudio]);

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
    onSummarizeFailed: (sessionId, eventId, message) => { handleSummarizeFailed(sessionId, eventId, message, "on-demand"); },
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

  const renderViewToggleButton = useCallback(() => (
    <IconButton
      data-testid="workspace-toggle-view"
      data-view-mode={activeViewMode}
      // This one control is rendered from two different parents — floating
      // over the terminal in one view, inline in the messages toolbar in the
      // other — so switching views remounts it. Without a stable identity the
      // new instance has no memory of the icon it replaced and skips the swap.
      swapIdentity="workspace-view-toggle"
      surface="soft"
      size="xs"
      denseTapTarget
      onClick={() => {
        if (workspace.activePane) {
          handlePaneToggleView(workspace.activePane, activeViewMode);
        }
      }}
      // Dimmed rather than `pending`: the view flips synchronously on click
      // while the pending window stays open through hydration, so hiding the
      // glyph would hide the icon swap that is the whole point of the control.
      disabled={viewSwitchPendingPane === workspace.activePane}
      aria-label={activeViewMode === "terminal" ? t(strings.workspace.switchToMessagesTitle) : t(strings.workspace.switchToTerminalTitle)}
    >
      {activeViewMode === "terminal" ? <MessageSquareText /> : <TerminalSquare />}
    </IconButton>
  ), [activeViewMode, handlePaneToggleView, t, viewSwitchPendingPane, workspace.activePane]);

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
  // The toolbar image button no longer injects "path\n" immediately. Picking an
  // image now STAGES it into the composer for review and opens the composer, so
  // the operator can add text and batch several images into one deliberate send.
  const mobileFileInputRef = useRef<HTMLInputElement>(null);

  const handleMobileUploadImage = useCallback(() => {
    mobileFileInputRef.current?.click();
  }, []);

  const handleMobileFileChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []);
    e.target.value = "";
    if (files.length === 0) return;
    composerAttachments.addFiles(files);
    openComposer();
  }, [composerAttachments, openComposer]);

  // Upload every staged file on send, resolving terminal paths in attachment
  // order. Rejecting keeps the staged attachments + typed text intact so the
  // operator loses nothing; the composer surfaces a retryable error.
  const resolveComposerAttachmentPaths = useCallback(async (): Promise<string[]> => {
    const pane = workspace.activePane;
    if (!pane) throw new Error("no active pane");
    const staged = composerAttachments.attachments;
    const paths: string[] = [];
    try {
      for (const att of staged) {
        composerAttachments.setStatus(att.id, "uploading");
        const path = await uploadFile(pane, att.file);
        paths.push(path);
        composerAttachments.setStatus(att.id, "staged");
      }
    } catch (err) {
      // Clear the spinners so the user can retry the whole batch.
      for (const att of staged) composerAttachments.setStatus(att.id, "staged");
      throw err;
    }
    return paths;
  }, [workspace.activePane, composerAttachments]);

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
  // Session lifecycle events (created/deleted/terminated from any origin) merge
  // into / drop from the pane list live; the store-reconciliation effect above
  // propagates those to the sidebar without a re-hydration.
  useGlobalEventStream({
    onSummarizeError: handlePaneSummarizeError,
    onSessionCreated: mergeExternalSession,
    onSessionEnded: endExternalSession,
  });
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

  // ── Top-chrome notices ──────────────────────────────────────────────────
  // Every condition that can raise a banner, declared in one place as data.
  // Falsy entries are inactive conditions. `BannerRegion` sorts by priority,
  // shows the top one, and collapses the rest — nothing here decides layout,
  // and no condition can quietly render a second surface.
  //
  // Assembled before the early returns below so the empty and hydrating
  // states get the same arbitration as the populated workspace.
  const sessionRecoveryNotice = useSessionRecoveryBanner();
  const crashRecoveryNotice = useCrashRecoveryBanner(openCrashArchive);

  const banners: MaybeBanner[] = [
    ...appBanners,
    sessionRecoveryNotice,
    crashRecoveryNotice,
    voiceInput.fallbackNotice &&
      voiceFallbackBanner(t, voiceInput.fallbackNotice, voiceInput.dismissFallbackNotice),
    voiceInput.rejectedAudio &&
      voiceRejectionBanner(t, voiceInput.rejectedAudio, {
        onRetry: () => { void voiceInput.retryWithoutFilter(); },
        onDismiss: voiceInput.dismissRejection,
      }),
    // What used to be one `VoiceRecoveryBanner` holding five unrelated
    // conditions and a row of buttons. They are independent states with
    // independent urgency, so they are independent banners.
    voiceInput.error && voiceErrorBanner(t, voiceInput.error),
    voiceInput.isTranscribing && voiceTranscribingBanner(t, handleVoiceCancel),
    voiceInput.staleLiveMicLease && voiceStaleMicBanner(t, voiceInput.releaseMicrophone),
    isTtsSpeaking && ttsSpeakingBanner(t, handleTtsStop),
    summarizeError &&
      summarizeErrorBanner(t, summarizeError, {
        onRetry: handleRetrySummarize,
        onDismiss: handleDismissSummarizeError,
      }),
    enableAudio &&
      !enableAudioSuppressed &&
      enableAudioBanner(t, {
        enabling: enablingAudio,
        onEnable: () => void handleEnableAudio(),
        onDismiss: handleDismissEnableAudio,
      }),
    // Only once there are panes. In the empty state the same failure already
    // renders as a card directly beneath the "New terminal" button that
    // produced it, which is the better place for it — a top-chrome copy would
    // just say the same thing twice.
    sessionPanes.length > 0 &&
      createError &&
      createErrorBanner(t, createError, {
        onDismiss: clearError,
        onRetry: createError.retry ? handleRetry : undefined,
      }),
    activeSessionTrackingDegraded && activeViewMode === "messages" && trackingDegradedBanner(t),
  ];

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
        <TopSafeArea testId="workspace-top-edge">
          <BannerRegion banners={banners} />
        </TopSafeArea>
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
                onRetry={hydrationError.retry ? () => { window.location.reload(); } : undefined}
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
          availableTargets={availableTargets}
          targetCatalog={targetCatalog}
          targetsLoading={targetsLoading}
          onRefreshTargets={refreshTargetCatalog}
          onOpenMachines={openMachines}
          groups={workspaceGroups}
          pendingGroupId={launcherGroupId}
          onDestinationChange={setLauncherGroupId}
          onCreateGroup={createLauncherGroup}
          appearance={launcherAppearance}
          onCreateGroupFromRoles={createGroupFromRoles}
          onEditShortcuts={openShortcutSettings}
          onEditTemplates={openTemplateSettings}
        />
        <ArchiveDrawer
          open={archiveDrawerOpen}
          initialSessionId={archiveInitialSessionId}
          onClose={closeArchiveDrawer}
          activeSessionId={workspace.activePane}
          onSendToComposer={sendArchivedMessageToComposer}
          onReopened={handleArchiveReopened}
          preferOrphans={archivePreferOrphans}
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
        playbackState={ttsPlayback ?? fallbackTtsPlayback}
        onSetPlaybackRate={handleTtsSetPlaybackRate}
        onSetVolume={handleTtsSetVolume}
        onSetMuted={handleTtsSetMuted}
        playbackFocusRequest={workspace.activePane === paneMeta.sessionId ? playbackFocusRequest : null}
        onActivate={activatePane}
        onRequestClose={handleRequestClose}
        onHandoff={openHandoff}
        onToggleView={handlePaneToggleView}
        onViewSwitchPendingChange={handleViewSwitchPendingChange}
        onStartArrangeDrag={startArrangeDrag}
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
    roles: workspace.roles,
    activePane: workspace.activePane,
    conversationSessions,
    viewModes: conversationViewModes,
    lastVisitedBySession,
    sortMode: workspace.sidebarSortMode,
  });
  // Provenance per session for the origin-bucketed sidebar. Origin lives on the
  // session (not the workspace pane metadata), so it comes from the session
  // manager's pane list rather than the store.
  const originBySession: Record<string, SessionOriginName> = {};
  for (const sp of sessionPanes) originBySession[sp.session.id] = sp.session.origin;
  const sidebarOriginBuckets = buildOriginBucketedNavigation({
    panes: orderedPanes,
    groups: workspace.groups ?? [],
    roles: workspace.roles,
    activePane: workspace.activePane,
    conversationSessions,
    viewModes: conversationViewModes,
    lastVisitedBySession,
    sortMode: workspace.sidebarSortMode,
    originBySession,
  });
  const activeNavigationItem = navigationItems.find(
    (item) => item.kind === "pane" && item.pane.sessionId === workspace.activePane,
  );
  const activeSidebarPane = activeNavigationItem?.kind === "pane" ? activeNavigationItem : null;
  const sidebarUnreadCount = countWorkspaceUnreadMessages(orderedPanes, conversationSessions);
  // With the mobile sidebar closed, this button is the only signal there is —
  // so a manually flagged session has to reach it too, as a dot (it has no
  // count of its own, and a real unread count outranks it).
  const sidebarHasFlagged = orderedPanes.some((pane) => pane.manuallyUnread);
  const hasTopChrome = workspace.displayMode === "tabs" || workspace.displayMode === "sidebar";
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
        onOpenSettings={() => { workspace.setSettingsModalOpen(true); }}
        onOpenMachines={openMachines}
        onOpenAi={() => { workspace.setAiModalOpen(true); }}
        onNewTerminal={() => handleLaunch()}
        onOpenLauncher={openLauncher}
        onExpandComposer={openComposer}
        isCreating={isCreating}
        voiceSupported={voiceInput.supported}
        voicePreparing={voiceInput.isPreparing}
        voiceRecording={voiceInput.isRecording}
        voicePersistentMode={workspace.persistentMode}
        voiceListening={voiceInput.isListening}
        voicePassive={voiceInput.isPassive}
        voiceTranscribing={voiceInput.isTranscribing}
        voiceError={voiceInput.error}
        voiceLevel={voiceInput.audioLevel}
        voiceActivity={voiceInput.voiceActivity}
        voiceBackend={voiceInput.backend}
        onVoicePrepare={voiceInput.prepareRecording}
        onVoiceStart={handleVoiceStart}
        onVoiceStop={handleVoiceStop}
        onVoiceExitPassive={voiceInput.exitPassiveMode}
      />

      <TopSafeArea testId="workspace-top-edge">
        <BannerRegion banners={banners} />

        {/* Tab bar (only in tabs mode) */}
        {workspace.displayMode === "tabs" && (
          <TabBar
            panes={orderedPanes}
            activePane={workspace.activePane}
            onNewTerminal={handleNewTerminal}
            onOpenLauncher={openLauncher}
            onClosePane={handleRequestClose}
            onDeletePanePermanently={setPendingDelete}
            isCreating={isCreating}
            onStartRole={startRoleFromSurface}
            onOpenRoleMenu={openRoleMenu}
            trailingActions={isMobile ? (
              <>
              <Button
                data-testid="tabbar-machines"
                variant="ghost"
                size="icon"
                className="h-11 w-11 shrink-0 mx-1 self-center md:h-7 md:w-7"
                onClick={() => { setMachinesOpen(true); }}
                title={t(strings.machines.openAriaLabel)}
              >
                <MonitorSmartphone className="h-4 w-4" />
              </Button>
              <Button
                data-testid="tabbar-settings"
                variant="ghost"
                size="icon"
                className="h-11 w-11 shrink-0 mx-1 self-center md:h-7 md:w-7"
                onClick={() => { workspace.setSettingsModalOpen(true); }}
                title={t(strings.workspace.settingsTitle)}
              >
                <Settings className="h-4 w-4" />
              </Button>
              </>
            ) : undefined}
          />
        )}

        {workspace.displayMode === "sidebar" && (
          <div
            data-testid="workspace-sidebar-topbar"
            className="wc-chrome-surface wc-chrome-fg flex h-10 shrink-0 items-center gap-2 border-b border-wc-default ps-[max(0.5rem,var(--wc-safe-left,0px))] pe-[max(0.5rem,var(--wc-safe-right,0px))] md:hidden"
          >
            <Button
              data-testid="workspace-sidebar-toggle"
              variant="ghost"
              size="icon"
              className="relative h-8 w-8"
              onClick={() => { setMobileSidebarOpen(true); }}
              title={t(strings.sessionSidebar.open)}
            >
              <Menu className="h-4 w-4" />
              {sidebarUnreadCount === 0 && sidebarHasFlagged && (
                <span
                  data-testid="workspace-sidebar-toggle-flagged"
                  className="absolute -end-0.5 -top-0.5 h-2 w-2 rounded-full bg-wc-accent"
                />
              )}
              {sidebarUnreadCount > 0 && (
                <span
                  data-testid="workspace-sidebar-toggle-unread"
                  className="absolute -end-1 -top-1 min-w-4 rounded-full bg-wc-accent px-1 text-[10px] font-semibold leading-4 text-wc-accent-fg"
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
              <span className="rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-wc-accent-fg">
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
              data-testid="workspace-machines"
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => { setMachinesOpen(true); }}
              title={t(strings.machines.openAriaLabel)}
            >
              <MonitorSmartphone className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => { workspace.setSettingsModalOpen(true); }}
              title={t(strings.workspace.settingsTitle)}
            >
              <Settings className="h-4 w-4" />
            </Button>
          </div>
        )}
      </TopSafeArea>

      {/* Main content area */}
      {isTabLikeMode ? (
        <div
          ref={workspace.displayMode === "sidebar" ? sidebarLayoutRef : undefined}
          className="flex flex-1 min-w-0 min-h-0 overflow-hidden"
        >
          {workspace.displayMode === "sidebar" && (
            <SessionSidebar
              buckets={sidebarOriginBuckets}
              containerRef={sidebarLayoutRef}
              isMobile={isMobile}
              mobileOpen={mobileSidebarOpen}
              isCreating={isCreating}
              onCloseMobile={() => { setMobileSidebarOpen(false); }}
              onOpenMobile={() => { setMobileSidebarOpen(true); }}
              onActivatePane={activatePane}
              onClosePane={handleRequestClose}
              onDeletePanePermanently={setPendingDelete}
              onNewTerminal={() => handleLaunch()}
              onOpenLauncher={openLauncher}
              onNewSessionInGroup={handleNewSessionInGroup}
          onStartRole={startRoleFromSurface}
          onHandoffToRole={handoffToRole}
          onOpenRoleMenu={openRoleMenu}
              onOpenSettings={() => { workspace.setSettingsModalOpen(true); }}
              onOpenArchiveDrawer={(sessionId) => {
                setMobileSidebarOpen(false);
                setArchiveInitialSessionId(sessionId ?? null);
                setArchiveDrawerOpen(true);
              }}
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
          {activeViewMode === "terminal" && workspace.activePane && workspace.panes.find((pane) => pane.sessionId === workspace.activePane)?.supportsMessagesView && (
            <div className="absolute end-2 top-2.5 z-wc-chrome-raised">
              {renderViewToggleButton()}
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
                  playbackState={ttsPlayback ?? fallbackTtsPlayback}
                  onSetPlaybackRate={handleTtsSetPlaybackRate}
                  onSetVolume={handleTtsSetVolume}
                  onSetMuted={handleTtsSetMuted}
                  playbackFocusRequest={workspace.activePane === paneMeta.sessionId ? playbackFocusRequest : null}
                  onActivate={activatePane}
                  onRequestClose={handleRequestClose}
                  onHandoff={openHandoff}
                  onToggleView={handlePaneToggleView}
                  onViewSwitchPendingChange={handleViewSwitchPendingChange}
                  messagesToolbarTrailingAction={activeViewMode === "messages" && paneMeta.sessionId === workspace.activePane ? renderViewToggleButton() : undefined}
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
        <div className="relative flex-1 min-w-0 min-h-0 overflow-hidden">
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
      <div className="relative z-wc-chrome shrink-0">
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
            ? (ttsPlayback ?? fallbackTtsPlayback)
            : { ...(ttsPlayback ?? fallbackTtsPlayback), isPaused: true };
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
          awaitOffset={handleAwaitInputOffset}
          subscribePendingInput={handleSubscribePendingInput}
          getPendingInputSnapshot={handleGetPendingInputSnapshot}
          discardPendingInput={handleDiscardPendingInput}
          discardAllPendingInput={handleDiscardAllPendingInput}
          flushPendingInputNow={handleFlushPendingInputNow}
          onFocusTerminal={handleFocusTerminal}
          activeSessionId={workspace.activePane}
          draft={composerDraft}
          onExpandComposer={openComposer}
          voice={{
            supported: voiceInput.supported,
            preparing: voiceInput.isPreparing,
            recording: voiceInput.isRecording,
            persistentMode: workspace.persistentMode,
            listening: voiceInput.isListening,
            passive: voiceInput.isPassive,
            transcribing: voiceInput.isTranscribing,
            error: voiceInput.error,
            level: voiceInput.audioLevel,
            activity: voiceInput.voiceActivity,
            partialTranscript: voiceInput.partialTranscript,
            backend: voiceInput.backend,
            onPrepare: voiceInput.prepareRecording,
            onStart: handleVoiceStart,
            onStop: handleVoiceStop,
            onExitPassive: voiceInput.exitPassiveMode,
            commandSuggestion: voiceInput.commandSuggestion,
            onCommandConfirm: handleVoiceCommandConfirm,
            onCommandDismiss: handleVoiceCommandDismiss,
          }}
          onUploadImage={handleMobileUploadImage}
          onOpenAi={() => { workspace.setAiSuggestActive(!workspace.aiSuggestActive); }}
          onAiSuggestExecute={(cmd) => {
            handleSendToTerminal(cmd, "bulk_text");
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
          multiple
          hidden
          onChange={handleMobileFileChange}
        />
      </div>

      {/* Full-screen composer — a portaled DrawerShell overlay shared by mobile
          (corner expand) and desktop (toolbar button + shortcut). It overlays
          the pane, so the xterm terminal stays mounted and never reflows. */}
      <FullScreenComposer
        open={composerOpen}
        onClose={closeComposer}
        draft={composerDraft}
        onInput={handleSendToTerminal}
        subscribeInputSettled={handleSubscribeInputSettled}
        awaitOffset={handleAwaitInputOffset}
        onFocusTerminal={handleFocusTerminal}
        interimTranscript={voiceInput.partialTranscript}
        attachments={composerAttachments.attachments}
        onAttachFiles={composerAttachments.addFiles}
        onRemoveAttachment={composerAttachments.removeFile}
        resolveAttachmentPaths={resolveComposerAttachmentPaths}
        onClearAttachments={composerAttachments.clearAll}
        mic={voiceInput.supported ? (
          <VoiceMicButton
            testId="voice-mic-btn"
            supported={voiceInput.supported}
            isPreparing={voiceInput.isPreparing}
            isRecording={voiceInput.isRecording}
            persistentMode={workspace.persistentMode}
            isListening={voiceInput.isListening}
            isPassive={voiceInput.isPassive}
            isTranscribing={voiceInput.isTranscribing}
            error={voiceInput.error}
            audioLevel={voiceInput.audioLevel}
            voiceActivity={voiceInput.voiceActivity}
            backend={voiceInput.backend}
            onPrepare={voiceInput.prepareRecording}
            onStart={handleVoiceStart}
            onStop={handleVoiceStop}
            onExitPassive={voiceInput.exitPassiveMode}
            // In the composer the mic is a primary, high-frequency action, so
            // give it a large tap target: the wrapper is a flex box that
            // stretches to the row height (= send button height), the button
            // fills it, min-width keeps it as wide as Send, and the icon is
            // enlarged to suit the bigger button.
            className="flex min-w-[5.5rem]"
            buttonClassName="flex w-full items-center justify-center"
            iconClassName="h-5 w-5"
          />
        ) : undefined}
      />

      {/* Terminal Launcher */}
      <TerminalLauncher
        open={launcherOpen}
        onClose={closeLauncher}
        onLaunch={handleLaunch}
        isCreating={isCreating}
        defaultBackend={defaultBackend}
        defaultPolicy={defaultPolicy}
        availableBackends={availableBackends}
        availableTargets={availableTargets}
        targetCatalog={targetCatalog}
        targetsLoading={targetsLoading}
        onRefreshTargets={refreshTargetCatalog}
        onOpenMachines={openMachines}
        groups={workspaceGroups}
        pendingGroupId={launcherGroupId}
        onDestinationChange={setLauncherGroupId}
        onCreateGroup={createLauncherGroup}
        appearance={launcherAppearance}
        onCreateGroupFromRoles={createGroupFromRoles}
        onEditShortcuts={openShortcutSettings}
        onEditTemplates={openTemplateSettings}
      />

      <MachinesDrawer open={machinesOpen} onClose={() => { setMachinesOpen(false); }} />

      {/* Settings Modal */}
      <SettingsModal
        sessions={sessionPanes}
        onDeleteSession={handleRequestClose}
      />

      {/* Appearance Modal */}
      <AppearanceModal />

      {/* Manage Groups drawer (opened from TabBar / SessionSidebar menus) */}
      <ManageGroupsDrawer />

      {/* Rendered once, here: both the sidebar and the tab strip open the
          group menu, and a dialog owned by whichever one was clicked would
          be two implementations of the same consequences. Closing a session
          goes through the SAME handler the tab menu's Close uses, so it
          archives rather than destroys and the archive drawer can reopen it. */}
      <CloseGroupDialog onCloseGroup={closeGroup} onCloseSession={(sessionId) => { void handleRequestClose(sessionId); }} />

      {/* Handoff: one generic verb, reachable from every surface that has a
          payload worth moving. */}
      <HandoffComposer
        open={handoffState !== null}
        onClose={() => { setHandoffState(null); }}
        sourceLabel={handoffState?.sourceLabel ?? ""}
        payload={handoffState?.payload ?? ""}
        targets={handoffTargets}
        initialSelection={handoffState?.initialSelection}
        onSend={runHandoff}
      />

      <GroupUndoBanner
        onUndo={restoreClosedGroup}
        onDismiss={dismissClosedGroupUndo}
      />

      {roleMenu && (
        <RoleMenu
          role={roleMenu.role}
          position={roleMenu.position}
          onStart={(role) => { setRoleMenu(null); startRoleFromSurface(role); }}
          onRename={(role) => {
            setRoleMenu(null);
            const next = window.prompt(t(strings.roles.roleLabel), role.label);
            if (next && next.trim()) updateRole(role.id, { label: next.trim() });
          }}
          onEditPrompt={(role) => {
            setRoleMenu(null);
            const next = window.prompt(t(strings.roles.roleIncomingPromptHint), role.incomingPrompt);
            if (next !== null) updateRole(role.id, { incomingPrompt: next });
          }}
          onDelete={(role) => { setRoleMenu(null); removeRole(role.id); }}
          onDismiss={() => { setRoleMenu(null); }}
        />
      )}

      <ArchiveDrawer
        open={archiveDrawerOpen}
        initialSessionId={archiveInitialSessionId}
        onClose={closeArchiveDrawer}
        activeSessionId={workspace.activePane}
        onSendToComposer={sendArchivedMessageToComposer}
        onReopened={handleArchiveReopened}
        preferOrphans={archivePreferOrphans}
      />

      {/* AI Modal */}
      <AiInput onExecute={(cmd) => { handleSendToTerminal(cmd, "bulk_text"); }} />

      {/* Permanent deletion remains explicit and confirmation-backed. */}
      <AlertDialog
        open={pendingDelete !== null}
        title={t(strings.confirmDelete.title)}
        description={t(strings.confirmDelete.body, {
          name: workspace.panes.find((p) => p.sessionId === pendingDelete)?.name ?? "terminal",
        })}
        cancelLabel={t(strings.confirmDelete.cancel)}
        confirmLabel={t(strings.confirmDelete.confirm)}
        destructive
        onConfirm={handleConfirmDelete}
        onCancel={() => { setPendingDelete(null); }}
        testIdPrefix="confirm-delete-session"
      />

      {archiveUndo && (
        <div
          role="status"
          data-testid="archive-undo-toast"
          className="fixed bottom-4 start-1/2 z-wc-toast flex -translate-x-1/2 items-center gap-3 rounded-xl border border-wc-default bg-wc-panel px-4 py-3 text-sm text-wc-primary shadow-xl"
        >
          <span>{t(strings.archiveToast.archived, { name: archiveUndo.pane.name })}</span>
          <button
            type="button"
            className="font-semibold text-wc-accent hover:underline"
            onClick={() => void handleUndoArchive()}
          >
            {t(strings.archiveToast.undo)}
          </button>
        </div>
      )}
    </div>
  );
}
