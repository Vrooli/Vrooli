import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from './renderWithProviders';

describe('renderWithProviders', () => {
  it('supplies a deterministic memory router boundary', () => {
    renderWithProviders(<a href="/apps">Apps</a>, { initialEntries: ['/apps'] });
    expect(screen.getByRole('link', { name: 'Apps' })).toHaveAttribute('href', '/apps');
  });
});
