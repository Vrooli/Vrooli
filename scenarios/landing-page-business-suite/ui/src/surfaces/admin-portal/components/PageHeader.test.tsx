import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Users } from 'lucide-react';
import { PageHeader } from './PageHeader';
import { Button } from '../../../shared/ui/button';

describe('PageHeader', () => {
  it('renders icon with title and description', () => {
    render(
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
    render(
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
    const { container } = render(
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
    render(
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
    render(
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
});
