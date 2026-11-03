import { fireEvent, screen, waitFor } from '@testing-library/react';
import { create } from '@bufbuild/protobuf';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { InvestigationSchema, InvestigationStatus } from '@vrooli/proto-types/system-monitor/v1/investigations/investigations_pb';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { ReactNode } from 'react';
import type { InvestigationScript } from '../../../types';
import { InvestigationScriptsPanel } from './InvestigationScriptsPanel';
import { InvestigationsPanel } from './InvestigationsPanel';
import { InvestigationScriptSchema } from '@vrooli/proto-types/system-monitor/v1/scripts/scripts_pb';

const mocks = vi.hoisted(() => ({
  protoFetch: vi.fn(),
  apiFetch: vi.fn(),
  showApiError: vi.fn(),
}));

vi.mock('../../../shared/api/apiFetch', () => ({
  protoFetch: mocks.protoFetch,
  apiFetch: mocks.apiFetch,
  extractErrorMessage: (error: unknown, fallback: string) => error instanceof Error ? error.message : fallback,
}));
vi.mock('../../../shared/api/proto-contracts', () => ({
  parseMetricsResponse: vi.fn(),
  parseDetailedMetrics: vi.fn(),
  parseProcessMonitorData: vi.fn(),
  parseListScriptsResponse: vi.fn(),
  parseListRunsResponse: vi.fn(),
  parseGetScriptResponse: vi.fn(),
}));
vi.mock('../../../shared/hooks/usePolling', () => ({ usePolling: vi.fn() }));
vi.mock('../../../shared/components/ToastProvider', () => ({
  ToastProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  useToast: () => ({ showApiError: mocks.showApiError }),
}));

const timestamp = timestampFromDate(new Date('2026-01-01T00:00:00Z'));

const scripts: InvestigationScript[] = [
  create(InvestigationScriptSchema, { id: 'disk-check', name: 'Disk check', description: 'Inspect disk pressure', category: 'storage', author: 'system', enabled: true, executionMode: 'native' }),
  create(InvestigationScriptSchema, { id: 'cpu-check', name: 'CPU check', description: 'Inspect CPU pressure', category: 'performance', author: 'system', enabled: true, executionMode: 'native' }),
  create(InvestigationScriptSchema, { id: 'disabled', name: 'Disabled check', description: 'Not available', category: 'system', author: 'system', enabled: false, executionMode: 'shell', skipReason: 'tool missing' }),
];

describe('investigation operator panels', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads, filters, opens, refreshes, and creates investigation scripts', async () => {
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url === '/investigations/scripts') return Promise.resolve({ scripts });
      if (url === '/investigations/runs?limit=5') return Promise.resolve({ runs: [
        { id: 'run-1', entryId: 'disk-check', status: 'completed', durationSeconds: 0.12 },
      ] });
      return Promise.resolve({ script: scripts[0], content: 'echo disk' });
    });
    const onOpen = vi.fn();
    const onShowAll = vi.fn();
    render(<InvestigationScriptsPanel onOpenScriptEditor={onOpen} maxVisible={1} onShowAll={onShowAll} />);
    expect(await screen.findByText('INVESTIGATION SCRIPTS')).toBeInTheDocument();
    expect(await screen.findByText('Disk check')).toBeInTheDocument();
    expect(screen.queryByText('Disabled check')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Show More Scripts' }));
    expect(onShowAll).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText('Disk check'));
    await waitFor(() => { expect(onOpen).toHaveBeenCalledWith(scripts[0], 'echo disk', 'view'); });
    fireEvent.click(screen.getByRole('button', { name: 'NEW SCRIPT' }));
    expect(onOpen).toHaveBeenCalledWith(undefined, '', 'create');
    fireEvent.click(screen.getByRole('button', { name: 'REFRESH' }));
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledTimes(3); });
    fireEvent.click(screen.getByRole('button', { name: 'HISTORY' }));
    expect(await screen.findByText(/Recent runs: disk-check completed/)).toBeInTheDocument();
  });

  it('shows script loading failures, retries them, and supports embedded empty state', async () => {
    mocks.protoFetch.mockRejectedValueOnce(new Error('scripts unavailable')).mockResolvedValueOnce({ scripts: [] });
    const onOpen = vi.fn();
    render(<InvestigationScriptsPanel onOpenScriptEditor={onOpen} />);
    expect(await screen.findByText('FAILED TO LOAD SCRIPTS')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'RETRY' }));
    await waitFor(() => expect(screen.getByText('NO SCRIPTS AVAILABLE')).toBeInTheDocument());
    render(<InvestigationScriptsPanel onOpenScriptEditor={onOpen} embedded />);
    expect(await screen.findByText('NO SCRIPTS AVAILABLE')).toBeInTheDocument();
  });

  it('handles empty and unavailable run history', async () => {
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url === '/investigations/scripts') return Promise.resolve({ scripts: [scripts[0]] });
      if (url === '/investigations/runs?limit=5') return Promise.resolve({});
      return Promise.resolve({});
    });
    render(<InvestigationScriptsPanel onOpenScriptEditor={vi.fn()} />);
    expect(await screen.findByText('INVESTIGATION SCRIPTS')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'HISTORY' }));
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledWith('/investigations/runs?limit=5', expect.anything()); });
    expect(screen.queryByText(/Recent runs:/)).not.toBeInTheDocument();

    mocks.protoFetch.mockRejectedValueOnce(new Error('history unavailable'));
    fireEvent.click(screen.getByRole('button', { name: 'HISTORY' }));
    await waitFor(() => { expect(mocks.showApiError).toHaveBeenCalledWith(expect.any(Error)); });
  });

  it('filters scripts by metadata', async () => {
    mocks.protoFetch.mockResolvedValue({ scripts });
    render(<InvestigationScriptsPanel onOpenScriptEditor={vi.fn()} searchFilter="storage" />);
    expect(await screen.findByText('Disk check')).toBeInTheDocument();
    expect(screen.queryByText('CPU check')).not.toBeInTheDocument();
  });

  it('renders investigation metadata, empty states, and trigger failures', async () => {
    mocks.apiFetch.mockResolvedValue({});
    const investigation = create(InvestigationSchema, {
      id: 'inv-1', status: InvestigationStatus.COMPLETED, startTime: timestamp,
      findings: 'Disk is healthy', progress: 70, confidenceScore: 0.8,
      details: {
        operation_mode: 'auto-fix', risk_level: 'low', agent_model: 'model-x',
        agent_resource: 'ollama', user_note: 'check storage', auto_fix: true,
      },
    });
    render(<InvestigationsPanel investigations={[investigation]} />);
    expect(screen.getByText('Investigation inv-1')).toBeInTheDocument();
    expect(screen.getByText('completed')).toBeInTheDocument();
    expect(screen.getByText('Auto-Fix')).toBeInTheDocument();
    expect(screen.getByText('Disk is healthy')).toBeInTheDocument();
    expect(screen.getByText('Progress: 70%')).toBeInTheDocument();
    expect(screen.getByText('80%')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'RUN ANOMALY CHECK' }));
    await waitFor(() => { expect(mocks.apiFetch).toHaveBeenCalledWith('/investigations/trigger', expect.objectContaining({ method: 'POST' })); });

    mocks.apiFetch.mockRejectedValueOnce(new Error('investigation blocked'));
    fireEvent.click(screen.getByRole('button', { name: 'RUN ANOMALY CHECK' }));
    await waitFor(() => { expect(mocks.showApiError).toHaveBeenCalledWith(expect.any(Error)); });
    render(<InvestigationsPanel investigations={[]} embedded />);
    expect(screen.getByText('No investigations yet')).toBeInTheDocument();
  });
});
