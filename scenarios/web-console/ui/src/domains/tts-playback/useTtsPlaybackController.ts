import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react";
import { getTTSSummarizeConfig, updateTTSSummarizeConfig } from "../../audio-integration";
import { summarizeEvent, type ConversationEvent } from "../../api/conversation";
import type { SummarizationLevel } from "../../components/tts/PlaybackModeControl";
import type {
  IncomingPlaybackAck,
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
  nextIntentAfterNaturalCompletion,
  nextIntentAfterUserPause,
  nextIntentAfterUserPlay,
  nextIntentAfterUserStop,
  shouldAutoPlayIncomingEvent,
  shouldShowPlaybackBar,
} from "./utils";
import { updateConversationCursor } from "../../api/conversation";
import { useConversationStore } from "../../stores/useConversationStore";
import { useTtsPlaybackIntentStore } from "./store";
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
// surfaces — version preference, summarization status, and focus requests —
// remain plain hook state because they are not temporal logic.
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
>;

const INITIAL_AUX: AuxState = {
  selectedVersions: {},
  preferredVersion: "active",
  summarizeLevel: "moderate",
  summarizingEventId: null,
  summarizeErrors: {},
  focusRequest: null,
};

interface UseTtsPlaybackControllerOptions {
  conversationSessions: Record<string, ConversationSessionLike | undefined>;
  activePaneId: string | null;
  autoTtsEnabled: boolean;
  audioState: SessionPlaybackAudioState;
  setViewMode: (sessionId: string, mode: "terminal" | "messages") => void;
  speakText: (sessionId: string, text: string, paragraphs: string[], opts: { eventId: string; version: PlaybackVersion; initiatedBy: "auto" | "manual" }) => Promise<string | undefined>;
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

function summarizeFailureMessage(error: unknown): string {
  if (error instanceof Error) {
    const raw = error.message.trim();
    if (!raw) return "Summarization failed";
    if (raw.includes("[deadline_exceeded]") || raw.toLowerCase().includes("timed out")) {
      return "Summarization timed out before audio-tools returned a result. Try again or increase the summarize timeout in voice settings.";
    }
    if (raw.includes("[unavailable]") || raw.includes("HTTP 502")) {
      return "Summarization failed: audio-tools is unavailable. Check that audio-tools and its Ollama summarizer are running.";
    }
    if (raw.includes("[failed_precondition]") || raw.toLowerCase().includes("model is not installed")) {
      return "Summarization failed: selected Ollama summarizer model is not installed. Choose an installed model in voice settings or run the shown ollama pull command.";
    }
    return raw.startsWith("Summarization failed") ? raw : `Summarization failed: ${raw}`;
  }
  return "Summarization failed";
}

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
  const playbackIntent = useTtsPlaybackIntentStore((state) => state.playbackIntent);
  const persistedTarget = useTtsPlaybackIntentStore((state) => state.selectedTarget);
  const setPlaybackIntent = useTtsPlaybackIntentStore((state) => state.setPlaybackIntent);
  const setPersistedTarget = useTtsPlaybackIntentStore((state) => state.setSelectedTarget);

  const smStateRef = useRef(smState);
  smStateRef.current = smState;
  const conversationSessionsRef = useRef(conversationSessions);
  conversationSessionsRef.current = conversationSessions;
  const auxRef = useRef(aux);
  auxRef.current = aux;
  const playbackIntentRef = useRef(playbackIntent);
  playbackIntentRef.current = playbackIntent;
  const lastStartedEventRef = useRef<{ sessionId: string; eventId: string } | null>(null);

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
        const message = summarizeFailureMessage(result.error ?? "Summarization failed");
        onSummarizeFailed?.(sessionId, event.id, message);
        setAux((prev) => ({
          ...prev,
          summarizeErrors: { ...prev.summarizeErrors, [event.id]: message },
        }));
      } catch (error: unknown) {
        const message = summarizeFailureMessage(error);
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

  const persistListened = useCallback((sessionId: string, event: ConversationEvent) => {
    const cursor = { lastListenedSequence: event.sequence, lastSeenSequence: event.sequence };
    useConversationStore.getState().updateCursor(sessionId, cursor);
    void updateConversationCursor(sessionId, cursor)
      .then((updated) => {
        useConversationStore.getState().updateCursor(sessionId, updated);
      })
      .catch(() => {
        // Best effort: retain the optimistic local cursor.
      });
  }, []);

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
    setPlaybackIntent(nextIntentAfterUserPlay());
    setPersistedTarget({ sessionId, eventId });
    stopPlayback(sessionId);
    void ensurePlaybackData(sessionId, event, auxRef.current.preferredVersion).then(({ text, paragraphs, version }) => {
      const current = smStateRef.current;
      if (current.status !== "loading" || current.loadId !== loadId) {
        // Superseded by a newer play / stop. Swallow.
        return;
      }
      smDispatch({ type: "loadResolved", loadId });
      lastStartedEventRef.current = { sessionId, eventId };
      void speakText(sessionId, text, paragraphs, { eventId, version, initiatedBy: "manual" });
    }).catch((error: unknown) => {
      const message = error instanceof Error ? error.message : "Playback prep failed";
      const current = smStateRef.current;
      if (current.status === "loading" && current.loadId === loadId) {
        smDispatch({ type: "loadFailed", loadId, message });
      }
    });
  }, [ensurePlaybackData, getEvent, setPersistedTarget, setPlaybackIntent, smDispatch, speakText, stopPlayback]);

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
        setPlaybackIntent(nextIntentAfterNaturalCompletion(playbackIntentRef.current));
        const endedEvent = getEvent(current.sessionId, endedEventId);
        if (endedEvent) {
          persistListened(current.sessionId, endedEvent);
          setPersistedTarget({ sessionId: current.sessionId, eventId: endedEventId });
        }
        const nextEvent = buildPlayNextEvent(current, generateLoadId());
        if (nextEvent && nextEvent.type === "play") {
          // Dispatch + drive side effects via beginQueue's path
          beginQueue(current.sessionId, current.queue, nextEvent.queueIndex);
        }
      }
    }
  }, [audioState.isSpeaking, audioState.playback?.isPaused, beginQueue, getEvent, persistListened, setPersistedTarget, setPlaybackIntent, smDispatch]);

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
    const isPersistedTarget = persistedTarget?.sessionId === sessionId && persistedTarget.eventId === eventId;
    if (!isCurrentTarget && !isPersistedTarget) return;
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
    setPlaybackIntent(nextIntentAfterUserPlay());
    setPersistedTarget({ sessionId, eventId });
    lastStartedEventRef.current = { sessionId, eventId };
    void speakText(sessionId, event.text, paragraphs, { eventId, version, initiatedBy: "manual" });
  }, [getEvent, persistedTarget, setPersistedTarget, setPlaybackIntent, smDispatch, speakText, stopPlayback, updateVersionPreference]);

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
        const message = summarizeFailureMessage(result.error ?? "Summarization failed");
        onSummarizeFailed?.(sessionId, eventId, message);
        setAux((prev) => ({
          ...prev,
          summarizingEventId: prev.summarizingEventId === eventId ? null : prev.summarizingEventId,
          summarizeErrors: { ...prev.summarizeErrors, [eventId]: message },
        }));
      })
      .catch((error: unknown) => {
        const message = summarizeFailureMessage(error);
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
  const handleIncomingEvent = useCallback((sessionId: string, event: ConversationEvent, sendAck: IncomingPlaybackAck) => {
    if (event.role !== "assistant") return;
    if (!shouldAutoPlayIncomingEvent({
      autoTtsEnabled,
      playbackIntent: playbackIntentRef.current,
      activePaneId,
      sessionId,
      event,
      isSpeaking: audioState.isSpeaking,
    })) {
      return;
    }
    setPersistedTarget({ sessionId, eventId: event.id });
    const loadId = generateLoadId();
    smDispatch({
      type: "play",
      sessionId,
      eventId: event.id,
      loadId,
      queue: [event.id],
      queueIndex: 0,
    });
    stopPlayback(sessionId);
    void ensurePlaybackData(sessionId, event, auxRef.current.preferredVersion)
      .then(({ text, paragraphs, version }) => {
        const current = smStateRef.current;
        if (current.status !== "loading" || current.loadId !== loadId) return;
        smDispatch({ type: "loadResolved", loadId });
        lastStartedEventRef.current = { sessionId, eventId: event.id };
        sendAck("playback_started");
        return speakText(sessionId, text, paragraphs, { eventId: event.id, version, initiatedBy: "auto" });
      })
      .then((usedBackend) => {
        const current = smStateRef.current;
        if (!usedBackend || current.status !== "playing" || current.sessionId !== sessionId || current.eventId !== event.id) {
          if (!usedBackend) {
            sendAck("playback_failed", "TTS provider not ready");
          }
          return;
        }
        sendAck("playback_succeeded", undefined, usedBackend);
      })
      .catch((error: unknown) => {
        const message = error instanceof Error ? error.message : "Speech failed";
        const current = smStateRef.current;
        if (current.status === "loading" && current.loadId === loadId) {
          smDispatch({ type: "loadFailed", loadId, message });
        } else if (current.status === "playing" && current.sessionId === sessionId && current.eventId === event.id) {
          smDispatch({ type: "playbackError", message });
        }
        sendAck("playback_failed", message);
      });
  }, [activePaneId, audioState.isSpeaking, autoTtsEnabled, ensurePlaybackData, setPersistedTarget, smDispatch, speakText, stopPlayback]);

  const handleTransportEventStart = useCallback((sessionId: string, eventId: string | null) => {
    if (!eventId) return;
    const current = smStateRef.current;
    const alreadyTracked = current.status !== "idle"
      && current.status !== "error"
      && current.sessionId === sessionId
      && current.queue.includes(eventId);
    if (!alreadyTracked) {
      setPersistedTarget({ sessionId, eventId });
    }
  }, [setPersistedTarget]);

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

  const pausePlayback = useCallback((sessionId: string | null) => {
    setPlaybackIntent(nextIntentAfterUserPause());
    if (sessionId) {
      // The provider pause is invoked by Workspace; this records durable intent.
      setPersistedTarget(smStateRef.current.status === "idle" ? persistedTarget : { sessionId, eventId: smStateRef.current.eventId });
    }
    smDispatch({ type: "pause" });
  }, [persistedTarget, setPersistedTarget, setPlaybackIntent, smDispatch]);

  const resumePlayback = useCallback((sessionId: string | null) => {
    setPlaybackIntent(nextIntentAfterUserPlay());
    if (sessionId) {
      const current = smStateRef.current;
      if (current.status === "paused") {
        smDispatch({ type: "resume" });
        return;
      }
      const target = current.status === "idle" ? persistedTarget : { sessionId: current.sessionId, eventId: current.eventId };
      if (target) beginQueue(target.sessionId, [target.eventId], 0);
    }
  }, [beginQueue, persistedTarget, setPlaybackIntent, smDispatch]);

  const stopPlaybackWithIntent = useCallback((sessionId: string | null) => {
    setPlaybackIntent(nextIntentAfterUserStop());
    if (sessionId) stopPlayback(sessionId);
    smDispatch({ type: "stop" });
  }, [setPlaybackIntent, smDispatch, stopPlayback]);

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
      ? persistedTarget
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
    };
  }, [aux, conversationSessions, persistedTarget, smState]);

  const barContext = useMemo<PlaybackEventContext | null>(() => {
    const target = derivedState.activeTarget ?? derivedState.replayTarget;
    return buildPlaybackContext(conversationSessions, derivedState, target, playbackIntent);
  }, [conversationSessions, derivedState, playbackIntent]);

  const buildBarContextSelector = useCallback((
    paneId: string | null,
    autoEnabled: boolean,
    currentAudioState: SessionPlaybackAudioState,
  ) => {
    if (!shouldShowPlaybackBar({
      autoTtsEnabled: autoEnabled,
      activePaneId: paneId,
      context: barContext,
      isSpeaking: currentAudioState.isSpeaking,
    })) return null;
    return barContext;
  }, [barContext]);

  const activeEventId = smState.status === "playing" || smState.status === "loading"
    ? smState.eventId
    : null;
  const loadingEventId = smState.status === "loading" ? smState.eventId : null;
  const focusRequest: PlaybackFocusRequest | null = aux.focusRequest;

  return {
    summarizeLevel: aux.summarizeLevel,
    summarizingEventId: aux.summarizingEventId,
    activeEventId,
    loadingEventId,
    focusRequest,
    getSelectedVersion,
    getSummarizeError: (eventId) => aux.summarizeErrors[eventId] || null,
    clearSummarizeError,
    playEvent,
    playFromHere,
    setVersionPreference: updateVersionPreference,
    toggleVersion,
    changeSummarizeLevel,
    handleIncomingEvent,
    handleTransportEventStart,
    handleTransportStopped,
    pausePlayback,
    resumePlayback,
    stopPlayback: stopPlaybackWithIntent,
    buildBarContext: buildBarContextSelector,
    focusCurrentEvent,
  };
}
