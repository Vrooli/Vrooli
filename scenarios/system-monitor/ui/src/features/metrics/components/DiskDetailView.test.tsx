import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { DetailedMetrics, MetricHistory, StorageIOInfo } from '../../../types';
import { DiskDetailView } from './DiskDetailView';

const mocks = vi.hoisted(() => ({ protoFetch: vi.fn() }));
vi.mock('../../../shared/api/apiFetch', () => ({ protoFetch: mocks.protoFetch }));
vi.mock('../../../shared/api/proto-contracts', () => ({ parseDiskDetailResponse: vi.fn() }));
vi.mock('./MetricDetailViews', () => ({
  MetricDetailLayout: ({ title, headline, subhead, onBack, children }: { title: string; headline: string; subhead?: string; onBack: () => void; children: React.ReactNode }) => <main><button type="button" onClick={onBack}>back</button><h1>{title}</h1><div>{headline}</div><div>{subhead}</div>{children}</main>,
  MetricLineChart: ({ data }: { data: unknown[] }) => <div data-testid="chart">{data.length} chart points</div>,
}));

const timestamp = timestampFromDate(new Date('2026-01-01T00:00:00Z'));
const diskResponse = {
  activeMount: '/', depth: 2, timestamp,
  partitions: [{ mountPoint: '/', device: '/dev/test', usedBytes: BigInt(8), sizeBytes: BigInt(10), usePercent: 80, usedHuman: '8 GB', availableHuman: '2 GB' }],
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
const storageIO = { diskQueueDepth: 1.2, ioWaitPercent: 4, readMbPerSec: 2, writeMbPerSec: 3 } as unknown as StorageIOInfo;
const history = { diskRead: [{ timestamp: '2026-01-01T00:00:00Z', value: 1 }], diskWrite: [{ timestamp: '2026-01-01T00:00:00Z', value: 2 }], diskUsage: [{ timestamp: '2026-01-01T00:00:00Z', value: 80 }] } as unknown as MetricHistory;

describe('DiskDetailView', () => {
  beforeEach(() => {
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
    mocks.protoFetch.mockRejectedValue(new Error('disk scan failed'));
    render(<DiskDetailView detailedMetrics={null} storageIO={null} metricHistory={null} diskLastUpdated="2026-01-01T00:00:00Z" onBack={vi.fn()} />);
    expect(await screen.findByText('Awaiting disk telemetry')).toBeInTheDocument();
    expect(screen.getByText('Storage I/O metrics unavailable.')).toBeInTheDocument();
    expect(screen.getByText('Partition information is unavailable on this platform.')).toBeInTheDocument();
    expect(await screen.findByText('Failed to analyze disk usage: disk scan failed')).toBeInTheDocument();
    expect(screen.getByText('No directories exceeded the scan threshold at this depth.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Stop Scan' }));
  });
});
