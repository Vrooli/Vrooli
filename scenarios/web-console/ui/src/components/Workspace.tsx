// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useState, useCallback, useEffect, useRef, type ChangeEvent } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { MessageSquareText, Plus, Settings, TerminalSquare } from "lucide-react";
import { SPLITTER_SIZE_PX, MIN_COLUMN_PX, MIN_ROW_PX } from "../consts/config";
import { useSessionManager } from "../hooks/useSessionManager";
import { useVoiceInput } from "../hooks/useVoiceInput";
import { useAppViewport } from "../hooks/useAppViewport";
import { useWakeLock } from "../hooks/useWakeLock";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
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
import { getSession, uploadFile, summarizeEvent, fetchCapabilities, getSessionDefaults, type BackendOption, type BackendID, type ExpirationPolicy } from "../lib/api";
import type { LaunchOptions } from "./TerminalLauncher";
import ErrorBanner from "./ErrorBanner";
import ErrorBoundary from "./ErrorBoundary";
import TerminalPane from "./TerminalPane";
import TerminalHeader from "./TerminalHeader";
import GridSplitter from "./GridSplitter";
import TerminalLauncher from "./TerminalLauncher";
import MobileToolbar from "./MobileToolbar";
import type { MobileToolbarHandle } from "./MobileToolbar";
import AiInput from "./AiInput";
import AiSuggestBar from "./AiSuggestBar";
import FloatingToolbar from "./FloatingToolbar";
import WorkspaceMinimap from "./WorkspaceMinimap";
import SettingsModal from "./SettingsModal";
import AppearanceModal from "./AppearanceModal";
import ConfirmCloseDialog from "./ConfirmCloseDialog";
import TabBar from "./TabBar";
import MessagesPane from "./MessagesPane";
import AudioPlayerBar from "./AudioPlayerBar";
import { useConversationStore } from "../stores/useConversationStore";
import type { TTSPlaybackState } from "../hooks/tts/types";

type ActiveResize = {
  axis: "column" | "row";
  index: number;
  startPointer: number;
  containerSize: number;
  startValues: number[];
};

type ActiveArrangeDrag = { paneId: string; dropIndex: number };

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
  const {
    panes: sessionPanes,
    isHydrated,
    isCreating,
    createError,
    clearError,
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
    speakSequenceOnPane,
    pauseTtsOnPane,
    resumeTtsOnPane,
    seekTtsOnPane,
    setTtsPlaybackRateOnPane,
    setTtsVolumeOnPane,
    getTtsStateOnPane,
  } = useSessionManager();

  const store = useWorkspaceStore();
  const { syncActivePane, syncPaneUpdate } = useWorkspaceSync();
  const setConversationViewMode = useConversationStore((state) => state.setViewMode);

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

  const clearConversationSession = useConversationStore((state) => state.clearSession);
  const conversationSessions = useConversationStore((state) => state.sessions);
  const conversationViewModes = useConversationStore((state) => state.viewModes);
  const gridRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const activeResizeRef = useRef<ActiveResize | null>(null);
  useAppViewport();
  const wakeLockStatus = useWakeLock(store.keepScreenAwake);
  const setWakeLockStatus = useWakeLockStatus((s) => s.setStatus);
  useEffect(() => { setWakeLockStatus(wakeLockStatus); }, [wakeLockStatus, setWakeLockStatus]);
  const mobileToolbarRef = useRef<MobileToolbarHandle>(null);

  const [launcherOpen, setLauncherOpen] = useState(false);

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
  const [mobileInputText, setMobileInputText] = useState("");
  const pendingActivePaneRef = useRef<string | null>(null);
  const [pendingClose, setPendingClose] = useState<string | null>(null);
  const exitedSessionsRef = useRef<Set<string>>(new Set());

  const activatePane = useCallback((sessionId: string) => {
    store.setActivePane(sessionId);
    syncActivePane(store.panes.map((p) => p.sessionId), sessionId);
  }, [store, syncActivePane]);

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
      const idx = store.panes.findIndex((p) => p.sessionId === paneId);
      if (idx === -1) return;
      e.preventDefault();
      e.stopPropagation();
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
      setActiveArrangeDrag({ paneId, dropIndex: idx });
    },
    [store.panes],
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
        if (prev) store.movePaneToIndex(prev.paneId, prev.dropIndex);
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
  }, [activeArrangeDrag, store]);

  // Reconcile session manager panes with workspace store.
  // Only remove stale store panes after session hydration completes —
  // otherwise the initial empty sessionPanes would nuke persisted metadata.
  useEffect(() => {
    const sessionIds = new Set(sessionPanes.map((p) => p.session.id));
    const storeIds = new Set(store.panes.map((p) => p.sessionId));

    // Add new sessions to store (auto-activate if user just created it)
    for (const sp of sessionPanes) {
      if (!storeIds.has(sp.session.id)) {
        const shouldActivate = pendingActivePaneRef.current === sp.session.id;
        if (shouldActivate) pendingActivePaneRef.current = null;
        store.addPane(sp.session.id, sp.session.shell ?? "terminal", shouldActivate, sp.supportsMessagesView);
        // Persist new pane metadata (including supportsMessagesView) to the backend
        syncPaneUpdate(sp.session.id, {
          name: sp.session.shell?.split("/").pop() ?? "terminal",
          supports_messages_view: sp.supportsMessagesView,
        });
      }
    }
    // Remove deleted sessions from store (only after hydration)
    if (isHydrated) {
      for (const sid of storeIds) {
        if (!sessionIds.has(sid)) {
          store.removePane(sid);
        }
      }
    }
  }, [sessionPanes, store, isHydrated, syncPaneUpdate]);

  // Auto-set active pane if none is set or the persisted value is stale
  useEffect(() => {
    if (store.panes.length === 0) return;
    const activePaneExists = store.activePane !== null && store.panes.some((p) => p.sessionId === store.activePane);
    if (!activePaneExists) {
      const lastPane = store.panes[store.panes.length - 1];
      if (lastPane) activatePane(lastPane.sessionId);
    }
  }, [store, activatePane]);

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

  const doRemovePane = useCallback(
    (sessionId: string) => {
      removeSessionPane(sessionId);
      store.removePane(sessionId);
      clearConversationSession(sessionId);
      exitedSessionsRef.current.delete(sessionId);
      try { localStorage.removeItem(`wc-mobile-draft-${sessionId}`); } catch { /* ignore */ }
    },
    [clearConversationSession, removeSessionPane, store],
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

  const handleExit = useCallback((sessionId: string) => {
    exitedSessionsRef.current.add(sessionId);
    sessionHandleExit(sessionId);
  }, [sessionHandleExit]);

  const handleSendToTerminal = useCallback(
    (data: string, source: InputSource): GateResult => {
      return submitToActiveTerminal(data, source, store.activePane ?? undefined);
    },
    [submitToActiveTerminal, store.activePane],
  );

  const handleSubscribeInputSettled = useCallback(
    (cb: (seq: number, ok: boolean) => void) =>
      subscribeActiveInputSettled(store.activePane ?? undefined, cb),
    [subscribeActiveInputSettled, store.activePane],
  );

  const handleSubscribePendingInput = useCallback(
    (cb: () => void) =>
      subscribeActivePendingInput(store.activePane ?? undefined, cb),
    [subscribeActivePendingInput, store.activePane],
  );

  const handleGetPendingInputSnapshot = useCallback(
    () => getActivePendingInputSnapshot(store.activePane ?? undefined),
    [getActivePendingInputSnapshot, store.activePane],
  );

  const handleFocusTerminal = useCallback(() => {
    focusActiveTerminal(store.activePane ?? undefined);
  }, [focusActiveTerminal, store.activePane]);

  // Switch the active pane from messages view back to terminal view.
  // Used by MobileToolbar to auto-switch after sending a command.
  const handleSwitchToTerminal = useCallback(() => {
    if (store.activePane) {
      setConversationViewMode(store.activePane, "terminal");
    }
  }, [store.activePane, setConversationViewMode]);

  // ── Stop TTS on the previous pane when switching tabs ──
  // Each TerminalPane manages its own TTS playback.  When the user switches
  // tabs, the newly-active pane may auto-play pending conversation events
  // (the auto-TTS effect in TerminalPane.tsx).  Without stopping the old
  // pane's TTS first, both audio streams play simultaneously — a jarring
  // experience.  We track the previous active pane and stop its TTS before
  // the new pane's auto-play effect fires.
  const prevActivePaneRef = useRef<string | null>(store.activePane);
  useEffect(() => {
    const prev = prevActivePaneRef.current;
    prevActivePaneRef.current = store.activePane;
    // Stop TTS on the pane we just left (if any)
    if (prev && prev !== store.activePane) {
      stopActiveTts(prev);
    }
  }, [store.activePane, stopActiveTts]);

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
    if (!store.activePane || isMobile) return;
    // Don't steal focus from open modals
    if (store.settingsModalOpen || store.aiModalOpen || store.aiSuggestActive || store.appearanceModalPane !== null) return;
    const paneId = store.activePane;
    const rafId = requestAnimationFrame(() => {
      focusActiveTerminal(paneId);
    });
    return () => cancelAnimationFrame(rafId);
  }, [store.activePane, isMobile, store.settingsModalOpen, store.aiModalOpen, store.aiSuggestActive, store.appearanceModalPane, focusActiveTerminal]);

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
    stopActiveTts(store.activePane ?? undefined);
    const vadAutoStop = useWorkspaceStore.getState().vadAutoStop;
    voiceInput.startRecording({ vadEnabled: vadAutoStop && opts?.vadEnabled });
  }, [store.activePane, stopActiveTts, voiceInput]);

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
          const pane = store.panes[index - 1];
          if (pane) store.setActivePane(pane.sessionId);
        },
        closeTab: () => {
          const active = store.activePane;
          if (active) doRemovePane(active);
        },
        sendToTerminal: (data: string) => { handleSendToTerminal(data, "voice"); },
        exitVoiceMode: () => voiceInput.stopRecording(),
      }, suggestion.args);
    });
  }, [voiceInput, handleSendToTerminal, handleLaunch, doRemovePane, store]);

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
      } else if (store.activePane) {
        focusActiveTerminal(store.activePane);
      }
    });
  }, [voiceInput.voiceState, isMobile, store.activePane, focusActiveTerminal]);

  // --- TTS speaking indicator ---
  // Track which panes are currently speaking so we can show a visual indicator
  // on the mic button. We only care about the active pane's speaking state.
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
  const isTtsSpeaking = store.activePane ? ttsSpeakingPanes.has(store.activePane) : false;
  const handleTtsStop = useCallback(() => {
    stopActiveTts(store.activePane ?? undefined);
  }, [store.activePane, stopActiveTts]);

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
  // Fix: the AudioPlayerBar now renders whenever `isTtsSpeaking` is
  // true, using `FALLBACK_TTS_PLAYBACK` when the poll hasn't returned
  // yet.  The fallback has sensible defaults (not paused, no duration,
  // playbackRate 1, volume 1) and exposes all capabilities so every
  // control is visible — the real provider values replace it within the
  // first poll tick.
  const FALLBACK_TTS_PLAYBACK: TTSPlaybackState = {
    currentTime: 0,
    duration: null,
    isPaused: false,
    playbackRate: 1,
    volume: 1,
    capabilities: { canPause: true, canSeek: false, canAdjustSpeed: true, canAdjustVolume: true },
  };
  const [ttsPlayback, setTtsPlayback] = useState<TTSPlaybackState | null>(null);
  useEffect(() => {
    if (!isTtsSpeaking || !store.activePane) {
      setTtsPlayback(null);
      return;
    }
    const activePane = store.activePane;
    // Poll immediately on start — don't wait for the first interval tick.
    // This closes the gap where audio is playing but the bar is invisible.
    const poll = () => {
      const state = getTtsStateOnPane(activePane);
      if (state) setTtsPlayback(state);
    };
    poll();
    const id = setInterval(poll, 100);
    return () => clearInterval(id);
  }, [isTtsSpeaking, store.activePane, getTtsStateOnPane]);

  const handleTtsPause = useCallback(() => {
    if (store.activePane) pauseTtsOnPane(store.activePane);
  }, [store.activePane, pauseTtsOnPane]);

  const handleTtsResume = useCallback(() => {
    if (store.activePane) resumeTtsOnPane(store.activePane);
  }, [store.activePane, resumeTtsOnPane]);

  const handleTtsSeek = useCallback((seconds: number) => {
    if (store.activePane) seekTtsOnPane(store.activePane, seconds);
  }, [store.activePane, seekTtsOnPane]);

  const handleTtsSetPlaybackRate = useCallback((rate: number) => {
    if (store.activePane) setTtsPlaybackRateOnPane(store.activePane, rate);
  }, [store.activePane, setTtsPlaybackRateOnPane]);

  const handleTtsSetVolume = useCallback((level: number) => {
    if (store.activePane) setTtsVolumeOnPane(store.activePane, level);
  }, [store.activePane, setTtsVolumeOnPane]);

  // --- Messages View TTS controls ---
  const [activeSpeakingEventId, setActiveSpeakingEventId] = useState<string | null>(null);
  const [isSummarizing, setIsSummarizing] = useState(false);

  // Persistent last-played event for the replay bar.  Unlike
  // activeSpeakingEventId (which is cleared when TTS stops), these survive
  // so the AudioPlayerBar can stay visible in a "replay" state when auto-TTS
  // is enabled.
  const [lastTtsEventId, setLastTtsEventId] = useState<string | null>(null);
  const [lastTtsPaneId, setLastTtsPaneId] = useState<string | null>(null);

  // Clear the active speaking indicator when TTS stops
  const prevTtsSpeaking = useRef(isTtsSpeaking);
  useEffect(() => {
    if (prevTtsSpeaking.current && !isTtsSpeaking) {
      setActiveSpeakingEventId(null);
    }
    prevTtsSpeaking.current = isTtsSpeaking;
  }, [isTtsSpeaking]);

  const handleSpeakFromHere = useCallback((sessionId: string, eventId: string) => {
    const session = useConversationStore.getState().sessions[sessionId];
    if (!session) return;
    const startIdx = session.events.findIndex((e) => e.id === eventId);
    if (startIdx === -1) return;
    const eventsFromHere = session.events.slice(startIdx);
    // Use speechParagraphs (normalized, no markdown) with raw text as fallback
    const texts = eventsFromHere.flatMap((e) => e.speechParagraphs?.length ? e.speechParagraphs : [e.text]);
    const ids = eventsFromHere.map((e) => e.id);
    setActiveSpeakingEventId(ids[0] ?? null);
    // Track the last event in the sequence for the persistent replay bar
    const lastId = ids[ids.length - 1] ?? null;
    if (lastId) { setLastTtsEventId(lastId); setLastTtsPaneId(sessionId); }
    void speakSequenceOnPane(sessionId, texts, (i) => {
      // Map flattened paragraph index back to event index for highlighting.
      // This is approximate — highlight the event whose paragraphs contain index i.
      let eventIdx = 0;
      let consumed = 0;
      for (let e = 0; e < eventsFromHere.length; e++) {
        const count = eventsFromHere[e]?.speechParagraphs?.length || 1;
        if (i < consumed + count) { eventIdx = e; break; }
        consumed += count;
      }
      const currentId = ids[eventIdx] ?? null;
      setActiveSpeakingEventId(currentId);
      if (currentId) { setLastTtsEventId(currentId); setLastTtsPaneId(sessionId); }
    });
  }, [speakSequenceOnPane]);

  const handleSpeakOne = useCallback((sessionId: string, eventId: string, text: string, paragraphs?: string[], opts?: { version?: "active" | "original" }) => {
    setActiveSpeakingEventId(eventId);
    setLastTtsEventId(eventId);
    setLastTtsPaneId(sessionId);
    speakTextOnPane(sessionId, text, paragraphs, { eventId, version: opts?.version });
  }, [speakTextOnPane]);

  // --- Mobile image upload ---
  const mobileFileInputRef = useRef<HTMLInputElement>(null);

  const handleMobileUploadImage = useCallback(() => {
    mobileFileInputRef.current?.click();
  }, []);

  const handleMobileFileChange = useCallback(async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file || !store.activePane) return;
    try {
      const path = await uploadFile(store.activePane, file);
      submitToActiveTerminal(path + "\n", "upload", store.activePane);
    } catch {
      // Upload errors are transient — user can retry
    }
  }, [store.activePane, submitToActiveTerminal]);

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
              ? store.columnFractions
              : store.rowFractions,
        };
      },
    [store.columnFractions, store.rowFractions],
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
        store.setColumnFractions(updated);
      } else {
        store.setRowFractions(updated);
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
  }, [store]);

  // Compute layout
  const orderedPanes = store.panes;
  const maxColumns = isMobile ? 1 : 2;
  const layout = resolveWorkspaceLayout(orderedPanes.length, maxColumns);
  const colFractions = reconcileTrackFractions(
    store.columnFractions,
    layout.columns,
  );
  const rowFractions = reconcileTrackFractions(
    store.rowFractions,
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
  // Additionally, in "tabs" display mode the grid is never rendered, so
  // fraction reconciliation is skipped entirely to avoid unnecessary store
  // writes.
  useEffect(() => {
    if (store.displayMode === "tabs") return;
    if (!fractionsMatch(colFractions, store.columnFractions)) {
      store.setColumnFractions(colFractions);
    }
  }, [colFractions, store]);

  useEffect(() => {
    if (store.displayMode === "tabs") return;
    if (!fractionsMatch(rowFractions, store.rowFractions)) {
      store.setRowFractions(rowFractions);
    }
  }, [rowFractions, store]);

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
      <div className="flex h-wc-app items-center justify-center bg-wc-surface-base text-wc-text-primary">
        <div className="text-center">
          <h1 className="text-2xl font-semibold mb-4">Web Console</h1>
          <p className="text-wc-text-muted mb-6">
            Browser terminal with PTY-backed sessions
          </p>
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
            <Plus className="mr-2 h-5 w-5" />
            {isCreating ? "Creating..." : "New Terminal"}
          </Button>
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

    const sessionConversation = conversationSessions[paneMeta.sessionId];
    const supportsMessagesView = paneMeta.supportsMessagesView;
    const viewMode = supportsMessagesView ? (conversationViewModes[paneMeta.sessionId] ?? "terminal") : "terminal";
    const unreadCount = supportsMessagesView && sessionConversation
      ? sessionConversation.events.filter((event) => event.role === "assistant" && event.sequence > sessionConversation.cursor.lastSeenSequence).length
      : 0;

    return (
      <div
        key={paneMeta.sessionId}
        data-testid="terminal-pane-container"
        data-session-id={paneMeta.sessionId}
        data-pane-index={idx}
        {...(isDropTarget ? { "data-drop-target": "" } : {})}
        className={cn(
          "relative flex flex-col rounded border overflow-hidden min-w-0 min-h-0 select-none",
          store.activePane === paneMeta.sessionId
            ? "border-wc-accent"
            : "border-wc-default",
          isBeingDragged && "opacity-40",
          isDropTarget && "ring-2 ring-blue-400/60 ring-inset",
        )}
        style={{ gridColumn, gridRow }}
        onClick={() => activatePane(paneMeta.sessionId)}
      >
        <TerminalHeader
          sessionId={paneMeta.sessionId}
          name={paneMeta.name}
          headerColor={paneMeta.headerColor}
          isActive={store.activePane === paneMeta.sessionId}
          viewMode={viewMode}
          unreadCount={unreadCount}
          onClose={() => handleRequestClose(paneMeta.sessionId)}
          onFocus={() => activatePane(paneMeta.sessionId)}
          onToggleView={supportsMessagesView ? () => setConversationViewMode(paneMeta.sessionId, viewMode === "terminal" ? "messages" : "terminal") : undefined}
          onDragStart={startArrangeDrag}
        />
        {/* overflow-hidden: same duplicate-scrollbar prevention as in tabs mode.
         * See the tabs-mode comment for the full explanation. */}
        <div className="relative flex-1 min-h-0 overflow-hidden">
          <ErrorBoundary region="terminal">
            <TerminalPane
              sessionId={paneMeta.sessionId}
              onExit={handleExit}
              onReady={() => handleTerminalReady(paneMeta.sessionId)}
              onVoiceStart={voiceInput.supported ? voiceInput.startRecording : undefined}
              onVoiceStop={voiceInput.supported ? voiceInput.stopRecording : undefined}
              onTtsSpeakingChange={(speaking) => handleTtsSpeakingChange(paneMeta.sessionId, speaking)}
              onSpeakingEventChange={(eventId) => {
                if (paneMeta.sessionId === store.activePane) {
                  setActiveSpeakingEventId(eventId);
                  if (eventId) { setLastTtsEventId(eventId); setLastTtsPaneId(paneMeta.sessionId); }
                }
              }}
              ref={(handle) =>
                registerTerminalRef(paneMeta.sessionId, handle)
              }
            />
          </ErrorBoundary>
          {supportsMessagesView && viewMode === "messages" && (
            <div className="absolute inset-0">
              <MessagesPane
                        sessionId={paneMeta.sessionId}
                        onSpeakFromHere={(eventId) => handleSpeakFromHere(paneMeta.sessionId, eventId)}
                        onSpeakOne={(eventId, text, paragraphs, opts) => handleSpeakOne(paneMeta.sessionId, eventId, text, paragraphs, opts)}
                        activeSpeakingEventId={store.activePane === paneMeta.sessionId ? activeSpeakingEventId : null}
                        isTtsSpeaking={isTtsSpeaking && store.activePane === paneMeta.sessionId}
                      />
            </div>
          )}
        </div>
      </div>
    );
  });

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
        hidden={isMobile && store.displayMode === "tabs"}
        onOpenSettings={() => store.setSettingsModalOpen(true)}
        onOpenAi={() => store.setAiModalOpen(true)}
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
        <div className="px-3 py-1.5 text-xs text-amber-300 bg-amber-500/10 border-b border-amber-500/30">
          {voiceInput.fallbackNotice}
        </div>
      )}
      {voiceInput.speakerNotice && (
        <div className="px-3 py-1.5 text-xs text-sky-200 bg-sky-500/10 border-b border-sky-500/30">
          {voiceInput.speakerNotice}
        </div>
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

      {/* Tab bar (only in tabs mode) */}
      {store.displayMode === "tabs" && (
        <TabBar
          panes={orderedPanes}
          activePane={store.activePane}
          onNewTerminal={() => handleLaunch()}
          onOpenLauncher={openLauncher}
          onClosePane={handleRequestClose}
          isCreating={isCreating}
          trailingActions={isMobile ? (
            <Button
              data-testid="tabbar-settings"
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0 mx-1 self-center"
              onClick={() => store.setSettingsModalOpen(true)}
              title="Settings"
            >
              <Settings className="h-4 w-4" />
            </Button>
          ) : undefined}
        />
      )}

      {/* Main content area */}
      {store.displayMode === "tabs" ? (
        /* Tab mode: stacked panes with display:none for inactive */
        <div className="relative flex-1 min-h-0 overflow-hidden">
          {/* Toggle between terminal and messages view.
           * Shows the icon for the view you'll switch TO (not the current view):
           *   • In terminal mode → show chat icon (click to switch to messages)
           *   • In messages mode → show terminal icon (click to switch back)
           * Circular icon button with a translucent background so it doesn't
           * obscure too much terminal content but is still easy to tap. */}
          {store.activePane && store.panes.find((pane) => pane.sessionId === store.activePane)?.supportsMessagesView && (
            <div className="absolute right-3 top-3 z-20">
              <button
                className="flex items-center justify-center h-8 w-8 rounded-full bg-wc-surface-raised/80 border border-wc-default text-wc-text-secondary hover:text-wc-text-primary hover:bg-wc-surface-input transition-colors backdrop-blur-sm"
                onClick={() => {
                  const current = conversationViewModes[store.activePane ?? ""] ?? "terminal";
                  setConversationViewMode(store.activePane ?? "", current === "terminal" ? "messages" : "terminal");
                }}
                title={(conversationViewModes[store.activePane] ?? "terminal") === "terminal" ? "Switch to messages view" : "Switch to terminal view"}
              >
                {(conversationViewModes[store.activePane] ?? "terminal") === "terminal"
                  ? <MessageSquareText className="h-3.5 w-3.5" />
                  : <TerminalSquare className="h-3.5 w-3.5" />}
              </button>
            </div>
          )}
          {orderedPanes.map((paneMeta) => {
            const isActive = paneMeta.sessionId === store.activePane;
            const supportsMessagesView = paneMeta.supportsMessagesView;
            const viewMode = supportsMessagesView ? (conversationViewModes[paneMeta.sessionId] ?? "terminal") : "terminal";
            return (
              <div
                key={paneMeta.sessionId}
                data-testid={`tab-pane-${paneMeta.sessionId}`}
                className="absolute inset-0 flex flex-col select-none"
                style={{ visibility: isActive ? "visible" : "hidden" }}
              >
                {/* overflow-hidden prevents a duplicate scrollbar from appearing.
                 * xterm.js has its own internal scrollable viewport (.xterm-viewport).
                 * Without clipping here, the browser adds a native scrollbar to this
                 * wrapper once the terminal buffer grows large enough, which on mobile
                 * captures all touch-scroll events and makes the real terminal
                 * un-scrollable unless the user carefully avoids the outer scrollbar. */}
                <div className="relative flex-1 min-h-0 overflow-hidden">
                  <ErrorBoundary region="terminal">
                    <TerminalPane
                      sessionId={paneMeta.sessionId}
                      onExit={handleExit}
                      onReady={() => handleTerminalReady(paneMeta.sessionId)}
                      onVoiceStart={voiceInput.supported ? voiceInput.startRecording : undefined}
                      onVoiceStop={voiceInput.supported ? voiceInput.stopRecording : undefined}
                      onTtsSpeakingChange={(speaking) => handleTtsSpeakingChange(paneMeta.sessionId, speaking)}
                      onSpeakingEventChange={(eventId) => {
                        if (paneMeta.sessionId === store.activePane) {
                          setActiveSpeakingEventId(eventId);
                          if (eventId) { setLastTtsEventId(eventId); setLastTtsPaneId(paneMeta.sessionId); }
                        }
                      }}
                      ref={(handle) =>
                        registerTerminalRef(paneMeta.sessionId, handle)
                      }
                    />
                  </ErrorBoundary>
                  {supportsMessagesView && viewMode === "messages" && (
                    <div className="absolute inset-0">
                      <MessagesPane
                        sessionId={paneMeta.sessionId}
                        onSpeakFromHere={(eventId) => handleSpeakFromHere(paneMeta.sessionId, eventId)}
                        onSpeakOne={(eventId, text, paragraphs, opts) => handleSpeakOne(paneMeta.sessionId, eventId, text, paragraphs, opts)}
                        activeSpeakingEventId={store.activePane === paneMeta.sessionId ? activeSpeakingEventId : null}
                        isTtsSpeaking={isTtsSpeaking && store.activePane === paneMeta.sessionId}
                      />
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        /* Grid mode: original grid layout with minimap */
        <div className="relative flex-1 min-h-0 overflow-hidden">
          <div
            ref={scrollContainerRef}
            className={cn("absolute inset-0 overflow-auto wc-hide-scrollbar", store.isMinimapVisible && "right-[34px]")}
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
        {/* TTS player bar — visible when audio is playing OR when auto-TTS
         * is enabled and there is a previous response available to replay.
         *
         * When actively speaking, the bar shows live playback state (polled
         * at 100 ms).  When idle with a replayable event, it shows a
         * "stopped" state where the play button triggers a replay.
         *
         * Uses isTtsSpeaking as the primary gate so the bar always appears
         * the instant audio starts.  ttsPlayback is populated by polling;
         * if the first poll hasn't fired yet we fall back to
         * FALLBACK_TTS_PLAYBACK (see comment above the polling effect). */}
        {(() => {
          const isReplayMode = !isTtsSpeaking
            && store.autoTtsEnabled
            && lastTtsEventId != null
            && store.activePane === lastTtsPaneId;
          if (!isTtsSpeaking && !isReplayMode) return null;

          const pb = isTtsSpeaking
            ? (ttsPlayback ?? FALLBACK_TTS_PLAYBACK)
            : { ...FALLBACK_TTS_PLAYBACK, isPaused: true };

          // Resolve the event to show context for — prefer the actively
          // speaking event, fall back to the last-played event for replay.
          const displayEventId = activeSpeakingEventId ?? lastTtsEventId;
          const activeEvent = displayEventId && store.activePane
            ? useConversationStore.getState().sessions[store.activePane]?.events.find((e) => e.id === displayEventId)
            : undefined;
          const hasOriginal = (activeEvent?.summarized ?? false) &&
            (activeEvent?.originalSpeechParagraphs?.length ?? 0) > 0;
          const canRequestSummarize = !!(activeEvent && !activeEvent.summarized && activeEvent.role === "assistant");
          return (
            <AudioPlayerBar
              isPaused={pb.isPaused}
              currentTime={pb.currentTime}
              duration={pb.duration}
              playbackRate={pb.playbackRate}
              volume={pb.volume}
              capabilities={pb.capabilities}
              isSummarized={activeEvent?.summarized ?? false}
              hasOriginalVersion={hasOriginal}
              canSummarize={canRequestSummarize}
              isSummarizing={isSummarizing}
              onPause={handleTtsPause}
              onResume={isReplayMode ? () => {
                // Replay the last TTS event
                if (activeEvent && store.activePane) {
                  const paragraphs = activeEvent.speechParagraphs?.length
                    ? activeEvent.speechParagraphs
                    : [activeEvent.text];
                  speakTextOnPane(store.activePane, activeEvent.text, paragraphs, { eventId: activeEvent.id });
                }
              } : handleTtsResume}
              onSeek={handleTtsSeek}
              onSetPlaybackRate={handleTtsSetPlaybackRate}
              onSetVolume={handleTtsSetVolume}
              onStop={isTtsSpeaking ? handleTtsStop : () => {
                // In replay mode, stop dismisses the bar
                setLastTtsEventId(null);
                setLastTtsPaneId(null);
              }}
              onToggleSummarized={hasOriginal && activeEvent && store.activePane ? (useSummarized) => {
                const activePaneId = store.activePane;
                if (!activePaneId) return;
                const paragraphs = useSummarized
                  ? activeEvent.speechParagraphs
                  : (activeEvent.originalSpeechParagraphs ?? activeEvent.speechParagraphs);
                speakTextOnPane(activePaneId, activeEvent.text, paragraphs, { eventId: activeEvent.id, version: useSummarized ? "active" : "original" });
              } : undefined}
              onRequestSummarize={canRequestSummarize && activeEvent && store.activePane ? () => {
                const sid = store.activePane;
                if (!sid) return;
                const eid = activeEvent.id;
                setIsSummarizing(true);
                void summarizeEvent(sid, eid).then((res) => {
                  if (res.summarized && res.speechParagraphs) {
                    // Update the conversation store with the summary
                    const convState = useConversationStore.getState();
                    const session = convState.sessions[sid];
                    if (session) {
                      const updatedEvents = session.events.map((ev) =>
                        ev.id === eid
                          ? { ...ev, summarized: true, originalSpeechParagraphs: ev.speechParagraphs, speechParagraphs: res.speechParagraphs ?? ev.speechParagraphs }
                          : ev,
                      );
                      useConversationStore.setState({
                        sessions: { ...convState.sessions, [sid]: { ...session, events: updatedEvents } },
                      });
                      // Replay with summarized version
                      speakTextOnPane(sid, activeEvent.text, res.speechParagraphs, { eventId: eid, version: "active" });
                    }
                  }
                }).finally(() => setIsSummarizing(false));
              } : undefined}
            />
          );
        })()}
        {/* AI suggestion bar (mobile only) */}
        {store.aiSuggestActive && (
          <AiSuggestBar
            inputText={mobileInputText}
            onExecute={(cmd) => {
              handleSendToTerminal(cmd, "toolbar-submit");
              mobileToolbarRef.current?.clearInput();
              store.setAiSuggestActive(false);
            }}
            onClose={() => store.setAiSuggestActive(false)}
          />
        )}
        {/* Mobile toolbar */}
        <MobileToolbar
          ref={mobileToolbarRef}
          onInput={handleSendToTerminal}
          subscribeInputSettled={handleSubscribeInputSettled}
          subscribePendingInput={handleSubscribePendingInput}
          getPendingInputSnapshot={handleGetPendingInputSnapshot}
          onFocusTerminal={handleFocusTerminal}
          activeSessionId={store.activePane}
          voiceSupported={voiceInput.supported}
          voicePreparing={voiceInput.isPreparing}
          voiceRecording={voiceInput.isRecording}
          voiceListening={voiceInput.isListening}
          voiceTranscribing={voiceInput.isTranscribing}
          voiceError={voiceInput.error}
          voiceLevel={voiceInput.audioLevel}
          voicePartialTranscript={voiceInput.partialTranscript}
          voiceBackend={voiceInput.backend}
          onVoiceStart={handleVoiceStart}
          onVoiceStop={handleVoiceStop}
          onVoiceCancel={handleVoiceCancel}
          voiceCommandSuggestion={voiceInput.commandSuggestion}
          onVoiceCommandConfirm={handleVoiceCommandConfirm}
          onVoiceCommandDismiss={handleVoiceCommandDismiss}
          onUploadImage={handleMobileUploadImage}
          onOpenAi={() => store.setAiSuggestActive(!store.aiSuggestActive)}
          onInputChange={setMobileInputText}
          aiSuggestActive={store.aiSuggestActive}
          isTtsSpeaking={isTtsSpeaking}
          onTtsStop={handleTtsStop}
          viewMode={store.activePane ? (conversationViewModes[store.activePane] ?? "terminal") : "terminal"}
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
        sessionName={store.panes.find((p) => p.sessionId === pendingClose)?.name ?? "terminal"}
        onConfirm={handleConfirmClose}
        onCancel={handleCancelClose}
      />
    </div>
  );
}
