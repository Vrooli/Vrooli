import { useCallback, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown, ChevronUp, Copy, Play, Search, Volume2 } from "lucide-react";
import { useConversationStore, getSessionConversationEvents } from "../stores/useConversationStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { summarizeEvent } from "../lib/api";
import { TERMINAL_FONT_SIZE } from "../consts/config";
import { cn } from "../lib/classnames";
import MessagesSearchDrawer from "./MessagesSearchDrawer";

interface MessagesPaneProps {
  sessionId: string;
  /** Speak from this event through all subsequent events. */
  onSpeakFromHere: (eventId: string) => void;
  /** Speak only this event's text (with optional pre-computed paragraphs). */
  onSpeakOne: (eventId: string, text: string, paragraphs?: string[], opts?: { version?: "active" | "original" }) => void;
  /** Event ID currently being spoken, or null. */
  activeSpeakingEventId: string | null;
  /** Whether TTS is active on this pane. */
  isTtsSpeaking: boolean;
}

interface AudioPopoverContentProps {
  eventId: string;
  volume: number;
  summarized: boolean;
  hasSummary: boolean;
  isSummarizing: boolean;
  summarizeError: string | null;
  onVolumeChange: (level: number) => void;
  onToggleSummarized: (useSummarized: boolean) => void;
  onRequestSummarize: () => void;
  onClose: () => void;
}

function AudioPopoverContent({
  eventId,
  volume,
  summarized,
  hasSummary,
  isSummarizing,
  summarizeError,
  onVolumeChange,
  onToggleSummarized,
  onRequestSummarize,
  onClose,
}: AudioPopoverContentProps) {
  return (
    <div className="space-y-3">
      {/* Volume slider */}
      <div>
        <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-wc-text-faint">
          Volume
        </label>
        <input
          data-testid={`msg-volume-slider-${eventId}`}
          type="range"
          min={0}
          max={1}
          step={0.05}
          value={volume}
          onChange={(e) => onVolumeChange(Number(e.target.value))}
          className={cn(
            "h-1.5 w-full cursor-pointer rounded-full",
            summarized
              ? "[&::-webkit-slider-thumb]:bg-amber-400 accent-amber-400"
              : "accent-wc-accent",
          )}
        />
        <div className="mt-0.5 flex justify-between text-[10px] text-wc-text-faint">
          <span>0</span>
          <span>{Math.round(volume * 100)}%</span>
        </div>
      </div>

      {/* Summarization toggle — shown when event already has both versions */}
      {hasSummary && (
        <div className="border-t border-wc-default pt-3">
          <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-wc-text-faint">
            Playback version
          </label>
          <div className="flex gap-1">
            <button
              data-testid={`msg-play-summarized-${eventId}`}
              className={cn(
                "flex-1 rounded-lg px-2 py-1.5 text-xs font-medium transition",
                summarized
                  ? "bg-amber-500/20 text-amber-300 ring-1 ring-amber-500/40"
                  : "bg-wc-surface-base text-wc-text-muted hover:bg-wc-surface-input",
              )}
              onClick={() => { onToggleSummarized(true); onClose(); }}
            >
              Summarized
            </button>
            <button
              data-testid={`msg-play-original-${eventId}`}
              className={cn(
                "flex-1 rounded-lg px-2 py-1.5 text-xs font-medium transition",
                !summarized
                  ? "bg-wc-accent/20 text-wc-accent ring-1 ring-wc-accent/40"
                  : "bg-wc-surface-base text-wc-text-muted hover:bg-wc-surface-input",
              )}
              onClick={() => { onToggleSummarized(false); onClose(); }}
            >
              Original
            </button>
          </div>
        </div>
      )}

      {/* Request summarization — shown when no summary exists yet */}
      {!hasSummary && (
        <div className="border-t border-wc-default pt-3">
          <button
            data-testid={`msg-request-summarize-${eventId}`}
            disabled={isSummarizing}
            className={cn(
              "w-full rounded-lg px-3 py-2 text-xs font-medium transition",
              isSummarizing
                ? "bg-wc-surface-base text-wc-text-faint cursor-wait"
                : "bg-amber-500/15 text-amber-300 hover:bg-amber-500/25",
            )}
            onClick={onRequestSummarize}
          >
            {isSummarizing ? "Summarizing…" : "Summarize for playback"}
          </button>
          {summarizeError && (
            <div
              data-testid={`msg-summarize-error-${eventId}`}
              className="mt-2 rounded-lg bg-red-500/10 px-3 py-2 text-[11px] text-red-400"
            >
              {summarizeError}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function MessagesPane({
  sessionId,
  onSpeakFromHere,
  onSpeakOne,
  activeSpeakingEventId,
  isTtsSpeaking,
}: MessagesPaneProps) {
  const events = useConversationStore((state) => getSessionConversationEvents(state, sessionId));
  const isMobile = useMediaQuery("(max-width: 767px)");
  // Sync message text size with the terminal's font size for this pane
  const fontSize = useWorkspaceStore(
    useCallback((s) => s.panes.find((p) => p.sessionId === sessionId)?.fontSize ?? TERMINAL_FONT_SIZE, [sessionId]),
  );
  const [openPopoverId, setOpenPopoverId] = useState<string | null>(null);
  const [volume, setVolume] = useState(1);
  const [playbackModes, setPlaybackModes] = useState<Record<string, boolean>>({});
  const [summarizingIds, setSummarizingIds] = useState<Set<string>>(new Set());
  const [summarizeErrors, setSummarizeErrors] = useState<Record<string, string>>({});
  const audioButtonRefs = useRef<Record<string, HTMLButtonElement | null>>({});

  // --- Copy-to-clipboard state ---
  const [copiedEventId, setCopiedEventId] = useState<string | null>(null);
  const handleCopy = useCallback((eventId: string, text: string) => {
    void navigator.clipboard.writeText(text);
    setCopiedEventId(eventId);
    setTimeout(() => setCopiedEventId((prev) => (prev === eventId ? null : prev)), 2000);
  }, []);

  // --- Search & navigation state ---
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  // The single focused event ID drives both the accent border highlight and
  // the starting point for chevron navigation. Set by clicking a card or
  // pressing the up/down chevrons.
  const [focusedEventId, setFocusedEventId] = useState<string | null>(null);
  const messageRefs = useRef<Map<string, HTMLElement>>(new Map());

  // Event IDs matching the current search query (case-insensitive)
  const searchMatchIds = useMemo(() => {
    if (!searchQuery) return [];
    const q = searchQuery.toLowerCase();
    return events.filter((e) => e.text.toLowerCase().includes(q)).map((e) => e.id);
  }, [events, searchQuery]);

  // The navigable list depends on mode: search matches when searching,
  // all events when not searching (so tapping any message + using arrows works).
  const navIds = useMemo(
    () => (searchQuery ? searchMatchIds : events.map((e) => e.id)),
    [searchQuery, searchMatchIds, events],
  );

  // Index of the focused event within the navigable list
  const focusedNavIndex = useMemo(
    () => (focusedEventId ? navIds.indexOf(focusedEventId) : -1),
    [focusedEventId, navIds],
  );

  // Index of focused event within search matches (for the "N of M" label)
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
  }, [scrollToEvent]);

  // Navigate to the previous item in the navigable list
  const handleNavUp = useCallback(() => {
    if (navIds.length === 0) return;
    const prev = focusedNavIndex <= 0 ? navIds.length - 1 : focusedNavIndex - 1;
    const targetId = navIds[prev];
    if (!targetId) return;
    focusAndScroll(targetId);
  }, [navIds, focusedNavIndex, focusAndScroll]);

  // Navigate to the next item in the navigable list
  const handleNavDown = useCallback(() => {
    if (navIds.length === 0) return;
    const next = focusedNavIndex < 0 ? 0 : (focusedNavIndex + 1) % navIds.length;
    const targetId = navIds[next];
    if (!targetId) return;
    focusAndScroll(targetId);
  }, [navIds, focusedNavIndex, focusAndScroll]);

  const handleCloseSearch = useCallback(() => {
    setSearchOpen(false);
    setSearchQuery("");
    setFocusedEventId(null);
  }, []);

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
    setSummarizingIds((prev) => new Set(prev).add(eventId));
    setSummarizeErrors((prev) => {
      const next = { ...prev };
      delete next[eventId];
      return next;
    });
    void summarizeEvent(sessionId, eventId).then((res) => {
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
        setSummarizeErrors((prev) => ({ ...prev, [eventId]: res.error! }));
      }
    }).catch((err: unknown) => {
      const message = err instanceof Error ? err.message : "Summarization failed";
      setSummarizeErrors((prev) => ({ ...prev, [eventId]: message }));
    }).finally(() => {
      setSummarizingIds((prev) => {
        const next = new Set(prev);
        next.delete(eventId);
        return next;
      });
    });
  }, [sessionId]);

  const getPopoverStyle = useCallback((eventId: string): React.CSSProperties => {
    const btn = audioButtonRefs.current[eventId];
    if (!btn) return { position: "fixed", top: 100, right: 16 };
    const rect = btn.getBoundingClientRect();
    // Position below the button, clamped to viewport
    const top = Math.min(rect.bottom + 4, window.innerHeight - 200);
    const right = Math.max(8, window.innerWidth - rect.right);
    return { position: "fixed", top, right };
  }, []);

  /**
   * Splits `text` on case-insensitive occurrences of `query`, wrapping each
   * match in a <mark> element. The currently focused match gets a stronger
   * highlight so the user can tell which result they navigated to.
   */
  const highlightMatches = useCallback(
    (text: string, query: string, isFocusedMatch: boolean): React.ReactNode => {
      if (!query) return text;
      const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
      const parts = text.split(new RegExp(`(${escaped})`, "gi"));
      return parts.map((part, i) =>
        part.toLowerCase() === query.toLowerCase() ? (
          <mark
            key={i}
            className={cn(
              "rounded-sm text-wc-text-primary",
              isFocusedMatch ? "bg-wc-accent/50" : "bg-wc-accent/20",
            )}
          >
            {part}
          </mark>
        ) : (
          part
        ),
      );
    },
    [],
  );

  return (
    <div
      data-testid={`messages-pane-${sessionId}`}
      className="h-full overflow-auto bg-wc-surface-base px-4 pb-4 pt-3"
    >
      <div className="mx-auto flex max-w-3xl flex-col gap-3">
        {/* Control strip: search + jump navigation (sticky at top of scroll).
         * pr-12 reserves space for the view-toggle button (h-9 w-9 at right-3)
         * that Workspace.tsx positions absolutely over this pane. */}
        <div
          data-testid="messages-control-strip"
          className="sticky top-0 z-10 flex items-center justify-end gap-1.5 bg-wc-surface-base/80 pb-1 pr-12 backdrop-blur-sm"
        >
          <button
            data-testid="messages-search-btn"
            onClick={() => setSearchOpen(true)}
            className="flex h-9 w-9 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm"
            title="Search messages"
          >
            <Search className="h-4 w-4" />
          </button>
          <button
            data-testid="messages-nav-up"
            onClick={handleNavUp}
            disabled={navIds.length === 0}
            className="flex h-9 w-9 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-30 disabled:pointer-events-none"
            title={searchQuery ? "Previous match" : "Previous message"}
          >
            <ChevronUp className="h-4 w-4" />
          </button>
          <button
            data-testid="messages-nav-down"
            onClick={handleNavDown}
            disabled={navIds.length === 0}
            className="flex h-9 w-9 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm disabled:opacity-30 disabled:pointer-events-none"
            title={searchQuery ? "Next match" : "Next message"}
          >
            <ChevronDown className="h-4 w-4" />
          </button>
        </div>

        {/* Search drawer (bottom sheet on mobile, portal on desktop) */}
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

            return (
              <article
                key={event.id}
                ref={(el) => { if (el) messageRefs.current.set(event.id, el); else messageRefs.current.delete(event.id); }}
                data-testid={`msg-card-${event.id}`}
                onClick={() => setFocusedEventId(event.id)}
                className={cn(
                  "cursor-pointer rounded-2xl border px-4 py-3 shadow-sm transition-colors",
                  isUser
                    ? "ml-auto max-w-[85%] bg-wc-accent/10"
                    : "bg-wc-surface",
                  // Focused card gets accent border; otherwise default gray
                  isFocused
                    ? "border-wc-accent"
                    : "border-wc-default",
                  isTtsActive && "border-l-[3px] border-l-wc-accent",
                )}
              >
                <div className="mb-2 flex items-center gap-2 text-[11px] uppercase tracking-[0.12em] text-wc-text-faint">
                  {/* Copy button — shown for all messages (user + assistant) */}
                  <button
                    data-testid={`msg-copy-${event.id}`}
                    onClick={() => handleCopy(event.id, event.text)}
                    className="rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10"
                    title="Copy message"
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
                      >
                        <Play className="h-3.5 w-3.5" />
                      </button>
                      <button
                        ref={(el) => { audioButtonRefs.current[event.id] = el; }}
                        data-testid={`msg-audio-${event.id}`}
                        onClick={() => {
                          handleSpeakOne(event);
                          setOpenPopoverId(isPopoverOpen ? null : event.id);
                        }}
                        className={cn(
                          "rounded p-0.5 transition",
                          hasSummary && useSummarized
                            ? "text-amber-400 hover:text-amber-300 hover:bg-amber-500/10"
                            : "text-wc-text-faint hover:text-wc-text-muted hover:bg-wc-accent/10",
                        )}
                        title="Audio options"
                      >
                        <Volume2 className="h-3 w-3" />
                      </button>

                      {/* Popover / bottom sheet — always via portal */}
                      {isPopoverOpen && createPortal(
                        isMobile ? (
                          <div className="fixed inset-0 z-[60]" onMouseDown={(e) => e.preventDefault()}>
                            <div
                              className="absolute inset-0 bg-wc-backdrop"
                              onClick={() => setOpenPopoverId(null)}
                            />
                            <div
                              data-testid={`audio-popover-${event.id}`}
                              className="absolute bottom-0 left-0 right-0 z-[61] rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-4 pb-[max(1rem,var(--wc-safe-bottom))] shadow-2xl"
                            >
                              <div className="mb-3 flex justify-center">
                                <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
                              </div>
                              <h3 className="mb-3 text-sm font-semibold text-wc-text-primary">Audio Settings</h3>
                              <AudioPopoverContent
                                eventId={event.id}
                                volume={volume}
                                summarized={useSummarized}
                                hasSummary={hasSummary}
                                isSummarizing={summarizingIds.has(event.id)}
                                summarizeError={summarizeErrors[event.id] ?? null}
                                onVolumeChange={setVolume}
                                onToggleSummarized={(use) => handleToggleSummarized(event.id, use)}
                                onRequestSummarize={() => handleRequestSummarize(event.id)}
                                onClose={() => setOpenPopoverId(null)}
                              />
                            </div>
                          </div>
                        ) : (
                          <>
                            <div
                              className="fixed inset-0 z-[60]"
                              onClick={() => setOpenPopoverId(null)}
                            />
                            <div
                              data-testid={`audio-popover-${event.id}`}
                              className="z-[61] w-56 rounded-xl border border-wc-default bg-wc-surface-raised p-3 shadow-lg"
                              style={getPopoverStyle(event.id)}
                            >
                              <AudioPopoverContent
                                eventId={event.id}
                                volume={volume}
                                summarized={useSummarized}
                                hasSummary={hasSummary}
                                isSummarizing={summarizingIds.has(event.id)}
                                summarizeError={summarizeErrors[event.id] ?? null}
                                onVolumeChange={setVolume}
                                onToggleSummarized={(use) => handleToggleSummarized(event.id, use)}
                                onRequestSummarize={() => handleRequestSummarize(event.id)}
                                onClose={() => setOpenPopoverId(null)}
                              />
                            </div>
                          </>
                        ),
                        document.body,
                      )}
                    </>
                  )}
                  <span className="flex-1">
                    {isUser
                      ? "You"
                      : event.source === "claude_hook"
                        ? "Claude Code"
                        : "Codex"}
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
                <div className="whitespace-pre-wrap text-wc-text-primary" style={{ fontSize: `${fontSize}px` }}>
                  {searchQuery
                    ? highlightMatches(event.text, searchQuery, searchMatchIds[currentMatchIndex] === event.id)
                    : event.text}
                </div>
              </article>
            );
          })
        )}
      </div>
    </div>
  );
}
