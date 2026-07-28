import { memo, useCallback, useEffect, useState, type CSSProperties, type PointerEvent as ReactPointerEvent, type ReactNode } from "react";
import { useConversationStore, type PaneViewMode } from "../stores/useConversationStore";
import type { PaneMetadata } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import type { TTSPlaybackState } from "../audio-integration";
import type { PlaybackFocusRequest, PlaybackVersion } from "../domains/tts-playback/types";
import type { ConversationEvent } from "../api/conversation";
import type { TerminalPaneHandle } from "./TerminalPane";
import ErrorBoundary from "./ErrorBoundary";
import TerminalPane from "./TerminalPane";
import TerminalHeader from "./TerminalHeader";
import MessagesPane from "./MessagesPane";
import type { SummarizationLevel } from "./tts/PlaybackModeControl";

interface WorkspacePaneShellProps {
  paneMeta: PaneMetadata;
  layoutMode: "grid" | "tabs";
  gridColumn?: string;
  gridRow?: string;
  paneIndex?: number;
  isActive: boolean;
  isBeingDragged?: boolean;
  isDropTarget?: boolean;
  isVisible?: boolean;
  isTtsSpeaking: boolean;
  activeSpeakingEventId: string | null;
  loadingEventId: string | null;
  summarizeLevel: SummarizationLevel;
  summarizingEventId: string | null;
  getSummarizeError: (eventId: string) => string | null;
  onClearSummarizeError: (eventId: string) => void;
  onToggleSummarized: (sessionId: string, eventId: string, useSummarized: boolean) => void;
  onChangeLevel: (sessionId: string, eventId: string, level: SummarizationLevel) => void;
  selectedVersionForEvent: (sessionId: string, event: ConversationEvent) => PlaybackVersion;
  playbackState: TTSPlaybackState;
  onSetPlaybackRate: (rate: number) => void;
  onSetVolume: (level: number) => void;
  onSetMuted: (next: boolean) => void;
  playbackFocusRequest: PlaybackFocusRequest | null;
  onActivate: (sessionId: string) => void;
  onRequestClose: (sessionId: string) => void;
  onToggleView: (sessionId: string, viewMode: PaneViewMode) => void;
  /** Notifies when this pane is mid view-switch so a shared toolbar button
   *  (e.g. the tabs-mode floating toggle) can show a loading indicator. */
  onViewSwitchPendingChange?: (sessionId: string, pending: boolean) => void;
  messagesToolbarTrailingAction?: ReactNode;
  onStartArrangeDrag?: (paneId: string, e: ReactPointerEvent) => void;
  onTerminalExit: (sessionId: string) => void;
  onTerminalRef: (sessionId: string, handle: TerminalPaneHandle | null) => void;
  onVoiceStart?: () => void;
  onVoiceStop?: () => void;
  onTtsSpeakingChange: (sessionId: string, speaking: boolean) => void;
  onSpeakingEventChange: (sessionId: string, eventId: string | null) => void;
  onConversationEventReceived: (
    sessionId: string,
    event: ConversationEvent,
    sendAck: (stage: string, message?: string, backend?: string) => void,
  ) => void;
  onNeedsUnlock: (payload: { sessionId: string; enable: () => Promise<boolean> } | null) => void;
  onPlayFromHere: (sessionId: string, eventId: string) => void;
  onPlayEvent: (sessionId: string, eventId: string) => void;
}

function WorkspacePaneShell({
  paneMeta,
  layoutMode,
  gridColumn,
  gridRow,
  paneIndex,
  isActive,
  isBeingDragged = false,
  isDropTarget = false,
  isVisible = true,
  isTtsSpeaking,
  activeSpeakingEventId,
  loadingEventId,
  summarizeLevel,
  summarizingEventId,
  getSummarizeError,
  onClearSummarizeError,
  onToggleSummarized,
  onChangeLevel,
  selectedVersionForEvent,
  playbackState,
  onSetPlaybackRate,
  onSetVolume,
  onSetMuted,
  playbackFocusRequest,
  onActivate,
  onRequestClose,
  onToggleView,
  onViewSwitchPendingChange,
  messagesToolbarTrailingAction,
  onStartArrangeDrag,
  onTerminalExit,
  onTerminalRef,
  onVoiceStart,
  onVoiceStop,
  onTtsSpeakingChange,
  onSpeakingEventChange,
  onConversationEventReceived,
  onNeedsUnlock,
  onPlayFromHere,
  onPlayEvent,
}: WorkspacePaneShellProps) {
  const { sessionId, name, headerColor, supportsMessagesView } = paneMeta;

  const viewMode = useConversationStore(
    useCallback(
      (state) => (
        supportsMessagesView ? (state.viewModes[sessionId] ?? "terminal") : "terminal"
      ),
      [sessionId, supportsMessagesView],
    ),
  );
  const unreadCount = useConversationStore(
    useCallback((state) => {
      if (!supportsMessagesView) return 0;
      const session = state.sessions[sessionId];
      if (!session) return 0;
      return session.events.filter(
        (event) => event.role === "assistant" && event.sequence > session.cursor.lastSeenSequence,
      ).length;
    }, [sessionId, supportsMessagesView]),
  );

  useEffect(() => {
    onViewSwitchPendingChange?.(sessionId, false);
    return () => onViewSwitchPendingChange?.(sessionId, false);
  }, [sessionId, onViewSwitchPendingChange]);

  const wrapperStyle: CSSProperties | undefined = layoutMode === "grid"
    ? { gridColumn, gridRow }
    : { visibility: isVisible ? "visible" : "hidden" };

  const handleToggleView = useCallback(() => {
    onToggleView(sessionId, viewMode);
  }, [onToggleView, sessionId, viewMode]);

  const handlePlayFromHere = useCallback((eventId: string) => {
    onPlayFromHere(sessionId, eventId);
  }, [onPlayFromHere, sessionId]);

  const handlePlayEvent = useCallback((eventId: string) => {
    onPlayEvent(sessionId, eventId);
  }, [onPlayEvent, sessionId]);

  const handleToggleSummarized = useCallback((eventId: string, useSummarized: boolean) => {
    onToggleSummarized(sessionId, eventId, useSummarized);
  }, [onToggleSummarized, sessionId]);

  const handleChangeLevel = useCallback((eventId: string, level: SummarizationLevel) => {
    onChangeLevel(sessionId, eventId, level);
  }, [onChangeLevel, sessionId]);

  const resolveSelectedVersion = useCallback((event: ConversationEvent) => {
    return selectedVersionForEvent(sessionId, event);
  }, [selectedVersionForEvent, sessionId]);

  return (
    <div
      key={sessionId}
      data-testid={layoutMode === "grid" ? "terminal-pane-container" : `tab-pane-${sessionId}`}
      data-session-id={layoutMode === "grid" ? sessionId : undefined}
      data-pane-index={layoutMode === "grid" && paneIndex != null ? paneIndex : undefined}
      {...(layoutMode === "grid" && isDropTarget ? { "data-drop-target": "" } : {})}
      className={cn(
        layoutMode === "grid"
          ? "relative flex flex-col rounded border overflow-hidden min-w-0 min-h-0 select-none"
          : "absolute inset-0 flex flex-col select-none",
        layoutMode === "grid" && (
          isActive ? "border-wc-accent" : "border-wc-default"
        ),
        layoutMode === "grid" && isBeingDragged && "opacity-40",
        layoutMode === "grid" && isDropTarget && "ring-2 ring-blue-400/60 ring-inset",
      )}
      style={wrapperStyle}
      onClick={layoutMode === "grid" ? () => onActivate(sessionId) : undefined}
    >
      {layoutMode === "grid" && (
        <TerminalHeader
          sessionId={sessionId}
          name={name}
          headerColor={headerColor}
          isActive={isActive}
          viewMode={viewMode}
          unreadCount={unreadCount}
          onClose={() => onRequestClose(sessionId)}
          onFocus={() => onActivate(sessionId)}
          onToggleView={supportsMessagesView ? handleToggleView : undefined}
          isViewSwitchPending={false}
          onDragStart={onStartArrangeDrag}
        />
      )}
      <div className="relative flex-1 min-h-0 overflow-hidden">
        <ErrorBoundary region="terminal">
          <TerminalPane
            sessionId={sessionId}
            onExit={onTerminalExit}
            onVoiceStart={onVoiceStart}
            onVoiceStop={onVoiceStop}
            onTtsSpeakingChange={(speaking) => onTtsSpeakingChange(sessionId, speaking)}
            onSpeakingEventChange={(eventId) => onSpeakingEventChange(sessionId, eventId)}
            onConversationEventReceived={onConversationEventReceived}
            onNeedsUnlock={onNeedsUnlock}
            ref={(handle) => onTerminalRef(sessionId, handle)}
          />
        </ErrorBoundary>
        {supportsMessagesView && isVisible && viewMode === "messages" && (
          <div className="absolute inset-0">
            <MessagesPane
              sessionId={sessionId}
              onPlayFromHere={handlePlayFromHere}
              onPlayEvent={handlePlayEvent}
              activeSpeakingEventId={activeSpeakingEventId}
              loadingEventId={loadingEventId}
              isTtsSpeaking={isTtsSpeaking}
              summarizeLevel={summarizeLevel}
              selectedVersionForEvent={resolveSelectedVersion}
              summarizingEventId={summarizingEventId}
              getSummarizeError={getSummarizeError}
              onClearSummarizeError={onClearSummarizeError}
              onToggleSummarized={handleToggleSummarized}
              onChangeLevel={handleChangeLevel}
              playbackState={playbackState}
              onSetPlaybackRate={onSetPlaybackRate}
              onSetVolume={onSetVolume}
              onSetMuted={onSetMuted}
              playbackFocusRequest={playbackFocusRequest}
              toolbarTrailingAction={messagesToolbarTrailingAction}
            />
          </div>
        )}
      </div>
    </div>
  );
}

export default memo(WorkspacePaneShell, (prev, next) => (
  prev.paneMeta === next.paneMeta
  && prev.layoutMode === next.layoutMode
  && prev.gridColumn === next.gridColumn
  && prev.gridRow === next.gridRow
  && prev.paneIndex === next.paneIndex
  && prev.isActive === next.isActive
  && prev.isBeingDragged === next.isBeingDragged
  && prev.isDropTarget === next.isDropTarget
  && prev.isVisible === next.isVisible
  && prev.isTtsSpeaking === next.isTtsSpeaking
  && prev.activeSpeakingEventId === next.activeSpeakingEventId
  && prev.loadingEventId === next.loadingEventId
  && prev.summarizeLevel === next.summarizeLevel
  && prev.summarizingEventId === next.summarizingEventId
  && prev.playbackState === next.playbackState
  && prev.playbackFocusRequest === next.playbackFocusRequest
  // In tabs mode the return-to-terminal affordance is passed through the
  // Messages toolbar.  It must participate in memo comparison: the Zustand
  // view-mode subscriber can re-render this shell before its parent supplies
  // the newly-created trailing action, otherwise the toolbar intermittently
  // retains the prior undefined prop.
  && prev.messagesToolbarTrailingAction === next.messagesToolbarTrailingAction
));
