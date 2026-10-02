import {
  createRecordingContextInitializer,
  type RawBrowserEvent,
} from '../../../src/recording';
import type { BrowserContext, Page, Route } from 'rebrowser-playwright';
import type { Logger } from 'winston';
import { createMockContext, createMockPage } from '../../helpers';

const mockLogger = {
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
} as unknown as Logger;

function contextFixture(overrides: Record<string, unknown> = {}): jest.Mocked<BrowserContext> {
  return createMockContext({
    addInitScript: jest.fn().mockResolvedValue(undefined),
    pages: jest.fn().mockReturnValue([]),
    on: jest.fn(),
    off: jest.fn(),
    ...overrides,
  });
}

describe('RecordingContextInitializer', () => {
  beforeEach(() => jest.clearAllMocks());

  it('installs the recording script through context.addInitScript', async () => {
    const context = contextFixture();
    const initializer = createRecordingContextInitializer({ logger: mockLogger });

    await initializer.initialize(context);

    expect(context.addInitScript).toHaveBeenCalledTimes(1);
    expect(context.route).not.toHaveBeenCalled();
    expect(initializer.getInjectionStrategyName()).toBe('init-script');
  });

  it('initializes once when called repeatedly or concurrently', async () => {
    const context = contextFixture();
    const initializer = createRecordingContextInitializer({ logger: mockLogger });

    await Promise.all([initializer.initialize(context), initializer.initialize(context)]);
    await initializer.initialize(context);

    expect(context.addInitScript).toHaveBeenCalledTimes(1);
  });

  it('allows initialization to retry after init-script registration fails', async () => {
    const addInitScript = jest
      .fn()
      .mockRejectedValueOnce(new Error('temporary init-script setup failure'))
      .mockResolvedValueOnce(undefined);
    const context = contextFixture({ addInitScript });
    const initializer = createRecordingContextInitializer({ logger: mockLogger });

    await expect(initializer.initialize(context)).rejects.toThrow('temporary init-script setup failure');
    await initializer.initialize(context);

    expect(addInitScript).toHaveBeenCalledTimes(2);
    expect(initializer.isInitialized()).toBe(true);
  });

  it('registers event routes for existing and newly created pages', async () => {
    const handlers = new Map<string, (route: Route) => Promise<void>>();
    const page = createMockPage({
      route: jest.fn().mockImplementation((pattern: string, handler: (route: Route) => Promise<void>) => {
        handlers.set(pattern, handler);
        return Promise.resolve();
      }),
    });
    const context = contextFixture({ pages: jest.fn().mockReturnValue([page]) });
    const initializer = createRecordingContextInitializer({ logger: mockLogger });

    await initializer.initialize(context);

    expect(page.route).toHaveBeenCalledTimes(1);
    expect(initializer.hasEventRoute(page)).toBe(true);
    const pageListener = (context.on as jest.Mock).mock.calls.find(([event]) => event === 'page')?.[1];
    expect(pageListener).toEqual(expect.any(Function));
  });

  it('retains event handler lifecycle and diagnostics accessors', async () => {
    const context = contextFixture();
    const initializer = createRecordingContextInitializer({ logger: mockLogger });
    const handler = jest.fn<(event: RawBrowserEvent) => void>();

    await initializer.initialize(context);
    initializer.setEventHandler(handler);
    expect(initializer.hasEventHandler()).toBe(true);
    expect(initializer.getInjectionStats()).toMatchObject({ attempted: 0, successful: 0, failed: 0 });

    initializer.resetStats();
    initializer.clearEventHandler();
    expect(initializer.hasEventHandler()).toBe(false);
    expect(initializer.getInjectionStrategyName()).toBe('init-script');
  });
});
