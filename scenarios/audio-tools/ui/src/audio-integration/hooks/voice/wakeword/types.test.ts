// types.ts contains only interfaces (no runtime logic) plus exported constants.
// These tests guard that the constant values remain at their specified tuning values
// so an accidental edit is caught immediately.

import { describe, it, expect } from "vitest";

import {
  NUM_MFCC_COEFFICIENTS,
  FRAME_LENGTH_MS,
  FRAME_HOP_MS,
  MEL_FILTER_COUNT,
  MEL_LOW_HZ,
  MEL_HIGH_HZ,
  DTW_BAND_RATIO,
  DEFAULT_WAKE_WORD_THRESHOLD,
  MIN_ENROLLMENT_SAMPLES,
  MAX_ENROLLMENT_SAMPLES,
} from "./types";

describe("wake-word tunable constants", () => {
  it("NUM_MFCC_COEFFICIENTS is 13", () => {
    expect(NUM_MFCC_COEFFICIENTS).toBe(13);
  });

  it("FRAME_LENGTH_MS is 25", () => {
    expect(FRAME_LENGTH_MS).toBe(25);
  });

  it("FRAME_HOP_MS is 10", () => {
    expect(FRAME_HOP_MS).toBe(10);
  });

  it("MEL_FILTER_COUNT is 26", () => {
    expect(MEL_FILTER_COUNT).toBe(26);
  });

  it("MEL_LOW_HZ is 300", () => {
    expect(MEL_LOW_HZ).toBe(300);
  });

  it("MEL_HIGH_HZ is 8000", () => {
    expect(MEL_HIGH_HZ).toBe(8000);
  });

  it("DTW_BAND_RATIO is 0.2", () => {
    expect(DTW_BAND_RATIO).toBe(0.2);
  });

  it("DEFAULT_WAKE_WORD_THRESHOLD is 0.7", () => {
    expect(DEFAULT_WAKE_WORD_THRESHOLD).toBe(0.7);
  });

  it("MIN_ENROLLMENT_SAMPLES is 3", () => {
    expect(MIN_ENROLLMENT_SAMPLES).toBe(3);
  });

  it("MAX_ENROLLMENT_SAMPLES is 5", () => {
    expect(MAX_ENROLLMENT_SAMPLES).toBe(5);
  });

  it("hop < frame length (overlapping windows)", () => {
    expect(FRAME_HOP_MS).toBeLessThan(FRAME_LENGTH_MS);
  });

  it("low frequency < high frequency for the mel filterbank", () => {
    expect(MEL_LOW_HZ).toBeLessThan(MEL_HIGH_HZ);
  });

  it("MIN_ENROLLMENT_SAMPLES <= MAX_ENROLLMENT_SAMPLES", () => {
    expect(MIN_ENROLLMENT_SAMPLES).toBeLessThanOrEqual(MAX_ENROLLMENT_SAMPLES);
  });
});
