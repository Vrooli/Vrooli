import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { Users } from 'lucide-react';
import { MemoryRouter } from 'react-router-dom';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { ComingSoonPage } from './ComingSoonPage';

describe('ComingSoonPage', () => {
  it('renders configurable content as React nodes without parsing HTML strings', () => {
    render(<MemoryRouter>
      <ComingSoonPage
        title="User accounts"
        description="Manage customers"
        icon={Users}
        iconBgClass="bg-emerald-500/10"
        iconColorClass="text-emerald-400"
        testId="users"
        intro="Account management will include:"
        features={['Search customers', 'Review entitlement status']}
        alternatives={<>Use <strong>Feedback</strong> while this page is being completed.</>}
      />
    </MemoryRouter>);

    expect(screen.getByTestId('users-header')).toBeInTheDocument();
    expect(screen.getByText('Coming Soon')).toBeInTheDocument();
    expect(screen.getByText('Search customers')).toBeInTheDocument();
    expect(screen.getByText('Feedback').tagName).toBe('STRONG');
  });
});
