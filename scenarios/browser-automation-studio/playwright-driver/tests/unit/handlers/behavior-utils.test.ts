import type { BrowserContext, Page, ElementHandle } from 'rebrowser-playwright';
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
import { createTestConfig } from '../../helpers';
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

  it('scrolls exactly to the requested position without human behavior', async () => {
    const evaluate = jest.fn().mockResolvedValueOnce({ x: 0, y: 0 }).mockResolvedValue(undefined);
    const target = { evaluate } as unknown as ElementHandle<Element>;
    await executeHumanScroll(target, 10, 20, null);
    expect(evaluate).toHaveBeenLastCalledWith(expect.any(Function), {
      x: 10,
      y: 20,
      smooth: false,
    });
    expect(mockSleep).not.toHaveBeenCalled();
  });

  it('steps toward both target axes and finishes at the exact destination', async () => {
    const evaluate = jest.fn().mockResolvedValueOnce({ x: 0, y: 0 }).mockResolvedValue(undefined);
    const target = { evaluate } as unknown as ElementHandle<Element>;
    const behavior = {
      getScrollSpeed: () => 10,
      shouldMicroPause: () => true,
      getMicroPauseDuration: () => 3,
    } as unknown as import('../../../src/browser-profile').HumanBehavior;
    await executeHumanScroll(target, 20, 0, behavior, { minStepDelayMs: 1, maxStepDelayMs: 1 });
    expect(evaluate.mock.calls.slice(1).map((call) => call[1])).toEqual([
      { x: 10, y: 0, smooth: false },
      { x: 20, y: 0, smooth: false },
    ]);
    expect(mockSleep).toHaveBeenNthCalledWith(1, 1);
    expect(mockSleep).toHaveBeenNthCalledWith(2, 3);
  });

  it('smooth scroll waits for actual completion and disposes the completion handle', async () => {
    const dispose = jest.fn().mockResolvedValue(undefined);
    const waitForFunction = jest.fn().mockResolvedValue({ dispose });
    const target = {
      evaluate: jest.fn().mockResolvedValue(undefined),
      ownerFrame: jest.fn().mockResolvedValue({ waitForFunction }),
    } as unknown as ElementHandle<Element>;
    const behavior = {
      shouldMicroPause: () => true,
      getMicroPauseDuration: () => 2,
    } as unknown as import('../../../src/browser-profile').HumanBehavior;
    await executeSmoothScroll(target, 10, 20, behavior, 1234);
    expect(waitForFunction).toHaveBeenCalledWith(
      expect.any(Function),
      { element: target, x: 10, y: 20 },
      { timeout: 1234, polling: 'raf' }
    );
    expect(dispose).toHaveBeenCalledTimes(1);
    expect(mockSleep).toHaveBeenCalledTimes(1);
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
