import { describe, expect, it } from 'vitest';
import type { ReplayFrame as ReplayMovieFrame } from '@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb';
import { toReplayFrame } from '@/export/frameMapping';

describe('toReplayFrame pointer geometry', () => {
  it('preserves CSS viewport and marks pointer samples observed only with geometry', () => {
    const mapped = toReplayFrame({
      index: 0,
      stepIndex: 0,
      viewport: { width: 1440, height: 900 },
      clickPosition: { x: 720, y: 450 },
    } as ReplayMovieFrame, 0, new Map());

    expect(mapped.viewport).toEqual({ width: 1440, height: 900 });
    expect(mapped.clickPosition).toEqual({ x: 720, y: 450 });
    expect(mapped.cursorProvenance).toBe('observed');
  });

  it('does not present pointer coordinates without viewport metadata as observed replay data', () => {
    const mapped = toReplayFrame({
      index: 0,
      stepIndex: 0,
      clickPosition: { x: 720, y: 450 },
    } as ReplayMovieFrame, 0, new Map());

    expect(mapped.viewport).toBeUndefined();
    expect(mapped.cursorProvenance).toBe('missing');
  });
});
