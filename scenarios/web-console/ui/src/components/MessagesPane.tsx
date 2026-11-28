import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { ArrowDown, CheckSquare, SlidersHorizontal } from "lucide-react";
import { useConversationStore, getSessionConversationEvents, getSessionSlice, resolveConversationView } from "../stores/useConversationStore";
import { loadConversationPageContaining, loadOlderConversationPage, refreshConversationSession } from "../hooks/useConversationSession";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useMessagesViewStore, type OpenReader } from "../stores/useMessagesViewStore";
import { clampReaderFontSize, useTypographyPinch } from "@vrooli/file-viewer";
import { useTrailingThrottle } from "../hooks/useTrailingThrottle";
import { captureTopPosition } from "./messages/scrollPosition";
import { useLiveStreamNotice } from "../hooks/useLiveStreamNotice";
import { writeText } from "../lib/clipboard";
import { CONVERSATION_SEARCH_KINDS, executeConversationControl, getAllConversationEvents, getConversationRange, preflightConversationControl, resolveConversationEvents, searchConversation, type ConversationControlPreflight, type ConversationEvent, type ConversationSearchMatch, type ConversationThreadSummary } from "../api/conversation";
import { coversSequence, isAgentReply, isConversationRow, isEmptyWork, isWorkRow } from "../lib/conversationRows";
import { cn } from "../lib/classnames";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Button } from "@vrooli/react-component-library/Button/2";
import { looksLikeFileReference } from "@vrooli/file-viewer/file-references";
import { rewriteLocalhostUrlForProxy } from "../lib/terminalFileLinks";
import { useVirtualList } from "../hooks/useVirtualList";
import { useReleaseOnElementInteraction } from "../hooks/useKeyboardListeners";
import type { MessageExportSelection, SearchOptionsState } from "./MessageJumpList";
import {
  DEFAULT_MESSAGE_FILTER_STATE,
  activeMessageFilterCount,
  availableSources,
  buildResults,
  buildSearchResults,
  filterAndSortEvents,
  searchLoadedEvents,
  type MessageFilterState,
  type SourceId,
} from "./MessageJumpList.helpers";
import { SearchField } from "./SearchField";
import { MessagesFilterPopover } from "./messages/MessagesFilterPopover";
import { MessagesSearchResults } from "./messages/MessagesSearchResults";
import MessageExportDrawer from "./MessageExportDrawer";
import { buildMessageExport, DEFAULT_MESSAGE_EXPORT_FORMAT } from "../lib/messageExport";
import type { SummarizationLevel } from "./tts/PlaybackModeControl";
import type { PlaybackFocusRequest, PlaybackVersion } from "../domains/tts-playback/types";
import type { PreviewSourceContext } from "@vrooli/file-viewer";
import HandoffSuggestionChip from "./handoff/HandoffSuggestionChip";
import { groupSuggestionsByRule } from "../lib/captureRules";
import { useHandoffSuggestions } from "../hooks/useHandoffSuggestions";
import MessagesMermaidViewer from "./MessagesMermaidViewer";
import MessagesPaneState from "./MessagesPaneState";
import { PaneEmptyState } from "./PaneEmptyState";
import MessagesPaneStatusLine from "./MessagesPaneStatusLine";
import { resolveMessagesPaneStatus } from "../lib/messagesPaneStatus";
import { SnippetSaveSheet } from "./snippets/SnippetSaveSheet";
import { MessageRow } from "./messages/MessageRow";
import { NoticeRow } from "./messages/NoticeRow";
import { ThreadGroupRow } from "./messages/ThreadGroupRow";
import { ThreadView } from "./messages/ThreadView";
import { WorkSpanRow } from "./messages/WorkSpanRow";
import { threadAnchors } from "./messages/threadAnchors";
import { MessagesReader } from "./messages/MessagesReader";
import { replyIndices, replyPosition, seekReply } from "./messages/replyTrack";
import type { MessageActionContext } from "./messages/messageActions";
import type { ActionsOrigin } from "./messages/MessageActionList";
import { usePressGesture } from "../hooks/usePressGesture";
import { useTouchControls } from "../hooks/useTouchControls";
import { SessionStateSlot } from "./messages/SessionStateSlot";
import { resolveStateSlot } from "./messages/resolveStateSlot";
import { answerPrompt } from "../api/promptAnswer";
import { useSessionActivityStore } from "../stores/useSessionActivityStore";
import { EchoRow } from "./messages/EchoRow";
import type { SentEcho } from "../lib/echoMatch";

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
  /** Switches this pane to its terminal (the state slot's "Open terminal"). */
  onOpenTerminal?: () => void;
  /** Presses Enter in this pane's terminal (an echo row's "Press Enter"). */
  onPressEnter?: () => void;
  /** Opens a preview in the workspace-level viewer host. */
  onOpenFilePreview?: (path: string, source: PreviewSourceContext) => void;
}

// ---------------------------------------------------------------------------
// Scroll model (docs/internal/INVARIANTS.md "Messages scroll"): one follow
// boolean plus the prepend anchor. The virtualizer's resize compensation is
// the only other thing allowed to move the viewport.
// ---------------------------------------------------------------------------
/** The list follows new content while the viewport bottom is this close to the end. */
const FOLLOW_THRESHOLD_PX = 200;
/** The reader's place is saved at most this often while scrolling (and on unmount). */
const POSITION_SAVE_THROTTLE_MS = 250;

interface PrependScrollAnchor {
  eventId: string;
  offsetFromViewportTop: number;
}

// ---------------------------------------------------------------------------
// MessagesPane
// ---------------------------------------------------------------------------

const NO_ECHOES: readonly SentEcho[] = [];

/** The Messages view searches its conversation unless the reader asks for tool calls and context too. */
function serverSearchOptions(options: SearchOptionsState) {
  const { includeWork, ...rest } = options;
  return includeWork ? rest : { ...rest, kinds: CONVERSATION_SEARCH_KINDS };
}

/** When the nearest conversation row before (-1) or after (1) a list position was written. */
function conversationTimeAt(list: readonly ConversationEvent[], index: number, direction: -1 | 1): string | undefined {
  for (let position = index + direction; position >= 0 && position < list.length; position += direction) {
    const row = list[position];
    if (row && !isWorkRow(row)) return row.occurredAt || row.createdAt;
  }
  return undefined;
}
const NO_THREADS: readonly ConversationThreadSummary[] = [];
const NO_THREAD_GROUPS: readonly ConversationThreadSummary[] = [];

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
  playbackFocusRequest,
  toolbarTrailingAction,
  readOnly = false,
  focusEventId = null,
  focusSequence = null,
  onSendToComposer,
  onOpenTerminal,
  onPressEnter,
  onOpenFilePreview,
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

  const messagesFontSize = clampReaderFontSize(useMessagesViewStore((state) => state.messagesFontSize));
  const setMessagesFontSize = useCallback((size: number) => {
    useMessagesViewStore.getState().setMessagesFontSize(clampReaderFontSize(size));
  }, []);

  // --- Copy ---
  const [copiedEventId, setCopiedEventId] = useState<string | null>(null);
  const [snippetSaveSource, setSnippetSaveSource] = useState<{ body: string; sourceLabel: string } | null>(null);
  const [rewindPreflight, setRewindPreflight] = useState<ConversationControlPreflight | null>(null);
  const [rewindOperation, setRewindOperation] = useState("restore_conversation");
  const [rewindError, setRewindError] = useState<string | null>(null);
  const [rewindNotice, setRewindNotice] = useState<string | null>(null);
  const [rewindBusy, setRewindBusy] = useState(false);
  const [rewindPreserveDraft, setRewindPreserveDraft] = useState(true);
  const handleCopy = useCallback((eventId: string, text: string) => {
    void writeText(text);
    setCopiedEventId(eventId);
    setTimeout(() => { setCopiedEventId((prev) => (prev === eventId ? null : prev)); }, 2000);
  }, []);

  // --- Search results ---
  // The toolbar field owns the query. The result list is a normal anchored
  // dropdown, and the same query dims non-matches in the conversation below.
  const [searchQuery, setSearchQuery] = useState("");
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchActiveIndex, setSearchActiveIndex] = useState(0);
  const searchFieldRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const [focusedEventId, setFocusedEventId] = useState<string | null>(null);
  const [serverSearchMatches, setServerSearchMatches] = useState<ConversationSearchMatch[]>([]);
  const [serverSearchReady, setServerSearchReady] = useState(false);
  const [searchTruncated, setSearchTruncated] = useState(false);
  const [searchHasMore, setSearchHasMore] = useState(false);
  const [searchNextAfterSequence, setSearchNextAfterSequence] = useState(0);
  const [searchContinueLoading, setSearchContinueLoading] = useState(false);
  const searchRequestId = useRef(0);
  const [searchError, setSearchError] = useState<string | undefined>(undefined);
  const [exportResolutionError, setExportResolutionError] = useState<string | undefined>(undefined);
  const exportResolutionRequest = useRef(0);
  const [searchOptions, setSearchOptions] = useState<SearchOptionsState>({ mode: "text", caseSensitive: false, wholeWord: false });
  const [filters, setFilters] = useState<MessageFilterState>(DEFAULT_MESSAGE_FILTER_STATE);
  const filterActiveCount = activeMessageFilterCount(filters)
    + (searchOptions.mode !== "text" ? 1 : 0)
    + (searchOptions.caseSensitive ? 1 : 0)
    + (searchOptions.wholeWord ? 1 : 0)
    + (searchOptions.includeWork ? 1 : 0);
  const sourceOptions = useMemo<SourceId[]>(() => availableSources(events), [events]);
  // Agent threads (subagents, teammates) are groups placed at the work row
  // that covers where each began; a work row of nothing but other threads'
  // rows is kept only when it carries such a group.
  const threads = useConversationStore((state) => state.sessions[sessionId]?.threads) ?? NO_THREADS;
  const threadsByAnchor = useMemo(() => threadAnchors(events, threads), [events, threads]);
  const visibleEvents = useMemo(
    () => filterAndSortEvents(events, filters).filter((event) => !isWorkRow(event) || !isEmptyWork(event.work) || threadsByAnchor.has(event.id)),
    [events, filters, threadsByAnchor],
  );
  // Work rows open in place; which are open survives the rows virtualising.
  const [expandedWork, setExpandedWork] = useState<ReadonlySet<string>>(new Set());
  const toggleWork = useCallback((eventId: string) => {
    setExpandedWork((current) => {
      const next = new Set(current);
      if (next.has(eventId)) next.delete(eventId);
      else next.add(eventId);
      return next;
    });
  }, []);
  const [openThreadId, setOpenThreadId] = useState<string | null>(null);
  const openThread = openThreadId == null ? null : threads.find((thread) => thread.threadId === openThreadId) ?? null;

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
  const [selectionMode, setSelectionMode] = useState(false);
  const selectionAnchorRef = useRef<string | null>(null);

  // --- File preview ---
  // Rules only ever offer. Nothing on this path can send.
  const handoffSuggestions = useHandoffSuggestions(sessionId);

  // --- Scroll: follow + prepend anchor ---
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const listPinchSize = useTypographyPinch(scrollContainerRef, messagesFontSize, setMessagesFontSize);
  const renderedMessagesFontSize = listPinchSize ?? messagesFontSize;
  // Where this session's list was left (persisted across reloads). An archive
  // hit being revealed takes precedence over it.
  const [savedPosition] = useState(() => (
    focusEventId ? null : useMessagesViewStore.getState().positions[sessionId] ?? null
  ));
  // True while the user is within FOLLOW_THRESHOLD_PX of the end. Only user
  // intent writes it: the user's own scrolling, jump-to-bottom, and explicit
  // jumps to a message. `following` mirrors it for rendering.
  const followRef = useRef(savedPosition ? savedPosition.follow : true);
  const [following, setFollowing] = useState(followRef.current);
  // A saved place still to be shown. Restore pages it in, scrolls to it once,
  // and clears it.
  const [restoreTarget, setRestoreTarget] = useState(savedPosition && !savedPosition.follow ? savedPosition : null);
  const [newMessageCount, setNewMessageCount] = useState(0);
  // Last event id seen by the follow rule, so appended events are counted and
  // a prepended older page is not mistaken for new messages.
  const lastEventIdRef = useRef<string | null>(null);
  const programmaticScrollRef = useRef(false);
  // Pagination inserts older rows before the visible window. This is the only
  // anchor besides the virtualizer's resize compensation: it keeps the visible
  // row fixed through the insertion so paging reads as continuous scrolling.
  const prependScrollAnchorRef = useRef<PrependScrollAnchor | null>(null);

  // --- Session activity: the one slot under the last row ---
  const activity = useSessionActivityStore((state) => state.activities[sessionId]);
  const stateSlot = useMemo(() => resolveStateSlot(activity), [activity]);
  // Answers the prompt the state slot shows, by its hash; the chosen option
  // shows as an echo row until the transcript or the screen confirms it.
  const answerSlotPrompt = useCallback(async (optionKey: string | null) => {
    if (stateSlot?.kind !== "waiting-answerable") return;
    const promptHash = stateSlot.prompt.hash ?? "";
    const result = await answerPrompt(sessionId, optionKey === null ? { promptHash, cancel: true } : { optionKey, promptHash, cancel: false });
    if (result.answer) useConversationStore.getState().addEcho(sessionId, { text: result.answer });
  }, [sessionId, stateSlot]);
  // Changes when the slot appears, goes, or changes shape — the follow rule
  // treats that like new content.
  const stateSlotKey = stateSlot
    ? `${stateSlot.kind}:${"prompt" in stateSlot ? `${String(stateSlot.prompt.text.length)}/${String(stateSlot.prompt.options.length)}` : ""}`
    : "";
  // Sends the harness has not recorded yet, shown after the last row.
  const echoes = useConversationStore((state) => state.echoes[sessionId] ?? NO_ECHOES);
  const expireEcho = useCallback((echoId: string) => {
    useConversationStore.getState().expireEcho(sessionId, echoId);
  }, [sessionId]);
  // Pressing Enter from an echo row (or the reader) both submits in the
  // terminal and records the optimistic submit so the row stops offering Enter
  // immediately — queued when the agent is already working.
  const handlePressEnter = useCallback(() => {
    useConversationStore.getState().markSubmitted(sessionId, { agentBusy: activity?.state === "working" });
    onPressEnter?.();
  }, [sessionId, activity?.state, onPressEnter]);
  // The newest unrecorded send drives the reader's pending pill.
  const pendingOutbox = echoes.length > 0 ? echoes[echoes.length - 1] : null;

  const setFollow = useCallback((next: boolean) => {
    followRef.current = next;
    setFollowing(next);
    if (next) setNewMessageCount(0);
  }, []);

  // Records the reader's place: the top visible message, its offset, and
  // follow. Never mid programmatic scroll, which would record a transit row.
  const persistPosition = useCallback(() => {
    const el = scrollContainerRef.current;
    if (!el || programmaticScrollRef.current) return;
    const top = captureTopPosition(el);
    if (top) useMessagesViewStore.getState().savePosition(sessionId, { ...top, follow: followRef.current });
  }, [sessionId]);
  const schedulePositionSave = useTrailingThrottle(persistPosition, POSITION_SAVE_THROTTLE_MS);

  // Marks the list as positioned by code. Only a user gesture (wheel, touch,
  // pointer, key — see useReleaseOnElementInteraction below) clears the mark,
  // never a timer: scroll events caused by layout (row measurement, resize
  // compensation, a page merge) must not decide follow, and a timer cannot
  // tell them apart from the user.
  const runProgrammaticScroll = useCallback((scroll: () => void) => {
    programmaticScrollRef.current = true;
    scroll();
  }, []);

  // --- Refresh: on mount, on browser tab focus, and as the error state's retry ---
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

  const estimateMessageHeight = useCallback((index: number) => {
    const event = visibleEvents[index];
    if (!event) return 140;
    const groups = (threadsByAnchor.get(event.id)?.length ?? 0) * 64;
    if (isWorkRow(event)) return (isEmptyWork(event.work) ? 0 : 37) + groups;
    if (!isConversationRow(event) || event.kind === "notice" || event.kind === "attachment") return 37;
    const lineEstimate = Math.ceil(event.text.length / 90);
    return Math.max(110, Math.min(520, 72 + lineEstimate * 22));
  }, [threadsByAnchor, visibleEvents]);
  const getMessageKey = useCallback((index: number) => visibleEvents[index]?.id ?? index, [visibleEvents]);
  const { itemRef, itemStart, isMeasured, totalSize, virtualItems, scrollToIndex, anchorItem, listScrollTop } = useVirtualList({
    count: visibleEvents.length,
    estimateSize: estimateMessageHeight,
    getItemKey: getMessageKey,
    overscan: 8,
    scrollElementRef: scrollContainerRef,
    enabled: visibleEvents.length > 40,
  });
  const isMeasuredRef = useRef(isMeasured);
  isMeasuredRef.current = isMeasured;
  const scrollToIndexRef = useRef(scrollToIndex);
  scrollToIndexRef.current = scrollToIndex;

  useEffect(() => {
    const el = scrollContainerRef.current;
    if (!el) return;

    const onScroll = () => {
      // A programmatic scroll never decides follow; the user's scrolling does.
      if (!programmaticScrollRef.current) {
        const remaining = el.scrollHeight - (el.scrollTop + el.clientHeight);
        setFollow(remaining <= FOLLOW_THRESHOLD_PX);
        schedulePositionSave();
      }
      // In list coordinates: mid-fling the rows may still carry a correction.
      if (listScrollTop() <= el.clientHeight * 2 && !prependScrollAnchorRef.current) {
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

    el.addEventListener("scroll", onScroll, { passive: true });
    return () => { el.removeEventListener("scroll", onScroll); };
  }, [listScrollTop, sessionId, setFollow, schedulePositionSave]);

  // A user gesture ends any programmatic scroll in flight, so the scroll
  // events it produces are the user's and decide follow. Without this, a user
  // who scrolls during a programmatic scroll would be ignored and pulled back.
  useReleaseOnElementInteraction(scrollContainerRef, () => {
    programmaticScrollRef.current = false;
    // The user has taken over: a saved place still waiting to fit is dropped.
    setRestoreTarget(null);
  });

  const scrollToBottom = useCallback(() => {
    setFollow(true);
    runProgrammaticScroll(() => {
      scrollContainerRef.current?.scrollTo({
        top: scrollContainerRef.current.scrollHeight,
        behavior: "smooth",
      });
    });
  }, [runProgrammaticScroll, setFollow]);

  // Search the entire session, rather than only the currently loaded window.
  // Highlighting still uses the returned ids against the bounded local window.
  useEffect(() => {
    const query = searchQuery.trim();
    if (!query) {
      setServerSearchMatches([]);
      setServerSearchReady(false);
      setSearchTruncated(false);
      setSearchHasMore(false);
      setSearchNextAfterSequence(0);
      setSearchError(undefined);
      return;
    }
    setServerSearchReady(false);
    setSearchError(undefined);
    setSearchHasMore(false);
    setSearchNextAfterSequence(0);
    const requestId = ++searchRequestId.current;
    let cancelled = false;
    const timer = window.setTimeout(() => {
      void searchConversation(sessionId, query, serverSearchOptions(searchOptions)).then((response) => {
        if (!cancelled && requestId === searchRequestId.current) {
          setServerSearchMatches(response.matches);
          setSearchTruncated(response.truncated);
          setSearchHasMore(response.hasMore ?? false);
          setSearchNextAfterSequence(response.nextAfterSequence ?? 0);
          setSearchError(response.error);
          setServerSearchReady(true);
        }
      }).catch(() => {
        if (!cancelled && requestId === searchRequestId.current) {
          setServerSearchMatches([]);
          setSearchTruncated(false);
          setSearchError("Search couldn't reach the conversation history. Check the connection and try again.");
          setServerSearchReady(true);
        }
      });
    }, 200);
    return () => {
      cancelled = true;
      searchRequestId.current += 1;
      window.clearTimeout(timer);
    };
  }, [searchQuery, searchOptions, sessionId]);

  const continueSearch = useCallback(async () => {
    const query = searchQuery.trim();
    if (!query || !searchHasMore || searchNextAfterSequence <= 0 || searchContinueLoading) return;
    const requestId = ++searchRequestId.current;
    setSearchContinueLoading(true);
    setSearchError(undefined);
    try {
      const response = await searchConversation(sessionId, query, { ...serverSearchOptions(searchOptions), afterSequence: searchNextAfterSequence });
      if (requestId !== searchRequestId.current) return;
      setServerSearchMatches((previous) => {
        const known = new Set(previous.map((match) => match.eventId));
        return [...previous, ...response.matches.filter((match) => !known.has(match.eventId))];
      });
      setSearchTruncated(response.truncated);
      const nextAfterSequence = response.nextAfterSequence ?? 0;
      setSearchHasMore((response.hasMore ?? false) && nextAfterSequence > searchNextAfterSequence);
      setSearchNextAfterSequence(nextAfterSequence);
    } catch {
      if (requestId === searchRequestId.current) setSearchError("More search results couldn't be loaded. Check the connection and try again.");
    } finally {
      if (requestId === searchRequestId.current) setSearchContinueLoading(false);
    }
  }, [searchContinueLoading, searchHasMore, searchNextAfterSequence, searchOptions, searchQuery, sessionId]);

  const searchMatchIds = useMemo(() => {
    // One search path: the server's hits. Nothing is matched client-side.
    if (!searchQuery || !serverSearchReady) return [];
    return serverSearchMatches.map((match) => match.eventId);
  }, [searchQuery, serverSearchMatches, serverSearchReady]);
  const searchMatchById = useMemo(
    () => new Map(serverSearchMatches.map((match) => [match.eventId, match])),
    [serverSearchMatches],
  );
  const searchMatchSet = useMemo(() => new Set(searchMatchIds), [searchMatchIds]);
  // Hits inside collapsed work (searching tool calls and context) mark the
  // work row that holds them, and are highlighted when it opens.
  const workHitSequences = useMemo(
    () => new Set(serverSearchMatches.filter((match) => !match.threadId && match.kind != null && !isConversationRow({ role: match.role, kind: match.kind })).map((match) => match.sequence)),
    [serverSearchMatches],
  );
  const isSearchHit = useCallback((event: ConversationEvent) => {
    if (searchMatchSet.has(event.id)) return true;
    if (!isWorkRow(event)) return false;
    for (const sequence of workHitSequences) if (coversSequence(event, sequence)) return true;
    return false;
  }, [searchMatchSet, workHitSequences]);

  // The result dropdown reads exactly the filters the list reads. With a query
  // it lists the server's whole-history hits; with an empty query it offers the
  // most recent messages from the loaded window so clicking the field is
  // immediately useful instead of a blank panel.
  const localSearchResults = useMemo(
    () => searchLoadedEvents(events, searchQuery, filters, searchOptions),
    [events, filters, searchOptions, searchQuery],
  );
  const searchResults = useMemo(() => {
    const navState = { ...filters, query: searchQuery };
    if (!searchQuery.trim()) {
      const loaded = buildResults(events, navState);
      return navState.sort === "oldest" ? loaded.slice(-30) : loaded.slice(0, 30);
    }
    if (serverSearchReady && !searchError) {
      const serverResults = buildSearchResults(serverSearchMatches, events, navState);
      return (serverResults.length > 0 ? serverResults : localSearchResults).slice(0, 100);
    }
    return localSearchResults.slice(0, 100);
  }, [events, filters, localSearchResults, searchError, searchQuery, serverSearchMatches, serverSearchReady]);

  useEffect(() => {
    setSearchActiveIndex(0);
  }, [searchQuery, searchResults.length]);

  const closeSearchResults = useCallback(() => { setSearchOpen(false); }, []);

  const openSearchResults = useCallback(() => {
    setSearchOpen(true);
  }, []);

  const eventIds = useMemo(() => visibleEvents.map((event) => event.id), [visibleEvents]);
  const eventIndexById = useMemo(
    () => new Map(eventIds.map((id, index) => [id, index])),
    [eventIds],
  );
  const allEventIndexById = useMemo(
    () => new Map(events.map((event, index) => [event.id, index] as const)),
    [events],
  );
  const searchMatchIndexById = useMemo(
    () => new Map(searchMatchIds.map((id, index) => [id, index])),
    [searchMatchIds],
  );

  const currentMatchIndex = useMemo(
    () => (focusedEventId ? (searchMatchIndexById.get(focusedEventId) ?? -1) : -1),
    [focusedEventId, searchMatchIndexById],
  );

  // useLayoutEffect runs after React has placed the prepended page but before
  // the browser paints it.  The former anchor is temporarily outside the
  // virtual window at this point, so use its new virtual index rather than a
  // DOM lookup. The virtualizer holds it in place, mid-fling without a
  // scrollTop write; later row measurement is anchored by the virtualizer too.
  useLayoutEffect(() => {
    const anchor = prependScrollAnchorRef.current;
    if (!anchor) return;
    const index = visibleEvents.findIndex((event) => event.id === anchor.eventId);
    if (index < 0) return;
    anchorItem(index, anchor.offsetFromViewportTop);
    prependScrollAnchorRef.current = null;
  }, [anchorItem, visibleEvents]);

  // Restore, step 1: a saved place outside the loaded window is paged in
  // before anything scrolls. A place that cannot be found is reported and the
  // list goes to the end once — never silently.
  useEffect(() => {
    if (!restoreTarget || viewState.kind !== "messages") return;
    const isLoaded = () => useConversationStore.getState().sessions[sessionId]?.events
      .some((event) => event.id === restoreTarget.topEventId) ?? false;
    if (isLoaded()) return;
    let cancelled = false;
    void loadConversationPageContaining(sessionId, restoreTarget.topSequence).then((loaded) => {
      if (cancelled || (loaded && isLoaded())) return;
      setRestoreTarget(null);
      showTransientNotice(t(strings.messagesPane.restoreFailed));
      setFollow(true);
      const el = scrollContainerRef.current;
      if (el) runProgrammaticScroll(() => { el.scrollTo({ top: el.scrollHeight }); });
    });
    return () => { cancelled = true; };
  }, [restoreTarget, runProgrammaticScroll, sessionId, setFollow, showTransientNotice, t, viewState.kind]);

  // Restore, step 2: once the saved message is in the list, put it back where
  // it was, before paint. Until its row is rendered it is placed from the
  // virtualizer's (estimated) geometry; once rendered it is corrected from where
  // the row actually is, and once more in the commit that applies the row's
  // measured height, so estimate errors cannot shift the place. A place near
  // the end can be cut short while rows below it still carry estimates. The
  // place stays pending until all of that holds, or the user takes over.
  useLayoutEffect(() => {
    if (!restoreTarget) return;
    const index = eventIndexById.get(restoreTarget.topEventId);
    const el = scrollContainerRef.current;
    if (index == null || !el) return;
    const row = el.querySelector<HTMLElement>(`[data-event-id="${restoreTarget.topEventId}"]`);
    const wanted = row
      ? el.scrollTop + (row.getBoundingClientRect().top - el.getBoundingClientRect().top) + restoreTarget.offsetPx
      : itemStart(index) + restoreTarget.offsetPx;
    if (Math.abs(el.scrollTop - wanted) > 1) runProgrammaticScroll(() => { el.scrollTop = wanted; });
    if (row && isMeasured(index) && el.scrollTop >= wanted - 1) setRestoreTarget(null);
  }, [eventIndexById, isMeasured, itemStart, restoreTarget, runProgrammaticScroll, totalSize, virtualItems]);

  // The follow rule. New content — an appended event, rows measuring taller,
  // or the state slot appearing or changing — moves the viewport only while
  // following, and then only to the end, once per change. While not following, appended events are counted
  // for the new-messages pill and the viewport is left alone.
  useLayoutEffect(() => {
    const lastId = events[events.length - 1]?.id ?? null;
    const previousLastId = lastEventIdRef.current;
    lastEventIdRef.current = lastId;
    let appended = 0;
    if (previousLastId != null && lastId !== previousLastId) {
      const previousIndex = allEventIndexById.get(previousLastId);
      appended = previousIndex == null ? 0 : events.slice(previousIndex + 1).filter(isConversationRow).length;
    }

    const el = scrollContainerRef.current;
    if (!el) return;
    if (followRef.current) {
      if (el.scrollHeight - el.clientHeight - el.scrollTop > 1) {
        runProgrammaticScroll(() => { el.scrollTo({ top: el.scrollHeight }); });
      }
    } else if (appended > 0) {
      setNewMessageCount((count) => count + appended);
    }
  }, [events, allEventIndexById, totalSize, stateSlotKey, echoes.length, runProgrammaticScroll]);

  const scrollToEvent = useCallback(async (selectedEventId: string) => {
    let eventId = selectedEventId;
    const fail = (message: string) => {
      setSearchError(message);
      setSearchOpen(true);
    };
    let index = eventIndexById.get(eventId);
    if (index == null) {
      const match = searchMatchById.get(eventId);
      if (!match) {
        fail("That message is no longer in the search results. Search again to refresh them.");
        return;
      }
      // A hit in a subagent's conversation opens that conversation.
      if (match.threadId) {
        setSearchOpen(false);
        setOpenThreadId(match.threadId);
        return;
      }
      const holder = (list: readonly ConversationEvent[]) => list.find((event) => event.id === eventId || (isWorkRow(event) && coversSequence(event, match.sequence)));
      const listed = () => {
        const session = useConversationStore.getState().sessions[sessionId];
        return holder(session?.events ?? []);
      };
      if (!listed() && !await loadConversationPageContaining(sessionId, match.sequence)) {
        fail("That message couldn't be loaded. Check the connection and try again.");
        return;
      }
      const target = listed();
      // A hit inside collapsed work opens the work row that holds it.
      if (target && isWorkRow(target)) {
        setExpandedWork((current) => new Set(current).add(target.id));
        setFocusedEventId(target.id);
      }
      index = target ? visibleEvents.findIndex((event) => event.id === target.id) : -1;
      if (index < 0 || !target) {
        fail("The selected message is hidden by the active filters. Clear a filter and try again.");
        return;
      }
      eventId = target.id;
    }
    setFollow(false);
    runProgrammaticScroll(() => { scrollToIndexRef.current(index!, "auto", "center"); });
    // First scroll uses estimated geometry to mount the target. Once the row
    // is rendered (and its size measured by the virtual list), align it again.
    for (let frame = 0; frame < 12; frame += 1) {
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
      const row = scrollContainerRef.current?.querySelector<HTMLElement>(`[data-testid="msg-card-${eventId}"]`);
      if (!row) continue;
      const currentIndex = eventIndexById.get(eventId) ?? -1;
      if (currentIndex < 0 || (visibleEvents.length > 40 && !isMeasuredRef.current(currentIndex))) continue;
      if (currentIndex >= 0) runProgrammaticScroll(() => { scrollToIndexRef.current(currentIndex, "auto", "center"); });
      return;
    }
    fail("The message loaded, but the list couldn't finish positioning it. Try selecting it again.");
  }, [eventIndexById, runProgrammaticScroll, searchMatchById, sessionId, setFollow, visibleEvents]);

  // Resolve selected stable IDs independently of the loaded window. Each new
  // selection invalidates older requests so a slow response cannot overwrite it.
  useEffect(() => {
    const ids = [...exportSelectedIds].filter((id) => !exportEventsById.has(id) && !allEventIndexById.has(id));
    if (!ids.length) return;
    const requestId = ++exportResolutionRequest.current;
    void resolveConversationEvents(sessionId, ids).then((result) => {
      if (requestId !== exportResolutionRequest.current) return;
      setExportEventsById((prev) => new Map([...prev, ...result.events.map((event) => [event.id, event] as const)]));
      setExportResolutionError(result.missingEventIds.length ? `${result.missingEventIds.length} selected message(s) no longer exist. Clear the selection and choose again.` : undefined);
    }).catch(() => {
      if (requestId === exportResolutionRequest.current) setExportResolutionError("Selected messages couldn't be loaded. Check the connection and try again.");
    });
    return () => { exportResolutionRequest.current += 1; };
  }, [allEventIndexById, exportEventsById, exportSelectedIds, sessionId]);

  // Selected events in conversation order for the drawer/formatter.
  const exportEvents = useMemo(() => {
    const loadedById = new Map(events.map((event) => [event.id, event]));
    return [...exportSelectedIds]
      .map((id) => exportEventsById.get(id) ?? loadedById.get(id))
      .filter((event): event is ConversationEvent => event !== undefined)
      .sort((left, right) => left.sequence - right.sequence);
  }, [events, exportEventsById, exportSelectedIds]);

  const selectConversationRange = useCallback((anchorEventId: string, eventId: string) => {
    const sequenceFor = (id: string) => exportEventsById.get(id)?.sequence
      ?? events.find((event) => event.id === id)?.sequence
      ?? searchMatchById.get(id)?.sequence;
    const anchorSequence = sequenceFor(anchorEventId);
    const endSequence = sequenceFor(eventId);
    if (anchorSequence == null || endSequence == null) {
      setExportResolutionError(t(strings.messageExport.rangeLoadFailed));
      return;
    }

    const requestId = ++exportResolutionRequest.current;
    setExportResolutionError(undefined);
    void getConversationRange(sessionId, Math.min(anchorSequence, endSequence), Math.max(anchorSequence, endSequence))
      .then((fullRange) => {
        if (requestId !== exportResolutionRequest.current) return;
        // An export is of the conversation the view lists, not the tool rows between.
        const range = { ...fullRange, events: fullRange.events.filter(isConversationRow) };
        const returnedIds = new Set(range.events.map((event) => event.id));
        if (!returnedIds.has(anchorEventId) || !returnedIds.has(eventId)) {
          setExportResolutionError(t(strings.messageExport.rangeLoadFailed));
          return;
        }
        setExportEventsById((previous) => new Map([
          ...previous,
          ...range.events.map((event) => [event.id, event] as const),
        ]));
        setExportSelectedIds((previous) => new Set([
          ...previous,
          ...range.events.map((event) => event.id),
        ]));
      })
      .catch(() => {
        if (requestId === exportResolutionRequest.current) {
          setExportResolutionError(t(strings.messageExport.rangeLoadFailed));
        }
      });
  }, [events, exportEventsById, searchMatchById, sessionId, t]);

  const exportSelection = useMemo<MessageExportSelection>(
    () => ({
      selectedIds: exportSelectedIds,
      selectedEvents: exportEvents,
      onToggle: (eventId) =>
        { setExportResolutionError(undefined); setExportSelectedIds((prev) => {
          const next = new Set(prev);
          if (next.has(eventId)) next.delete(eventId);
          else next.add(eventId);
          return next;
        }); },
      onSelectRange: selectConversationRange,
      onSelectAll: () => {
        const requestId = ++exportResolutionRequest.current;
        setExportResolutionError(undefined);
        void getAllConversationEvents(sessionId).then((allEvents) => {
          if (requestId !== exportResolutionRequest.current) return;
          setExportEventsById(new Map(allEvents.map((event) => [event.id, event])));
          setExportSelectedIds(new Set(allEvents.map((event) => event.id)));
        }).catch(() => {
          if (requestId === exportResolutionRequest.current) setExportResolutionError("Conversation history couldn't be fully loaded. Check the connection and try again.");
        });
      },
      onSelectVisible: (visibleIds) => { setExportSelectedIds(new Set(visibleIds)); },
      onClear: () => { exportResolutionRequest.current += 1; setExportResolutionError(undefined); setExportSelectedIds(new Set()); },
      onContinue: () => { setExportDrawerOpen(true); },
      canContinue: exportEvents.length === exportSelectedIds.size && !exportResolutionError,
      ...(exportResolutionError ? { error: exportResolutionError } : {}),
    }),
    [exportSelectedIds, sessionId, exportEvents, exportResolutionError, selectConversationRange],
  );

  const selectionTokenEstimate = useMemo(
    () => buildMessageExport(exportEvents, DEFAULT_MESSAGE_EXPORT_FORMAT).tokenEstimate,
    [exportEvents],
  );

  const enterSelection = useCallback(() => {
    setSearchOpen(false);
    selectionAnchorRef.current = null;
    setSelectionMode(true);
  }, []);

  const exitSelection = useCallback(() => {
    setSelectionMode(false);
    selectionAnchorRef.current = null;
    setExportResolutionError(undefined);
    setExportSelectedIds(new Set());
  }, []);

  const toggleSelected = useCallback((eventId: string, shiftKey: boolean) => {
    setExportResolutionError(undefined);
    if (shiftKey && selectionAnchorRef.current) {
      selectConversationRange(selectionAnchorRef.current, eventId);
      return;
    }
    selectionAnchorRef.current = eventId;
    setExportSelectedIds((previous) => {
      const next = new Set(previous);
      if (next.has(eventId)) next.delete(eventId);
      else next.add(eventId);
      return next;
    });
  }, [selectConversationRange]);

  const resetFilters = useCallback(() => {
    setFilters(DEFAULT_MESSAGE_FILTER_STATE);
    setSearchOptions({ mode: "text", caseSensitive: false, wholeWord: false });
  }, []);

  const focusAndScroll = useCallback((eventId: string) => {
    setFocusedEventId(eventId);
    void scrollToEvent(eventId);
  }, [scrollToEvent]);

  const handleSearchKeyDown = useCallback((keyEvent: React.KeyboardEvent<HTMLInputElement>) => {
    if (keyEvent.key === "Escape") {
      keyEvent.preventDefault();
      if (searchOpen) setSearchOpen(false);
      else setSearchQuery("");
      return;
    }
    if (!searchOpen) return;
    if (keyEvent.key === "ArrowDown") {
      keyEvent.preventDefault();
      if (searchResults.length > 0) setSearchActiveIndex((index) => (index + 1) % searchResults.length);
      return;
    }
    if (keyEvent.key === "ArrowUp") {
      keyEvent.preventDefault();
      if (searchResults.length > 0) setSearchActiveIndex((index) => (index - 1 + searchResults.length) % searchResults.length);
      return;
    }
    if (keyEvent.key === "Enter") {
      const result = searchResults[Math.min(Math.max(searchActiveIndex, 0), searchResults.length - 1)];
      if (!result) return;
      keyEvent.preventDefault();
      setSearchOpen(false);
      focusAndScroll(result.event.id);
    }
  }, [focusAndScroll, searchActiveIndex, searchOpen, searchResults]);

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

  useEffect(() => {
    if (!playbackFocusRequest) return;
    focusAndScroll(playbackFocusRequest.eventId);
  }, [focusAndScroll, playbackFocusRequest]);

  const handleMarkdownLinkClick = useCallback((href: string, event: React.MouseEvent<HTMLAnchorElement>) => {
    const proxiedUrl = rewriteLocalhostUrlForProxy(href, window.location.origin);
    if (proxiedUrl) {
      event.preventDefault();
      window.open(proxiedUrl, "_blank", "noopener,noreferrer");
      return;
    }
    if (!looksLikeFileReference(href)) return;
    event.preventDefault();
    onOpenFilePreview?.(href, "message_link");
  }, [onOpenFilePreview]);

  const handleInlineCodeFileClick = useCallback((path: string) => {
    onOpenFilePreview?.(path, "inline_code");
  }, [onOpenFilePreview]);

  const [mermaidViewer, setMermaidViewer] = useState<{ code: string } | null>(null);
  const handleMermaidOpen = useCallback((code: string) => {
    setMermaidViewer({ code });
  }, []);
  const closeMermaidViewer = useCallback(() => { setMermaidViewer(null); }, []);

  // --- Reader: a reply in full, in the pane in place of the list, which stays
  // laid out and live underneath. A live pane's reader outlives a tab switch
  // (the pane remounts), so it is kept in the store; the archive's is its own.
  const [archiveReader, setArchiveReader] = useState<OpenReader | null>(null);
  const storedReader = useMessagesViewStore((state) => (readOnly ? undefined : state.readers[sessionId]));
  const reader = readOnly ? archiveReader : storedReader ?? null;
  const setReader = useCallback((next: OpenReader | null) => {
    if (readOnly) setArchiveReader(next);
    else useMessagesViewStore.getState().setReader(sessionId, next);
  }, [readOnly, sessionId]);
  const readerEventId = reader?.eventId ?? null;
  const readerIndex = readerEventId == null ? undefined : eventIndexById.get(readerEventId);
  const readerEvent = readerIndex == null ? null : visibleEvents[readerIndex] ?? null;
  const readerShown = readerEvent != null;
  const keyboardOpen = useWorkspaceStore((state) => state.keyboardOpen);
  // Replies past everything the reader has shown arrived while it was open.
  const newReplyCount = reader == null
    ? 0
    : events.filter((candidate) => isAgentReply(candidate) && candidate.sequence > reader.seenThrough).length;
  // The reader steps between replies; the operator's own messages are skipped.
  const replies = useMemo(() => replyIndices(visibleEvents), [visibleEvents]);
  const canPrevReply = readerIndex != null && seekReply(replies, readerIndex, -1).moved > 0;
  const canNextReply = readerIndex != null && seekReply(replies, readerIndex, 1).moved > 0;
  const readerPosition = readerIndex == null ? { ordinal: 0, total: replies.length } : replyPosition(replies, readerIndex);
  const readerFontSize = messagesFontSize;
  const openReader = useCallback((eventId: string) => {
    if (!eventIndexById.has(eventId)) return;
    setReader({ eventId, seenThrough: events[events.length - 1]?.sequence ?? 0 });
  }, [eventIndexById, events, setReader]);
  // A held scrub can step several times between renders, so a seek starts
  // from the reader it last set, not from the one this render saw.
  const seekState = useRef({ reader, replies, visibleEvents, eventIndexById });
  seekState.current = { reader, replies, visibleEvents, eventIndexById };
  const seekReader = useCallback((delta: number): number => {
    const { reader: current, replies: track, visibleEvents: list, eventIndexById: indexById } = seekState.current;
    const from = current == null ? undefined : indexById.get(current.eventId);
    if (current == null || from == null) return 0;
    const { index, moved } = seekReply(track, from, delta);
    const target = index == null ? undefined : list[index];
    if (!target || moved === 0) return 0;
    const next = { eventId: target.id, seenThrough: Math.max(current.seenThrough, target.sequence) };
    seekState.current = { ...seekState.current, reader: next };
    setReader(next);
    return moved;
  }, [setReader]);
  // The list and its controls are out of reach while the reader covers them.
  const controlStripRef = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    for (const node of [controlStripRef.current, scrollContainerRef.current]) node?.toggleAttribute("inert", readerShown || openThread != null);
  }, [openThread, readerShown]);
  const closeReader = useCallback(() => {
    const eventId = readerEventId;
    setReader(null);
    // Return focus to the row the reader showed last; the list did not move.
    if (eventId) {
      requestAnimationFrame(() => {
        scrollContainerRef.current
          ?.querySelector<HTMLElement>(`[data-testid="msg-card-${eventId}"]`)
          ?.focus({ preventScroll: true });
      });
    }
  }, [readerEventId, setReader]);

  // --- Message actions: the pane owns which row's list is open (one at a time) ---
  const coarsePointer = useTouchControls();
  const [actionsTarget, setActionsTarget] = useState<{ eventId: string; origin: ActionsOrigin | null } | null>(null);
  const openActions = useCallback((eventId: string, origin: ActionsOrigin | null) => {
    setActionsTarget({ eventId, origin });
  }, []);
  const closeActions = useCallback(() => { setActionsTarget(null); }, []);
  // On touch a tap on a row reveals its inline actions, as hover does on a
  // fine pointer: one row at a time, and a second tap hides them. A tap that
  // lands on a control inside the row (a link, a revealed action) is that
  // control's, not a reveal. Long-press opens the full sheet; movement past
  // the gesture threshold cancels both, so scrolling never does either.
  const [tapRevealedId, setTapRevealedId] = useState<string | null>(null);
  const toggleTapReveal = useCallback((eventId: string) => {
    setTapRevealedId((current) => (current === eventId ? null : eventId));
  }, []);
  const { getGestureHandlers: getPressHandlers } = usePressGesture<string>({ onTap: toggleTapReveal, onLongPress: openActions });

  // Keyboard: j/k move between messages, Enter opens the focused message's
  // actions, Escape closes them. Typing in a field inside the pane is left alone.
  const handleListKeyDown = useCallback((keyEvent: React.KeyboardEvent<HTMLDivElement>) => {
    // The reader owns the keys while it covers the list.
    if (readerShown) return;
    const target = keyEvent.target as HTMLElement;
    if (target.closest("input, textarea, select, [contenteditable='true'], [role='menu']")) return;
    // Cmd/Ctrl+K: search this session.
    if ((keyEvent.metaKey || keyEvent.ctrlKey) && !keyEvent.altKey && keyEvent.key.toLowerCase() === "k") {
      keyEvent.preventDefault();
      setSearchOpen(true);
      searchInputRef.current?.focus();
      searchInputRef.current?.select();
      return;
    }
    if (keyEvent.metaKey || keyEvent.ctrlKey || keyEvent.altKey) return;
    if (keyEvent.key === "Escape" && selectionMode) {
      keyEvent.preventDefault();
      exitSelection();
      return;
    }
    if (keyEvent.key === "Escape" && actionsTarget) {
      closeActions();
      return;
    }
    if (keyEvent.key === "Enter" && focusedEventId) {
      const row = scrollContainerRef.current?.querySelector<HTMLElement>(`[data-testid="msg-card-${focusedEventId}"]`);
      const rect = row?.getBoundingClientRect();
      keyEvent.preventDefault();
      openActions(focusedEventId, rect ? { x: rect.right, y: rect.top } : null);
      return;
    }
    if (keyEvent.key !== "j" && keyEvent.key !== "k") return;
    const el = scrollContainerRef.current;
    const fromId = focusedEventId ?? (el ? captureTopPosition(el)?.topEventId : undefined);
    const from = fromId ? (eventIndexById.get(fromId) ?? -1) : -1;
    // With nothing focused yet, j lands on the top visible message itself.
    const step = keyEvent.key === "j" ? (focusedEventId ? 1 : 0) : -1;
    // j/k move between conversation rows; the work lines between them are stepped over.
    let next = Math.min(visibleEvents.length - 1, Math.max(0, from + step));
    const direction = step === 0 ? 1 : step;
    const isWorkAt = (position: number) => { const row = visibleEvents[position]; return row != null && isWorkRow(row); };
    while (isWorkAt(next) && next + direction >= 0 && next + direction < visibleEvents.length) next += direction;
    const nextId = visibleEvents[next]?.id;
    if (!nextId) return;
    keyEvent.preventDefault();
    focusAndScroll(nextId);
  }, [actionsTarget, closeActions, eventIndexById, exitSelection, focusAndScroll, focusedEventId, openActions, readerShown, selectionMode, visibleEvents]);

  // MessageRow's memo comparator compares the action context entry by entry,
  // so any closure rebuilt per render defeats it and re-renders every mounted
  // row on every scroll event. These two were the only always-changing
  // entries; both are stable now.
  const handleRewindRequest = useCallback((eventId: string) => {
    setRewindError(null);
    setRewindNotice(null);
    setRewindPreserveDraft(true);
    void preflightConversationControl(sessionId, eventId).then((preflight) => {
      setRewindOperation(preflight.operation || preflight.capability.options?.find((option) => option.default)?.id || "restore_conversation");
      setRewindPreflight(preflight);
    }).catch((error: unknown) => {
      setRewindError(error instanceof Error ? error.message : t(strings.messagesPane.rewindSafetyCheckFailed));
    });
  }, [sessionId, t]);

  const handleSaveAsSnippet = useCallback((event: ConversationEvent) => {
    setSnippetSaveSource({
      body: event.text,
      sourceLabel: t(strings.snippets.save.fromMessage, { session: sessionId, sequence: event.sequence }),
    });
  }, [sessionId, t]);

  // One action context per message, shared by its row and the reader.
  const actionContextFor = (event: ConversationEvent): MessageActionContext => ({
    event,
    sessionId,
    readOnly,
    copied: copiedEventId === event.id,
    isPlaintext: plaintextIds.has(event.id),
    isAudioLoading: event.role !== "user" && loadingEventId === event.id,
    isTtsSpeaking,
    activeSpeakingEventId,
    summarizeLevel,
    selectedVersion: selectedVersionForEvent(event),
    summarizingEventId,
    getSummarizeError,
    onClearSummarizeError,
    onToggleSummarized,
    onChangeLevel,
    onCopy: handleCopy,
    onPlayFromHere,
    onOpenReader: openReader,
    onToggleRenderMode: toggleRenderMode,
    onSendToComposer,
    onRewind: readOnly ? undefined : handleRewindRequest,
    onSaveAsSnippet: handleSaveAsSnippet,
    onHandoff: readOnly ? undefined : onHandoff,
  });

  return (
    <div
      data-testid={`messages-pane-${sessionId}`}
      aria-readonly={readOnly}
      onKeyDown={handleListKeyDown}
      className="relative flex h-full flex-col bg-wc-surface-base px-2 pb-4 select-text"
    >
      <div
        ref={controlStripRef}
        data-testid="messages-control-strip"
        className="z-wc-chrome flex h-[53px] shrink-0 items-center justify-start gap-1.5 border-b border-transparent bg-wc-surface-base/80 py-1.5 backdrop-blur-sm"
      >
        <div ref={searchFieldRef} className="min-w-0 flex-1">
          <SearchField
            ref={searchInputRef}
            testId="messages-search-field"
            groupTestId="messages-search-field-group"
            data-count={totalCount}
            value={searchQuery}
            onValueChange={(value) => {
              setSearchQuery(value);
              setFocusedEventId(null);
              setSearchOpen(true);
            }}
            placeholder={t(strings.messagesPane.searchField, { count: totalCount })}
            label={t(strings.messageJumpList.searchAriaLabel)}
            clearLabel={t(strings.messageJumpList.clearSearch)}
            onFocus={openSearchResults}
            onClick={openSearchResults}
            onBlur={(blurEvent) => {
              const next = blurEvent.relatedTarget as Node | null;
              if (next && (searchFieldRef.current?.contains(next) || document.getElementById("messages-search-results")?.contains(next))) return;
              setSearchOpen(false);
            }}
            onKeyDown={handleSearchKeyDown}
            shortcut="⌘K"
            role="combobox"
            aria-expanded={searchOpen}
            aria-controls="messages-search-results"
            aria-autocomplete="list"
            aria-activedescendant={searchOpen && searchResults.length > 0
              ? `messages-search-result-${searchResults[Math.min(Math.max(searchActiveIndex, 0), searchResults.length - 1)]?.event.id ?? ""}`
              : undefined}
            className="w-full"
          />
        </div>
        <MessagesFilterPopover
          filters={filters}
          onFiltersChange={setFilters}
          searchOptions={searchOptions}
          onSearchOptionsChange={setSearchOptions}
          sourceOptions={sourceOptions}
          activeCount={filterActiveCount}
          onReset={resetFilters}
        />
        <IconButton
          data-testid="messages-select-toggle"
          aria-label={selectionMode ? "Cancel message selection" : "Select messages"}
          title={selectionMode ? "Cancel selection" : "Select messages"}
          surface="soft"
          size="xs"
          shape="circle"
          selected={selectionMode}
          onClick={() => {
            if (selectionMode) exitSelection();
            else enterSelection();
          }}
        >
          <CheckSquare />
        </IconButton>
        {toolbarTrailingAction && (
          <div data-testid="messages-control-trailing" className="ms-auto flex items-center gap-1.5">
            {toolbarTrailingAction}
          </div>
        )}
      </div>

      <MessagesSearchResults
        open={searchOpen}
        anchorRef={searchFieldRef}
        onClose={closeSearchResults}
        query={searchQuery}
        results={searchResults}
        loading={Boolean(searchQuery.trim()) && !serverSearchReady && !searchError && localSearchResults.length === 0}
        searchingAll={Boolean(searchQuery.trim()) && !serverSearchReady && !searchError && localSearchResults.length > 0}
        {...(searchError ? { error: searchError } : {})}
        truncated={searchTruncated}
        hasMore={searchHasMore}
        loadingMore={searchContinueLoading}
        onLoadMore={continueSearch}
        activeIndex={searchActiveIndex}
        onActiveIndexChange={setSearchActiveIndex}
        onSelect={(eventId) => {
          setSearchOpen(false);
          focusAndScroll(eventId);
        }}
        totalCount={totalCount}
      />

      <MessageExportDrawer
        open={exportDrawerOpen}
        events={exportEvents}
        error={exportResolutionError}
        onClose={() => { setExportDrawerOpen(false); }}
        onHandoff={onHandoff && !readOnly ? (payload) => {
          setExportDrawerOpen(false);
          onHandoff(sessionId, payload);
        } : undefined}
      />


      <MessagesPaneStatusLine
        status={resolveMessagesPaneStatus({
          refreshError,
          liveInterrupted,
          liveInterruptedText: t(strings.messagesPane.liveDisconnected),
          transient: transientNotice,
        })}
      />

        <div
          ref={scrollContainerRef}
        data-testid="messages-scroll"
        data-follow={String(following)}
        tabIndex={0}
        aria-label={t(strings.messagesPane.listLabel)}
        // Browser scroll anchoring is off: the virtualizer's resize
        // compensation is the single mechanism that keeps rows still, and two
        // mechanisms would each correct the same shift.
        className={cn(
          "relative min-h-0 flex-1 overflow-auto [overflow-anchor:none]",
          selectionMode && "pb-24",
        )}
      >
        {viewState.kind !== "messages" ? (
          <MessagesPaneState view={viewState} onRetry={() => void handleRefresh()} />
        ) : visibleEvents.length === 0 ? (
          <PaneEmptyState
            testId="messages-filter-empty"
            icon={<SlidersHorizontal />}
            title="No messages match these filters"
            description="Clear the filters to see the full conversation again."
            actions={(
              <Button type="button" variant="secondary" size="sm" onClick={resetFilters}>
                Clear filters
              </Button>
            )}
          />
        ) : (
          <div className="relative" style={{ height: `${String(totalSize)}px` }}>
            {virtualItems.map(({ index, start }) => {
              const event = visibleEvents[index];
              if (!event) return null;
              return (
                // The wrapper, not the article, is measured: it also holds the
                // row's handoff chips, and a slot sized without them lets the
                // next row overlap this one.
                <div
                  key={event.id}
                  ref={itemRef(index)}
                  data-event-id={event.id}
                  data-sequence={event.sequence}
                  className="absolute left-0 right-0"
                  style={{ top: `${String(start)}px` }}
                >
                  {isWorkRow(event) ? (
                    <WorkSpanRow
                      sessionId={sessionId}
                      event={event}
                      previousAt={conversationTimeAt(visibleEvents, index, -1)}
                      nextAt={conversationTimeAt(visibleEvents, index, 1)}
                      expanded={expandedWork.has(event.id)}
                      onToggle={toggleWork}
                      highlightSequences={workHitSequences}
                    />
                  ) : event.kind === "notice" || event.kind === "attachment" ? (
                    <NoticeRow
                      event={event}
                      isDimmed={!!searchQuery && serverSearchReady && !isSearchHit(event)}
                      isSearchFocused={searchMatchSet.has(event.id) && currentMatchIndex >= 0 && searchMatchIds[currentMatchIndex] === event.id}
                    />
                  ) : (
                    <MessageRow
                      imageSessionId={sessionId}
                      actionContext={actionContextFor(event)}
                      fontSize={renderedMessagesFontSize}
                      isFocused={focusedEventId === event.id}
                      isSearchFocused={searchMatchSet.has(event.id) && currentMatchIndex >= 0 && searchMatchIds[currentMatchIndex] === event.id}
                      isDimmed={!!searchQuery && serverSearchReady && !isSearchHit(event)}
                      selectionMode={selectionMode}
                      selected={exportSelectedIds.has(event.id)}
                      onToggleSelected={toggleSelected}
                      coarsePointer={coarsePointer}
                      tapRevealed={coarsePointer && tapRevealedId === event.id}
                      actionsOpen={actionsTarget?.eventId === event.id}
                      actionsOrigin={actionsTarget?.eventId === event.id ? actionsTarget.origin : null}
                      onOpenActions={openActions}
                      onCloseActions={closeActions}
                      getPressHandlers={getPressHandlers}
                      onLinkClick={handleMarkdownLinkClick}
                      onFileReferenceClick={handleInlineCodeFileClick}
                      onMermaidOpen={handleMermaidOpen}
                    />
                  )}
                  {/* An agent thread that began inside this span is one group here. */}
                  {(threadsByAnchor.get(event.id) ?? NO_THREAD_GROUPS).map((thread) => (
                    <ThreadGroupRow key={thread.threadId} thread={thread} onOpen={setOpenThreadId} />
                  ))}
                  {/* Suggestions render INSIDE the message's own block, so
                      offering one never moves the transcript the operator is
                      reading. */}
                  {onHandoff && groupSuggestionsByRule(handoffSuggestions.forEvent(event.id)).map((group) => (
                    <HandoffSuggestionChip
                      key={group[0]?.ruleId ?? ""}
                      suggestions={group}
                      onOpen={(payload) => { onHandoff(sessionId, payload); }}
                      onDismiss={handoffSuggestions.dismiss}
                    />
                  ))}
                </div>
              );
            })}
          </div>
        )}
        {/* Outside the virtual list: the slot is live state, not a message,
            and it never takes part in row measurement or compensation. */}
        {echoes.map((echo) => (
          <EchoRow
            key={echo.id}
            echo={echo}
            onPressEnter={handlePressEnter}
            onOpenTerminal={onOpenTerminal}
            onExpire={expireEcho}
          />
        ))}
        <SessionStateSlot slot={stateSlot} onOpenTerminal={onOpenTerminal} onAnswer={answerSlotPrompt} />
      </div>

      {selectionMode && (
        <div
          data-testid="messages-selection-bar"
          className="pointer-events-none absolute inset-x-3 bottom-[max(0.75rem,var(--wc-safe-bottom,0px))] z-wc-chrome-raised flex justify-center"
        >
          <div className="pointer-events-auto flex max-w-full flex-wrap items-center gap-1.5 rounded-full border border-wc-default bg-wc-surface-raised/95 px-2 py-1.5 shadow-xl backdrop-blur-md">
            <span className="px-2 text-xs text-wc-text-secondary">
              <b className="text-wc-text-primary">{exportSelection.selectedIds.size}</b> selected
              <span className="ms-1 text-[10px] text-wc-text-faint">· ~{selectionTokenEstimate.toLocaleString()} tokens</span>
            </span>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              data-testid="messages-select-all"
              onClick={() => { exportSelection.onSelectAll(); }}
            >
              Select all
            </Button>
            {exportSelection.selectedIds.size > 0 && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                data-testid="messages-select-clear"
                onClick={() => { exportSelection.onClear(); selectionAnchorRef.current = null; }}
              >
                Clear
              </Button>
            )}
            <Button
              type="button"
              variant="secondary"
              size="sm"
              data-testid="messages-select-cancel"
              onClick={exitSelection}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant={exportSelection.selectedIds.size > 0 ? "primary" : "secondary"}
              size="sm"
              data-testid="messages-select-export"
              disabled={exportSelection.selectedIds.size === 0 || !exportSelection.canContinue}
              className={exportSelection.selectedIds.size === 0 ? "opacity-60" : undefined}
              onClick={() => { exportSelection.onContinue(); }}
            >
              Export
            </Button>
          </div>
        </div>
      )}
      {exportResolutionError && selectionMode && (
        <div
          role="alert"
          data-testid="messages-selection-error"
          className="pointer-events-none absolute inset-x-3 bottom-24 z-wc-chrome-raised flex justify-center"
        >
          <div className="pointer-events-auto rounded-lg bg-red-500/10 px-3 py-2 text-[11px] text-red-300">{exportResolutionError}</div>
        </div>
      )}

      {/* Centred by a full-width row, not a transform: the library button
          resets `transform` on hover and press, which slid it sideways. */}
      {!readerShown && (newMessageCount > 0 || (!following && visibleEvents.length > 0)) && (
        <div
          data-testid="msg-bottom-actions"
          className="pointer-events-none absolute inset-x-0 bottom-[max(1rem,var(--wc-safe-bottom,0px))] z-wc-chrome-raised flex justify-center"
        >
          {newMessageCount > 0 ? (
            <button
              data-testid="msg-new-pill"
              data-count={newMessageCount}
              onClick={scrollToBottom}
              className="pointer-events-auto rounded-full border border-wc-default bg-wc-surface-raised px-4 py-2 text-xs font-medium text-wc-text-primary shadow-lg backdrop-blur-sm transition-colors hover:bg-wc-surface-input"
              type="button"
            >
              <ArrowDown className="me-1.5 inline-block h-3.5 w-3.5" />
              {t(strings.messagesPane.newMessages, { count: newMessageCount })}
            </button>
          ) : (
            <IconButton
              data-testid="msg-jump-bottom"
              aria-label={t(strings.messagesPane.jumpToBottomAria)}
              onClick={scrollToBottom}
              surface="soft"
              className="pointer-events-auto shadow-lg"
            >
              <ArrowDown />
            </IconButton>
          )}
        </div>
      )}

      {rewindPreflight && (
        <div role="dialog" aria-modal="true" aria-label={t(strings.messagesPane.rewindDialogLabel)} className="absolute inset-x-3 bottom-3 z-wc-chrome-raised max-h-[min(32rem,calc(100%-1.5rem))] overflow-y-auto rounded-lg border border-wc-default bg-wc-surface-raised p-4 shadow-xl">
          <h2 className="font-semibold text-wc-text-primary">{t(strings.messagesPane.rewindHeading, { sequence: rewindPreflight.sequence })}</h2>
          <p className="mt-1 break-words text-sm text-wc-text-secondary">{t(strings.messagesPane.rewindNativeTarget, {
            provider: rewindPreflight.capability.provider,
            launchMode: rewindPreflight.capability.launchMode || "unknown",
            controlMode: rewindPreflight.capability.controlMode || "unknown",
          })}</p>
          {(rewindPreflight.capability.options?.filter((option) => option.supported).length ?? 0) > 0 && (
            <label className="mt-3 block text-sm text-wc-text-secondary">
              <span className="mb-1 block font-medium text-wc-text-primary">{t(strings.messagesPane.rewindOperationLabel)}</span>
              <select value={rewindOperation} disabled={rewindBusy} className="w-full rounded border border-wc-default bg-wc-surface px-2 py-2" onChange={(event) => {
                const operation = event.target.value;
                setRewindOperation(operation);
                setRewindError(null);
                void preflightConversationControl(sessionId, rewindPreflight.eventId, operation).then((preflight) => {
                  setRewindOperation(preflight.operation || operation);
                  setRewindPreflight(preflight);
                }).catch((error: unknown) => {
                  setRewindError(error instanceof Error ? error.message : t(strings.messagesPane.rewindSafetyCheckFailed));
                });
              }}>
                {rewindPreflight.capability.options?.filter((option) => option.supported).map((option) => <option key={option.id} value={option.id}>{option.label}</option>)}
              </select>
            </label>
          )}
          <ul className="my-3 list-disc ps-5 text-sm text-wc-text-secondary">
            {rewindPreflight.consequences.map((consequence) => <li key={consequence}>{consequence}</li>)}
          </ul>
          {!rewindPreflight.capability.available && (
            <p className="mb-3 text-sm text-wc-text-secondary">{t(strings.messagesPane.rewindUnavailable)}</p>
          )}
          {rewindPreflight.capability.available && (
            <label className="mb-3 flex items-center gap-2 text-sm text-wc-text-secondary">
              <input type="checkbox" checked={rewindPreserveDraft} onChange={(event) => { setRewindPreserveDraft(event.target.checked); }} />
              {t(strings.messagesPane.rewindPreserveDraft)}
            </label>
          )}
          {rewindBusy && <p role="status" aria-live="polite" className="mb-3 text-sm text-wc-text-secondary">{t(strings.messagesPane.rewindBusy)}</p>}
          <div className="flex justify-end gap-2">
            <button type="button" disabled={rewindBusy} className="rounded border border-wc-default px-3 py-2 text-sm disabled:opacity-50" onClick={() => { setRewindPreflight(null); }}>{t(strings.messagesPane.rewindCancel)}</button>
            <button type="button" disabled={rewindBusy} className="rounded bg-wc-accent px-3 py-2 text-sm text-wc-accent-fg disabled:opacity-50" onClick={() => { onOpenTerminal?.(); setRewindPreflight(null); }}>{t(strings.messagesPane.rewindOpenTerminal)}</button>
            {rewindPreflight.capability.available && rewindPreflight.confirmationKey && (
              <button type="button" disabled={rewindBusy} className="rounded bg-wc-danger px-3 py-2 text-sm text-white disabled:opacity-50" onClick={() => {
                setRewindBusy(true);
                void executeConversationControl(sessionId, { operationId: rewindPreflight.operationId, eventId: rewindPreflight.eventId, operation: rewindOperation, confirmationKey: rewindPreflight.confirmationKey ?? "", preserveDraft: rewindPreserveDraft }).then((outcome) => {
                  if (outcome.state === "succeeded") setRewindNotice(outcome.detail);
                  else setRewindError(outcome.detail);
                }).catch((error: unknown) => { setRewindError(error instanceof Error ? error.message : "Native rewind failed safely"); }).finally(() => { setRewindBusy(false); setRewindPreflight(null); });
              }}>{t(strings.messagesPane.rewindConfirm)}</button>
            )}
          </div>
        </div>
      )}
      {rewindError && <div role="alert" className="absolute inset-x-3 bottom-3 z-wc-chrome-raised rounded border border-red-500/50 bg-wc-surface-raised p-3 text-sm text-red-300">{rewindError}</div>}
      {rewindNotice && <div role="status" aria-live="polite" className="absolute inset-x-3 bottom-3 z-wc-chrome-raised rounded border border-emerald-500/50 bg-wc-surface-raised p-3 text-sm text-emerald-300">{rewindNotice}</div>}

      {readerEvent && (
        <MessagesReader
          imageSessionId={sessionId}
          actionContext={{ ...actionContextFor(readerEvent), onOpenReader: undefined, onPlayMessage: readOnly ? undefined : onPlayEvent }}
          fontSize={readerFontSize}
          onFontSizeChange={setMessagesFontSize}
          coarsePointer={coarsePointer}
          hideFooter={coarsePointer && keyboardOpen}
          onClose={closeReader}
          onSeek={seekReader}
          canPrev={canPrevReply}
          canNext={canNextReply}
          position={readerPosition}
          newReplyCount={newReplyCount}
          onLinkClick={handleMarkdownLinkClick}
          onFileReferenceClick={handleInlineCodeFileClick}
          onMermaidOpen={handleMermaidOpen}
          pendingOutbox={pendingOutbox}
          onPressEnter={pendingOutbox ? handlePressEnter : undefined}
        />
      )}

      {openThread && !readerEvent && (
        <ThreadView
          sessionId={sessionId}
          thread={openThread}
          fontSize={renderedMessagesFontSize}
          onClose={() => { setOpenThreadId(null); }}
          onLinkClick={handleMarkdownLinkClick}
          onFileReferenceClick={handleInlineCodeFileClick}
          onMermaidOpen={handleMermaidOpen}
        />
      )}

      <MessagesMermaidViewer
        open={mermaidViewer !== null}
        code={mermaidViewer?.code ?? ""}
        onClose={closeMermaidViewer}
      />

      <SnippetSaveSheet
        open={snippetSaveSource !== null}
        onClose={() => { setSnippetSaveSource(null); }}
        mode="create"
        initialBody={snippetSaveSource?.body ?? ""}
        sourceLabel={snippetSaveSource?.sourceLabel}
      />
    </div>
  );
}
