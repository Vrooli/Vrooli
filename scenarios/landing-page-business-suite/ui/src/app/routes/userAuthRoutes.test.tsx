import { Routes } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '../../test-utils/renderWithProviders';
import { AdminAuthProvider } from '../providers/AdminAuthProvider';
import { userAuthRoutes } from './userAuthRoutes';

const { mockCheckAdminSession } = vi.hoisted(() => ({
  mockCheckAdminSession: vi.fn(),
}));

vi.mock('../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../shared/api')>('../../shared/api');
  return { ...actual, checkAdminSession: mockCheckAdminSession };
});

function renderRoute(route: string) {
  const routes = <Routes>{userAuthRoutes}</Routes>;
  if (route.startsWith('/admin/')) {
    window.history.replaceState({}, '', route);
    mockCheckAdminSession.mockResolvedValue({ authenticated: false, reset_enabled: false });
    return renderWithProviders(<AdminAuthProvider>{routes}</AdminAuthProvider>, { route });
  }
  return renderWithProviders(routes, { route });
}

afterEach(() => {
  window.history.replaceState({}, '', '/');
  vi.clearAllMocks();
});

describe('userAuthRoutes', () => {
  it('loads the customer login route through the route error and profiler boundary', async () => {
    renderRoute('/auth/login');
    expect(await screen.findByRole('heading', { name: 'Sign In' })).toBeInTheDocument();
  });

  it('loads the administrator login route', async () => {
    renderRoute('/admin/login');
    expect(await screen.findByRole('heading', { name: 'Admin Portal' })).toBeInTheDocument();
  });

  it('loads the verification route and preserves its missing-token validation', async () => {
    renderRoute('/auth/verify');
    expect(await screen.findByText('No verification token provided')).toBeInTheDocument();
  });
});
