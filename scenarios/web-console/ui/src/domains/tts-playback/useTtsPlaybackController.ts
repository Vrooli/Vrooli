import { useCallback, useEffect, useMemo, useState } from "react";
import { getTTSSummarizeConfig, updateTTSSummarizeConfig } from "../../api/tts";
import { summarizeEvent, type ConversationEvent } from "../../api/conversation";
import type { SummarizationLevel } from "../../components/tts/PlaybackModeControl";
import type { SessionPlaybackAudioState, SessionPlaybackController, SessionPlaybackControllerState } from "./types";
import { buildPlaybackContext, buildPlaybackQueue, playbackEventKey, resolvePlaybackParagraphs, resolvePlaybackVersion } from "./utils";

// AI_CHECK: web_console_tts_playback_unification=2 | LAST: 2026-04-26

interface ConversationSessionLike {
  events: ConversationEvent[];
}

interface UseTtsPlaybackControllerOptions {
  conversationSessions: Record<string, ConversationSessionLike | undefined>;
  activePaneId: string | null;
  autoTtsEnabled: boolean;
  audioState: SessionPlaybackAudioState;
  setViewMode: (sessionId: string, mode: "terminal" | "messages") => void;
  speakText: (sessionId: string, text: string, paragraphs: string[], opts: { eventId: string; version: "active" | "original" }) => void;
  speakSequence: (
    sessionId: string,
    entries: Array<{ eventId: string; text: string; paragraphs: string[]; version: "active" | "original" }>,
    onEventStart: (index: number, eventId: string) => void,
  ) => Promise<void>;
  stopPlayback: (targetId?: string) => void;
  applySummarizeResult: (sessionId: string, eventId: string, speechParagraphs: string[]) => void;
  onSummarizeFailed?: (sessionId: string, eventId: string, message: string) => void;
  onSummarizeSucceeded?: (sessionId: string, eventId: string) => void;
}

const INITIAL_STATE: SessionPlaybackControllerState = {
  selectedVersions: {},
  preferredVersion: "active",
  replayTarget: null,
  activeTarget: null,
  queueSessionId: null,
  queueEntries: [],
  queueIndex: 0,
  summarizeLevel: "moderate",
  summarizingEventId: null,
  summarizeErrors: {},
  focusRequest: null,
  replayDismissed: false,
};

export function useTtsPlaybackController({
  conversationSessions,
  activePaneId,
  autoTtsEnabled,
  audioState,
  setViewMode,
  speakText,
  speakSequence,
  stopPlayback,
  applySummarizeResult,
  onSummarizeFailed,
  onSummarizeSucceeded,
}: UseTtsPlaybackControllerOptions): SessionPlaybackController {
  const [state, setState] = useState<SessionPlaybackControllerState>(INITIAL_STATE);

  useEffect(() => {
    let cancelled = false;
    void getTTSSummarizeConfig().then((cfg) => {
      if (!cancelled) {
        setState((prev) => ({ ...prev, summarizeLevel: cfg.level }));
      }
    }).catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  const getEvent = useCallback((sessionId: string, eventId: string): ConversationEvent | null => {
    const session = conversationSessions[sessionId];
    return session?.events.find((event) => event.id === eventId) ?? null;
  }, [conversationSessions]);

  const getSelectedVersion = useCallback((sessionId: string, event: ConversationEvent) => {
    return resolvePlaybackVersion(state.selectedVersions, state.preferredVersion, sessionId, event);
  }, [state.preferredVersion, state.selectedVersions]);

  const updateVersionPreference = useCallback((sessionId: string, eventId: string, version: "active" | "original") => {
    setState((prev) => ({
      ...prev,
      preferredVersion: version,
      selectedVersions: {
        ...prev.selectedVersions,
        [playbackEventKey(sessionId, eventId)]: version,
      },
    }));
  }, []);

  const setReplayTarget = useCallback((sessionId: string, eventId: string) => {
    setState((prev) => ({
      ...prev,
      activeTarget: { sessionId, eventId },
      replayTarget: { sessionId, eventId },
      replayDismissed: false,
    }));
  }, []);

  const ensurePlaybackData = useCallback(async (
    sessionId: string,
    event: ConversationEvent,
    preferredVersion: "active" | "original",
  ): Promise<{ text: string; paragraphs: string[]; version: "active" | "original" }> => {
    const resolvedVersion = resolvePlaybackVersion(state.selectedVersions, preferredVersion, sessionId, event);
    if (
      resolvedVersion === "active"
      && event.role === "assistant"
      && !event.summarized
    ) {
      try {
        const result = await summarizeEvent(sessionId, event.id);
        if (result.summarized && result.speechParagraphs) {
          applySummarizeResult(sessionId, event.id, result.speechParagraphs);
          onSummarizeSucceeded?.(sessionId, event.id);
          setState((prev) => ({
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
        setState((prev) => ({
          ...prev,
          summarizeErrors: { ...prev.summarizeErrors, [event.id]: message },
        }));
      } catch (error: unknown) {
        const message = error instanceof Error ? error.message : "Summarization failed";
        onSummarizeFailed?.(sessionId, event.id, message);
        setState((prev) => ({
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
  }, [applySummarizeResult, onSummarizeFailed, onSummarizeSucceeded, state.selectedVersions]);

  const playEvent = useCallback((sessionId: string, eventId: string) => {
    const event = getEvent(sessionId, eventId);
    if (!event) return;
    stopPlayback(sessionId);
    void ensurePlaybackData(sessionId, event, state.preferredVersion).then(({ text, paragraphs, version }) => {
      setState((prev) => ({
        ...prev,
        activeTarget: { sessionId, eventId },
        replayTarget: { sessionId, eventId },
        queueSessionId: sessionId,
        queueEntries: [{ eventId, sequence: event.sequence, version }],
        queueIndex: 0,
        replayDismissed: false,
      }));
      speakText(sessionId, text, paragraphs, { eventId, version });
    });
  }, [ensurePlaybackData, getEvent, speakText, state.preferredVersion, stopPlayback]);

  const playFromHere = useCallback((sessionId: string, eventId: string) => {
    const session = conversationSessions[sessionId];
    if (!session) return;
    const startIndex = session.events.findIndex((event) => event.id === eventId);
    if (startIndex === -1) return;
    const queuedEvents = session.events.slice(startIndex);
    if (queuedEvents.length === 0) return;
    const first = queuedEvents[0];
    if (!first) return;
    stopPlayback(sessionId);
    void Promise.all(queuedEvents.map(async (event) => {
      const prepared = await ensurePlaybackData(sessionId, event, state.preferredVersion);
      return {
        event,
        prepared,
      };
    })).then((preparedEntries) => {
      const queueEntries = preparedEntries.map(({ event, prepared }) => ({
        eventId: event.id,
        sequence: event.sequence,
        version: prepared.version,
      }));
      const transportEntries = preparedEntries.map(({ event, prepared }) => ({
        eventId: event.id,
        text: prepared.text,
        paragraphs: prepared.paragraphs,
        version: prepared.version,
      }));
      setState((prev) => ({
        ...prev,
        activeTarget: { sessionId, eventId: first.id },
        replayTarget: { sessionId, eventId: first.id },
        queueSessionId: sessionId,
        queueEntries,
        queueIndex: 0,
        replayDismissed: false,
      }));
      void speakSequence(sessionId, transportEntries, (index, currentEventId) => {
        setState((prev) => ({
          ...prev,
          activeTarget: { sessionId, eventId: currentEventId },
          replayTarget: { sessionId, eventId: currentEventId },
          queueIndex: index,
          replayDismissed: false,
        }));
      });
    });
  }, [conversationSessions, ensurePlaybackData, speakSequence, state.preferredVersion, stopPlayback]);

  const toggleVersion = useCallback((sessionId: string, eventId: string, useSummarized: boolean) => {
    const event = getEvent(sessionId, eventId);
    if (!event) return;
    const version = useSummarized ? "active" : "original";
    updateVersionPreference(sessionId, eventId, version);
    const isCurrentTarget = state.activeTarget?.sessionId === sessionId && state.activeTarget?.eventId === eventId;
    const isReplayTarget = state.replayTarget?.sessionId === sessionId && state.replayTarget?.eventId === eventId;
    if (isCurrentTarget || isReplayTarget) {
      const paragraphs = resolvePlaybackParagraphs(event, version);
      setReplayTarget(sessionId, eventId);
      speakText(sessionId, event.text, paragraphs, { eventId, version });
    }
  }, [getEvent, setReplayTarget, speakText, state.activeTarget, state.replayTarget, updateVersionPreference]);

  const changeSummarizeLevel = useCallback((sessionId: string, eventId: string, level: SummarizationLevel) => {
    const event = getEvent(sessionId, eventId);
    if (!event) return;
    setState((prev) => ({
      ...prev,
      summarizingEventId: eventId,
      summarizeErrors: {
        ...prev.summarizeErrors,
        [eventId]: "",
      },
    }));
    const persistLevel = level !== state.summarizeLevel
      ? updateTTSSummarizeConfig({ level }).then((cfg) => {
          setState((prev) => ({ ...prev, summarizeLevel: cfg.level }));
        })
      : Promise.resolve();
    void persistLevel
      .then(() => summarizeEvent(sessionId, eventId))
      .then((result) => {
        if (result.summarized && result.speechParagraphs) {
          applySummarizeResult(sessionId, eventId, result.speechParagraphs);
          onSummarizeSucceeded?.(sessionId, eventId);
          updateVersionPreference(sessionId, eventId, "active");
          const isCurrentTarget = state.activeTarget?.sessionId === sessionId && state.activeTarget?.eventId === eventId;
          const isReplayTarget = state.replayTarget?.sessionId === sessionId && state.replayTarget?.eventId === eventId;
          if (isCurrentTarget || isReplayTarget) {
            const refreshedEvent = getEvent(sessionId, eventId);
            const playbackEvent = refreshedEvent ?? event;
            setReplayTarget(sessionId, eventId);
            speakText(sessionId, playbackEvent.text, result.speechParagraphs, { eventId, version: "active" });
          }
          setState((prev) => {
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
        setState((prev) => ({
          ...prev,
          summarizingEventId: prev.summarizingEventId === eventId ? null : prev.summarizingEventId,
          summarizeErrors: { ...prev.summarizeErrors, [eventId]: message },
        }));
      })
      .catch((error: unknown) => {
        const message = error instanceof Error ? error.message : "Summarization failed";
        onSummarizeFailed?.(sessionId, eventId, message);
        setState((prev) => ({
          ...prev,
          summarizingEventId: prev.summarizingEventId === eventId ? null : prev.summarizingEventId,
          summarizeErrors: { ...prev.summarizeErrors, [eventId]: message },
        }));
      });
  }, [applySummarizeResult, getEvent, onSummarizeFailed, onSummarizeSucceeded, setReplayTarget, speakText, state.activeTarget, state.replayTarget, state.summarizeLevel, updateVersionPreference]);

  const handleTransportEventStart = useCallback((sessionId: string, eventId: string | null) => {
    if (!eventId) {
      setState((prev) => ({ ...prev, activeTarget: null }));
      return;
    }
    setState((prev) => {
      const queueIndex = prev.queueSessionId === sessionId
        ? prev.queueEntries.findIndex((entry) => entry.eventId === eventId)
        : -1;
          return {
            ...prev,
            activeTarget: { sessionId, eventId },
            replayTarget: { sessionId, eventId },
            queueSessionId: queueIndex >= 0 ? prev.queueSessionId : sessionId,
            queueEntries: queueIndex >= 0 ? prev.queueEntries : (() => {
              const event = getEvent(sessionId, eventId);
              if (!event) return prev.queueEntries;
              return buildPlaybackQueue(sessionId, [event], prev.selectedVersions, prev.preferredVersion);
            })(),
            queueIndex: queueIndex >= 0 ? queueIndex : 0,
            replayDismissed: false,
      };
    });
  }, [getEvent]);

  const handleTransportStopped = useCallback(() => {
    setState((prev) => {
      if (!prev.activeTarget) return prev;
      return { ...prev, activeTarget: null };
    });
  }, []);

  const clearSummarizeError = useCallback((eventId: string) => {
    setState((prev) => {
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
    setState((prev) => ({
      ...prev,
      activeTarget: null,
      replayTarget: null,
      queueSessionId: null,
      queueEntries: [],
      queueIndex: 0,
      replayDismissed: true,
    }));
  }, [stopPlayback]);

  const focusCurrentEvent = useCallback((paneId: string | null) => {
    const target = state.activeTarget ?? state.replayTarget;
    if (!paneId || !target || target.sessionId !== paneId) return;
    setViewMode(target.sessionId, "messages");
    setState((prev) => ({
      ...prev,
      focusRequest: {
        eventId: target.eventId,
        nonce: (prev.focusRequest?.nonce ?? 0) + 1,
      },
    }));
  }, [setViewMode, state.activeTarget, state.replayTarget]);

  const barContext = useMemo(() => {
    const currentTarget = state.activeTarget ?? state.replayTarget;
    return buildPlaybackContext(conversationSessions, state, currentTarget);
  }, [conversationSessions, state]);

  const buildBarContextSelector = useCallback((
    paneId: string | null,
    autoEnabled: boolean,
    currentAudioState: SessionPlaybackAudioState,
  ) => {
    if (!barContext?.event || !barContext.sessionId) return null;
    if (paneId !== barContext.sessionId) return null;
    if (currentAudioState.isSpeaking) return barContext;
    if (!autoEnabled || state.replayDismissed) return null;
    return barContext;
  }, [barContext, state.replayDismissed]);

  useEffect(() => {
    if (!autoTtsEnabled || audioState.isSpeaking) return;
    if (activePaneId && state.replayTarget?.sessionId === activePaneId) return;
    if (!state.replayTarget) return;
    setState((prev) => ({ ...prev, replayDismissed: false }));
  }, [activePaneId, audioState.isSpeaking, autoTtsEnabled, state.replayTarget]);

  return {
    summarizeLevel: state.summarizeLevel,
    summarizingEventId: state.summarizingEventId,
    activeEventId: state.activeTarget?.eventId ?? null,
    focusRequest: state.focusRequest,
    getSelectedVersion,
    getSummarizeError: (eventId) => state.summarizeErrors[eventId] || null,
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
