import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  formatCredits,
  formatSubscriptionStatus,
  getUserDetails,
  getUserSessions,
  listUsers,
  parseUserAgent,
  revokeAllSessions,
  revokeSession,
} from './users.service';
import * as api from '../../../shared/api/common';

vi.mock('../../../shared/api/common', () => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  apiDelete: vi.fn(),
}));

const apiGet = vi.mocked(api.apiGet);
const apiPost = vi.mocked(api.apiPost);
const apiDelete = vi.mocked(api.apiDelete);

describe('users service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('encodes list filters without emitting empty query parameters', async () => {
    apiGet.mockResolvedValue({ users: [], total: 0, page: 1, per_page: 20, total_pages: 0 });

    await listUsers();
    await listUsers({ search: 'billing+ops@example.com', page: 2, per_page: 50 });

    expect(apiGet).toHaveBeenNthCalledWith(1, '/admin/users');
    expect(apiGet).toHaveBeenNthCalledWith(2, '/admin/users?search=billing%2Bops%40example.com&page=2&per_page=50');
  });

  it('uses scoped customer endpoints for details and session operations', async () => {
    apiGet.mockResolvedValueOnce({ id: 'user_123' }).mockResolvedValueOnce([]);
    apiDelete.mockResolvedValue({ success: true, message: 'revoked' });
    apiPost.mockResolvedValue({ success: true, message: 'revoked all', sessions_revoked: 2 });

    await getUserDetails('user_123');
    await getUserSessions('user_123');
    await revokeSession('user_123', 'session_456');
    await revokeAllSessions('user_123');

    expect(apiGet).toHaveBeenNthCalledWith(1, '/admin/users/user_123');
    expect(apiGet).toHaveBeenNthCalledWith(2, '/admin/users/user_123/sessions');
    expect(apiDelete).toHaveBeenCalledWith('/admin/users/user_123/sessions/session_456');
    expect(apiPost).toHaveBeenCalledWith('/admin/users/user_123/sessions/revoke-all');
  });

  it('formats subscription and credit values safely for customer-facing administration', () => {
    expect(formatSubscriptionStatus()).toBe('No subscription');
    expect(formatSubscriptionStatus({ plan_tier: 'pro', status: 'active' })).toBe('Pro');
    expect(formatSubscriptionStatus({ plan_tier: '', status: 'past_due' })).toBe('Unknown (past_due)');
    expect(formatCredits()).toBe('No credits');
    expect(formatCredits({ balance: 1_000, bonus: 25 })).toBe('1,025');
  });

  it.each([
    ['Mozilla/5.0 Chrome Windows', 'Chrome on Windows'],
    ['Mozilla/5.0 Chrome Mac', 'Chrome on Mac'],
    ['Mozilla/5.0 Firefox Windows', 'Firefox on Windows'],
    ['Mozilla/5.0 Firefox Mac', 'Firefox on Mac'],
    ['Mozilla/5.0 Safari', 'Safari on Mac'],
    ['Mozilla/5.0 Edge', 'Edge on Windows'],
    ['Mozilla/5.0 Mobile', 'Mobile Browser'],
    ['unknown', 'Unknown browser'],
    [undefined, 'Unknown device'],
  ])('renders a privacy-minimizing device label for %s', (agent, expected) => {
    expect(parseUserAgent(agent)).toBe(expected);
  });
});
