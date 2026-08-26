// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
// DOC: docs/internal/SEAMS.md#1-entry-presentation
import { useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState, forwardRef } from "react";
import "@xterm/xterm/css/xterm.css";
import { useTranslation } from "react-i18next";
import { useXtermLifecycle } from "../hooks/terminal/useXtermLifecycle";
import { useFollowerPresentation } from "../hooks/terminal/useFollowerPresentation";
import { usePaneAttachments } from "../hooks/terminal/usePaneAttachments";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { strings } from "../consts/strings";
import { useTerminalSession } from "../hooks/terminal/useTerminalSession";
import { useTerminalBackgroundDetector } from "../hooks/terminal/useTerminalBackgroundDetector";
import { chromeTheme } from "../lib/chromeTheme";
import { isTabLikeDisplayMode } from "../lib/workspaceDisplayMode";
import type { GateResult, InputIntent } from "./terminal/inputGate";
import type { InputSettlementCallback, InputSettledListener } from "../hooks/terminal/useStdinStream";
import { useEffectiveFontSize, useWorkspaceStore, type TerminalPaneStatus } from "../stores/useWorkspaceStore";
import { TERMINAL_THEMES, DEFAULT_THEME_ID } from "../consts/config";
import { useTerminalVoiceShortcut } from "../hooks/useKeyboardListeners";
import { PaneSelectionLayer } from "./PaneSelectionLayer";
import type { ConversationEvent } from "../api/conversation";
import { usePaneSpeech, type PaneSpeechPlaybackHandle } from "../hooks/terminal/usePaneSpeech";
import { usePaneSelection } from "../hooks/terminal/usePaneSelection";
import { DeviceFrame } from "./terminal/DeviceFrame";

interface TerminalPaneProps {
  sessionId: string;
  onExit?: (sessionId: string) => void;
  onVoiceStart?: () => void;
  onVoiceStop?: () => void;
  /** Called when TTS speaking state changes for this pane. */
  onTtsSpeakingChange?: (speaking: boolean) => void;
  /** Called when the currently-speaking conversation event changes (for summarize controls). */
  onSpeakingEventChange?: (eventId: string | null) => void;
  onConversationEventReceived?: (
    sessionId: string,
    event: ConversationEvent,
    sendAck: (stage: string, message?: string, backend?: string) => void,
  ) => void;
  /**
   * Called when auto-TTS playback is rejected by the browser's autoplay
   * policy and no user gesture has unlocked the audio element yet. Pass
   * `null` to clear (when the condition resolves or the user dismisses).
   * Consumers should render a persistent "Enable voice" affordance and
   * call `enable()` on click — a successful return replays pending events.
   */
  onNeedsUnlock?: (payload: { sessionId: string; enable: () => Promise<boolean> } | null) => void;
}

// [REQ:P0-007b] Terminal Key/Chord Mapping - expose grouped pane operations.
export interface TerminalPaneHandle {
  input: {
    submit: (data: string, intent: Exclude<InputIntent, "control">) => GateResult;
    subscribeSettled: (cb: InputSettledListener) => () => void;
    awaitOffset: (offset: number, cb: InputSettlementCallback) => () => void;
  };
  control: {
    send: (data: string) => boolean;
    scroll: (lines: number) => void;
    focus: () => void;
  };
  selection: {
    copy: () => Promise<boolean>;
    paste: () => Promise<boolean>;
  };
  pendingInput: {
    subscribe: (cb: () => void) => () => void;
    snapshot: () => readonly { data: string; addedAt: number }[];
  };
  playback: PaneSpeechPlaybackHandle;
}

// [REQ:P0-002d] xterm.js Terminal Rendering
const TerminalPane = forwardRef<TerminalPaneHandle, TerminalPaneProps>(
  function TerminalPane({ sessionId, onExit, onVoiceStart, onVoiceStop, onTtsSpeakingChange, onSpeakingEventChange, onNeedsUnlock, onConversationEventReceived }, ref) {
    const { t } = useTranslation();
    const paneStatus = useWorkspaceStore((state) => state.paneStatuses?.[sessionId] ?? null);
    const updatePaneStatus = useWorkspaceStore((state) => state.setPaneStatus);
    const setPaneStatus = useCallback(
      (status: TerminalPaneStatus | null) => {
        updatePaneStatus?.(sessionId, status);
      },
      [sessionId, updatePaneStatus],
    );
    const [pinchPreviewFontSize, setPinchPreviewFontSize] = useState<number | null>(null);
    const commitPinchFontSize = useCallback((size: number) => {
      useWorkspaceStore.getState().setDeviceFontSize(sessionId, size);
    }, [sessionId]);

    // Per-pane selectors with fallbacks for old persisted data
	// A follower renders the leader's grid. Its local font stepper is disabled
	// so it cannot fight the authoritative leader dimensions.
    const paneFontSize = useEffectiveFontSize(sessionId);
    const wheelScrollSensitivity = useWorkspaceStore((s) => s.wheelScrollSensitivity);
    const paneThemeId = useWorkspaceStore(
      useCallback((s) => s.panes.find((p) => p.sessionId === sessionId)?.themeId ?? DEFAULT_THEME_ID, [sessionId]),
    );
    const paneTheme = useMemo(() => {
      const fallback = { background: "#0f172a", foreground: "#e2e8f0", cursor: "#38bdf8" } as const;
      return TERMINAL_THEMES[paneThemeId]?.colors ?? TERMINAL_THEMES[DEFAULT_THEME_ID]?.colors ?? fallback;
    }, [paneThemeId]);
    const activePane = useWorkspaceStore((state) => state.activePane);
    const displayMode = useWorkspaceStore((state) => state.displayMode);
    const adaptiveChrome = useWorkspaceStore((state) => state.adaptiveChrome);

    const sendResizeRef = useRef<(cols: number, rows: number) => void>(() => {});
    const serverSizeRef = useRef<{ cols: number; rows: number } | null>(null);
    const isFollowerRef = useRef(false);
    const isFollowerForLifecycle = useCallback(() => isFollowerRef.current, []);
    const sendResizeToSession = useCallback((cols: number, rows: number) => { sendResizeRef.current(cols, rows); }, []);
    const getServerSizeFromSession = useCallback(() => serverSizeRef.current, []);
    const renamePaneById = useWorkspaceStore((state) => state.renamePaneById);
    const { syncPaneUpdate } = useWorkspaceSync();
    const { containerRef, fitRef, terminal, paneSize } = useXtermLifecycle({
      sessionId,
      paneFontSize,
      paneTheme,
      wheelScrollSensitivity,
      sendResize: sendResizeToSession,
      getServerSize: getServerSizeFromSession,
      isFollower: isFollowerForLifecycle,
      renamePaneById,
      syncPaneUpdate,
    });

    // Delegate all WebSocket protocol handling to the session hook
  const { submitInput, sendControl, setMouseMode, mouseMode, scrollBy, sendResize, serverSize, isFollower, leaderDevice, takeLease, subscribeInputSettled, awaitOffset, subscribePendingInput, getPendingInputSnapshot, sendConversationAck } = useTerminalSession({
      sessionId,
      terminal,
      predictionContainer: containerRef.current,
      onStatus: setPaneStatus,
      onExit,
      onReady: () => {
        // After socket connects and snapshot replay completes, re-fit to
        // ensure dimensions match this client's actual container size.
        requestAnimationFrame(() => {
          const fit = fitRef.current;
          if (!fit || !terminal) return;
          fit.fit();
		  // useTerminalSession is the sole connect-time size declaration.
        });
      },
    });

    const handleMouseModeToggle = useCallback((enabled: boolean) => {
      if (!setMouseMode(enabled)) {
        setPaneStatus({ kind: "error", detail: "This pane does not support tmux mouse mode" });
      }
    }, [setMouseMode]);

    useEffect(() => {
      setPaneStatus(null);
      return () => { setPaneStatus(null); };
    }, [sessionId, setPaneStatus]);

    useEffect(() => {
      if (!paneStatus || paneStatus.kind === "session-ended") return;
      const timer = window.setTimeout(() => { setPaneStatus(null); }, 6000);
      return () => { window.clearTimeout(timer); };
    }, [paneStatus, setPaneStatus]);
    sendResizeRef.current = sendResize;
    serverSizeRef.current = serverSize;
    isFollowerRef.current = isFollower;
    const followerFrame = useFollowerPresentation({ terminal, fitRef, serverSize, isFollower, paneSize });

    const { supported: speechSupported, playback } = usePaneSpeech({
      sessionId,
      onSpeakingEventChange,
      onTtsSpeakingChange,
      onNeedsUnlock,
      onConversationEventReceived,
      sendConversationAck,
    });

    // Pending-input draft round-trip. Offscreen terminals are unmounted to keep
    // cost flat in N; without this, anything typed-but-not-yet-sent would be
    // lost when the pane unmounts. On unmount we stash the unsent input; on the
    // next mount we re-inject it (the gate queues it until the WS is ready).
    const setPendingInputBuffer = useWorkspaceStore((s) => s.setPendingInputBuffer);
    const consumePendingInputBuffer = useWorkspaceStore((s) => s.consumePendingInputBuffer);
    const getPendingInputSnapshotRef = useRef(getPendingInputSnapshot);
    getPendingInputSnapshotRef.current = getPendingInputSnapshot;
    const submitInputRef = useRef(submitInput);
    submitInputRef.current = submitInput;
    useEffect(() => {
      const entries = consumePendingInputBuffer(sessionId);
      for (const entry of entries ?? []) submitInputRef.current(entry.data, entry.intent);
      return () => {
        const entries = getPendingInputSnapshotRef.current().map((entry) => ({ data: entry.data, intent: entry.intent }));
        setPendingInputBuffer(sessionId, entries);
      };
    }, [sessionId, consumePendingInputBuffer, setPendingInputBuffer]);

    const selection = usePaneSelection({
      terminal,
      containerRef,
      sendControl,
      scrollBy,
      fontSize: paneFontSize,
      isFollower,
      onFontSizeCommit: commitPinchFontSize,
      onFontSizePreview: setPinchPreviewFontSize,
      submitInput,
      awaitOffset,
      subscribeInputSettled,
      rejectedMessage: t(strings.terminalPane.inputRejected),
    });

    // Expose grouped pane operations for the toolbar, launcher and workspace.
    useImperativeHandle(ref, () => ({
      input: { submit: submitInput, subscribeSettled: subscribeInputSettled, awaitOffset },
      control: {
        send: sendControl,
        scroll: (lines: number) => { scrollBy(lines, "programmatic"); },
        focus: () => { terminal?.focus(); },
      },
      selection: {
        copy: selection.copySelection,
        paste: selection.pasteFromClipboard,
      },
      pendingInput: { subscribe: subscribePendingInput, snapshot: getPendingInputSnapshot },
      playback,
    }), [awaitOffset, getPendingInputSnapshot, playback, selection.copySelection, selection.pasteFromClipboard, sendControl, scrollBy, subscribeInputSettled, subscribePendingInput, submitInput, terminal]);

    const closeContextMenu = selection.closeContextMenu;
    const handleCtxSpeak = useCallback(() => {
      const selectedText = terminal?.getSelection();
      if (selectedText) void playback.speak(selectedText, [selectedText]);
      closeContextMenu();
    }, [closeContextMenu, playback, terminal]);
    const attachments = usePaneAttachments(sessionId, submitInput, closeContextMenu);
    const { fileInputRef, dragOver, uploading, uploadError, handleCtxUploadImage, handleFileInputChange, handlePaste, handleDragOver, handleDragLeave, handleDrop } = attachments;

    // Adaptive app-chrome: detect the rendered terminal background for the
    // focused pane in single-focus modes and feed it to the imperative chrome
    // applier. Disabled in grid mode, when this pane isn't focused, or when the
    // adaptive setting is off — so only one detector runs at a time and the
    // heavy terminal subtree never re-renders on color changes.
    const chromeDetectorEnabled =
      adaptiveChrome && isTabLikeDisplayMode(displayMode) && activePane === sessionId;
    useTerminalBackgroundDetector(terminal, {
      enabled: chromeDetectorEnabled,
      defaultBackground: paneTheme.background,
      onColor: useCallback((hex: string | null) => { chromeTheme.setDetected(sessionId, hex); }, [sessionId]),
    });

    // Intercept configurable voice shortcut before xterm processes it.
    const voiceShortcut = useWorkspaceStore((s) => s.voiceShortcut);
    useTerminalVoiceShortcut(containerRef, voiceShortcut, onVoiceStart, onVoiceStop);

	    return (
      <div
        ref={containerRef}
        data-testid="terminal-pane"
        data-session-id={sessionId}
        // overflow-hidden is critical: xterm.js manages its own scrolling via
        // an internal .xterm-viewport element (overflow-y: scroll). Without
        // clipping overflow here, the browser creates a SECOND native scrollbar
        // on this container (or an ancestor) once the terminal buffer grows
        // large enough for xterm's rendered DOM to exceed the container bounds.
        // That phantom outer scrollbar captures touch/wheel events on mobile,
        // making the terminal unscrollable unless the user carefully avoids it.
        className={`h-full w-full overflow-hidden relative p-1${dragOver ? " ring-2 ring-inset ring-blue-400/60" : ""}`}
        onPasteCapture={handlePaste}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        <PaneSelectionLayer
          contextMenu={selection.contextMenu}
          hasSelection={selection.hasSelection}
          inputError={selection.inputError}
          paneStatus={paneStatus}
          uploadError={uploadError}
          uploading={uploading}
          uploadingLabel={t(strings.terminalPane.uploadingImage)}
          ttsSupported={speechSupported}
          onCopy={selection.handleCopy}
          onPaste={selection.handlePaste}
          onSelectAll={selection.selectAll}
          onClear={selection.clear}
          onUploadImage={handleCtxUploadImage}
          onSpeak={handleCtxSpeak}
          mouseMode={mouseMode ?? undefined}
          onToggleMouseMode={mouseMode === null ? undefined : handleMouseModeToggle}
          onClose={selection.closeContextMenu}
        >
          {pinchPreviewFontSize !== null && <div data-testid="pinch-font-preview" role="status" className="absolute top-2 right-2 z-wc-chrome-raised rounded bg-slate-900/90 px-2 py-1 text-xs text-slate-100 shadow-lg">{pinchPreviewFontSize}px</div>}
          {followerFrame && <DeviceFrame archetype={followerFrame.archetype} chromeTier={followerFrame.tier} rect={followerFrame.rect} leaderDevice={leaderDevice} gridCols={followerFrame.cols} gridRows={followerFrame.rows} onTakeOver={takeLease} />}
        </PaneSelectionLayer>
        <input ref={fileInputRef} type="file" accept="image/*" hidden onChange={handleFileInputChange} />
      </div>
    );
  },
);

export default TerminalPane;
