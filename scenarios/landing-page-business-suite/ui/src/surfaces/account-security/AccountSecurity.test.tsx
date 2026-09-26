import { cleanup, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@vrooli/api-base/testing';
import { expectNoA11yViolations } from '@vrooli/api-base/testing';
import { MemoryRouter } from 'react-router-dom';
import { AccountSecurity } from './AccountSecurity';

const mocks = vi.hoisted(() => ({
  list: vi.fn(), listPasskeys: vi.fn(), revoke: vi.fn(), revokeOthers: vi.fn(), start: vi.fn(), complete: vi.fn(), renamePasskey: vi.fn(), revokePasskey: vi.fn(),
}));
vi.mock('../../shared/api', () => ({
  listAccountSessions: mocks.list,
  listPasskeys: mocks.listPasskeys,
  revokeAccountSession: mocks.revoke,
  revokeOtherAccountSessions: mocks.revokeOthers,
  startAccountReauthentication: mocks.start,
  completeAccountReauthentication: mocks.complete,
  renamePasskey: mocks.renamePasskey,
  revokePasskey: mocks.revokePasskey,
}));
vi.mock('../../app/providers/useUserAuth', () => ({ useUserAuth: () => ({ isAuthenticated: true, isSessionLoading: false }) }));

describe('AccountSecurity', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.list.mockResolvedValue({ sessions: [
      { id: 'current', deviceLabel: 'Chrome on macOS', ipHint: '203.0.113.x', authMethod: 'email_code', current: true },
      { id: 'other', deviceLabel: 'Safari on iOS', ipHint: '198.51.100.x', authMethod: 'email_code', current: false },
    ] });
    mocks.revoke.mockResolvedValue({ revoked: true });
    mocks.revokeOthers.mockResolvedValue({ revokedCount: 1 });
    mocks.start.mockResolvedValue({});
    mocks.listPasskeys.mockResolvedValue({ passkeys: [] });
  });
  afterEach(() => cleanup());

  it('lists sessions and revokes another device', async () => {
    renderWithProviders(<MemoryRouter><AccountSecurity /></MemoryRouter>);
    expect(await screen.findByText('Chrome on macOS · This device')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Revoke' }));
    await waitFor(() => expect(mocks.revoke).toHaveBeenCalledWith('other'));
    await waitFor(() => expect(screen.queryByText('Safari on iOS')).not.toBeInTheDocument());
  });

  it('signs out elsewhere without offering to revoke the current session', async () => {
    renderWithProviders(<MemoryRouter><AccountSecurity /></MemoryRouter>);
    await screen.findByText('Safari on iOS');
    await userEvent.click(screen.getByRole('button', { name: 'Sign out elsewhere' }));
    await waitFor(() => expect(mocks.revokeOthers).toHaveBeenCalledOnce());
    expect(screen.getByText('Chrome on macOS · This device')).toBeInTheDocument();
  });

  it('has no detectable accessibility violations', async () => {
    const { container } = renderWithProviders(<MemoryRouter><AccountSecurity /></MemoryRouter>);
    await screen.findByText('Safari on iOS');
    await expectNoA11yViolations(container);
  });
});
