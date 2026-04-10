import type { BrowserContext, Page } from 'rebrowser-playwright';
import { InjectionAutoDetector } from '../../../src/recording/injection/auto-detector';
import type { InjectionStrategy, InjectionStrategyOptions } from '../../../src/recording/injection/types';
import { createMockContext, createMockPage } from '../../helpers';
import { logger } from '../../../src/utils';

const createByName = jest.fn();
const factoryConstructor = jest.fn();

jest.mock('../../../src/recording/injection/factory', () => ({
  DEFAULT_STRATEGY_ORDER: ['init-script', 'route-injection'],
  InjectionStrategyFactory: jest.fn().mockImplementation(() => {
    factoryConstructor();
    return { createByName };
  }),
}));

const waitForScriptReady = jest.fn();

jest.mock('../../../src/recording/validation/verification', () => ({
  waitForScriptReady: (...args: unknown[]) => waitForScriptReady(...args),
}));

function createStrategy(name: InjectionStrategy['name']): InjectionStrategy {
  return {
    name,
    initialize: jest.fn().mockResolvedValue(undefined),
    injectScript: jest.fn().mockResolvedValue({
      success: true,
      strategy: name,
      timestamp: new Date().toISOString(),
    }),
    verify: jest.fn().mockResolvedValue(true),
    getStats: jest.fn().mockReturnValue({
      attempted: 0,
      successful: 0,
      failed: 0,
      avgInjectionTimeMs: 0,
      lastInjectionAt: null,
    }),
    resetStats: jest.fn(),
    cleanup: jest.fn().mockResolvedValue(undefined),
    supportsProvider: jest.fn().mockReturnValue(true),
  };
}

describe('InjectionAutoDetector', () => {
  const strategyOptions: Omit<InjectionStrategyOptions, 'onFirstInjection'> = {
    bindingName: '__test',
    logger,
    diagnosticsEnabled: true,
  };

  let context: BrowserContext;
  let page: Page;

  beforeEach(() => {
    createByName.mockReset();
    factoryConstructor.mockClear();
    waitForScriptReady.mockReset();
    page = createMockPage({
      goto: jest.fn().mockResolvedValue(undefined),
      close: jest.fn().mockResolvedValue(undefined),
    });
    context = createMockContext({
      newPage: jest.fn().mockResolvedValue(page),
    });
  });

  it('selects the first strategy that verifies successfully', async () => {
    const strategy = createStrategy('init-script');
    createByName.mockReturnValue(strategy);
    waitForScriptReady.mockResolvedValue({
      loaded: true,
      ready: true,
      inMainContext: true,
      handlersCount: 1,
    });

    const detector = new InjectionAutoDetector({
      logger,
      strategyOrder: ['init-script'],
    });

    const result = await detector.detect(context, strategyOptions);

    expect(result.strategyName).toBe('init-script');
    expect(result.attempts).toHaveLength(1);
    expect(strategy.initialize).toHaveBeenCalled();
    expect(strategy.cleanup).toHaveBeenCalled();
  });

  it('returns null when all strategies fail verification', async () => {
    const first = createStrategy('init-script');
    const second = createStrategy('route-injection');
    createByName.mockImplementation((name: string) => (name === 'init-script' ? first : second));
    waitForScriptReady.mockResolvedValue({
      loaded: false,
      ready: false,
      inMainContext: false,
      handlersCount: 0,
    });

    const detector = new InjectionAutoDetector({
      logger,
      strategyOrder: ['init-script', 'route-injection'],
    });

    const result = await detector.detect(context, strategyOptions);

    expect(result.strategy).toBeNull();
    expect(result.attempts).toHaveLength(2);
    expect(first.cleanup).toHaveBeenCalled();
    expect(second.cleanup).toHaveBeenCalled();
  });

  it('uses cached strategy when it still works', async () => {
    const cached = createStrategy('init-script');
    createByName.mockReturnValue(cached);
    waitForScriptReady.mockResolvedValue({
      loaded: true,
      ready: true,
      inMainContext: true,
      handlersCount: 1,
    });

    const detector = new InjectionAutoDetector({
      logger,
      strategyOrder: ['init-script', 'route-injection'],
    });

    const result = await detector.detectWithCache(context, strategyOptions, 'init-script');

    expect(result.strategyName).toBe('init-script');
    expect(result.attempts).toHaveLength(1);
  });

  it('falls back to full detection when cached strategy fails', async () => {
    const cached = createStrategy('route-injection');
    const fallback = createStrategy('init-script');
    createByName.mockImplementation((name: string) => (name === 'route-injection' ? cached : fallback));

    waitForScriptReady
      .mockResolvedValueOnce({
        loaded: false,
        ready: false,
        inMainContext: false,
        handlersCount: 0,
      })
      .mockResolvedValueOnce({
        loaded: true,
        ready: true,
        inMainContext: true,
        handlersCount: 1,
      });

    const detector = new InjectionAutoDetector({
      logger,
      strategyOrder: ['init-script'],
    });

    const result = await detector.detectWithCache(context, strategyOptions, 'route-injection');

    expect(result.strategyName).toBe('init-script');
    expect(createByName).toHaveBeenCalledWith('route-injection');
    expect(createByName).toHaveBeenCalledWith('init-script');
  });
});
