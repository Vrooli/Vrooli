// RouteGate enforces deep_link_policy from ui/flow/navigation.json:
// auth-required routes redirect to /login preserving the target,
// admin/beta routes bounce to /dashboard when their gating predicate
// is not met, and auth pages redirect to /dashboard when already
// logged in.
import { describe, expect, it } from 'vitest';
import { Route, Routes } from 'react-router-dom';
import { screen } from '@testing-library/react';

import { renderWithProviders } from '../test-utils/renderWithProviders';
import { RouteGate } from './RouteGate';

function Probe({ tag }: { tag: string }) {
  return <div data-testid={`probe-${tag}`}>{tag}</div>;
}

function Harness() {
  return (
    <Routes>
      <Route
        path="/login"
        element={
          <RouteGate redirectLoggedIn>
            <Probe tag="login" />
          </RouteGate>
        }
      />
      <Route path="/" element={<Probe tag="dashboard" />} />
      <Route
        path="/admin/users"
        element={
          <RouteGate requireAuth requireRole="admin">
            <Probe tag="admin" />
          </RouteGate>
        }
      />
      <Route
        path="/beta"
        element={
          <RouteGate requireAuth requireBeta>
            <Probe tag="beta" />
          </RouteGate>
        }
      />
      <Route
        path="/tasks"
        element={
          <RouteGate requireAuth>
            <Probe tag="tasks" />
          </RouteGate>
        }
      />
    </Routes>
  );
}

describe('RouteGate', () => {
  it('redirects /tasks to /login when logged out (deep_link_policy auth_required_routes_redirect_to_login)', () => {
    renderWithProviders(<Harness />, { initialEntries: ['/tasks'], auth: 'logged_out' });
    expect(screen.queryByTestId('probe-tasks')).not.toBeInTheDocument();
  });

  it('redirects /login to / when already logged in (auth_pages_redirect_when_already_logged_in)', () => {
    renderWithProviders(<Harness />, { initialEntries: ['/login'], auth: 'logged_in' });
    expect(screen.getByTestId('probe-dashboard')).toBeInTheDocument();
    expect(screen.queryByTestId('probe-login')).not.toBeInTheDocument();
  });

  it('redirects /admin/users to / for non-admins (admin_routes_redirect_non_admins)', () => {
    renderWithProviders(<Harness />, { initialEntries: ['/admin/users'], auth: 'logged_in', role: 'viewer' });
    expect(screen.getByTestId('probe-dashboard')).toBeInTheDocument();
    expect(screen.queryByTestId('probe-admin')).not.toBeInTheDocument();
  });

  it('renders /admin/users for admins', () => {
    renderWithProviders(<Harness />, { initialEntries: ['/admin/users'], auth: 'logged_in', role: 'admin' });
    expect(screen.getByTestId('probe-admin')).toBeInTheDocument();
  });

  it('redirects /beta to / when feature_beta=false (beta_routes_redirect_when_flag_off)', () => {
    renderWithProviders(<Harness />, { initialEntries: ['/beta'], auth: 'logged_in', featureBeta: false });
    expect(screen.getByTestId('probe-dashboard')).toBeInTheDocument();
    expect(screen.queryByTestId('probe-beta')).not.toBeInTheDocument();
  });

  it('renders /beta when feature_beta=true', () => {
    renderWithProviders(<Harness />, { initialEntries: ['/beta'], auth: 'logged_in', featureBeta: true });
    expect(screen.getByTestId('probe-beta')).toBeInTheDocument();
  });
});
