import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react";
import { getTTSSummarizeConfig, updateTTSSummarizeConfig } from "../../api/tts";
import { summarizeEvent, type ConversationEvent } from "../../api/conversation";
import type { SummarizationLevel } from "../../components/tts/PlaybackModeControl";
import type {
  PlaybackEventContext,
  PlaybackFocusRequest,
  PlaybackQueueEntry,
  PlaybackVersion,
  SessionPlaybackAudioState,
  SessionPlaybackController,
  SessionPlaybackControllerState,
} from "./types";
import {
  buildPlaybackContext,
  playbackEventKey,
  resolvePlaybackParagraphs,
  resolvePlaybackVersion,
} from "./utils";
import {
  buildPlayNextEvent,
  initialPlaybackTransportState,
  transitionPlaybackTransport,
  type PlaybackTransportEvent,
  type PlaybackTransportState,
} from "./flow/transition";

// AI_CHECK: web_console_tts_playback_unification=3 | LAST: 2026-05-13
//
// Transport state (loading/playing/paused/idle/error + queue) is owned by the
// PlaybackTransport state machine in ./flow/transition. The non-transport
// surfaces — version preference, summarization status, focus requests, replay
// dismissal — remain plain hook state because they are not temporal logic.
// See docs/internal/TEMPORAL-FLOWS.md and flow/flow.json for the contract.

interface ConversationSessionLike {
  events: ConversationEvent[];
}

type AuxState = Pick<
  SessionPlaybackControllerState,
  | "selectedVersions"
  | "preferredVersion"
  | "summarizeLevel"
  | "summarizingEventId"
  | "summarizeErrors"
  | "focusRequest"
  | "replayDismissed"
>;

const INITIAL_AUX: AuxState = {
  selectedVersions: {},
  preferredVersion: "active",
  summarizeLevel: "moderate",
  summarizingEventId: null,
  summarizeErrors: {},
  focusRequest: null,
  replayDismissed: false,
};

interface UseTtsPlaybackControllerOptions {
  conversationSessions: Record<string, ConversationSessionLike | undefined>;
  activePaneId: string | null;
  autoTtsEnabled: boolean;
  audioState: SessionPlaybackAudioState;
  setViewMode: (sessionId: string, mode: "terminal" | "messages") => void;
  speakText: (sessionId: string, text: string, paragraphs: string[], opts: { eventId: string; version: PlaybackVersion }) => void;
  stopPlayback: (targetId?: string) => void;
  applySummarizeResult: (sessionId: string, eventId: string, speechParagraphs: string[]) => void;
  onSummarizeFailed?: (sessionId: string, eventId: string, message: string) => void;
  onSummarizeSucceeded?: (sessionId: string, eventId: string) => void;
}

let loadIdCounter = 0;
const generateLoadId = (): string => {
  loadIdCounter += 1;
  return `load-${Date.now()}-${loadIdCounter}`;
};

const queueEntriesFromState = (
  state: PlaybackTransportState,
  conversationSessions: Record<string, ConversationSessionLike | undefined>,
  selectedVersions: Record<string, PlaybackVersion>,
  preferredVersion: PlaybackVersion,
): { sessionId: string; entries: PlaybackQueueEntry[]; queueIndex: number } | null => {
  if (state.status === "idle") return null;
  const sessionEvents = conversationSessions[state.sessionId]?.events ?? [];
  const entries = state.queue.map((eventId): PlaybackQueueEntry => {
    const event = sessionEvents.find((candidate) => candidate.id === eventId);
    const version = event
      ? resolvePlaybackVersion(selectedVersions, preferredVersion, state.sessionId, event)
      : preferredVersion;
    return {
      eventId,
      sequence: event?.sequence ?? 0,
      version,
    };
  });
  return { sessionId: state.sessionId, entries, queueIndex: state.queueIndex };
};

export function useTtsPlaybackController({
  conversationSessions,
  activePaneId,
  autoTtsEnabled,
  audioState,
  setViewMode,
  speakText,
  stopPlayback,
  applySummarizeResult,
  onSummarizeFailed,
  onSummarizeSucceeded,
}: UseTtsPlaybackControllerOptions): SessionPlaybackController {
  const [aux, setAux] = useState<AuxState>(INITIAL_AUX);
  const [smState, smDispatchRaw] = useReducer(transitionPlaybackTransport, initialPlaybackTransportState);

  const smStateRef = useRef(smState);
  smStateRef.current = smState;
  const conversationSessionsRef = useRef(conversationSessions);
  conversationSessionsRef.current = conversationSessions;
  const auxRef = useRef(aux);
  auxRef.current = aux;

  const smDispatch = useCallback((event: PlaybackTransportEvent) => {
    smDispatchRaw(event);
  }, []);

  useEffect(() => {
    let cancelled = false;
    void getTTSSummarizeConfig().then((cfg) => {
      if (!cancelled) {
        setAux((prev) => ({ ...prev, summarizeLevel: cfg.level }));
      }
    }).catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  const getEvent = useCallback((sessionId: string, eventId: string): ConversationEvent | null => {
    const session = conversationSessionsRef.current[sessionId];
    return session?.events.find((event) => event.id === eventId) ?? null;
  }, []);

  const getSelectedVersion = useCallback((sessionId: string, event: ConversationEvent) => {
    return resolvePlaybackVersion(aux.selectedVersions, aux.preferredVersion, sessionId, event);
  }, [aux.preferredVersion, aux.selectedVersions]);

  const updateVersionPreference = useCallback((sessionId: string, eventId: string, version: PlaybackVersion) => {
    setAux((prev) => ({
      ...prev,
      preferredVersion: version,
      selectedVersions: {
        ...prev.selectedVersions,
        [playbackEventKey(sessionId, eventId)]: version,
      },
    }));
  }, []);

  const ensurePlaybackData = useCallback(async (
    sessionId: string,
    event: ConversationEvent,
    preferredVersion: PlaybackVersion,
  ): Promise<{ text: string; paragraphs: string[]; version: PlaybackVersion }> => {
    const currentVersions = auxRef.current.selectedVersions;
    const resolvedVersion = resolvePlaybackVersion(currentVersions, preferredVersion, sessionId, event);
    if (resolvedVersion === "active" && event.role === "assistant" && !event.summarized) {
      try {
        const result = await summarizeEvent(sessionId, event.id);
        if (result.summarized && result.speechParagraphs) {
          applySummarizeResult(sessionId, event.id, result.speechParagraphs);
          onSummarizeSucceeded?.(sessionId, event.id);
          setAux((prev) => ({
            ...prev,
            selectedVersions: {
              ...prev.selectedVersions,
              [playbackEventKey(sessionId, event.id)]: "active",
            },
          }));
          return {
            text: event.text,
            paragraphs: result.speechParagraphs,
            version: "active",
          };
        }
        const message = result.error ?? "Summarization failed";
        onSummarizeFailed?.(sessionId, event.id, message);
        setAux((prev) => ({
          ...prev,
          summarizeErrors: { ...prev.summarizeErrors, [event.id]: message },
        }));
      } catch (error: unknown) {
        const message = error instanceof Error ? error.message : "Summarization failed";
        onSummarizeFailed?.(sessionId, event.id, message);
        setAux((prev) => ({
          ...prev,
          summarizeErrors: { ...prev.summarizeErrors, [event.id]: message },
        }));
      }
    }
    return {
      text: event.text,
      paragraphs: resolvePlaybackParagraphs(event, resolvedVersion),
      version: resolvedVersion,
    };
  }, [applySummarizeResult, onSummarizeFailed, onSummarizeSucceeded]);

  // Begin playing a queue (single-event or multi-event) starting at startIndex.
  // Drives the SM through play → loading, awaits ensurePlaybackData, then
  // dispatches loadResolved + invokes speakText. Stale completions (from a
  // later play() superseding this one) are swallowed by loadId matching.
  const beginQueue = useCallback((
    sessionId: string,
    queue: readonly string[],
    startIndex: number,
  ) => {
    if (queue.length === 0) return;
    const eventId = queue[startIndex];
    if (!eventId) return;
    const event = getEvent(sessionId, eventId);
    if (!event) return;
    const loadId = generateLoadId();
    smDispatch({
      type: "play",
      sessionId,
      eventId,
      loadId,
      queue,
      queueIndex: startIndex,
    });
    setAux((prev) => ({ ...prev, replayDismissed: false }));
    stopPlayback(sessionId);
    void ensurePlaybackData(sessionId, event, auxRef.current.preferredVersion).then(({ text, paragraphs, version }) => {
      const current = smStateRef.current;
      if (current.status !== "loading" || current.loadId !== loadId) {
        // Superseded by a newer play / stop. Swallow.
        return;
      }
      smDispatch({ type: "loadResolved", loadId });
      speakText(sessionId, text, paragraphs, { eventId, version });
    }).catch((error: unknown) => {
      const message = error instanceof Error ? error.message : "Playback prep failed";
      const current = smStateRef.current;
      if (current.status === "loading" && current.loadId === loadId) {
        smDispatch({ type: "loadFailed", loadId, message });
      }
    });
  }, [ensurePlaybackData, getEvent, smDispatch, speakText, stopPlayback]);

  const playEvent = useCallback((sessionId: string, eventId: string) => {
    beginQueue(sessionId, [eventId], 0);
  }, [beginQueue]);

  const playFromHere = useCallback((sessionId: string, eventId: string) => {
    const session = conversationSessions[sessionId];
    if (!session) return;
    const startIndex = session.events.findIndex((event) => event.id === eventId);
    if (startIndex === -1) return;
    const queueIds = session.events.slice(startIndex).map((event) => event.id);
    if (queueIds.length === 0) return;
    beginQueue(sessionId, queueIds, 0);
  }, [beginQueue, conversationSessions]);

  // Observe audio-layer state. The audio layer surfaces three transitions we
  // care about, and we synthesize SM events for each:
  //   isSpeaking: true→false  → trackEnded for the current playing target
  //   isPaused:   false→true  → pause (while isSpeaking)
  //   isPaused:   true→false  → resume (while isSpeaking)
  const prevAudioRef = useRef<{ isSpeaking: boolean; isPaused: boolean }>({ isSpeaking: false, isPaused: false });
  useEffect(() => {
    const prev = prevAudioRef.current;
    const isSpeaking = audioState.isSpeaking;
    const isPaused = audioState.playback?.isPaused ?? false;
    prevAudioRef.current = { isSpeaking, isPaused };

    const current = smStateRef.current;

    // pause / resume edges
    if (isSpeaking && !prev.isPaused && isPaused) {
      if (current.status === "playing") {
        smDispatch({ type: "pause" });
      }
    } else if (isSpeaking && prev.isPaused && !isPaused) {
      if (current.status === "paused") {
        smDispatch({ type: "resume" });
      }
    }

    // trackEnded edge: audio stopped speaking. Auto-advance only fires when
    // the SM was in `playing` — the formal model guarantees `paused` swallows
    // TrackEnded (PausedSwallowsAutoAdvance), so pause-then-track-end never
    // advances the queue.
    if (prev.isSpeaking && !isSpeaking) {
      if (current.status === "playing") {
        const endedEventId = current.eventId;
        smDispatch({ type: "trackEnded", eventId: endedEventId });
        const nextEvent = buildPlayNextEvent(current, generateLoadId());
        if (nextEvent && nextEvent.type === "play") {
          // Dispatch + drive side effects via beginQueue's path
          beginQueue(current.sessionId, current.queue, nextEvent.queueIndex);
        }
      }
    }
  }, [audioState.isSpeaking, audioState.playback?.isPaused, beginQueue, smDispatch]);

  const toggleVersion = useCallback((sessionId: string, eventId: string, useSummarized: boolean) => {
    const event = getEvent(sessionId, eventId);
    if (!event) return;
    const version: PlaybackVersion = useSummarized ? "active" : "original";
    updateVersionPreference(sessionId, eventId, version);
    const current = smStateRef.current;
    const isCurrentTarget = current.status !== "idle"
      && current.status !== "error"
      && current.sessionId === sessionId
      && current.eventId === eventId;
    if (!isCurrentTarget) return;
    // Toggle is fast-path: paragraphs are already available on the event, so
    // there is no async data prep. Drive the SM through play → loadResolved
    // synchronously so the UI re-renders in playing state on the same tick
    // and speakText fires before any awaiters resolve.
    const paragraphs = resolvePlaybackParagraphs(event, version);
    const loadId = generateLoadId();
    smDispatch({
      type: "play",
      sessionId,
      eventId,
      loadId,
      queue: [eventId],
      queueIndex: 0,
    });
    stopPlayback(sessionId);
    smDispatch({ type: "loadResolved", loadId });
    setAux((prev) => ({ ...prev, replayDismissed: false }));
    speakText(sessionId, event.text, paragraphs, { eventId, version });
  }, [getEvent, smDispatch, speakText, stopPlayback, updateVersionPreference]);

  const changeSummarizeLevel = useCallback((sessionId: string, eventId: string, level: SummarizationLevel) => {
    const event = getEvent(sessionId, eventId);
    if (!event) return;
    setAux((prev) => ({
      ...prev,
      summarizingEventId: eventId,
      summarizeErrors: { ...prev.summarizeErrors, [eventId]: "" },
    }));
    const currentLevel = auxRef.current.summarizeLevel;
    const persistLevel = level !== currentLevel
      ? updateTTSSummarizeConfig({ level }).then((cfg) => {
          setAux((prev) => ({ ...prev, summarizeLevel: cfg.level }));
        })
      : Promise.resolve();
    void persistLevel
      .then(() => summarizeEvent(sessionId, eventId))
      .then((result) => {
        if (result.summarized && result.speechParagraphs) {
          applySummarizeResult(sessionId, eventId, result.speechParagraphs);
          onSummarizeSucceeded?.(sessionId, eventId);
          updateVersionPreference(sessionId, eventId, "active");
          const current = smStateRef.current;
          const isCurrentTarget = current.status !== "idle"
            && current.status !== "error"
            && current.sessionId === sessionId
            && current.eventId === eventId;
          if (isCurrentTarget) {
            beginQueue(sessionId, [eventId], 0);
          }
          setAux((prev) => {
            const nextErrors = { ...prev.summarizeErrors };
            delete nextErrors[eventId];
            return {
              ...prev,
              summarizingEventId: prev.summarizingEventId === eventId ? null : prev.summarizingEventId,
              summarizeErrors: nextErrors,
            };
          });
          return;
        }
        const message = result.error ?? "Summarization failed";
        onSummarizeFailed?.(sessionId, eventId, message);
        setAux((prev) => ({
          ...prev,
          summarizingEventId: prev.summarizingEventId === eventId ? null : prev.summarizingEventId,
          summarizeErrors: { ...prev.summarizeErrors, [eventId]: message },
        }));
      })
      .catch((error: unknown) => {
        const message = error instanceof Error ? error.message : "Summarization failed";
        onSummarizeFailed?.(sessionId, eventId, message);
        setAux((prev) => ({
          ...prev,
          summarizingEventId: prev.summarizingEventId === eventId ? null : prev.summarizingEventId,
          summarizeErrors: { ...prev.summarizeErrors, [eventId]: message },
        }));
      });
  }, [applySummarizeResult, beginQueue, getEvent, onSummarizeFailed, onSummarizeSucceeded, updateVersionPreference]);

  // The transport-event-start callback no longer mutates queue state directly;
  // queue authority lives in the SM. It still acts as a fallback to seed the
  // SM when audio playback originates outside the controller (e.g. live
  // assistant streaming begins speaking before the user hits play).
  const handleTransportEventStart = useCallback((sessionId: string, eventId: string | null) => {
    if (!eventId) return;
    const current = smStateRef.current;
    const alreadyTracked = current.status !== "idle"
      && current.status !== "error"
      && current.sessionId === sessionId
      && current.queue.includes(eventId);
    if (!alreadyTracked) {
      beginQueue(sessionId, [eventId], 0);
    }
  }, [beginQueue]);

  // No-op kept for API parity: the SM observes audioState.isSpeaking directly
  // to synthesize trackEnded, so this callback no longer needs to drive state.
  const handleTransportStopped = useCallback(() => {
    // intentionally empty
  }, []);

  const clearSummarizeError = useCallback((eventId: string) => {
    setAux((prev) => {
      if (!(eventId in prev.summarizeErrors)) return prev;
      const nextErrors = { ...prev.summarizeErrors };
      delete nextErrors[eventId];
      return { ...prev, summarizeErrors: nextErrors };
    });
  }, []);

  const dismissBar = useCallback((paneId: string | null, isSpeaking: boolean) => {
    if (isSpeaking && paneId) {
      stopPlayback(paneId);
    }
    smDispatch({ type: "stop" });
    setAux((prev) => ({ ...prev, replayDismissed: true }));
  }, [smDispatch, stopPlayback]);

  const focusCurrentEvent = useCallback((paneId: string | null) => {
    const current = smStateRef.current;
    if (current.status === "idle") return;
    if (!paneId || current.sessionId !== paneId) return;
    setViewMode(current.sessionId, "messages");
    setAux((prev) => ({
      ...prev,
      focusRequest: {
        eventId: current.eventId,
        nonce: (prev.focusRequest?.nonce ?? 0) + 1,
      },
    }));
  }, [setViewMode]);

  // Derive a synthetic SessionPlaybackControllerState for utils + bar context.
  const derivedState = useMemo<SessionPlaybackControllerState>(() => {
    const queueInfo = queueEntriesFromState(smState, conversationSessions, aux.selectedVersions, aux.preferredVersion);
    const target = smState.status === "idle"
      ? null
      : { sessionId: smState.sessionId, eventId: smState.eventId };
    const activeTarget = (smState.status === "playing" || smState.status === "loading") ? target : null;
    return {
      selectedVersions: aux.selectedVersions,
      preferredVersion: aux.preferredVersion,
      replayTarget: target,
      activeTarget,
      queueSessionId: queueInfo?.sessionId ?? null,
      queueEntries: queueInfo?.entries ?? [],
      queueIndex: queueInfo?.queueIndex ?? 0,
      summarizeLevel: aux.summarizeLevel,
      summarizingEventId: aux.summarizingEventId,
      summarizeErrors: aux.summarizeErrors,
      focusRequest: aux.focusRequest,
      replayDismissed: aux.replayDismissed,
    };
  }, [aux, conversationSessions, smState]);

  const barContext = useMemo<PlaybackEventContext | null>(() => {
    const target = derivedState.activeTarget ?? derivedState.replayTarget;
    return buildPlaybackContext(conversationSessions, derivedState, target);
  }, [conversationSessions, derivedState]);

  const buildBarContextSelector = useCallback((
    paneId: string | null,
    autoEnabled: boolean,
    currentAudioState: SessionPlaybackAudioState,
  ) => {
    if (!barContext?.event || !barContext.sessionId) return null;
    if (paneId !== barContext.sessionId) return null;
    if (currentAudioState.isSpeaking) return barContext;
    if (!autoEnabled || derivedState.replayDismissed) return null;
    return barContext;
  }, [barContext, derivedState.replayDismissed]);

  // When auto-tts becomes available again on the active pane, surface the
  // bar by clearing the dismiss flag. Mirrors prior controller behaviour.
  useEffect(() => {
    if (!autoTtsEnabled || audioState.isSpeaking) return;
    if (activePaneId && derivedState.replayTarget?.sessionId === activePaneId) return;
    if (!derivedState.replayTarget) return;
    setAux((prev) => (prev.replayDismissed ? { ...prev, replayDismissed: false } : prev));
  }, [activePaneId, audioState.isSpeaking, autoTtsEnabled, derivedState.replayTarget]);

  const activeEventId = smState.status === "playing" || smState.status === "loading"
    ? smState.eventId
    : null;
  const focusRequest: PlaybackFocusRequest | null = aux.focusRequest;

  return {
    summarizeLevel: aux.summarizeLevel,
    summarizingEventId: aux.summarizingEventId,
    activeEventId,
    focusRequest,
    getSelectedVersion,
    getSummarizeError: (eventId) => aux.summarizeErrors[eventId] || null,
    clearSummarizeError,
    playEvent,
    playFromHere,
    setVersionPreference: updateVersionPreference,
    toggleVersion,
    changeSummarizeLevel,
    handleTransportEventStart,
    handleTransportStopped,
    buildBarContext: buildBarContextSelector,
    dismissBar,
    focusCurrentEvent,
  };
}
