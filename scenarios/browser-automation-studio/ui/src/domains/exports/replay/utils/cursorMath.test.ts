import { describe, expect, it } from 'vitest';
import { interpolatePath } from './cursorMath';

describe('interpolatePath', () => {
  it('travels between nonzero endpoints without overshooting or jumping back', () => {
    const path = [{ x: 0.2, y: 0.3 }, { x: 0.8, y: 0.7 }];
    expect(interpolatePath(path, 0)).toEqual(path[0]);
    expect(interpolatePath(path, 0.5)).toEqual({ x: 0.5, y: 0.5 });
    expect(interpolatePath(path, 1)).toEqual(path[1]);
  });

  it('keeps distance-based timing across unequal and stationary segments', () => {
    const path = [{ x: 10, y: 20 }, { x: 10, y: 20 }, { x: 40, y: 20 }, { x: 40, y: 60 }];
    expect(interpolatePath(path, 30 / 70)).toEqual({ x: 40, y: 20 });
    expect(interpolatePath(path, 50 / 70)).toEqual({ x: 40, y: 40 });
    expect(interpolatePath(path, 1)).toEqual({ x: 40, y: 60 });
  });

  it('preserves reverse travel and clamps the path endpoints', () => {
    const path = [{ x: 0.9, y: 0.8 }, { x: 0.1, y: 0.2 }];
    const midpoint = interpolatePath(path, 0.5);
    expect(midpoint.x).toBeCloseTo(0.5);
    expect(midpoint.y).toBeCloseTo(0.5);
    expect(interpolatePath(path, -1)).toEqual(path[0]);
    const end = interpolatePath(path, 2);
    expect(end.x).toBeCloseTo(0.1);
    expect(end.y).toBeCloseTo(0.2);
  });
});
