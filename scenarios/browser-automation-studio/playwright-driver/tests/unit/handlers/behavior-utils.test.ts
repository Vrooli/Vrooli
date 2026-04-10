import type { BrowserContext, Page } from 'rebrowser-playwright';
import {
  resolveTimeout,
  resolveTimeoutFromContext,
  getBehaviorFromBrowserContext,
  getBehaviorFromContext,
  applyPreActionDelay,
  applyPostActionPause,
  executeHumanScroll,
  executeSmoothScroll,
  moveMouseNaturally,
  getElementCenter,
} from '../../../src/handlers/behavior-utils';
import { createMockPage, createTestConfig } from '../../helpers';
import { BEHAVIOR_SETTINGS_KEY } from '../../../src/browser-profile';
import type { HandlerContext } from '../../../src/handlers/base';
import { sleep } from '../../../src/utils';

jest.mock('../../../src/utils', () => ({
  sleep: jest.fn().mockResolvedValue(undefined),
}));

const mockSleep = sleep as jest.MockedFunction<typeof sleep>;

describe('behavior-utils', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('resolveTimeout respects explicit parameter first', () => {
    const config = createTestConfig();
    expect(resolveTimeout(1234, config, 'navigation')).toBe(1234);
  });

  it('resolveTimeout falls back to config per category', () => {
    const config = createTestConfig({ execution: { navigationTimeoutMs: 45001 } });
    expect(resolveTimeout(undefined, config, 'navigation')).toBe(45001);
  });

  it('resolveTimeoutFromContext uses context config', () => {
    const config = createTestConfig({ execution: { assertionTimeoutMs: 2222 } });
    const context = { config } as HandlerContext;
    expect(resolveTimeoutFromContext(undefined, context, 'assertion')).toBe(2222);
  });

  it('getBehaviorFromBrowserContext returns null when disabled', () => {
    const context = {} as BrowserContext;
    expect(getBehaviorFromBrowserContext(context)).toBeNull();
  });

  it('getBehaviorFromBrowserContext returns behavior when enabled', () => {
    const context = {
      [BEHAVIOR_SETTINGS_KEY]: {
        click_delay_max: 10,
        click_delay_min: 5,
        micro_pause_enabled: true,
        micro_pause_frequency: 1,
        micro_pause_min_ms: 1,
        micro_pause_max_ms: 2,
        mouse_movement_style: 'linear',
        scroll_speed_min: 100,
        scroll_speed_max: 200,
        typing_delay_min: 0,
        typing_delay_max: 0,
        typing_start_delay_min: 0,
        typing_start_delay_max: 0,
        typing_paste_threshold: 0,
        typing_variance_enabled: false,
        mouse_jitter_amount: 0,
        scroll_style: 'smooth',
      },
    } as BrowserContext;

    const behavior = getBehaviorFromBrowserContext(context);
    expect(behavior).not.toBeNull();
    expect(behavior?.isEnabled()).toBe(true);
  });

  it('getBehaviorFromContext uses page.context', () => {
    const browserContext = {
      [BEHAVIOR_SETTINGS_KEY]: {
        click_delay_max: 10,
        click_delay_min: 5,
        micro_pause_enabled: true,
        micro_pause_frequency: 1,
        micro_pause_min_ms: 1,
        micro_pause_max_ms: 2,
        mouse_movement_style: 'linear',
        scroll_speed_min: 100,
        scroll_speed_max: 200,
        typing_delay_min: 0,
        typing_delay_max: 0,
        typing_start_delay_min: 0,
        typing_start_delay_max: 0,
        typing_paste_threshold: 0,
        typing_variance_enabled: false,
        mouse_jitter_amount: 0,
        scroll_style: 'smooth',
      },
    } as BrowserContext;

    const context = {
      page: { context: jest.fn().mockReturnValue(browserContext) },
    } as unknown as HandlerContext;

    const behavior = getBehaviorFromContext(context);
    expect(behavior).not.toBeNull();
  });

  it('applyPreActionDelay applies delay and micro-pause', async () => {
    const behavior = {
      shouldMicroPause: () => true,
      getMicroPauseDuration: () => 15,
    } as unknown as import('../../../src/browser-profile').HumanBehavior;

    await applyPreActionDelay(behavior, () => 20);

    expect(mockSleep).toHaveBeenCalledTimes(2);
    expect(mockSleep).toHaveBeenCalledWith(20);
    expect(mockSleep).toHaveBeenCalledWith(15);
  });

  it('applyPostActionPause applies micro-pause when enabled', async () => {
    const behavior = {
      shouldMicroPause: () => true,
      getMicroPauseDuration: () => 12,
    } as unknown as import('../../../src/browser-profile').HumanBehavior;

    await applyPostActionPause(behavior);

    expect(mockSleep).toHaveBeenCalledWith(12);
  });

  it('executeHumanScroll performs instant scroll when no behavior', async () => {
    const page = {
      evaluate: jest.fn().mockResolvedValue(undefined),
    } as unknown as Page;

    await executeHumanScroll(page, 10, 20, null);

    expect(page.evaluate).toHaveBeenCalledTimes(1);
  });

  it('executeHumanScroll short-circuits when already close to target', async () => {
    const page = {
      evaluate: jest
        .fn()
        .mockResolvedValueOnce({ x: 0, y: 0 })
        .mockResolvedValueOnce(undefined),
    } as unknown as Page;

    const behavior = {
      getScrollSpeed: () => 5,
      shouldMicroPause: () => false,
    } as unknown as import('../../../src/browser-profile').HumanBehavior;

    await executeHumanScroll(page, 5, 0, behavior);

    expect(page.evaluate).toHaveBeenCalledTimes(2);
  });

  it('executeHumanScroll performs stepped scroll with delays', async () => {
    const page = {
      evaluate: jest
        .fn()
        .mockResolvedValueOnce({ x: 0, y: 0 })
        .mockResolvedValue(undefined),
    } as unknown as Page;

    const behavior = {
      getScrollSpeed: () => 10,
      shouldMicroPause: () => true,
      getMicroPauseDuration: () => 3,
    } as unknown as import('../../../src/browser-profile').HumanBehavior;

    await executeHumanScroll(page, 20, 0, behavior, { minStepDelayMs: 1, maxStepDelayMs: 1 });

    expect(page.evaluate).toHaveBeenCalled();
    expect(mockSleep).toHaveBeenCalled();
  });

  it('executeSmoothScroll applies micro-pause and waits for estimated duration', async () => {
    const page = {
      evaluate: jest
        .fn()
        .mockResolvedValueOnce(undefined)
        .mockResolvedValueOnce({ x: 0, y: 0 }),
    } as unknown as Page;

    const behavior = {
      shouldMicroPause: () => true,
      getMicroPauseDuration: () => 2,
    } as unknown as import('../../../src/browser-profile').HumanBehavior;

    await executeSmoothScroll(page, 10, 0, behavior);

    expect(mockSleep).toHaveBeenCalledTimes(2);
  });

  it('moveMouseNaturally falls back to direct move without behavior', async () => {
    const page = {
      mouse: { move: jest.fn().mockResolvedValue(undefined) },
    } as unknown as Page;

    await moveMouseNaturally(page, 5, 5, null);

    expect(page.mouse.move).toHaveBeenCalledWith(5, 5);
  });

  it('moveMouseNaturally follows generated path with delays', async () => {
    const page = {
      mouse: { move: jest.fn().mockResolvedValue(undefined) },
    } as unknown as Page;

    const behavior = {
      getMouseMovementStyle: () => 'natural',
      generateMousePath: () => [{ x: 0, y: 0 }, { x: 5, y: 5 }, { x: 10, y: 10 }],
    } as unknown as import('../../../src/browser-profile').HumanBehavior;

    await moveMouseNaturally(page, 10, 10, behavior, { durationMs: 10 });

    expect(page.mouse.move).toHaveBeenCalledTimes(2);
    expect(mockSleep).toHaveBeenCalled();
  });

  it('getElementCenter returns null when element is missing', async () => {
    const page = {
      waitForSelector: jest.fn().mockResolvedValue(null),
    } as unknown as Page;

    const result = await getElementCenter(page, '#missing');
    expect(result).toBeNull();
  });

  it('getElementCenter returns center point when element found', async () => {
    const page = {
      waitForSelector: jest.fn().mockResolvedValue({
        boundingBox: jest.fn().mockResolvedValue({ x: 10, y: 20, width: 30, height: 40 }),
      }),
    } as unknown as Page;

    const result = await getElementCenter(page, '#target');
    expect(result).toEqual({ x: 25, y: 40 });
  });
});
