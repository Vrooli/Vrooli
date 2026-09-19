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

  it('puts failing checks first regardless of the order the envelope supplied', () => {
    // On a crash-forensics surface a CRITICAL check buried among twenty OK
    // rows is a check nobody reads. The envelope does not guarantee an order,
    // so the panel imposes one.
    render(<AutohealChecksPanel envelope={{
      available: true,
      checks: [
        { checkId: 'a-ok', status: 'ok' },
        { checkId: 'b-pending', status: 'pending' },
        { checkId: 'c-warning', status: 'warning' },
        { checkId: 'd-critical', status: 'critical' },
      ],
    }} />);

    const rendered = screen.getAllByRole('listitem').map(item => item.textContent ?? '');
    expect(rendered[0]).toContain('d-critical');
    expect(rendered[1]).toContain('c-warning');
    // An unrecognised verdict is not reassuring, so it must not sort below ok.
    expect(rendered[2]).toContain('b-pending');
    expect(rendered[3]).toContain('a-ok');
  });

  it('does not mutate the caller\'s checks array while sorting', () => {
    const checks = [
      { checkId: 'a-ok', status: 'ok' },
      { checkId: 'd-critical', status: 'critical' },
    ];
    render(<AutohealChecksPanel envelope={{ available: true, checks }} />);
    expect(checks.map(c => c.checkId)).toEqual(['a-ok', 'd-critical']);
  });
});
