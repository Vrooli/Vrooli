import type { VoiceActivitySnapshot, VoiceMode } from "./types";
import type { VadRefs } from "./vad";

export const VAD_AUTO_STOP_VISUAL_GRACE_MS = 300;

export const IDLE_VOICE_ACTIVITY: VoiceActivitySnapshot = {
  phase: "idle",
  audioLevel: 0,
  rms: 0,
  speechThreshold: 0,
  silenceThreshold: 0,
  silenceElapsedMs: 0,
  silenceTimeoutMs: 0,
  autoStopProgress: 0,
  autoStopVisible: false,
};

function clamp01(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(1, value));
}

export interface BuildVoiceActivitySnapshotInput {
  vadActive: boolean;
  vad: VadRefs;
  rms: number;
  audioLevel: number;
  nowMs: number;
  silenceTimeoutMs: number;
  voiceMode: VoiceMode;
}

export function buildVoiceActivitySnapshot({
  vadActive,
  vad,
  rms,
  audioLevel,
  nowMs,
  silenceTimeoutMs,
  voiceMode,
}: BuildVoiceActivitySnapshotInput): VoiceActivitySnapshot {
  if (!vadActive || vad.state === "idle") {
    return { ...IDLE_VOICE_ACTIVITY, audioLevel: clamp01(audioLevel), rms: Math.max(0, rms) };
  }

  const base = {
    audioLevel: clamp01(audioLevel),
    rms: Math.max(0, rms),
    speechThreshold: Math.max(0, vad.speechThreshold),
    silenceThreshold: Math.max(0, vad.silenceThreshold),
    silenceElapsedMs: 0,
    silenceTimeoutMs: Math.max(0, silenceTimeoutMs),
    autoStopProgress: 0,
    autoStopVisible: false,
  };

  if (vad.state === "calibrating") {
    return { ...base, phase: "calibrating" };
  }
  if (vad.state === "waitingForSpeech") {
    return { ...base, phase: "waiting-for-speech" };
  }
  if (vad.state === "speechDetected") {
    return { ...base, phase: "speech" };
  }

  const silenceElapsedMs = Math.max(0, nowMs - vad.silenceStart);
  const effectiveTimeoutMs = Math.max(0, silenceTimeoutMs);
  const autoStopProgress = effectiveTimeoutMs > 0
    ? clamp01(silenceElapsedMs / effectiveTimeoutMs)
    : 0;
  const autoStopVisible = voiceMode === "one-shot"
    && effectiveTimeoutMs > 0
    && silenceElapsedMs >= VAD_AUTO_STOP_VISUAL_GRACE_MS;

  return {
    ...base,
    phase: "silence",
    silenceElapsedMs,
    silenceTimeoutMs: effectiveTimeoutMs,
    autoStopProgress,
    autoStopVisible,
  };
}

export function voiceActivitySnapshotsEqual(a: VoiceActivitySnapshot, b: VoiceActivitySnapshot): boolean {
  return a.phase === b.phase
    && Math.abs(a.audioLevel - b.audioLevel) < 0.01
    && Math.abs(a.rms - b.rms) < 0.001
    && Math.abs(a.speechThreshold - b.speechThreshold) < 0.001
    && Math.abs(a.silenceThreshold - b.silenceThreshold) < 0.001
    && Math.abs(a.silenceElapsedMs - b.silenceElapsedMs) < 16
    && a.silenceTimeoutMs === b.silenceTimeoutMs
    && Math.abs(a.autoStopProgress - b.autoStopProgress) < 0.01
    && a.autoStopVisible === b.autoStopVisible;
}
