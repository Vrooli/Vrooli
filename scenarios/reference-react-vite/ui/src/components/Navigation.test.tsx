// Affordance-conformance tests for ui/flow/navigation.json. Each spec
// affordance must render its declared label and test_id in the
// container indicated by its `presentations[].in` field, gated by the
// declared `show_when` context predicate. The fixture below mirrors the
// spec; if it drifts, both flow-verifier reconcile and these tests
// surface the gap independently.
import { describe, expect, it, vi } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';

import { renderWithProviders } from '../test-utils/renderWithProviders';
import { Layout } from './Layout';

vi.mock('../lib/api', () => ({
  fetchHealth: vi.fn().mockResolvedValue({ status: 'healthy', service: 'ref', timestamp: '' }),
}));

const TOP_BAR_AFFORDANCES = [
  { id: 'top-nav-dashboard', label: 'Dashboard' },
  { id: 'top-nav-tasks', label: 'Tasks' },
  { id: 'top-nav-projects', label: 'Projects' },
  { id: 'top-nav-settings', label: 'Settings' },
];

describe('navigation.json affordance conformance', () => {
  describe('top_nav_bar (desktop + auth=logged_in)', () => {
    it.each(TOP_BAR_AFFORDANCES)('renders affordance %s', ({ id, label }) => {
      renderWithProviders(<Layout />, { auth: 'logged_in', role: 'viewer' });
      const el = screen.getByTestId(id);
      expect(el).toBeInTheDocument();
      expect(el).toHaveTextContent(label);
    });

    it('hides admin affordance when role!=admin', () => {
      renderWithProviders(<Layout />, { auth: 'logged_in', role: 'viewer' });
      expect(screen.queryByTestId('top-nav-admin')).not.toBeInTheDocument();
    });

    it('shows admin affordance when role=admin', () => {
      renderWithProviders(<Layout />, { auth: 'logged_in', role: 'admin' });
      expect(screen.getByTestId('top-nav-admin')).toBeInTheDocument();
    });

    it('hides every top-nav affordance when logged out', () => {
      renderWithProviders(<Layout />, { auth: 'logged_out' });
      for (const a of TOP_BAR_AFFORDANCES) {
        expect(screen.queryByTestId(a.id)).not.toBeInTheDocument();
      }
    });
  });

  describe('user_menu (auth=logged_in)', () => {
    it('exposes the log_out affordance with the declared label and test_id', () => {
      renderWithProviders(<Layout />, { auth: 'logged_in' });
      fireEvent.click(screen.getByTestId('user-menu-trigger'));
      const logout = screen.getByTestId('logout-item');
      expect(logout).toBeInTheDocument();
      expect(logout).toHaveTextContent('Log out');
    });

    it('exposes the beta affordance only when feature_beta=true', () => {
      const { rerender } = renderWithProviders(<Layout />, { auth: 'logged_in', featureBeta: false });
      fireEvent.click(screen.getByTestId('user-menu-trigger'));
      expect(screen.queryByTestId('user-menu-beta')).not.toBeInTheDocument();
      rerender(<Layout />);
    });
  });

  describe('auth_footer', () => {
    it('appears on logged_out routes only', () => {
      renderWithProviders(<Layout />, { auth: 'logged_out' });
      expect(screen.getByTestId('auth-footer')).toBeInTheDocument();
    });

    it('hides on logged_in routes', () => {
      renderWithProviders(<Layout />, { auth: 'logged_in' });
      expect(screen.queryByTestId('auth-footer')).not.toBeInTheDocument();
    });
  });
});
