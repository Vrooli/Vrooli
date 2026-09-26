import { describe, expect, it } from "vitest";
import {
  advanceMeterEnvelope,
  LEVEL_ANALYSER_FFT_SIZE,
  LEVEL_METER_RANGE_DB,
  LEVEL_METER_RELEASE_MS,
  LEVEL_TICK_MS,
  meterLevelFromEnvelope,
} from "./audioUtils";
import { VAD_MIN_SILENCE_THRESHOLD } from "./vad";

// The meter used to read a 1.33 ms slice of audio (fftSize 128, and only
// `frequencyBinCount` — half of it — was ever read) and map it with a fixed
// `min(1, rms * 4)`. A speaking voice pulses every 3.9-11.8 ms, so successive
// reads landed at effectively random points in a peaky waveform and the bar
// reported the gaps between glottal pulses rather than the phrase. Speech
// detection kept working throughout, because the VAD compares the same RMS
// against thresholds it derives from the running noise floor and is therefore
// scale-invariant; the meter was not.
//
// These tests pin the two properties that fixed it: the window is long enough
// to be an envelope, and the mapping shares the VAD's adaptive floor.

/** RMS of speech measured over a 30 ms window, from the repo's speech fixture. */
const QUIET_SPEECH_RMS = 0.035;
const NORMAL_SPEECH_RMS = 0.065;
const LOUD_SPEECH_RMS = 0.15;

describe("level meter window", () => {
  it("covers at least one full glottal period at 48 kHz", () => {
    // The lowest speaking fundamental frequency worth metering is ~85 Hz, one
    // period every 11.8 ms. A window shorter than that cannot see an envelope.
    const windowMs = (LEVEL_ANALYSER_FFT_SIZE / 48_000) * 1000;
    expect(windowMs).toBeGreaterThanOrEqual(11.8);
  });
});

describe("meterLevelFromEnvelope", () => {
  const floor = VAD_MIN_SILENCE_THRESHOLD;

  it("reads empty at and below the noise floor", () => {
    expect(meterLevelFromEnvelope(floor, floor)).toBe(0);
    expect(meterLevelFromEnvelope(floor / 2, floor)).toBe(0);
    expect(meterLevelFromEnvelope(0, floor)).toBe(0);
  });

  it("puts ordinary speech in the visible middle of the bar, not on the floor", () => {
    const level = meterLevelFromEnvelope(NORMAL_SPEECH_RMS, floor);
    // The old mapping gave `min(1, 0.065 * 4)` = 0.26 for this input, and on a
    // 44 px control that is 11 px — but only when the window happened to catch
    // a pulse. The floor-relative mapping is both higher and stable.
    expect(level).toBeGreaterThan(0.35);
    expect(level).toBeLessThan(0.75);
  });

  it("is monotonic across the speaking range and saturates on loud speech", () => {
    const quiet = meterLevelFromEnvelope(QUIET_SPEECH_RMS, floor);
    const normal = meterLevelFromEnvelope(NORMAL_SPEECH_RMS, floor);
    const loud = meterLevelFromEnvelope(LOUD_SPEECH_RMS, floor);
    expect(quiet).toBeLessThan(normal);
    expect(normal).toBeLessThan(loud);
    expect(quiet).toBeGreaterThan(0);
    expect(loud).toBeLessThanOrEqual(1);
  });

  it("self-calibrates: the same speech-to-noise ratio reads the same in any room", () => {
    // A noisy room raises the VAD's floor. A meter anchored to an absolute
    // scale would peg high there; one anchored to the floor should not.
    const quietRoom = meterLevelFromEnvelope(0.08, 0.02);
    const noisyRoom = meterLevelFromEnvelope(0.32, 0.08);
    expect(noisyRoom).toBeCloseTo(quietRoom, 6);
  });

  it("never exceeds the bar, and treats a corrupt reading as no signal", () => {
    expect(meterLevelFromEnvelope(10, floor)).toBe(1);
    // A non-finite sample is garbage, not maximum loudness. Reading it as a
    // full bar would flash the meter on a numeric fault; reading it as empty
    // lets the next good tick correct it.
    expect(meterLevelFromEnvelope(Number.NaN, floor)).toBe(0);
    expect(meterLevelFromEnvelope(Number.POSITIVE_INFINITY, floor)).toBe(0);
    expect(meterLevelFromEnvelope(0.5, 0)).toBe(0);
    expect(meterLevelFromEnvelope(0.5, Number.NaN)).toBe(0);
  });

  it("spans the full bar over its declared dynamic range", () => {
    const top = floor * 10 ** (LEVEL_METER_RANGE_DB / 20);
    expect(meterLevelFromEnvelope(top, floor)).toBeCloseTo(1, 6);
  });
});

describe("advanceMeterEnvelope", () => {
  it("adopts a rising level on the tick it appears", () => {
    expect(advanceMeterEnvelope(0.01, 0.09, LEVEL_TICK_MS)).toBe(0.09);
  });

  it("decays a falling level instead of dropping to the gap between pulses", () => {
    const afterOneTick = advanceMeterEnvelope(0.09, 0.001, LEVEL_TICK_MS);
    expect(afterOneTick).toBeLessThan(0.09);
    expect(afterOneTick).toBeGreaterThan(0.05);
  });

  it("holds a phrase together across a single silent window", () => {
    // One unlucky window landing between glottal pulses must not blank the bar.
    // The property is retention, not a magic number: most of the height must
    // survive a single dead tick.
    const speaking = advanceMeterEnvelope(0, NORMAL_SPEECH_RMS, LEVEL_TICK_MS);
    const afterGap = advanceMeterEnvelope(speaking, 0.002, LEVEL_TICK_MS);
    const before = meterLevelFromEnvelope(speaking, VAD_MIN_SILENCE_THRESHOLD);
    const after = meterLevelFromEnvelope(afterGap, VAD_MIN_SILENCE_THRESHOLD);
    expect(after / before).toBeGreaterThan(0.7);
  });

  it("reaches the floor once speech actually stops", () => {
    let envelope = NORMAL_SPEECH_RMS;
    // Five release constants is a complete decay by any practical measure.
    for (let elapsed = 0; elapsed < LEVEL_METER_RELEASE_MS * 5; elapsed += LEVEL_TICK_MS) {
      envelope = advanceMeterEnvelope(envelope, 0.0005, LEVEL_TICK_MS);
    }
    expect(meterLevelFromEnvelope(envelope, VAD_MIN_SILENCE_THRESHOLD)).toBe(0);
  });

  it("ignores a non-finite or negative sample rather than poisoning the envelope", () => {
    expect(advanceMeterEnvelope(0.09, Number.NaN, LEVEL_TICK_MS)).toBe(0.09);
    expect(advanceMeterEnvelope(0.09, -1, LEVEL_TICK_MS)).toBe(0.09);
  });
});
