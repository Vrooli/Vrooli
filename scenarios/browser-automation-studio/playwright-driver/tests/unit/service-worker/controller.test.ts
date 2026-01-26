import { ServiceWorkerController } from '../../../src/service-worker/controller';
import type { BrowserContext, Page, CDPSession } from 'rebrowser-playwright';
import { getCachedCDPSession } from '../../../src/session/cdp-session';

jest.mock('../../../src/session/cdp-session', () => ({
  getCachedCDPSession: jest.fn(),
}));

const mockGetCachedCDPSession = getCachedCDPSession as jest.MockedFunction<typeof getCachedCDPSession>;

describe('ServiceWorkerController', () => {
  const buildCdpSession = () => {
    const handlers = new Map<string, (payload: unknown) => void>();
    const session = {
      send: jest.fn().mockResolvedValue(undefined),
      on: jest.fn((event: string, handler: (payload: unknown) => void) => {
        handlers.set(event, handler);
      }),
    };
    return { session: session as unknown as CDPSession, handlers };
  };

  const buildPage = (): Page =>
    ({
      context: jest.fn(),
    }) as unknown as Page;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('enables monitoring and registers CDP listeners', async () => {
    const { session, handlers } = buildCdpSession();
    mockGetCachedCDPSession.mockResolvedValue(session);

    const controller = new ServiceWorkerController('session-1', { mode: 'allow' });
    await controller.enable(buildPage());

    expect(session.send).toHaveBeenCalledWith('ServiceWorker.enable');
    expect(handlers.has('ServiceWorker.workerRegistrationUpdated')).toBe(true);
    expect(handlers.has('ServiceWorker.workerVersionUpdated')).toBe(true);
    expect(handlers.has('ServiceWorker.workerErrorReported')).toBe(true);
  });

  it('tracks registrations and versions for getWorkers', async () => {
    const { session, handlers } = buildCdpSession();
    mockGetCachedCDPSession.mockResolvedValue(session);

    const controller = new ServiceWorkerController('session-1', { mode: 'allow' });
    await controller.enable(buildPage());

    handlers.get('ServiceWorker.workerRegistrationUpdated')?.({
      registrations: [
        { registrationId: 'reg-1', scopeURL: 'https://example.com/', isDeleted: false },
      ],
    });
    handlers.get('ServiceWorker.workerVersionUpdated')?.({
      versions: [
        { versionId: 'v1', registrationId: 'reg-1', scriptURL: '/sw.js', runningStatus: 'running', status: 'activated' },
      ],
    });

    const workers = controller.getWorkers();
    expect(workers).toHaveLength(1);
    expect(workers[0]?.scopeURL).toBe('https://example.com/');
    expect(workers[0]?.status).toBe('running');
    expect(workers[0]?.scriptURL).toBe('/sw.js');
  });

  it('unregisters all non-deleted registrations', async () => {
    const { session, handlers } = buildCdpSession();
    mockGetCachedCDPSession.mockResolvedValue(session);

    const controller = new ServiceWorkerController('session-1', { mode: 'allow' });
    await controller.enable(buildPage());

    handlers.get('ServiceWorker.workerRegistrationUpdated')?.({
      registrations: [
        { registrationId: 'reg-1', scopeURL: 'https://example.com/', isDeleted: false },
        { registrationId: 'reg-2', scopeURL: 'https://delete.me/', isDeleted: true },
      ],
    });

    const count = await controller.unregisterAll();
    expect(count).toBe(1);
    expect(session.send).toHaveBeenCalledWith('ServiceWorker.unregister', { scopeURL: 'https://example.com/' });
  });

  it('returns false when unregister fails', async () => {
    const { session } = buildCdpSession();
    session.send = jest.fn().mockRejectedValue(new Error('fail'));
    mockGetCachedCDPSession.mockResolvedValue(session);

    const controller = new ServiceWorkerController('session-1', { mode: 'allow' });
    await controller.enable(buildPage());

    const success = await controller.unregister('https://example.com/');
    expect(success).toBe(false);
  });

  it('shouldBlockDomain respects overrides and wildcard matching', () => {
    const controller = new ServiceWorkerController('session-1', {
      mode: 'block-on-domain',
      blockedDomains: ['*.example.com'],
      domainOverrides: [{ domain: 'example.com', mode: 'allow' }],
    });

    expect(controller.shouldBlockDomain('example.com')).toBe(false);
    expect(controller.shouldBlockDomain('sub.example.com')).toBe(true);
  });

  it('setupBlockingForContext skips when allow with no overrides', async () => {
    const controller = new ServiceWorkerController('session-1', { mode: 'allow' });
    const context = { addInitScript: jest.fn() } as unknown as BrowserContext;

    await controller.setupBlockingForContext(context);

    expect(context.addInitScript).not.toHaveBeenCalled();
  });

  it('setupBlockingForContext injects script when blocking', async () => {
    const controller = new ServiceWorkerController('session-1', { mode: 'block', blockedDomains: ['example.com'] });
    const context = { addInitScript: jest.fn().mockResolvedValue(undefined) } as unknown as BrowserContext;

    await controller.setupBlockingForContext(context);

    expect(context.addInitScript).toHaveBeenCalled();
  });
});
