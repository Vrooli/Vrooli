import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { APIError, DetailedMetrics, MetricHistory, StorageIOInfo } from '../../../types';
import { DiskDetailView } from './DiskDetailView';

const mocks = vi.hoisted(() => ({ protoFetch: vi.fn() }));
const activityMocks = vi.hoisted(() => ({ fetchDiskReadActivity: vi.fn() }));
vi.mock('../../../shared/api/apiFetch', () => ({ protoFetch: mocks.protoFetch }));
vi.mock('../../../shared/api/proto-contracts', () => ({ parseDiskDetailResponse: vi.fn() }));
vi.mock('../diskReadActivity', () => ({ fetchDiskReadActivity: activityMocks.fetchDiskReadActivity }));
vi.mock('./MetricDetailViews', () => ({
  DetailSection: ({ children }: { children: React.ReactNode }) => <section>{children}</section>,
  MetricDetailLayout: ({ title, headline, subhead, onBack, children }: { title: string; headline: string; subhead?: string; onBack: () => void; children: React.ReactNode }) => <main><button type="button" onClick={onBack}>back</button><h1>{title}</h1><div>{headline}</div><div>{subhead}</div>{children}</main>,
  MetricLineChart: ({ data }: { data: unknown[] }) => <div data-testid="chart">{data.length} chart points</div>,
}));

const timestamp = timestampFromDate(new Date('2026-01-01T00:00:00Z'));
const diskResponse = {
  activeMount: '/', depth: 2, timestamp,
  partitions: [
    { mountPoint: '/', device: '/dev/test', usedBytes: BigInt(8), sizeBytes: BigInt(10), usePercent: 80, usedHuman: '8 GB', availableHuman: '2 GB' },
    { mountPoint: '/data', device: '/dev/data', usedBytes: BigInt(4), sizeBytes: BigInt(10), usePercent: 40, usedHuman: '4 GB', availableHuman: '6 GB' },
  ],
  topDirectories: [{ path: '/var', sizeHuman: '2 GB' }],
  largestFiles: [{ path: '/var/log/app.log', sizeHuman: '60 MB' }],
  notes: ['scan complete'],
};
const detailedMetrics = {
  timestamp,
  memoryDetails: { diskUsage: { used: BigInt(8), total: BigInt(10), percent: 80 } },
  systemDetails: {
    fileDescriptors: { used: 80, max: 100, percent: 80 },
    inotifyWatchers: { supported: true, watchesUsed: 80, watchesMax: 100, watchesPercent: 80, instancesUsed: 4, instancesMax: 10, instancesPercent: 40 },
  },
} as unknown as DetailedMetrics;
const storageIO = {
  diskQueueDepth: 0, ioWaitPercent: 0, readMbPerSec: 0, writeMbPerSec: 0,
  diskQueueDepthState: { state: { case: 'unsupportedReason', value: 'Disk queue depth is not collected' } },
  ioWaitState: { state: { case: 'unsupportedReason', value: 'Disk I/O wait is not collected' } },
  readRate: { state: { case: 'measured', value: 2 } },
  writeRate: { state: { case: 'measured', value: 3 } },
} as unknown as StorageIOInfo;
const history = { diskRead: [{ timestamp: '2026-01-01T00:00:00Z', value: 1 }], diskWrite: [{ timestamp: '2026-01-01T00:00:00Z', value: 2 }], diskUsage: [{ timestamp: '2026-01-01T00:00:00Z', value: 80 }] } as unknown as MetricHistory;

describe('DiskDetailView', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/');
    vi.clearAllMocks();
    mocks.protoFetch.mockResolvedValue(diskResponse);
  });

  it('renders disk evidence, mounted volumes, analysis controls, and scan results', async () => {
    const onBack = vi.fn();
    render(<DiskDetailView detailedMetrics={detailedMetrics} storageIO={storageIO} metricHistory={history} onBack={onBack} />);
    expect(await screen.findByText('DISK PERFORMANCE')).toBeInTheDocument();
    expect(screen.getByText('80.0% utilized on /')).toBeInTheDocument();
    expect(await screen.findByText('/var')).toBeInTheDocument();
    await waitFor(() => { expect(document.body.textContent).toContain('scan complete'); });
    expect(screen.getByText(/summed across physical devices/)).toBeInTheDocument();
    expect(screen.getByText('Not collected: Disk queue depth is not collected')).toBeInTheDocument();
    expect(screen.getByText('Not collected: Disk I/O wait is not collected')).toBeInTheDocument();
    expect(screen.getByText('2.00 MB/s')).toBeInTheDocument();
    expect(screen.getAllByText('1 chart points')).toHaveLength(2);
    expect(screen.getByText('File Descriptor Utilization')).toBeInTheDocument();
    expect(screen.getByText('Inotify Watcher Utilization')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Find Largest Files (>50MB)' }));
    expect(screen.getByText('/var/log/app.log')).toBeInTheDocument();
    fireEvent.change(screen.getByRole('combobox'), { target: { value: '3' } });
    await waitFor(() => { expect(screen.getByRole('button', { name: 'Refresh Scan' })).toBeEnabled(); });
    fireEvent.click(screen.getByRole('button', { name: 'Refresh Scan' }));
    fireEvent.click(screen.getByRole('button', { name: 'back' }));
    await waitFor(() => { expect(onBack).toHaveBeenCalledOnce(); });
    expect(mocks.protoFetch.mock.calls.some(([url]) => String(url).includes('include_files=true'))).toBe(true);
  });

  it('shows unavailable platform data and reports scan failures', async () => {
    window.history.replaceState({}, '', '?disk_files=true');
    mocks.protoFetch.mockRejectedValue(new Error('disk scan failed'));
    render(<DiskDetailView detailedMetrics={null} storageIO={null} metricHistory={null} diskLastUpdated="2026-01-01T00:00:00Z" onBack={vi.fn()} />);
    expect(await screen.findByText('Awaiting disk telemetry')).toBeInTheDocument();
    expect(screen.getByText('Storage I/O metrics unavailable.')).toBeInTheDocument();
    expect(screen.getByText('Partition information is unavailable on this platform.')).toBeInTheDocument();
    expect(await screen.findByText('No successful directory scan is available for this request.')).toBeInTheDocument();
    expect(screen.getByText('No successful largest-file scan is available for this request.')).toBeInTheDocument();
    expect(screen.queryByText('No directories exceeded the scan threshold at this depth.')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Stop Scan' }));
  });

  it('does not report a healthy empty scan when filesystem attribution is unavailable', async () => {
    mocks.protoFetch.mockResolvedValue({
      ...diskResponse,
      topDirectories: [],
      largestFiles: [],
      notes: [
        'Directory attribution is unavailable: permission denied',
        'Largest-file attribution is unavailable: permission denied',
      ],
    });
    render(<DiskDetailView detailedMetrics={detailedMetrics} storageIO={storageIO} metricHistory={history} onBack={vi.fn()} />);
    expect(await screen.findByText('Directory results are unavailable; see the analysis notes for the read failure.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Find Largest Files (>50MB)' }));
    expect(await screen.findByText('Largest-file results are unavailable; see the analysis notes for the read failure.')).toBeInTheDocument();
  });

  it('samples host-wide hot writers on demand and keeps the prior sample after refresh failure', async () => {
    activityMocks.fetchDiskReadActivity
      .mockResolvedValueOnce({
        status: 'partial', platform: 'linux', windowSeconds: 2, matched: 12, permissionDenied: 3,
        firstObservedAt: '2026-01-01T00:00:00Z', lastObservedAt: '2026-01-01T00:00:02Z',
        rows: [], storageRows: [{ pid: 42, name: 'writer', owner: 'scenario-a', storageBytesPerSecond: 2_000_000, returnedBytesPerSecond: 5_000_000, intervalSeconds: 2, observedChildExits: 0 }],
        semantics: 'host-wide; process rows may overlap',
      })
      .mockRejectedValueOnce(new Error('activity sample failed'));
    render(<DiskDetailView detailedMetrics={detailedMetrics} storageIO={storageIO} metricHistory={history} onBack={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Find hot writers (2-second sample)' }));

    expect(await screen.findByText('writer')).toBeInTheDocument();
    expect(screen.getByText('2.00')).toBeInTheDocument();
    expect(screen.getByText('5.00')).toBeInTheDocument();
    expect(screen.getByText(/not attributed to \/\./)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Find hot writers (2-second sample)' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('activity sample failed');
    expect(screen.getByRole('alert')).toHaveTextContent('The previous dated sample remains visible.');
    expect(screen.getByText('writer')).toBeInTheDocument();
  });

  it('keeps the last successful mount results and retries only the failed mount scan', async () => {
    const recovered = {
      ...diskResponse,
      activeMount: '/data',
      topDirectories: [{ path: '/data/cache', sizeHuman: '1 GB' }],
      largestFiles: [],
    };
    mocks.protoFetch
      .mockResolvedValueOnce(diskResponse)
      .mockRejectedValueOnce(new Error('mount read failed'))
      .mockResolvedValueOnce(recovered);
    render(<DiskDetailView detailedMetrics={detailedMetrics} storageIO={storageIO} metricHistory={history} onBack={vi.fn()} />);
    expect(await screen.findByText('/var')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /\/data/ }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Disk analysis was not completed for /data');
    expect(screen.getByRole('alert')).toHaveTextContent('Still showing the last successful results for / at depth 2');
    expect(screen.getByText('/var')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Retry this scan' }));
    expect(await screen.findByText('/data/cache')).toBeInTheDocument();
    expect(mocks.protoFetch).toHaveBeenCalledTimes(3);
    expect(String(mocks.protoFetch.mock.calls[2]?.[0])).toContain('mount=%2Fdata');
  });

  it('cancels an in-flight mount walk while retaining the last successful results', async () => {
    let scanSignal: AbortSignal | undefined;
    const recovered = { ...diskResponse, activeMount: '/data', topDirectories: [{ path: '/data/cache', sizeHuman: '1 GB' }] };
    mocks.protoFetch
      .mockResolvedValueOnce(diskResponse)
      .mockImplementationOnce((_url, _parser, options) => {
        scanSignal = (options as RequestInit | undefined)?.signal ?? undefined;
        return new Promise((_resolve, reject) => {
          scanSignal?.addEventListener('abort', () => {
            const error = new Error('The operation was aborted');
            error.name = 'AbortError';
            reject(error);
          }, { once: true });
        });
      })
      .mockResolvedValueOnce(recovered);
    render(<DiskDetailView detailedMetrics={detailedMetrics} storageIO={storageIO} metricHistory={history} onBack={vi.fn()} />);
    expect(await screen.findByText('/var')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /\/data/ }));
    await waitFor(() => { expect(scanSignal).toBeDefined(); });
    expect(screen.getByRole('status')).toHaveTextContent('Scanning /data at depth 2, directories only.');
    fireEvent.click(screen.getByRole('button', { name: 'Stop Scan' }));

    expect(scanSignal?.aborted).toBe(true);
    expect(screen.getByRole('alert')).toHaveTextContent('Scan stopped before it completed');
    expect(screen.getByRole('alert')).toHaveTextContent('Still showing the last successful results for / at depth 2');
    expect(screen.getByText('/var')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Retry this scan' }));
    expect(await screen.findByText('/data/cache')).toBeInTheDocument();
    expect(String(mocks.protoFetch.mock.calls[2]?.[0])).toContain('mount=%2Fdata');
  });

  it('keeps dated disk history visible and retries only the timeline when refresh fails', () => {
    const onRetryHistory = vi.fn();
    render(<DiskDetailView
      detailedMetrics={detailedMetrics}
      storageIO={storageIO}
      metricHistory={{ ...history, windowStart: '2026-01-01T00:00:00Z' } as unknown as MetricHistory}
      historyError={{ error: 'timeline read failed' } as APIError}
      onRetryHistory={onRetryHistory}
      onBack={vi.fn()}
    />);
    expect(screen.getByRole('alert')).toHaveTextContent('The previous dated history remains visible.');
    expect(screen.getByRole('alert')).toHaveTextContent('timeline read failed');
    expect(screen.getAllByText('1 chart points')).toHaveLength(2);
    fireEvent.click(screen.getByRole('button', { name: 'Retry history' }));
    expect(onRetryHistory).toHaveBeenCalledOnce();
    expect(mocks.protoFetch).toHaveBeenCalledTimes(1);
  });
});
