import { describe, it, expect, vi } from "vitest";
import { playRecordingStartCue, playRecordingStopCue, playRecordingFaultCue } from "./audioCues";

/** Records the oscillator frequencies a cue schedules, in start order. */
function recordingContext() {
  const started: Array<{ freq: number; at: number }> = [];
  const gain = () => ({
    gain: {
      setValueAtTime: vi.fn(),
      linearRampToValueAtTime: vi.fn(),
      exponentialRampToValueAtTime: vi.fn(),
    },
    connect: vi.fn(() => ({ connect: vi.fn() })),
  });
  const ctx = {
    state: "running",
    currentTime: 0,
    destination: {},
    createOscillator: () => {
      const osc = {
        type: "sine",
        frequency: { value: 0 },
        connect: vi.fn(() => ({ connect: vi.fn() })),
        start: (at: number) => started.push({ freq: osc.frequency.value, at }),
        stop: vi.fn(),
      };
      return osc;
    },
    createGain: gain,
    resume: vi.fn(),
  };
  return { ctx: ctx as unknown as AudioContext, started };
}

async function capture(play: (o: { getContext: () => AudioContext; keepAudioContextAwake: () => void }) => void) {
  const { ctx, started } = recordingContext();
  play({ getContext: () => ctx, keepAudioContextAwake: () => {} });
  // The cue awaits a possible context resume before scheduling.
  await Promise.resolve();
  await Promise.resolve();
  return started.sort((a, b) => a.at - b.at).map((s) => s.freq);
}

describe("voice audio cues", () => {
  it("rises to signal that capture started", async () => {
    expect(await capture(playRecordingStartCue)).toEqual([523, 659]);
  });

  it("falls to signal that capture stopped normally", async () => {
    expect(await capture(playRecordingStopCue)).toEqual([659, 523]);
  });

  // Regression: a turn that ended on a provider fault used to play nothing at
  // all, on the grounds that the "done" chime would misreport a failure. It
  // does — but silence misreports it worse: the mic is dead, the countdown ring
  // is gone, and a speaker who is not watching the screen keeps talking. The
  // fault cue must be audible AND distinguishable from a normal stop.
  it("plays a distinct cue when capture ends on a fault", async () => {
    const fault = await capture(playRecordingFaultCue);
    expect(fault.length).toBeGreaterThan(0);
    expect(fault).not.toEqual(await capture(playRecordingStopCue));
    expect(fault).not.toEqual(await capture(playRecordingStartCue));
  });
});
