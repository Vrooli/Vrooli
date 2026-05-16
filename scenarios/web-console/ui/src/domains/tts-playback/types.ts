import type { ConversationEvent } from "../../api/conversation";
import type { SummarizationLevel } from "../../components/tts/PlaybackModeControl";
import type { TTSPlaybackState } from "../audio";

export type PlaybackVersion = "active" | "original";
export type PlaybackIntent = "continuous" | "paused" | "stopped";

export interface PlaybackQueueEntry {
  eventId: string;
  sequence: number;
  version: PlaybackVersion;
}

export interface PlaybackFocusRequest {
  eventId: string;
  nonce: number;
}

export interface PlaybackTarget {
  sessionId: string;
  eventId: string;
}

export interface PlaybackEventContext {
  event: ConversationEvent | null;
  sessionId: string | null;
  version: PlaybackVersion;
  queueLabel: string | null;
  hasQueuedNext: boolean;
  intent: PlaybackIntent;
}

export interface SessionPlaybackAudioState {
  playback: TTSPlaybackState | null;
  isSpeaking: boolean;
}

export interface SessionPlaybackControllerState {
  selectedVersions: Record<string, PlaybackVersion>;
  preferredVersion: PlaybackVersion;
  replayTarget: PlaybackTarget | null;
  activeTarget: PlaybackTarget | null;
  queueSessionId: string | null;
  queueEntries: PlaybackQueueEntry[];
  queueIndex: number;
  summarizeLevel: SummarizationLevel;
  summarizingEventId: string | null;
  summarizeErrors: Record<string, string>;
  focusRequest: PlaybackFocusRequest | null;
}

export interface IncomingPlaybackAck {
  (stage: string, message?: string, backend?: string): void;
}

export interface SessionPlaybackController {
  summarizeLevel: SummarizationLevel;
  summarizingEventId: string | null;
  activeEventId: string | null;
  focusRequest: PlaybackFocusRequest | null;
  getSelectedVersion: (sessionId: string, event: ConversationEvent) => PlaybackVersion;
  getSummarizeError: (eventId: string) => string | null;
  clearSummarizeError: (eventId: string) => void;
  playEvent: (sessionId: string, eventId: string) => void;
  playFromHere: (sessionId: string, eventId: string) => void;
  setVersionPreference: (sessionId: string, eventId: string, version: PlaybackVersion) => void;
  toggleVersion: (sessionId: string, eventId: string, useSummarized: boolean) => void;
  changeSummarizeLevel: (sessionId: string, eventId: string, level: SummarizationLevel) => void;
  handleIncomingEvent: (sessionId: string, event: ConversationEvent, sendAck: IncomingPlaybackAck) => void;
  handleTransportEventStart: (sessionId: string, eventId: string | null) => void;
  handleTransportStopped: () => void;
  pausePlayback: (sessionId: string | null) => void;
  resumePlayback: (sessionId: string | null) => void;
  stopPlayback: (sessionId: string | null) => void;
  buildBarContext: (
    activePaneId: string | null,
    autoTtsEnabled: boolean,
    audioState: SessionPlaybackAudioState,
  ) => PlaybackEventContext | null;
  focusCurrentEvent: (activePaneId: string | null) => void;
}
