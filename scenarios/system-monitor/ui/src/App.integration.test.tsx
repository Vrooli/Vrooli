// provider-free-exception: App is the root component and owns BrowserRouter plus all application providers.
import { act, fireEvent, render, screen, waitFor, cleanup } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';

type Parser = (value: unknown) => unknown;

const mocks = vi.hoisted(() => ({
  useSystemMonitor: vi.fn(),
  useInvestigationAgents: vi.fn(),
  useScriptExecution: vi.fn(),
  protoFetch: vi.fn(),
  fetchForensicsSummary: vi.fn(),
  fetchCapacityOverview: vi.fn(),
  fetchCapacityReconciliation: vi.fn(),
  fetchCapacityPolicy: vi.fn(),
  setCapacityPolicy: vi.fn(),
  fetchLogs: vi.fn(),
  fetchUnits: vi.fn(),
  fetchBoots: vi.fn(),
}));

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
  apiFetch: vi.fn(),
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
    growthPatterns: [{ process: 'api', growthMbPerHour: 2, riskLevel: 'low' }],
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

function installFixtures() {
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
    lastSuccessfulFetch: new Date('2026-02-02T00:00:00Z'),
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
  await waitFor(() => { expect(document.body.textContent).not.toContain('Loading...'); }, { timeout: 5000 });
}

describe('App production surfaces', () => {
  beforeEach(() => {
    installFixtures();
    Object.defineProperty(window, 'innerWidth', { value: 1440, configurable: true });
    window.history.replaceState({}, '', '/');
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('renders the live dashboard and deferred operator sections', async () => {
    render(<App />);

    expect(screen.getByText('System Monitor')).toBeInTheDocument();
    expect(screen.getByText('CPU USAGE')).toBeInTheDocument();
    expect(screen.getByText('67.0')).toBeInTheDocument();
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
      await waitFor(() => expect(screen.getByText(title)).toBeInTheDocument(), { timeout: 5000 });
      expect(document.querySelector('main')).toBeTruthy();
    }
    await act(async () => { await openPath('/'); });
    expect(screen.getByText('CPU USAGE')).toBeInTheDocument();
  });

  it('exercises operator controls and shared time state', async () => {
    const user = await import('@testing-library/user-event').then(module => module.default.setup());
    render(<App />);
    await user.selectOptions(screen.getByLabelText('Shared time range'), '24h');
    await user.click(screen.getByRole('button', { name: 'Pause live updates' }));
    expect(screen.getByRole('button', { name: 'Resume live updates' })).toBeInTheDocument();
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
