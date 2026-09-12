import { describe, expect, it, vi } from 'vitest';
import { onProfilerRender } from './profiler';

describe('onProfilerRender', () => {
  it('records a React render measurement', () => {
    const measure = vi.spyOn(performance, 'measure');
    const now = vi.spyOn(performance, 'now').mockReturnValue(100);

    onProfilerRender('public-landing', 'mount', 12, 0, 0, 0);

    expect(measure).toHaveBeenCalledWith('⚛ public-landing (mount)', {
      start: 88,
      duration: 12,
    });
    now.mockRestore();
    measure.mockRestore();
  });

  it('does not let browser measurement failures affect rendering', () => {
    const measure = vi.spyOn(performance, 'measure').mockImplementation(() => {
      throw new Error('unsupported');
    });

    expect(() => {
      onProfilerRender('public-landing', 'update', 4, 0, 0, 0);
    }).not.toThrow();

    measure.mockRestore();
  });
});
