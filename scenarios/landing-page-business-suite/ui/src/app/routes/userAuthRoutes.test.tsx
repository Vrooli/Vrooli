import { MemoryRouter, Routes } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from "@vrooli/api-base/testing";
import { AdminAuthProvider } from '../providers/AdminAuthProvider';
import { UserAuthProvider } from '../providers/UserAuthProvider';
import { UserAuthContext } from '../providers/UserAuthContext';
import { userAuthRoutes } from './userAuthRoutes';

const { mockCheckAdminSession, mockListAccountSessions, mockListPasskeys } = vi.hoisted(() => ({
  mockCheckAdminSession: vi.fn(),
  mockListAccountSessions: vi.fn(),
  mockListPasskeys: vi.fn(),
}));

vi.mock('../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../shared/api')>('../../shared/api');
  return { ...actual, checkAdminSession: mockCheckAdminSession, listAccountSessions: mockListAccountSessions, listPasskeys: mockListPasskeys };
});

function renderRoute(route: string) {
  const routes = <MemoryRouter initialEntries={[route]}><Routes>{userAuthRoutes}</Routes></MemoryRouter>;
  if (route.startsWith('/admin/')) {
    window.history.replaceState({}, '', route);
    mockCheckAdminSession.mockResolvedValue({ authenticated: false, reset_enabled: false });
    return renderWithProviders(<AdminAuthProvider>{routes}</AdminAuthProvider>, { route });
  }
  return renderWithProviders(<UserAuthProvider>{routes}</UserAuthProvider>, { route });
}

afterEach(() => {
  window.history.replaceState({}, '', '/');
  vi.clearAllMocks();
});

describe('userAuthRoutes', () => {
  it('loads the customer login route through the route error and profiler boundary', async () => {
    renderRoute('/auth/login');
    expect(await screen.findByTestId('email-input')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(/^Sign in to /);
  });

  it('loads the administrator login route', async () => {
    renderRoute('/admin/login');
    expect(await screen.findByRole('heading', { name: 'Admin sign-in' })).toBeInTheDocument();
  });

  it('loads the verification route and preserves its missing-token validation', async () => {
    renderRoute('/auth/verify');
    expect(await screen.findByRole('heading', { name: 'This link doesn’t work' })).toBeInTheDocument();
    expect(screen.getByText(/missing its sign-in token/)).toBeInTheDocument();
  });

  it('drops cached private sessions when authoritative account identity changes in the mounted route', async () => {
    mockListAccountSessions.mockResolvedValueOnce({ sessions: [{ id: 'old-session', deviceLabel: 'Old buyer device', current: true }] })
      .mockResolvedValueOnce({ sessions: [{ id: 'new-session', deviceLabel: 'New buyer device', current: true }] });
    mockListPasskeys.mockResolvedValue({ passkeys: [] });
    function account(id: string) {
      return <MemoryRouter initialEntries={['/account/security']}><UserAuthContext.Provider value={{
        isAuthenticated: true, isSessionLoading: false, user: { id, email: `${id}@buyer-entry.invalid`, email_verified: true },
        logout: () => Promise.resolve(), refreshSession: () => Promise.resolve(),
      }}><Routes>{userAuthRoutes}</Routes></UserAuthContext.Provider></MemoryRouter>;
    }
    const view = renderWithProviders(account('old-buyer'));
    expect(await screen.findByText('Old buyer device · This device')).toBeInTheDocument();
    view.rerender(account('new-buyer'));
    expect(screen.queryByText('Old buyer device · This device')).not.toBeInTheDocument();
    expect(await screen.findByText('New buyer device · This device')).toBeInTheDocument();
    expect(mockListAccountSessions).toHaveBeenCalledTimes(2);
  });
});
