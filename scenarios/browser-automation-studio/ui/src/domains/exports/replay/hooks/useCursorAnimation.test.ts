import { describe, expect, it } from 'vitest';
import { cursorSampleAtProgress } from './useCursorAnimation';
import type { ReplayFrame } from '../types';

describe('cursorSampleAtProgress', () => {
  const frame: ReplayFrame = {
    id: 'timed',
    stepIndex: 0,
    success: true,
    durationMs: 1000,
    cursorProvenance: 'observed',
    cursorTrailSamples: [
      { x: 10, y: 20, elapsedMs: 200 },
      { x: 30, y: 40, elapsedMs: 600 },
    ],
  };

  it('selects the most recent observed sample at replay time', () => {
    expect(cursorSampleAtProgress(frame, 0)).toBeUndefined();
    expect(cursorSampleAtProgress(frame, 0.4)).toEqual({ x: 10, y: 20 });
    expect(cursorSampleAtProgress(frame, 0.8)).toEqual({ x: 30, y: 40 });
  });

  it('reaches the final sample before a frame hold completes', () => {
    const heldFrame = { ...frame, totalDurationMs: 2000 };
    expect(cursorSampleAtProgress(heldFrame, 0.2)).toEqual({ x: 10, y: 20 });
    expect(cursorSampleAtProgress(heldFrame, 0.3)).toEqual({ x: 30, y: 40 });
    expect(cursorSampleAtProgress(heldFrame, 1)).toEqual({ x: 30, y: 40 });
  });

  it('does not expose samples when provenance is missing', () => {
    expect(cursorSampleAtProgress({ ...frame, cursorProvenance: 'missing' }, 1)).toBeUndefined();
  });
});
