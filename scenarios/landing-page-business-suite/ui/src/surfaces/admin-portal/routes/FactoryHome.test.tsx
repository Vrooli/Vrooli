import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import FactoryHome from './FactoryHome';

vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));

describe('FactoryHome', () => {
  it('clearly directs operators to generate and operate the resulting landing scenario', () => {
    render(<FactoryHome />);

    expect(screen.getByRole('heading', { name: 'Generate landing-page scenarios, not a landing page.' })).toBeInTheDocument();
    expect(screen.getByText('saas-landing-page')).toBeInTheDocument();
    expect(screen.getByText(/template-manager lifecycle generate landing-page-business-suite/)).toBeInTheDocument();
    expect(screen.getByText(/admin portal lives in the template/i)).toBeInTheDocument();
  });
});
