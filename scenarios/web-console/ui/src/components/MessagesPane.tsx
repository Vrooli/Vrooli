import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import {
  ArrowDown,
  ArrowUpRight,
  Check,
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Copy,
  FileCode2,
  Loader2,
  Play,
  RotateCw,
  Search,
  Volume2,
} from "lucide-react";
import { useConversationStore, getSessionConversationEvents, getSessionSlice, resolveConversationView } from "../stores/useConversationStore";
import { loadConversationPageContaining, loadOlderConversationPage, refreshConversationSession } from "../hooks/useConversationSession";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useLiveStreamNotice } from "../hooks/useLiveStreamNotice";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useAnchoredPopoverPosition, type FloatingPlacement } from "../hooks/useFloatingPosition";
import { writeText } from "../lib/clipboard";
import { getConversationRange, searchConversation, type ConversationEvent, type ConversationSearchMatch } from "../api/conversation";
import { useFilePreviewController } from "./file-preview/useFilePreviewController";
import { TERMINAL_FONT_SIZE } from "../consts/config";
import { cn } from "../lib/classnames";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { looksLikeFileReference } from "../lib/fileReferences";
import { MarkdownRenderer } from "./markdown";
import { useVirtualList } from "../hooks/useVirtualList";
import { useReleaseOnElementInteraction } from "../hooks/useKeyboardListeners";
import MessageJumpList, { type MessageExportSelection } from "./MessageJumpList";
import { getDerived } from "./MessageJumpList.helpers";
import MessageExportDrawer from "./MessageExportDrawer";
import { AudioSettingsContent } from "./tts/AudioSettingsContent";
import { PlaybackModeControl, type SummarizationLevel } from "./tts/PlaybackModeControl";
import type { TTSPlaybackState } from "../audio-integration";
import type { PlaybackFocusRequest, PlaybackVersion } from "../domains/tts-playback/types";
import MessagesFileViewer from "./MessagesFileViewer";
import HandoffSuggestionChip from "./handoff/HandoffSuggestionChip";
import { useHandoffSuggestions } from "../hooks/useHandoffSuggestions";
import MessagesMermaidViewer from "./MessagesMermaidViewer";
import MessagesPaneState from "./MessagesPaneState";
import MessagesPaneStatusLine from "./MessagesPaneStatusLine";
import { resolveMessagesPaneStatus } from "../lib/messagesPaneStatus";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MessagesPaneProps {
  sessionId: string;
  /**
   * Hand a file (or a matched passage) from this session to another in its
   * group. Threaded down to the file viewer, which already holds both the
   * resolved path and this session id.
   */
  onHandoff?: (sessionId: string, payload: string) => void;
  onPlayFromHere: (eventId: string) => void;
  onPlayEvent: (eventId: string) => void;
  activeSpeakingEventId: string | null;
  loadingEventId?: string | null;
  isTtsSpeaking: boolean;
  summarizeLevel: SummarizationLevel;
  selectedVersionForEvent: (event: ConversationEvent) => PlaybackVersion;
  summarizingEventId: string | null;
  getSummarizeError: (eventId: string) => string | null;
  onClearSummarizeError: (eventId: string) => void;
  onToggleSummarized: (eventId: string, useSummarized: boolean) => void;
  onChangeLevel: (eventId: string, level: SummarizationLevel) => void;
  playbackState: TTSPlaybackState;
  onSetPlaybackRate: (rate: number) => void;
  onSetVolume: (level: number) => void;
  onSetMuted: (next: boolean) => void;
  playbackFocusRequest: PlaybackFocusRequest | null;
  toolbarTrailingAction?: ReactNode;
  /** Removes every transcript-mutating affordance while preserving navigation,
   * rendering, copy/export, Mermaid, and file preview behavior. */
  readOnly?: boolean;
  /** Optional archive hit to reveal after its containing page is loaded. */
  focusEventId?: string | null;
  focusSequence?: number | null;
  /** Stages a message in the operator-selected live session composer. */
  onSendToComposer?: (text: string) => void;
}

// ---------------------------------------------------------------------------
// Collapse threshold (px of rendered content before collapsing)
// ---------------------------------------------------------------------------
const COLLAPSE_THRESHOLD_PX = 400;

// ---------------------------------------------------------------------------
// Scroll snapshot persistence — keeps a per-session record of where the user
// left the messages pane so re-mounting after a view switch restores their
// position instead of dumping them somewhere in the middle.
// ---------------------------------------------------------------------------
interface ScrollSnapshot {
  atBottom: boolean;
  topEventId: string | null;
}

interface PrependScrollAnchor {
  eventId: string;
  offsetFromViewportTop: number;
}

const scrollSnapshotKey = (sessionId: string) => `wc.messagesScroll.${sessionId}`;

function readScrollSnapshot(sessionId: string): ScrollSnapshot | null {
  try {
    const raw = sessionStorage.getItem(scrollSnapshotKey(sessionId));
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<ScrollSnapshot>;
    if (typeof parsed.atBottom !== "boolean") return null;
    return {
      atBottom: parsed.atBottom,
      topEventId: typeof parsed.topEventId === "string" ? parsed.topEventId : null,
    };
  } catch {
    return null;
  }
}

function writeScrollSnapshot(sessionId: string, snapshot: ScrollSnapshot): void {
  try {
    sessionStorage.setItem(scrollSnapshotKey(sessionId), JSON.stringify(snapshot));
  } catch {
    // Ignore — sessionStorage may be unavailable in some embeddings.
  }
}

// ---------------------------------------------------------------------------
// MessagesPane
// ---------------------------------------------------------------------------

// The subset of TTSPlaybackState a message row's audio-settings popover needs.
// We deliberately exclude the high-frequency fields (currentTime/duration/
// isPaused) that the player polls at ~10 Hz during playback — including them
// would re-render every visible row on each poll. The parent memoizes this
// object so its identity is stable until one of these four values changes.
type MessageAudioSettings = Pick<
  TTSPlaybackState,
  "volume" | "isMuted" | "playbackRate" | "capabilities"
>;

interface MessageRowProps {
  event: ConversationEvent;
  index: number;
  registerItem: (index: number, node: HTMLElement | null) => void;
  fontSize: number;
  copiedEventId: string | null;
  onCopy: (eventId: string, text: string) => void;
  onPlayFromHere: (eventId: string) => void;
  onPlayEvent: (eventId: string) => void;
  isTtsSpeaking: boolean;
  activeSpeakingEventId: string | null;
  loadingEventId: string | null;
  summarizeLevel: SummarizationLevel;
  selectedVersionForEvent: (event: ConversationEvent) => PlaybackVersion;
  summarizingEventId: string | null;
  getSummarizeError: (eventId: string) => string | null;
  onClearSummarizeError: (eventId: string) => void;
  onToggleSummarized: (eventId: string, useSummarized: boolean) => void;
  onChangeLevel: (eventId: string, level: SummarizationLevel) => void;
  audioSettings: MessageAudioSettings;
  onSetPlaybackRate: (rate: number) => void;
  onSetVolume: (level: number) => void;
  onSetMuted: (next: boolean) => void;
  isMobile: boolean;
  isFocused: boolean;
  isSearchFocused: boolean;
  isDimmed: boolean;
  isExpanded: boolean;
  onToggleExpanded: (eventId: string) => void;
  isPlaintext: boolean;
  onToggleRenderMode: (eventId: string) => void;
  onLinkClick: (href: string, event: React.MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick: (path: string) => void;
  onMermaidOpen: (code: string) => void;
  readOnly: boolean;
  onSendToComposer?: (text: string) => void;
}

/** Anchored placement order for popovers opening below their trigger. */
const BELOW_ANCHOR_PLACEMENTS: FloatingPlacement[] = ["bottom-end", "bottom-start", "top-end", "top-start"];

const MessageRow = memo(function MessageRow({
  event,
  index,
  registerItem,
  fontSize,
  copiedEventId,
  onCopy,
  onPlayFromHere,
  onPlayEvent,
  isTtsSpeaking,
  activeSpeakingEventId,
  loadingEventId,
  summarizeLevel,
  selectedVersionForEvent,
  summarizingEventId,
  getSummarizeError,
  onClearSummarizeError,
  onToggleSummarized,
  onChangeLevel,
  audioSettings,
  onSetPlaybackRate,
  onSetVolume,
  onSetMuted,
  isMobile,
  isFocused,
  isSearchFocused,
  isDimmed,
  isExpanded,
  onToggleExpanded,
  isPlaintext,
  onToggleRenderMode,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
  readOnly,
  onSendToComposer,
}: MessageRowProps) {
  const { t } = useTranslation();
  const [openPopoverId, setOpenPopoverId] = useState<string | null>(null);
  const [isTall, setIsTall] = useState(false);
  const audioButtonRef = useRef<HTMLButtonElement | null>(null);
  const contentRef = useRef<HTMLDivElement | null>(null);
  // Desktop audio popover anchors below the per-message audio button,
  // end-aligned, via the shared anchored-floating math.
  const audioPopoverRef = useRef<HTMLDivElement | null>(null);
  const audioPopoverStyle = useAnchoredPopoverPosition(
    openPopoverId === event.id && !isMobile,
    audioButtonRef,
    audioPopoverRef,
    BELOW_ANCHOR_PLACEMENTS,
  );

  useEffect(() => {
    const node = contentRef.current;
    if (!node) return;

    const measure = () => { setIsTall(node.scrollHeight > COLLAPSE_THRESHOLD_PX); };
    measure();

    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(() => { measure(); });
    observer.observe(node);
    return () => { observer.disconnect(); };
  }, [event.text, isExpanded, isPlaintext]);

  const isUser = event.role === "user";
  const isTtsActive = !isUser && isTtsSpeaking && activeSpeakingEventId === event.id;
  const isAudioLoading = !isUser && loadingEventId === event.id;
  const hasSummary = event.summarized && event.originalSpeechParagraphs != null && event.originalSpeechParagraphs.length > 0;
  const selectedVersion = selectedVersionForEvent(event);
  const useSummarized = selectedVersion === "active" && hasSummary;
  const isPopoverOpen = openPopoverId === event.id;
  const isCollapsed = isTall && !isExpanded;
  const accentColor = isTtsActive
    ? "border-l-wc-accent"
    : isUser
      ? "border-l-sky-500/60"
      : "border-l-emerald-500/60";

  return (
    <article
      ref={(node) => { registerItem(index, node); }}
      data-testid={`msg-card-${event.id}`}
      className={cn(
        "border-b border-wc-default border-l-[3px] py-3 ps-3 pe-1 transition-colors",
        accentColor,
        isFocused && "bg-wc-accent/5",
        isSearchFocused && "ring-1 ring-wc-accent/50 rounded-r-lg",
        isDimmed && "opacity-40",
      )}
    >
      <div className="mb-1.5 flex items-center gap-2 text-[11px] uppercase tracking-[0.12em] text-wc-text-faint">
        <button
          data-testid={`msg-copy-${event.id}`}
          onClick={() => { onCopy(event.id, event.text); }}
          className="rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10"
          title={t(strings.messagesPane.copyMessageTitle)}
          type="button"
        >
          {copiedEventId === event.id
            ? <Check className="h-3.5 w-3.5 text-green-400" />
            : <Copy className="h-3.5 w-3.5" />}
        </button>

        <button
          data-testid={`msg-render-toggle-${event.id}`}
          onClick={() => { onToggleRenderMode(event.id); }}
          aria-pressed={isPlaintext}
          className={cn(
            "rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10",
            isPlaintext && "text-wc-accent",
          )}
          title={isPlaintext
            ? t(strings.messagesPane.viewAsMarkdownTitle)
            : t(strings.messagesPane.viewAsPlainTextTitle)}
          type="button"
        >
          <FileCode2 className="h-3.5 w-3.5" />
        </button>

        {!readOnly && !isUser && (
          <>
            <button
              data-testid={`msg-speak-from-${event.id}`}
              onClick={() => { onPlayFromHere(event.id); }}
              disabled={isAudioLoading}
              className={cn(
                "rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10",
                isAudioLoading && "cursor-wait opacity-60",
              )}
              title={t(strings.messagesPane.readFromHereTitle)}
              type="button"
            >
              <Play className="h-3.5 w-3.5" />
            </button>
            <PlaybackModeControl
              testIdPrefix={`msg-${event.id}`}
              isSummarized={useSummarized && hasSummary}
              hasOriginalVersion={hasSummary}
              canSummarize
              isSummarizing={summarizingEventId === event.id}
              currentLevel={summarizeLevel}
              onToggleSummarized={(use) => { onToggleSummarized(event.id, use); }}
              onChangeLevel={(level) => { onChangeLevel(event.id, level); }}
            />
            <button
              ref={audioButtonRef}
              data-testid={`msg-audio-${event.id}`}
              onClick={() => {
                onPlayEvent(event.id);
                setOpenPopoverId(isPopoverOpen ? null : event.id);
              }}
              disabled={isAudioLoading}
              className={cn(
                "rounded p-0.5 text-wc-text-faint transition hover:text-wc-text-muted hover:bg-wc-accent/10",
                isAudioLoading && "cursor-wait text-wc-accent opacity-80",
              )}
              title={t(strings.messagesPane.playAudioSettingsTitle)}
              type="button"
            >
              {isAudioLoading
                ? <Loader2 data-testid={`msg-audio-loading-${event.id}`} className="h-3 w-3 animate-spin" />
                : <Volume2 className="h-3 w-3" />}
            </button>

            {isPopoverOpen && createPortal(
              isMobile ? (
                <div className="fixed inset-0 z-wc-popover-backdrop" onMouseDown={(e) => { e.preventDefault(); }}>
                  <div className="absolute inset-0 bg-wc-backdrop" onClick={() => { setOpenPopoverId(null); }} />
                  <div
                    data-testid={`audio-popover-${event.id}`}
                    className="wc-stable-theme absolute bottom-0 left-0 right-0 z-wc-popover rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-4 pb-[max(1rem,var(--wc-safe-bottom))] ps-[max(1rem,var(--wc-safe-left,0px))] pe-[max(1rem,var(--wc-safe-right,0px))] shadow-2xl"
                  >
                    <div className="mb-3 flex justify-center">
                      <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
                    </div>
                    <h3 className="mb-3 text-sm font-semibold text-wc-text-primary">{t(strings.messagesPane.audioSettingsHeading)}</h3>
                    <AudioSettingsContent
                      testIdPrefix={`msg-${event.id}`}
                      volume={audioSettings.volume}
                      isMuted={audioSettings.isMuted}
                      playbackRate={audioSettings.playbackRate}
                      isSummarized={useSummarized && hasSummary}
                      capabilities={audioSettings.capabilities}
                      onVolumeChange={onSetVolume}
                      onSetMuted={onSetMuted}
                      onSetPlaybackRate={onSetPlaybackRate}
                    />
                    {isAudioLoading && (
                      <div data-testid={`audio-popover-loading-${event.id}`} className="mt-3 flex items-center gap-2 rounded-lg bg-wc-surface-base px-3 py-2 text-xs text-wc-text-muted">
                        <Loader2 className="h-3.5 w-3.5 animate-spin text-wc-accent" />
                        <span>{t(strings.app.loading)}</span>
                      </div>
                    )}
                    {getSummarizeError(event.id) && (
                      <div
                        data-testid={`msg-summarize-error-${event.id}`}
                        className="mt-2 rounded-lg bg-red-500/10 px-3 py-2 text-[11px] text-red-400"
                      >
                        {getSummarizeError(event.id)}
                      </div>
                    )}
                    {getSummarizeError(event.id) && (
                      <button
                        data-testid={`msg-clear-summarize-error-${event.id}`}
                        className="mt-2 w-full rounded-lg bg-wc-surface-base px-3 py-2 text-xs font-medium text-wc-text-muted transition hover:bg-wc-surface-input"
                        onClick={() => { onClearSummarizeError(event.id); }}
                      >
                        {t(strings.messagesPane.dismissError)}
                      </button>
                    )}
                  </div>
                </div>
              ) : (
                <>
                  <div className="fixed inset-0 z-wc-popover-backdrop" onClick={() => { setOpenPopoverId(null); }} />
                  <div
                    ref={audioPopoverRef}
                    data-testid={`audio-popover-${event.id}`}
                    className="wc-stable-theme z-wc-popover w-56 rounded-xl border border-wc-default bg-wc-surface-raised p-3 shadow-lg"
                    style={audioPopoverStyle}
                  >
                    <AudioSettingsContent
                      testIdPrefix={`msg-${event.id}`}
                      volume={audioSettings.volume}
                      isMuted={audioSettings.isMuted}
                      playbackRate={audioSettings.playbackRate}
                      isSummarized={useSummarized && hasSummary}
                      capabilities={audioSettings.capabilities}
                      onVolumeChange={onSetVolume}
                      onSetMuted={onSetMuted}
                      onSetPlaybackRate={onSetPlaybackRate}
                    />
                    {isAudioLoading && (
                      <div data-testid={`audio-popover-loading-${event.id}`} className="mt-3 flex items-center gap-2 rounded-lg bg-wc-surface-base px-3 py-2 text-xs text-wc-text-muted">
                        <Loader2 className="h-3.5 w-3.5 animate-spin text-wc-accent" />
                        <span>{t(strings.app.loading)}</span>
                      </div>
                    )}
                    {getSummarizeError(event.id) && (
                      <div
                        data-testid={`msg-summarize-error-${event.id}`}
                        className="mt-2 rounded-lg bg-red-500/10 px-3 py-2 text-[11px] text-red-400"
                      >
                        {getSummarizeError(event.id)}
                      </div>
                    )}
                    {getSummarizeError(event.id) && (
                      <button
                        data-testid={`msg-clear-summarize-error-${event.id}`}
                        className="mt-2 w-full rounded-lg bg-wc-surface-base px-3 py-2 text-xs font-medium text-wc-text-muted transition hover:bg-wc-surface-input"
                        onClick={() => { onClearSummarizeError(event.id); }}
                      >
                        {t(strings.messagesPane.dismissError)}
                      </button>
                    )}
                  </div>
                </>
              ),
              document.body,
            )}
          </>
        )}

        {onSendToComposer && (
          <button
            data-testid={`msg-send-to-composer-${event.id}`}
            onClick={() => { onSendToComposer(event.text); }}
            className="rounded p-0.5 text-wc-text-muted transition hover:bg-wc-accent/10 hover:text-wc-text-primary"
            title={t(strings.messagesPane.sendToComposerTitle)}
            type="button"
          >
            <ArrowUpRight className="h-3.5 w-3.5" />
          </button>
        )}

        <span className="flex-1" />
        <span>#{event.sequence}</span>
      </div>

      <div className={cn("relative", isCollapsed && "max-h-[400px] overflow-hidden")}>
        <div
          ref={contentRef}
          data-testid={`msg-markdown-${event.id}`}
          style={{ fontSize: `${fontSize}px` }}
          className="text-wc-text-primary"
        >
          {isPlaintext ? (
            <pre
              data-testid={`msg-plaintext-${event.id}`}
              className="whitespace-pre-wrap break-words [overflow-wrap:anywhere] font-mono"
            >
              {event.text}
            </pre>
          ) : (
            <MarkdownRenderer content={event.text} onLinkClick={onLinkClick} onFileReferenceClick={onFileReferenceClick} onMermaidOpen={onMermaidOpen} />
          )}
        </div>

        {isCollapsed && (
          <div className="absolute bottom-0 left-0 right-0 h-20 bg-gradient-to-t from-wc-surface-base to-transparent pointer-events-none" />
        )}
      </div>

      {isTall && (
        <button
          data-testid={`msg-collapse-${event.id}`}
          onClick={() => { onToggleExpanded(event.id); }}
          className="mt-1 text-xs text-wc-accent hover:text-wc-accent/80 transition-colors"
          type="button"
        >
          {isExpanded ? t(strings.messagesPane.showLess) : t(strings.messagesPane.showMore)}
        </button>
      )}
    </article>
  );
}, (prevProps, nextProps) => (
  prevProps.event === nextProps.event &&
  prevProps.fontSize === nextProps.fontSize &&
  prevProps.copiedEventId === nextProps.copiedEventId &&
  prevProps.isTtsSpeaking === nextProps.isTtsSpeaking &&
  prevProps.activeSpeakingEventId === nextProps.activeSpeakingEventId &&
  prevProps.loadingEventId === nextProps.loadingEventId &&
  prevProps.summarizeLevel === nextProps.summarizeLevel &&
  prevProps.summarizingEventId === nextProps.summarizingEventId &&
  prevProps.audioSettings === nextProps.audioSettings &&
  prevProps.isMobile === nextProps.isMobile &&
  prevProps.isFocused === nextProps.isFocused &&
  prevProps.isSearchFocused === nextProps.isSearchFocused &&
  prevProps.isDimmed === nextProps.isDimmed &&
  prevProps.isExpanded === nextProps.isExpanded &&
  prevProps.isPlaintext === nextProps.isPlaintext &&
  prevProps.readOnly === nextProps.readOnly &&
  prevProps.onSendToComposer === nextProps.onSendToComposer &&
  prevProps.onLinkClick === nextProps.onLinkClick &&
  prevProps.onFileReferenceClick === nextProps.onFileReferenceClick &&
  prevProps.onMermaidOpen === nextProps.onMermaidOpen
));

export default function MessagesPane({
  sessionId,
  onHandoff,
  onPlayFromHere,
  onPlayEvent,
  activeSpeakingEventId,
  loadingEventId = null,
  isTtsSpeaking,
  summarizeLevel,
  selectedVersionForEvent,
  summarizingEventId,
  getSummarizeError,
  onClearSummarizeError,
  onToggleSummarized,
  onChangeLevel,
  playbackState,
  onSetPlaybackRate,
  onSetVolume,
  onSetMuted,
  playbackFocusRequest,
  toolbarTrailingAction,
  readOnly = false,
  focusEventId = null,
  focusSequence = null,
  onSendToComposer,
}: MessagesPaneProps) {
  const { t } = useTranslation();
  const events = useConversationStore((state) => getSessionConversationEvents(state, sessionId));
  const totalCount = useConversationStore((state) => state.sessions[sessionId]?.totalCount ?? events.length);
  // The session slice is referentially stable, so deriving the view from it in
  // a memo avoids re-rendering the pane on every unrelated store write. The
  // view — not events.length — decides what this pane shows.
  const sessionSlice = useConversationStore((state) => getSessionSlice(state, sessionId));
  const viewState = useMemo(() => resolveConversationView(sessionSlice), [sessionSlice]);
  // Only after the interruption outlives its grace period; most drops recover
  // faster than the sentence can be read.
  const liveInterrupted = useLiveStreamNotice();
  const isMobile = useMediaQuery("(max-width: 767px)");

  // Stable subset of playbackState for the per-message audio popover. Keeping
  // its identity stable across the player's ~10 Hz time polls is what stops
  // every visible row from re-rendering during TTS playback.
  const audioSettings = useMemo<MessageAudioSettings>(
    () => ({
      volume: playbackState.volume,
      isMuted: playbackState.isMuted,
      playbackRate: playbackState.playbackRate,
      capabilities: playbackState.capabilities,
    }),
    [playbackState.volume, playbackState.isMuted, playbackState.playbackRate, playbackState.capabilities],
  );
  const fontSize = useWorkspaceStore(
    useCallback((s) => s.panes.find((p) => p.sessionId === sessionId)?.fontSize ?? TERMINAL_FONT_SIZE, [sessionId]),
  );

  // --- Copy ---
  const [copiedEventId, setCopiedEventId] = useState<string | null>(null);
  const handleCopy = useCallback((eventId: string, text: string) => {
    void writeText(text);
    setCopiedEventId(eventId);
    setTimeout(() => { setCopiedEventId((prev) => (prev === eventId ? null : prev)); }, 2000);
  }, []);

  // --- Search & navigation ---
  // The navigator panel owns search editing, but the query is lifted here so
  // the message list keeps dimming non-matches and the toolbar match-stepping
  // arrows keep working while (and after) the navigator is open.
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedEventId, setFocusedEventId] = useState<string | null>(null);
  const [serverSearchMatches, setServerSearchMatches] = useState<ConversationSearchMatch[]>([]);
  const [serverSearchReady, setServerSearchReady] = useState(false);
  const [searchTruncated, setSearchTruncated] = useState(false);

  // --- Navigator panel ---
  const [navOpen, setNavOpen] = useState(false);
  const [navInitialFocus, setNavInitialFocus] = useState<"search" | "list">("list");
  const openNavigator = useCallback((focus: "search" | "list") => {
    setNavInitialFocus(focus);
    setNavOpen(true);
  }, []);
  const handleNavQueryChange = useCallback((q: string) => {
    setSearchQuery(q);
    setFocusedEventId(null);
  }, []);

  // --- Collapse ---
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

  // --- Render mode (markdown by default; ids in this set show plain text) ---
  const [plaintextIds, setPlaintextIds] = useState<Set<string>>(new Set());
  const toggleRenderMode = useCallback((eventId: string) => {
    setPlaintextIds((prev) => {
      const next = new Set(prev);
      if (next.has(eventId)) next.delete(eventId);
      else next.add(eventId);
      return next;
    });
  }, []);

  // --- Export selection (session-scoped source of truth shared by the
  // navigator's selection mode and the export drawer) ---
  const [exportSelectedIds, setExportSelectedIds] = useState<ReadonlySet<string>>(new Set());
  const [exportEventsById, setExportEventsById] = useState<ReadonlyMap<string, ConversationEvent>>(new Map());
  const [exportDrawerOpen, setExportDrawerOpen] = useState(false);

  // --- File preview ---
  const filePreview = useFilePreviewController(sessionId);
  // Rules only ever offer. Nothing on this path can send.
  const handoffSuggestions = useHandoffSuggestions(sessionId);

  // --- Auto-scroll ---
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const isNearBottomRef = useRef(true);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [newMessageCount, setNewMessageCount] = useState(0);
  const prevEventCountRef = useRef(events.length);
  // While true, the totalSize-change effect re-pins the scroll position to the
  // bottom. Cleared once the user scrolls away from the bottom.
  const pinToBottomRef = useRef(true);
  // While set, the totalSize-change effect re-scrolls to this event id until
  // the row's measured size is stable. Cleared after restore completes or the
  // user manually scrolls.
  const pinToEventIdRef = useRef<string | null>(null);
  const programmaticScrollRef = useRef(false);
  const pinSettleCountRef = useRef(0);
  const pinTargetRef = useRef<string | null>(null);
  const programmaticTimeoutRef = useRef<number | null>(null);
  const restoreAppliedRef = useRef(false);
  // Pagination inserts older rows before the visible window. Keep a DOM
  // anchor through that insertion so loading another page feels like normal
  // continuous upward scrolling rather than a jump or a captured wheel.
  const prependScrollAnchorRef = useRef<PrependScrollAnchor | null>(null);

  const runProgrammaticScroll = useCallback((scroll: () => void) => {
    programmaticScrollRef.current = true;
    if (programmaticTimeoutRef.current != null) window.clearTimeout(programmaticTimeoutRef.current);
    scroll();
    const el = scrollContainerRef.current;
    if (el && "onscrollend" in el) {
      el.addEventListener("scrollend", () => {
        programmaticScrollRef.current = false;
      }, { once: true });
    }
    let previous = el?.scrollTop ?? 0;
    let settledFrames = 0;
    const settle = () => {
      const current = el?.scrollTop ?? 0;
      settledFrames = current === previous ? settledFrames + 1 : 0;
      previous = current;
      if (settledFrames >= 2) {
        programmaticScrollRef.current = false;
        return;
      }
      requestAnimationFrame(settle);
    };
    requestAnimationFrame(settle);
    programmaticTimeoutRef.current = window.setTimeout(() => {
      programmaticScrollRef.current = false;
    }, 1200);
  }, []);

  useEffect(() => () => {
    if (programmaticTimeoutRef.current != null) window.clearTimeout(programmaticTimeoutRef.current);
  }, []);

  // --- Refresh: on mount, on browser tab focus, and via manual button ---
  const [isRefreshing, setIsRefreshing] = useState(false);
  // What the last manual refresh did. A spinner that stops is not feedback:
  // "fetched successfully, nothing new" and "the request failed" previously
  // looked identical, which is why refresh appeared to do nothing at all.
  // A failed refresh persists until the next attempt; a successful one is a
  // brief confirmation. They are separate because they have different
  // lifetimes and different priority against the live-stream notice.
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [transientNotice, setTransientNotice] = useState<string | null>(null);
  const refreshNoticeTimer = useRef<number | null>(null);

  const showTransientNotice = useCallback((text: string) => {
    setTransientNotice(text);
    if (refreshNoticeTimer.current != null) window.clearTimeout(refreshNoticeTimer.current);
    refreshNoticeTimer.current = window.setTimeout(() => { setTransientNotice(null); }, 3000);
  }, []);

  useEffect(() => () => {
    if (refreshNoticeTimer.current != null) window.clearTimeout(refreshNoticeTimer.current);
  }, []);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    try {
      const outcome = await refreshConversationSession(sessionId);
      if (!outcome.ok) {
        setRefreshError(outcome.error.message);
        setTransientNotice(null);
      } else {
        setRefreshError(null);
        showTransientNotice(outcome.addedEvents > 0
          ? t(strings.messagesPane.refreshAdded, { count: outcome.addedEvents })
          : t(strings.messagesPane.refreshUpToDate));
      }
    } finally {
      setIsRefreshing(false);
    }
  }, [sessionId, showTransientNotice, t]);

  // Refresh on pane mount (covers switching view mode back to messages). Also
  // refresh whenever the tab becomes visible again — the server may have
  // delivered events while the WS was background-throttled or dropped on a
  // full client buffer (conversation_out_of_sync also handles the latter, but
  // a missed signal can happen during reconnects).
  useEffect(() => {
    void refreshConversationSession(sessionId);
    const onVisibility = () => {
      if (!document.hidden) {
        void refreshConversationSession(sessionId);
      }
    };
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("focus", onVisibility);
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("focus", onVisibility);
    };
  }, [sessionId]);

  useEffect(() => {
    const el = scrollContainerRef.current;
    if (!el) return;

    const updateNearBottom = (allowPagination = true) => {
      const remaining = el.scrollHeight - (el.scrollTop + el.clientHeight);
      const nearBottom = remaining <= 200;
      isNearBottomRef.current = nearBottom;
      setIsNearBottom(nearBottom);
      if (nearBottom) setNewMessageCount(0);
      if (!nearBottom && !programmaticScrollRef.current) {
        pinToBottomRef.current = false;
        pinToEventIdRef.current = null;
      }
      if (allowPagination && el.scrollTop <= el.clientHeight * 2 && !prependScrollAnchorRef.current) {
        const containerTop = el.getBoundingClientRect().top;
        const anchor = [...el.querySelectorAll<HTMLElement>("[data-event-id]")]
          .find((row) => row.getBoundingClientRect().bottom > containerTop);
        if (anchor?.dataset.eventId) {
          prependScrollAnchorRef.current = {
            eventId: anchor.dataset.eventId,
            offsetFromViewportTop: anchor.getBoundingClientRect().top - containerTop,
          };
        }
        void loadOlderConversationPage(sessionId).then((loaded) => {
          // A failed/no-op request never causes a render where the layout
          // effect can consume this anchor.
          if (!loaded) prependScrollAnchorRef.current = null;
        });
      }
    };

    updateNearBottom(false);
    const onScroll = () => { updateNearBottom(true); };
    el.addEventListener("scroll", onScroll, { passive: true });
    return () => { el.removeEventListener("scroll", onScroll); };
  }, [sessionId]);

  // Release the bottom-pin or event-pin as soon as the user scrolls away from
  // it. We do this from a wheel/touchstart listener so synthetic re-scrolls
  // from the totalSize-change effect don't release the pin.
  useReleaseOnElementInteraction(scrollContainerRef, () => {
    pinToBottomRef.current = false;
    pinToEventIdRef.current = null;
  });

  // Auto-scroll on new events (when near bottom) or show pill
  useEffect(() => {
    const newCount = events.length - prevEventCountRef.current;
    prevEventCountRef.current = events.length;

    if (newCount <= 0) return;

    if (isNearBottomRef.current) {
      requestAnimationFrame(() => {
        runProgrammaticScroll(() => {
          scrollContainerRef.current?.scrollTo({
            top: scrollContainerRef.current.scrollHeight,
            behavior: "auto",
          });
        });
      });
    } else {
      setNewMessageCount((prev) => prev + newCount);
    }
  }, [events.length, runProgrammaticScroll]);

  const scrollToBottom = useCallback(() => {
    pinToBottomRef.current = true;
    pinToEventIdRef.current = null;
    runProgrammaticScroll(() => {
      scrollContainerRef.current?.scrollTo({
        top: scrollContainerRef.current.scrollHeight,
        behavior: "smooth",
      });
    });
    setNewMessageCount(0);
  }, [runProgrammaticScroll]);

  // Search the entire session, rather than only the currently loaded window.
  // Highlighting still uses the returned ids against the bounded local window.
  useEffect(() => {
    const query = searchQuery.trim();
    if (!query) {
      setServerSearchMatches([]);
      setServerSearchReady(false);
      setSearchTruncated(false);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      void searchConversation(sessionId, query).then((response) => {
        if (!cancelled) {
          setServerSearchMatches(response.matches);
          setSearchTruncated(response.truncated);
          setServerSearchReady(true);
        }
      }).catch(() => {
        if (!cancelled) {
          setServerSearchMatches([]);
          setSearchTruncated(false);
          setServerSearchReady(false);
        }
      });
    }, 200);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [searchQuery, sessionId]);

  const searchMatchIds = useMemo(() => {
    if (!searchQuery) return [];
    if (serverSearchReady) return serverSearchMatches.map((match) => match.eventId);
    const query = searchQuery.toLowerCase();
    return events.filter((event) => getDerived(event).previewLower.includes(query)).map((event) => event.id);
  }, [events, searchQuery, serverSearchMatches, serverSearchReady]);
  const searchMatchById = useMemo(
    () => new Map(serverSearchMatches.map((match) => [match.eventId, match])),
    [serverSearchMatches],
  );
  const searchMatchSet = useMemo(() => new Set(searchMatchIds), [searchMatchIds]);
  const eventIds = useMemo(() => events.map((event) => event.id), [events]);
  const eventIndexById = useMemo(
    () => new Map(eventIds.map((id, index) => [id, index])),
    [eventIds],
  );
  const searchMatchIndexById = useMemo(
    () => new Map(searchMatchIds.map((id, index) => [id, index])),
    [searchMatchIds],
  );

  const navIds = useMemo(
    () => (searchQuery ? searchMatchIds : eventIds),
    [searchQuery, searchMatchIds, eventIds],
  );

  const focusedNavIndex = useMemo(
    () => {
      if (!focusedEventId) return -1;
      return searchQuery
        ? (searchMatchIndexById.get(focusedEventId) ?? -1)
        : (eventIndexById.get(focusedEventId) ?? -1);
    },
    [eventIndexById, focusedEventId, searchMatchIndexById, searchQuery],
  );

  const currentMatchIndex = useMemo(
    () => (focusedEventId ? (searchMatchIndexById.get(focusedEventId) ?? -1) : -1),
    [focusedEventId, searchMatchIndexById],
  );

  const estimateMessageHeight = useCallback((index: number) => {
    const event = events[index];
    if (!event) return 140;
    const lineEstimate = Math.ceil(event.text.length / 90);
    return Math.max(110, Math.min(520, 72 + lineEstimate * 22));
  }, [events]);
  const getMessageKey = useCallback((index: number) => events[index]?.id ?? index, [events]);
  const { registerItem, totalSize, virtualItems, scrollToIndex } = useVirtualList({
    count: events.length,
    estimateSize: estimateMessageHeight,
    getItemKey: getMessageKey,
    overscan: 8,
    scrollElementRef: scrollContainerRef,
    enabled: events.length > 40,
  });

  // useLayoutEffect runs after React has placed the prepended page but before
  // the browser paints it.  The former anchor is temporarily outside the
  // virtual window at this point, so use its new virtual index rather than a
  // DOM lookup. Subsequent row measurement adjustments are anchored by the
  // virtualizer itself.
  useLayoutEffect(() => {
    const anchor = prependScrollAnchorRef.current;
    const el = scrollContainerRef.current;
    if (!anchor || !el) return;
    const index = events.findIndex((event) => event.id === anchor.eventId);
    if (index < 0) return;
    scrollToIndex(index, "auto", "start");
    el.scrollTop -= anchor.offsetFromViewportTop;
    prependScrollAnchorRef.current = null;
  }, [events, scrollToIndex]);

  const scrollToEvent = useCallback(async (eventId: string) => {
    const index = eventIndexById.get(eventId);
    if (index == null) {
      const match = searchMatchById.get(eventId);
      if (!match || !await loadConversationPageContaining(sessionId, match.sequence)) return;
      requestAnimationFrame(() => {
        const loadedIndex = useConversationStore.getState().sessions[sessionId]?.events.findIndex((event) => event.id === eventId) ?? -1;
        if (loadedIndex >= 0) runProgrammaticScroll(() => { scrollToIndex(loadedIndex, "auto", "center"); });
      });
      return;
    }
    pinToBottomRef.current = false;
    pinToEventIdRef.current = null;
    runProgrammaticScroll(() => { scrollToIndex(index, "smooth", "center"); });
  }, [eventIndexById, runProgrammaticScroll, scrollToIndex, searchMatchById, sessionId]);

  // Restore scroll position on mount: read the snapshot saved when the pane
  // last unmounted and pin to the appropriate target. The pin is held until
  // the virtualizer's measured sizes stabilize (handled by the totalSize
  // effect below).
  useEffect(() => {
    if (restoreAppliedRef.current) return;
    if (events.length === 0) return;
    const snapshot = readScrollSnapshot(sessionId);
    if (!snapshot || snapshot.atBottom) {
      pinToBottomRef.current = true;
      pinToEventIdRef.current = null;
    } else if (snapshot.topEventId && eventIndexById.has(snapshot.topEventId)) {
      pinToBottomRef.current = false;
      pinToEventIdRef.current = snapshot.topEventId;
    } else {
      pinToBottomRef.current = true;
      pinToEventIdRef.current = null;
    }
    restoreAppliedRef.current = true;
    // Trigger an immediate apply; the totalSize effect will keep re-applying
    // as measurements settle.
    const el = scrollContainerRef.current;
    if (!el) return;
    if (pinToBottomRef.current) {
      runProgrammaticScroll(() => { el.scrollTo({ top: el.scrollHeight }); });
    } else if (pinToEventIdRef.current) {
      const index = eventIndexById.get(pinToEventIdRef.current);
      if (index != null) runProgrammaticScroll(() => { scrollToIndex(index, "auto", "start"); });
    }
  }, [events.length, sessionId, eventIndexById, runProgrammaticScroll, scrollToIndex]);

  // Re-apply the active pin whenever the virtualizer's totalSize changes.
  // This is what fixes the "lands in the middle" bug: estimated sizes are
  // smaller than actual, so the initial scrollTo lands too high; once rows
  // measure their real heights totalSize grows and we re-scroll.
  useEffect(() => {
    const el = scrollContainerRef.current;
    if (!el) return;
    const target = pinToBottomRef.current ? "bottom" : pinToEventIdRef.current;
    if (!target) return;
    pinSettleCountRef.current = pinTargetRef.current === target ? pinSettleCountRef.current + 1 : 0;
    pinTargetRef.current = target;
    if (pinSettleCountRef.current >= 8) {
      pinToBottomRef.current = false;
      pinToEventIdRef.current = null;
      pinTargetRef.current = null;
      return;
    }
    if (pinToBottomRef.current) {
      runProgrammaticScroll(() => { el.scrollTo({ top: el.scrollHeight }); });
    } else if (pinToEventIdRef.current) {
      const index = eventIndexById.get(pinToEventIdRef.current);
      if (index != null) runProgrammaticScroll(() => { scrollToIndex(index, "auto", "start"); });
    }
  }, [totalSize, eventIndexById, runProgrammaticScroll, scrollToIndex]);

  // Save snapshot on unmount and whenever sessionId changes. We compute
  // `atBottom` directly from the live DOM instead of trusting
  // isNearBottomRef.current — under React StrictMode the dev-only second
  // mount unmounts synchronously, before the (async/passive) scroll listener
  // has had a chance to observe the restore's programmatic scrollTo, so the
  // ref can be stale (`true` from its initial value). Reading geometry here
  // is the source of truth.
  useEffect(() => {
    const containerEl = scrollContainerRef.current;
    return () => {
      const el = containerEl;
      if (!el) return;
      const remaining = el.scrollHeight - (el.scrollTop + el.clientHeight);
      const atBottom = remaining <= 200;
      let topEventId: string | null = null;
      if (!atBottom) {
        const containerTop = el.getBoundingClientRect().top;
        const rows = el.querySelectorAll<HTMLElement>("[data-event-id]");
        for (const row of rows) {
          if (row.getBoundingClientRect().bottom - containerTop > 0) {
            topEventId = row.dataset.eventId ?? null;
            break;
          }
        }
      }
      // Skip writing if we have nothing actionable: it'd just overwrite a
      // previously-saved good snapshot with a default `atBottom: true`.
      if (!atBottom && !topEventId) return;
      writeScrollSnapshot(sessionId, { atBottom, topEventId });
    };
  }, [sessionId]);

  // Normalize the export selection whenever the conversation refreshes so it
  // never references events that no longer exist in the session.
  useEffect(() => {
    setExportSelectedIds((prev) => {
      if (prev.size === 0) return prev;
      const next = new Set<string>();
      for (const id of prev) {
        if (eventIndexById.has(id)) next.add(id);
      }
      return next.size === prev.size ? prev : next;
    });
  }, [eventIndexById]);

  const exportSelection = useMemo<MessageExportSelection>(
    () => ({
      selectedIds: exportSelectedIds,
      onToggle: (eventId) =>
        { setExportSelectedIds((prev) => {
          const next = new Set(prev);
          if (next.has(eventId)) next.delete(eventId);
          else next.add(eventId);
          return next;
        }); },
      onSelectAll: () => {
        setExportSelectedIds(new Set(eventIds));
        void getConversationRange(sessionId, 1, totalCount).then((response) => {
          setExportSelectedIds(new Set(response.events.map((event) => event.id)));
          setExportEventsById(new Map(response.events.map((event) => [event.id, event])));
        }).catch(() => undefined);
      },
      onSelectVisible: (visibleIds) => { setExportSelectedIds(new Set(visibleIds)); },
      onClear: () => { setExportSelectedIds(new Set()); },
      onContinue: () => { setExportDrawerOpen(true); },
    }),
    [exportSelectedIds, eventIds, sessionId, totalCount],
  );

  // Selected events in conversation order for the drawer/formatter.
  const exportEvents = useMemo(() => {
    const loadedById = new Map(events.map((event) => [event.id, event]));
    return [...exportSelectedIds]
      .map((id) => exportEventsById.get(id) ?? loadedById.get(id))
      .filter((event): event is ConversationEvent => event !== undefined)
      .sort((left, right) => left.sequence - right.sequence);
  }, [events, exportEventsById, exportSelectedIds]);

  const focusAndScroll = useCallback((eventId: string) => {
    setFocusedEventId(eventId);
    scrollToEvent(eventId);
    // Auto-expand if collapsed and a search match
    if (searchQuery) {
      setExpandedIds((prev) => new Set(prev).add(eventId));
    }
  }, [scrollToEvent, searchQuery]);

  useEffect(() => {
    if (!focusEventId || !focusSequence) return;
    let cancelled = false;
    const reveal = async () => {
      const present = useConversationStore.getState().sessions[sessionId]?.events.some((event) => event.id === focusEventId);
      if (!present && !await loadConversationPageContaining(sessionId, focusSequence)) return;
      if (!cancelled) requestAnimationFrame(() => { focusAndScroll(focusEventId); });
    };
    void reveal();
    return () => { cancelled = true; };
  }, [focusEventId, focusSequence, focusAndScroll, sessionId]);

  const handleNavUp = useCallback(() => {
    if (navIds.length === 0) return;
    const prev = focusedNavIndex <= 0 ? navIds.length - 1 : focusedNavIndex - 1;
    const targetId = navIds[prev];
    if (targetId) focusAndScroll(targetId);
  }, [navIds, focusedNavIndex, focusAndScroll]);

  const handleNavDown = useCallback(() => {
    if (navIds.length === 0) return;
    const next = focusedNavIndex < 0 ? 0 : (focusedNavIndex + 1) % navIds.length;
    const targetId = navIds[next];
    if (targetId) focusAndScroll(targetId);
  }, [navIds, focusedNavIndex, focusAndScroll]);

  // --- Current message position for jump trigger ---
  const focusedEventIndex = focusedEventId ? (eventIndexById.get(focusedEventId) ?? -1) : -1;
  const jumpLabel = focusedEventIndex >= 0
    ? `${focusedEventIndex + 1} / ${totalCount}`
    : `${events.length}`;

  const toggleExpanded = useCallback((eventId: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(eventId)) next.delete(eventId);
      else next.add(eventId);
      return next;
    });
  }, []);

  useEffect(() => {
    if (!playbackFocusRequest) return;
    focusAndScroll(playbackFocusRequest.eventId);
  }, [focusAndScroll, playbackFocusRequest]);

  const { openPreview } = filePreview;
  const handleMarkdownLinkClick = useCallback((href: string, event: React.MouseEvent<HTMLAnchorElement>) => {
    if (!looksLikeFileReference(href)) return;
    event.preventDefault();
    void openPreview(href, "message_link");
  }, [openPreview]);

  const handleInlineCodeFileClick = useCallback((path: string) => {
    void openPreview(path, "inline_code");
  }, [openPreview]);

  const [mermaidViewer, setMermaidViewer] = useState<{ code: string } | null>(null);
  const handleMermaidOpen = useCallback((code: string) => {
    setMermaidViewer({ code });
  }, []);
  const closeMermaidViewer = useCallback(() => { setMermaidViewer(null); }, []);

  return (
    <div
      data-testid={`messages-pane-${sessionId}`}
      aria-readonly={readOnly}
      className="relative flex h-full flex-col bg-wc-surface-base px-2 pb-4 pt-1 select-text"
    >
      <div
        data-testid="messages-control-strip"
        className="z-wc-chrome flex items-center justify-start gap-1.5 bg-wc-surface-base/80 py-1.5 backdrop-blur-sm"
      >
        <IconButton
          data-testid="messages-search-btn"
          onClick={() => { openNavigator("search"); }}
          selected={!!searchQuery}
          surface="soft"
          size="xs"
          denseTapTarget
          className={cn(searchQuery && "ring-1 ring-wc-accent/50")}
          aria-label={t(strings.messagesPane.searchMessagesTitle)}
        >
          <Search />
        </IconButton>

        <button
          data-testid="msg-jump-trigger"
          onClick={() => { navOpen ? setNavOpen(false) : openNavigator("list"); }}
          disabled={events.length === 0}
          className="flex h-8 items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2.5 text-xs text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-30 disabled:pointer-events-none"
          title={t(strings.messagesPane.jumpToMessageTitle)}
          type="button"
        >
          <ChevronsUpDown className="h-3.5 w-3.5" />
          <span className="font-mono">{jumpLabel}</span>
        </button>

        <IconButton
          data-testid="messages-nav-up"
          onClick={handleNavUp}
          disabled={navIds.length === 0}
          surface="soft"
          size="xs"
          denseTapTarget
          aria-label={searchQuery ? t(strings.messagesPane.prevMatchTitle) : t(strings.messagesPane.prevMessageTitle)}
        >
          <ChevronUp />
        </IconButton>
        <IconButton
          data-testid="messages-nav-down"
          onClick={handleNavDown}
          disabled={navIds.length === 0}
          surface="soft"
          size="xs"
          denseTapTarget
          aria-label={searchQuery ? t(strings.messagesPane.nextMatchTitle) : t(strings.messagesPane.nextMessageTitle)}
        >
          <ChevronDown />
        </IconButton>
        <IconButton
          data-testid="messages-refresh-btn"
          onClick={handleRefresh}
          // The control owns the busy affordance, so the spin is no longer a
          // class the call site has to remember to add and remove.
          pending={isRefreshing}
          pendingLabel={t(strings.messagesPane.refreshTitle)}
          surface="soft"
          size="xs"
          denseTapTarget
          aria-label={t(strings.messagesPane.refreshTitle)}
        >
          <RotateCw />
        </IconButton>
        {toolbarTrailingAction && (
          <div data-testid="messages-control-trailing" className="ms-auto flex items-center">
            {toolbarTrailingAction}
          </div>
        )}
      </div>

      {navOpen && (
        <MessageJumpList
          events={events}
          focusedEventId={focusedEventId}
          onSelect={focusAndScroll}
          onClose={() => { setNavOpen(false); }}
          mode="jump"
          initialFocus={navInitialFocus}
          query={searchQuery}
          onQueryChange={handleNavQueryChange}
          searchMatchCount={searchQuery ? serverSearchMatches.length : undefined}
          searchTruncated={searchTruncated}
          exportSelection={exportSelection}
        />
      )}

      <MessageExportDrawer
        open={exportDrawerOpen}
        events={exportEvents}
        onClose={() => { setExportDrawerOpen(false); }}
      />


      <MessagesPaneStatusLine
        status={resolveMessagesPaneStatus({
          refreshError,
          liveInterrupted,
          liveInterruptedText: t(strings.messagesPane.liveDisconnected),
          transient: transientNotice,
        })}
      />

      <div ref={scrollContainerRef} className="relative min-h-0 flex-1 overflow-auto">
        {viewState.kind !== "messages" ? (
          <MessagesPaneState view={viewState} onRetry={() => void handleRefresh()} />
        ) : (
          <div className="relative" style={{ height: `${totalSize}px` }}>
            {virtualItems.map(({ index, start }) => {
              const event = events[index];
              if (!event) return null;
              return (
                <div key={event.id} data-event-id={event.id} className="absolute left-0 right-0" style={{ top: `${start}px` }}>
                  <MessageRow
                    event={event}
                    index={index}
                    registerItem={registerItem}
                    fontSize={fontSize}
                    copiedEventId={copiedEventId}
                    onCopy={handleCopy}
                    onPlayFromHere={onPlayFromHere}
                    onPlayEvent={onPlayEvent}
                    isTtsSpeaking={isTtsSpeaking}
                    activeSpeakingEventId={activeSpeakingEventId}
                    loadingEventId={loadingEventId}
                    summarizeLevel={summarizeLevel}
                    selectedVersionForEvent={selectedVersionForEvent}
                    summarizingEventId={summarizingEventId}
                    getSummarizeError={getSummarizeError}
                    onClearSummarizeError={onClearSummarizeError}
                    onToggleSummarized={onToggleSummarized}
                    onChangeLevel={onChangeLevel}
                    audioSettings={audioSettings}
                    onSetPlaybackRate={onSetPlaybackRate}
                    onSetVolume={onSetVolume}
                    onSetMuted={onSetMuted}
                    isMobile={isMobile}
                    isFocused={focusedEventId === event.id}
                    isSearchFocused={searchMatchSet.has(event.id) && currentMatchIndex >= 0 && searchMatchIds[currentMatchIndex] === event.id}
                    isDimmed={!!searchQuery && !searchMatchSet.has(event.id)}
                    isExpanded={expandedIds.has(event.id)}
                    onToggleExpanded={toggleExpanded}
                    isPlaintext={plaintextIds.has(event.id)}
                    onToggleRenderMode={toggleRenderMode}
                    onLinkClick={handleMarkdownLinkClick}
                    onFileReferenceClick={handleInlineCodeFileClick}
                    onMermaidOpen={handleMermaidOpen}
                    readOnly={readOnly}
                    onSendToComposer={onSendToComposer}
                  />
                  {/* Suggestions render INSIDE the message's own block, so
                      offering one never moves the transcript the operator is
                      reading. */}
                  {onHandoff && handoffSuggestions.forEvent(event.id).map((suggestion) => (
                    <HandoffSuggestionChip
                      key={`${suggestion.ruleId}:${suggestion.payload}`}
                      suggestion={suggestion}
                      onOpen={(s) => { onHandoff(sessionId, s.payload); }}
                      onDismiss={handoffSuggestions.dismiss}
                    />
                  ))}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {newMessageCount > 0 && (
        <button
          data-testid="msg-new-pill"
          onClick={scrollToBottom}
          className="absolute bottom-[max(1rem,var(--wc-safe-bottom,0px))] left-1/2 z-wc-chrome-raised -translate-x-1/2 rounded-full border border-wc-default bg-wc-surface-raised px-4 py-2 text-xs font-medium text-wc-text-primary shadow-lg backdrop-blur-sm transition-all hover:bg-wc-surface-input"
          type="button"
        >
          <ArrowDown className="me-1.5 inline-block h-3.5 w-3.5" />
          {t(strings.messagesPane.newMessages, { count: newMessageCount })}
        </button>
      )}

      {newMessageCount === 0 && !isNearBottom && events.length > 0 && (
        <IconButton
          data-testid="msg-jump-bottom"
          aria-label={t(strings.messagesPane.jumpToBottomAria)}
          onClick={scrollToBottom}
          surface="soft"
          className="absolute bottom-[max(1rem,var(--wc-safe-bottom,0px))] left-1/2 z-wc-chrome-raised -translate-x-1/2 shadow-lg"
        >
          <ArrowDown />
        </IconButton>
      )}

      <MessagesFileViewer
        state={filePreview.state}
        onHandoff={onHandoff ? (path) => { onHandoff(sessionId, path); } : undefined}
        onClose={filePreview.close}
        onReopen={filePreview.reopen}
        onRendererError={filePreview.reportError}
        onNavigate={filePreview.navigateTo}
        onNavigateBack={filePreview.navigateBack}
        onLoadMore={filePreview.loadMore}
        onListOptionsChange={filePreview.setListOptions}
      />

      <MessagesMermaidViewer
        open={mermaidViewer !== null}
        code={mermaidViewer?.code ?? ""}
        onClose={closeMermaidViewer}
      />
    </div>
  );
}
