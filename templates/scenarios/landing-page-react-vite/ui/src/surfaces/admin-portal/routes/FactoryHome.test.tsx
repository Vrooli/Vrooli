import { describe, it, expect } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils';
import FactoryHome from './FactoryHome';

describe('FactoryHome', () => {
  it('renders the factory heading and template guidance', () => {
    render(<FactoryHome />);
    expect(
      screen.getByRole('heading', { name: /Generate landing-page scenarios, not a landing page\./i }),
    ).toBeInTheDocument();
    expect(screen.getByText('Landing Page Template')).toBeInTheDocument();
    expect(screen.getByText('saas-landing-page')).toBeInTheDocument();
  });

  it('shows the CLI generation command and the ordered working steps', () => {
    render(<FactoryHome />);
    expect(
      screen.getByText(/template-manager lifecycle generate landing-page-react-vite/i),
    ).toBeInTheDocument();

    const steps = screen.getAllByRole('listitem');
    expect(steps).toHaveLength(4);
    expect(steps[0]).toHaveTextContent(/Generate a new landing scenario via the CLI/i);
  });

  it('warns that the admin portal belongs in the template, not the factory', () => {
    render(<FactoryHome />);
    expect(screen.getByText('Admin portal lives in the template')).toBeInTheDocument();
  });
});
