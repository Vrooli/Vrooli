import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { App, Resource } from '@/types';
import { AppGrid } from './TabSwitcherCards';
import { ResourcesSection } from './TabSwitcherSections';

const resource: Resource = { id: 'ssh', name: 'SSH', type: 'service', status: 'online' };
const sortOptions = [{ value: 'status' as const, label: 'Status' }];

describe('ResourcesSection', () => {
  it('keeps known resources visible and retries a failed inventory refresh', () => {
    const onRetry = vi.fn();
    render(
      <ResourcesSection
        resources={[resource]}
        normalizedSearch=""
        sortOption="status"
        onSortChange={vi.fn()}
        onSelect={vi.fn()}
        errorMessage="Unable to refresh resources. Showing last known data."
        onRetry={onRetry}
        sortOptions={sortOptions}
      />,
    );

    expect(screen.getByRole('button', { name: /SSH/ })).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent(/last known data/i);
    fireEvent.click(screen.getByRole('button', { name: 'Retry resource inventory' }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it('keeps the app catalog visible while a refresh is still running', () => {
    render(
      <AppGrid
        apps={[{ id: 'demo', name: 'Demo', scenario_name: 'demo' } as App]}
        onSelect={vi.fn()}
        errorMessage="Unable to refresh scenarios. Showing last known data."
        onRetry={vi.fn()}
        isRefreshing
      />,
    );

    expect(screen.getByRole('button', { name: /demo/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry loading' })).toBeDisabled();
  });
});
