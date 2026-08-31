import { useCallback, useEffect, useMemo, useRef } from "react";
import { useTextToSpeech } from "../useTextToSpeech";
import { useConversationSession } from "../useConversationSession";
import { useConversationStore } from "../../stores/useConversationStore";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import type { ConversationEvent } from "../../api/conversation";
import type { TTSPlaybackState } from "../../audio-integration";

const EMPTY_CONVERSATION_EVENTS: ConversationEvent[] = [];
const EMPTY_CONVERSATION_CURSOR = { lastSeenSequence: 0, lastListenedSequence: 0 } as const;
const AUTO_TTS_MAX_AGE_MS = 60_000;
const TTS_MAX_CHUNK_BYTES = 4500;
const textEncoder = new TextEncoder();

export type PaneSpeechOptions = {
  eventId?: string;
  version?: "active" | "original";
  initiatedBy?: "auto" | "manual";
};

export interface PaneSpeechPlaybackHandle {
  stop: () => void;
  speak: (text: string, paragraphs?: string[], options?: PaneSpeechOptions) => Promise<string | undefined>;
  pause: () => void;
  resume: () => void;
  seek: (seconds: number) => void;
  setPlaybackRate: (rate: number) => void;
  setVolume: (level: number) => void;
  setMuted: (next: boolean) => void;
  getState: () => TTSPlaybackState | null;
  getBackendReason?: () => string;
}

function utf8ByteLength(value: string): number {
  return textEncoder.encode(value).length;
}

/** Keep speech requests below the backend's UTF-8 byte limit. */
export function ensureSpeechChunks(paragraphs: string[]): string[] {
  const result: string[] = [];
  for (const paragraph of paragraphs) {
    if (utf8ByteLength(paragraph) <= TTS_MAX_CHUNK_BYTES) {
      result.push(paragraph);
      continue;
    }
    let remaining = paragraph;
    while (utf8ByteLength(remaining) > TTS_MAX_CHUNK_BYTES) {
      let low = 0;
      let high = remaining.length;
      while (low < high) {
        const middle = (low + high + 1) >>> 1;
        if (utf8ByteLength(remaining.slice(0, middle)) <= TTS_MAX_CHUNK_BYTES) low = middle;
        else high = middle - 1;
      }
      const spaceAt = remaining.lastIndexOf(" ", low);
      const cut = spaceAt > 0 ? spaceAt : low;
      result.push(remaining.slice(0, cut).trim());
      remaining = remaining.slice(cut).trim();
    }
    if (remaining) result.push(remaining);
  }
  return result.filter(Boolean);
}

export function usePaneSpeech(options: {
  sessionId: string;
  onSpeakingEventChange?: (eventId: string | null) => void;
  onTtsSpeakingChange?: (speaking: boolean) => void;
  onNeedsUnlock?: (payload: { sessionId: string; enable: () => Promise<boolean> } | null) => void;
  onConversationEventReceived?: (
    sessionId: string,
    event: ConversationEvent,
    sendAck: (stage: string, message?: string, backend?: string) => void,
  ) => void;
  sendConversationAck: (eventId: string, source: string, stage: string, message?: string, backend?: string) => void;
}) {
  const { sessionId, onSpeakingEventChange, onTtsSpeakingChange, onNeedsUnlock, onConversationEventReceived, sendConversationAck } = options;
  const ttsVoice = useWorkspaceStore((state) => state.ttsVoice);
  const ttsRate = useWorkspaceStore((state) => state.ttsRate);
  const ttsPitch = useWorkspaceStore((state) => state.ttsPitch);
  const kokoroVoice = useWorkspaceStore((state) => state.kokoroVoice);
  const kokoroSpeed = useWorkspaceStore((state) => state.kokoroSpeed);
  const ttsBackendPreference = useWorkspaceStore((state) => state.ttsBackendPreference);
  const resolvedSettings = useMemo(() => ({
    voice: ttsVoice,
    rate: ttsRate,
    pitch: ttsPitch,
    kokoroVoice,
    kokoroSpeed,
    backendPreference: ttsBackendPreference,
  }), [ttsVoice, ttsRate, ttsPitch, kokoroVoice, kokoroSpeed, ttsBackendPreference]);
  const {
    speakParagraphs, stop, pause, resume, seek, setPlaybackRate, setVolume, setMuted,
    getPlaybackState, backendReason, supported, isSpeaking, needsUnlock, unlockAudio,
  } = useTextToSpeech(resolvedSettings, { source: "terminal_auto", sessionId });

  const onSpeakingEventChangeRef = useRef(onSpeakingEventChange);
  onSpeakingEventChangeRef.current = onSpeakingEventChange;
  useEffect(() => onTtsSpeakingChange?.(isSpeaking), [isSpeaking, onTtsSpeakingChange]);

  const onNeedsUnlockRef = useRef(onNeedsUnlock);
  onNeedsUnlockRef.current = onNeedsUnlock;
  useEffect(() => {
    if (!needsUnlock) {
      onNeedsUnlockRef.current?.(null);
      return;
    }
    onNeedsUnlockRef.current?.({ sessionId, enable: unlockAudio });
  }, [needsUnlock, sessionId, unlockAudio]);

  const activePane = useWorkspaceStore((state) => state.activePane);
  const conversationSession = useConversationStore((state) => state.sessions[sessionId]);
  const conversationEvents = conversationSession?.events ?? EMPTY_CONVERSATION_EVENTS;
  const conversationCursor = conversationSession?.cursor ?? EMPTY_CONVERSATION_CURSOR;
  const conversationHydrated = conversationSession?.hydrated ?? false;
  const { persistCursor } = useConversationSession(sessionId, { hydrate: false });

  useEffect(() => {
    if (activePane !== sessionId) return;
    const latestSequence = conversationEvents[conversationEvents.length - 1]?.sequence;
    if (latestSequence && latestSequence > conversationCursor.lastSeenSequence) {
      void persistCursor({ lastSeenSequence: latestSequence });
    }
  }, [activePane, conversationCursor.lastSeenSequence, conversationEvents, persistCursor, sessionId]);

  const autoTtsBaselineRef = useRef<number | null>(null);
  useEffect(() => {
    if (!conversationHydrated) return;
    const maxSequence = conversationEvents[conversationEvents.length - 1]?.sequence ?? 0;
    const baseline = autoTtsBaselineRef.current;
    if (baseline === null || activePane !== sessionId) {
      autoTtsBaselineRef.current = maxSequence;
      return;
    }
    if (maxSequence <= baseline) return;
    autoTtsBaselineRef.current = maxSequence;
    if (!supported || !onConversationEventReceived) return;
    let latest: ConversationEvent | undefined;
    for (let index = conversationEvents.length - 1; index >= 0; index -= 1) {
      const candidate = conversationEvents[index];
      if (!candidate || candidate.sequence <= baseline) break;
      if (candidate.role === "assistant") {
        latest = candidate;
        break;
      }
    }
    if (!latest) return;
    const ageMs = Date.now() - new Date(latest.createdAt).getTime();
    if (Number.isFinite(ageMs) && ageMs > AUTO_TTS_MAX_AGE_MS) return;
    onConversationEventReceived(sessionId, latest, (stage, message, backend) => {
      sendConversationAck(latest.id, latest.source, stage, message, backend);
    },
    );
  }, [activePane, conversationEvents, conversationHydrated, onConversationEventReceived, sendConversationAck, sessionId, supported]);

  const speak = useCallback((text: string, paragraphs?: string[], speechOptions?: PaneSpeechOptions) => {
    if (speechOptions?.initiatedBy !== "auto") setMuted(false);
    stop();
    onSpeakingEventChangeRef.current?.(speechOptions?.eventId ?? null);
    return speakParagraphs(ensureSpeechChunks(paragraphs ?? [text]), speechOptions).finally(() => {
      onSpeakingEventChangeRef.current?.(null);
    });
  }, [setMuted, speakParagraphs, stop]);

  const playback = useMemo<PaneSpeechPlaybackHandle>(() => ({
    stop: () => {
      stop();
      onSpeakingEventChangeRef.current?.(null);
    },
    speak,
    pause,
    resume: () => {
      setMuted(false);
      resume();
    },
    seek,
    setPlaybackRate,
    setVolume,
    setMuted,
    getState: getPlaybackState,
    getBackendReason: () => backendReason,
  }), [backendReason, getPlaybackState, pause, resume, seek, setMuted, setPlaybackRate, setVolume, speak, stop]);

  return { supported, playback };
}
