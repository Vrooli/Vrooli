import { describe, expect, it, vi } from 'vitest';
import {
  fetchIncomingRemoteProfileSessions,
  getRemoteProfileSessionLinks,
  normalizeRemoteProfileForm,
  validateRemoteProfileForm,
  validateRemoteProfileLoginForm,
} from './remoteProfiles.service';
import type {
  getRemoteProfileSessionLinksAdmin,
  listIncomingRemoteProfileSessionsAdmin,
} from '../../../shared/api';

type GetSessionLinksFn = typeof getRemoteProfileSessionLinksAdmin;
type ListIncomingFn = typeof listIncomingRemoteProfileSessionsAdmin;

const getRemoteProfileSessionLinksAdminMock = vi.fn<Parameters<GetSessionLinksFn>, ReturnType<GetSessionLinksFn>>();
const listIncomingRemoteProfileSessionsAdminMock = vi.fn<Parameters<ListIncomingFn>, ReturnType<ListIncomingFn>>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getRemoteProfileSessionLinksAdmin: (...args: Parameters<GetSessionLinksFn>) =>
      getRemoteProfileSessionLinksAdminMock(...args),
    listIncomingRemoteProfileSessionsAdmin: (...args: Parameters<ListIncomingFn>) =>
      listIncomingRemoteProfileSessionsAdminMock(...args),
  };
});

describe('remoteProfiles.service', () => {
  it('normalizes form by trimming values', () => {
    const payload = normalizeRemoteProfileForm({
      tag: '  prod  ',
      label: '  Production  ',
      apiBase: '  https://example.com/api/v1  ',
    });

    expect(payload).toEqual({
      tag: 'prod',
      label: 'Production',
      api_base: 'https://example.com/api/v1',
    });
  });

  it('validates required remote profile form fields', () => {
    expect(validateRemoteProfileForm({ tag: '', label: '', apiBase: '' })).toBe('Tag is required');
    expect(validateRemoteProfileForm({ tag: 'prod', label: '', apiBase: '' })).toBe('API base is required');
    expect(validateRemoteProfileForm({ tag: 'prod', label: '', apiBase: 'https://example.com' })).toBe(
      'API base must include /api/v1'
    );
    expect(validateRemoteProfileForm({ tag: 'prod', label: '', apiBase: 'https://example.com/api/v1' })).toBeNull();
  });

  it('validates remote login form fields', () => {
    expect(validateRemoteProfileLoginForm({ email: '', password: 'x' })).toBe('Email is required');
    expect(validateRemoteProfileLoginForm({ email: 'admin@example.com', password: '' })).toBe('Password is required');
    expect(validateRemoteProfileLoginForm({ email: 'admin@example.com', password: 'secret' })).toBeNull();
  });

  it('normalizes undefined remote sessions to empty list', async () => {
    getRemoteProfileSessionLinksAdminMock.mockResolvedValue({
      profile_id: 7,
      profile_tag: 'prod',
      connector_id: 'connector-1',
      local_has_session: true,
      local_status: 'active',
      remote_sessions: undefined,
    });

    const result = await getRemoteProfileSessionLinks(7);

    expect(result.remote_sessions).toEqual([]);
  });

  it('returns incoming sessions from API response', async () => {
    listIncomingRemoteProfileSessionsAdminMock.mockResolvedValue({
      sessions: [
        {
          session_id: 'remote-session-1',
          admin_email: 'admin@example.com',
          connector_id: 'connector-1',
          created_at: '2025-01-01T00:00:00Z',
          last_activity: '2025-01-01T01:00:00Z',
          expires_at: '2025-01-01T02:00:00Z',
        },
      ],
    });

    const sessions = await fetchIncomingRemoteProfileSessions();

    expect(sessions).toHaveLength(1);
    expect(sessions[0]?.session_id).toBe('remote-session-1');
  });
});
