import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import DiagnosticsTab from '@/components/tabs/DiagnosticsTab';
import { createMockDiagnostics, createMockDiagnosticsWithWarnings } from './fixtures';

describe('DiagnosticsTab', () => {
  it('shows loading state', () => {
    render(<DiagnosticsTab diagnostics={null} loading />);
    expect(screen.getByText('Running diagnostics...')).toBeInTheDocument();
  });

  it('shows error state with retry', () => {
    const onRetry = vi.fn();
    render(<DiagnosticsTab diagnostics={null} error="Failed to load" onRetry={onRetry} />);
    expect(screen.getByText('Failed to load')).toBeInTheDocument();
    expect(screen.getByText('Retry')).toBeInTheDocument();
  });

  it('shows empty state', () => {
    render(<DiagnosticsTab diagnostics={null} />);
    expect(screen.getByText('No diagnostics available')).toBeInTheDocument();
  });

  it('renders severity badge', () => {
    render(<DiagnosticsTab diagnostics={createMockDiagnostics({ severity: 'ok' })} />);
    const badge = screen.getByText('OK');
    expect(badge).toHaveClass('diagnostics-severity--success');
  });

  it('renders warnings list', () => {
    render(<DiagnosticsTab diagnostics={createMockDiagnosticsWithWarnings(2)} />);
    expect(screen.getByText('Warning 1: Something needs attention')).toBeInTheDocument();
    expect(screen.getByText('Warning 2: Something needs attention')).toBeInTheDocument();
  });

  it('shows success message when no issues', () => {
    render(<DiagnosticsTab diagnostics={createMockDiagnostics()} />);
    expect(screen.getByText('All diagnostics passed! No issues detected.')).toBeInTheDocument();
  });

  it('renders health checks', () => {
    render(
      <DiagnosticsTab
        diagnostics={createMockDiagnostics({
          health_checks: {
            checks: [
              { id: 'hc-1', name: 'API Health', status: 'pass', latency_ms: 42 },
              { id: 'hc-2', name: 'DB Health', status: 'fail', message: 'Connection refused' },
            ],
          },
        })}
      />,
    );
    expect(screen.getByText('API Health')).toBeInTheDocument();
    expect(screen.getByText('42ms')).toBeInTheDocument();
    expect(screen.getByText('DB Health')).toBeInTheDocument();
    expect(screen.getByText('Connection refused')).toBeInTheDocument();
  });
});
