import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { ArrowDown, Search } from "lucide-react";
import { useConversationStore, getSessionConversationEvents, getSessionSlice, resolveConversationView } from "../stores/useConversationStore";
import { loadConversationPageContaining, loadOlderConversationPage, refreshConversationSession } from "../hooks/useConversationSession";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { MESSAGES_FONT_DEFAULT, useMessagesViewStore, type OpenReader } from "../stores/useMessagesViewStore";
import { clampMessagesFont, useMessagesTypographyPinch } from "../hooks/useMessagesTypographyPinch";
import { useTrailingThrottle } from "../hooks/useTrailingThrottle";
import { captureTopPosition } from "./messages/scrollPosition";
import { useLiveStreamNotice } from "../hooks/useLiveStreamNotice";
import { writeText } from "../lib/clipboard";
import { executeConversationControl, getConversationRange, preflightConversationControl, searchConversation, type ConversationControlPreflight, type ConversationEvent, type ConversationSearchMatch } from "../api/conversation";
import { useFilePreviewController } from "./file-preview/useFilePreviewController";
import { TERMINAL_FONT_SIZE } from "../consts/config";
import { cn } from "../lib/classnames";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { looksLikeFileReference } from "../lib/fileReferences";
import { useVirtualList } from "../hooks/useVirtualList";
import { useReleaseOnElementInteraction } from "../hooks/useKeyboardListeners";
import MessageJumpList, { type MessageExportSelection, type SearchOptionsState } from "./MessageJumpList";
import MessageExportDrawer from "./MessageExportDrawer";
import type { SummarizationLevel } from "./tts/PlaybackModeControl";
import type { PlaybackFocusRequest, PlaybackVersion } from "../domains/tts-playback/types";
import MessagesFileViewer from "./MessagesFileViewer";
import HandoffSuggestionChip from "./handoff/HandoffSuggestionChip";
import { groupSuggestionsByRule } from "../lib/captureRules";
import { useHandoffSuggestions } from "../hooks/useHandoffSuggestions";
import MessagesMermaidViewer from "./MessagesMermaidViewer";
import MessagesPaneState from "./MessagesPaneState";
import MessagesPaneStatusLine from "./MessagesPaneStatusLine";
import { resolveMessagesPaneStatus } from "../lib/messagesPaneStatus";
import { SnippetSaveSheet } from "./snippets/SnippetSaveSheet";
import { MessageRow, type PressHandlers } from "./messages/MessageRow";
import { MessagesReader } from "./messages/MessagesReader";
import type { MessageActionContext } from "./messages/messageActions";
import { holdKeyboardForNextField } from "../lib/keyboardFocus";
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
  /** This session's current terminal screen as text (read from the server), or null. */
  getTerminalText?: () => Promise<string | null>;
  /** Presses Enter in this pane's terminal (an echo row's "Press Enter"). */
  onPressEnter?: () => void;
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
  getTerminalText,
  onPressEnter,
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

  const messagesFontSize = clampMessagesFont(useMessagesViewStore((state) => state.messagesFontSize ?? MESSAGES_FONT_DEFAULT));
  const setMessagesFontSize = useCallback((size: number) => {
    useMessagesViewStore.getState().setMessagesFontSize(clampMessagesFont(size));
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

  // --- Search & navigation ---
  // The navigator panel owns search editing, but the query is lifted here so
  // the message list keeps dimming non-matches and the toolbar match-stepping
  // arrows keep working while (and after) the navigator is open.
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedEventId, setFocusedEventId] = useState<string | null>(null);
  const [serverSearchMatches, setServerSearchMatches] = useState<ConversationSearchMatch[]>([]);
  const [serverSearchReady, setServerSearchReady] = useState(false);
  const [searchTruncated, setSearchTruncated] = useState(false);
  const [searchError, setSearchError] = useState<string | undefined>(undefined);
  const [searchOptions, setSearchOptions] = useState<SearchOptionsState>({ mode: "text", caseSensitive: false, wholeWord: false });

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

  // --- Scroll: follow + prepend anchor ---
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const listPinchSize = useMessagesTypographyPinch(scrollContainerRef, messagesFontSize, setMessagesFontSize);
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
    if (result.answer) useConversationStore.getState().addEcho(sessionId, result.answer);
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
    const event = events[index];
    if (!event) return 140;
    const lineEstimate = Math.ceil(event.text.length / 90);
    return Math.max(110, Math.min(520, 72 + lineEstimate * 22));
  }, [events]);
  const getMessageKey = useCallback((index: number) => events[index]?.id ?? index, [events]);
  const { itemRef, itemStart, isMeasured, totalSize, virtualItems, scrollToIndex, anchorItem, listScrollTop } = useVirtualList({
    count: events.length,
    estimateSize: estimateMessageHeight,
    getItemKey: getMessageKey,
    overscan: 8,
    scrollElementRef: scrollContainerRef,
    enabled: events.length > 40,
  });

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
      setSearchError(undefined);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      void searchConversation(sessionId, query, searchOptions).then((response) => {
        if (!cancelled) {
          setServerSearchMatches(response.matches);
          setSearchTruncated(response.truncated);
          setSearchError(response.error);
          setServerSearchReady(true);
        }
      }).catch(() => {
        if (!cancelled) {
          setServerSearchMatches([]);
          setSearchTruncated(false);
          setSearchError(undefined);
          setServerSearchReady(false);
        }
      });
    }, 200);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [searchQuery, searchOptions, sessionId]);

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
  const eventIds = useMemo(() => events.map((event) => event.id), [events]);
  const eventIndexById = useMemo(
    () => new Map(eventIds.map((id, index) => [id, index])),
    [eventIds],
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
    const index = events.findIndex((event) => event.id === anchor.eventId);
    if (index < 0) return;
    anchorItem(index, anchor.offsetFromViewportTop);
    prependScrollAnchorRef.current = null;
  }, [anchorItem, events]);

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
      const previousIndex = eventIndexById.get(previousLastId);
      appended = previousIndex == null ? 0 : events.length - 1 - previousIndex;
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
  }, [events, eventIndexById, totalSize, stateSlotKey, echoes.length, runProgrammaticScroll]);

  const scrollToEvent = useCallback(async (eventId: string) => {
    const index = eventIndexById.get(eventId);
    if (index == null) {
      const match = searchMatchById.get(eventId);
      if (!match || !await loadConversationPageContaining(sessionId, match.sequence)) return;
      requestAnimationFrame(() => {
        const loadedIndex = useConversationStore.getState().sessions[sessionId]?.events.findIndex((event) => event.id === eventId) ?? -1;
        if (loadedIndex >= 0) runProgrammaticScroll(() => { scrollToIndex(loadedIndex, "auto", "center"); });
      });
      setFollow(false);
      return;
    }
    // Jumping to a message is the user choosing where to read: stop following.
    setFollow(false);
    runProgrammaticScroll(() => { scrollToIndex(index, "smooth", "center"); });
  }, [eventIndexById, runProgrammaticScroll, scrollToIndex, searchMatchById, sessionId, setFollow]);

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
    void scrollToEvent(eventId);
  }, [scrollToEvent]);

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
  const readerEvent = readerIndex == null ? null : events[readerIndex] ?? null;
  const readerShown = readerEvent != null;
  const keyboardOpen = useWorkspaceStore((state) => state.keyboardOpen);
  // Replies past everything the reader has shown arrived while it was open.
  const newReplyCount = reader == null
    ? 0
    : events.filter((candidate) => candidate.role !== "user" && candidate.sequence > reader.seenThrough).length;
  // The reader steps between replies; the operator's own messages are skipped.
  const replyNear = (from: number, direction: -1 | 1): string | null => {
    for (let index = from + direction; index >= 0 && index < events.length; index += direction) {
      const candidate = events[index];
      if (candidate && candidate.role !== "user") return candidate.id;
    }
    return null;
  };
  const prevReplyId = readerIndex == null ? null : replyNear(readerIndex, -1);
  const nextReplyId = readerIndex == null ? null : replyNear(readerIndex, 1);
  const readerFontSize = messagesFontSize;
  const openReader = useCallback((eventId: string) => {
    if (!eventIndexById.has(eventId)) return;
    setReader({ eventId, seenThrough: events[events.length - 1]?.sequence ?? 0 });
  }, [eventIndexById, events, setReader]);
  const stepReader = (eventId: string) => {
    const target = events[eventIndexById.get(eventId) ?? -1];
    if (!reader || !target) return;
    setReader({ eventId, seenThrough: Math.max(reader.seenThrough, target.sequence) });
  };
  // The list and its controls are out of reach while the reader covers them.
  const controlStripRef = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    for (const node of [controlStripRef.current, scrollContainerRef.current]) node?.toggleAttribute("inert", readerShown);
  }, [readerShown]);
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
  const tapOnControlRef = useRef(false);
  const toggleTapReveal = useCallback((eventId: string) => {
    if (tapOnControlRef.current) return;
    setTapRevealedId((current) => (current === eventId ? null : eventId));
  }, []);
  const { getGestureHandlers } = usePressGesture<string>({ onTap: toggleTapReveal, onLongPress: openActions });
  const getPressHandlers = useCallback((eventId: string): PressHandlers => {
    const handlers = getGestureHandlers(eventId);
    return {
      ...handlers,
      onPointerDown: (pointerEvent) => {
        tapOnControlRef.current = pointerEvent.target instanceof Element
          && pointerEvent.target.closest("a, button, input, textarea, select, summary, [role='button']") !== null;
        handlers.onPointerDown(pointerEvent);
      },
    };
  }, [getGestureHandlers]);

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
      openNavigator("search");
      return;
    }
    if (keyEvent.metaKey || keyEvent.ctrlKey || keyEvent.altKey) return;
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
    const nextId = events[Math.min(events.length - 1, Math.max(0, from + step))]?.id;
    if (!nextId) return;
    keyEvent.preventDefault();
    focusAndScroll(nextId);
  }, [actionsTarget, closeActions, eventIndexById, events, focusAndScroll, focusedEventId, openActions, openNavigator, readerShown]);

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

  // One stable handler per event id. The label needs the event's sequence, so
  // the closure is per-event; caching it keeps its identity stable across
  // renders of the same event.
  const snippetHandlersRef = useRef(new Map<string, (text: string) => void>());
  useEffect(() => {
    snippetHandlersRef.current.clear();
  }, [sessionId, t]);
  const saveAsSnippetHandlerFor = useCallback((event: ConversationEvent) => {
    const existing = snippetHandlersRef.current.get(event.id);
    if (existing) return existing;
    const handler = (text: string) => {
      setSnippetSaveSource({
        body: text,
        sourceLabel: t(strings.snippets.save.fromMessage, { session: sessionId, sequence: event.sequence }),
      });
    };
    snippetHandlersRef.current.set(event.id, handler);
    return handler;
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
    onSaveAsSnippet: saveAsSnippetHandlerFor(event),
    onHandoff: readOnly ? undefined : onHandoff,
  });

  return (
    <div
      data-testid={`messages-pane-${sessionId}`}
      aria-readonly={readOnly}
      onKeyDown={handleListKeyDown}
      className="relative flex h-full flex-col bg-wc-surface-base px-2 pb-4 pt-1 select-text"
    >
      <div
        ref={controlStripRef}
        data-testid="messages-control-strip"
        className="z-wc-chrome flex items-center justify-start gap-1.5 bg-wc-surface-base/80 py-1.5 backdrop-blur-sm"
      >
        <button
          type="button"
          data-testid="messages-search-field"
          data-count={totalCount}
          onClick={() => {
            // The search box mounts a commit later, inside the sheet's portal.
            if (coarsePointer) holdKeyboardForNextField();
            openNavigator("search");
          }}
          className={cn(
            "flex min-h-11 min-w-0 flex-1 items-center gap-2 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-start text-sm text-wc-text-faint transition hover:border-wc-accent/40",
            searchQuery && "text-wc-text-primary ring-1 ring-wc-accent/40",
          )}
        >
          <Search className="h-3.5 w-3.5 shrink-0" />
          <span className="truncate">{searchQuery || t(strings.messagesPane.searchField, { count: totalCount })}</span>
          <kbd className="ms-auto hidden font-mono text-[10px] text-wc-text-faint sm:inline">⌘K</kbd>
        </button>
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
          initialFocus={navInitialFocus}
          query={searchQuery}
          onQueryChange={handleNavQueryChange}
          search={{
            hits: serverSearchReady ? serverSearchMatches : null,
            truncated: searchTruncated,
            ...(searchError ? { error: searchError } : {}),
            options: searchOptions,
            onOptionsChange: setSearchOptions,
          }}
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

        <div
          ref={scrollContainerRef}
        data-testid="messages-scroll"
        data-follow={String(following)}
        tabIndex={0}
        aria-label={t(strings.messagesPane.listLabel)}
        // Browser scroll anchoring is off: the virtualizer's resize
        // compensation is the single mechanism that keeps rows still, and two
        // mechanisms would each correct the same shift.
        className="relative min-h-0 flex-1 overflow-auto [overflow-anchor:none]"
      >
        {viewState.kind !== "messages" ? (
          <MessagesPaneState view={viewState} onRetry={() => void handleRefresh()} />
        ) : (
          <div className="relative" style={{ height: `${String(totalSize)}px` }}>
            {virtualItems.map(({ index, start }) => {
              const event = events[index];
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
                  <MessageRow
                    actionContext={actionContextFor(event)}
                    fontSize={renderedMessagesFontSize}
                    isFocused={focusedEventId === event.id}
                    isSearchFocused={searchMatchSet.has(event.id) && currentMatchIndex >= 0 && searchMatchIds[currentMatchIndex] === event.id}
                    isDimmed={!!searchQuery && serverSearchReady && !searchMatchSet.has(event.id)}
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
            getTerminalText={getTerminalText}
            onPressEnter={onPressEnter}
            onOpenTerminal={onOpenTerminal}
            onExpire={expireEcho}
          />
        ))}
        <SessionStateSlot slot={stateSlot} onOpenTerminal={onOpenTerminal} onAnswer={answerSlotPrompt} />
      </div>

      {/* Centred by a full-width row, not a transform: the library button
          resets `transform` on hover and press, which slid it sideways. */}
      {!readerShown && (newMessageCount > 0 || (!following && events.length > 0)) && (
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
          actionContext={{ ...actionContextFor(readerEvent), onOpenReader: undefined, onPlayMessage: readOnly ? undefined : onPlayEvent }}
          fontSize={readerFontSize}
          onFontSizeChange={setMessagesFontSize}
          coarsePointer={coarsePointer}
          hideFooter={coarsePointer && keyboardOpen}
          onClose={closeReader}
          onPrev={prevReplyId ? () => { stepReader(prevReplyId); } : undefined}
          onNext={nextReplyId ? () => { stepReader(nextReplyId); } : undefined}
          newReplyCount={newReplyCount}
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
