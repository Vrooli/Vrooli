import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  createRemoteProfileAdmin,
  getRemoteProfileSessionLinksAdmin,
  listIncomingRemoteProfileSessionsAdmin,
  revokeIncomingRemoteProfileSessionAdmin,
} from './remoteProfiles';
import { createFetchMock, getFetchCall, installFetchMock, mockResponses } from '../test-utils/api-mocks';

describe('remoteProfiles API', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('encodes connector_id when listing incoming sessions', async () => {
    fetchMock.mockResolvedValue(mockResponses.success({ sessions: [] }));

    await listIncomingRemoteProfileSessionsAdmin('connector with spaces/slash');

    const [url] = getFetchCall(fetchMock);
    expect(url).toContain('/admin/remote-profile-sessions?connector_id=');
    expect(url).toContain(encodeURIComponent('connector with spaces/slash'));
  });

  it('throws when create response schema is invalid', async () => {
    fetchMock.mockResolvedValue(mockResponses.success({ id: 10 }));

    await expect(createRemoteProfileAdmin({
      tag: 'prod',
      api_base: 'https://example.com/api/v1',
    })).rejects.toThrow('Invalid remote profile response from API');
  });

  it('throws when session-links response schema is invalid', async () => {
    fetchMock.mockResolvedValue(mockResponses.success({ profile_id: 7 }));

    await expect(getRemoteProfileSessionLinksAdmin(7)).rejects.toThrow(
      'Invalid remote profile session links response from API'
    );
  });

  it('returns fallback empty session list when payload is invalid', async () => {
    fetchMock.mockResolvedValue(mockResponses.success({ hello: 'world' }));

    const result = await listIncomingRemoteProfileSessionsAdmin();

    expect(result).toEqual({ sessions: [] });
  });

  it('sends delete request for incoming remote session revoke', async () => {
    fetchMock.mockResolvedValue(mockResponses.success({ success: true }));

    await revokeIncomingRemoteProfileSessionAdmin('remote-session-42');

    const [url, options] = getFetchCall(fetchMock);
    expect(url).toContain('/admin/remote-profile-sessions/remote-session-42');
    expect(options.method).toBe('DELETE');
  });
});
