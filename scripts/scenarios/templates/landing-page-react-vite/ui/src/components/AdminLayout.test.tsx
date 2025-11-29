import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AdminLayout } from './AdminLayout';
import * as api from '../lib/api';

vi.mock('../lib/api', () => ({
  adminLogout: vi.fn(),
}));

const renderWithRouter = (ui: React.ReactElement, { route = '/admin' } = {}) => {
  return render(
    <MemoryRouter initialEntries={[route]}>
      {ui}
    </MemoryRouter>
  );
};

describe('AdminLayout [REQ:ADMIN-NAV,ADMIN-BREADCRUMB]', () => {
  it('[REQ:ADMIN-NAV] should render navigation links', () => {
    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>);

    expect(screen.getByTestId('nav-home')).toBeInTheDocument();
    expect(screen.getByTestId('nav-analytics')).toBeInTheDocument();
    expect(screen.getByTestId('nav-customization')).toBeInTheDocument();
    expect(screen.getByTestId('nav-logout')).toBeInTheDocument();
  });

  it('[REQ:ADMIN-BREADCRUMB] should render breadcrumb for admin home', () => {
    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>, { route: '/admin' });

    const breadcrumb = screen.getByTestId('admin-breadcrumb');
    expect(breadcrumb).toBeInTheDocument();
    expect(screen.getByTestId('breadcrumb-0')).toHaveTextContent('Admin');
  });

  it('[REQ:ADMIN-BREADCRUMB] should render breadcrumb for analytics page', () => {
    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>, { route: '/admin/analytics' });

    expect(screen.getByTestId('breadcrumb-0')).toHaveTextContent('Admin');
    expect(screen.getByTestId('breadcrumb-1')).toHaveTextContent('Analytics');
  });

  it('[REQ:ADMIN-BREADCRUMB] should render breadcrumb for variant analytics', () => {
    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>, { route: '/admin/analytics/variant-a' });

    expect(screen.getByTestId('breadcrumb-0')).toHaveTextContent('Admin');
    expect(screen.getByTestId('breadcrumb-1')).toHaveTextContent('Analytics');
    expect(screen.getByTestId('breadcrumb-2')).toHaveTextContent('Variant variant-a');
  });

  it('[REQ:ADMIN-BREADCRUMB] should render breadcrumb for customization page', () => {
    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>, { route: '/admin/customization' });

    expect(screen.getByTestId('breadcrumb-0')).toHaveTextContent('Admin');
    expect(screen.getByTestId('breadcrumb-1')).toHaveTextContent('Customization');
  });

  it('[REQ:ADMIN-BREADCRUMB] should render breadcrumb for variant customization', () => {
    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>, { route: '/admin/customization/variants/variant-a' });

    expect(screen.getByTestId('breadcrumb-0')).toHaveTextContent('Admin');
    expect(screen.getByTestId('breadcrumb-1')).toHaveTextContent('Customization');
    expect(screen.getByTestId('breadcrumb-2')).toHaveTextContent('Variant variant-a');
  });

  it('[REQ:ADMIN-BREADCRUMB] should render breadcrumb for section editor', () => {
    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>, { route: '/admin/customization/variants/variant-a/sections/1' });

    expect(screen.getByTestId('breadcrumb-0')).toHaveTextContent('Admin');
    expect(screen.getByTestId('breadcrumb-1')).toHaveTextContent('Customization');
    expect(screen.getByTestId('breadcrumb-2')).toHaveTextContent('Variant variant-a');
    expect(screen.getByTestId('breadcrumb-3')).toHaveTextContent('Section 1');
  });

  it('should render children content', () => {
    renderWithRouter(<AdminLayout><div data-testid="child-content">Test Content</div></AdminLayout>);

    expect(screen.getByTestId('child-content')).toHaveTextContent('Test Content');
  });

  it('should call logout API when logout button clicked', async () => {
    const user = userEvent.setup();
    const mockLogout = vi.fn().mockResolvedValue({});
    vi.mocked(api.adminLogout).mockImplementation(mockLogout);

    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>);

    const logoutButton = screen.getByTestId('nav-logout');
    await user.click(logoutButton);

    expect(mockLogout).toHaveBeenCalled();
  });

  it('should display brand name in header', () => {
    renderWithRouter(<AdminLayout><div>Content</div></AdminLayout>);

    expect(screen.getByText('Landing Manager')).toBeInTheDocument();
  });
});
