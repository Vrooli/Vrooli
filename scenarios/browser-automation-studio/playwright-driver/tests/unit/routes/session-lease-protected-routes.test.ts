import {
  createMockHttpRequest,
  createMockHttpResponse,
} from '../../helpers';
import type { SessionState } from '../../../src/types/session';
import { handleSessionStorageState } from '../../../src/routes/session-storage';
import {
  handleSessionServiceWorkers,
  handleSessionServiceWorkersDelete,
  handleSessionServiceWorkerDelete,
} from '../../../src/routes/session-service-workers';

const ownerHeaders = {
  'x-bas-execution-id': 'execution-owner',
  'x-bas-session-lease-id': 'lease-current',
};

describe('lease-protected session routes', () => {
  it('exports storage state only for the exact current execution lease', async () => {
    const getStorageStateForLease = jest.fn().mockResolvedValue({ cookies: [{ name: 'auth' }] });
    const session = {} as SessionState;
    const manager = {
      getSessionForLease: jest.fn((sessionId: string, executionId: string, leaseId: string) => {
        if (sessionId !== 'session-1' || executionId !== 'execution-owner' || leaseId !== 'lease-current') {
          throw new Error('session not found');
        }
        return session;
      }),
      getStorageStateForLease,
    } as never;

    const allowed = createMockHttpResponse();
    await handleSessionStorageState(createMockHttpRequest({ headers: ownerHeaders }), allowed, 'session-1', manager);
    expect(allowed.statusCode).toBe(200);
    expect(getStorageStateForLease).toHaveBeenCalledWith('session-1', 'execution-owner', 'lease-current');

    const denied = createMockHttpResponse();
    await handleSessionStorageState(
      createMockHttpRequest({ headers: { ...ownerHeaders, 'x-bas-session-lease-id': 'stale-lease' } }),
      denied,
      'session-1',
      manager
    );
    expect(denied.statusCode).toBe(404);
    expect(getStorageStateForLease).toHaveBeenCalledTimes(1);

    const missing = createMockHttpResponse();
    await handleSessionStorageState(createMockHttpRequest(), missing, 'session-1', manager);
    expect(missing.statusCode).toBe(404);
    expect(getStorageStateForLease).toHaveBeenCalledTimes(1);
  });

  it('denies service-worker inspection and mutation from another lease', async () => {
    const unregisterAll = jest.fn().mockResolvedValue(1);
    const session = {
      serviceWorkerController: {
        getWorkers: jest.fn().mockReturnValue([]),
        getControl: jest.fn().mockReturnValue({ mode: 'allow' }),
        unregisterAll,
      },
    } as unknown as SessionState;
    const manager = {
      getSessionForLease: jest.fn((sessionId: string, executionId: string, leaseId: string) => {
        if (sessionId !== 'session-2' || executionId !== 'execution-owner' || leaseId !== 'lease-current') {
          throw new Error('session not found');
        }
        return session;
      }),
    } as never;

    const deniedRead = createMockHttpResponse();
    await handleSessionServiceWorkers(
      createMockHttpRequest({ headers: { ...ownerHeaders, 'x-bas-execution-id': 'foreign-owner' } }),
      deniedRead,
      'session-2',
      manager
    );
    expect(deniedRead.statusCode).toBe(404);

    const deniedDelete = createMockHttpResponse();
    await handleSessionServiceWorkersDelete(
      createMockHttpRequest({ method: 'DELETE', headers: { ...ownerHeaders, 'x-bas-session-lease-id': 'stale-lease' } }),
      deniedDelete,
      'session-2',
      manager
    );
    expect(deniedDelete.statusCode).toBe(404);
    expect(unregisterAll).not.toHaveBeenCalled();

    const allowedDelete = createMockHttpResponse();
    await handleSessionServiceWorkersDelete(
      createMockHttpRequest({ method: 'DELETE', headers: ownerHeaders }),
      allowedDelete,
      'session-2',
      manager
    );
    expect(allowedDelete.statusCode).toBe(200);
    expect(unregisterAll).toHaveBeenCalledTimes(1);
  });

  it('checks every service-worker scope before bulk unregister and permits an in-scope worker', async () => {
    const unregisterAll = jest.fn().mockResolvedValue(1);
    const getWorkers = jest.fn().mockReturnValue([
      { registrationId: 'registration-1', scopeURL: 'https://outside.example/app/', scriptURL: 'https://outside.example/sw.js', status: 'running' },
    ]);
    const session = {
      page: { url: () => 'https://allowed.example/app/' },
      createdAt: new Date(0),
      instructionCount: 0,
      spec: { permission_profile: { effect: 'act', approval: 'none', view: true, control: true, url_allow: ['https://allowed.example/*'] } },
      serviceWorkerController: { getWorkers, getControl: jest.fn().mockReturnValue({ mode: 'allow' }), unregisterAll },
    } as unknown as SessionState;
    const manager = { getSessionForLease: jest.fn(() => session) } as never;

    const denied = createMockHttpResponse();
    await handleSessionServiceWorkersDelete(
      createMockHttpRequest({ method: 'DELETE', headers: ownerHeaders }), denied, 'session-2', manager
    );
    expect(denied.statusCode).toBe(403);
    expect(unregisterAll).not.toHaveBeenCalled();

    getWorkers.mockReturnValue([
      { registrationId: 'registration-1', scopeURL: 'https://allowed.example/app/', scriptURL: 'https://allowed.example/sw.js', status: 'running' },
    ]);
    const allowed = createMockHttpResponse();
    await handleSessionServiceWorkersDelete(
      createMockHttpRequest({ method: 'DELETE', headers: ownerHeaders }), allowed, 'session-2', manager
    );
    expect(allowed.statusCode).toBe(200);
    expect(unregisterAll).toHaveBeenCalledTimes(1);
  });

  it('denies single service-worker deletion outside URL scope before controller effect', async () => {
    const unregister = jest.fn().mockResolvedValue(true);
    const session = {
      page: { url: () => 'https://allowed.example/app/' },
      createdAt: new Date(0),
      instructionCount: 0,
      spec: { permission_profile: { effect: 'act', approval: 'none', view: true, control: true, url_allow: ['https://allowed.example/*'] } },
      serviceWorkerController: {
        getWorkers: jest.fn(), getControl: jest.fn(), unregister,
      },
    } as unknown as SessionState;
    const manager = { getSessionForLease: jest.fn(() => session) } as never;

    const denied = createMockHttpResponse();
    await handleSessionServiceWorkerDelete(
      createMockHttpRequest({ method: 'DELETE', headers: ownerHeaders }),
      denied,
      'session-2',
      encodeURIComponent('https://outside.example/app/'),
      manager
    );
    expect(denied.statusCode).toBe(403);
    expect(unregister).not.toHaveBeenCalled();

    const allowed = createMockHttpResponse();
    await handleSessionServiceWorkerDelete(
      createMockHttpRequest({ method: 'DELETE', headers: ownerHeaders }),
      allowed,
      'session-2',
      encodeURIComponent('https://allowed.example/app/'),
      manager
    );
    expect(allowed.statusCode).toBe(200);
    expect(unregister).toHaveBeenCalledTimes(1);
    expect(unregister).toHaveBeenCalledWith('https://allowed.example/app/');
  });

  it('fails closed when the profile requires approval that the driver route cannot receive', async () => {
    const unregister = jest.fn().mockResolvedValue(true);
    const session = {
      page: { url: () => 'https://allowed.example/app/' },
      createdAt: new Date(0),
      instructionCount: 0,
      spec: { permission_profile: { effect: 'act', approval: 'every_action', view: true, control: true, url_allow: ['https://allowed.example/*'] } },
      serviceWorkerController: { getWorkers: jest.fn(), getControl: jest.fn(), unregister },
    } as unknown as SessionState;
    const manager = { getSessionForLease: jest.fn(() => session) } as never;
    const denied = createMockHttpResponse();

    await handleSessionServiceWorkerDelete(
      createMockHttpRequest({ method: 'DELETE', headers: ownerHeaders }),
      denied,
      'session-2',
      encodeURIComponent('https://allowed.example/app/'),
      manager
    );
    expect(denied.statusCode).toBe(403);
    expect(unregister).not.toHaveBeenCalled();
  });
});
