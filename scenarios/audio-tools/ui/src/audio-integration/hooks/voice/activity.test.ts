import { describe, it, expect } from "vitest";

import {
  IDLE_VOICE_ACTIVITY,
  VAD_AUTO_STOP_VISUAL_GRACE_MS,
  buildVoiceActivitySnapshot,
  voiceActivitySnapshotsEqual,
} from "./activity";
import { createVadRefs } from "./vad";
import type { VadRefs } from "./vad";
import type { VoiceActivitySnapshot } from "./types";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeVad(state: VadRefs["state"], speechThreshold = 0.06, silenceThreshold = 0.02): VadRefs {
  const vad = createVadRefs();
  vad.state = state;
  vad.speechThreshold = speechThreshold;
  vad.silenceThreshold = silenceThreshold;
  vad.silenceStart = 0;
  return vad;
}

const BASE_INPUT = {
  vadActive: true,
  rms: 0.1,
  audioLevel: 0.5,
  nowMs: 1000,
  silenceTimeoutMs: 2000,
  voiceMode: "one-shot" as const,
};

// ---------------------------------------------------------------------------
// IDLE_VOICE_ACTIVITY constant
// ---------------------------------------------------------------------------

describe("IDLE_VOICE_ACTIVITY", () => {
  it("has phase = 'idle'", () => {
    expect(IDLE_VOICE_ACTIVITY.phase).toBe("idle");
  });

  it("has all numeric fields set to 0", () => {
    expect(IDLE_VOICE_ACTIVITY.audioLevel).toBe(0);
    expect(IDLE_VOICE_ACTIVITY.rms).toBe(0);
    expect(IDLE_VOICE_ACTIVITY.speechThreshold).toBe(0);
    expect(IDLE_VOICE_ACTIVITY.silenceThreshold).toBe(0);
    expect(IDLE_VOICE_ACTIVITY.silenceElapsedMs).toBe(0);
    expect(IDLE_VOICE_ACTIVITY.silenceTimeoutMs).toBe(0);
    expect(IDLE_VOICE_ACTIVITY.autoStopProgress).toBe(0);
  });

  it("has autoStopVisible = false", () => {
    expect(IDLE_VOICE_ACTIVITY.autoStopVisible).toBe(false);
  });
});

describe("VAD_AUTO_STOP_VISUAL_GRACE_MS", () => {
  it("is 300ms", () => {
    expect(VAD_AUTO_STOP_VISUAL_GRACE_MS).toBe(300);
  });
});

// ---------------------------------------------------------------------------
// buildVoiceActivitySnapshot — not active / idle fallback
// ---------------------------------------------------------------------------

describe("buildVoiceActivitySnapshot — inactive/idle", () => {
  it("returns idle when vadActive = false", () => {
    const snapshot = buildVoiceActivitySnapshot({
      ...BASE_INPUT,
      vadActive: false,
      vad: makeVad("waitingForSpeech"),
    });
    expect(snapshot.phase).toBe("idle");
  });

  it("returns idle when vad.state = 'idle'", () => {
    const snapshot = buildVoiceActivitySnapshot({
      ...BASE_INPUT,
      vad: makeVad("idle"),
    });
    expect(snapshot.phase).toBe("idle");
  });

  it("clamps audioLevel to [0, 1] in the idle path", () => {
    const s1 = buildVoiceActivitySnapshot({ ...BASE_INPUT, vadActive: false, vad: makeVad("idle"), audioLevel: 2 });
    expect(s1.audioLevel).toBe(1);
    const s2 = buildVoiceActivitySnapshot({ ...BASE_INPUT, vadActive: false, vad: makeVad("idle"), audioLevel: -1 });
    expect(s2.audioLevel).toBe(0);
    const s3 = buildVoiceActivitySnapshot({ ...BASE_INPUT, vadActive: false, vad: makeVad("idle"), audioLevel: Infinity });
    expect(s3.audioLevel).toBe(0);
  });

  it("clamps negative rms to 0 in the idle path", () => {
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vadActive: false, vad: makeVad("idle"), rms: -0.5 });
    expect(s.rms).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// buildVoiceActivitySnapshot — per-state phases
// ---------------------------------------------------------------------------

describe("buildVoiceActivitySnapshot — calibrating", () => {
  it("returns phase = 'calibrating'", () => {
    const vad = makeVad("calibrating");
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad });
    expect(s.phase).toBe("calibrating");
    expect(s.audioLevel).toBeCloseTo(0.5, 5);
    expect(s.rms).toBeCloseTo(0.1, 5);
  });
});

describe("buildVoiceActivitySnapshot — waitingForSpeech", () => {
  it("returns phase = 'waiting-for-speech'", () => {
    const vad = makeVad("waitingForSpeech");
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad });
    expect(s.phase).toBe("waiting-for-speech");
  });

  it("includes correct thresholds", () => {
    const vad = makeVad("waitingForSpeech", 0.12, 0.04);
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad });
    expect(s.speechThreshold).toBe(0.12);
    expect(s.silenceThreshold).toBe(0.04);
  });
});

describe("buildVoiceActivitySnapshot — speechDetected", () => {
  it("returns phase = 'speech'", () => {
    const vad = makeVad("speechDetected");
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad });
    expect(s.phase).toBe("speech");
    expect(s.autoStopProgress).toBe(0);
    expect(s.autoStopVisible).toBe(false);
  });
});

describe("buildVoiceActivitySnapshot — watchingSilence", () => {
  it("returns phase = 'silence'", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 500;
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad, nowMs: 1000, silenceTimeoutMs: 2000 });
    expect(s.phase).toBe("silence");
  });

  it("computes silenceElapsedMs = nowMs - vad.silenceStart", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 500;
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad, nowMs: 1000, silenceTimeoutMs: 2000 });
    expect(s.silenceElapsedMs).toBe(500);
  });

  it("clamps silenceElapsedMs to 0 when nowMs < silenceStart", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 2000;
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad, nowMs: 1000, silenceTimeoutMs: 2000 });
    expect(s.silenceElapsedMs).toBe(0);
  });

  it("computes autoStopProgress as silenceElapsedMs / silenceTimeoutMs (clamped)", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 0;
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad, nowMs: 1000, silenceTimeoutMs: 2000 });
    expect(s.autoStopProgress).toBeCloseTo(0.5, 5);
  });

  it("clamps autoStopProgress to 1 when elapsed exceeds timeout", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 0;
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad, nowMs: 5000, silenceTimeoutMs: 2000 });
    expect(s.autoStopProgress).toBe(1);
  });

  it("autoStopProgress = 0 when silenceTimeoutMs = 0", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 0;
    const s = buildVoiceActivitySnapshot({ ...BASE_INPUT, vad, nowMs: 1000, silenceTimeoutMs: 0 });
    expect(s.autoStopProgress).toBe(0);
  });

  it("autoStopVisible = true in one-shot mode after grace period", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 0;
    const s = buildVoiceActivitySnapshot({
      ...BASE_INPUT,
      vad,
      nowMs: VAD_AUTO_STOP_VISUAL_GRACE_MS,
      silenceTimeoutMs: 2000,
      voiceMode: "one-shot",
    });
    expect(s.autoStopVisible).toBe(true);
  });

  it("autoStopVisible = false before grace period in one-shot mode", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 0;
    const s = buildVoiceActivitySnapshot({
      ...BASE_INPUT,
      vad,
      nowMs: VAD_AUTO_STOP_VISUAL_GRACE_MS - 1,
      silenceTimeoutMs: 2000,
      voiceMode: "one-shot",
    });
    expect(s.autoStopVisible).toBe(false);
  });

  it("autoStopVisible = false in persistent mode even after grace period", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 0;
    const s = buildVoiceActivitySnapshot({
      ...BASE_INPUT,
      vad,
      nowMs: 5000,
      silenceTimeoutMs: 2000,
      voiceMode: "persistent",
    });
    expect(s.autoStopVisible).toBe(false);
  });

  it("autoStopVisible = false when silenceTimeoutMs = 0", () => {
    const vad = makeVad("watchingSilence");
    vad.silenceStart = 0;
    const s = buildVoiceActivitySnapshot({
      ...BASE_INPUT,
      vad,
      nowMs: 5000,
      silenceTimeoutMs: 0,
      voiceMode: "one-shot",
    });
    expect(s.autoStopVisible).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// voiceActivitySnapshotsEqual
// ---------------------------------------------------------------------------

describe("voiceActivitySnapshotsEqual", () => {
  const base: VoiceActivitySnapshot = {
    phase: "idle",
    audioLevel: 0.5,
    rms: 0.05,
    speechThreshold: 0.06,
    silenceThreshold: 0.02,
    silenceElapsedMs: 100,
    silenceTimeoutMs: 2000,
    autoStopProgress: 0.05,
    autoStopVisible: false,
  };

  it("returns true for identical snapshots", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base })).toBe(true);
  });

  it("returns false when phase differs", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, phase: "speech" })).toBe(false);
  });

  it("returns true when audioLevel differs by < 0.01", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, audioLevel: base.audioLevel + 0.009 })).toBe(true);
  });

  it("returns false when audioLevel differs by >= 0.01", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, audioLevel: base.audioLevel + 0.01 })).toBe(false);
  });

  it("returns true when rms differs by < 0.001", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, rms: base.rms + 0.0009 })).toBe(true);
  });

  it("returns false when rms differs by >= 0.001", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, rms: base.rms + 0.001 })).toBe(false);
  });

  it("returns true when speechThreshold differs by < 0.001", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, speechThreshold: base.speechThreshold + 0.0009 })).toBe(true);
  });

  it("returns false when speechThreshold differs by >= 0.001", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, speechThreshold: base.speechThreshold + 0.001 })).toBe(false);
  });

  it("returns true when silenceThreshold differs by < 0.001", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, silenceThreshold: base.silenceThreshold + 0.0009 })).toBe(true);
  });

  it("returns false when silenceThreshold differs by >= 0.001", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, silenceThreshold: base.silenceThreshold + 0.001 })).toBe(false);
  });

  it("returns true when silenceElapsedMs differs by < 16", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, silenceElapsedMs: base.silenceElapsedMs + 15 })).toBe(true);
  });

  it("returns false when silenceElapsedMs differs by >= 16", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, silenceElapsedMs: base.silenceElapsedMs + 16 })).toBe(false);
  });

  it("returns false when silenceTimeoutMs differs (exact comparison)", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, silenceTimeoutMs: base.silenceTimeoutMs + 1 })).toBe(false);
  });

  it("returns true when autoStopProgress differs by < 0.01", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, autoStopProgress: base.autoStopProgress + 0.009 })).toBe(true);
  });

  it("returns false when autoStopProgress differs by >= 0.01", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, autoStopProgress: base.autoStopProgress + 0.01 })).toBe(false);
  });

  it("returns false when autoStopVisible differs", () => {
    expect(voiceActivitySnapshotsEqual(base, { ...base, autoStopVisible: true })).toBe(false);
  });
});
