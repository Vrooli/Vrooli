import {
  createMockHttpRequest,
  createMockHttpResponse,
} from '../../helpers';
import type { SessionState } from '../../../src/types/session';
import { handleSessionStorageState } from '../../../src/routes/session-storage';
import {
  handleSessionServiceWorkers,
  handleSessionServiceWorkersDelete,
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
});
