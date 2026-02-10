import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import OverviewTab from '../OverviewTab';
import { createMockApp, createMockDiagnostics, createMockDiagnosticsWithWarnings } from './fixtures';

const defaultProps = () => ({
  app: createMockApp(),
  normalizedStatus: 'running',
  primaryPortLabel: 'UI_PORT',
  primaryPortValue: '3000',
  apiPort: '4000',
  typeLabel: 'SCENARIO',
  uptime: '2h 30m',
  runtime: null as string | null,
  otherPorts: [],
  proxyRoutes: [],
  diagnostics: createMockDiagnostics(),
  diagnosticsLoading: false,
  onOpenDiagnostics: vi.fn(),
});

describe('OverviewTab', () => {
  it('renders status with correct color class', () => {
    render(<OverviewTab {...defaultProps()} />);
    const statusEl = screen.getByText('RUNNING');
    expect(statusEl).toHaveClass('status-running');
  });

  it('renders port values', () => {
    render(<OverviewTab {...defaultProps()} />);
    expect(screen.getByText('3000')).toBeInTheDocument();
    expect(screen.getByText('4000')).toBeInTheDocument();
  });

  it('shows diagnostics alert for passed state', () => {
    render(<OverviewTab {...defaultProps()} />);
    expect(screen.getByText('All diagnostics passed')).toBeInTheDocument();
  });

  it('shows warning count in diagnostics alert', () => {
    render(
      <OverviewTab
        {...defaultProps()}
        diagnostics={createMockDiagnosticsWithWarnings(3)}
      />,
    );
    expect(screen.getByText(/3 diagnostic issues? detected/)).toBeInTheDocument();
  });

  it('shows loading state for diagnostics', () => {
    render(
      <OverviewTab {...defaultProps()} diagnosticsLoading={true} diagnostics={null} />,
    );
    expect(screen.getByText('Loading diagnostics...')).toBeInTheDocument();
  });

  it('renders description when present', () => {
    render(<OverviewTab {...defaultProps()} />);
    expect(screen.getByText('A test application')).toBeInTheDocument();
  });

  it('renders tags', () => {
    render(<OverviewTab {...defaultProps()} />);
    expect(screen.getByText('test')).toBeInTheDocument();
    expect(screen.getByText('demo')).toBeInTheDocument();
  });

  it('does not render description when empty', () => {
    const app = createMockApp({ description: undefined });
    render(<OverviewTab {...defaultProps()} app={app} />);
    expect(screen.queryByText('Description')).not.toBeInTheDocument();
  });

  it('renders other ports when provided', () => {
    render(
      <OverviewTab
        {...defaultProps()}
        otherPorts={[{ label: 'DEBUG_PORT', value: '9229' }]}
      />,
    );
    expect(screen.getByText('DEBUG_PORT')).toBeInTheDocument();
    expect(screen.getByText('9229')).toBeInTheDocument();
  });
});
