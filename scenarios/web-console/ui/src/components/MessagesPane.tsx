import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
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
import { summarizeEvent, updateTTSSummarizeConfig } from "../lib/api";
import { TERMINAL_FONT_SIZE } from "../consts/config";
import { cn } from "../lib/classnames";
import { MarkdownRenderer } from "./markdown";
import MessagesSearchDrawer from "./MessagesSearchDrawer";
import MessageJumpList from "./MessageJumpList";
import { AudioSettingsContent } from "./tts/AudioSettingsContent";
import { PlaybackModeControl, type SummarizationLevel } from "./tts/PlaybackModeControl";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MessagesPaneProps {
  sessionId: string;
  onSpeakFromHere: (eventId: string) => void;
  onSpeakOne: (eventId: string, text: string, paragraphs?: string[], opts?: { version?: "active" | "original" }) => void;
  activeSpeakingEventId: string | null;
  isTtsSpeaking: boolean;
  /** Current global summarization level (from TTSSummarizeConfig). */
  summarizeLevel: SummarizationLevel;
  /** Called after the level is changed globally so parent can update its cache. */
  onSummarizeLevelChanged: (level: SummarizationLevel) => void;
  /** Called when an on-demand summarize attempt fails. Parent surfaces a persistent banner. */
  onSummarizeFailed?: (eventId: string, message: string) => void;
}

// ---------------------------------------------------------------------------
// Collapse threshold (px of rendered content before collapsing)
// ---------------------------------------------------------------------------
const COLLAPSE_THRESHOLD_PX = 400;

// ---------------------------------------------------------------------------
// MessagesPane
// ---------------------------------------------------------------------------

export default function MessagesPane({
  sessionId,
  onSpeakFromHere,
  onSpeakOne,
  activeSpeakingEventId,
  isTtsSpeaking,
  summarizeLevel,
  onSummarizeLevelChanged,
  onSummarizeFailed,
}: MessagesPaneProps) {
  const events = useConversationStore((state) => getSessionConversationEvents(state, sessionId));
  const isMobile = useMediaQuery("(max-width: 767px)");
  const fontSize = useWorkspaceStore(
    useCallback((s) => s.panes.find((p) => p.sessionId === sessionId)?.fontSize ?? TERMINAL_FONT_SIZE, [sessionId]),
  );

  // --- Audio popover state ---
  const [openPopoverId, setOpenPopoverId] = useState<string | null>(null);
  const [volume, setVolume] = useState(1);
  const [playbackModes, setPlaybackModes] = useState<Record<string, boolean>>({});
  const [summarizingIds, setSummarizingIds] = useState<Set<string>>(new Set());
  const [summarizeErrors, setSummarizeErrors] = useState<Record<string, string>>({});
  const summarizeAbortControllers = useRef<Record<string, AbortController>>({});
  const audioButtonRefs = useRef<Record<string, HTMLButtonElement | null>>({});

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
  const messageRefs = useRef<Map<string, HTMLElement>>(new Map());

  // --- Jump list ---
  const [jumpListOpen, setJumpListOpen] = useState(false);

  // --- Collapse ---
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [tallIds, setTallIds] = useState<Set<string>>(new Set());
  const contentRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  // --- Auto-scroll ---
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const isNearBottomRef = useRef(true);
  const [newMessageCount, setNewMessageCount] = useState(0);
  const prevEventCountRef = useRef(events.length);

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

  // Track "near bottom" via IntersectionObserver on the sentinel div
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry) {
          isNearBottomRef.current = entry.isIntersecting;
          if (entry.isIntersecting) setNewMessageCount(0);
        }
      },
      { threshold: 0 },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, []);

  // Auto-scroll to bottom on initial load
  useEffect(() => {
    if (events.length > 0) {
      scrollContainerRef.current?.scrollTo({ top: scrollContainerRef.current.scrollHeight });
    }
    // Only run on first hydration
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [events.length > 0]);

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
    scrollContainerRef.current?.scrollTo({
      top: scrollContainerRef.current.scrollHeight,
      behavior: "smooth",
    });
    setNewMessageCount(0);
  }, []);

  // --- Collapse: measure content heights ---
  useEffect(() => {
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const eventId = (entry.target as HTMLElement).dataset.eventId;
        if (!eventId) continue;
        const isTall = entry.contentRect.height > COLLAPSE_THRESHOLD_PX;
        setTallIds((prev) => {
          const next = new Set(prev);
          if (isTall) next.add(eventId);
          else next.delete(eventId);
          return next;
        });
      }
    });

    for (const [, el] of contentRefs.current) {
      observer.observe(el);
    }

    return () => observer.disconnect();
  }, [events.length]); // Re-observe when events change

  // Search match IDs
  const searchMatchIds = useMemo(() => {
    if (!searchQuery) return [];
    const q = searchQuery.toLowerCase();
    return events.filter((e) => e.text.toLowerCase().includes(q)).map((e) => e.id);
  }, [events, searchQuery]);

  const navIds = useMemo(
    () => (searchQuery ? searchMatchIds : events.map((e) => e.id)),
    [searchQuery, searchMatchIds, events],
  );

  const focusedNavIndex = useMemo(
    () => (focusedEventId ? navIds.indexOf(focusedEventId) : -1),
    [focusedEventId, navIds],
  );

  const currentMatchIndex = useMemo(
    () => (focusedEventId ? searchMatchIds.indexOf(focusedEventId) : -1),
    [focusedEventId, searchMatchIds],
  );

  const scrollToEvent = useCallback((eventId: string) => {
    messageRefs.current.get(eventId)?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, []);

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

  // --- Audio helpers ---
  const handleToggleSummarized = useCallback((eventId: string, useSummarized: boolean) => {
    setPlaybackModes((prev) => ({ ...prev, [eventId]: useSummarized }));
  }, []);

  const handleSpeakOne = useCallback((event: typeof events[number]) => {
    const useSummarized = playbackModes[event.id] ?? event.summarized;
    const paragraphs = useSummarized
      ? event.speechParagraphs
      : (event.originalSpeechParagraphs ?? event.speechParagraphs);
    onSpeakOne(event.id, event.text, paragraphs, { version: useSummarized ? "active" : "original" });
  }, [onSpeakOne, playbackModes]);

  const handleRequestSummarize = useCallback((eventId: string) => {
    // Abort any in-flight request for this event
    summarizeAbortControllers.current[eventId]?.abort();
    const controller = new AbortController();
    summarizeAbortControllers.current[eventId] = controller;

    setSummarizingIds((prev) => new Set(prev).add(eventId));
    setSummarizeErrors((prev) => {
      const next = { ...prev };
      delete next[eventId];
      return next;
    });
    void summarizeEvent(sessionId, eventId, controller.signal).then((res) => {
      if (res.summarized && res.speechParagraphs) {
        const convState = useConversationStore.getState();
        const session = convState.sessions[sessionId];
        if (session) {
          const updatedEvents = session.events.map((ev) =>
            ev.id === eventId
              ? { ...ev, summarized: true, originalSpeechParagraphs: ev.speechParagraphs, speechParagraphs: res.speechParagraphs ?? ev.speechParagraphs }
              : ev,
          );
          useConversationStore.setState({
            sessions: { ...convState.sessions, [sessionId]: { ...session, events: updatedEvents } },
          });
        }
      } else if (res.error) {
        const msg = res.error ?? "Unknown error";
        setSummarizeErrors((prev) => ({ ...prev, [eventId]: msg }));
        onSummarizeFailed?.(eventId, msg);
      }
    }).catch((err: unknown) => {
      if (err instanceof DOMException && err.name === "AbortError") return;
      const message = err instanceof Error ? err.message : "Summarization failed";
      setSummarizeErrors((prev) => ({ ...prev, [eventId]: message }));
      onSummarizeFailed?.(eventId, message);
    }).finally(() => {
      delete summarizeAbortControllers.current[eventId];
      setSummarizingIds((prev) => {
        const next = new Set(prev);
        next.delete(eventId);
        return next;
      });
    });
  }, [sessionId, onSummarizeFailed]);

  const handleCancelSummarize = useCallback((eventId: string) => {
    summarizeAbortControllers.current[eventId]?.abort();
    delete summarizeAbortControllers.current[eventId];
    setSummarizingIds((prev) => {
      const next = new Set(prev);
      next.delete(eventId);
      return next;
    });
  }, []);

  const handleChangeLevelForEvent = useCallback((eventId: string, level: SummarizationLevel) => {
    const apply = level !== summarizeLevel
      ? updateTTSSummarizeConfig({ level }).then((cfg) => {
          onSummarizeLevelChanged(cfg.level);
        }).catch(() => {
          // If the global update fails, still try to summarize with existing level.
        })
      : Promise.resolve();
    void apply.then(() => {
      handleRequestSummarize(eventId);
    });
  }, [summarizeLevel, onSummarizeLevelChanged, handleRequestSummarize]);

  // Abort all in-flight summarizations on unmount
  useEffect(() => {
    const controllers = summarizeAbortControllers.current;
    return () => {
      for (const controller of Object.values(controllers)) {
        controller.abort();
      }
    };
  }, []);

  const getPopoverStyle = useCallback((eventId: string): React.CSSProperties => {
    const btn = audioButtonRefs.current[eventId];
    if (!btn) return { position: "fixed", top: 100, right: 16 };
    const rect = btn.getBoundingClientRect();
    const top = Math.min(rect.bottom + 4, window.innerHeight - 200);
    const right = Math.max(8, window.innerWidth - rect.right);
    return { position: "fixed", top, right };
  }, []);

  // --- Current message position for jump trigger ---
  const focusedEventIndex = focusedEventId ? events.findIndex((e) => e.id === focusedEventId) : -1;
  const jumpLabel = focusedEventIndex >= 0
    ? `${focusedEventIndex + 1} / ${events.length}`
    : `${events.length}`;

  return (
    <div
      ref={scrollContainerRef}
      data-testid={`messages-pane-${sessionId}`}
      className="relative h-full overflow-auto bg-wc-surface-base px-2 pb-4 pt-1 select-text"
    >
      <div className="flex flex-col">
        {/* Control strip */}
        <div
          data-testid="messages-control-strip"
          className="sticky top-0 z-10 flex items-center justify-start gap-1.5 bg-wc-surface-base/80 py-1.5 backdrop-blur-sm"
        >
          <button
            data-testid="messages-search-btn"
            onClick={() => setSearchOpen(true)}
            className="flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm"
            title="Search messages"
            type="button"
          >
            <Search className="h-3.5 w-3.5" />
          </button>

          {/* Jump trigger */}
          <button
            data-testid="msg-jump-trigger"
            onClick={() => setJumpListOpen((v) => !v)}
            disabled={events.length === 0}
            className="flex h-8 items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2.5 text-xs text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-30 disabled:pointer-events-none"
            title="Jump to message"
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
            title={searchQuery ? "Previous match" : "Previous message"}
            type="button"
          >
            <ChevronUp className="h-3.5 w-3.5" />
          </button>
          <button
            data-testid="messages-nav-down"
            onClick={handleNavDown}
            disabled={navIds.length === 0}
            className="flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-30 disabled:pointer-events-none"
            title={searchQuery ? "Next match" : "Next message"}
            type="button"
          >
            <ChevronDown className="h-3.5 w-3.5" />
          </button>
          <button
            data-testid="messages-refresh-btn"
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-60 disabled:pointer-events-none"
            title="Refresh messages from server"
            type="button"
          >
            <RotateCw className={cn("h-3.5 w-3.5", isRefreshing && "animate-spin")} />
          </button>
        </div>

        {/* Search drawer */}
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

        {/* Jump list */}
        {jumpListOpen && (
          <MessageJumpList
            events={events}
            focusedEventId={focusedEventId}
            onSelect={focusAndScroll}
            onClose={() => setJumpListOpen(false)}
          />
        )}

        {/* Messages */}
        {events.length === 0 ? (
          <div className="rounded-xl border border-dashed border-wc-default bg-wc-surface px-4 py-6 text-sm text-wc-text-muted">
            No conversation events yet for this session.
          </div>
        ) : (
          events.map((event) => {
            const isUser = event.role === "user";
            const isTtsActive = !isUser && isTtsSpeaking && activeSpeakingEventId === event.id;
            const isFocused = focusedEventId === event.id;
            const hasSummary = event.summarized && event.originalSpeechParagraphs != null && event.originalSpeechParagraphs.length > 0;
            const useSummarized = playbackModes[event.id] ?? event.summarized;
            const isPopoverOpen = openPopoverId === event.id;

            // Collapse logic
            const isTall = tallIds.has(event.id);
            const isExpanded = expandedIds.has(event.id);
            const isCollapsed = isTall && !isExpanded;

            // Accent bar color
            const accentColor = isTtsActive
              ? "border-l-wc-accent"
              : isUser
                ? "border-l-sky-500/60"
                : "border-l-emerald-500/60";

            return (
              <article
                key={event.id}
                ref={(el) => { if (el) messageRefs.current.set(event.id, el); else messageRefs.current.delete(event.id); }}
                data-testid={`msg-card-${event.id}`}
                className={cn(
                  "border-b border-wc-default border-l-[3px] py-3 pl-3 pr-1 transition-colors",
                  accentColor,
                  isFocused && "bg-wc-accent/5",
                  // Dim non-matching messages during search
                  searchQuery && !searchMatchIds.includes(event.id) && "opacity-40",
                )}
              >
                {/* Header row */}
                <div className="mb-1.5 flex items-center gap-2 text-[11px] uppercase tracking-[0.12em] text-wc-text-faint">
                  <button
                    data-testid={`msg-copy-${event.id}`}
                    onClick={() => handleCopy(event.id, event.text)}
                    className="rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10"
                    title="Copy message"
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
                        onClick={() => onSpeakFromHere(event.id)}
                        className="rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10"
                        title="Read from here"
                        type="button"
                      >
                        <Play className="h-3.5 w-3.5" />
                      </button>
                      <PlaybackModeControl
                        testIdPrefix={`msg-${event.id}`}
                        isSummarized={useSummarized && hasSummary}
                        hasOriginalVersion={hasSummary}
                        canSummarize
                        isSummarizing={summarizingIds.has(event.id)}
                        currentLevel={summarizeLevel}
                        onToggleSummarized={(use) => handleToggleSummarized(event.id, use)}
                        onChangeLevel={(level) => handleChangeLevelForEvent(event.id, level)}
                      />
                      <button
                        ref={(el) => { audioButtonRefs.current[event.id] = el; }}
                        data-testid={`msg-audio-${event.id}`}
                        onClick={() => {
                          handleSpeakOne(event);
                          setOpenPopoverId(isPopoverOpen ? null : event.id);
                        }}
                        className="rounded p-0.5 text-wc-text-faint transition hover:text-wc-text-muted hover:bg-wc-accent/10"
                        title="Play and open audio settings"
                        type="button"
                      >
                        <Volume2 className="h-3 w-3" />
                      </button>

                      {/* Audio popover (volume) */}
                      {isPopoverOpen && createPortal(
                        isMobile ? (
                          <div className="fixed inset-0 z-[60]" onMouseDown={(e) => e.preventDefault()}>
                            <div className="absolute inset-0 bg-wc-backdrop" onClick={() => setOpenPopoverId(null)} />
                            <div
                              data-testid={`audio-popover-${event.id}`}
                              className="absolute bottom-0 left-0 right-0 z-[61] rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-4 pb-[max(1rem,var(--wc-safe-bottom))] shadow-2xl"
                            >
                              <div className="mb-3 flex justify-center">
                                <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
                              </div>
                              <h3 className="mb-3 text-sm font-semibold text-wc-text-primary">Audio Settings</h3>
                              <AudioSettingsContent
                                testIdPrefix={`msg-${event.id}`}
                                volume={volume}
                                playbackRate={1}
                                isSummarized={useSummarized && hasSummary}
                                capabilities={{ canPause: false, canSeek: false, canAdjustSpeed: false, canAdjustVolume: true }}
                                onVolumeChange={setVolume}
                                onSetPlaybackRate={() => { /* speed adjustment not applicable per-message */ }}
                              />
                              {summarizeErrors[event.id] && (
                                <div
                                  data-testid={`msg-summarize-error-${event.id}`}
                                  className="mt-2 rounded-lg bg-red-500/10 px-3 py-2 text-[11px] text-red-400"
                                >
                                  {summarizeErrors[event.id]}
                                </div>
                              )}
                              {summarizingIds.has(event.id) && (
                                <button
                                  data-testid={`msg-cancel-summarize-${event.id}`}
                                  className="mt-2 w-full rounded-lg bg-red-500/10 px-3 py-2 text-xs font-medium text-red-400 transition hover:bg-red-500/20"
                                  onClick={() => handleCancelSummarize(event.id)}
                                >
                                  Cancel summarization
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
                              style={getPopoverStyle(event.id)}
                            >
                              <AudioSettingsContent
                                testIdPrefix={`msg-${event.id}`}
                                volume={volume}
                                playbackRate={1}
                                isSummarized={useSummarized && hasSummary}
                                capabilities={{ canPause: false, canSeek: false, canAdjustSpeed: false, canAdjustVolume: true }}
                                onVolumeChange={setVolume}
                                onSetPlaybackRate={() => { /* speed adjustment not applicable per-message */ }}
                              />
                              {summarizeErrors[event.id] && (
                                <div
                                  data-testid={`msg-summarize-error-${event.id}`}
                                  className="mt-2 rounded-lg bg-red-500/10 px-3 py-2 text-[11px] text-red-400"
                                >
                                  {summarizeErrors[event.id]}
                                </div>
                              )}
                              {summarizingIds.has(event.id) && (
                                <button
                                  data-testid={`msg-cancel-summarize-${event.id}`}
                                  className="mt-2 w-full rounded-lg bg-red-500/10 px-3 py-2 text-xs font-medium text-red-400 transition hover:bg-red-500/20"
                                  onClick={() => handleCancelSummarize(event.id)}
                                >
                                  Cancel summarization
                                </button>
                              )}
                            </div>
                          </>
                        ),
                        document.body,
                      )}
                    </>
                  )}

                  <span className="flex-1">
                    {isUser ? "You" : event.source === "claude_hook" ? "Claude Code" : "Codex"}
                  </span>

                  {hasSummary && (
                    <span
                      data-testid={`msg-summarized-badge-${event.id}`}
                      className={cn(
                        "rounded-full px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider",
                        useSummarized
                          ? "bg-amber-500/20 text-amber-400"
                          : "bg-wc-surface-base text-wc-text-faint",
                      )}
                    >
                      {useSummarized ? "Summarized" : "Original"}
                    </span>
                  )}
                  <span>#{event.sequence}</span>
                </div>

                {/* Message content with markdown rendering */}
                <div
                  className={cn("relative", isCollapsed && "max-h-[400px] overflow-hidden")}
                >
                  <div
                    ref={(el) => { if (el) contentRefs.current.set(event.id, el); else contentRefs.current.delete(event.id); }}
                    data-event-id={event.id}
                    data-testid={`msg-markdown-${event.id}`}
                    style={{ fontSize: `${fontSize}px` }}
                    className="text-wc-text-primary"
                  >
                    <MarkdownRenderer
                      content={event.text}
                      searchQuery={searchQuery || undefined}
                      isSearchFocused={searchMatchIds[currentMatchIndex] === event.id}
                    />
                  </div>

                  {/* Gradient fade when collapsed */}
                  {isCollapsed && (
                    <div className="absolute bottom-0 left-0 right-0 h-20 bg-gradient-to-t from-wc-surface-base to-transparent pointer-events-none" />
                  )}
                </div>

                {/* Collapse toggle */}
                {isTall && (
                  <button
                    data-testid={`msg-collapse-${event.id}`}
                    onClick={() => {
                      setExpandedIds((prev) => {
                        const next = new Set(prev);
                        if (next.has(event.id)) next.delete(event.id);
                        else next.add(event.id);
                        return next;
                      });
                    }}
                    className="mt-1 text-xs text-wc-accent hover:text-wc-accent/80 transition-colors"
                    type="button"
                  >
                    {isExpanded ? "Show less" : "Show more"}
                  </button>
                )}
              </article>
            );
          })
        )}

        {/* Sentinel for auto-scroll detection */}
        <div ref={sentinelRef} className="h-1" aria-hidden="true" />
      </div>

      {/* "New messages" pill */}
      {newMessageCount > 0 && (
        <button
          data-testid="msg-new-pill"
          onClick={scrollToBottom}
          className="fixed bottom-16 left-1/2 z-20 -translate-x-1/2 rounded-full border border-wc-default bg-wc-surface-raised px-4 py-2 text-xs font-medium text-wc-text-primary shadow-lg backdrop-blur-sm transition-all hover:bg-wc-surface-input"
          type="button"
        >
          <ArrowDown className="mr-1.5 inline-block h-3.5 w-3.5" />
          {newMessageCount} new message{newMessageCount !== 1 ? "s" : ""}
        </button>
      )}
    </div>
  );
}
