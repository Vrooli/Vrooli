import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { ReactNode } from 'react';
import { TriggerCondition } from '../../../types';
import type { Investigation, InvestigationScript } from '../../../types';
import { AutomaticTriggersSection } from './AutomaticTriggersSection';
import { InvestigationScriptsPanel } from './InvestigationScriptsPanel';
import { InvestigationsPanel } from './InvestigationsPanel';

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
  parseGetTriggersResponse: vi.fn(),
  parseGetCooldownStatusResponse: vi.fn(),
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
  { id: 'disk-check', name: 'Disk check', description: 'Inspect disk pressure', category: 'storage', author: 'system', enabled: true, executionMode: 'native' },
  { id: 'cpu-check', name: 'CPU check', description: 'Inspect CPU pressure', category: 'performance', author: 'system', enabled: true, executionMode: 'native' },
  { id: 'disabled', name: 'Disabled check', description: 'Not available', category: 'system', author: 'system', enabled: false, executionMode: 'shell', skipReason: 'tool missing' },
] as unknown as InvestigationScript[];

function installTriggerFixtures() {
  mocks.protoFetch.mockImplementation((url: string) => {
    if (url === '/investigations/triggers') {
      return Promise.resolve({ triggers: {
        high_cpu: { id: 'high_cpu', name: 'CPU pressure', description: 'CPU is busy', icon: 'cpu', enabled: true, autoFix: false, threshold: 75, unit: '%', condition: TriggerCondition.ABOVE },
        memory_pressure: { id: 'memory_pressure', name: 'Memory pressure', description: 'Memory is low', icon: 'database', enabled: false, autoFix: true, threshold: 10, unit: '%', condition: TriggerCondition.BELOW },
        disk_space: { id: 'disk_space', name: 'Disk space', description: '', icon: 'hard-drive', enabled: true, autoFix: true, threshold: 80, unit: '%', condition: TriggerCondition.ABOVE },
      } });
    }
    if (url === '/investigations/cooldown') {
      return Promise.resolve({ cooldown: { cooldownPeriodSeconds: 300, remainingSeconds: 90, lastTriggerTime: timestamp, isReady: false } });
    }
    if (url === '/metrics/current') return Promise.resolve({ cpuUsage: 62, memoryUsage: 84, tcpConnections: 12 });
    if (url === '/metrics/detailed') return Promise.resolve({ memoryDetails: { diskUsage: { percent: 82 } } });
    if (url === '/metrics/processes') return Promise.resolve({ processHealth: { zombieProcesses: [{}], highThreadCount: [{}, {}] } });
    return Promise.resolve({});
  });
  mocks.apiFetch.mockResolvedValue({});
}

describe('investigation operator panels', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads trigger state and exercises toggles, threshold editing, cooldown, and reset', async () => {
    installTriggerFixtures();
    const onUpdateTrigger = vi.fn();
    render(<AutomaticTriggersSection onUpdateTrigger={onUpdateTrigger} />);

    expect(await screen.findByText('Automatic Triggers')).toBeInTheDocument();
    expect(screen.getByText('CPU pressure')).toBeInTheDocument();
    expect(screen.getByText('1:30')).toBeInTheDocument();
    expect(screen.getByText('62% / 75%')).toBeInTheDocument();

    fireEvent.click(screen.getByTitle('Enable auto-fix'));
    await waitFor(() => { expect(mocks.apiFetch).toHaveBeenCalledWith('/investigations/triggers/high_cpu', expect.objectContaining({ method: 'PUT' })); });
    expect(onUpdateTrigger).toHaveBeenCalledWith('high_cpu', { autoFix: true });

    fireEvent.click(screen.getAllByTitle('Disable trigger')[0] ?? (() => { throw new Error('trigger toggle was not rendered'); })());
    await waitFor(() => { expect(onUpdateTrigger).toHaveBeenCalledWith('high_cpu', { enabled: false }); });

    fireEvent.click(screen.getAllByTitle('Configure threshold')[0]);
    const thresholdInput = screen.getAllByRole('spinbutton')[0];
    if (!thresholdInput) throw new Error('threshold input was not rendered');
    fireEvent.change(thresholdInput, { target: { value: '85' } });
    fireEvent.keyDown(thresholdInput, { key: 'Enter' });
    await waitFor(() => { expect(mocks.apiFetch).toHaveBeenCalledWith('/investigations/triggers/high_cpu/threshold', expect.objectContaining({ method: 'PUT' })); });

    fireEvent.change(screen.getByRole('slider'), { target: { value: '600' } });
    await waitFor(() => { expect(mocks.apiFetch).toHaveBeenCalledWith('/investigations/cooldown/period', expect.objectContaining({ method: 'PUT' })); });
    fireEvent.click(screen.getByRole('button', { name: 'Reset' }));
    await waitFor(() => { expect(mocks.apiFetch).toHaveBeenCalledWith('/investigations/cooldown/reset', expect.objectContaining({ method: 'POST' })); });

    expect(screen.getAllByTitle('Disable auto-fix').some(button => (button as HTMLButtonElement).disabled)).toBe(true);
  });

  it('reports trigger API action failures and supports canceling threshold edits', async () => {
    installTriggerFixtures();
    mocks.apiFetch.mockRejectedValue(new Error('trigger update failed'));
    render(<AutomaticTriggersSection onUpdateTrigger={vi.fn()} />);
    expect(await screen.findByText('Automatic Triggers')).toBeInTheDocument();
    fireEvent.click(screen.getByTitle('Enable auto-fix'));
    await waitFor(() => { expect(mocks.showApiError).toHaveBeenCalledWith(expect.any(Error)); });
    fireEvent.click(screen.getAllByTitle('Configure threshold')[0] ?? (() => { throw new Error('threshold control was not rendered'); })());
    fireEvent.click(screen.getAllByTitle('Cancel')[0] ?? (() => { throw new Error('cancel control was not rendered'); })());
    expect(screen.getAllByTitle('Configure threshold').length).toBeGreaterThan(0);
  });

  it('handles ready cooldowns, missing metrics, unknown icons, and edit cancellation', async () => {
    mocks.apiFetch.mockResolvedValue({});
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url === '/investigations/triggers') return Promise.resolve({ triggers: {
        unknown: { id: 'unknown', name: 'Unknown trigger', description: '', icon: 'mystery', enabled: false, autoFix: true, threshold: 0, unit: '', condition: TriggerCondition.BELOW },
      } });
      if (url === '/investigations/cooldown') return Promise.resolve({ cooldown: { cooldownPeriodSeconds: 300, remainingSeconds: 0, isReady: true } });
      return Promise.resolve({});
    });
    render(<AutomaticTriggersSection onUpdateTrigger={vi.fn()} />);
    expect(await screen.findByText('Automatic Triggers')).toBeInTheDocument();
    expect(screen.getByText('Ready')).toBeInTheDocument();
    expect(screen.getByText('Unknown trigger')).toBeInTheDocument();
    fireEvent.click(screen.getByTitle('Configure threshold'));
    const input = screen.getByRole('spinbutton');
    fireEvent.keyDown(input, { key: 'Escape' });
    expect(screen.getByTitle('Disable auto-fix')).toBeDisabled();
    fireEvent.click(screen.getByTitle('Enable trigger'));
    await waitFor(() => { expect(mocks.apiFetch).toHaveBeenCalledWith('/investigations/triggers/unknown', expect.objectContaining({ method: 'PUT' })); });
    expect(screen.getByTitle('Disable auto-fix')).toBeEnabled();
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
    const investigation = {
      id: 'inv-1', status: 'completed', startTime: timestamp, findings: 'Disk is healthy', progress: 70, confidenceScore: 8,
      details: { operation_mode: 'auto-fix', risk_level: 'low', agent_model: 'model-x', agent_resource: 'ollama', user_note: 'check storage', auto_fix: true },
    } as unknown as Investigation;
    render(<InvestigationsPanel investigations={[investigation]} />);
    expect(screen.getByText('Investigation inv-1')).toBeInTheDocument();
    expect(screen.getByText('Auto-Fix')).toBeInTheDocument();
    expect(screen.getByText('Disk is healthy')).toBeInTheDocument();
    expect(screen.getByText('Progress: 70%')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'RUN ANOMALY CHECK' }));
    await waitFor(() => { expect(mocks.apiFetch).toHaveBeenCalledWith('/investigations/trigger', expect.objectContaining({ method: 'POST' })); });

    mocks.apiFetch.mockRejectedValueOnce(new Error('investigation blocked'));
    fireEvent.click(screen.getByRole('button', { name: 'RUN ANOMALY CHECK' }));
    await waitFor(() => { expect(mocks.showApiError).toHaveBeenCalledWith(expect.any(Error)); });
    render(<InvestigationsPanel investigations={[]} embedded />);
    expect(screen.getByText('No investigations yet')).toBeInTheDocument();
  });
});
