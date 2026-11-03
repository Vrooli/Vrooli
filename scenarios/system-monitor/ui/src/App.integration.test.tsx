// provider-free-exception: App is the root component and owns BrowserRouter plus all application providers.
import { act, fireEvent, render, screen, waitFor, cleanup } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App, { revealWorkspace, WorkspaceFailure } from './App';
import { fromJson } from '@bufbuild/protobuf';
import { GraphSchema } from '@vrooli/proto-types/system-monitor/v1/devicegraph/devicegraph_pb';

type Parser = (value: unknown) => unknown;

const mocks = vi.hoisted(() => ({
  useSystemMonitor: vi.fn(),
  useServerAlerts: vi.fn(),
  useInvestigationAgents: vi.fn(),
  useScriptExecution: vi.fn(),
  protoFetch: vi.fn(),
  apiFetch: vi.fn(),
  fetchForensicsSummary: vi.fn(),
  fetchCapacityOverview: vi.fn(),
  fetchCapacityReconciliation: vi.fn(),
  fetchCapacityPolicy: vi.fn(),
  setCapacityPolicy: vi.fn(),
  fetchLogs: vi.fn(),
  fetchUnits: vi.fn(),
  fetchBoots: vi.fn(),
}));

vi.mock('./features/monitoring/hooks/useServerAlerts', async importOriginal => {
  const actual = await importOriginal<typeof import('./features/monitoring/hooks/useServerAlerts')>();
  return { useServerAlerts: (enabled: boolean) => mocks.useServerAlerts.getMockImplementation() ? mocks.useServerAlerts(enabled) : actual.useServerAlerts(enabled) };
});
vi.mock('./features/monitoring/hooks/useSystemMonitor', () => ({
  useSystemMonitor: mocks.useSystemMonitor,
}));
vi.mock('./features/investigations/hooks/useInvestigationAgents', () => ({
  useInvestigationAgents: mocks.useInvestigationAgents,
}));
vi.mock('./features/investigations/hooks/useScriptExecution', () => ({
  useScriptExecution: mocks.useScriptExecution,
}));
vi.mock('./shared/api/apiFetch', () => ({
  protoFetch: mocks.protoFetch,
  apiFetch: mocks.apiFetch,
  extractErrorMessage: (error: unknown, fallback: string) => error instanceof Error ? error.message : fallback,
  isApiError: () => false,
  toApiError: (error: unknown) => ({ error: error instanceof Error ? error.message : 'request failed' }),
}));
vi.mock('./features/forensics/api', () => ({
  fetchForensicsSummary: mocks.fetchForensicsSummary,
}));
vi.mock('./features/capacity/api', () => ({
  fetchCapacityOverview: mocks.fetchCapacityOverview,
  fetchCapacityReconciliation: mocks.fetchCapacityReconciliation,
  fetchCapacityPolicy: mocks.fetchCapacityPolicy,
  setCapacityPolicy: mocks.setCapacityPolicy,
}));
vi.mock('./features/logs/api', () => ({
  fetchLogs: mocks.fetchLogs,
  fetchUnits: mocks.fetchUnits,
  fetchBoots: mocks.fetchBoots,
}));

const timestamp = { seconds: BigInt(1770000000), nanos: 0 };
const metric = (value: number) => ({ state: { case: 'measured' as const, value }, observedAt: timestamp });

const systemMetrics = {
  timestamp,
  cpu: metric(22),
  memory: metric(44),
  gpu: metric(12),
  disk: metric(67),
  connections: metric(8),
};

const detailedMetrics = {
  timestamp,
  cpuDetails: {
    usage: 22,
    loadAverage: [0.2, 0.3, 0.4],
    contextSwitches: 12,
    totalGoroutines: 7,
    topProcesses: [{ name: 'api', pid: 10, cpuPercent: 3, memoryMb: 40 }],
  },
  memoryDetails: {
    usage: 44,
    diskUsage: { used: BigInt(7), total: BigInt(10), percent: 70 },
    swapUsage: { used: BigInt(1), total: BigInt(4), percent: 25 },
    topProcesses: [{ name: 'api', pid: 10, cpuPercent: 3, memoryMb: 40 }],
  },
  networkDetails: {
    tcpStates: { total: 8, established: 5, listening: 2, timeWait: 1 },
    networkStats: { bandwidthInMbps: 1.2, bandwidthOutMbps: 0.8, packetLoss: 0.1, dnsSuccessRate: 99, dnsLatencyMs: 4 },
    portUsage: { used: 8, total: 64 },
    connectionPools: [{ name: 'db', active: 2, idle: 3, waiting: 0, maxSize: 10, leakRisk: 'low', healthy: true }],
  },
  gpuDetails: {
    driverVersion: 'test-driver',
    primaryModel: 'Test GPU',
    summary: { deviceCount: 1, averageUtilizationPercent: 12, usedMemoryMb: 100, totalMemoryMb: 1000, averageTemperatureC: 42 },
    devices: [{ index: 0, uuid: 'gpu-0', name: 'Test GPU', utilizationPercent: 12, memoryUsedMb: 100, memoryTotalMb: 1000, temperatureC: 42, processes: [] }],
    errors: [],
  },
  systemDetails: {
    fileDescriptors: { used: 10, max: 100, percent: 10 },
    inotifyWatchers: { supported: true, watchesUsed: 2, watchesMax: 10, watchesPercent: 20, instancesUsed: 1, instancesMax: 10, instancesPercent: 10 },
    serviceDependencies: [{ name: 'api', status: 'healthy', latencyMs: 2 }],
    certificates: [{ domain: 'localhost', status: 'valid', daysToExpiry: 100 }],
  },
};

const infrastructureData = {
  timestamp,
  databasePools: [{ name: 'db', active: 1, idle: 2, maxSize: 10, healthy: true }],
  httpClientPools: [{ name: 'http', active: 1, waiting: 0, maxSize: 10, healthy: true, leakRisk: 'low' }],
  messageQueues: { redisPubsub: { subscribers: 1, channels: 2 }, backgroundJobs: { pending: 0, active: 1, failed: 0 } },
  storageIo: { readMbPerSec: 1, writeMbPerSec: 2, diskQueueDepth: 0.2, ioWaitPercent: 1 },
};

const processMonitorData = {
  processHealth: { totalProcesses: 10, zombieProcesses: [], highThreadCount: [], leakCandidates: [] },
};

const history = {
  windowSeconds: 3600,
  sampleIntervalSeconds: 60,
  cpu: [{ timestamp: '2026-02-02T00:00:00Z', value: 22 }],
  memory: [{ timestamp: '2026-02-02T00:00:00Z', value: 44 }],
  network: [{ timestamp: '2026-02-02T00:00:00Z', value: 8 }],
  gpu: [{ timestamp: '2026-02-02T00:00:00Z', value: 12 }],
  diskUsage: [{ timestamp: '2026-02-02T00:00:00Z', value: 67 }],
  diskRead: [{ timestamp: '2026-02-02T00:00:00Z', value: 1 }],
  diskWrite: [{ timestamp: '2026-02-02T00:00:00Z', value: 2 }],
};

const forensicsSummary = {
  generatedAt: '2026-02-02T00:00:00Z',
  pstore: { available: false, reason: 'pstore unavailable' },
  bootHistory: { available: true, data: { boots: [] } },
  mce: { available: true, data: { window: '1h', uncorrected: 0, corrected: 0 } },
  autoheal: { available: true, checks: [{ checkId: 'disk', status: 'OK', message: 'healthy' }] },
};

const fleet = [
  { id: '', name: 'This machine', os: 'linux', arch: 'x86_64', online: true, heartbeat_fresh: true, dispatchable: true, status: 'local' },
  { id: 'minimouse-id', name: 'minimouse', os: 'darwin', arch: 'amd64', online: true, heartbeat_fresh: true, heartbeat_age_seconds: 8, dispatchable: true, status: 'NODE_STATUS_ONLINE', grant: 'Read and operate; destructive actions withheld', scopes: ['*:read', '*:write'] },
  { id: 'swarminator-id', name: 'swarminator', os: 'linux', arch: 'amd64', online: false, heartbeat_fresh: false, heartbeat_age_seconds: 639479, dispatchable: false, status: 'NODE_STATUS_OFFLINE', readiness: [{ identity: 'registry_record', passed: true }, { identity: 'heartbeat_fresh', passed: false }] },
];

function installFixtures() {
  mocks.useServerAlerts.mockReset();
  // The dashboard asks for the fleet on mount. An unmocked apiFetch resolves to
  // undefined, which the machine control cannot tell apart from "no machines".
  mocks.apiFetch.mockImplementation((path: string) =>
    path === '/machines' ? Promise.resolve(fleet) : Promise.resolve(undefined)
  );
  mocks.useSystemMonitor.mockReturnValue({
    metrics: systemMetrics,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
    investigations: [],
    metricHistory: history,
    isLoading: false,
    error: null,
    subsystemErrors: {},
    isStale: false,
    retryAttempt: 0,
    lastSuccessfulFetch: new Date('2026-02-02T00:00:00Z'),
    lastAttemptAt: new Date('2026-02-02T00:00:00Z'),
    retryIntervalSeconds: 5,
    healthStatus: { status: 'healthy', service: 'system-monitor', maintenance_state: 'active' },
    healthError: null,
    toggleMonitoring: vi.fn().mockResolvedValue(undefined),
    refreshHealth: vi.fn().mockResolvedValue(undefined),
    refresh: vi.fn(),
    refreshMetrics: vi.fn(),
  });
  mocks.useInvestigationAgents.mockReturnValue({
    agents: [],
    isSpawningAgent: false,
    spawnAgentError: null,
    stoppingAgentIds: new Set<string>(),
    agentErrors: {},
    refreshAgents: vi.fn().mockResolvedValue(undefined),
    spawnAgent: vi.fn().mockResolvedValue(undefined),
    stopAgent: vi.fn().mockResolvedValue(undefined),
  });
  mocks.useScriptExecution.mockReturnValue({
    modalState: { reportModal: { isOpen: false, loading: false }, scriptEditor: { isOpen: false, mode: 'view' }, scriptResults: { isOpen: false } },
    openScriptEditor: vi.fn(),
    closeScriptEditor: vi.fn(),
    closeScriptResults: vi.fn(),
    executeScript: vi.fn().mockResolvedValue(undefined),
    saveScript: vi.fn().mockResolvedValue(undefined),
  });
  mocks.fetchForensicsSummary.mockResolvedValue(forensicsSummary);
  mocks.fetchCapacityOverview.mockResolvedValue({ success: true, sensingAvailable: true, warnings: [], gpus: [], claims: [] });
  mocks.fetchCapacityReconciliation.mockResolvedValue({ success: true, findings: [] });
  mocks.fetchCapacityPolicy.mockResolvedValue([]);
  mocks.setCapacityPolicy.mockResolvedValue([]);
  mocks.fetchUnits.mockResolvedValue({ available: true, units: ['systemd'] });
  mocks.fetchBoots.mockResolvedValue({ available: true, boots: [] });
  mocks.fetchLogs.mockResolvedValue({ available: true, entries: [], nextCursor: undefined });
  mocks.protoFetch.mockImplementation((url: string, parser: Parser) => {
    if (url === '/reports') return parser({ reports: [] });
    if (url === '/investigations/scripts') return parser({ scripts: [] });
    if (url.includes('/metrics/disk/details')) return parser({ active_mount: '/', depth: 2, partitions: [], top_directories: [], largest_files: [] });
    return parser({});
  });
}

async function openPath(path: string) {
  window.history.pushState({}, '', path);
  window.dispatchEvent(new PopStateEvent('popstate'));
  // Each caller waits for the route-specific landmark below. Waiting for a
  // generic loading string here races lazy route transitions and can leave a
  // valid route test timing out on an unrelated fallback surface.
  await act(async () => { await Promise.resolve(); });
}

describe('App production surfaces', () => {
  it('offers an explicit reload when a workspace cannot render', () => {
    const reload = vi.fn();
    render(<WorkspaceFailure name="Investigations section" reload={reload} />);
    expect(screen.getByRole('alert')).toHaveTextContent('unsaved changes may be lost');
    expect(reload).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Reload page' }));
    expect(reload).toHaveBeenCalledOnce();
  });
  beforeEach(() => {
    installFixtures();
    Object.defineProperty(window, 'innerWidth', { value: 1440, configurable: true });
    window.history.replaceState({}, '', '/');
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('discloses the complete device graph without inventing a count for unknown states', async () => {
    const base = mocks.useSystemMonitor();
    const view = render(<App />);
    expect(await screen.findByText('Waiting for device graph')).toBeInTheDocument();
    expect(screen.getByText('Waiting for device graph')).toHaveAttribute('aria-live', 'polite');
    const summary = screen.getByText('Device graph', { selector: 'summary > span' }).closest('summary');
    if (!summary) throw new Error('Device graph summary was not rendered');
    expect(summary.parentElement).not.toHaveAttribute('open');
    mocks.useSystemMonitor.mockReturnValue({ ...base, error: { error: 'unrelated metrics failure' } });
    view.rerender(<App />);
    expect(screen.getByText('Device graph not received')).toBeInTheDocument();
    expect(summary).not.toHaveTextContent('unrelated metrics failure');
    mocks.useSystemMonitor.mockReturnValue({ ...base, subsystemErrors: { deviceGraph: { error: 'device read failed' } } });
    view.rerender(<App />);
    expect(screen.getByText('Unavailable: device read failed')).toBeInTheDocument();
    const unavailable = fromJson(GraphSchema, { available: false, unavailableReason: 'not supported' });
    mocks.useSystemMonitor.mockReturnValue({ ...base, deviceGraph: unavailable });
    view.rerender(<App />);
    expect(screen.getByText('Unavailable: not supported')).toBeInTheDocument();
    const empty = fromJson(GraphSchema, { available: true, devices: [], subsystems: [] });
    mocks.useSystemMonitor.mockReturnValue({ ...base, deviceGraph: empty });
    view.rerender(<App />);
    expect(screen.getByText('0 devices · 0 subsystems')).toBeInTheDocument();
    mocks.useSystemMonitor.mockReturnValue({ ...base, deviceGraph: empty, subsystemErrors: { deviceGraph: { error: 'refresh failed' } } });
    view.rerender(<App />);
    expect(screen.getByText('0 devices · 0 subsystems · Last received; refresh failed: refresh failed')).toBeInTheDocument();
    fireEvent.click(summary);
    expect(summary.parentElement).toHaveAttribute('open');
    expect(screen.getByText('No graded devices were observed.')).toBeVisible();
    mocks.useSystemMonitor.mockReturnValue({ ...base, deviceGraph: empty });
    view.rerender(<App />);
    expect(screen.getByText('0 devices · 0 subsystems')).toBeInTheDocument();
    fireEvent.click(summary);
    expect(summary.parentElement).not.toHaveAttribute('open');
  });

  it('carries the selected condition snapshot into detail and restores its action on return', async () => {
    mocks.useServerAlerts.mockReturnValue({ alerts: [{ id: 'cpu-other', category: 'cpu_other', severity: 'critical', message: 'Other CPU condition', timestamp: '2026-02-02T00:01:00Z', resolved: false }, { id: 'cpu-alert', category: 'cpu_usage', severity: 'critical', message: 'CPU elevated', timestamp: '2026-02-02T00:00:00Z', resolved: false }], feed: { status: 'error', error: 'offline', ruleIssues: [] } });
    const scroll = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
    render(<App />);
    const action = screen.getAllByRole('button', { name: 'Inspect cpu' }).at(1);
    if (!action) throw new Error('Expected the second CPU inspection action');
    action.focus();
    fireEvent.click(action);
    const context = await screen.findByRole('region', { name: 'Originating condition' }, { timeout: 5000 });
    expect(context).toHaveTextContent('CRITICAL · cpu_usage');
    fireEvent.click(screen.getByText('CRITICAL · cpu_usage — View condition details'));
    expect(context).toHaveTextContent('CPU elevated');
    expect(context).toHaveTextContent('Last known condition; it was not revalidated.');
    expect(context).toHaveTextContent('Overview window bounds unavailable at navigation');
    fireEvent.click(await screen.findByRole('button', { name: /Back To Dashboard/i }));
    await waitFor(() => { expect(screen.getAllByRole('button', { name: 'Inspect cpu' })[1]).toHaveFocus(); });
    expect(screen.queryByRole('region', { name: 'Originating condition' })).not.toBeInTheDocument();
    scroll.mockRestore();
  });

  it('records the evaluated condition and observation window at navigation', async () => {
    const base = mocks.useSystemMonitor();
    mocks.useSystemMonitor.mockReturnValue({ ...base, metricHistory: { ...history, windowStart: '2026-02-01T23:00:00Z', windowEnd: '2026-02-02T00:00:00Z' } });
    mocks.useServerAlerts.mockReturnValue({ alerts: [{ id: 'cpu-alert', category: 'cpu_usage', severity: 'critical', message: 'CPU elevated', timestamp: '2026-02-01T23:57:00Z', resolved: false }], feed: { status: 'ready', evaluatedAt: '2026-02-02T00:00:00Z', ruleIssues: [] } });
    render(<App />);
    fireEvent.click(screen.getByRole('button', { name: 'Inspect cpu' }));
    const context = await screen.findByRole('region', { name: 'Originating condition' }, { timeout: 5000 });
    expect(context).toHaveTextContent('Condition status at navigation; it may have changed.');
    expect(context).toHaveTextContent('Last evaluation at navigation:');
    expect(context).toHaveTextContent('Overview window at navigation:');
    expect(context).not.toHaveTextContent('bounds unavailable');
  });

  it('makes the machine the subject of the app, and puts it in the URL', async () => {
    render(<App />);

    const picker = await screen.findByTestId('machine-picker');
    expect(picker).toHaveTextContent('This machine');
    fireEvent.click(picker);
    fireEvent.click(await screen.findByRole('option', { name: /minimouse/ }));

    // The subject is stated for as long as it is not this computer.
    const strip = (await screen.findAllByRole('status')).find(element => element.textContent?.includes('Viewing minimouse'));
    expect(strip).toHaveTextContent('Viewing minimouse — darwin/amd64, not this computer');

    // In the URL, so a reload keeps the subject and the view is shareable.
    expect(window.location.search).toBe('?machine=minimouse-id');

    fireEvent.click(screen.getByTestId('back-to-local'));
    expect(window.location.search).toBe('');
    expect(await screen.findByTestId('machine-picker')).toHaveTextContent('This machine');
  });

  it('keeps the subject visible and retries machine discovery after failure', async () => {
    window.history.replaceState({}, '', '/logs');
    mocks.apiFetch.mockImplementationOnce(() => Promise.reject(new Error('unavailable')));
    render(<App />);
    const retry = await screen.findByRole('button', { name: 'This machine; machine list unavailable; retry' });
    fireEvent.click(retry);
    const picker = await screen.findByTestId('machine-picker');
    expect(picker).toHaveTextContent('This machine');
    await waitFor(() => expect(picker).toHaveFocus());
    expect(window.location.pathname).toBe('/logs');
  });

  it('does not steal focus if the operator moves away while discovery retries', async () => {
    let resolveRetry!: (value: typeof fleet) => void;
    mocks.apiFetch.mockImplementationOnce(() => Promise.reject(new Error('unavailable')))
      .mockImplementationOnce(() => new Promise(resolve => { resolveRetry = resolve; }));
    render(<App />);
    const retry = await screen.findByRole('button', { name: 'This machine; machine list unavailable; retry' });
    retry.focus();
    fireEvent.click(retry);
    const range = screen.getByLabelText('Shared time range');
    range.focus();
    act(() => { resolveRetry(fleet); });
    expect(await screen.findByTestId('machine-picker')).toBeInTheDocument();
    expect(range).toHaveFocus();
  });

  it('retains a remote subject and withholds local Logs while discovery recovers', async () => {
    window.history.replaceState({}, '', '/logs?machine=minimouse-id');
    mocks.apiFetch.mockImplementationOnce(() => Promise.reject(new Error('unavailable')));
    render(<App />);
    const retry = await screen.findByRole('button', { name: 'minimouse-id; machine list unavailable; retry' });
    expect(screen.getByText(/Local evidence is not shown under its name/)).toBeInTheDocument();
    expect(mocks.fetchLogs).not.toHaveBeenCalled();
    fireEvent.click(retry);
    expect(await screen.findByTestId('machine-picker')).toHaveTextContent('minimouse');
    expect(window.location.search).toBe('?machine=minimouse-id');
    expect(mocks.fetchLogs).not.toHaveBeenCalled();
  });

  it('releases an inspected chart when the subject changes', async () => {
    render(<App />);
    const chart = screen.getByRole('img', { name: 'CPU USAGE history' });
    fireEvent.keyDown(chart, { key: 'ArrowLeft' });
    expect(screen.getByRole('button', { name: 'Return to latest' })).toBeInTheDocument();
    fireEvent.click(await screen.findByTestId('machine-picker'));
    fireEvent.click(await screen.findByRole('option', { name: /minimouse/ }));
    expect(screen.queryByRole('button', { name: 'Return to latest' })).not.toBeInTheDocument();
  });

  it('restores the fixed chart inspection after returning from resource detail', async () => {
    const scroll = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
    render(<App />);
    fireEvent.keyDown(screen.getByRole('img', { name: 'CPU USAGE history' }), { key: 'ArrowLeft' });
    const selection = screen.getByText(/Fixed selection:/, { selector: 'div.readout-caption' }).textContent;
    const detail = screen.getByRole('button', { name: 'Open CPU detail' });
    detail.focus();
    fireEvent.click(detail);
    const sampleContext = await screen.findByRole('region', { name: 'Selected Overview sample' }, { timeout: 5000 });
    expect(sampleContext).toHaveTextContent('22.0%');
    expect(sampleContext).toHaveTextContent('Detail evidence may be newer and is not necessarily from this sample time.');
    fireEvent.click(await screen.findByRole('button', { name: /Back To Dashboard/i }));
    expect(await screen.findByText(/Fixed selection:/, { selector: 'div.readout-caption' })).toHaveTextContent(selection ?? '');
    await waitFor(() => { expect(screen.getByRole('button', { name: 'Open CPU detail' })).toHaveFocus(); });
    expect(scroll).toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Return to latest' }));
    expect(screen.queryByText(/Fixed selection:/, { selector: 'div.readout-caption' })).not.toBeInTheDocument();
    scroll.mockRestore();
  });

  it('keeps selected sample and zoom as reference only for their subject', async () => {
    const base = mocks.useSystemMonitor();
    mocks.useSystemMonitor.mockReturnValue({ ...base, metricHistory: { ...history, windowStart: '2026-02-01T23:00:00Z', windowEnd: '2026-02-02T00:00:00Z', cpu: [{ timestamp: '2026-02-01T23:20:00Z', value: 42 }, { timestamp: '2026-02-01T23:40:00Z', value: 18 }] } });
    render(<App />);
    fireEvent.keyDown(screen.getByRole('img', { name: 'CPU USAGE history' }), { key: 'ArrowLeft' });
    fireEvent.click(screen.getByRole('button', { name: 'Zoom around selected sample' }));
    fireEvent.click(screen.getByRole('button', { name: 'Open CPU detail' }));
    const sample = await screen.findByRole('region', { name: 'Selected Overview sample' }, { timeout: 5000 });
    expect(sample).toHaveTextContent('42.0%');
    expect(sample).toHaveTextContent('Overview zoom at navigation:');
    expect(sample).toHaveTextContent('Detail evidence may be newer and is not necessarily from this sample time.');
    fireEvent.click(await screen.findByTestId('machine-picker'));
    fireEvent.click(await screen.findByRole('option', { name: /minimouse/ }));
    expect(screen.queryByRole('region', { name: 'Selected Overview sample' })).not.toBeInTheDocument();
  });

  it('opens on the machine the link names', async () => {
    window.history.replaceState({}, '', '/?machine=minimouse-id');
    render(<App />);
    await waitFor(() => { expect(screen.getAllByRole('status').some(element => element.textContent?.includes('Viewing minimouse'))).toBe(true); });
  });

  it('explains a remote reading failure and offers retry before a first sample', async () => {
    const current = mocks.useSystemMonitor();
    const refresh = vi.fn();
    mocks.useSystemMonitor.mockReturnValue({ ...current, metrics: null, lastSuccessfulFetch: null, subsystemErrors: { metrics: { error: 'remote node is not available' } }, refresh });
    window.history.replaceState({}, '', '/?machine=minimouse-id');
    render(<App />);
    expect(await screen.findByText(/No reading has been received for this machine/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry readings' }));
    expect(refresh).toHaveBeenCalledOnce();
  });

  it('keeps a history failure visible independently of current readings and offers retry', async () => {
    const current = mocks.useSystemMonitor();
    const refresh = vi.fn();
    mocks.useSystemMonitor.mockReturnValue({ ...current, metricHistory: { ...history, windowStart: '2026-02-01T23:00:00Z' }, subsystemErrors: { history: { error: 'timeline unavailable' } }, refresh });
    render(<App />);
    expect(await screen.findByText(/History could not be refreshed: timeline unavailable. The previous history remains visible/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry history' }));
    expect(refresh).toHaveBeenCalledOnce();
  });

  it('explains untimed timeline cycles omitted from local plots', async () => {
    const current = mocks.useSystemMonitor();
    mocks.useSystemMonitor.mockReturnValue({ ...current, metricHistory: { ...history, missingTimestampCount: 2 } });
    render(<App />);
    expect(await screen.findByText('2 timeline cycles were omitted from plots because their timestamps were unavailable.')).toBeInTheDocument();
  });

  it('exposes dashboard workspace links on the local overview', () => {
    render(<App />);
    expect(screen.getByRole('link', { name: 'Processes and ownership' })).toHaveAttribute('href', '/metrics/cpu');
    expect(screen.getByRole('link', { name: 'Processes and ownership' })).toHaveAccessibleDescription('CPU and process activity');
    expect(screen.getByRole('link', { name: 'Memory contributors' })).toHaveAttribute('href', '/metrics/memory');
    expect(screen.getByRole('link', { name: 'Memory contributors' })).toHaveAccessibleDescription('Memory use by process');
    expect(screen.getByRole('link', { name: 'Infrastructure' })).toHaveAttribute('href', '#infrastructure-workspace');
    expect(screen.getByRole('link', { name: 'Infrastructure' })).toHaveAccessibleDescription('Devices, pools and services');
    expect(screen.getByRole('link', { name: 'Investigations and reports' })).toHaveAttribute('href', '#investigations-workspace');
    expect(screen.getByRole('link', { name: 'Investigations and reports' })).toHaveAccessibleDescription('Investigations and historical reports');
  });

  it('opens the ownership workspace at the existing process section', async () => {
    const scroll = vi.fn();
    const original = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'scrollIntoView');
    HTMLElement.prototype.scrollIntoView = scroll;
    render(<App />);
    fireEvent.click(screen.getByRole('link', { name: 'Processes and ownership' }));
    await waitFor(() => { expect(document.querySelector('[data-detail-section="process-monitor"]')).toBeInTheDocument(); });
    await waitFor(() => { expect(scroll).toHaveBeenCalled(); });
    if (original) Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', original);
    else Reflect.deleteProperty(HTMLElement.prototype, 'scrollIntoView');
  });

  it('keeps a delayed workspace visible until the user takes control', async () => {
    const target = document.createElement('section');
    target.scrollIntoView = vi.fn();
    target.getBoundingClientRect = () => new DOMRect(0, window.innerHeight + 100, 0, 0);
    document.body.appendChild(target);
    const scrollIntoView = vi.fn();
    target.scrollIntoView = scrollIntoView;
    const stop = revealWorkspace(() => target);
    await waitFor(() => { expect(scrollIntoView).toHaveBeenCalledTimes(1); });
    target.appendChild(document.createElement('p'));
    await waitFor(() => { expect(scrollIntoView).toHaveBeenCalledTimes(2); });
    window.dispatchEvent(new Event('wheel'));
    target.appendChild(document.createElement('p'));
    await new Promise(resolve => setTimeout(resolve, 50));
    expect(scrollIntoView).toHaveBeenCalledTimes(2);
    stop();
    target.remove();
  });

  it('focuses a delayed keyboard destination without reclaiming focus after user input', async () => {
    const target = document.createElement('section');
    target.tabIndex = -1;
    target.scrollIntoView = vi.fn();
    let mounted = false;
    const stop = revealWorkspace(() => mounted ? target : null, true);
    document.body.appendChild(target);
    mounted = true;
    target.appendChild(document.createElement('p'));
    await waitFor(() => expect(target).toHaveFocus());
    const other = document.createElement('button');
    document.body.appendChild(other);
    window.dispatchEvent(new Event('keydown'));
    other.focus();
    target.appendChild(document.createElement('p'));
    await new Promise(resolve => setTimeout(resolve, 50));
    expect(other).toHaveFocus();
    stop();
    target.remove();
    other.remove();
  });

  it('announces a requested lazy workspace panel without claiming its data is ready', async () => {
    render(<App />);
    fireEvent.click(screen.getByRole('link', { name: 'Investigations and reports' }), { detail: 0 });
    expect(await screen.findByText('Investigations and reports panel loaded. Data may still be loading.')).toHaveAttribute('role', 'status');
    fireEvent.click(screen.getByRole('link', { name: 'Logs', exact: true }));
    await waitFor(() => { expect(window.location.pathname).toBe('/logs'); });
    await waitFor(() => expect(screen.queryByText('Investigations and reports panel loaded. Data may still be loading.')).not.toBeInTheDocument());
    fireEvent.click(await screen.findByRole('link', { name: 'Dashboard', exact: true }));
    await waitFor(() => { expect(window.location.pathname).toBe('/'); });
    expect(screen.queryByText('Investigations and reports panel loaded. Data may still be loading.')).not.toBeInTheDocument();
  });

  it('does not retain a local workspace announcement after changing subject', async () => {
    render(<App />);
    fireEvent.click(screen.getByRole('link', { name: 'Investigations and reports' }), { detail: 0 });
    expect(await screen.findByText('Investigations and reports panel loaded. Data may still be loading.')).toBeInTheDocument();
    fireEvent.click(await screen.findByTestId('machine-picker'));
    fireEvent.click(await screen.findByRole('option', { name: /minimouse/ }));
    await waitFor(() => expect(screen.queryByText('Investigations and reports panel loaded. Data may still be loading.')).not.toBeInTheDocument());
    fireEvent.click(await screen.findByTestId('machine-picker'));
    fireEvent.click(await screen.findByRole('option', { name: /This machine/ }));
    await waitFor(() => { expect(window.location.search).toBe(''); });
    expect(screen.queryByText('Investigations and reports panel loaded. Data may still be loading.')).not.toBeInTheDocument();
  });

  it('retains an unknown subject until the operator deliberately returns locally', async () => {
    window.history.replaceState({}, '', '/?machine=retired-node');
    render(<App />);

    expect(await screen.findByRole('status')).toHaveTextContent('retired-node');
    await waitFor(() => { expect(screen.getByRole('status')).toHaveTextContent('No readings are shown.'); });
    expect(window.location.search).toBe('?machine=retired-node');
    expect(screen.queryByText(/OPEN CPU DETAIL/i)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Back to this computer' }));
    expect(window.location.search).toBe('');
  });

  it('does not describe this computer under another machine\'s name', async () => {
    window.history.replaceState({}, '', '/?machine=minimouse-id');
    render(<App />);

    await screen.findByRole('status');
    // Incident history and alerts are computed from local collectors. Left in
    // place they would read as findings about minimouse.
    expect(screen.queryByText('Incident timeline')).not.toBeInTheDocument();
    expect(screen.queryByText(/ACTIVE ALERTS/i)).not.toBeInTheDocument();
    expect(await screen.findByTestId('local-only-panels')).toHaveTextContent('not shown for minimouse');
  });

  it.each([
    ['/logs', 'Logs'], ['/forensics', 'Forensics'], ['/capacity', 'Capacity'],
    ['/scripts', 'Scripts'], ['/metrics/disk', 'Disk detail'],
  ])('keeps %s local-only for a remote subject and offers deliberate return', async (path, name) => {
    window.history.replaceState({}, '', `${path}?machine=minimouse-id`);
    render(<App />);
    expect(await screen.findByRole('heading', { name: `${name} is available on this computer` })).toBeInTheDocument();
    expect(window.location.search).toBe('?machine=minimouse-id');
    fireEvent.click(screen.getByRole('button', { name: 'View on this computer' }));
    await waitFor(() => { expect(window.location.search).toBe(''); });
    await waitFor(() => { expect(screen.queryByRole('heading', { name: `${name} is available on this computer` })).not.toBeInTheDocument(); });
  });

  it('shows no readings at all for a machine it cannot reach', async () => {
    window.history.replaceState({}, '', '/?machine=swarminator-id');
    render(<App />);

    const note = await screen.findByTestId('machine-presence-note');
    expect(note).toHaveTextContent('Nothing to read from swarminator');
    // Empty metric cards under this note would read as measurements of zero.
    expect(screen.queryByText(/OPEN CPU DETAIL/i)).not.toBeInTheDocument();
    // And it does not report vitals, so it must not be said to.
    expect(screen.queryByTestId('local-only-panels')).not.toBeInTheDocument();
  });

  it('renders the live dashboard and deferred operator sections', async () => {
    render(<App />);

    expect(screen.getByText('Vega')).toBeInTheDocument();
    expect(screen.getByText('CPU USAGE')).toBeInTheDocument();
    expect(screen.getByText(/67.0% space used/)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText('INFRASTRUCTURE MONITOR')).toBeInTheDocument(), { timeout: 5000 });
    await waitFor(() => expect(screen.getByText('PLAYBACK REPORTS')).toBeInTheDocument(), { timeout: 5000 });
    expect(screen.getByText('No threshold crossings or investigations in this window. Signals will appear here when attention is required.')).toBeInTheDocument();
  });

  it('loads every governed route and returns to the dashboard', async () => {
    render(<App />);
    const routes: Array<[string, RegExp]> = [
      ['/forensics', /Crash Forensics/], ['/logs', /Logs/], ['/capacity', /Capacity/], ['/scripts', /Investigation Scripts Library/],
      ['/metrics/cpu', /CPU PERFORMANCE/], ['/metrics/memory', /MEMORY UTILIZATION/], ['/metrics/network', /NETWORK ACTIVITY/],
      ['/metrics/gpu', /GPU UTILIZATION/], ['/metrics/disk', /DISK PERFORMANCE/],
    ];
    for (const [route, title] of routes) {
      await act(async () => { await openPath(route); });
      await waitFor(() => {
        const landmark = route.startsWith('/metrics/')
          ? screen.getByText(title)
          : screen.getByText(title, { selector: 'h1, h2, h3' });
        expect(landmark).toBeInTheDocument();
      }, { timeout: 5000 });
      expect(document.querySelector('main')).toBeTruthy();
    }
    await act(async () => { await openPath('/'); });
    expect(screen.getByText('CPU USAGE')).toBeInTheDocument();
  }, 20000);

  it('exercises operator controls and shared time state', async () => {
    const user = await import('@testing-library/user-event').then(module => module.default.setup());
    render(<App />);
    await user.selectOptions(screen.getByLabelText('Shared time range'), '24h');
    expect(screen.getByText(/Requested 24 hours · Showing previous 60-minute history/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Pause monitoring view updates' }));
    expect(screen.getByRole('button', { name: 'Resume monitoring view updates' })).toBeInTheDocument();
    await user.click(screen.getByTitle('Toggle system output'));
    await user.click(screen.getByTitle('Open system settings'));
    await waitFor(() => expect(screen.getByText('System Monitor Settings')).toBeInTheDocument(), { timeout: 5000 });
  });

  it('toggles dashboard cards and deferred panels in both directions', async () => {
    render(<App />);
    await waitFor(() => expect(screen.getByText('INFRASTRUCTURE MONITOR')).toBeInTheDocument(), { timeout: 5000 });
    const cpuCard = screen.getByText('CPU USAGE').closest('.metric-card');
    if (!cpuCard) throw new Error('CPU card was not rendered');
    fireEvent.click(cpuCard);
    fireEvent.click(cpuCard);
    const infrastructure = screen.getByText('INFRASTRUCTURE MONITOR').closest('.monitor-panel-header');
    if (!infrastructure) throw new Error('infrastructure panel was not rendered');
    fireEvent.click(infrastructure);
    fireEvent.click(infrastructure);
  });
});
