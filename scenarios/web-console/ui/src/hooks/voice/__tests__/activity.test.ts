import { describe, expect, it } from "vitest";
import { buildVoiceActivitySnapshot, IDLE_VOICE_ACTIVITY, VAD_AUTO_STOP_VISUAL_GRACE_MS } from "../activity";
import { createVadRefs } from "../vad";

describe("buildVoiceActivitySnapshot", () => {
  it("returns an idle snapshot when VAD is inactive", () => {
    const vad = createVadRefs();
    const snapshot = buildVoiceActivitySnapshot({
      vadActive: false,
      vad,
      rms: 0.25,
      audioLevel: 2,
      nowMs: 1_000,
      silenceTimeoutMs: 2_000,
      voiceMode: "one-shot",
    });

    expect(snapshot).toEqual({ ...IDLE_VOICE_ACTIVITY, audioLevel: 1, rms: 0.25 });
  });

  it("maps VAD states to UI phases", () => {
    const vad = createVadRefs();
    vad.state = "calibrating";
    expect(buildVoiceActivitySnapshot({
      vadActive: true,
      vad,
      rms: 0,
      audioLevel: 0,
      nowMs: 0,
      silenceTimeoutMs: 2_000,
      voiceMode: "one-shot",
    }).phase).toBe("calibrating");

    vad.state = "waitingForSpeech";
    expect(buildVoiceActivitySnapshot({
      vadActive: true,
      vad,
      rms: 0,
      audioLevel: 0,
      nowMs: 0,
      silenceTimeoutMs: 2_000,
      voiceMode: "one-shot",
    }).phase).toBe("waiting-for-speech");

    vad.state = "speechDetected";
    expect(buildVoiceActivitySnapshot({
      vadActive: true,
      vad,
      rms: 0.2,
      audioLevel: 0.8,
      nowMs: 0,
      silenceTimeoutMs: 2_000,
      voiceMode: "one-shot",
    }).phase).toBe("speech");
  });

  it("grace-gates one-shot silence auto-stop visibility", () => {
    const vad = createVadRefs();
    vad.state = "watchingSilence";
    vad.silenceStart = 1_000;

    const beforeGrace = buildVoiceActivitySnapshot({
      vadActive: true,
      vad,
      rms: 0.001,
      audioLevel: 0.004,
      nowMs: 1_000 + VAD_AUTO_STOP_VISUAL_GRACE_MS - 1,
      silenceTimeoutMs: 2_000,
      voiceMode: "one-shot",
    });
    expect(beforeGrace.phase).toBe("silence");
    expect(beforeGrace.autoStopVisible).toBe(false);

    const afterGrace = buildVoiceActivitySnapshot({
      vadActive: true,
      vad,
      rms: 0.001,
      audioLevel: 0.004,
      nowMs: 2_000,
      silenceTimeoutMs: 2_000,
      voiceMode: "one-shot",
    });
    expect(afterGrace.autoStopVisible).toBe(true);
    expect(afterGrace.autoStopProgress).toBe(0.5);
  });

  it("does not expose one-shot auto-stop visibility in persistent mode", () => {
    const vad = createVadRefs();
    vad.state = "watchingSilence";
    vad.silenceStart = 1_000;

    const snapshot = buildVoiceActivitySnapshot({
      vadActive: true,
      vad,
      rms: 0.001,
      audioLevel: 0.004,
      nowMs: 2_500,
      silenceTimeoutMs: 2_000,
      voiceMode: "persistent",
    });

    expect(snapshot.phase).toBe("silence");
    expect(snapshot.autoStopProgress).toBe(0.75);
    expect(snapshot.autoStopVisible).toBe(false);
  });

  it("clamps silence progress", () => {
    const vad = createVadRefs();
    vad.state = "watchingSilence";
    vad.silenceStart = 1_000;

    const snapshot = buildVoiceActivitySnapshot({
      vadActive: true,
      vad,
      rms: 0,
      audioLevel: -1,
      nowMs: 4_000,
      silenceTimeoutMs: 2_000,
      voiceMode: "one-shot",
    });

    expect(snapshot.audioLevel).toBe(0);
    expect(snapshot.autoStopProgress).toBe(1);
  });
});
