import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import {
  ArrowDown,
  Check,
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Copy,
  Play,
  RotateCw,
  Search,
  Volume2,
} from "lucide-react";
import { useConversationStore, getSessionConversationEvents } from "../stores/useConversationStore";
import { refreshConversationSession } from "../hooks/useConversationSession";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { APIError } from "../lib/errors";
import {
  getFileReferenceContent,
  resolveFileReference,
  type ConversationEvent,
  type FileReferenceContentResponse,
  type FileReferenceResolveResponse,
} from "../api/conversation";
import { TERMINAL_FONT_SIZE } from "../consts/config";
import { cn } from "../lib/classnames";
import { looksLikeFileReference } from "../lib/fileReferences";
import { MarkdownRenderer } from "./markdown";
import { useVirtualList } from "../hooks/useVirtualList";
import MessagesSearchDrawer from "./MessagesSearchDrawer";
import MessageJumpList from "./MessageJumpList";
import { AudioSettingsContent } from "./tts/AudioSettingsContent";
import { PlaybackModeControl, type SummarizationLevel } from "./tts/PlaybackModeControl";
import type { TTSPlaybackState } from "../hooks/tts/types";
import type { PlaybackFocusRequest, PlaybackVersion } from "../domains/tts-playback/types";
import MessagesFileViewer from "./MessagesFileViewer";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MessagesPaneProps {
  sessionId: string;
  onPlayFromHere: (eventId: string) => void;
  onPlayEvent: (eventId: string) => void;
  activeSpeakingEventId: string | null;
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
  isMobile: boolean;
  isFocused: boolean;
  isSearchFocused: boolean;
  isDimmed: boolean;
  isExpanded: boolean;
  onToggleExpanded: (eventId: string) => void;
  onLinkClick: (href: string, event: React.MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick: (path: string) => void;
}

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
  isMobile,
  isFocused,
  isSearchFocused,
  isDimmed,
  isExpanded,
  onToggleExpanded,
  onLinkClick,
  onFileReferenceClick,
}: MessageRowProps) {
  const { t } = useTranslation();
  const [openPopoverId, setOpenPopoverId] = useState<string | null>(null);
  const [isTall, setIsTall] = useState(false);
  const audioButtonRef = useRef<HTMLButtonElement | null>(null);
  const contentRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const node = contentRef.current;
    if (!node) return;

    const measure = () => setIsTall(node.scrollHeight > COLLAPSE_THRESHOLD_PX);
    measure();

    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(() => measure());
    observer.observe(node);
    return () => observer.disconnect();
  }, [event.text, isExpanded]);

  const isUser = event.role === "user";
  const isTtsActive = !isUser && isTtsSpeaking && activeSpeakingEventId === event.id;
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

  const getPopoverStyle = (): React.CSSProperties => {
    const btn = audioButtonRef.current;
    if (!btn) return { position: "fixed", top: 100, right: 16 };
    const rect = btn.getBoundingClientRect();
    const top = Math.min(rect.bottom + 4, window.innerHeight - 200);
    const right = Math.max(8, window.innerWidth - rect.right);
    return { position: "fixed", top, right };
  };

  return (
    <article
      ref={(node) => registerItem(index, node)}
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
          onClick={() => onCopy(event.id, event.text)}
          className="rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10"
          title={t(strings.messagesPane.copyMessageTitle)}
          type="button"
        >
          {copiedEventId === event.id
            ? <Check className="h-3.5 w-3.5 text-green-400" />
            : <Copy className="h-3.5 w-3.5" />}
        </button>

        {!isUser && (
          <>
            <button
              data-testid={`msg-speak-from-${event.id}`}
              onClick={() => onPlayFromHere(event.id)}
              className="rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10"
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
              onToggleSummarized={(use) => onToggleSummarized(event.id, use)}
              onChangeLevel={(level) => onChangeLevel(event.id, level)}
            />
            <button
              ref={audioButtonRef}
              data-testid={`msg-audio-${event.id}`}
              onClick={() => {
                onPlayEvent(event.id);
                setOpenPopoverId(isPopoverOpen ? null : event.id);
              }}
              className="rounded p-0.5 text-wc-text-faint transition hover:text-wc-text-muted hover:bg-wc-accent/10"
              title={t(strings.messagesPane.playAudioSettingsTitle)}
              type="button"
            >
              <Volume2 className="h-3 w-3" />
            </button>

            {isPopoverOpen && createPortal(
              isMobile ? (
                <div className="fixed inset-0 z-[60]" onMouseDown={(e) => e.preventDefault()}>
                  <div className="absolute inset-0 bg-wc-backdrop" onClick={() => setOpenPopoverId(null)} />
                  <div
                    data-testid={`audio-popover-${event.id}`}
                    className="absolute bottom-0 left-0 right-0 z-[61] rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-4 pb-[max(1rem,var(--wc-safe-bottom))] ps-[max(1rem,var(--wc-safe-left,0px))] pe-[max(1rem,var(--wc-safe-right,0px))] shadow-2xl"
                  >
                    <div className="mb-3 flex justify-center">
                      <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
                    </div>
                    <h3 className="mb-3 text-sm font-semibold text-wc-text-primary">{t(strings.messagesPane.audioSettingsHeading)}</h3>
                    <AudioSettingsContent
                      testIdPrefix={`msg-${event.id}`}
                      volume={playbackState.volume}
                      isMuted={playbackState.isMuted}
                      playbackRate={playbackState.playbackRate}
                      isSummarized={useSummarized && hasSummary}
                      capabilities={playbackState.capabilities}
                      onVolumeChange={onSetVolume}
                      onSetMuted={onSetMuted}
                      onSetPlaybackRate={onSetPlaybackRate}
                    />
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
                        onClick={() => onClearSummarizeError(event.id)}
                      >
                        {t(strings.messagesPane.dismissError)}
                      </button>
                    )}
                  </div>
                </div>
              ) : (
                <>
                  <div className="fixed inset-0 z-[60]" onClick={() => setOpenPopoverId(null)} />
                  <div
                    data-testid={`audio-popover-${event.id}`}
                    className="z-[61] w-56 rounded-xl border border-wc-default bg-wc-surface-raised p-3 shadow-lg"
                    style={getPopoverStyle()}
                  >
                    <AudioSettingsContent
                      testIdPrefix={`msg-${event.id}`}
                      volume={playbackState.volume}
                      isMuted={playbackState.isMuted}
                      playbackRate={playbackState.playbackRate}
                      isSummarized={useSummarized && hasSummary}
                      capabilities={playbackState.capabilities}
                      onVolumeChange={onSetVolume}
                      onSetMuted={onSetMuted}
                      onSetPlaybackRate={onSetPlaybackRate}
                    />
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
                        onClick={() => onClearSummarizeError(event.id)}
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
          <MarkdownRenderer content={event.text} onLinkClick={onLinkClick} onFileReferenceClick={onFileReferenceClick} />
        </div>

        {isCollapsed && (
          <div className="absolute bottom-0 left-0 right-0 h-20 bg-gradient-to-t from-wc-surface-base to-transparent pointer-events-none" />
        )}
      </div>

      {isTall && (
        <button
          data-testid={`msg-collapse-${event.id}`}
          onClick={() => onToggleExpanded(event.id)}
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
  prevProps.summarizeLevel === nextProps.summarizeLevel &&
  prevProps.summarizingEventId === nextProps.summarizingEventId &&
  prevProps.playbackState === nextProps.playbackState &&
  prevProps.isMobile === nextProps.isMobile &&
  prevProps.isFocused === nextProps.isFocused &&
  prevProps.isSearchFocused === nextProps.isSearchFocused &&
  prevProps.isDimmed === nextProps.isDimmed &&
  prevProps.isExpanded === nextProps.isExpanded &&
  prevProps.onLinkClick === nextProps.onLinkClick &&
  prevProps.onFileReferenceClick === nextProps.onFileReferenceClick
));

export default function MessagesPane({
  sessionId,
  onPlayFromHere,
  onPlayEvent,
  activeSpeakingEventId,
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
}: MessagesPaneProps) {
  const { t } = useTranslation();
  const events = useConversationStore((state) => getSessionConversationEvents(state, sessionId));
  const isMobile = useMediaQuery("(max-width: 767px)");
  const fontSize = useWorkspaceStore(
    useCallback((s) => s.panes.find((p) => p.sessionId === sessionId)?.fontSize ?? TERMINAL_FONT_SIZE, [sessionId]),
  );

  // --- Copy ---
  const [copiedEventId, setCopiedEventId] = useState<string | null>(null);
  const handleCopy = useCallback((eventId: string, text: string) => {
    void navigator.clipboard.writeText(text);
    setCopiedEventId(eventId);
    setTimeout(() => setCopiedEventId((prev) => (prev === eventId ? null : prev)), 2000);
  }, []);

  // --- Search & navigation ---
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedEventId, setFocusedEventId] = useState<string | null>(null);

  // --- Jump list ---
  const [jumpListOpen, setJumpListOpen] = useState(false);

  // --- Collapse ---
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [fileViewerOpen, setFileViewerOpen] = useState(false);
  const [fileViewerLoading, setFileViewerLoading] = useState(false);
  const [fileViewerError, setFileViewerError] = useState<string | null>(null);
  const [requestedFilePath, setRequestedFilePath] = useState<string | null>(null);
  const [resolvedFile, setResolvedFile] = useState<FileReferenceResolveResponse | null>(null);
  const [fileContent, setFileContent] = useState<FileReferenceContentResponse | null>(null);

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
  const restoreAppliedRef = useRef(false);

  // --- Refresh: on mount, on browser tab focus, and via manual button ---
  const [isRefreshing, setIsRefreshing] = useState(false);
  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    try {
      await refreshConversationSession(sessionId);
    } finally {
      setIsRefreshing(false);
    }
  }, [sessionId]);

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

    const updateNearBottom = () => {
      const remaining = el.scrollHeight - (el.scrollTop + el.clientHeight);
      const nearBottom = remaining <= 200;
      isNearBottomRef.current = nearBottom;
      setIsNearBottom(nearBottom);
      if (nearBottom) setNewMessageCount(0);
    };

    updateNearBottom();
    el.addEventListener("scroll", updateNearBottom, { passive: true });
    return () => el.removeEventListener("scroll", updateNearBottom);
  }, []);

  // Release the bottom-pin or event-pin as soon as the user scrolls away from
  // it. We do this from a wheel/touchstart listener so synthetic re-scrolls
  // from the totalSize-change effect don't release the pin.
  useEffect(() => {
    const el = scrollContainerRef.current;
    if (!el) return;
    const release = () => {
      pinToBottomRef.current = false;
      pinToEventIdRef.current = null;
    };
    el.addEventListener("wheel", release, { passive: true });
    el.addEventListener("touchstart", release, { passive: true });
    el.addEventListener("keydown", release);
    return () => {
      el.removeEventListener("wheel", release);
      el.removeEventListener("touchstart", release);
      el.removeEventListener("keydown", release);
    };
  }, []);

  // Auto-scroll on new events (when near bottom) or show pill
  useEffect(() => {
    const newCount = events.length - prevEventCountRef.current;
    prevEventCountRef.current = events.length;

    if (newCount <= 0) return;

    if (isNearBottomRef.current) {
      requestAnimationFrame(() => {
        scrollContainerRef.current?.scrollTo({
          top: scrollContainerRef.current.scrollHeight,
          behavior: "smooth",
        });
      });
    } else {
      setNewMessageCount((prev) => prev + newCount);
    }
  }, [events.length]);

  const scrollToBottom = useCallback(() => {
    pinToBottomRef.current = true;
    pinToEventIdRef.current = null;
    scrollContainerRef.current?.scrollTo({
      top: scrollContainerRef.current.scrollHeight,
      behavior: "smooth",
    });
    setNewMessageCount(0);
  }, []);

  // Search match IDs
  const searchMatchIds = useMemo(() => {
    if (!searchQuery) return [];
    const q = searchQuery.toLowerCase();
    return events.filter((e) => e.text.toLowerCase().includes(q)).map((e) => e.id);
  }, [events, searchQuery]);
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
  const { registerItem, totalSize, virtualItems, scrollToIndex } = useVirtualList({
    count: events.length,
    estimateSize: estimateMessageHeight,
    overscan: 8,
    scrollElementRef: scrollContainerRef,
    enabled: events.length > 40,
  });

  const scrollToEvent = useCallback((eventId: string) => {
    const index = eventIndexById.get(eventId);
    if (index == null) return;
    scrollToIndex(index, "smooth", "center");
  }, [eventIndexById, scrollToIndex]);

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
      el.scrollTo({ top: el.scrollHeight });
    } else if (pinToEventIdRef.current) {
      const index = eventIndexById.get(pinToEventIdRef.current);
      if (index != null) scrollToIndex(index, "auto", "start");
    }
  }, [events.length, sessionId, eventIndexById, scrollToIndex]);

  // Re-apply the active pin whenever the virtualizer's totalSize changes.
  // This is what fixes the "lands in the middle" bug: estimated sizes are
  // smaller than actual, so the initial scrollTo lands too high; once rows
  // measure their real heights totalSize grows and we re-scroll.
  useEffect(() => {
    const el = scrollContainerRef.current;
    if (!el) return;
    if (pinToBottomRef.current) {
      el.scrollTo({ top: el.scrollHeight });
    } else if (pinToEventIdRef.current) {
      const index = eventIndexById.get(pinToEventIdRef.current);
      if (index != null) scrollToIndex(index, "auto", "start");
    }
  }, [totalSize, eventIndexById, scrollToIndex]);

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

  const focusAndScroll = useCallback((eventId: string) => {
    setFocusedEventId(eventId);
    scrollToEvent(eventId);
    // Auto-expand if collapsed and a search match
    if (searchQuery) {
      setExpandedIds((prev) => new Set(prev).add(eventId));
    }
  }, [scrollToEvent, searchQuery]);

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

  const handleCloseSearch = useCallback(() => {
    setSearchOpen(false);
    setSearchQuery("");
    setFocusedEventId(null);
  }, []);

  // --- Current message position for jump trigger ---
  const focusedEventIndex = focusedEventId ? (eventIndexById.get(focusedEventId) ?? -1) : -1;
  const jumpLabel = focusedEventIndex >= 0
    ? `${focusedEventIndex + 1} / ${events.length}`
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

  const closeFileViewer = useCallback(() => {
    setFileViewerOpen(false);
    setFileViewerLoading(false);
    setFileViewerError(null);
    setRequestedFilePath(null);
    setResolvedFile(null);
    setFileContent(null);
  }, []);

  const openFileReference = useCallback(async (href: string) => {
    setRequestedFilePath(href);
    setResolvedFile(null);
    setFileContent(null);
    setFileViewerError(null);
    setFileViewerOpen(true);
    setFileViewerLoading(true);

    try {
      const resolved = await resolveFileReference(sessionId, href);
      setResolvedFile(resolved);
      if (!resolved.can_preview) {
        setFileViewerError(t(strings.messagesPane.fileNotPreviewable));
        return;
      }
      const contentPath = resolved.line ? `${resolved.resolved_path}:${resolved.line}` : resolved.resolved_path;
      const content = await getFileReferenceContent(sessionId, contentPath);
      setFileContent(content);
    } catch (err) {
      if (err instanceof APIError) {
        setFileViewerError(err.message);
      } else if (err instanceof Error) {
        setFileViewerError(err.message);
      } else {
        setFileViewerError(t(strings.messagesPane.fileOpenFailed));
      }
    } finally {
      setFileViewerLoading(false);
    }
  }, [sessionId, t]);

  const handleMarkdownLinkClick = useCallback((href: string, event: React.MouseEvent<HTMLAnchorElement>) => {
    if (!looksLikeFileReference(href)) return;
    event.preventDefault();
    void openFileReference(href);
  }, [openFileReference]);

  const handleInlineCodeFileClick = useCallback((path: string) => {
    void openFileReference(path);
  }, [openFileReference]);

  return (
    <div
      data-testid={`messages-pane-${sessionId}`}
      className="relative flex h-full flex-col bg-wc-surface-base px-2 pb-4 pt-1 select-text"
    >
      <div
        data-testid="messages-control-strip"
        className="z-10 flex items-center justify-start gap-1.5 bg-wc-surface-base/80 py-1.5 backdrop-blur-sm"
      >
        <button
          data-testid="messages-search-btn"
          onClick={() => setSearchOpen(true)}
          className="flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm"
          title={t(strings.messagesPane.searchMessagesTitle)}
          type="button"
        >
          <Search className="h-3.5 w-3.5" />
        </button>

        <button
          data-testid="msg-jump-trigger"
          onClick={() => setJumpListOpen((v) => !v)}
          disabled={events.length === 0}
          className="flex h-8 items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2.5 text-xs text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-30 disabled:pointer-events-none"
          title={t(strings.messagesPane.jumpToMessageTitle)}
          type="button"
        >
          <ChevronsUpDown className="h-3.5 w-3.5" />
          <span className="font-mono">{jumpLabel}</span>
        </button>

        <button
          data-testid="messages-nav-up"
          onClick={handleNavUp}
          disabled={navIds.length === 0}
          className="flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-30 disabled:pointer-events-none"
          title={searchQuery ? t(strings.messagesPane.prevMatchTitle) : t(strings.messagesPane.prevMessageTitle)}
          type="button"
        >
          <ChevronUp className="h-3.5 w-3.5" />
        </button>
        <button
          data-testid="messages-nav-down"
          onClick={handleNavDown}
          disabled={navIds.length === 0}
          className="flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-30 disabled:pointer-events-none"
          title={searchQuery ? t(strings.messagesPane.nextMatchTitle) : t(strings.messagesPane.nextMessageTitle)}
          type="button"
        >
          <ChevronDown className="h-3.5 w-3.5" />
        </button>
        <button
          data-testid="messages-refresh-btn"
          onClick={handleRefresh}
          disabled={isRefreshing}
          className="flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-60 disabled:pointer-events-none"
          title={t(strings.messagesPane.refreshTitle)}
          type="button"
        >
          <RotateCw className={cn("h-3.5 w-3.5", isRefreshing && "animate-spin")} />
        </button>
      </div>

      <MessagesSearchDrawer
        open={searchOpen}
        onClose={handleCloseSearch}
        query={searchQuery}
        onQueryChange={(q) => {
          setSearchQuery(q);
          setFocusedEventId(null);
        }}
        matchCount={searchMatchIds.length}
        currentMatchIndex={currentMatchIndex}
        onPrevMatch={handleNavUp}
        onNextMatch={handleNavDown}
      />

      {jumpListOpen && (
        <MessageJumpList
          events={events}
          focusedEventId={focusedEventId}
          onSelect={focusAndScroll}
          onClose={() => setJumpListOpen(false)}
        />
      )}

      <div ref={scrollContainerRef} className="relative min-h-0 flex-1 overflow-auto">
        {events.length === 0 ? (
          <div className="rounded-xl border border-dashed border-wc-default bg-wc-surface px-4 py-6 text-sm text-wc-text-muted">
            {t(strings.messagesPane.noEvents)}
          </div>
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
                    summarizeLevel={summarizeLevel}
                    selectedVersionForEvent={selectedVersionForEvent}
                    summarizingEventId={summarizingEventId}
                    getSummarizeError={getSummarizeError}
                    onClearSummarizeError={onClearSummarizeError}
                    onToggleSummarized={onToggleSummarized}
                    onChangeLevel={onChangeLevel}
                    playbackState={playbackState}
                    onSetPlaybackRate={onSetPlaybackRate}
                    onSetVolume={onSetVolume}
                    onSetMuted={onSetMuted}
                    isMobile={isMobile}
                    isFocused={focusedEventId === event.id}
                    isSearchFocused={searchMatchSet.has(event.id) && currentMatchIndex >= 0 && searchMatchIds[currentMatchIndex] === event.id}
                    isDimmed={!!searchQuery && !searchMatchSet.has(event.id)}
                    isExpanded={expandedIds.has(event.id)}
                    onToggleExpanded={toggleExpanded}
                    onLinkClick={handleMarkdownLinkClick}
                    onFileReferenceClick={handleInlineCodeFileClick}
                  />
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
          className="absolute bottom-[max(1rem,var(--wc-safe-bottom,0px))] left-1/2 z-20 -translate-x-1/2 rounded-full border border-wc-default bg-wc-surface-raised px-4 py-2 text-xs font-medium text-wc-text-primary shadow-lg backdrop-blur-sm transition-all hover:bg-wc-surface-input"
          type="button"
        >
          <ArrowDown className="me-1.5 inline-block h-3.5 w-3.5" />
          {t(strings.messagesPane.newMessages, { count: newMessageCount })}
        </button>
      )}

      {newMessageCount === 0 && !isNearBottom && events.length > 0 && (
        <button
          data-testid="msg-jump-bottom"
          aria-label={t(strings.messagesPane.jumpToBottomAria)}
          onClick={scrollToBottom}
          className="absolute bottom-[max(1rem,var(--wc-safe-bottom,0px))] left-1/2 z-20 -translate-x-1/2 rounded-full border border-wc-default bg-wc-surface-raised/60 p-2 text-wc-text-secondary shadow-lg backdrop-blur-sm transition-all hover:bg-wc-surface-input hover:text-wc-text-primary"
          type="button"
        >
          <ArrowDown className="h-4 w-4" />
        </button>
      )}

      <MessagesFileViewer
        open={fileViewerOpen}
        loading={fileViewerLoading}
        error={fileViewerError}
        requestedPath={requestedFilePath}
        resolved={resolvedFile}
        content={fileContent}
        onClose={closeFileViewer}
      />
    </div>
  );
}
