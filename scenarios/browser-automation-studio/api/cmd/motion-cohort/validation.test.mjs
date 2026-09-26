import assert from 'node:assert/strict';
import test from 'node:test';
import { validateObservation } from './validation.mjs';

const maxFrameBytes = 12 * 1024 * 1024 + 4 * 1024;

function passingObservation() {
  return {
    baseline: {
      durationMs: 300_000,
      renderedFrames: 9_000,
      uniqueFixtureFrames: 9_000,
      renderedFps: 30,
      p95FrameAgeMs: 50,
      maxFrameBytes: 8_000,
      p95DecodeMs: 5,
      maxDecodeMs: 10,
    },
    slowReader: {
      durationMs: 18_000,
      stallCount: 3,
      receivedFrames: 540,
      decodedFrames: 210,
      renderedFrames: 210,
      maxConcurrentDecodes: 1,
      maxApiQueueBytes: 9_440_256,
      samples: [{ frameAgeMs: 341, frameBytes: 8_000 }],
    },
  };
}

test('accepts a slow-reader window when every decoded frame is rendered', () => {
  assert.doesNotThrow(() => validateObservation(passingObservation(), maxFrameBytes));
});

test('accepts one-frame finite-window uncertainty with the full five-minute and frame floors', () => {
  const observation = passingObservation();
  observation.baseline.durationMs = 300_054;
  observation.baseline.renderedFrames = 9_001;
  observation.baseline.uniqueFixtureFrames = 9_001;
  observation.baseline.renderedFps = 9_001 / (300_054 / 1000);
  assert.doesNotThrow(() => validateObservation(observation, maxFrameBytes));
  observation.baseline.renderedFrames = 8_999;
  assert.throws(() => validateObservation(observation, maxFrameBytes), /five-minute live motion/);
  observation.baseline.renderedFrames = 9_001;
  observation.baseline.durationMs = 301_000;
  observation.baseline.renderedFps = 9_001 / 301;
  assert.throws(() => validateObservation(observation, maxFrameBytes), /five-minute live motion/);
});

test('rejects slow-reader windows without a receive-to-render backlog', () => {
  const observation = passingObservation();
  observation.slowReader.receivedFrames = observation.slowReader.renderedFrames;
  assert.throws(() => validateObservation(observation, maxFrameBytes), /slow-reader observation/);
});

test('rejects slow-reader windows whose render count exceeds decode attempts', () => {
  const observation = passingObservation();
  observation.slowReader.decodedFrames = observation.slowReader.renderedFrames - 1;
  assert.throws(() => validateObservation(observation, maxFrameBytes), /slow-reader observation/);
});
