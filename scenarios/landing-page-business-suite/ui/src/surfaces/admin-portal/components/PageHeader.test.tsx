import { describe, it, expect } from 'vitest';
import type { ReactElement } from 'react';
import { render, screen } from '@testing-library/react';
import { Users } from 'lucide-react';
import { MemoryRouter } from 'react-router-dom';
import { PageHeader } from './PageHeader';
import { Button } from '../../../shared/ui/button';

const renderWithRouter = (ui: ReactElement, route = '/__test') => {
  return render(
    <MemoryRouter initialEntries={[route]}>
      {ui}
    </MemoryRouter>
  );
};

describe('PageHeader', () => {
  it('renders icon with title and description', () => {
    renderWithRouter(
      <PageHeader
        title="User Accounts"
        description="Manage users"
        icon={Users}
        iconBgClass="bg-emerald-500/10"
        iconColorClass="text-emerald-400"
        testId="header"
      />
    );

    expect(screen.getByTestId('header')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'User Accounts' })).toBeInTheDocument();
    expect(screen.getByText('Manage users')).toBeInTheDocument();
  });

  it('renders without description', () => {
    renderWithRouter(
      <PageHeader
        title="Settings"
        icon={Users}
        iconBgClass="bg-emerald-500/10"
        iconColorClass="text-emerald-400"
        testId="header"
      />
    );

    expect(screen.getByRole('heading', { name: 'Settings' })).toBeInTheDocument();
  });

  it('applies icon styling classes', () => {
    const { container } = renderWithRouter(
      <PageHeader
        title="User Accounts"
        icon={Users}
        iconBgClass="bg-emerald-500/10"
        iconColorClass="text-emerald-400"
        testId="header"
      />
    );

    expect(container.querySelector('.bg-emerald-500\\/10')).toBeInTheDocument();
  });

  it('renders with actions', () => {
    renderWithRouter(
      <PageHeader
        title="User Accounts"
        icon={Users}
        iconBgClass="bg-emerald-500/10"
        iconColorClass="text-emerald-400"
        actions={<Button data-testid="action-btn">Add User</Button>}
        testId="header"
      />
    );

    expect(screen.getByTestId('action-btn')).toBeInTheDocument();
  });

  it('accepts deprecated variant prop for backwards compatibility', () => {
    renderWithRouter(
      <PageHeader
        variant="icon-title"
        title="User Accounts"
        icon={Users}
        iconBgClass="bg-emerald-500/10"
        iconColorClass="text-emerald-400"
        testId="header"
      />
    );

    expect(screen.getByRole('heading', { name: 'User Accounts' })).toBeInTheDocument();
  });

  it('shows documentation link when page has docs', () => {
    renderWithRouter(
      <PageHeader
        title="Admin Home"
        icon={Users}
        iconBgClass="bg-emerald-500/10"
        iconColorClass="text-emerald-400"
        testId="header"
      />,
      '/admin'
    );

    const docLink = screen.getByTestId('page-docs-link');
    expect(docLink).toBeInTheDocument();
    expect(docLink).toHaveAttribute('href', '/admin/docs?doc=guides%2FADMIN_GUIDE.md#admin-home');
  });
});
