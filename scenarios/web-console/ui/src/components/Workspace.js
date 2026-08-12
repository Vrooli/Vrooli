import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import { Loader2, Menu, MessageSquareText, Plus, Settings, TerminalSquare, X } from "lucide-react";
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
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { resolveWorkspaceLayout, reconcileTrackFractions, buildGridTrackTemplate, updateAdjacentFractions, fractionsMatch, } from "../lib/gridLayout";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";
import { uploadFile } from "../api/uploads";
import { fetchCapabilities } from "../api/capabilities";
import { getSessionDefaults } from "../api/settings";
import { getSession } from "../api/sessions";
import ErrorBanner from "./ErrorBanner";
import GridSplitter from "./GridSplitter";
import TerminalLauncher from "./TerminalLauncher";
import MobileToolbar from "./MobileToolbar";
import AiInput from "./AiInput";
import FloatingToolbar from "./FloatingToolbar";
import FullScreenComposer from "./FullScreenComposer";
import VoiceMicButton from "./VoiceMicButton";
import { useComposerDraft } from "../hooks/useComposerDraft";
import { useComposerAttachments } from "../hooks/useComposerAttachments";
import { useComposerHotkey } from "../hooks/useComposerHotkey";
import VoiceRejectionBanner from "./VoiceRejectionBanner";
import WorkspaceMinimap from "./WorkspaceMinimap";
import SettingsModal from "./SettingsModal";
import AppearanceModal from "./AppearanceModal";
import ManageGroupsDrawer from "./ManageGroupsDrawer";
import { ConfirmDialog } from "./ConfirmDialog";
import WorkspacePaneShell from "./WorkspacePaneShell";
import TabBar from "./TabBar";
import SessionSidebar from "./SessionSidebar";
import AudioPlayerBar from "./AudioPlayerBar";
import SummarizeErrorBanner from "./SummarizeErrorBanner";
import EnableAudioBanner from "./EnableAudioBanner";
import RecoverableSessionsBanner from "./RecoverableSessionsBanner";
import SessionRecoveryBanner from "./SessionRecoveryBanner";
import TopSafeArea from "./TopSafeArea";
import { useConversationStore } from "../stores/useConversationStore";
const FALLBACK_TTS_PLAYBACK = {
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
export default function Workspace({ topSafeAreaReserved = false } = {}) {
    const { t } = useTranslation();
    const { panes: sessionPanes, isHydrated, isCreating, createError, hydrationError, clearError, clearHydrationError, launchSession, removePane: removeSessionPane, mergeExternalSession, endExternalSession, handleExit: sessionHandleExit, submitToActiveTerminal, subscribeActiveInputSettled, subscribeActivePendingInput, getActivePendingInputSnapshot, focusActiveTerminal, registerTerminalRef, stopActiveTts, speakTextOnPane, pauseTtsOnPane, resumeTtsOnPane, seekTtsOnPane, setTtsPlaybackRateOnPane, setTtsVolumeOnPane, setTtsMutedOnPane, getTtsStateOnPane, } = useSessionManager();
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
    const activeWorkspacePane = workspace.activePane;
    const activeSessionTrackingDegraded = sessionPanes.find((pane) => pane.session.id === workspace.activePane)?.session.tracking_degraded;
    const addWorkspacePane = workspace.addPane;
    const setWorkspacePaneGroup = workspace.setPaneGroup;
    const removeWorkspacePane = workspace.removePane;
    const setActiveWorkspacePane = workspace.setActivePane;
    const vadAutoStop = workspace.vadAutoStop;
    const { syncActivePane, syncPaneUpdate, syncPaneOrder, syncPaneMove } = useWorkspaceSync();
    const conversationState = useConversationStore(useShallow((state) => ({
        sessions: state.sessions,
        viewModes: state.viewModes,
        setViewMode: state.setViewMode,
        clearSession: state.clearSession,
        activeViewMode: workspace.activePane ? (state.viewModes[workspace.activePane] ?? "terminal") : "terminal",
    })));
    const { sessions: conversationSessions, viewModes: conversationViewModes, setViewMode: setConversationViewMode, clearSession: clearConversationSession, activeViewMode, } = conversationState;
    // Fetch available backends once on mount (they don't change at runtime)
    const [availableBackends, setAvailableBackends] = useState();
    useEffect(() => {
        let cancelled = false;
        fetchCapabilities().then((caps) => {
            if (cancelled)
                return;
            if (caps.session_backends)
                setAvailableBackends(caps.session_backends);
        }).catch(() => { });
        return () => { cancelled = true; };
    }, []);
    const gridRef = useRef(null);
    const sidebarLayoutRef = useRef(null);
    const scrollContainerRef = useRef(null);
    const activeResizeRef = useRef(null);
    useAppViewport();
    const needsTouchControls = useTouchControls();
    const wakeLockStatus = useWakeLock(workspace.keepScreenAwake);
    const setWakeLockStatus = useWakeLockStatus((s) => s.setStatus);
    useEffect(() => { setWakeLockStatus(wakeLockStatus); }, [wakeLockStatus, setWakeLockStatus]);
    const mobileToolbarRef = useRef(null);
    // Single shared per-session draft: the collapsed toolbar input and the
    // full-screen composer read/write ONE value that cannot diverge.
    const composerDraft = useComposerDraft(workspace.activePane);
    const composerAttachments = useComposerAttachments();
    const [composerOpen, setComposerOpen] = useState(false);
    const openComposer = useCallback(() => setComposerOpen(true), []);
    const closeComposer = useCallback(() => setComposerOpen(false), []);
    // Desktop keyboard shortcut (Ctrl/Cmd+Shift+K) opens the composer.
    useComposerHotkey(openComposer);
    const [launcherOpen, setLauncherOpen] = useState(false);
    const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
    const [lastVisitedBySession, setLastVisitedBySession] = useState({});
    // Fetch session defaults on mount AND each time the launcher opens so
    // the dropdown shows the correct server default immediately.
    const [defaultBackend, setDefaultBackend] = useState("standard");
    const [defaultPolicy, setDefaultPolicy] = useState();
    const fetchDefaults = useCallback(() => {
        getSessionDefaults().then((d) => {
            if (d.default_backend)
                setDefaultBackend(d.default_backend);
            if (d.default_policy)
                setDefaultPolicy(d.default_policy);
        }).catch(() => { });
    }, []);
    // Fetch on mount to avoid the "standard" initial state persisting.
    useEffect(() => { fetchDefaults(); }, [fetchDefaults]);
    // Re-fetch when launcher opens so Settings changes are reflected.
    useEffect(() => {
        if (launcherOpen)
            fetchDefaults();
    }, [launcherOpen, fetchDefaults]);
    const pendingActivePaneRef = useRef(null);
    const pendingGroupBySessionRef = useRef(new Map());
    const pendingLauncherGroupRef = useRef(null);
    const [pendingClose, setPendingClose] = useState(null);
    const exitedSessionsRef = useRef(new Set());
    const activatePane = useCallback((sessionId) => {
        // Activation clears a manual unread flag — but only on a real transition,
        // so flagging the session you are looking at survives until you leave and
        // come back. See the store's setActivePane.
        const clearedUnread = setActiveWorkspacePane(sessionId);
        syncActivePane(workspacePanes.map((pane) => pane.sessionId), sessionId);
        if (clearedUnread)
            syncPaneUpdate(sessionId, { manually_unread: false });
    }, [setActiveWorkspacePane, syncActivePane, syncPaneUpdate, workspacePanes]);
    const isTabLikeMode = isTabLikeDisplayMode(workspace.displayMode);
    // Adaptive app-chrome: tell the imperative chrome controller which pane owns
    // the chrome (the focused pane in single-focus modes), whether tinting is
    // active, and the owner's configured theme background (the detection
    // fallback). The per-pane detector feeds detected colors straight to the
    // controller; this effect only carries the low-frequency config, so it never
    // re-renders on color changes. See lib/chromeTheme.ts.
    const activeChromeThemeId = workspace.panes.find((p) => p.sessionId === workspace.activePane)?.themeId;
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
    const [isMobile, setIsMobile] = useState(() => typeof window !== "undefined" && window.innerWidth < 768);
    useEffect(() => {
        const onResize = () => setIsMobile(window.innerWidth < 768);
        window.addEventListener("resize", onResize);
        return () => window.removeEventListener("resize", onResize);
    }, []);
    // --- Pane drag-and-drop reordering ---
    const [activeArrangeDrag, setActiveArrangeDrag] = useState(null);
    const startArrangeDrag = useCallback((paneId, e) => {
        const idx = workspacePanes.findIndex((p) => p.sessionId === paneId);
        if (idx === -1)
            return;
        e.preventDefault();
        e.stopPropagation();
        e.currentTarget.setPointerCapture(e.pointerId);
        setActiveArrangeDrag({ paneId, dropIndex: idx });
    }, [workspacePanes]);
    useEffect(() => {
        if (activeArrangeDrag === null)
            return;
        const handleMove = (e) => {
            const el = document.elementFromPoint(e.clientX, e.clientY);
            const paneEl = el?.closest("[data-pane-index]");
            if (paneEl) {
                const idx = Number(paneEl.getAttribute("data-pane-index"));
                if (Number.isFinite(idx)) {
                    setActiveArrangeDrag((prev) => prev ? { ...prev, dropIndex: idx } : null);
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
                if (shouldActivate)
                    pendingActivePaneRef.current = null;
                const pendingGroupId = pendingGroupBySessionRef.current.get(sp.session.id) ?? null;
                if (pendingGroupId)
                    pendingGroupBySessionRef.current.delete(sp.session.id);
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
        setWorkspacePaneGroup,
        isHydrated,
        removeWorkspacePane,
        sessionPanes,
        syncPaneOrder,
        syncPaneUpdate,
        workspace.defaultFontSize,
        workspace.defaultHeaderColor,
        workspace.defaultThemeId,
        workspacePanes,
    ]);
    // Auto-set active pane if none is set or the persisted value is stale
    useEffect(() => {
        if (workspacePanes.length === 0)
            return;
        const activePaneExists = workspace.activePane !== null && workspacePanes.some((p) => p.sessionId === workspace.activePane);
        if (!activePaneExists) {
            const lastPane = workspacePanes[workspacePanes.length - 1];
            if (lastPane)
                activatePane(lastPane.sessionId);
        }
    }, [workspace.activePane, workspacePanes, activatePane]);
    const openLauncher = useCallback(() => setLauncherOpen(true), []);
    const closeLauncher = useCallback(() => {
        pendingLauncherGroupRef.current = null;
        setLauncherOpen(false);
    }, []);
    const handleLaunch = useCallback(async (opts) => {
        try {
            const session = await launchSession(opts);
            if (session) {
                setLauncherOpen(false);
                // Mark session for auto-activation. The reconciliation effect
                // will add the pane and activate it atomically in a single
                // zustand set(), avoiding cross-system state races.
                pendingActivePaneRef.current = session.id;
                const pendingGroupId = pendingLauncherGroupRef.current;
                if (pendingGroupId) {
                    pendingGroupBySessionRef.current.set(session.id, pendingGroupId);
                    pendingLauncherGroupRef.current = null;
                }
            }
            else {
                pendingLauncherGroupRef.current = null;
            }
        }
        catch (error) {
            pendingLauncherGroupRef.current = null;
            throw error;
        }
    }, [launchSession]);
    const handleNewSessionInGroup = useCallback((groupId) => {
        pendingLauncherGroupRef.current = groupId;
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
    const doRemovePane = useCallback((sessionId) => {
        removeSessionPane(sessionId);
        removeWorkspacePane(sessionId);
        clearConversationSession(sessionId);
        exitedSessionsRef.current.delete(sessionId);
        try {
            localStorage.removeItem(`wc-mobile-draft-${sessionId}`);
        }
        catch { /* ignore */ }
    }, [clearConversationSession, removeSessionPane, removeWorkspacePane]);
    const handleRequestClose = useCallback(async (sessionId) => {
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
        }
        catch {
            // If the fetch fails (e.g. session already gone), close without dialog
            doRemovePane(sessionId);
            return;
        }
        setPendingClose(sessionId);
    }, [doRemovePane]);
    const handleConfirmClose = useCallback(() => {
        if (pendingClose)
            doRemovePane(pendingClose);
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
        if (!workspace.activePane)
            return;
        setLastVisitedBySession((prev) => ({
            ...prev,
            [workspace.activePane]: new Date().toISOString(),
        }));
    }, [workspace.activePane]);
    const handleExit = useCallback((sessionId) => {
        exitedSessionsRef.current.add(sessionId);
        sessionHandleExit(sessionId);
    }, [sessionHandleExit]);
    const handleSendToTerminal = useCallback((data, source) => {
        return submitToActiveTerminal(data, source, workspace.activePane ?? undefined);
    }, [submitToActiveTerminal, workspace.activePane]);
    const handleSubscribeInputSettled = useCallback((cb) => subscribeActiveInputSettled(workspace.activePane ?? undefined, cb), [subscribeActiveInputSettled, workspace.activePane]);
    const handleSubscribePendingInput = useCallback((cb) => subscribeActivePendingInput(workspace.activePane ?? undefined, cb), [subscribeActivePendingInput, workspace.activePane]);
    const handleGetPendingInputSnapshot = useCallback(() => getActivePendingInputSnapshot(workspace.activePane ?? undefined), [getActivePendingInputSnapshot, workspace.activePane]);
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
    const prevActivePaneRef = useRef(workspace.activePane);
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
        if (!workspace.activePane || isMobile)
            return;
        // Don't steal focus from open modals or the full-screen composer (which
        // owns focus while open and restores it to the opener on close).
        if (workspace.settingsModalOpen || workspace.aiModalOpen || workspace.aiSuggestActive || workspace.appearanceModalPane !== null || composerOpen)
            return;
        const paneId = workspace.activePane;
        const rafId = requestAnimationFrame(() => {
            focusActiveTerminal(paneId);
        });
        return () => cancelAnimationFrame(rafId);
    }, [workspace.activePane, isMobile, workspace.settingsModalOpen, workspace.aiModalOpen, workspace.aiSuggestActive, workspace.appearanceModalPane, composerOpen, focusActiveTerminal]);
    const handleVoiceTranscript = useCallback((text) => {
        if (composerOpen) {
            // Dictating into the full-screen composer — insert at its caret.
            composerDraft.appendAtCaret(text);
        }
        else if (isMobile) {
            // On mobile, inject into the toolbar text box for review before sending
            mobileToolbarRef.current?.appendText(text);
        }
        else {
            handleSendToTerminal(text, "voice");
        }
    }, [composerOpen, composerDraft, isMobile, handleSendToTerminal]);
    const voiceInput = useVoiceInput(handleVoiceTranscript);
    const handleVoiceStart = useCallback((opts) => {
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
    const handleVoiceCommandConfirm = useCallback((suggestion) => {
        voiceInput.dismissCommandSuggestion();
        // Command execution is handled by the command vocabulary in commands.ts.
        import("../hooks/voice/commands").then(({ VOICE_COMMANDS }) => {
            const cmd = VOICE_COMMANDS.find((c) => c.id === suggestion.commandId);
            if (!cmd)
                return;
            cmd.execute({
                createTab: () => handleLaunch(),
                switchToTab: (index) => {
                    const pane = workspacePanes[index - 1];
                    if (pane)
                        setActiveWorkspacePane(pane.sessionId);
                },
                closeTab: () => {
                    const active = activeWorkspacePane;
                    if (active)
                        doRemovePane(active);
                },
                sendToTerminal: (data) => { handleSendToTerminal(data, "voice"); },
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
        if (voiceInput.voiceState !== "recording" || !wasNotRecording)
            return;
        // Small delay: the permission dialog may still be visually dismissing,
        // and focus calls during that animation are sometimes swallowed.
        requestAnimationFrame(() => {
            if (isMobile) {
                mobileToolbarRef.current?.focusInput();
            }
            else if (workspace.activePane) {
                focusActiveTerminal(workspace.activePane);
            }
        });
    }, [voiceInput.voiceState, isMobile, workspace.activePane, focusActiveTerminal]);
    // --- TTS speaking state ---
    // Track which panes are currently speaking so voice input can stop active TTS
    // before recording. Playback presentation belongs to the audio bar.
    const [ttsSpeakingPanes, setTtsSpeakingPanes] = useState(new Set());
    const handleTtsSpeakingChange = useCallback((sessionId, speaking) => {
        setTtsSpeakingPanes(prev => {
            const has = prev.has(sessionId);
            if (speaking && has)
                return prev; // already tracked — no state change
            if (!speaking && !has)
                return prev; // already absent — no state change
            const next = new Set(prev);
            if (speaking)
                next.add(sessionId);
            else
                next.delete(sessionId);
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
    const fallbackTtsPlayback = useMemo(() => ({
        ...FALLBACK_TTS_PLAYBACK,
        isMuted: workspace.startMutedOnLoad,
    }), [workspace.startMutedOnLoad]);
    const [ttsPlayback, setTtsPlayback] = useState(null);
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
            if (state)
                setTtsPlayback(state);
        };
        poll();
        const id = setInterval(poll, 100);
        return () => clearInterval(id);
    }, [isTtsSpeaking, workspace.activePane, getTtsStateOnPane]);
    const handleTtsPause = useCallback(() => {
        if (workspace.activePane)
            pauseTtsOnPane(workspace.activePane);
    }, [workspace.activePane, pauseTtsOnPane]);
    const handleTtsResume = useCallback(() => {
        if (workspace.activePane)
            resumeTtsOnPane(workspace.activePane);
    }, [workspace.activePane, resumeTtsOnPane]);
    const handleTtsSeek = useCallback((seconds) => {
        if (workspace.activePane)
            seekTtsOnPane(workspace.activePane, seconds);
    }, [workspace.activePane, seekTtsOnPane]);
    const handleTtsSetPlaybackRate = useCallback((rate) => {
        if (workspace.activePane)
            setTtsPlaybackRateOnPane(workspace.activePane, rate);
    }, [workspace.activePane, setTtsPlaybackRateOnPane]);
    const handleTtsSetVolume = useCallback((level) => {
        if (workspace.activePane)
            setTtsVolumeOnPane(workspace.activePane, level);
    }, [workspace.activePane, setTtsVolumeOnPane]);
    const handleTtsSetMuted = useCallback((next) => {
        if (workspace.activePane)
            setTtsMutedOnPane(workspace.activePane, next);
    }, [workspace.activePane, setTtsMutedOnPane]);
    const applySummarizeResult = useCallback((sessionId, eventId, speechParagraphs) => {
        const convState = useConversationStore.getState();
        const session = convState.sessions[sessionId];
        if (!session)
            return;
        const updatedEvents = session.events.map((event) => event.id === eventId
            ? {
                ...event,
                summarized: true,
                originalSpeechParagraphs: event.originalSpeechParagraphs ?? event.speechParagraphs,
                speechParagraphs,
            }
            : event);
        useConversationStore.setState({
            sessions: { ...convState.sessions, [sessionId]: { ...session, events: updatedEvents } },
        });
    }, []);
    // --- Messages View / replay-bar TTS controls ---
    const [summarizeError, setSummarizeError] = useState(null);
    // Enable-audio affordance: set by TerminalPane when auto-TTS is rejected by
    // the browser's autoplay policy. `suppressed` is session-scoped — once the
    // user dismisses the banner it never reappears until the tab reloads.
    const [enableAudio, setEnableAudio] = useState(null);
    const [enableAudioSuppressed, setEnableAudioSuppressed] = useState(false);
    const handleNeedsUnlock = useCallback((payload) => {
        setEnableAudio(payload);
    }, []);
    const handleEnableAudio = useCallback(async () => {
        if (!enableAudio)
            return false;
        const ok = await enableAudio.enable();
        if (ok)
            setEnableAudio(null);
        return ok;
    }, [enableAudio]);
    const handleDismissEnableAudio = useCallback(() => {
        setEnableAudioSuppressed(true);
        setEnableAudio(null);
    }, []);
    const handleSummarizeFailed = useCallback((sessionId, eventId, message, source) => {
        setSummarizeError({ sessionId, eventId, message, source, status: "failed" });
    }, []);
    const handleDismissSummarizeError = useCallback(() => {
        setSummarizeError(null);
    }, []);
    const handleSummarizeSucceeded = useCallback((sessionId, eventId) => {
        setSummarizeError((prev) => (prev && prev.sessionId === sessionId && prev.eventId === eventId ? null : prev));
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
    const handlePaneToggleView = useCallback((sessionId, viewMode) => {
        setConversationViewMode(sessionId, viewMode === "terminal" ? "messages" : "terminal");
    }, [setConversationViewMode]);
    // Tracks which pane (if any) is mid view-switch so the tabs-mode floating
    // toggle button can show a loading spinner. Each pane shell reports its own
    // transition state; we only surface the active pane's.
    const [viewSwitchPendingPane, setViewSwitchPendingPane] = useState(null);
    const handleViewSwitchPendingChange = useCallback((sessionId, pending) => {
        setViewSwitchPendingPane((prev) => {
            if (pending)
                return sessionId;
            return prev === sessionId ? null : prev;
        });
    }, []);
    const renderViewToggleButton = useCallback(() => (_jsx("button", { "data-testid": "workspace-toggle-view", "data-view-mode": activeViewMode, className: "flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary backdrop-blur-sm transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary", onClick: () => {
            if (workspace.activePane) {
                handlePaneToggleView(workspace.activePane, activeViewMode);
            }
        }, title: activeViewMode === "terminal" ? t(strings.workspace.switchToMessagesTitle) : t(strings.workspace.switchToTerminalTitle), type: "button", children: viewSwitchPendingPane === workspace.activePane
            ? _jsx(Loader2, { "data-testid": "workspace-toggle-view-pending", className: "h-3.5 w-3.5 animate-spin" })
            : activeViewMode === "terminal"
                ? _jsx(MessageSquareText, { className: "h-3.5 w-3.5" })
                : _jsx(TerminalSquare, { className: "h-3.5 w-3.5" }) })), [activeViewMode, handlePaneToggleView, t, viewSwitchPendingPane, workspace.activePane]);
    const handlePaneTransportSpeakingEvent = useCallback((sessionId, eventId) => {
        if (sessionId === workspace.activePane) {
            handlePaneTransportEventStart(sessionId, eventId);
        }
    }, [handlePaneTransportEventStart, workspace.activePane]);
    const handlePaneSummarizeError = useCallback((sessionId, eventId, message) => {
        handleSummarizeFailed(sessionId, eventId, message, "auto");
    }, [handleSummarizeFailed]);
    const handleRetrySummarize = useCallback(() => {
        setSummarizeError((prev) => {
            if (!prev)
                return prev;
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
    const mobileFileInputRef = useRef(null);
    const handleMobileUploadImage = useCallback(() => {
        mobileFileInputRef.current?.click();
    }, []);
    const handleMobileFileChange = useCallback((e) => {
        const files = Array.from(e.target.files ?? []);
        e.target.value = "";
        if (files.length === 0)
            return;
        composerAttachments.addFiles(files);
        openComposer();
    }, [composerAttachments, openComposer]);
    // Upload every staged file on send, resolving terminal paths in attachment
    // order. Rejecting keeps the staged attachments + typed text intact so the
    // operator loses nothing; the composer surfaces a retryable error.
    const resolveComposerAttachmentPaths = useCallback(async () => {
        const pane = workspace.activePane;
        if (!pane)
            throw new Error("no active pane");
        const staged = composerAttachments.attachments;
        const paths = [];
        try {
            for (const att of staged) {
                composerAttachments.setStatus(att.id, "uploading");
                const path = await uploadFile(pane, att.file);
                paths.push(path);
                composerAttachments.setStatus(att.id, "staged");
            }
        }
        catch (err) {
            // Clear the spinners so the user can retry the whole batch.
            for (const att of staged)
                composerAttachments.setStatus(att.id, "staged");
            throw err;
        }
        return paths;
    }, [workspace.activePane, composerAttachments]);
    // --- Resize logic ---
    const startResize = useCallback((axis, index) => (e) => {
        const container = gridRef.current;
        if (!container)
            return;
        const rect = container.getBoundingClientRect();
        activeResizeRef.current = {
            axis,
            index,
            startPointer: axis === "column" ? e.clientX : e.clientY,
            containerSize: axis === "column" ? rect.width : rect.height,
            startValues: axis === "column"
                ? workspace.columnFractions
                : workspace.rowFractions,
        };
    }, [workspace.columnFractions, workspace.rowFractions]);
    useEffect(() => {
        const handleMove = (e) => {
            const resize = activeResizeRef.current;
            if (!resize)
                return;
            e.preventDefault();
            const delta = (resize.axis === "column" ? e.clientX : e.clientY) -
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
            }
            else {
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
    const [recentSessions, setRecentSessions] = useState([]);
    useEffect(() => {
        const active = workspace.activePane;
        if (!active)
            return;
        setRecentSessions((prev) => {
            if (prev[0] === active)
                return prev;
            return [active, ...prev.filter((id) => id !== active)].slice(0, WARM_SET_SIZE);
        });
    }, [workspace.activePane]);
    const mountedTabSessions = useMemo(() => {
        const existing = new Set(orderedPanes.map((p) => p.sessionId));
        const ids = new Set();
        if (workspace.activePane && existing.has(workspace.activePane))
            ids.add(workspace.activePane);
        for (const id of recentSessions) {
            if (ids.size >= WARM_SET_SIZE)
                break;
            if (existing.has(id))
                ids.add(id);
        }
        return ids;
    }, [orderedPanes, recentSessions, workspace.activePane]);
    const maxColumns = isMobile ? 1 : 2;
    const layout = resolveWorkspaceLayout(orderedPanes.length, maxColumns);
    const colFractions = reconcileTrackFractions(workspace.columnFractions, layout.columns);
    const rowFractions = reconcileTrackFractions(workspace.rowFractions, layout.rows);
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
        if (isTabLikeMode)
            return;
        if (!fractionsMatch(colFractions, workspace.columnFractions)) {
            workspace.setColumnFractions(colFractions);
        }
    }, [colFractions, isTabLikeMode, workspace]);
    useEffect(() => {
        if (isTabLikeMode)
            return;
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
    const sidebarHeaderPressGesture = usePressGesture({
        longPressMs: SIDEBAR_HEADER_LONG_PRESS_MS,
        moveThresholdPx: SIDEBAR_HEADER_PRESS_MOVE_THRESHOLD,
        onTap: () => { },
        onLongPress: (sessionId, point) => {
            workspace.setTabContextMenu({ sessionId, position: { x: point.x, y: point.y } });
        },
    });
    // While session hydration is in flight, show a loading screen to prevent
    // the empty state ("New Terminal" button) from flashing before we know
    // whether any sessions exist.
    if (!isHydrated) {
        return (_jsx("div", { className: "flex h-wc-app items-center justify-center bg-wc-surface-base text-wc-text-muted", children: "Loading..." }));
    }
    // Empty state
    if (sessionPanes.length === 0) {
        return (_jsxs("div", { className: "flex h-wc-app flex-col bg-wc-surface-base text-wc-text-primary", children: [_jsxs(TopSafeArea, { testId: "workspace-top-edge", enabled: !topSafeAreaReserved, fillClassName: "bg-wc-surface-base", children: [_jsx(SessionRecoveryBanner, {}), _jsx(RecoverableSessionsBanner, { onRecovered: (result) => {
                                pendingActivePaneRef.current = result.new_session_id;
                            } })] }), _jsx("div", { className: "flex flex-1 items-center justify-center", children: _jsxs("div", { className: "text-center", children: [_jsx("h1", { className: "text-2xl font-semibold mb-4", children: t(strings.app.title) }), _jsx("p", { className: "text-wc-text-muted mb-6", children: t(strings.workspace.tagline) }), hydrationError && (_jsx(ErrorBanner, { error: hydrationError, onDismiss: clearHydrationError, onRetry: hydrationError.retry ? () => window.location.reload() : undefined, className: "mb-4" })), createError && (_jsx(ErrorBanner, { error: createError, onDismiss: clearError, onRetry: createError.retry ? handleRetry : undefined, className: "mb-4" })), _jsxs(Button, { "data-testid": "new-terminal-button", onClick: openLauncher, disabled: isCreating, size: "lg", children: [_jsx(Plus, { className: "me-2 h-5 w-5" }), isCreating ? t(strings.workspace.creating) : t(strings.workspace.newTerminalButton)] })] }) }), _jsx(TerminalLauncher, { open: launcherOpen, onClose: closeLauncher, onLaunch: handleLaunch, isCreating: isCreating, defaultBackend: defaultBackend, defaultPolicy: defaultPolicy, availableBackends: availableBackends })] }));
    }
    // Build splitter elements
    const columnSplitters = [];
    for (let i = 0; i < colFractions.length - 1; i++) {
        const gridCol = `${2 + i * 2}`;
        columnSplitters.push(_jsx(GridSplitter, { axis: "column", gridColumn: gridCol, gridRow: `1 / -1`, onPointerDown: startResize("column", i), label: `Resize column ${i + 1}` }, `col-${i}`));
    }
    const rowSplitters = [];
    for (let i = 0; i < rowFractions.length - 1; i++) {
        const gridRow = `${2 + i * 2}`;
        rowSplitters.push(_jsx(GridSplitter, { axis: "row", gridColumn: `1 / -1`, gridRow: gridRow, onPointerDown: startResize("row", i), label: `Resize row ${i + 1}` }, `row-${i}`));
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
        const isDropTarget = isDragging &&
            !isBeingDragged &&
            activeArrangeDrag?.dropIndex === idx;
        return (_jsx(WorkspacePaneShell, { paneMeta: paneMeta, layoutMode: "grid", gridColumn: gridColumn, gridRow: gridRow, paneIndex: idx, isActive: workspace.activePane === paneMeta.sessionId, isBeingDragged: isBeingDragged, isDropTarget: isDropTarget, isTtsSpeaking: isTtsSpeaking && workspace.activePane === paneMeta.sessionId, activeSpeakingEventId: workspace.activePane === paneMeta.sessionId ? ttsPlaybackController.activeEventId : null, loadingEventId: workspace.activePane === paneMeta.sessionId ? ttsPlaybackController.loadingEventId : null, summarizeLevel: ttsPlaybackController.summarizeLevel, summarizingEventId: ttsPlaybackController.summarizingEventId, getSummarizeError: getPlaybackSummarizeError, onClearSummarizeError: clearPlaybackSummarizeError, onToggleSummarized: togglePanePlaybackVersion, onChangeLevel: changePaneSummarizeLevel, selectedVersionForEvent: getSelectedPlaybackVersion, playbackState: ttsPlayback ?? fallbackTtsPlayback, onSetPlaybackRate: handleTtsSetPlaybackRate, onSetVolume: handleTtsSetVolume, onSetMuted: handleTtsSetMuted, playbackFocusRequest: workspace.activePane === paneMeta.sessionId ? playbackFocusRequest : null, onActivate: activatePane, onRequestClose: handleRequestClose, onToggleView: handlePaneToggleView, onViewSwitchPendingChange: handleViewSwitchPendingChange, onStartArrangeDrag: startArrangeDrag, onTerminalExit: handleExit, onTerminalRef: registerTerminalRef, onVoiceStart: voiceInput.supported ? voiceInput.startRecording : undefined, onVoiceStop: voiceInput.supported ? voiceInput.stopRecording : undefined, onTtsSpeakingChange: handleTtsSpeakingChange, onSpeakingEventChange: handlePaneTransportSpeakingEvent, onConversationEventReceived: handleConversationEventReceived, onNeedsUnlock: handleNeedsUnlock, onPlayFromHere: playPaneFromHere, onPlayEvent: playPaneEvent }, paneMeta.sessionId));
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
    // Provenance per session for the origin-bucketed sidebar. Origin lives on the
    // session (not the workspace pane metadata), so it comes from the session
    // manager's pane list rather than the store.
    const originBySession = {};
    for (const sp of sessionPanes)
        originBySession[sp.session.id] = sp.session.origin;
    const sidebarOriginBuckets = buildOriginBucketedNavigation({
        panes: orderedPanes,
        groups: workspace.groups ?? [],
        activePane: workspace.activePane,
        conversationSessions,
        viewModes: conversationViewModes,
        lastVisitedBySession,
        sortMode: workspace.sidebarSortMode,
        originBySession,
    });
    const activeNavigationItem = navigationItems.find((item) => item.kind === "pane" && item.pane.sessionId === workspace.activePane);
    const activeSidebarPane = activeNavigationItem?.kind === "pane" ? activeNavigationItem : null;
    const sidebarUnreadCount = countWorkspaceUnreadMessages(orderedPanes, conversationSessions);
    // With the mobile sidebar closed, this button is the only signal there is —
    // so a manually flagged session has to reach it too, as a dot (it has no
    // count of its own, and a real unread count outranks it).
    const sidebarHasFlagged = orderedPanes.some((pane) => pane.manuallyUnread);
    const hasTopChrome = workspace.displayMode === "tabs" || workspace.displayMode === "sidebar";
    const workspaceTopSafeEnabled = !topSafeAreaReserved;
    const statusFillClassName = voiceInput.fallbackNotice
        ? "bg-amber-500/10"
        : hasTopChrome
            ? "wc-chrome-surface"
            : "bg-wc-surface-base";
    // h-wc-app maps to var(--wc-app-height, 100dvh) — the actual visible
    // viewport height set by useAppViewport(). This is the root layout
    // container; all descendants use flex to fill this height.
    // See src/hooks/useAppViewport.ts for the full design rationale.
    return (_jsxs("div", { className: "flex flex-col bg-wc-surface-base text-wc-text-primary h-wc-app", children: [_jsx(FloatingToolbar, { hidden: isMobile && isTabLikeMode, onOpenSettings: () => workspace.setSettingsModalOpen(true), onOpenAi: () => workspace.setAiModalOpen(true), onNewTerminal: () => handleLaunch(), onOpenLauncher: openLauncher, onExpandComposer: openComposer, isCreating: isCreating, voiceSupported: voiceInput.supported, voicePreparing: voiceInput.isPreparing, voiceRecording: voiceInput.isRecording, voicePersistentMode: workspace.persistentMode, voiceListening: voiceInput.isListening, voicePassive: voiceInput.isPassive, voiceTranscribing: voiceInput.isTranscribing, voiceStaleLiveMic: voiceInput.staleLiveMicLease, voiceError: voiceInput.error, voiceLevel: voiceInput.audioLevel, voiceActivity: voiceInput.voiceActivity, voicePartialTranscript: voiceInput.partialTranscript, voiceBackend: voiceInput.backend, voiceCanExportDiagnostic: voiceInput.turnDiagnostic !== null, onVoiceExportDiagnostic: voiceInput.exportTurnDiagnostic, onVoicePrepare: voiceInput.prepareRecording, onVoiceStart: handleVoiceStart, onVoiceStop: handleVoiceStop, onVoiceExitPassive: voiceInput.exitPassiveMode, onVoiceReleaseMic: voiceInput.releaseMicrophone, onVoiceCancel: handleVoiceCancel, isTtsSpeaking: isTtsSpeaking, onTtsStop: handleTtsStop }), _jsxs(TopSafeArea, { testId: "workspace-top-edge", enabled: workspaceTopSafeEnabled, fillClassName: statusFillClassName, children: [voiceInput.fallbackNotice && (_jsxs("div", { "data-testid": "voice-status-banner", className: "wc-stable-theme flex items-center gap-2 border-b border-amber-500/30 bg-amber-500/10 py-1.5 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))] text-xs text-amber-300", role: "status", children: [_jsx("span", { className: "min-w-0 flex-1 break-words", children: voiceInput.fallbackNotice }), _jsx("button", { type: "button", "data-testid": "voice-status-dismiss", onClick: voiceInput.dismissFallbackNotice, className: "shrink-0 rounded border border-amber-400/40 bg-amber-500/20 p-1 text-amber-100 transition active:bg-amber-500/30", "aria-label": t(strings.app.connectionBanner.dismissAriaLabel), title: t(strings.app.connectionBanner.dismissAriaLabel), children: _jsx(X, { className: "h-3.5 w-3.5", "aria-hidden": true }) })] })), voiceInput.rejectedAudio && (_jsx(VoiceRejectionBanner, { rejection: voiceInput.rejectedAudio, onRetry: voiceInput.retryWithoutFilter, onDismiss: voiceInput.dismissRejection })), summarizeError && (_jsx(SummarizeErrorBanner, { state: summarizeError, onRetry: handleRetrySummarize, onDismiss: handleDismissSummarizeError })), enableAudio && !enableAudioSuppressed && (_jsx(EnableAudioBanner, { onEnable: handleEnableAudio, onDismiss: handleDismissEnableAudio })), createError && (_jsx(ErrorBanner, { error: createError, onDismiss: clearError, onRetry: createError.retry ? handleRetry : undefined, className: "border-b border-wc-error" })), _jsx(SessionRecoveryBanner, {}), _jsx(RecoverableSessionsBanner, { onRecovered: (result) => {
                            pendingActivePaneRef.current = result.new_session_id;
                        } }), activeSessionTrackingDegraded && activeViewMode === "messages" && (_jsx("div", { role: "status", className: "border-b border-amber-700/40 bg-amber-900/20 px-3 py-1 text-xs text-amber-100", children: "Conversation tracking may be delayed for this recovered Claude session. Terminal activity is continuing." })), workspace.displayMode === "tabs" && (_jsx(TabBar, { panes: orderedPanes, activePane: workspace.activePane, onNewTerminal: handleNewTerminal, onOpenLauncher: openLauncher, onClosePane: handleRequestClose, isCreating: isCreating, trailingActions: isMobile ? (_jsx(Button, { "data-testid": "tabbar-settings", variant: "ghost", size: "icon", className: "h-7 w-7 shrink-0 mx-1 self-center", onClick: () => workspace.setSettingsModalOpen(true), title: t(strings.workspace.settingsTitle), children: _jsx(Settings, { className: "h-4 w-4" }) })) : undefined })), workspace.displayMode === "sidebar" && (_jsxs("div", { "data-testid": "workspace-sidebar-topbar", className: "wc-chrome-surface wc-chrome-fg flex h-10 shrink-0 items-center gap-2 border-b border-wc-default ps-[max(0.5rem,var(--wc-safe-left,0px))] pe-[max(0.5rem,var(--wc-safe-right,0px))] md:hidden", children: [_jsxs(Button, { "data-testid": "workspace-sidebar-toggle", variant: "ghost", size: "icon", className: "relative h-8 w-8", onClick: () => setMobileSidebarOpen(true), title: t(strings.sessionSidebar.open), children: [_jsx(Menu, { className: "h-4 w-4" }), sidebarUnreadCount === 0 && sidebarHasFlagged && (_jsx("span", { "data-testid": "workspace-sidebar-toggle-flagged", className: "absolute -end-0.5 -top-0.5 h-2 w-2 rounded-full bg-wc-accent" })), sidebarUnreadCount > 0 && (_jsx("span", { "data-testid": "workspace-sidebar-toggle-unread", className: "absolute -end-1 -top-1 min-w-4 rounded-full bg-wc-accent px-1 text-[10px] font-semibold leading-4 text-wc-accent-fg", children: sidebarUnreadCount > 99 ? "99+" : sidebarUnreadCount }))] }), _jsx("div", { "data-testid": "workspace-sidebar-active-title", className: "min-w-0 flex-1 select-none truncate text-sm font-medium touch-manipulation", title: activeSidebarPane?.pane.name ?? t(strings.sessionSidebar.title), ...(activeSidebarPane
                                    ? sidebarHeaderPressGesture.getGestureHandlers(activeSidebarPane.pane.sessionId)
                                    : {}), children: activeSidebarPane?.pane.name ?? t(strings.sessionSidebar.title) }), activeSidebarPane && activeSidebarPane.unreadCount > 0 && (_jsx("span", { className: "rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-wc-accent-fg", children: activeSidebarPane.unreadCount })), _jsx(Button, { variant: "ghost", size: "icon", className: "h-8 w-8", disabled: isCreating, onClick: () => {
                                    if (workspace.plusButtonBehavior === "launcher") {
                                        openLauncher();
                                    }
                                    else {
                                        handleLaunch();
                                    }
                                }, title: workspace.plusButtonBehavior === "launcher" ? t(strings.floatingToolbar.launcherFirstTitle) : t(strings.floatingToolbar.terminalFirstTitle), children: _jsx(Plus, { className: "h-4 w-4" }) }), _jsx(Button, { variant: "ghost", size: "icon", className: "h-8 w-8", onClick: () => workspace.setSettingsModalOpen(true), title: t(strings.workspace.settingsTitle), children: _jsx(Settings, { className: "h-4 w-4" }) })] }))] }), isTabLikeMode ? (_jsxs("div", { ref: workspace.displayMode === "sidebar" ? sidebarLayoutRef : undefined, className: "flex flex-1 min-h-0 overflow-hidden", children: [workspace.displayMode === "sidebar" && (_jsx(SessionSidebar, { buckets: sidebarOriginBuckets, containerRef: sidebarLayoutRef, isMobile: isMobile, mobileOpen: mobileSidebarOpen, isCreating: isCreating, onCloseMobile: () => setMobileSidebarOpen(false), onActivatePane: activatePane, onClosePane: handleRequestClose, onNewTerminal: () => handleLaunch(), onOpenLauncher: openLauncher, onNewSessionInGroup: handleNewSessionInGroup, onOpenSettings: () => workspace.setSettingsModalOpen(true) })), _jsxs("div", { className: "relative flex-1 min-h-0 overflow-hidden", children: [activeViewMode === "terminal" && workspace.activePane && workspace.panes.find((pane) => pane.sessionId === workspace.activePane)?.supportsMessagesView && (_jsx("div", { className: "absolute end-2 top-2.5 z-wc-chrome-raised", children: renderViewToggleButton() })), orderedPanes.filter((paneMeta) => mountedTabSessions.has(paneMeta.sessionId)).map((paneMeta) => {
                                return (_jsx(WorkspacePaneShell, { paneMeta: paneMeta, layoutMode: "tabs", isActive: paneMeta.sessionId === workspace.activePane, isVisible: paneMeta.sessionId === workspace.activePane, isTtsSpeaking: isTtsSpeaking && workspace.activePane === paneMeta.sessionId, activeSpeakingEventId: workspace.activePane === paneMeta.sessionId ? ttsPlaybackController.activeEventId : null, loadingEventId: workspace.activePane === paneMeta.sessionId ? ttsPlaybackController.loadingEventId : null, summarizeLevel: ttsPlaybackController.summarizeLevel, summarizingEventId: ttsPlaybackController.summarizingEventId, getSummarizeError: getPlaybackSummarizeError, onClearSummarizeError: clearPlaybackSummarizeError, onToggleSummarized: togglePanePlaybackVersion, onChangeLevel: changePaneSummarizeLevel, selectedVersionForEvent: getSelectedPlaybackVersion, playbackState: ttsPlayback ?? fallbackTtsPlayback, onSetPlaybackRate: handleTtsSetPlaybackRate, onSetVolume: handleTtsSetVolume, onSetMuted: handleTtsSetMuted, playbackFocusRequest: workspace.activePane === paneMeta.sessionId ? playbackFocusRequest : null, onActivate: activatePane, onRequestClose: handleRequestClose, onToggleView: handlePaneToggleView, onViewSwitchPendingChange: handleViewSwitchPendingChange, messagesToolbarTrailingAction: activeViewMode === "messages" && paneMeta.sessionId === workspace.activePane ? renderViewToggleButton() : undefined, onTerminalExit: handleExit, onTerminalRef: registerTerminalRef, onVoiceStart: voiceInput.supported ? voiceInput.startRecording : undefined, onVoiceStop: voiceInput.supported ? voiceInput.stopRecording : undefined, onTtsSpeakingChange: handleTtsSpeakingChange, onSpeakingEventChange: handlePaneTransportSpeakingEvent, onConversationEventReceived: handleConversationEventReceived, onNeedsUnlock: handleNeedsUnlock, onPlayFromHere: playPaneFromHere, onPlayEvent: playPaneEvent }, paneMeta.sessionId));
                            })] })] })) : (
            /* Grid mode: original grid layout with minimap */
            _jsxs("div", { className: "relative flex-1 min-h-0 overflow-hidden", children: [_jsx("div", { ref: scrollContainerRef, className: cn("absolute inset-0 overflow-auto wc-hide-scrollbar", workspace.isMinimapVisible && "right-[34px]"), children: _jsxs("div", { ref: gridRef, "data-testid": "pane-grid", className: cn("grid gap-0 p-1", isDragging && "select-none cursor-grabbing [&_.xterm]:pointer-events-none"), style: {
                                gridTemplateColumns: colTemplate,
                                gridTemplateRows: rowTemplate,
                                height: `${minimumGridHeightPx}px`,
                                minHeight: `${minimumGridHeightPx}px`,
                            }, children: [paneCells, columnSplitters, rowSplitters] }) }), _jsx(WorkspaceMinimap, { scrollRef: scrollContainerRef, rowCount: layout.rows })] })), _jsxs("div", { className: "relative z-wc-chrome shrink-0", children: [(() => {
                        const pb = isTtsSpeaking
                            ? (ttsPlayback ?? fallbackTtsPlayback)
                            : { ...(ttsPlayback ?? fallbackTtsPlayback), isPaused: true };
                        const context = ttsPlaybackController.buildBarContext(workspace.activePane, workspace.autoTtsEnabled, { playback: ttsPlayback, isSpeaking: isTtsSpeaking });
                        if (!context?.event || !context.sessionId)
                            return null;
                        const activeEvent = context.event;
                        const isReplayMode = !isTtsSpeaking;
                        const hasOriginal = (activeEvent.originalSpeechParagraphs?.length ?? 0) > 0;
                        const canRequestSummarize = activeEvent.role === "assistant";
                        const isPlayingSummarized = context.version === "active" && hasOriginal;
                        return (_jsx(AudioPlayerBar, { isPaused: pb.isPaused, currentTime: pb.currentTime, duration: pb.duration, playbackRate: pb.playbackRate, volume: pb.volume, isMuted: pb.isMuted, capabilities: pb.capabilities, isSummarized: isPlayingSummarized, hasOriginalVersion: hasOriginal, canSummarize: canRequestSummarize, isSummarizing: ttsPlaybackController.summarizingEventId === activeEvent.id, isLoading: ttsPlaybackController.loadingEventId === activeEvent.id, currentLevel: ttsPlaybackController.summarizeLevel, currentMessageLabel: context.queueLabel, currentMessageId: activeEvent.id, messageSelectorEvents: conversationSessions[context.sessionId]?.events ?? [], hasQueuedNext: context.hasQueuedNext, onPause: () => {
                                ttsPlaybackController.pausePlayback(context.sessionId);
                                handleTtsPause();
                            }, onResume: isReplayMode ? () => {
                                ttsPlaybackController.resumePlayback(context.sessionId);
                            } : () => {
                                ttsPlaybackController.resumePlayback(context.sessionId);
                                handleTtsResume();
                            }, onSeek: handleTtsSeek, onSetPlaybackRate: handleTtsSetPlaybackRate, onSetVolume: handleTtsSetVolume, onSetMuted: handleTtsSetMuted, onSelectMessage: (eventId) => {
                                ttsPlaybackController.playEvent(context.sessionId, eventId);
                            }, onToggleSummarized: hasOriginal && activeEvent && workspace.activePane ? (useSummarized) => {
                                ttsPlaybackController.toggleVersion(context.sessionId, activeEvent.id, useSummarized);
                            } : undefined, onChangeLevel: canRequestSummarize && activeEvent && workspace.activePane ? (level) => {
                                ttsPlaybackController.changeSummarizeLevel(context.sessionId, activeEvent.id, level);
                            } : undefined, onDismiss: !workspace.autoTtsEnabled ? () => {
                                ttsPlaybackController.stopPlayback(context.sessionId);
                                stopActiveTts(context.sessionId);
                            } : undefined }));
                    })(), _jsx(MobileToolbar, { ref: mobileToolbarRef, visible: needsTouchControls, onInput: handleSendToTerminal, subscribeInputSettled: handleSubscribeInputSettled, subscribePendingInput: handleSubscribePendingInput, getPendingInputSnapshot: handleGetPendingInputSnapshot, onFocusTerminal: handleFocusTerminal, activeSessionId: workspace.activePane, draft: composerDraft, onExpandComposer: openComposer, voiceSupported: voiceInput.supported, voicePreparing: voiceInput.isPreparing, voiceRecording: voiceInput.isRecording, voicePersistentMode: workspace.persistentMode, voiceListening: voiceInput.isListening, voicePassive: voiceInput.isPassive, voiceTranscribing: voiceInput.isTranscribing, voiceStaleLiveMic: voiceInput.staleLiveMicLease, voiceError: voiceInput.error, voiceLevel: voiceInput.audioLevel, voiceActivity: voiceInput.voiceActivity, voicePartialTranscript: voiceInput.partialTranscript, voiceBackend: voiceInput.backend, voiceCanExportDiagnostic: voiceInput.turnDiagnostic !== null, onVoiceExportDiagnostic: voiceInput.exportTurnDiagnostic, onVoicePrepare: voiceInput.prepareRecording, onVoiceStart: handleVoiceStart, onVoiceStop: handleVoiceStop, onVoiceExitPassive: voiceInput.exitPassiveMode, onVoiceReleaseMic: voiceInput.releaseMicrophone, onVoiceCancel: handleVoiceCancel, voiceCommandSuggestion: voiceInput.commandSuggestion, onVoiceCommandConfirm: handleVoiceCommandConfirm, onVoiceCommandDismiss: handleVoiceCommandDismiss, onUploadImage: handleMobileUploadImage, onOpenAi: () => workspace.setAiSuggestActive(!workspace.aiSuggestActive), aiSuggestActive: workspace.aiSuggestActive, onAiSuggestExecute: (cmd) => {
                            handleSendToTerminal(cmd, "toolbar-submit");
                            mobileToolbarRef.current?.clearInput();
                            workspace.setAiSuggestActive(false);
                        }, isTtsSpeaking: isTtsSpeaking, onTtsStop: handleTtsStop, viewMode: activeViewMode, onSwitchToTerminal: handleSwitchToTerminal }), _jsx("input", { ref: mobileFileInputRef, type: "file", accept: "image/*", multiple: true, hidden: true, onChange: handleMobileFileChange })] }), _jsx(FullScreenComposer, { open: composerOpen, onClose: closeComposer, draft: composerDraft, onInput: handleSendToTerminal, subscribeInputSettled: handleSubscribeInputSettled, onFocusTerminal: handleFocusTerminal, attachments: composerAttachments.attachments, onAttachFiles: composerAttachments.addFiles, onRemoveAttachment: composerAttachments.removeFile, resolveAttachmentPaths: resolveComposerAttachmentPaths, onClearAttachments: composerAttachments.clearAll, mic: voiceInput.supported ? (_jsx(VoiceMicButton, { testId: "voice-mic-btn", supported: voiceInput.supported, isPreparing: voiceInput.isPreparing, isRecording: voiceInput.isRecording, persistentMode: workspace.persistentMode, isListening: voiceInput.isListening, isPassive: voiceInput.isPassive, isTranscribing: voiceInput.isTranscribing, staleLiveMic: voiceInput.staleLiveMicLease, error: voiceInput.error, audioLevel: voiceInput.audioLevel, voiceActivity: voiceInput.voiceActivity, partialTranscript: voiceInput.partialTranscript, backend: voiceInput.backend, canExportDiagnostic: voiceInput.turnDiagnostic !== null, onExportDiagnostic: voiceInput.exportTurnDiagnostic, isTtsSpeaking: isTtsSpeaking, onPrepare: voiceInput.prepareRecording, onStart: handleVoiceStart, onStop: handleVoiceStop, onExitPassive: voiceInput.exitPassiveMode, onReleaseMic: voiceInput.releaseMicrophone, onCancel: handleVoiceCancel, onTtsStop: handleTtsStop, 
                    // In the composer the mic is a primary, high-frequency action, so
                    // give it a large tap target: the wrapper is a flex box that
                    // stretches to the row height (= send button height), the button
                    // fills it, min-width keeps it as wide as Send, and the icon is
                    // enlarged to suit the bigger button.
                    className: "flex min-w-[5.5rem]", buttonClassName: "flex w-full items-center justify-center", iconClassName: "h-5 w-5" })) : undefined }), _jsx(TerminalLauncher, { open: launcherOpen, onClose: closeLauncher, onLaunch: handleLaunch, isCreating: isCreating, defaultBackend: defaultBackend, defaultPolicy: defaultPolicy, availableBackends: availableBackends }), _jsx(SettingsModal, { sessions: sessionPanes, onDeleteSession: handleRequestClose }), _jsx(AppearanceModal, {}), _jsx(ManageGroupsDrawer, {}), _jsx(AiInput, { onExecute: (cmd) => { handleSendToTerminal(cmd, "toolbar-submit"); } }), _jsx(ConfirmDialog, { open: pendingClose !== null, title: t(strings.confirmClose.title), body: t(strings.confirmClose.body, {
                    name: workspace.panes.find((p) => p.sessionId === pendingClose)?.name ?? "terminal",
                }), cancelLabel: t(strings.confirmClose.cancel), confirmLabel: t(strings.confirmClose.confirm), destructive: true, onConfirm: handleConfirmClose, onCancel: handleCancelClose, testIdPrefix: "confirm-close" })] }));
}
