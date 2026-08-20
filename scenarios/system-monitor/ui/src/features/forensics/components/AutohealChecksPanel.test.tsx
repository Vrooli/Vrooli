import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import { AutohealChecksPanel } from './AutohealChecksPanel';

describe('AutohealChecksPanel', () => {
  it('renders unavailable and empty states honestly', () => {
    const unavailable = render(<AutohealChecksPanel envelope={{ available: false, reason: '' }} />);
    expect(screen.getByText('Autoheal Checks')).toBeInTheDocument();
    expect(screen.getByText('autoheal offline')).toBeInTheDocument();
    unavailable.unmount();

    render(<AutohealChecksPanel envelope={{ available: true }} />);
    expect(screen.getByText('No forensics-relevant checks reported.')).toBeInTheDocument();
  });

  it('renders status classes and optional messages', () => {
    render(<AutohealChecksPanel envelope={{
      available: true,
      checks: [
        { checkId: 'ok', status: 'OK', message: 'healthy' },
        { checkId: 'failed', status: 'failed', message: '' },
        { checkId: 'warn', status: 'warning' },
        { checkId: 'unknown', status: 'pending' },
      ],
    }} />);
    expect(screen.getByText('healthy')).toBeInTheDocument();
    expect(screen.getByText('OK')).toHaveClass('text-success');
    expect(screen.getAllByText('failed').find(element => element.classList.contains('text-error'))).toBeDefined();
    expect(screen.getByText('warning')).toHaveClass('text-warning');
    expect(screen.getByText('pending')).toHaveClass('text-muted');
  });
});
