import {
  HumanBehavior,
  typeWithDelay,
  typeWithEnhancedVariance,
  moveMouseAlongPath,
} from '../../../src/browser-profile/human-behavior';
import type { BehaviorSettings } from '../../../src/types/browser-profile';
import { sleep } from '../../../src/utils/timing';

jest.mock('../../../src/utils/timing', () => ({
  sleep: jest.fn().mockResolvedValue(undefined),
}));

describe('HumanBehavior', () => {
  const baseSettings: BehaviorSettings = {
    typing_delay_min: 100,
    typing_delay_max: 100,
    typing_start_delay_min: 0,
    typing_start_delay_max: 0,
    typing_paste_threshold: 0,
    typing_variance_enabled: true,
    mouse_movement_style: 'linear',
    mouse_jitter_amount: 0,
    click_delay_min: 0,
    click_delay_max: 0,
    scroll_style: 'smooth',
    scroll_speed_min: 100,
    scroll_speed_max: 200,
    micro_pause_enabled: false,
    micro_pause_min_ms: 0,
    micro_pause_max_ms: 0,
    micro_pause_frequency: 0,
  };

  beforeEach(() => {
    jest.spyOn(Math, 'random').mockReturnValue(0.5);
    (sleep as jest.Mock).mockClear();
  });

  afterEach(() => {
    (Math.random as jest.Mock).mockRestore();
  });

  it('reports whether behavior modifications are enabled', () => {
    const disabled = new HumanBehavior({});
    expect(disabled.isEnabled()).toBe(false);

    const enabled = new HumanBehavior({
      click_delay_max: 10,
    });
    expect(enabled.isEnabled()).toBe(true);

    const movementEnabled = new HumanBehavior({
      mouse_movement_style: 'bezier',
    });
    expect(movementEnabled.isEnabled()).toBe(true);
  });

  it('handles paste thresholds', () => {
    const alwaysPaste = new HumanBehavior({ typing_paste_threshold: -1 });
    expect(alwaysPaste.shouldPaste(1)).toBe(true);

    const alwaysType = new HumanBehavior({ typing_paste_threshold: 0 });
    expect(alwaysType.shouldPaste(100)).toBe(false);

    const threshold = new HumanBehavior({ typing_paste_threshold: 5 });
    expect(threshold.shouldPaste(3)).toBe(false);
    expect(threshold.shouldPaste(6)).toBe(true);
  });

  it('calculates typing delays with variance and digraphs', () => {
    const behavior = new HumanBehavior(baseSettings);

    // Prime the lastChar
    expect(behavior.getTypingDelayForChar('t')).toBe(100);

    // "th" is a fast digraph; with Math.random=0.5 we expect 70ms
    expect(behavior.getTypingDelayForChar('h')).toBe(70);

    // Shifted char should be slower (145% of base with Math.random=0.5)
    behavior.resetTypingState();
    expect(behavior.getTypingDelayForChar('A')).toBe(145);

    // Space gets a quicker multiplier (80% of base with Math.random=0.5)
    behavior.resetTypingState();
    expect(behavior.getTypingDelayForChar(' ')).toBe(80);

    // Number and uncommon symbols apply additional slowdown
    behavior.resetTypingState();
    expect(behavior.getTypingDelayForChar('1')).toBe(120);
    behavior.resetTypingState();
    expect(behavior.getTypingDelayForChar('^')).toBe(232);
  });

  it('returns base delay when variance is disabled', () => {
    const behavior = new HumanBehavior({
      ...baseSettings,
      typing_variance_enabled: false,
    });

    expect(behavior.getTypingDelayForChar('a')).toBe(100);
    expect(behavior.getTypingDelayForChar('b')).toBe(100);
  });

  it('generates mouse paths for each movement style', () => {
    const linear = new HumanBehavior({
      ...baseSettings,
      mouse_movement_style: 'linear',
    });
    const linearPath = linear.generateMousePath({ x: 0, y: 0 }, { x: 10, y: 10 }, 4);
    expect(linearPath[0]).toEqual({ x: 0, y: 0 });
    expect(linearPath[linearPath.length - 1]).toEqual({ x: 10, y: 10 });
    expect(linearPath).toHaveLength(5);

    const bezier = new HumanBehavior({
      ...baseSettings,
      mouse_movement_style: 'bezier',
      mouse_jitter_amount: 0,
    });
    const bezierPath = bezier.generateMousePath({ x: 0, y: 0 }, { x: 10, y: 10 }, 4);
    expect(bezierPath[0]).toEqual({ x: 0, y: 0 });
    expect(bezierPath[bezierPath.length - 1]).toEqual({ x: 10, y: 10 });

    const natural = new HumanBehavior({
      ...baseSettings,
      mouse_movement_style: 'natural',
      mouse_jitter_amount: 0,
    });
    const naturalPath = natural.generateMousePath({ x: 0, y: 0 }, { x: 10, y: 10 }, 4);
    expect(naturalPath[0]).toEqual({ x: 0, y: 0 });
    expect(naturalPath[naturalPath.length - 1]).toEqual({ x: 10, y: 10 });
    expect(naturalPath).toHaveLength(5);
  });

  it('exposes timing helpers and style getters', () => {
    const behavior = new HumanBehavior({
      typing_start_delay_min: 10,
      typing_start_delay_max: 20,
      typing_delay_min: 5,
      typing_delay_max: 15,
      click_delay_min: 3,
      click_delay_max: 3,
      scroll_speed_min: 200,
      scroll_speed_max: 300,
      micro_pause_enabled: true,
      micro_pause_min_ms: 4,
      micro_pause_max_ms: 6,
      micro_pause_frequency: 1,
      mouse_movement_style: 'bezier',
      scroll_style: 'stepped',
    });

    expect(behavior.getTypingStartDelay()).toBe(15);
    expect(behavior.getTypingDelay()).toBe(10);
    expect(behavior.getClickDelay()).toBe(3);
    expect(behavior.getScrollSpeed()).toBe(250);
    expect(behavior.shouldMicroPause()).toBe(true);
    expect(behavior.getMicroPauseDuration()).toBe(5);
    expect(behavior.getMouseMovementStyle()).toBe('bezier');
    expect(behavior.getScrollStyle()).toBe('stepped');
  });
});

describe('HumanBehavior helpers', () => {
  beforeEach(() => {
    (sleep as jest.Mock).mockClear();
  });

  it('types text with delays and micro-pauses', async () => {
    const behavior = new HumanBehavior({
      typing_delay_min: 0,
      typing_delay_max: 0,
      typing_variance_enabled: false,
      micro_pause_enabled: true,
      micro_pause_min_ms: 5,
      micro_pause_max_ms: 5,
      micro_pause_frequency: 1,
    });

    const typeChar = jest.fn().mockResolvedValue(undefined);
    jest.spyOn(behavior, 'getTypingDelay').mockReturnValue(10);
    jest.spyOn(behavior, 'shouldMicroPause').mockReturnValue(true);
    jest.spyOn(behavior, 'getMicroPauseDuration').mockReturnValue(5);

    await typeWithDelay(typeChar, 'ab', behavior);

    expect(typeChar).toHaveBeenCalledTimes(2);
    expect(sleep).toHaveBeenCalledTimes(4);
    expect((sleep as jest.Mock).mock.calls[0]?.[0]).toBe(10);
    expect((sleep as jest.Mock).mock.calls[1]?.[0]).toBe(5);
  });

  it('uses enhanced variance typing with reset', async () => {
    const behavior = new HumanBehavior({
      typing_delay_min: 0,
      typing_delay_max: 0,
      typing_variance_enabled: true,
      micro_pause_enabled: false,
    });

    const typeChar = jest.fn().mockResolvedValue(undefined);
    const resetSpy = jest.spyOn(behavior, 'resetTypingState');
    const delaySpy = jest.spyOn(behavior, 'getTypingDelayForChar').mockReturnValue(15);

    await typeWithEnhancedVariance(typeChar, 'ab', behavior);

    expect(resetSpy).toHaveBeenCalledTimes(1);
    expect(delaySpy).toHaveBeenCalledTimes(2);
    expect(typeChar).toHaveBeenCalledTimes(2);
    expect(sleep).toHaveBeenCalledTimes(2);
  });

  it('moves mouse along a path with timing', async () => {
    const moveMouse = jest.fn().mockResolvedValue(undefined);
    const path = [
      { x: 0, y: 0 },
      { x: 5, y: 5 },
      { x: 10, y: 10 },
    ];

    await moveMouseAlongPath(moveMouse, path, 200);

    expect(moveMouse).toHaveBeenCalledTimes(2);
    expect(moveMouse).toHaveBeenNthCalledWith(1, 5, 5);
    expect(moveMouse).toHaveBeenNthCalledWith(2, 10, 10);
    expect(sleep).toHaveBeenCalledTimes(2);
    expect((sleep as jest.Mock).mock.calls[0]?.[0]).toBe(100);
  });

  it('skips mouse movement for short paths', async () => {
    const moveMouse = jest.fn().mockResolvedValue(undefined);
    await moveMouseAlongPath(moveMouse, [{ x: 0, y: 0 }], 200);

    expect(moveMouse).not.toHaveBeenCalled();
    expect(sleep).not.toHaveBeenCalled();
  });
});
